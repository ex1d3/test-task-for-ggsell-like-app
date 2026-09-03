package order

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type HttpProvider struct {
	baseURL string
	client  *http.Client
}

func NewHttpProvider(
	baseURL string,
	client *http.Client,
) *HttpProvider {
	if client == nil {
		client = &http.Client{
			Timeout: 5 * time.Second,
		}
	}

	return &HttpProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

type issueRequest struct {
	RequestID string `json:"request_id"`
	SKU       string `json:"sku"`
}

type issueResponse struct {
	Code string `json:"code"`
}

func (p *HttpProvider) Issue(
	ctx context.Context,
	requestID string,
	sku string,
) (string, error) {
	body, err := json.Marshal(issueRequest{
		RequestID: requestID,
		SKU:       sku,
	})
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.baseURL+"/issue",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return "", fmt.Errorf("%w: %v", ErrProviderTimedOut, err)
		}

		return "", fmt.Errorf("%w: do request: %v", ErrProviderIssueFailed, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
	case http.StatusConflict:
		return "", ErrProviderOutOfStock
	case http.StatusInternalServerError:
		return "", ErrProviderIssueFailed
	case http.StatusRequestTimeout:
		return "", ErrProviderTimedOut
	default:
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result issueResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}

	return result.Code, nil
}
