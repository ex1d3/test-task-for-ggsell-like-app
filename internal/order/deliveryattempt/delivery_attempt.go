package deliveryattempt

import "time"

type DeliveryAttempt struct {
	ID            int64
	RequestID     string
	OrderID       int64
	Provider      Provider
	Status        Status
	AttemptNumber int64
	ProcessAfter  time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewDeliveryAttempt(
	requestID string,
	orderID int64,
	provider Provider,
	attemptNumber int64,
	processAfter time.Time,
	now time.Time,
) DeliveryAttempt {
	if processAfter.Before(now) {
		panic("process after is before now")
	}

	return DeliveryAttempt{
		RequestID:     requestID,
		OrderID:       orderID,
		Provider:      provider,
		Status:        StatusPending,
		AttemptNumber: attemptNumber,
		ProcessAfter:  processAfter,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func (a *DeliveryAttempt) NextProcessAfter(now time.Time) time.Time {
	var nextBackoff time.Duration
	if backoff := a.ProcessAfter.Sub(a.CreatedAt); backoff == 0 {
		nextBackoff = time.Second * 5
	} else {
		nextBackoff = backoff * 2
	}

	return now.Add(nextBackoff)
}

func (a *DeliveryAttempt) NextAttemptNumber() int64 {
	return a.AttemptNumber + 1
}

type Provider string

const (
	ProviderUnknown Provider = ""
	ProviderA       Provider = "a"
	ProviderB       Provider = "b"
)

var providersSequence = map[Provider]Provider{
	ProviderA: ProviderB,
	ProviderB: ProviderA,
}

func (p Provider) Next() Provider {
	np, ok := providersSequence[p]
	if !ok {
		panic("providers sequence is broken")
	}

	return np
}

type Status string

const (
	StatusUnknown    Status = ""
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusSuccess    Status = "success"
	StatusFailed     Status = "failed"
)
