package item

import (
	"context"
	"errors"
	"fmt"
	"gg-sell-like-core/pkg/transactor"
	"time"
)

type Usecase struct {
	repo Repository
	tx   transactor.Transactor
}

func NewUsecase(repo Repository, tx transactor.Transactor) *Usecase {
	return &Usecase{
		repo: repo,
		tx:   tx,
	}
}

type ListByOrderInput struct {
	OrderID    int64
	Pagination ListByOrderPagination
}

type ListByOrderPagination struct {
	Skip  int64
	Limit int64
}

type ListOutput struct {
	Items []Item
	Total int64
}

var (
	ErrInvalidSkip  = errors.New("invalid skip")
	ErrInvalidLimit = errors.New("invalid limit")
)

func (uc *Usecase) ListByOrder(
	ctx context.Context,
	input ListByOrderInput,
) (ListOutput, error) {
	if input.Pagination.Skip < 0 {
		return ListOutput{}, ErrInvalidSkip
	}

	if input.Pagination.Limit < 1 {
		return ListOutput{}, ErrInvalidLimit
	}

	var (
		items []Item
		total int64
	)

	if err := uc.tx.WithTransaction(
		ctx,
		transactor.Options{
			IsolationLevel: transactor.IsolationLevelReadCommitted,
			ReadOnly:       true,
		},
		func(ctx context.Context) error {
			var err error

			items, err = uc.repo.List(
				ctx,
				ListInput{
					Filter: ListFilter{
						OrderID: input.OrderID,
					},
					Pagination: ListPagination{
						Skip:  input.Pagination.Skip,
						Limit: input.Pagination.Limit,
					},
				},
			)
			if err != nil {
				return fmt.Errorf("list: %w", err)
			}

			total, err = uc.repo.Count(
				ctx,
				CountInput{
					Filter: CountFilter{
						OrderID: input.OrderID,
					},
				},
			)
			if err != nil {
				return fmt.Errorf("count: %w", err)
			}

			return nil
		},
	); err != nil {
		return ListOutput{}, fmt.Errorf("tx: %w", err)
	}

	return ListOutput{Items: items, Total: total}, nil
}

type CreateInput struct {
	OrderID int64
	SKU     string
	Amount  int64
}

func (uc *Usecase) Create(ctx context.Context, input CreateInput, now time.Time) (Item, error) {
	item := NewItem(input.OrderID, input.SKU, input.Amount, now)
	id, err := uc.repo.Create(ctx, item)
	if err != nil {
		return Item{}, fmt.Errorf("repo: %w", err)
	}
	item.ID = id

	return item, nil
}

var (
	ErrConcurrentModification = errors.New("concurrent modification")
)

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

var (
	ErrNotFound = errors.New("not found")
)

func (uc *Usecase) SetCode(
	ctx context.Context,
	id int64,
	code string,
	now time.Time,
) error {
	affectedCount, err := uc.repo.Update(
		ctx,
		UpdateInput{
			Filter: UpdateFilter{
				ID: id,
			},
			Data: UpdateData{
				Code:      code,
				UpdatedAt: now,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	if affectedCount == 0 {
		return ErrNotFound
	}

	return nil
}
