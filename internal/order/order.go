package order

import "time"

type Order struct {
	ID        int64
	Status    Status
	SKU       string
	Amount    int64
	Code      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewOrder(
	sku string,
	amount int64,
	now time.Time,
) Order {
	return Order{
		Status:    StatusCreated,
		SKU:       sku,
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
	StatusUnknown        Status = ""
	StatusCreated        Status = "created"
	StatusPaid           Status = "paid"
	StatusDelivering     Status = "delivering"
	StatusDelivered      Status = "delivered"
	StatusPaymentFailed  Status = "payment_failed"
	StatusOutOfStock     Status = "out_of_stock"
	StatusDeliveryFailed Status = "delivery_failed"
)

type WebhookStatus string

const (
	WebhookStatusUnknown WebhookStatus = ""
	WebhookStatusPaid    WebhookStatus = "paid"
	WebhookStatusFailed  WebhookStatus = "failed"
)
