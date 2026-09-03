package key

import (
	"context"
	"errors"
	"fmt"
	"gg-sell-like-core/pkg/ptr"
	"time"
)

type Usecase struct {
	repo Repository
}

func NewUsecase(repo Repository) *Usecase {
	return &Usecase{repo: repo}
}

var (
	ErrAlreadyUsed = errors.New("already used")
)

func (uc *Usecase) MarkUsed(ctx context.Context, id int64, now time.Time) error {
	key, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}

	if key.Used {
		return ErrAlreadyUsed
	}

	affected, err := uc.repo.Update(
		ctx,
		UpdateInput{
			Filter: UpdateFilter{
				ID:   id,
				Used: ptr.Ptr(false),
			},
			Data: UpdateData{
				Used:      ptr.Ptr(true),
				UpdatedAt: now,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	if affected == 0 {
		return ErrAlreadyUsed
	}

	return nil
}
