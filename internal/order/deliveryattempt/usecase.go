package deliveryattempt

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Usecase struct {
	repo Repository
}

func NewUsecase(repo Repository) *Usecase {
	return &Usecase{repo: repo}
}

var (
	ErrConcurrentModification = errors.New("concurrent modification")
)

type CreateInput struct {
	RequestID     string
	OrderID       int64
	Provider      Provider
	AttemptNumber int64
	ProcessAfter  time.Time
}

func (uc *Usecase) Create(ctx context.Context, input CreateInput, now time.Time) (DeliveryAttempt, error) {
	attempt := NewDeliveryAttempt(
		input.RequestID,
		input.OrderID,
		input.Provider,
		input.AttemptNumber,
		input.ProcessAfter,
		now,
	)
	id, err := uc.repo.Create(ctx, attempt)
	if err != nil {
		return DeliveryAttempt{}, fmt.Errorf("create: %w", err)
	}

	attempt.ID = id
	return attempt, nil
}

func (uc *Usecase) UpdateStatus(
	ctx context.Context,
	id int64,
	oldStatus Status,
	newStatus Status,
	now time.Time,
) error {
	affectedCount, err := uc.repo.Update(
		ctx,
		UpdateInput{
			Filter: UpdateFilter{
				ID:     id,
				Status: oldStatus,
			},
			Data: UpdateData{
				Status:    newStatus,
				UpdatedAt: now,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	if affectedCount == 0 {
		return ErrConcurrentModification
	}

	return nil
}
