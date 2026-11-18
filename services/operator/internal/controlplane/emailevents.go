package controlplane

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

func (c *Client) EmailEvents(ctx context.Context) ([]instanceapi.EmailEvent, error) {
	res, err := c.api.ListEmailEventsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("list system email events: %w", err)
	}
	if res.StatusCode() == http.StatusOK && res.JSON200 != nil {
		return res.JSON200.Events, nil
	}
	return nil, failure(
		res.HTTPResponse,
		res.ApplicationproblemJSON401,
		res.ApplicationproblemJSON429,
	)
}

func (c *Client) AckEmailEvents(ctx context.Context, ids []uuid.UUID) error {
	res, err := c.api.AckEmailEventsWithResponse(ctx, instanceapi.EmailEventAck{Ids: ids})
	if err != nil {
		return fmt.Errorf("acknowledge system email events: %w", err)
	}
	if res.StatusCode() == http.StatusNoContent {
		return nil
	}
	return failure(
		res.HTTPResponse,
		res.ApplicationproblemJSON400,
		res.ApplicationproblemJSON401,
		res.ApplicationproblemJSON429,
	)
}
