package deliveryattempt

import (
	"context"
	"errors"
	"fmt"
	"gg-sell-like-core/pkg/transactor"
	"github.com/jackc/pgx/v5"
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
) ([]DeliveryAttempt, error) {
	where := make([]string, 0, 2)
	args := make([]any, 0, 4)

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

	if !input.Filter.ProcessAfterLte.IsZero() {
		args = append(args, input.Filter.ProcessAfterLte)
		where = append(where, fmt.Sprintf(
			"process_after <= $%d",
			len(args),
		))
	}

	query := `
		SELECT
			id,
			request_id,
			order_id,
			provider,
			status,
			attempt_number,
			process_after,
			created_at,
			updated_at
		FROM delivery_attempts
	`

	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	query += " ORDER BY created_at, id"

	args = append(args, input.Pagination.Limit)
	query += fmt.Sprintf(" LIMIT $%d", len(args))

	args = append(args, input.Pagination.Skip)
	query += fmt.Sprintf(" OFFSET $%d", len(args))

	rows, err := transactor.SelectExecutor(ctx, r.db).Query(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list delivery attempts: %w", err)
	}
	defer rows.Close()

	attempts := make([]DeliveryAttempt, 0, input.Pagination.Limit)

	for rows.Next() {
		var a DeliveryAttempt

		if err := rows.Scan(
			&a.ID,
			&a.RequestID,
			&a.OrderID,
			&a.Provider,
			&a.Status,
			&a.AttemptNumber,
			&a.ProcessAfter,
			&a.CreatedAt,
			&a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan delivery attempt: %w", err)
		}

		attempts = append(attempts, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate delivery attempts: %w", err)
	}

	return attempts, nil
}

func (r *PostgreRepository) Create(ctx context.Context, a DeliveryAttempt) (int64, error) {
	var id int64
	if err := transactor.SelectExecutor(ctx, r.db).QueryRow(ctx, `
		INSERT INTO delivery_attempts (
		    request_id,
		    order_id,
		    provider,
		    status,
		    attempt_number,
		    process_after,
		    created_at,
		    updated_at
		) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`,
		a.RequestID,
		a.OrderID,
		a.Provider,
		a.Status,
		a.AttemptNumber,
		a.ProcessAfter,
		a.CreatedAt,
		a.UpdatedAt,
	).Scan(&id); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return 0, ErrAlreadyExists
		default:
			return 0, fmt.Errorf("scan: %w", err)
		}
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
		UPDATE delivery_attempts
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
