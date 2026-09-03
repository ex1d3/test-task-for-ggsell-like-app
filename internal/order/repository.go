package order

import (
	"context"
	"errors"
	"time"
)

type ListInput struct {
	Filter     ListFilter
	Pagination ListPagination
}

type ListFilter struct {
	Statuses []Status
}

type ListPagination struct {
	Limit int64
	Skip  int64
}

type UpdateInput struct {
	Filter UpdateFilter
	Data   UpdateData
}

type UpdateFilter struct {
	ID     int64
	Status Status
}

type UpdateData struct {
	Status    Status
	Code      string
	UpdatedAt time.Time
}

var (
	ErrNotFound = errors.New("not found")
)

type Repository interface {
	GetByID(ctx context.Context, id int64) (Order, error)
	Create(ctx context.Context, o Order) (int64, error)
	Update(ctx context.Context, input UpdateInput) (int64, error)
}
