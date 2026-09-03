package key

import (
	"context"
	"errors"
	"time"
)

type UpdateInput struct {
	Filter UpdateFilter
	Data   UpdateData
}

type UpdateFilter struct {
	ID   int64
	Used *bool
}

type UpdateData struct {
	Used      *bool
	UpdatedAt time.Time
}

var (
	ErrAlreadyExists = errors.New("already exists")
	ErrNotFound      = errors.New("not found")
)

type Repository interface {
	GetByID(ctx context.Context, id int64) (Key, error)
	GetUnusedBySKU(ctx context.Context, sku string) (Key, error)
	Create(ctx context.Context, key Key) (int64, error)
	Update(ctx context.Context, input UpdateInput) (int64, error)
}
