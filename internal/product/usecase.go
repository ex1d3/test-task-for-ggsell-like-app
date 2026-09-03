package product

import (
	"context"
	"errors"
	"fmt"
	"gg-sell-like-core/pkg/transactor"
)

type Usecase struct {
	tx   transactor.Transactor
	repo Repository
}

func NewUsecase(tx transactor.Transactor, repo Repository) *Usecase {
	return &Usecase{
		tx:   tx,
		repo: repo,
	}
}

type ListInput struct {
	Filter     ListFilter
	Pagination ListPagination
}

type ListFilter struct {
}

type ListPagination struct {
	Limit int64
	Skip  int64
}

type ListOutput struct {
	Products []Product
	Total    int64
}

var (
	ErrInvalidSkip  = errors.New("invalid skip")
	ErrInvalidLimit = errors.New("invalid limit")
)

func (uc *Usecase) List(ctx context.Context, input ListInput) (ListOutput, error) {
	if input.Pagination.Skip < 0 {
		return ListOutput{}, ErrInvalidSkip
	}

	if input.Pagination.Limit < 1 {
		return ListOutput{}, ErrInvalidLimit
	}

	var (
		products []Product
		total    int64
	)

	if err := uc.tx.WithTransaction(
		ctx,
		transactor.Options{
			IsolationLevel: transactor.IsolationLevelReadCommitted,
			ReadOnly:       true,
		},
		func(ctx context.Context) error {
			var err error

			products, err = uc.repo.List(ctx, input)
			if err != nil {
				return fmt.Errorf("list: %w", err)
			}

			total, err = uc.repo.Count(ctx, CountInput{})
			if err != nil {
				return fmt.Errorf("count: %w", err)
			}

			return nil
		},
	); err != nil {
		return ListOutput{}, fmt.Errorf("tx: %w", err)
	}

	return ListOutput{Products: products, Total: total}, nil
}
