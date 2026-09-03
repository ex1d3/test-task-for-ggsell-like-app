package product

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

func (r *PostgreRepository) GetBySKU(ctx context.Context, sku string) (Product, error) {
	var p Product
	if err := transactor.SelectExecutor(ctx, r.db).QueryRow(ctx, `
		SELECT 
		    sku,
			name,
			product_type,
			price,
			currency,
			image
		FROM products
		WHERE sku = $1
		LIMIT 1
	`, sku).Scan(
		&p.SKU,
		&p.Name,
		&p.Type,
		&p.Price,
		&p.Currency,
		&p.Image,
	); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return Product{}, ErrNotFound
		default:
			return Product{}, fmt.Errorf("query rows: %w", err)
		}
	}

	return p, nil
}

func (r *PostgreRepository) List(ctx context.Context, input ListInput) ([]Product, error) {
	rows, err := transactor.SelectExecutor(ctx, r.db).Query(ctx, `
		SELECT
			sku,
			name,
			product_type,
			price,
			currency,
			image
		FROM products
		ORDER BY sku
		LIMIT $1 OFFSET $2
	`, input.Pagination.Limit, input.Pagination.Skip,
	)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	products := make([]Product, 0, input.Pagination.Limit)

	for rows.Next() {
		var p Product

		if err := rows.Scan(
			&p.SKU,
			&p.Name,
			&p.Type,
			&p.Price,
			&p.Currency,
			&p.Image,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return products, nil
}

func (r *PostgreRepository) Count(ctx context.Context, _ CountInput) (int64, error) {
	var total int64

	if err := transactor.SelectExecutor(ctx, r.db).QueryRow(ctx, `
		SELECT COUNT(*)
		FROM products
	`).Scan(&total); err != nil {
		return 0, fmt.Errorf("count products: %w", err)
	}

	return total, nil
}

func (r *PostgreRepository) Create(ctx context.Context, p Product) error {
	tag, err := transactor.SelectExecutor(ctx, r.db).Exec(ctx, `
		INSERT INTO products (
		    sku,
		    name,
		    product_type,
		    price,
		    currency,
		    image,
		    created_at
		) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (sku) DO NOTHING
	`, p.SKU, p.Name, p.Type, p.Price, p.Currency, p.Image, p.CreatedAt)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrAlreadyExists
	}

	return nil
}
