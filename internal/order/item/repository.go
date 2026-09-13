package item

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
	OrderID  int64
	Statuses []Status
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
	ID      int64
	OrderID int64
	Status  Status
}

type UpdateData struct {
	Status    Status
	Code      string
	UpdatedAt time.Time
}

var (
	ErrCodeAlreadyUsed = errors.New("code already used")
)

type Repository interface {
	List(ctx context.Context, input ListInput) ([]Item, error)
	Count(ctx context.Context, input CountInput) (int64, error)
	Create(ctx context.Context, i Item) (int64, error)
	Update(ctx context.Context, input UpdateInput) (int64, error)
}
