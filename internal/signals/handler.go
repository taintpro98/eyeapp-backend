package signals

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/alumieye/eyeapp-backend/internal/apierrors"
	"github.com/alumieye/eyeapp-backend/internal/models"
	"github.com/alumieye/eyeapp-backend/internal/repositories"
	"github.com/go-chi/chi/v5"
)

const (
	defaultLimit = 15
	maxLimit     = 20
)

type Handler struct {
	repo repositories.SignalRepository
}

func NewHandler(repo repositories.SignalRepository) *Handler {
	return &Handler{repo: repo}
}

type signalResponse struct {
	ID           int64   `json:"id"`
	Symbol       string  `json:"symbol"`
	MarketID     int     `json:"market_id"`
	Timestamp    int64   `json:"timestamp"`
	TimestampStr string  `json:"timestamp_str"`
	Side         string  `json:"side"`
	SignalType   string  `json:"signal_type"`
	MainPosition string  `json:"main_position"`
	Price        float64 `json:"price"`
	Quantity     float64 `json:"quantity"`
	Confidence   float64 `json:"confidence"`
	CandleID     *int64  `json:"candle_id"`
}

type listResponse struct {
	Total int              `json:"total"`
	Items []signalResponse `json:"items"`
}

// List handles GET /api/v1/signals/{marketId}
// @Summary List signals
// @Description List trade signals from eyebroker, paginated by offset
// @Tags signals
// @Produce json
// @Security BearerAuth
// @Param marketId path  int    true  "Market: 1=crypto 2=vnstock"
// @Param limit    query int    false "Page size (default 15, max 20)"
// @Param offset   query int    false "Number of items to skip (default 0)"
// @Param symbol   query string false "Filter by symbol, e.g. VNM (case-insensitive)"
// @Success 200 {object} listResponse
// @Failure 400 {object} apierrors.ErrorResponse
// @Failure 401 {object} apierrors.ErrorResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /api/v1/signals/{marketId} [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// marketId (path param)
	marketIDStr := chi.URLParam(r, "marketId")
	marketID, err := strconv.Atoi(marketIDStr)
	if err != nil || (marketID != 1 && marketID != 2) {
		apierrors.ValidationError(w, "marketId must be 1 (crypto) or 2 (vnstock)")
		return
	}

	// limit
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

	// offset
	offset := 0
	if os := q.Get("offset"); os != "" {
		v, err := strconv.Atoi(os)
		if err != nil || v < 0 {
			apierrors.ValidationError(w, "offset must be a non-negative integer")
			return
		}
		offset = v
	}

	result, err := h.repo.List(r.Context(), repositories.SignalFilter{
		MarketID: marketID,
		Limit:    limit,
		Offset:   offset,
		Symbol:   q.Get("symbol"),
	})
	if err != nil {
		apierrors.InternalError(w)
		return
	}

	items := make([]signalResponse, len(result.Items))
	for i, s := range result.Items {
		items[i] = toResponse(s)
	}

	apierrors.JSON(w, http.StatusOK, listResponse{
		Total: result.Total,
		Items: items,
	})
}

func toResponse(s *models.Signal) signalResponse {
	return signalResponse{
		ID:           s.ID,
		Symbol:       s.Symbol,
		MarketID:     s.MarketID,
		Timestamp:    s.Timestamp,
		TimestampStr: s.TimestampStr,
		Side:         s.Side,
		SignalType:   s.SignalType,
		MainPosition: s.MainPosition,
		Price:        s.Price,
		Quantity:     s.Quantity,
		Confidence:   s.Confidence,
		CandleID:     s.CandleID,
	}
}
