package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url, bytes.NewReader(b))
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

type httpError struct {
	status int
}

// Error returns a human-readable HTTP audit error.
func (e *httpError) Error() string {
	return "audit http status: " + http.StatusText(e.status)
}
