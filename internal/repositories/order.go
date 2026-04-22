package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/alumieye/eyeapp-backend/internal/models"
	"github.com/alumieye/eyeapp-backend/pkg/db"
)

type OrderFilter struct {
	Symbol    string
	Side      string
	OrderType string
	FromMS    *int64 // UTC epoch ms, inclusive
	ToMS      *int64 // UTC epoch ms, inclusive
	// Cursor: last seen (timestamp ms, id) for newest→oldest pagination
	CursorTS *int64
	CursorID *int64
	Limit    int // must be > 0
}

type OrderRepository interface {
	List(ctx context.Context, f OrderFilter) ([]*models.Order, error)
}

type orderPostgres struct {
	db *sql.DB
}

func NewOrderRepository(database *db.EyebrokerDB) OrderRepository {
	return &orderPostgres{db: database.DB}
}

func (r *orderPostgres) List(ctx context.Context, f OrderFilter) ([]*models.Order, error) {
	args := []interface{}{}
	conds := []string{}
	n := 1

	base := `
		SELECT o.id, a.symbol, o.timestamp, o.timestamp_str, o.side, o.order_type,
		       o.main_position, o.price, o.quantity, o.candle_id, o.created_at
		FROM orders o
		JOIN assets a ON a.id = o.asset_id`

	if f.Symbol != "" {
		conds = append(conds, fmt.Sprintf("a.symbol = $%d", n))
		args = append(args, f.Symbol)
		n++
	}
	if f.Side != "" {
		conds = append(conds, fmt.Sprintf("o.side = $%d", n))
		args = append(args, f.Side)
		n++
	}
	if f.OrderType != "" {
		conds = append(conds, fmt.Sprintf("o.order_type = $%d", n))
		args = append(args, f.OrderType)
		n++
	}
	if f.FromMS != nil {
		conds = append(conds, fmt.Sprintf("o.timestamp >= $%d", n))
		args = append(args, *f.FromMS)
		n++
	}
	if f.ToMS != nil {
		conds = append(conds, fmt.Sprintf("o.timestamp <= $%d", n))
		args = append(args, *f.ToMS)
		n++
	}
	if f.CursorTS != nil && f.CursorID != nil {
		conds = append(conds, fmt.Sprintf("(o.timestamp < $%d OR (o.timestamp = $%d AND o.id < $%d))", n, n+1, n+2))
		args = append(args, *f.CursorTS, *f.CursorTS, *f.CursorID)
		n += 3
	}

	query := base
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY o.timestamp DESC, o.id DESC LIMIT $%d", n)
	args = append(args, f.Limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		o := &models.Order{}
		if err := rows.Scan(
			&o.ID,
			&o.Symbol,
			&o.Timestamp,
			&o.TimestampStr,
			&o.Side,
			&o.OrderType,
			&o.MainPosition,
			&o.Price,
			&o.Quantity,
			&o.CandleID,
			&o.CreatedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}
