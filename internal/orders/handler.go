package orders

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alumieye/eyeapp-backend/internal/apierrors"
	"github.com/alumieye/eyeapp-backend/internal/models"
	"github.com/alumieye/eyeapp-backend/internal/repositories"
)

const (
	defaultLimit = 20
	maxLimit     = 100
	// GMT+7 time string format used by eyebroker
	gmtPlus7Format = "2006-01-02 15:04:05"
)

type Handler struct {
	repo repositories.OrderRepository
}

func NewHandler(repo repositories.OrderRepository) *Handler {
	return &Handler{repo: repo}
}

type orderResponse struct {
	ID           int64    `json:"id"`
	Symbol       string   `json:"symbol"`
	Timestamp    int64    `json:"timestamp"`
	TimestampStr string   `json:"timestamp_str"`
	Side         string   `json:"side"`
	OrderType    string   `json:"order_type"`
	MainPosition string   `json:"main_position"`
	Price        float64  `json:"price"`
	Quantity     float64  `json:"quantity"`
	CandleID     *int64   `json:"candle_id"`
	CreatedAt    string   `json:"created_at"`
}

type paginationResponse struct {
	Limit      int     `json:"limit"`
	NextCursor *string `json:"next_cursor,omitempty"`
	HasMore    bool    `json:"has_more"`
}

type listResponse struct {
	Data       []orderResponse    `json:"data"`
	Pagination paginationResponse `json:"pagination"`
}

// List handles GET /api/v1/orders
// @Summary List orders
// @Description List trade orders from eyebroker with optional filters and cursor pagination
// @Tags orders
// @Produce json
// @Security BearerAuth
// @Param symbol     query string false "Filter by asset symbol (e.g. ETHUSDT)"
// @Param side       query string false "Filter by side: buy or sell"
// @Param order_type query string false "Filter by order type (e.g. market, limit)"
// @Param from       query string false "Start of time range GMT+7 (YYYY-MM-DD HH:MM:SS)"
// @Param to         query string false "End of time range GMT+7 (YYYY-MM-DD HH:MM:SS)"
// @Param limit      query int    false "Page size (default 20, max 100)"
// @Param cursor     query string false "Cursor from previous response for next page"
// @Success 200 {object} listResponse
// @Failure 400 {object} apierrors.ErrorResponse
// @Failure 401 {object} apierrors.ErrorResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /api/v1/orders [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// --- limit ---
	limit := defaultLimit
	if ls := q.Get("limit"); ls != "" {
		v, err := strconv.Atoi(ls)
		if err != nil || v <= 0 {
			apierrors.ValidationError(w, "limit must be a positive integer")
			return
		}
		if v > maxLimit {
			apierrors.ValidationError(w, fmt.Sprintf("limit must not exceed %d", maxLimit))
			return
		}
		limit = v
	}

	// --- side ---
	side := q.Get("side")
	if side != "" && side != "buy" && side != "sell" {
		apierrors.ValidationError(w, "side must be 'buy' or 'sell'")
		return
	}

	// --- from / to (GMT+7 → UTC epoch ms) ---
	var fromMS, toMS *int64
	if fs := q.Get("from"); fs != "" {
		t, err := parseGMT7ToMS(fs)
		if err != nil {
			apierrors.ValidationError(w, "from must be in format YYYY-MM-DD HH:MM:SS (GMT+7)")
			return
		}
		fromMS = &t
	}
	if ts := q.Get("to"); ts != "" {
		t, err := parseGMT7ToMS(ts)
		if err != nil {
			apierrors.ValidationError(w, "to must be in format YYYY-MM-DD HH:MM:SS (GMT+7)")
			return
		}
		toMS = &t
	}

	// --- cursor ---
	var cursorTS, cursorID *int64
	if cs := q.Get("cursor"); cs != "" {
		ts, id, err := decodeCursor(cs)
		if err != nil {
			apierrors.ValidationError(w, "invalid cursor")
			return
		}
		cursorTS = &ts
		cursorID = &id
	}

	filter := repositories.OrderFilter{
		Symbol:    q.Get("symbol"),
		Side:      side,
		OrderType: q.Get("order_type"),
		FromMS:    fromMS,
		ToMS:      toMS,
		CursorTS:  cursorTS,
		CursorID:  cursorID,
		Limit:     limit + 1, // fetch one extra to detect has_more
	}

	rows, err := h.repo.List(r.Context(), filter)
	if err != nil {
		apierrors.InternalError(w)
		return
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	data := make([]orderResponse, len(rows))
	for i, o := range rows {
		data[i] = toResponse(o)
	}

	var nextCursor *string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		c := encodeCursor(last.Timestamp, last.ID)
		nextCursor = &c
	}

	apierrors.JSON(w, http.StatusOK, listResponse{
		Data: data,
		Pagination: paginationResponse{
			Limit:      limit,
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	})
}

func toResponse(o *models.Order) orderResponse {
	return orderResponse{
		ID:           o.ID,
		Symbol:       o.Symbol,
		Timestamp:    o.Timestamp,
		TimestampStr: o.TimestampStr,
		Side:         o.Side,
		OrderType:    o.OrderType,
		MainPosition: o.MainPosition,
		Price:        o.Price,
		Quantity:     o.Quantity,
		CandleID:     o.CandleID,
		CreatedAt:    o.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// encodeCursor returns a base64-encoded "timestamp:id" string.
func encodeCursor(ts, id int64) string {
	raw := fmt.Sprintf("%d:%d", ts, id)
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// decodeCursor parses a base64-encoded cursor back to (timestamp ms, id).
func decodeCursor(cursor string) (int64, int64, error) {
	b, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, 0, err
	}
	parts := strings.SplitN(string(b), ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("malformed cursor")
	}
	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return ts, id, nil
}

// parseGMT7ToMS parses a "YYYY-MM-DD HH:MM:SS" GMT+7 string and returns UTC epoch ms.
func parseGMT7ToMS(s string) (int64, error) {
	gmt7 := time.FixedZone("GMT+7", 7*3600)
	t, err := time.ParseInLocation(gmtPlus7Format, s, gmt7)
	if err != nil {
		return 0, err
	}
	return t.UnixMilli(), nil
}
