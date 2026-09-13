package item

import "time"

type Item struct {
	ID        int64
	OrderID   int64
	SKU       string
	Amount    int64
	Code      string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewItem(orderID int64, sku string, amount int64, now time.Time) Item {
	return Item{
		OrderID:   orderID,
		SKU:       sku,
		Amount:    amount,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

type Status string

const (
	StatusUnknown        Status = ""
	StatusPending        Status = "pending"
	StatusDelivering     Status = "delivering"
	StatusDelivered      Status = "delivered"
	StatusOutOfStock     Status = "out_of_stock"
	StatusDeliveryFailed Status = "delivery_failed"
)
