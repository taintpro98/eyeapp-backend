package models

type Signal struct {
	ID           int64   `json:"id"`
	Symbol       string  `json:"symbol"`
	MarketID     int     `json:"market_id"`
	Timestamp    int64   `json:"timestamp"`     // Unix epoch seconds (UTC)
	TimestampStr string  `json:"timestamp_str"` // GMT+7 "YYYY-MM-DD HH:MM:SS"
	Side         string  `json:"side"`          // "buy" | "sell"
	SignalType   string  `json:"signal_type"`
	MainPosition string  `json:"main_position"`
	Price        float64 `json:"price"`
	Quantity     float64 `json:"quantity"`
	Confidence   float64 `json:"confidence"`
	CandleID     *int64  `json:"candle_id"`
}
