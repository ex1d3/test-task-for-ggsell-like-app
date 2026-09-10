package order

import (
	"context"
	"errors"
	"fmt"
	"gg-sell-like-core/pkg/transactor"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type PostgreRepository struct {
	db *pgxpool.Pool
}

func NewPostgreRepository(db *pgxpool.Pool) *PostgreRepository {
	return &PostgreRepository{
		db: db,
	}
}

func (r *PostgreRepository) Exists(
	ctx context.Context,
	id int64,
) (bool, error) {
	var exists bool

	if err := transactor.SelectExecutor(ctx, r.db).QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM orders
			WHERE id = $1
		)
	`, id).Scan(&exists); err != nil {
		return false, fmt.Errorf("scan: %w", err)
	}

	return exists, nil
}

func (r *PostgreRepository) GetByID(
	ctx context.Context,
	id int64,
) (Order, error) {
	var order Order
	var code pgtype.Text
	if err := transactor.SelectExecutor(ctx, r.db).QueryRow(ctx, `
		SELECT 
		    id,
		    status,
		    amount,
		    created_at,
		    updated_at
		FROM orders
		WHERE id = $1
		LIMIT 1
	`, id,
	).Scan(
		&order.ID,
		&order.Status,
		&order.Amount,
		&order.CreatedAt,
		&order.UpdatedAt,
	); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return Order{}, ErrNotFound
		default:
			return Order{}, fmt.Errorf("scan: %w", err)
		}
	}

	if code.Valid {
		order.Code = code.String
	}

	return order, nil
}

func (r *PostgreRepository) Create(ctx context.Context, o Order) (int64, error) {
	var id int64
	if err := transactor.SelectExecutor(ctx, r.db).QueryRow(ctx, `
		INSERT INTO orders (
		    status,
		    amount,
		    created_at,
		    updated_at
		) 
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`,
		o.Status,
		o.Amount,
		o.CreatedAt,
		o.UpdatedAt,
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("scan: %w", err)
	}

	return id, nil
}

func (r *PostgreRepository) Update(
	ctx context.Context,
	input UpdateInput,
) (int64, error) {
	set := make([]string, 0, 2)
	where := make([]string, 0, 2)
	args := make([]any, 0, 4)

	if input.Data.Status != StatusUnknown {
		args = append(args, input.Data.Status)
		set = append(set, fmt.Sprintf(
			"status = $%d",
			len(args),
		))
	}
	if !input.Data.UpdatedAt.IsZero() {
		args = append(args, input.Data.UpdatedAt)
		set = append(set, fmt.Sprintf(
			"updated_at = $%d",
			len(args),
		))
	}

	if len(set) == 0 {
		return 0, nil
	}

	if input.Filter.ID != 0 {
		args = append(args, input.Filter.ID)
		where = append(where, fmt.Sprintf(
			"id = $%d",
			len(args),
		))
	}
	if input.Filter.Status != StatusUnknown {
		args = append(args, input.Filter.Status)
		where = append(where, fmt.Sprintf(
			"status = $%d",
			len(args),
		))
	}

	query := fmt.Sprintf(`
		UPDATE orders
		SET %s
		WHERE %s
	`,
		strings.Join(set, ", "),
		strings.Join(where, " AND "),
	)

	tag, err := transactor.SelectExecutor(ctx, r.db).Exec(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return 0, fmt.Errorf("exec: %w", err)
	}

	affected := tag.RowsAffected()

	if affected == 0 {
		return 0, nil
	}

	return tag.RowsAffected(), nil
}
