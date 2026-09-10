package item

import (
	"context"
	"time"
)

type ListInput struct {
	Filter     ListFilter
	Pagination ListPagination
}

type ListFilter struct {
	OrderID int64
}

type ListPagination struct {
	Limit int64
	Skip  int64
}

type CountInput struct {
	Filter CountFilter
}

type CountFilter struct {
	OrderID int64
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

type Repository interface {
	List(ctx context.Context, input ListInput) ([]Item, error)
	Count(ctx context.Context, input CountInput) (int64, error)
	Create(ctx context.Context, i Item) (int64, error)
	Update(ctx context.Context, input UpdateInput) (int64, error)
}
