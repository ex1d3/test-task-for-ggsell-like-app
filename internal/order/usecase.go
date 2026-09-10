package order

import (
	"context"
	"errors"
	"fmt"
	deliveryattemptdomain "gg-sell-like-core/internal/order/deliveryattempt"
	"gg-sell-like-core/internal/order/item"
	productdomain "gg-sell-like-core/internal/product"
	"gg-sell-like-core/pkg/transactor"
	"github.com/google/uuid"
	"log/slog"
	"time"
)

type ProductReader interface {
	GetBySKU(ctx context.Context, sku string) (productdomain.Product, error)
}

type DeliveryAttemptReader interface {
	List(ctx context.Context, input deliveryattemptdomain.ListInput) ([]deliveryattemptdomain.DeliveryAttempt, error)
}

type DeliveryAttemptWriter interface {
	Create(
		ctx context.Context,
		input deliveryattemptdomain.CreateInput,
		now time.Time,
	) (deliveryattemptdomain.DeliveryAttempt, error)
	UpdateStatus(
		ctx context.Context,
		id int64,
		oldStatus deliveryattemptdomain.Status,
		newStatus deliveryattemptdomain.Status,
		now time.Time,
	) error
}

type ItemWriter interface {
	Create(
		ctx context.Context,
		input item.CreateInput,
		now time.Time,
	) (item.Item, error)
}

type Usecase struct {
	log *slog.Logger
	tx  transactor.Transactor

	repo                  Repository
	productReader         ProductReader
	itemWriter            ItemWriter
	deliveryAttemptReader DeliveryAttemptReader
	deliveryAttemptWriter DeliveryAttemptWriter

	providerA Provider
	providerB Provider
}

func NewUsecase(
	log *slog.Logger,
	tx transactor.Transactor,
	repo Repository,
	productReader ProductReader,
	itemWriter ItemWriter,
	deliveryAttemptReader DeliveryAttemptReader,
	deliveryAttemptWriter DeliveryAttemptWriter,
	providerA Provider,
	providerB Provider,
) *Usecase {
	return &Usecase{
		log:                   log.With("component", "order"),
		tx:                    tx,
		repo:                  repo,
		productReader:         productReader,
		itemWriter:            itemWriter,
		deliveryAttemptReader: deliveryAttemptReader,
		deliveryAttemptWriter: deliveryAttemptWriter,
		providerA:             providerA,
		providerB:             providerB,
	}
}

type CreateInput struct {
	SKUs []string
}

var (
	ErrInvalidSKU = errors.New("invalid sku")
)

func (uc *Usecase) Create(
	ctx context.Context,
	input CreateInput,
	now time.Time,
) (Order, error) {
	var order Order
	if err := uc.tx.WithTransaction(
		ctx,
		transactor.Options{
			IsolationLevel: transactor.IsolationLevelReadCommitted,
		},
		func(ctx context.Context) error {
			var totalAmount int64
			skus := make(map[string]int64, len(input.SKUs))
			for _, sku := range input.SKUs {
				product, err := uc.productReader.GetBySKU(ctx, sku)
				switch {
				case errors.Is(err, productdomain.ErrNotFound):
					return ErrInvalidSKU
				}
				totalAmount += product.Price
				skus[sku] = product.Price
			}

			order = NewOrder(totalAmount, now)
			orderID, err := uc.repo.Create(ctx, order)
			if err != nil {
				return fmt.Errorf("create: %w", err)
			}
			order.ID = orderID

			for sku, amount := range skus {
				if _, err := uc.itemWriter.Create(
					ctx,
					item.CreateInput{
						OrderID: orderID,
						SKU:     sku,
						Amount:  amount,
					},
					now,
				); err != nil {
					return fmt.Errorf("create item: %w", err)
				}
			}

			return nil
		},
	); err != nil {
		return Order{}, err
	}

	return order, nil
}

func (uc *Usecase) GetByID(
	ctx context.Context,
	id int64,
) (Order, error) {
	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return Order{}, fmt.Errorf("get by id: %w", err)
	}

	return order, nil
}

var (
	ErrConcurrentModification = errors.New("concurrent modification")
	ErrCannotHandleWebhook    = errors.New("cannot handle webhook")
)

func (uc *Usecase) HandleWebhookStatusUpdate(
	ctx context.Context,
	id int64,
	webhookStatus WebhookStatus,
	now time.Time,
) error {
	for i := 0; i < 5; i++ {
		order, err := uc.repo.GetByID(ctx, id)
		if err != nil {
			return fmt.Errorf("get: %w", err)
		}

		if !order.CanHandleWebhook() {
			return ErrCannotHandleWebhook
		}

		var status Status
		switch webhookStatus {
		case WebhookStatusPaid:
			status = StatusPaid
		case WebhookStatusFailed:
			status = StatusPaymentFailed
		default:
			panic("unknown webhook status: " + webhookStatus)
		}

		if err := uc.updateStatus(
			ctx,
			id,
			order.Status,
			status,
			now,
		); err != nil {
			switch {
			case errors.Is(err, ErrConcurrentModification):
				if i != 4 {
					continue
				}
				fallthrough
			default:
				return fmt.Errorf("update status: %w", err)
			}
		}

		if status == StatusPaymentFailed {
			return nil
		}

		if _, err := uc.deliveryAttemptWriter.Create(
			ctx,
			deliveryattemptdomain.CreateInput{
				RequestID:     uuid.NewString(),
				OrderID:       order.ID,
				Provider:      deliveryattemptdomain.ProviderA,
				ProcessAfter:  now,
				AttemptNumber: 1,
			},
			now,
		); err != nil {
			return fmt.Errorf("create delivery attempt: %w", err)
		}

		break
	}

	return nil
}

// Deliver is not thread-safe
func (uc *Usecase) Deliver(ctx context.Context, now time.Time) error {
	attempts, err := uc.deliveryAttemptReader.List(
		ctx,
		deliveryattemptdomain.ListInput{
			Filter: deliveryattemptdomain.ListFilter{
				Statuses: []deliveryattemptdomain.Status{
					deliveryattemptdomain.StatusPending,
					deliveryattemptdomain.StatusProcessing,
				},
				ProcessAfterLte: now,
			},
			Pagination: deliveryattemptdomain.ListPagination{
				Limit: 500,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("list attempts: %w", err)
	}

	for _, attempt := range attempts {
		order, err := uc.repo.GetByID(ctx, attempt.OrderID)
		if err != nil {
			return fmt.Errorf("get order: %w", err)
		}

		if err := uc.tx.WithTransaction(
			ctx,
			transactor.Options{
				IsolationLevel: transactor.IsolationLevelReadCommitted,
			},
			func(ctx context.Context) error {
				if order.Status == StatusPaid ||
					order.Status == StatusOutOfStock ||
					order.Status == StatusDeliveryFailed {
					if err := uc.updateStatus(
						ctx,
						order.ID,
						order.Status,
						StatusDelivering,
						now,
					); err != nil {
						return fmt.Errorf("update status to delivering: %w", err)
					}
				}

				if attempt.Status != deliveryattemptdomain.StatusProcessing {
					if err := uc.deliveryAttemptWriter.UpdateStatus(
						ctx,
						attempt.ID,
						attempt.Status,
						deliveryattemptdomain.StatusProcessing,
						now,
					); err != nil {
						return fmt.Errorf("update attempt status: %w", err)
					}
				}
				return nil
			},
		); err != nil {
			return fmt.Errorf("tx: %w", err)
		}

		var provider Provider
		switch attempt.Provider {
		case deliveryattemptdomain.ProviderA:
			provider = uc.providerA
			uc.log.Info("provider A selected", "order_id", order.ID, "attempt_id", attempt.ID)
		case deliveryattemptdomain.ProviderB:
			provider = uc.providerB
			uc.log.Info("provider B selected", "order_id", order.ID, "attempt_id", attempt.ID)
		default:
			panic("unknown provider: " + attempt.Provider)
		}

		key, err := provider.Issue(ctx, attempt.RequestID, order.SKU)
		if err == nil {
			if err := uc.tx.WithTransaction(
				ctx,
				transactor.Options{
					IsolationLevel: transactor.IsolationLevelReadCommitted,
				},
				func(ctx context.Context) error {
					affectedCount, err := uc.repo.Update(
						ctx,
						UpdateInput{
							Filter: UpdateFilter{
								ID:     order.ID,
								Status: StatusDelivering,
							},
							Data: UpdateData{
								Status:    StatusDelivered,
								Code:      key,
								UpdatedAt: now,
							},
						},
					)
					if err != nil {
						return fmt.Errorf("update: %w", err)
					}

					if affectedCount == 0 {
						return ErrConcurrentModification
					}

					if err := uc.deliveryAttemptWriter.UpdateStatus(
						ctx,
						attempt.ID,
						deliveryattemptdomain.StatusProcessing,
						deliveryattemptdomain.StatusSuccess,
						now,
					); err != nil {
						return fmt.Errorf("update attempt status: %w", err)
					}
					return nil
				},
			); err != nil {
				return fmt.Errorf("tx: %w", err)
			}
			uc.log.Info("key issued", "order_id", order.ID, "attempt_id", attempt.ID, "key", key)
			continue
		}

		switch {
		case errors.Is(err, ErrProviderOutOfStock):
			uc.log.Info("provider is out of stock", "order_id", order.ID, "attempt_id", attempt.ID)
			if err := uc.tx.WithTransaction(
				ctx,
				transactor.Options{
					IsolationLevel: transactor.IsolationLevelReadCommitted,
				},
				func(ctx context.Context) error {
					if err := uc.updateStatus(
						ctx,
						order.ID,
						StatusDelivering,
						StatusOutOfStock,
						now,
					); err != nil {
						return fmt.Errorf("update status to out_of_stock: %w", err)
					}

					if err := uc.deliveryAttemptWriter.UpdateStatus(
						ctx,
						attempt.ID,
						deliveryattemptdomain.StatusProcessing,
						deliveryattemptdomain.StatusFailed,
						now,
					); err != nil {
						return fmt.Errorf("update attempt status: %w", err)
					}

					processAfter := attempt.NextProcessAfter(now)

					if _, err := uc.deliveryAttemptWriter.Create(
						ctx,
						deliveryattemptdomain.CreateInput{
							RequestID:     uuid.NewString(),
							OrderID:       order.ID,
							Provider:      attempt.Provider.Next(),
							ProcessAfter:  processAfter,
							AttemptNumber: attempt.NextAttemptNumber(),
						},
						now,
					); err != nil {
						return fmt.Errorf("create delivery attempt: %w", err)
					}

					uc.log.Info("next attempt scheduled", "order_id", order.ID, "attempt_id", attempt.ID, "at", processAfter)

					return nil
				},
			); err != nil {
				return fmt.Errorf("tx: %w", err)
			}
		case errors.Is(err, ErrProviderIssueFailed):
			uc.log.Info("provider issue failed", "order_id", order.ID, "attempt_id", attempt.ID)

			if err := uc.tx.WithTransaction(
				ctx,
				transactor.Options{
					IsolationLevel: transactor.IsolationLevelReadCommitted,
				},
				func(ctx context.Context) error {
					if err := uc.updateStatus(
						ctx,
						order.ID,
						StatusDelivering,
						StatusDeliveryFailed,
						now,
					); err != nil {
						return fmt.Errorf("update status to delivered: %w", err)
					}

					if err := uc.deliveryAttemptWriter.UpdateStatus(
						ctx,
						attempt.ID,
						deliveryattemptdomain.StatusProcessing,
						deliveryattemptdomain.StatusFailed,
						now,
					); err != nil {
						return fmt.Errorf("update attempt status: %w", err)
					}

					processAfter := attempt.NextProcessAfter(now)

					if _, err := uc.deliveryAttemptWriter.Create(
						ctx,
						deliveryattemptdomain.CreateInput{
							RequestID:     uuid.NewString(),
							OrderID:       order.ID,
							Provider:      attempt.Provider.Next(),
							ProcessAfter:  processAfter,
							AttemptNumber: attempt.NextAttemptNumber(),
						},
						now,
					); err != nil {
						return fmt.Errorf("create delivery attempt: %w", err)
					}

					uc.log.Info("next attempt scheduled", "order_id", order.ID, "attempt_id", attempt.ID, "at", processAfter)

					return nil
				},
			); err != nil {
				return fmt.Errorf("tx: %w", err)
			}
		default:
			switch {
			case errors.Is(err, ErrProviderTimedOut):
				uc.log.Info("provider timed out", "order_id", order.ID, "attempt_id", attempt.ID)
			default:
				uc.log.Info("unknown provider error", "order_id", order.ID, "attempt_id", attempt.ID, "err", err)
			}

			if err := uc.tx.WithTransaction(
				ctx,
				transactor.Options{
					IsolationLevel: transactor.IsolationLevelReadCommitted,
				},
				func(ctx context.Context) error {
					if err := uc.deliveryAttemptWriter.UpdateStatus(
						ctx,
						attempt.ID,
						deliveryattemptdomain.StatusProcessing,
						deliveryattemptdomain.StatusFailed,
						now,
					); err != nil {
						return fmt.Errorf("update attempt status: %w", err)
					}

					processAfter := attempt.NextProcessAfter(now)

					if _, err := uc.deliveryAttemptWriter.Create(
						ctx,
						deliveryattemptdomain.CreateInput{
							RequestID:     attempt.RequestID,
							OrderID:       order.ID,
							Provider:      attempt.Provider,
							ProcessAfter:  processAfter,
							AttemptNumber: attempt.NextAttemptNumber(),
						},
						now,
					); err != nil {
						return fmt.Errorf("create delivery attempt: %w", err)
					}

					uc.log.Info("next attempt scheduled", "order_id", order.ID, "attempt_id", attempt.ID, "at", processAfter)

					return nil
				},
			); err != nil {
				return fmt.Errorf("tx: %w", err)
			}
		}
	}

	return nil
}

func (uc *Usecase) updateStatus(
	ctx context.Context,
	id int64,
	oldStatus Status,
	newStatus Status,
	now time.Time,
) error {
	affectedCount, err := uc.repo.Update(
		ctx,
		UpdateInput{
			Filter: UpdateFilter{
				ID:     id,
				Status: oldStatus,
			},
			Data: UpdateData{
				Status:    newStatus,
				UpdatedAt: now,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	if affectedCount == 0 {
		return ErrConcurrentModification
	}

	return nil
}
