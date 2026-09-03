package product

import (
	"context"
	"errors"
)

var (
	ErrAlreadyExists = errors.New("already exists")
	ErrNotFound      = errors.New("not found")
)

type CountInput struct{}

type Repository interface {
	List(ctx context.Context, input ListInput) ([]Product, error)
	GetBySKU(ctx context.Context, sku string) (Product, error)
	Count(ctx context.Context, input CountInput) (int64, error)
	Create(ctx context.Context, p Product) error
}
