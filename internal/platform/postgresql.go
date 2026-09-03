package platform

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
)

func NewPostgreSQL(ctx context.Context, url string, log *slog.Logger) (*pgxpool.Pool, error) {
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("new: %w", err)
	}

	if err := db.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	dbLog := log.With("component", "postgresql")
	dbLog.Info("started")

	return db, nil
}
