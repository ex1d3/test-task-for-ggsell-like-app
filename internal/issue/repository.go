package issue

import (
	"context"
	"errors"
)

var (
	ErrAlreadyExists = errors.New("already exists")
	ErrNotFound      = errors.New("not found")
)

type Repository interface {
	GetByRequest(ctx context.Context, requestID string) (Issue, error)
	Create(ctx context.Context, i Issue) error
}
