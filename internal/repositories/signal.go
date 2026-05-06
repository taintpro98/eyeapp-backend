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

type SignalFilter struct {
	MarketID int
	Limit    int
	Offset   int
	Symbol   string
}

type SignalListResult struct {
	Total int
	Items []*models.Signal
}

type SignalRepository interface {
	List(ctx context.Context, f SignalFilter) (*SignalListResult, error)
}

type eyebrokerClient struct {
	baseURL string
	http    *http.Client
}

func NewSignalHTTPRepository(baseURL string) SignalRepository {
	return &eyebrokerClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// eyebrokerResponse matches the eyebroker API JSON shape.
type eyebrokerResponse struct {
	Total int               `json:"total"`
	Items []eyebrokerSignal `json:"items"`
}

type eyebrokerSignal struct {
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

func (c *eyebrokerClient) List(ctx context.Context, f SignalFilter) (*SignalListResult, error) {
	u, err := url.Parse(c.baseURL + "/v1/api/market/" + strconv.Itoa(f.MarketID) + "/signals")
	if err != nil {
		return nil, fmt.Errorf("eyebroker: parse url: %w", err)
	}
	q := u.Query()
	q.Set("limit", strconv.Itoa(f.Limit))
	q.Set("offset", strconv.Itoa(f.Offset))
	if f.Symbol != "" {
		q.Set("symbol", f.Symbol)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("eyebroker: create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eyebroker: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eyebroker: unexpected status %d", resp.StatusCode)
	}

	var body eyebrokerResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("eyebroker: decode response: %w", err)
	}

	items := make([]*models.Signal, len(body.Items))
	for i, s := range body.Items {
		items[i] = &models.Signal{
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

	return &SignalListResult{Total: body.Total, Items: items}, nil
}
