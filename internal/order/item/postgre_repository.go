package item

import (
	"context"
	"errors"
	"fmt"
	"gg-sell-like-core/pkg/transactor"
	"github.com/jackc/pgx/v5/pgconn"
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

func (r *PostgreRepository) List(
	ctx context.Context,
	input ListInput,
) ([]Item, error) {
	where := make([]string, 0, 2)
	args := make([]any, 0, 4)

	if input.Filter.OrderID != 0 {
		args = append(args, input.Filter.OrderID)
		where = append(where, fmt.Sprintf(
			"order_id = $%d",
			len(args),
		))
	}

	if len(input.Filter.Statuses) > 0 {
		statuses := make([]string, len(input.Filter.Statuses))
		for i, status := range input.Filter.Statuses {
			statuses[i] = string(status)
		}

		args = append(args, statuses)
		where = append(where, fmt.Sprintf(
			"status = ANY($%d)",
			len(args),
		))
	}

	query := `
		SELECT
		    id,
			order_id,
			sku,
			amount,
			code,
			status,
			created_at,
			updated_at
		FROM order_items
	`

	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	query += " ORDER BY created_at, id"

	if input.Pagination.Limit > 0 {
		args = append(args, input.Pagination.Limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	if input.Pagination.Skip > 0 {
		args = append(args, input.Pagination.Skip)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := transactor.SelectExecutor(ctx, r.db).Query(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list order items: %w", err)
	}
	defer rows.Close()

	items := make([]Item, 0, input.Pagination.Limit)

	for rows.Next() {
		var i Item
		var code pgtype.Text

		if err := rows.Scan(
			&i.ID,
			&i.OrderID,
			&i.SKU,
			&i.Amount,
			&code,
			&i.Status,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}

		if code.Valid {
			i.Code = code.String
		}

		items = append(items, i)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order items: %w", err)
	}

	return items, nil
}

func (r *PostgreRepository) Count(ctx context.Context, input CountInput) (int64, error) {
	where := make([]string, 0, 2)
	args := make([]any, 0, 2)

	if input.Filter.OrderID != 0 {
		args = append(args, input.Filter.OrderID)
		where = append(where, fmt.Sprintf(
			"order_id = $%d",
			len(args),
		))
	}

	query := `
		SELECT COUNT(*)
		FROM order_items
	`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	var total int64
	if err := transactor.SelectExecutor(ctx, r.db).QueryRow(
		ctx,
		query,
		args...,
	).Scan(&total); err != nil {
		return 0, fmt.Errorf("count order items: %w", err)
	}

	return total, nil
}

func (r *PostgreRepository) Create(ctx context.Context, i Item) (int64, error) {
	var id int64
	if err := transactor.SelectExecutor(ctx, r.db).QueryRow(ctx, `
		INSERT INTO order_items (
		    order_id,
		    sku,
		    amount,
		    status,
		    created_at,
		    updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`,
		i.OrderID,
		i.SKU,
		i.Amount,
		i.Status,
		i.CreatedAt,
		i.UpdatedAt,
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("scan: %w", err)
	}

	return id, nil
}

func (r *PostgreRepository) Update(
	ctx context.Context,
	input UpdateInput,
) (int64, error) {
	set := make([]string, 0, 3)
	where := make([]string, 0, 2)
	args := make([]any, 0, 5)

	if input.Data.Status != StatusUnknown {
		args = append(args, input.Data.Status)
		set = append(set, fmt.Sprintf(
			"status = $%d",
			len(args),
		))
	}
	if input.Data.Code != "" {
		args = append(args, input.Data.Code)
		set = append(set, fmt.Sprintf(
			"code = $%d",
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
		UPDATE order_items
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
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			return 0, ErrCodeAlreadyUsed
		default:
			return 0, fmt.Errorf("exec: %w", err)
		}
	}

	affected := tag.RowsAffected()

	if affected == 0 {
		return 0, nil
	}

	return tag.RowsAffected(), nil
}
