package key

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

func (r *PostgreRepository) GetByID(ctx context.Context, id int64) (Key, error) {
	var key Key
	if err := transactor.SelectExecutor(ctx, r.db).QueryRow(ctx, `
		SELECT 
		    id,
		    sku,
		    value,
		    is_used,
		    created_at,
		    updated_at
		FROM keys
		WHERE id = $1
		LIMIT 1
	`, id,
	).Scan(
		&key.ID,
		&key.SKU,
		&key.Value,
		&key.Used,
		&key.CreatedAt,
		&key.UpdatedAt,
	); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return Key{}, ErrNotFound
		default:
			return Key{}, fmt.Errorf("query rows: %w", err)
		}
	}

	return key, nil
}

func (r *PostgreRepository) GetUnusedBySKU(
	ctx context.Context,
	sku string,
) (Key, error) {
	fmt.Println("sku", sku)
	var key Key
	if err := transactor.SelectExecutor(ctx, r.db).QueryRow(ctx, `
		SELECT 
		    id,
		    sku,
		    value,
		    is_used,
		    created_at,
		    updated_at
		FROM keys
		WHERE is_used = false AND sku = $1
		LIMIT 1
	`, sku,
	).Scan(
		&key.ID,
		&key.SKU,
		&key.Value,
		&key.Used,
		&key.CreatedAt,
		&key.UpdatedAt,
	); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return Key{}, ErrNotFound
		default:
			return Key{}, fmt.Errorf("query rows: %w", err)
		}
	}

	return key, nil
}

func (r *PostgreRepository) Create(ctx context.Context, k Key) (int64, error) {
	var id int64
	if err := transactor.SelectExecutor(ctx, r.db).QueryRow(ctx, `
		INSERT INTO keys (
		    sku,
		    value,
		    is_used,
		    created_at,
		    updated_at
		) 
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (value) DO NOTHING
		RETURNING id
	`,
		k.SKU,
		k.Value,
		k.Used,
		k.CreatedAt,
		k.UpdatedAt,
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

	if input.Data.Used != nil {
		args = append(args, input.Data.Used)
		set = append(set, fmt.Sprintf(
			"is_used = $%d",
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
	if input.Filter.Used != nil {
		args = append(args, *input.Filter.Used)
		where = append(where, fmt.Sprintf(
			"is_used = $%d",
			len(args),
		))
	}

	query := fmt.Sprintf(`
		UPDATE keys
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
		return 0, fmt.Errorf("update key: %w", err)
	}

	affected := tag.RowsAffected()

	if affected == 0 {
		return 0, nil
	}

	return tag.RowsAffected(), nil
}
