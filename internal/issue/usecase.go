package issue

import (
	"context"
	"fmt"
	"time"
)

type Usecase struct {
	repo Repository
}

func NewUsecase(repo Repository) *Usecase {
	return &Usecase{
		repo: repo,
	}
}

type CreateInput struct {
	RequestID string
	KeyID     int64
}

func (uc *Usecase) Create(
	ctx context.Context,
	input CreateInput,
	now time.Time,
) (Issue, error) {
	issue := NewIssue(input.RequestID, input.KeyID, now)
	if err := uc.repo.Create(ctx, issue); err != nil {
		return Issue{}, fmt.Errorf("create: %w", err)
	}

	return issue, nil
}
