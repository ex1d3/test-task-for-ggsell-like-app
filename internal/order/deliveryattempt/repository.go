package deliveryattempt

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
	Statuses        []Status
	ProcessAfterLte time.Time
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
	UpdatedAt time.Time
}

var (
	ErrAlreadyExists = errors.New("already exists")
	ErrNotFound      = errors.New("not found")
)

type Repository interface {
	List(ctx context.Context, input ListInput) ([]DeliveryAttempt, error)
	Create(ctx context.Context, a DeliveryAttempt) (int64, error)
	Update(ctx context.Context, input UpdateInput) (int64, error)
}
