package models

type Position struct {
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
	InitSignalID  int64    `json:"init_signal_id"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type PositionOrder struct {
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

type PositionDetail struct {
	Position
	Orders []PositionOrder `json:"orders"`
}
