package models

import "time"

type Signal struct {
	ID           int64      `json:"id"`
	Symbol       string     `json:"symbol"`
	Timestamp    int64      `json:"timestamp"`      // UTC epoch ms
	TimestampStr string     `json:"timestamp_str"`  // GMT+7 string
	Side         string     `json:"side"`           // "buy" | "sell"
	SignalType   string     `json:"signal_type"`
	MainPosition string     `json:"main_position"`
	Price        float64    `json:"price"`
	Quantity     float64    `json:"quantity"`
	CandleID     *int64     `json:"candle_id"`
	CreatedAt    time.Time  `json:"created_at"`
}
