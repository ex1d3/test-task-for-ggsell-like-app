package timer

import (
	"context"
	orderdomain "gg-sell-like-core/internal/order"
	"gg-sell-like-core/pkg/timer"
	"log/slog"
	"time"
)

type Handler struct {
	t *timer.Fixed
	p *orderdomain.DeliveryProcessor
}

func NewHandler(
	ctx context.Context,
	log *slog.Logger,
	p *orderdomain.DeliveryProcessor,
) *Handler {
	return &Handler{
		p: p,
		t: timer.NewFixed(
			ctx,
			func(ctx context.Context) error {
				return p.Deliver(ctx, time.Now().UTC())
			},
			log.With("component", "order.timer"),
			time.Second*5,
			time.Date(
				2026,
				1,
				1,
				0,
				0,
				0,
				0,
				time.UTC,
			),
		),
	}
}

func (h *Handler) Start() {
	h.t.Start()
}
