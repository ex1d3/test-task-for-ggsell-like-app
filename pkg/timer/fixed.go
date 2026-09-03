package timer

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Fixed struct {
	ctx context.Context
	wg  sync.WaitGroup

	tick func(ctx context.Context) error

	log *slog.Logger

	interval time.Duration
	base     time.Time
}

func NewFixed(
	ctx context.Context,
	tick func(ctx context.Context) error,
	log *slog.Logger,
	interval time.Duration,
	base time.Time,
) *Fixed {
	return &Fixed{
		ctx:      ctx,
		tick:     tick,
		log:      log,
		interval: interval,
		base:     base,
	}
}

func (f *Fixed) Start() {
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()

		nextRun := f.nextTick()
		f.log.Debug("next", "date", nextRun.String())
		timer := time.NewTimer(time.Until(nextRun))
		defer timer.Stop()

		for {
			select {
			case <-timer.C:
				if err := f.tick(f.ctx); err != nil {
					f.log.Error("tick", "err", err)
				}
				nextRun = f.nextTick()
				timer.Reset(time.Until(nextRun))
				f.log.Debug("next", "date", nextRun.String())
			case <-f.ctx.Done():
				return
			}
		}
	}()
	f.log.Debug("started")
}

func (f *Fixed) nextTick() time.Time {
	now := time.Now().UTC()
	delta := now.Sub(f.base)
	n := int64(delta / f.interval)

	return f.base.Add(time.Duration(n+1) * f.interval)
}
