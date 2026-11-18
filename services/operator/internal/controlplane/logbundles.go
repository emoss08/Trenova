package controlplane

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

func (c *Client) RegisterLogBundle(
	ctx context.Context,
	reg *instanceapi.LogBundleRegistration,
) (uuid.UUID, error) {
	res, err := c.api.RegisterLogBundleWithResponse(ctx, *reg)
	if err != nil {
		return uuid.Nil, fmt.Errorf("register a log bundle: %w", err)
	}
	if res.StatusCode() == http.StatusCreated {
		if res.JSON201 == nil {
			return uuid.Nil, errors.New("register a log bundle: empty body")
		}
		return res.JSON201.BundleId, nil
	}
	return uuid.Nil, failure(
		res.HTTPResponse,
		res.ApplicationproblemJSON400,
		res.ApplicationproblemJSON401,
		res.ApplicationproblemJSON409,
		res.ApplicationproblemJSON429,
	)
}

func (c *Client) UpdateLogBundle(
	ctx context.Context,
	id uuid.UUID,
	update instanceapi.LogBundleUpdate,
) error {
	res, err := c.api.UpdateLogBundleWithResponse(ctx, id, update)
	if err != nil {
		return fmt.Errorf("report a log bundle: %w", err)
	}
	if res.StatusCode() == http.StatusNoContent || res.StatusCode() == http.StatusNotFound {
		return nil
	}
	return failure(
		res.HTTPResponse,
		res.ApplicationproblemJSON400,
		res.ApplicationproblemJSON401,
		res.ApplicationproblemJSON429,
	)
}
