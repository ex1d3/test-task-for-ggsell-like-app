package item

import "time"

type Item struct {
	ID        int64
	OrderID   int64
	SKU       string
	Code      string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewItem(orderID int64, sku string, now time.Time) Item {
	return Item{
		OrderID:   orderID,
		SKU:       sku,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

type Status string

const (
	StatusUnknown Status = ""
	StatusPending Status = "pending"
)
