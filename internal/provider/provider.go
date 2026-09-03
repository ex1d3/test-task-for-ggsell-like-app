package provider

import (
	"context"
	"errors"
	"fmt"
	issuedomain "gg-sell-like-core/internal/issue"
	keydomain "gg-sell-like-core/internal/key"
	"gg-sell-like-core/pkg/transactor"
	"log/slog"
	"math/rand/v2"
	"time"
)

type IssueReader interface {
	GetByRequest(ctx context.Context, requestID string) (issuedomain.Issue, error)
}

type IssueWriter interface {
	Create(ctx context.Context, input issuedomain.CreateInput, now time.Time) (issuedomain.Issue, error)
}

type KeyReader interface {
	GetByID(ctx context.Context, id int64) (keydomain.Key, error)
	GetUnusedBySKU(ctx context.Context, sku string) (keydomain.Key, error)
}

type KeyWriter interface {
	MarkUsed(ctx context.Context, id int64, now time.Time) error
}

type Provider struct {
	log *slog.Logger

	failureRate float64
	timeoutRate float64
	timeout     time.Duration

	keyReader   KeyReader
	keyWriter   KeyWriter
	issueReader IssueReader
	issueWriter IssueWriter

	tx transactor.Transactor
}

func NewProvider(
	log *slog.Logger,
	failureRate float64,
	timeoutRate float64,
	timeout time.Duration,
	keyReader KeyReader,
	keyWriter KeyWriter,
	issueReader IssueReader,
	issueWriter IssueWriter,
	tx transactor.Transactor,
) *Provider {
	return &Provider{
		log:         log.With("component", "provider"),
		failureRate: failureRate,
		timeoutRate: timeoutRate,
		timeout:     timeout,
		keyReader:   keyReader,
		keyWriter:   keyWriter,
		issueReader: issueReader,
		issueWriter: issueWriter,
		tx:          tx,
	}
}

type IssueInput struct {
	RequestID string
	SKU       string
}

var (
	ErrOutOfStock  = errors.New("out of stock")
	ErrTimedOut    = errors.New("timed out")
	ErrIssueFailed = errors.New("issue failed")
)

func (p *Provider) Issue(
	ctx context.Context,
	input IssueInput,
	now time.Time,
) (string, error) {
	p.log.Info("received issue request", "request_id", input.RequestID)
	r := rand.Float64()

	issue, err := p.issueReader.GetByRequest(ctx, input.RequestID)
	if err != nil {
		switch {
		case errors.Is(err, issuedomain.ErrNotFound):
		default:
			return "", fmt.Errorf("get existing issue: %w", err)
		}
	} else {
		key, err := p.keyReader.GetByID(ctx, issue.KeyID)
		if err != nil {
			return "", fmt.Errorf("get key for existing issue: %w", err)
		}

		p.log.Info("existing issue found", "request_id", input.RequestID)
		return key.Value, nil
	}

	if r < p.failureRate {
		p.log.Info("issue failed", "request_id", input.RequestID)
		return "", ErrIssueFailed
	}

	var keyValue string
	// не лучшее решение, но для прототипа подойдет
	for {
		key, err := p.keyReader.GetUnusedBySKU(ctx, input.SKU)
		if err != nil {
			switch {
			case errors.Is(err, keydomain.ErrNotFound):
				return "", ErrOutOfStock
			default:
				return "", fmt.Errorf("get unused key: %w", err)
			}
		}

		if err := p.tx.WithTransaction(
			ctx,
			transactor.Options{
				IsolationLevel: transactor.IsolationLevelReadCommitted,
			},
			func(ctx context.Context) error {
				if err := p.keyWriter.MarkUsed(
					ctx,
					key.ID,
					now,
				); err != nil {
					switch {
					case errors.Is(err, keydomain.ErrAlreadyUsed):
						return nil
					default:
						return fmt.Errorf("mark key used: %w", err)
					}
				}

				if _, err := p.issueWriter.Create(
					ctx,
					issuedomain.CreateInput{
						RequestID: input.RequestID,
						KeyID:     key.ID,
					},
					now,
				); err != nil {
					switch {
					case errors.Is(err, keydomain.ErrAlreadyExists):
						return nil
					default:
						return fmt.Errorf("create issue: %w", err)
					}
				}

				keyValue = key.Value
				p.log.Info("key issued", "request_id", input.RequestID)

				return nil
			},
		); err != nil {
			return "", fmt.Errorf("tx: %w", err)
		}

		if keyValue != "" {
			break
		}
	}

	if r < p.failureRate+p.timeoutRate {
		p.log.Info("timed out", "request_id", input.RequestID)
		time.Sleep(p.timeout)
		return "", ErrTimedOut
	}

	return keyValue, nil
}
