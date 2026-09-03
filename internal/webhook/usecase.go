package webhook

import (
	"context"
	"errors"
	"fmt"
	"gg-sell-like-core/internal/order"
	"gg-sell-like-core/pkg/currency"
	"gg-sell-like-core/pkg/transactor"
	"log/slog"
	"time"
)

type OrderReader interface {
	Exists(ctx context.Context, id int64) (bool, error)
}

type OrderWriter interface {
	HandleWebhookStatusUpdate(
		ctx context.Context,
		id int64,
		webhookStatus order.WebhookStatus,
		now time.Time,
	) error
}

type Usecase struct {
	log *slog.Logger

	tx          transactor.Transactor
	repo        Repository
	orderReader OrderReader
	orderWriter OrderWriter
}

func NewUsecase(
	log *slog.Logger,
	tx transactor.Transactor,
	repo Repository,
	orderReader OrderReader,
	orderWriter OrderWriter,
) *Usecase {
	return &Usecase{
		log: log.With(
			"component", "webhook",
		),
		tx:          tx,
		repo:        repo,
		orderReader: orderReader,
		orderWriter: orderWriter,
	}
}

var (
	ErrOrderNotFound   = errors.New("order not found")
	ErrInvalidStatus   = errors.New("invalid status")
	ErrInvalidCurrency = errors.New("invalid currency")
)

type HandleInput struct {
	EventID  string
	OrderID  int64
	Status   Status
	Amount   int64
	Currency currency.Currency
}

func (uc *Usecase) Handle(ctx context.Context, w HandleInput, now time.Time) error {
	if !w.Status.IsValid() {
		return ErrInvalidStatus
	}
	if !w.Currency.IsValid() {
		return ErrInvalidCurrency
	}

	return uc.tx.WithTransaction(
		ctx,
		transactor.Options{
			IsolationLevel: transactor.IsolationLevelReadCommitted,
		},
		func(ctx context.Context) error {
			exists, err := uc.orderReader.Exists(ctx, w.OrderID)
			if err != nil {
				uc.log.Info("order doesnt exist", "event_id", w.EventID, "order_id", w.OrderID)
				return fmt.Errorf("order exists: %w", err)
			}

			if !exists {
				return ErrOrderNotFound
			}

			if _, err := uc.repo.Create(
				ctx,
				NewWebhook(
					w.EventID,
					w.OrderID,
					w.Status,
					w.Amount,
					w.Currency,
					now,
				),
			); err != nil {
				switch {
				case errors.Is(err, ErrAlreadyExists):
					uc.log.Info("webhook already processed", "event_id", w.EventID, "order_id", w.OrderID)
					return nil
				}
				return fmt.Errorf("create: %w", err)
			}

			if err := uc.orderWriter.HandleWebhookStatusUpdate(
				ctx,
				w.OrderID,
				StatusToOrderWebhookStatus(w.Status),
				now,
			); err != nil {
				switch {
				case errors.Is(err, order.ErrCannotHandleWebhook):
					uc.log.Info("webhook is out of order", "event_id", w.EventID, "order_id", w.OrderID)
					return nil
				default:
					return fmt.Errorf("handle webhook status update: %w", err)
				}
			}

			uc.log.Info("webhook handled", "event_id", w.EventID, "order_id", w.OrderID)

			return nil
		},
	)
}
