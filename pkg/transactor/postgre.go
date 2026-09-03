package transactor

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgreTransactor struct {
	db *pgxpool.Pool
}

func NewPostgreTransactor(db *pgxpool.Pool) *PostgreTransactor {
	return &PostgreTransactor{
		db: db,
	}
}

func (t *PostgreTransactor) WithTransaction(
	ctx context.Context,
	opts Options,
	fn func(ctx context.Context) error,
) error {
	if _, ok := TxFromContext(ctx); ok {
		return mapPostgreError(fn(ctx))
	}

	tx, err := t.db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   isolationLevelToPGX(opts.IsolationLevel),
		AccessMode: accessModeToPGX(opts.ReadOnly),
	})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	ctx = context.WithValue(ctx, txContextKey{}, tx)

	if err := fn(ctx); err != nil {
		return mapPostgreError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return mapPostgreError(err)
	}

	return nil
}

func TxFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txContextKey{}).(pgx.Tx)
	return tx, ok
}

func isolationLevelToPGX(level IsolationLevel) pgx.TxIsoLevel {
	switch level {
	case IsolationLevelReadCommitted:
		return pgx.ReadCommitted
	case IsolationLevelRepeatableRead:
		return pgx.RepeatableRead
	case IsolationLevelSerializable:
		return pgx.Serializable
	default:
		return ""
	}
}

func accessModeToPGX(readOnly bool) pgx.TxAccessMode {
	if readOnly {
		return pgx.ReadOnly
	}

	return pgx.ReadWrite
}

func mapPostgreError(err error) error {
	var pgErr *pgconn.PgError

	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case "40001",
		"40P01":
		return fmt.Errorf("%w: %v", ErrTransactionConflict, err)

	default:
		return err
	}
}
