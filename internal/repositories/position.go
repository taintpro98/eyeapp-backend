package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/alumieye/eyeapp-backend/internal/models"
)

type PositionFilter struct {
	MarketID int
	Limit    int
	Offset   int
	IsActive *bool
	Status   string
	Symbol   string
}

type PositionListResult struct {
	Total int
	Items []*models.Position
}

type PositionRepository interface {
	List(ctx context.Context, f PositionFilter) (*PositionListResult, error)
	Get(ctx context.Context, marketID int, positionID int64) (*models.PositionDetail, error)
}

type positionHTTPClient struct {
	baseURL string
	http    *http.Client
}

func NewPositionHTTPRepository(baseURL string) PositionRepository {
	return &positionHTTPClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

type eyebrokerPositionListResponse struct {
	Total int                `json:"total"`
	Items []eyebrokerPosition `json:"items"`
}

type eyebrokerPosition struct {
	ID            int64   `json:"id"`
	Symbol        string  `json:"symbol"`
	MarketID      int     `json:"market_id"`
	Side          string  `json:"side"`
	Status        string  `json:"status"`
	Term          string  `json:"term"`
	Active        bool    `json:"active"`
	Timestamp     int64   `json:"timestamp"`
	TimestampStr  string  `json:"timestamp_str"`
	AvgPrice      float64 `json:"avg_price"`
	Size          float64 `json:"size"`
	Capacity      float64 `json:"capacity"`
	DriveCandleID int64   `json:"drive_candle_id"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type eyebrokerPositionDetail struct {
	eyebrokerPosition
	Orders []eyebrokerOrder `json:"orders"`
}

type eyebrokerOrder struct {
	ID           int64   `json:"id"`
	PositionID   int64   `json:"position_id"`
	Timestamp    int64   `json:"timestamp"`
	TimestampStr string  `json:"timestamp_str"`
	Side         string  `json:"side"`
	OrderType    string  `json:"order_type"`
	Price        float64 `json:"price"`
	Quantity     float64 `json:"quantity"`
	SignalID      *int64  `json:"signal_id"`
	CreatedAt    string  `json:"created_at"`
}

func toModelPosition(p eyebrokerPosition) *models.Position {
	return &models.Position{
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
		DriveCandleID: p.DriveCandleID,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

func (c *positionHTTPClient) List(ctx context.Context, f PositionFilter) (*PositionListResult, error) {
	u, err := url.Parse(c.baseURL + "/v1/api/market/" + strconv.Itoa(f.MarketID) + "/positions")
	if err != nil {
		return nil, fmt.Errorf("eyebroker positions: parse url: %w", err)
	}
	q := u.Query()
	q.Set("limit", strconv.Itoa(f.Limit))
	q.Set("offset", strconv.Itoa(f.Offset))
	if f.IsActive != nil {
		q.Set("is_active", strconv.FormatBool(*f.IsActive))
	}
	if f.Status != "" {
		q.Set("status", f.Status)
	}
	if f.Symbol != "" {
		q.Set("symbol", f.Symbol)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("eyebroker positions: create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eyebroker positions: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eyebroker positions: unexpected status %d", resp.StatusCode)
	}

	var body eyebrokerPositionListResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("eyebroker positions: decode response: %w", err)
	}

	items := make([]*models.Position, len(body.Items))
	for i, p := range body.Items {
		items[i] = toModelPosition(p)
	}
	return &PositionListResult{Total: body.Total, Items: items}, nil
}

func (c *positionHTTPClient) Get(ctx context.Context, marketID int, positionID int64) (*models.PositionDetail, error) {
	path := fmt.Sprintf("%s/v1/api/market/%d/positions/%d", c.baseURL, marketID, positionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("eyebroker position detail: create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eyebroker position detail: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eyebroker position detail: unexpected status %d", resp.StatusCode)
	}

	var body eyebrokerPositionDetail
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("eyebroker position detail: decode response: %w", err)
	}

	orders := make([]models.PositionOrder, len(body.Orders))
	for i, o := range body.Orders {
		orders[i] = models.PositionOrder{
			ID:           o.ID,
			PositionID:   o.PositionID,
			Timestamp:    o.Timestamp,
			TimestampStr: o.TimestampStr,
			Side:         o.Side,
			OrderType:    o.OrderType,
			Price:        o.Price,
			Quantity:     o.Quantity,
			SignalID:     o.SignalID,
			CreatedAt:    o.CreatedAt,
		}
	}

	return &models.PositionDetail{
		Position: *toModelPosition(body.eyebrokerPosition),
		Orders:   orders,
	}, nil
}
