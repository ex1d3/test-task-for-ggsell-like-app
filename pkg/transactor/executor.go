package transactor

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Executor interface {
	Exec(
		context.Context,
		string,
		...any,
	) (pgconn.CommandTag, error)
	Query(
		context.Context,
		string,
		...any,
	) (pgx.Rows, error)
	QueryRow(
		context.Context,
		string,
		...any,
	) pgx.Row
}

func SelectExecutor(ctx context.Context, pool *pgxpool.Pool) Executor {
	if tx, ok := TxFromContext(ctx); ok {
		return tx
	}

	return pool
}
