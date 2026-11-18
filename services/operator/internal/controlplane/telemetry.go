package controlplane

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

func (c *Client) ReportTelemetry(ctx context.Context, report instanceapi.TelemetryReport) error {
	res, err := c.api.ReportTelemetryWithResponse(ctx, report)
	if err != nil {
		return fmt.Errorf("report telemetry: %w", err)
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
