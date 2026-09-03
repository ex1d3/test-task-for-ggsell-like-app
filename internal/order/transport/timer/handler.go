package timer

import (
	"context"
	orderdomain "gg-sell-like-core/internal/order"
	"gg-sell-like-core/pkg/timer"
	"log/slog"
	"time"
)

type Handler struct {
	t  *timer.Fixed
	uc *orderdomain.Usecase
}

func NewHandler(
	ctx context.Context,
	log *slog.Logger,
	uc *orderdomain.Usecase,
) *Handler {
	return &Handler{
		uc: uc,
		t: timer.NewFixed(
			ctx,
			func(ctx context.Context) error {
				return uc.Deliver(ctx, time.Now().UTC())
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
