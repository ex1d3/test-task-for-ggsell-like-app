package issue

import "time"

type Issue struct {
	RequestID string
	KeyID     int64
	CreatedAt time.Time
}

func NewIssue(
	requestID string,
	keyID int64,
	now time.Time,
) Issue {
	return Issue{
		RequestID: requestID,
		KeyID:     keyID,
		CreatedAt: now,
	}
}
