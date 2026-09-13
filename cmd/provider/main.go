package main

import (
	"context"
	"fmt"
	"gg-sell-like-core/internal/issue"
	"gg-sell-like-core/internal/key"
	"gg-sell-like-core/internal/platform"
	"gg-sell-like-core/internal/provider"
	providerhttp "gg-sell-like-core/internal/provider/transport/http"
	"gg-sell-like-core/pkg/transactor"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	baseLog := platform.NewLogger()
	appLog := baseLog.With(
		"component", "app",
	)
	if err := run(baseLog, appLog); err != nil {
		appLog.Error("run", "err", err)
		os.Exit(1)
	}

	appLog.Info("stopped")
}

func run(baseLog *slog.Logger, appLog *slog.Logger) error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	appLog.Info("starting")

	cfg, err := platform.NewProviderConfig()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	db, err := platform.NewPostgreSQL(ctx, cfg.DatabaseURL, baseLog)
	if err != nil {
		return fmt.Errorf("init postgresql: %w", err)
	}

	tx := transactor.NewPostgreTransactor(db)
	mux := http.NewServeMux()

	issueRepo := issue.NewPostgreRepository(db)
	issueUC := issue.NewUsecase(issueRepo)

	keyRepo := key.NewPostgreRepository(db)
	keyUC := key.NewUsecase(keyRepo)

	providerUC := provider.NewProvider(
		baseLog,
		cfg.DoubleIssueRate,
		cfg.FailureRate,
		cfg.TimeoutRate,
		cfg.Timeout,
		keyRepo,
		keyUC,
		issueRepo,
		issueUC,
		tx,
	)
	providerHTTP := providerhttp.NewHandler(providerUC, baseLog)
	providerHTTP.Register(mux)

	httpServer := platform.NewHttpServer(cfg.HttpAddress, mux)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return httpServer.Run(ctx, baseLog)
	})

	appLog.Info("started")

	if err := g.Wait(); err != nil {
		db.Close()
		return err
	}

	db.Close()

	return nil
}
