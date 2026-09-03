package webhook

import (
	"gg-sell-like-core/internal/order"
	"gg-sell-like-core/pkg/currency"
	"time"
)

type Webhook struct {
	ID        int64
	EventID   string
	OrderID   int64
	Status    Status
	Amount    int64
	Currency  currency.Currency
	CreatedAt time.Time
}

func NewWebhook(
	eventID string,
	orderID int64,
	status Status,
	amount int64,
	curr currency.Currency,
	now time.Time,
) Webhook {
	return Webhook{
		EventID:   eventID,
		OrderID:   orderID,
		Status:    status,
		Amount:    amount,
		Currency:  curr,
		CreatedAt: now,
	}
}

type Status string

const (
	StatusUnknown Status = ""
	StatusPaid    Status = "paid"
	StatusFailed  Status = "failed"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusPaid, StatusFailed:
		return true
	default:
		return false
	}
}

func StatusToOrderWebhookStatus(s Status) order.WebhookStatus {
	switch s {
	case StatusPaid:
		return order.WebhookStatusPaid
	case StatusFailed:
		return order.WebhookStatusFailed
	default:
		panic("unknown status: " + s)
	}
}
