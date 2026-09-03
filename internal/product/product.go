package product

import (
	"gg-sell-like-core/pkg/currency"
	"time"
)

type Product struct {
	SKU       string
	Name      string
	Type      Type
	Price     int64
	Currency  currency.Currency
	Image     string
	CreatedAt time.Time
}

func NewProduct(
	sku string,
	name string,
	t Type,
	price int64,
	currency currency.Currency,
	image string,
	now time.Time,
) Product {
	return Product{
		SKU:       sku,
		Name:      name,
		Type:      t,
		Price:     price,
		Currency:  currency,
		Image:     image,
		CreatedAt: now,
	}
}

type Type string

const (
	TypeUnknown      Type = ""
	TypeTopUp        Type = "top_up"
	TypeKey          Type = "key"
	TypeSubscription Type = "subscription"
	TypeGiftCard     Type = "gift_card"
)
