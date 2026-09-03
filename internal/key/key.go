package key

import "time"

type Key struct {
	ID        int64
	SKU       string
	Value     string
	Used      bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewKey(sku string, value string, now time.Time) Key {
	return Key{
		SKU:       sku,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
