package webhook

import (
	"context"
	"errors"
	"fmt"
	"gg-sell-like-core/pkg/transactor"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgreRepository struct {
	db *pgxpool.Pool
}

func NewPostgreRepository(db *pgxpool.Pool) *PostgreRepository {
	return &PostgreRepository{
		db: db,
	}
}

func (r *PostgreRepository) Create(ctx context.Context, w Webhook) (int64, error) {
	var id int64
	if err := transactor.SelectExecutor(ctx, r.db).QueryRow(ctx, `
		INSERT INTO webhooks (
		    event_id,
		    order_id,
		    status,
		    amount,
		    currency,
		    created_at
		) 
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (event_id) DO NOTHING
		RETURNING id
	`,
		w.EventID,
		w.OrderID,
		w.Status,
		w.Amount,
		w.Currency,
		w.CreatedAt,
	).Scan(&id); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return 0, ErrAlreadyExists
		default:
			return 0, fmt.Errorf("query row: %w", err)
		}
	}

	return id, nil
}
