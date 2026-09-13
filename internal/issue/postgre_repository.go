package issue

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

func (r *PostgreRepository) GetByRequest(
	ctx context.Context,
	requestID string,
) (Issue, error) {
	var issue Issue
	if err := transactor.SelectExecutor(ctx, r.db).QueryRow(ctx, `
		SELECT 
		    request_id,
		    key_id,
		    created_at
		FROM issues
		WHERE request_id = $1
		LIMIT 1
	`, requestID,
	).Scan(
		&issue.RequestID,
		&issue.KeyID,
		&issue.CreatedAt,
	); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return Issue{}, ErrNotFound
		default:
			return Issue{}, fmt.Errorf("query rows: %w", err)
		}
	}

	return issue, nil
}

func (r *PostgreRepository) Create(ctx context.Context, i Issue) error {
	if _, err := transactor.SelectExecutor(ctx, r.db).Exec(ctx, `
		INSERT INTO issues (
		    request_id,
		    key_id,
		    created_at
		)
		VALUES ($1, $2, $3)
		ON CONFLICT (request_id) DO NOTHING
	`,
		i.RequestID,
		i.KeyID,
		i.CreatedAt,
	); err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	return nil
}
