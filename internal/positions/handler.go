package positions

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
	maxLimit     = 50
)

type Handler struct {
	repo repositories.PositionRepository
}

func NewHandler(repo repositories.PositionRepository) *Handler {
	return &Handler{repo: repo}
}

type positionResponse struct {
	ID            int64    `json:"id"`
	Symbol        string   `json:"symbol"`
	MarketID      int      `json:"market_id"`
	Side          string   `json:"side"`
	Status        string   `json:"status"`
	Term          string   `json:"term"`
	Active        bool     `json:"active"`
	Timestamp     int64    `json:"timestamp"`
	TimestampStr  string   `json:"timestamp_str"`
	AvgPrice      float64  `json:"avg_price"`
	Size          float64  `json:"size"`
	Capacity      float64  `json:"capacity"`
	RealizedPnl   *float64 `json:"realized_pnl"`
	DriveCandleID int64    `json:"drive_candle_id"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type orderResponse struct {
	ID           int64   `json:"id"`
	PositionID   int64   `json:"position_id"`
	Timestamp    int64   `json:"timestamp"`
	TimestampStr string  `json:"timestamp_str"`
	Side         string  `json:"side"`
	OrderType    string  `json:"order_type"`
	Price        float64 `json:"price"`
	Quantity     float64 `json:"quantity"`
	OrderPnl     float64 `json:"order_pnl"`
	PositionPnl  float64 `json:"position_pnl"`
	SignalID     *int64  `json:"signal_id"`
	CreatedAt    string  `json:"created_at"`
}

type positionDetailResponse struct {
	positionResponse
	Orders []orderResponse `json:"orders"`
}

type listResponse struct {
	Total int                `json:"total"`
	Items []positionResponse `json:"items"`
}

// List handles GET /api/v1/positions/{marketId}
// @Summary List positions
// @Description List trading positions from eyebroker, paginated with optional filters
// @Tags positions
// @Produce json
// @Security BearerAuth
// @Param marketId  path  int    true  "Market: 1=crypto 2=vnstock"
// @Param limit     query int    false "Page size (default 15, max 50)"
// @Param offset    query int    false "Number of items to skip (default 0)"
// @Param is_active query bool   false "Filter by active status"
// @Param status    query string false "Filter by lifecycle status: opening|opened|closing|closed"
// @Param symbol    query string false "Filter by asset symbol (case-insensitive)"
// @Success 200 {object} listResponse
// @Failure 400 {object} apierrors.ErrorResponse
// @Failure 401 {object} apierrors.ErrorResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /api/v1/positions/{marketId} [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	marketID, err := parseMarketID(chi.URLParam(r, "marketId"))
	if err != nil {
		apierrors.ValidationError(w, err.Error())
		return
	}

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

	offset := 0
	if os := q.Get("offset"); os != "" {
		v, err := strconv.Atoi(os)
		if err != nil || v < 0 {
			apierrors.ValidationError(w, "offset must be a non-negative integer")
			return
		}
		offset = v
	}

	f := repositories.PositionFilter{
		MarketID: marketID,
		Limit:    limit,
		Offset:   offset,
		Symbol:   q.Get("symbol"),
		Status:   q.Get("status"),
	}
	if ia := q.Get("is_active"); ia != "" {
		v, err := strconv.ParseBool(ia)
		if err != nil {
			apierrors.ValidationError(w, "is_active must be true or false")
			return
		}
		f.IsActive = &v
	}

	result, err := h.repo.List(r.Context(), f)
	if err != nil {
		apierrors.InternalError(w)
		return
	}

	items := make([]positionResponse, len(result.Items))
	for i, p := range result.Items {
		items[i] = toResponse(p)
	}
	apierrors.JSON(w, http.StatusOK, listResponse{Total: result.Total, Items: items})
}

// Get handles GET /api/v1/positions/{marketId}/{positionId}
// @Summary Get position detail
// @Description Get a single position with its full order history
// @Tags positions
// @Produce json
// @Security BearerAuth
// @Param marketId    path int true "Market: 1=crypto 2=vnstock"
// @Param positionId  path int true "Position ID"
// @Success 200 {object} positionDetailResponse
// @Failure 400 {object} apierrors.ErrorResponse
// @Failure 401 {object} apierrors.ErrorResponse
// @Failure 404 {object} apierrors.ErrorResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /api/v1/positions/{marketId}/{positionId} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	marketID, err := parseMarketID(chi.URLParam(r, "marketId"))
	if err != nil {
		apierrors.ValidationError(w, err.Error())
		return
	}

	positionID, err := strconv.ParseInt(chi.URLParam(r, "positionId"), 10, 64)
	if err != nil || positionID <= 0 {
		apierrors.ValidationError(w, "positionId must be a positive integer")
		return
	}

	detail, err := h.repo.Get(r.Context(), marketID, positionID)
	if err != nil {
		apierrors.InternalError(w)
		return
	}
	if detail == nil {
		apierrors.Error(w, http.StatusNotFound, apierrors.CodeNotFound, "position not found")
		return
	}

	orders := make([]orderResponse, len(detail.Orders))
	for i, o := range detail.Orders {
		orders[i] = orderResponse{
			ID:           o.ID,
			PositionID:   o.PositionID,
			Timestamp:    o.Timestamp,
			TimestampStr: o.TimestampStr,
			Side:         o.Side,
			OrderType:    o.OrderType,
			Price:        o.Price,
			Quantity:     o.Quantity,
			OrderPnl:     o.OrderPnl,
			PositionPnl:  o.PositionPnl,
			SignalID:     o.SignalID,
			CreatedAt:    o.CreatedAt,
		}
	}
	apierrors.JSON(w, http.StatusOK, positionDetailResponse{
		positionResponse: toResponse(&detail.Position),
		Orders:           orders,
	})
}

func parseMarketID(s string) (int, error) {
	v, err := strconv.Atoi(s)
	if err != nil || (v != 1 && v != 2) {
		return 0, fmt.Errorf("marketId must be 1 (crypto) or 2 (vnstock)")
	}
	return v, nil
}

func toResponse(p *models.Position) positionResponse {
	return positionResponse{
		ID:            p.ID,
		Symbol:        p.Symbol,
		MarketID:      p.MarketID,
		Side:          p.Side,
		Status:        p.Status,
		Term:          p.Term,
		Active:        p.Active,
		Timestamp:     p.Timestamp,
		TimestampStr:  p.TimestampStr,
		AvgPrice:      p.AvgPrice,
		Size:          p.Size,
		Capacity:      p.Capacity,
		RealizedPnl:   p.RealizedPnl,
		DriveCandleID: p.DriveCandleID,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}
