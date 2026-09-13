package main

import (
	"context"
	"fmt"
	"gg-sell-like-core/internal/order"
	orderdeliveryattempt "gg-sell-like-core/internal/order/deliveryattempt"
	orderitem "gg-sell-like-core/internal/order/item"
	orderitemhttp "gg-sell-like-core/internal/order/item/transport/http"
	orderhttp "gg-sell-like-core/internal/order/transport/http"
	ordertimer "gg-sell-like-core/internal/order/transport/timer"
	"gg-sell-like-core/internal/platform"
	"gg-sell-like-core/internal/product"
	producthttp "gg-sell-like-core/internal/product/transport/http"
	"gg-sell-like-core/internal/webhook"
	webhookhttp "gg-sell-like-core/internal/webhook/transport/http"
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

	cfg := platform.NewConfig()
	db, err := platform.NewPostgreSQL(ctx, cfg.DatabaseURL, baseLog)
	if err != nil {
		return fmt.Errorf("init postgresql: %w", err)
	}

	tx := transactor.NewPostgreTransactor(db)
	mux := http.NewServeMux()

	productRepo := product.NewPostgreRepository(db)
	productUC := product.NewUsecase(tx, productRepo)
	productHTTPHandler := producthttp.NewHandler(productUC, baseLog)
	productHTTPHandler.Register(mux)

	orderDeliveryAttemptRepo := orderdeliveryattempt.NewPostgreRepository(db)
	orderDeliveryAttemptUC := orderdeliveryattempt.NewUsecase(orderDeliveryAttemptRepo)

	httpClient := &http.Client{}

	orderItemRepo := orderitem.NewPostgreRepository(db)
	orderItemUC := orderitem.NewUsecase(orderItemRepo, tx)
	orderItemHTTPHandler := orderitemhttp.NewHandler(orderItemUC, baseLog)
	orderItemHTTPHandler.Register(mux)

	orderRepo := order.NewPostgreRepository(db)
	orderStatusUpdater := order.NewStatusUpdater(orderRepo)
	orderProviderA := order.NewHttpProvider(cfg.ProviderAURL, httpClient)
	orderProviderB := order.NewHttpProvider(cfg.ProviderBURL, httpClient)
	orderUC := order.NewUsecase(
		baseLog,
		tx,
		orderRepo,
		orderStatusUpdater,
		productRepo,
		orderItemUC,
		orderDeliveryAttemptUC,
	)
	orderPaymentGateway := order.NewNoopPaymentGateway()
	orderDeliveryProcessor := order.NewDeliveryProcessor(
		baseLog,
		tx,
		orderRepo,
		orderStatusUpdater,
		orderItemRepo,
		orderItemUC,
		orderPaymentGateway,
		orderDeliveryAttemptRepo,
		orderDeliveryAttemptUC,
		orderProviderA,
		orderProviderB,
	)
	orderHTTPHandler := orderhttp.NewHandler(orderUC, baseLog)
	orderHTTPHandler.Register(mux)
	orderTimerHandler := ordertimer.NewHandler(ctx, baseLog, orderDeliveryProcessor)

	webhookRepo := webhook.NewPostgreRepository(db)
	webhookUC := webhook.NewUsecase(
		baseLog,
		tx,
		webhookRepo,
		orderRepo,
		orderUC,
	)
	webhookHTTPHandler := webhookhttp.NewHandler(webhookUC, baseLog)
	webhookHTTPHandler.Register(mux)

	httpServer := platform.NewHttpServer(cfg.HttpAddress, mux)

	orderTimerHandler.Start()

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
