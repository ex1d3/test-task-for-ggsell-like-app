package order

import "time"

type Order struct {
	ID              int64
	Status          Status
	Amount          int64
	DeliveredAmount int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewOrder(
	amount int64,
	now time.Time,
) Order {
	return Order{
		Status:    StatusCreated,
		Amount:    amount,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (o *Order) CanHandleWebhook() bool {
	switch o.Status {
	case StatusCreated:
		return true
	default:
		return false
	}
}

type Status string

const (
	StatusUnknown            Status = ""
	StatusCreated            Status = "created"
	StatusPaymentFailed      Status = "payment_failed"
	StatusPaid               Status = "paid"
	StatusDelivering         Status = "delivering"
	StatusPartiallyDelivered Status = "partially_delivered"
	StatusDelivered          Status = "delivered"
	StatusDeliveryFailed     Status = "delivery_failed"
)

type WebhookStatus string

const (
	WebhookStatusUnknown WebhookStatus = ""
	WebhookStatusPaid    WebhookStatus = "paid"
	WebhookStatusFailed  WebhookStatus = "failed"
)
