package tms

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bytedance/sonic"
)

type EmailEvent struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Payload  string `json:"payload"`
}

func (e EmailEvent) String() string {
	return "system email event " + e.ID + " from " + e.Provider
}

func (e EmailEvent) GoString() string {
	return e.String()
}

func (c *Client) RelayEmailEvents(ctx context.Context, events []EmailEvent) ([]string, error) {
	body, err := sonic.Marshal(struct {
		Events []EmailEvent `json:"events"`
	}{Events: events})
	if err != nil {
		return nil, fmt.Errorf("relay system email events to the TMS: cannot encode %d events", len(events))
	}
	u := c.base.JoinPath("managed", "v1", "system-email", "events")
	out, err := c.do(ctx, http.MethodPost, u, body, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("relay system email events to the TMS: %w", err)
	}
	var answer struct {
		Handled []string `json:"handled"`
	}
	if err = sonic.Unmarshal(out, &answer); err != nil {
		return nil, fmt.Errorf("relay system email events to the TMS: %w", err)
	}
	if answer.Handled == nil {
		answer.Handled = []string{}
	}
	return answer.Handled, nil
}
