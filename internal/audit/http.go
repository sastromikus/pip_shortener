package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const httpRetryCount = 3

// HTTPObserver sends audit events to a remote HTTP endpoint using POST with JSON body.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver creates an HTTP-based audit observer.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url: url,
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

// Notify sends one audit event to the configured HTTP endpoint.
func (o *HTTPObserver) Notify(ctx context.Context, e Event) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt < httpRetryCount; attempt++ {
		if attempt > 0 {
			if err := sleepBeforeRetry(ctx, attempt); err != nil {
				return err
			}
		}

		err := o.post(ctx, b)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	return lastErr
}

func (o *HTTPObserver) post(ctx context.Context, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return &httpError{status: resp.StatusCode}
	}
	return nil
}

func sleepBeforeRetry(ctx context.Context, attempt int) error {
	timer := time.NewTimer(time.Duration(attempt) * 100 * time.Millisecond)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type httpError struct {
	status int
}

// Error returns a human-readable HTTP audit error.
func (e *httpError) Error() string {
	return fmt.Sprintf("audit http status: %d %s", e.status, http.StatusText(e.status))
}
