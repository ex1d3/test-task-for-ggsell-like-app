package transactor

import (
	"context"
	"errors"
)

type IsolationLevel uint8

const (
	IsolationLevelDefault IsolationLevel = iota
	IsolationLevelReadCommitted
	IsolationLevelRepeatableRead
	IsolationLevelSerializable
)

type Options struct {
	IsolationLevel IsolationLevel
	ReadOnly       bool
}

type Transactor interface {
	WithTransaction(
		ctx context.Context,
		opts Options,
		fn func(ctx context.Context) error,
	) error
}

var (
	ErrTransactionConflict = errors.New("transaction conflict")
)

type txContextKey struct{}
