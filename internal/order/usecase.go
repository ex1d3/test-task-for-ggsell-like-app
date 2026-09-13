package order

import (
	"context"
	"errors"
	"fmt"
	deliveryattemptdomain "gg-sell-like-core/internal/order/deliveryattempt"
	itemdomain "gg-sell-like-core/internal/order/item"
	productdomain "gg-sell-like-core/internal/product"
	"gg-sell-like-core/pkg/ptr"
	"gg-sell-like-core/pkg/transactor"
	"github.com/google/uuid"
	"log/slog"
	"strconv"
	"time"
)

type ProductReader interface {
	GetBySKU(ctx context.Context, sku string) (productdomain.Product, error)
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
		input itemdomain.CreateInput,
		now time.Time,
	) (itemdomain.Item, error)
	UpdateStatus(
		ctx context.Context,
		id int64,
		oldStatus itemdomain.Status,
		newStatus itemdomain.Status,
		now time.Time,
	) error
	SetCode(
		ctx context.Context,
		id int64,
		code string,
		now time.Time,
	) error
}

type PaymentGateway interface {
	Refund(ctx context.Context, orderID int64, amount int64) error
}

type NoopPaymentGateway struct{}

func NewNoopPaymentGateway() *NoopPaymentGateway {
	return &NoopPaymentGateway{}
}

func (g *NoopPaymentGateway) Refund(_ context.Context, _ int64, _ int64) error {
	return nil
}

type Usecase struct {
	log *slog.Logger
	tx  transactor.Transactor

	repo                  Repository
	statusUpdater         *StatusUpdater
	productReader         ProductReader
	itemWriter            ItemWriter
	deliveryAttemptWriter DeliveryAttemptWriter
}

func NewUsecase(
	log *slog.Logger,
	tx transactor.Transactor,
	repo Repository,
	statusUpdater *StatusUpdater,
	productReader ProductReader,
	itemWriter ItemWriter,
	deliveryAttemptWriter DeliveryAttemptWriter,
) *Usecase {
	return &Usecase{
		log:                   log.With("component", "order"),
		tx:                    tx,
		repo:                  repo,
		statusUpdater:         statusUpdater,
		productReader:         productReader,
		itemWriter:            itemWriter,
		deliveryAttemptWriter: deliveryAttemptWriter,
	}
}

type CreateInput struct {
	SKUs []string
}

var (
	ErrInvalidSKU            = errors.New("invalid sku")
	ErrAtLeastOneSKURequired = errors.New("at least one sku required")
)

type itemInfo struct {
	SKU    string
	Amount int64
}

func (uc *Usecase) Create(
	ctx context.Context,
	input CreateInput,
	now time.Time,
) (Order, error) {
	if len(input.SKUs) < 1 {
		return Order{}, ErrAtLeastOneSKURequired
	}

	var order Order
	if err := uc.tx.WithTransaction(
		ctx,
		transactor.Options{
			IsolationLevel: transactor.IsolationLevelReadCommitted,
		},
		func(ctx context.Context) error {
			var totalAmount int64
			items := make([]itemInfo, len(input.SKUs))
			for i, sku := range input.SKUs {

				product, err := uc.productReader.GetBySKU(ctx, sku)
				switch {
				case errors.Is(err, productdomain.ErrNotFound):
					return ErrInvalidSKU
				}
				totalAmount += product.Price
				items[i] = itemInfo{
					SKU:    sku,
					Amount: product.Price,
				}
			}

			order = NewOrder(totalAmount, now)
			orderID, err := uc.repo.Create(ctx, order)
			if err != nil {
				return fmt.Errorf("create: %w", err)
			}
			order.ID = orderID

			for _, item := range items {
				if _, err := uc.itemWriter.Create(
					ctx,
					itemdomain.CreateInput{
						OrderID: orderID,
						SKU:     item.SKU,
						Amount:  item.Amount,
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

		if err := uc.statusUpdater.UpdateStatus(
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

type DeliveryAttemptReader interface {
	List(ctx context.Context, input deliveryattemptdomain.ListInput) ([]deliveryattemptdomain.DeliveryAttempt, error)
}

type ItemReader interface {
	List(ctx context.Context, input itemdomain.ListInput) ([]itemdomain.Item, error)
}

type DeliveryProcessor struct {
	log *slog.Logger
	tx  transactor.Transactor

	repo                  Repository
	statusUpdater         *StatusUpdater
	itemReader            ItemReader
	itemWriter            ItemWriter
	paymentGateway        PaymentGateway
	deliveryAttemptReader DeliveryAttemptReader
	deliveryAttemptWriter DeliveryAttemptWriter
	providerA             Provider
	providerB             Provider
}

func NewDeliveryProcessor(
	log *slog.Logger,
	tx transactor.Transactor,
	repo Repository,
	statusUpdater *StatusUpdater,
	itemReader ItemReader,
	itemWriter ItemWriter,
	paymentGateway PaymentGateway,
	deliveryAttemptReader DeliveryAttemptReader,
	deliveryAttemptWriter DeliveryAttemptWriter,
	providerA Provider,
	providerB Provider,
) *DeliveryProcessor {
	return &DeliveryProcessor{
		log:                   log.With("component", "order.delivery_processor"),
		tx:                    tx,
		repo:                  repo,
		statusUpdater:         statusUpdater,
		itemReader:            itemReader,
		itemWriter:            itemWriter,
		paymentGateway:        paymentGateway,
		deliveryAttemptReader: deliveryAttemptReader,
		deliveryAttemptWriter: deliveryAttemptWriter,
		providerA:             providerA,
		providerB:             providerB,
	}
}

// Deliver is not thread-safe
func (p *DeliveryProcessor) Deliver(ctx context.Context, now time.Time) error {
	attempts, err := p.deliveryAttemptReader.List(
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
		if err := p.processDeliveryAttempt(ctx, attempt, now); err != nil {
			p.log.Error(
				"process delivery attempt",
				"attempt_id", attempt.ID,
				"err", err,
			)
		}
	}

	return nil
}

var (
	errProviderIssuedUsedCode = errors.New("provider issued used key")
)

func (p *DeliveryProcessor) processDeliveryAttempt(
	ctx context.Context,
	attempt deliveryattemptdomain.DeliveryAttempt,
	now time.Time,
) error {
	order, err := p.repo.GetByID(ctx, attempt.OrderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	if err := p.tx.WithTransaction(
		ctx,
		transactor.Options{
			IsolationLevel: transactor.IsolationLevelReadCommitted,
		},
		func(ctx context.Context) error {
			if order.Status == StatusPaid ||
				order.Status == StatusDeliveryFailed {
				if err := p.statusUpdater.UpdateStatus(
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
				if err := p.deliveryAttemptWriter.UpdateStatus(
					ctx,
					attempt.ID,
					attempt.Status,
					deliveryattemptdomain.StatusProcessing,
					now,
				); err != nil {
					return fmt.Errorf("update attempt status to processing: %w", err)
				}
			}
			return nil
		}); err != nil {
		return err
	}

	items, err := p.itemReader.List(
		ctx,
		itemdomain.ListInput{
			Filter: itemdomain.ListFilter{
				OrderID: order.ID,
				Statuses: []itemdomain.Status{
					itemdomain.StatusDelivering,
					itemdomain.StatusPending,
					itemdomain.StatusOutOfStock,
					itemdomain.StatusDeliveryFailed,
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("list items: %w", err)
	}

	attemptIDStr := strconv.FormatInt(attempt.ID, 10)
	provider := p.selectProvider(attempt.Provider)

	var errOccurred bool
	for _, item := range items {
		if err := p.itemWriter.UpdateStatus(
			ctx,
			item.ID,
			item.Status,
			itemdomain.StatusDelivering,
			now,
		); err != nil {
			return fmt.Errorf("update item status to delivering: %w", err)
		}

		key, err := provider.Issue(
			ctx,
			strconv.FormatInt(item.ID, 10)+"-"+attemptIDStr,
			item.SKU,
		)
		if err != nil {
			errOccurred = true
			itemStatus := itemdomain.StatusDeliveryFailed
			switch {
			case errors.Is(err, ErrProviderOutOfStock):
				p.log.Info(
					"provider is out of stock",
					"order_id", order.ID,
					"item_id", item.ID,
					"attempt_id", attempt.ID,
				)
				itemStatus = itemdomain.StatusOutOfStock
			case errors.Is(err, ErrProviderIssueFailed):
				p.log.Info(
					"provider issue failed",
					"order_id", order.ID,
					"item_id", item.ID,
					"attempt_id", attempt.ID,
				)
			case errors.Is(err, ErrProviderTimedOut):
				p.log.Info(
					"provider timed out",
					"order_id", order.ID,
					"item_id", item.ID,
					"attempt_id", attempt.ID,
				)
			default:
				p.log.Info(
					"unknown provider error",
					"order_id", order.ID,
					"item_id", item.ID,
					"attempt_id", attempt.ID,
					"err", err,
				)
			}

			if err := p.itemWriter.UpdateStatus(
				ctx,
				item.ID,
				itemdomain.StatusDelivering,
				itemStatus,
				now,
			); err != nil {
				return fmt.Errorf("update item status on fail: %w", err)
			}

			continue
		}

		if err := p.tx.WithTransaction(
			ctx,
			transactor.Options{},
			func(ctx context.Context) error {
				if err := p.itemWriter.UpdateStatus(
					ctx,
					item.ID,
					itemdomain.StatusDelivering,
					itemdomain.StatusDelivered,
					now,
				); err != nil {
					return fmt.Errorf("update item status to delivered: %w", err)
				}

				if err := p.itemWriter.SetCode(
					ctx,
					item.ID,
					key,
					now,
				); err != nil {
					switch {
					case errors.Is(err, itemdomain.ErrCodeAlreadyUsed):
						p.log.Info(
							"provider issued same code twice",
							"order_id", order.ID,
							"item_id", item.ID,
							"attempt_id", attempt.ID,
						)
						errOccurred = true
						return errProviderIssuedUsedCode
					default:
						return fmt.Errorf("set code to item: %w", err)
					}
				}

				order, err = p.repo.GetByID(ctx, order.ID)
				if err != nil {
					return fmt.Errorf("get order: %w", err)
				}

				if _, err := p.repo.Update(
					ctx,
					UpdateInput{
						Filter: UpdateFilter{
							ID:              order.ID,
							DeliveredAmount: &order.DeliveredAmount,
						},
						Data: UpdateData{
							DeliveredAmount: ptr.Ptr(order.DeliveredAmount + item.Amount),
						},
					},
				); err != nil {
					return fmt.Errorf("update delivered amount: %w", err)
				}

				return nil
			},
		); err != nil {
			switch {
			case errors.Is(err, errProviderIssuedUsedCode):
				continue
			default:
				return err
			}
		}

		p.log.Info(
			"key issued",
			"order_id", order.ID,
			"item_id", item.ID,
			"attempt_id", attempt.ID,
			"key", key,
		)
	}

	if !errOccurred {
		return p.tx.WithTransaction(
			ctx,
			transactor.Options{
				IsolationLevel: transactor.IsolationLevelReadCommitted,
			},
			func(ctx context.Context) error {
				if err := p.deliveryAttemptWriter.UpdateStatus(
					ctx,
					attempt.ID,
					deliveryattemptdomain.StatusProcessing,
					deliveryattemptdomain.StatusSuccess,
					now,
				); err != nil {
					return fmt.Errorf("update delivery attempt status to success: %w", err)
				}

				if err := p.statusUpdater.UpdateStatus(
					ctx,
					order.ID,
					StatusDelivering,
					StatusDelivered,
					now,
				); err != nil {
					return fmt.Errorf("update status to delivered: %w", err)
				}

				p.log.Info(
					"order completed",
					"order_id", order.ID,
					"attempt_id", attempt.ID,
				)

				return nil
			},
		)
	}

	return p.tx.WithTransaction(
		ctx,
		transactor.Options{},
		func(ctx context.Context) error {
			if err := p.deliveryAttemptWriter.UpdateStatus(
				ctx,
				attempt.ID,
				deliveryattemptdomain.StatusProcessing,
				deliveryattemptdomain.StatusFailed,
				now,
			); err != nil {
				return fmt.Errorf("update attempt status to failed: %w", err)
			}

			if attempt.AttemptNumber == 3 {
				orderStatus := StatusDeliveryFailed
				order, err = p.repo.GetByID(ctx, order.ID)
				if err != nil {
					return fmt.Errorf("get order: %w", err)
				}
				if order.DeliveredAmount > 0 {
					orderStatus = StatusPartiallyDelivered
				}

				if err := p.statusUpdater.UpdateStatus(
					ctx,
					order.ID,
					StatusDelivering,
					orderStatus,
					now,
				); err != nil {
					return fmt.Errorf("update status to delivery failed: %w", err)
				}

				refundAmount := order.Amount - order.DeliveredAmount

				if err := p.paymentGateway.Refund(
					ctx,
					order.ID,
					refundAmount,
				); err != nil {
					return fmt.Errorf("refund: %w", err)
				}

				p.log.Info(
					"order refunded",
					"is_partially_delivered", order.DeliveredAmount > 0,
					"order_id", order.ID,
					"attempt_id", attempt.ID,
					"refund_amount", refundAmount,
				)

				return nil
			}

			processAfter := attempt.NextProcessAfter(now)
			_, err := p.deliveryAttemptWriter.Create(ctx, deliveryattemptdomain.CreateInput{
				OrderID:       order.ID,
				Provider:      attempt.Provider.Next(),
				ProcessAfter:  processAfter,
				AttemptNumber: attempt.NextAttemptNumber(),
			}, now)
			if err != nil {
				return fmt.Errorf("create delivery attempt: %w", err)
			}

			p.log.Info(
				"next attempt scheduled",
				"order_id", order.ID,
				"attempt_id", attempt.ID,
				"at", processAfter,
			)

			return nil
		},
	)
}

func (p *DeliveryProcessor) selectProvider(pv deliveryattemptdomain.Provider) Provider {
	switch pv {
	case deliveryattemptdomain.ProviderA:
		return p.providerA
	case deliveryattemptdomain.ProviderB:
		return p.providerB
	default:
		panic("unknown provider: " + pv)
	}
}

type StatusUpdater struct {
	repo Repository
}

func NewStatusUpdater(repo Repository) *StatusUpdater {
	return &StatusUpdater{
		repo: repo,
	}
}

func (u *StatusUpdater) UpdateStatus(
	ctx context.Context,
	id int64,
	oldStatus Status,
	newStatus Status,
	now time.Time,
) error {
	affectedCount, err := u.repo.Update(
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
