package audit

import (
	"context"
	"time"
)

type Event struct {
	TS     int64  `json:"ts"`
	Action string `json:"action"`
	UserID string `json:"user_id,omitempty"`
	URL    string `json:"url"`
}

type Observer interface {
	Notify(ctx context.Context, e Event) error
}

type Notifier struct {
	observers []Observer
}

func NewNotifier(observers ...Observer) *Notifier {
	return &Notifier{observers: observers}
}

func (n *Notifier) Enabled() bool {
	return n != nil && len(n.observers) > 0
}

func (n *Notifier) NotifyAllAsync(ctx context.Context, e Event) {
	if n == nil || len(n.observers) == 0 {
		return
	}

	if e.TS == 0 {
		e.TS = time.Now().Unix()
	}

	for _, obs := range n.observers {
		o := obs
		go func() {
			_ = o.Notify(ctx, e)
		}()
	}
}