package controlplane

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

var ErrUnauthenticated = errors.New("the control plane does not accept this certificate")

type RetryLaterError struct {
	Reason     string
	RetryAfter time.Duration
}

func (e *RetryLaterError) Error() string {
	return fmt.Sprintf("the control plane asks to retry after %s: %s", e.RetryAfter, e.Reason)
}

type ProblemError struct {
	Status int
	Title  string
	Detail string
	Code   string
}

func (e *ProblemError) Error() string {
	msg := fmt.Sprintf("control plane answered %d", e.Status)
	if e.Code != "" {
		msg += " " + e.Code
	}
	if e.Title != "" {
		msg += ": " + e.Title
	}
	if e.Detail != "" {
		msg += " (" + e.Detail + ")"
	}
	return msg
}

type Client struct {
	api *instanceapi.ClientWithResponses
}

func New(baseURL string, doer *http.Client) (*Client, error) {
	api, err := instanceapi.NewClientWithResponses(
		strings.TrimSuffix(baseURL, "/")+"/",
		instanceapi.WithHTTPClient(doer),
	)
	if err != nil {
		return nil, fmt.Errorf("instance API client: %w", err)
	}
	return &Client{api: api}, nil
}

type Desired struct {
	Revision int64
	Spec     instanceapi.InstanceSpec
	ETag     string
}

func (c *Client) Spec(ctx context.Context, etag string) (Desired, bool, error) {
	params := &instanceapi.GetSpecParams{}
	if etag != "" {
		params.IfNoneMatch = &etag
	}
	res, err := c.api.GetSpecWithResponse(ctx, params)
	if err != nil {
		return Desired{}, false, fmt.Errorf("get the spec: %w", err)
	}
	switch res.StatusCode() {
	case http.StatusOK:
		if res.JSON200 == nil {
			return Desired{}, false, errors.New("get the spec: empty body")
		}
		return Desired{
			Revision: res.JSON200.Revision,
			Spec:     res.JSON200.Spec,
			ETag:     res.HTTPResponse.Header.Get("ETag"),
		}, true, nil
	case http.StatusNotModified:
		return Desired{}, false, nil
	default:
		return Desired{}, false, failure(
			res.HTTPResponse,
			res.ApplicationproblemJSON401,
			res.ApplicationproblemJSON429,
			res.ApplicationproblemJSON503,
		)
	}
}

func (c *Client) ReportStatus(ctx context.Context, report instanceapi.StatusReport) error {
	res, err := c.api.ReportStatusWithResponse(ctx, report)
	if err != nil {
		return fmt.Errorf("report status: %w", err)
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

func (c *Client) ReportUsage(
	ctx context.Context,
	report instanceapi.UsageReport,
) (instanceapi.UsageAck, error) {
	res, err := c.api.ReportUsageWithResponse(ctx, report)
	if err != nil {
		return instanceapi.UsageAck{}, fmt.Errorf("report usage: %w", err)
	}
	if res.StatusCode() == http.StatusOK && res.JSON200 != nil {
		return *res.JSON200, nil
	}
	return instanceapi.UsageAck{}, failure(
		res.HTTPResponse,
		res.ApplicationproblemJSON400,
		res.ApplicationproblemJSON401,
		res.ApplicationproblemJSON429,
	)
}

func (c *Client) Secrets(ctx context.Context) (instanceapi.InstanceSecrets, error) {
	res, err := c.api.GetSecretsWithResponse(ctx)
	if err != nil {
		return instanceapi.InstanceSecrets{}, fmt.Errorf("get the escrowed secrets: %w", err)
	}
	if res.StatusCode() == http.StatusOK && res.JSON200 != nil {
		return *res.JSON200, nil
	}
	return instanceapi.InstanceSecrets{}, failure(
		res.HTTPResponse,
		res.ApplicationproblemJSON401,
		res.ApplicationproblemJSON429,
	)
}

// Commands lists what the control plane asks the instance to do.
func (c *Client) Commands(ctx context.Context) ([]instanceapi.Command, error) {
	res, err := c.api.ListCommandsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("list commands: %w", err)
	}
	if res.StatusCode() == http.StatusOK && res.JSON200 != nil {
		return res.JSON200.Commands, nil
	}
	return nil, failure(res.HTTPResponse, res.ApplicationproblemJSON401, res.ApplicationproblemJSON429)
}

// CompleteCommand says how a command ended. A command the control plane no
// longer has (it expired) is answered: there is nothing left to say.
func (c *Client) CompleteCommand(ctx context.Context, id uuid.UUID, result instanceapi.CommandResult) error {
	res, err := c.api.CompleteCommandWithResponse(ctx, id, result)
	if err != nil {
		return fmt.Errorf("answer a command: %w", err)
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

func (c *Client) Renew(ctx context.Context, csrPEM []byte) (credential.Issued, error) {
	res, err := c.api.RenewCertificateWithResponse(
		ctx,
		instanceapi.RenewCertificateRequest{Csr: string(csrPEM)},
	)
	if err != nil {
		return credential.Issued{}, fmt.Errorf("renew the certificate: %w", err)
	}
	if res.StatusCode() == http.StatusOK && res.JSON200 != nil {
		return credential.Issued{
			CertPEM: []byte(res.JSON200.Certificate),
			CAPEM:   []byte(res.JSON200.CaBundle),
		}, nil
	}
	return credential.Issued{}, failure(
		res.HTTPResponse,
		res.ApplicationproblemJSON400,
		res.ApplicationproblemJSON401,
		res.ApplicationproblemJSON429,
	)
}

func failure(res *http.Response, problems ...*instanceapi.Problem) error {
	switch res.StatusCode {
	case http.StatusUnauthorized:
		return ErrUnauthenticated
	case http.StatusTooManyRequests:
		return &RetryLaterError{
			Reason:     "rate limited",
			RetryAfter: retryAfter(res.Header.Get("Retry-After")),
		}
	case http.StatusServiceUnavailable:
		return &RetryLaterError{
			Reason:     "the desired spec is not ready",
			RetryAfter: retryAfter(res.Header.Get("Retry-After")),
		}
	}
	pe := &ProblemError{Status: res.StatusCode}
	for _, p := range problems {
		if p == nil {
			continue
		}
		pe.Title = p.Title
		if p.Detail != nil {
			pe.Detail = *p.Detail
		}
		if p.Code != nil {
			pe.Code = string(*p.Code)
		}
	}
	return pe
}

func retryAfter(h string) time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(h))
	if err != nil || secs <= 0 {
		return time.Minute
	}
	return time.Duration(secs) * time.Second
}
