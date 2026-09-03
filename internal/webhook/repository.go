package webhook

import (
	"context"
	"errors"
)

var (
	ErrAlreadyExists = errors.New("already exists")
)

type Repository interface {
	Create(ctx context.Context, w Webhook) (int64, error)
}
