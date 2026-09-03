package order

import (
	"context"
	"errors"
)

var (
	ErrProviderTimedOut    = errors.New("provider timed out")
	ErrProviderOutOfStock  = errors.New("provider out of stock")
	ErrProviderIssueFailed = errors.New("provider issue failed")
)

type Provider interface {
	Issue(ctx context.Context, requestID string, sku string) (string, error)
}
