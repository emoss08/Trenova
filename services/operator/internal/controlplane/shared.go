package controlplane

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

type SharedDesired struct {
	List instanceapi.SharedTenantList
	ETag string
}

func (c *Client) SharedTenants(ctx context.Context, etag string) (SharedDesired, bool, error) {
	params := &instanceapi.ListSharedTenantsParams{}
	if etag != "" {
		params.IfNoneMatch = &etag
	}
	res, err := c.api.ListSharedTenantsWithResponse(ctx, params)
	if err != nil {
		return SharedDesired{}, false, fmt.Errorf("get the shared tenants: %w", err)
	}
	switch res.StatusCode() {
	case http.StatusOK:
		if res.JSON200 == nil {
			return SharedDesired{}, false, errors.New("get the shared tenants: empty body")
		}
		return SharedDesired{
			List: *res.JSON200,
			ETag: res.HTTPResponse.Header.Get("ETag"),
		}, true, nil
	case http.StatusNotModified:
		return SharedDesired{}, false, nil
	default:
		return SharedDesired{}, false, failure(
			res.HTTPResponse,
			res.ApplicationproblemJSON401,
			res.ApplicationproblemJSON403,
			res.ApplicationproblemJSON429,
		)
	}
}

func (c *Client) ReportSharedStatus(
	ctx context.Context,
	report *instanceapi.SharedStatusReport,
) (instanceapi.SharedReportAck, error) {
	res, err := c.api.ReportSharedStatusWithResponse(ctx, *report)
	if err != nil {
		return instanceapi.SharedReportAck{}, fmt.Errorf(
			"report the shared server's status: %w",
			err,
		)
	}
	if res.StatusCode() == http.StatusOK && res.JSON200 != nil {
		return *res.JSON200, nil
	}
	return instanceapi.SharedReportAck{}, failure(
		res.HTTPResponse,
		res.ApplicationproblemJSON400,
		res.ApplicationproblemJSON401,
		res.ApplicationproblemJSON403,
		res.ApplicationproblemJSON429,
	)
}

func (c *Client) SharedStorage(ctx context.Context) (instanceapi.SharedStorage, error) {
	res, err := c.api.GetSharedStorageWithResponse(ctx)
	if err != nil {
		return instanceapi.SharedStorage{}, fmt.Errorf("get the archive storage: %w", err)
	}
	if res.StatusCode() == http.StatusOK && res.JSON200 != nil {
		return *res.JSON200, nil
	}
	return instanceapi.SharedStorage{}, failure(
		res.HTTPResponse,
		res.ApplicationproblemJSON401,
		res.ApplicationproblemJSON403,
		res.ApplicationproblemJSON429,
	)
}

func (c *Client) ReportSharedUsage(
	ctx context.Context,
	report instanceapi.SharedUsageReport,
) (instanceapi.SharedUsageAck, error) {
	res, err := c.api.ReportSharedUsageWithResponse(ctx, report)
	if err != nil {
		return instanceapi.SharedUsageAck{}, fmt.Errorf("report the tenants' usage: %w", err)
	}
	if res.StatusCode() == http.StatusOK && res.JSON200 != nil {
		return *res.JSON200, nil
	}
	return instanceapi.SharedUsageAck{}, failure(
		res.HTTPResponse,
		res.ApplicationproblemJSON400,
		res.ApplicationproblemJSON401,
		res.ApplicationproblemJSON403,
		res.ApplicationproblemJSON429,
	)
}
