package controlplane

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

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

func (c *Client) ReportStatus(ctx context.Context, report *instanceapi.StatusReport) error {
	res, err := c.api.ReportStatusWithResponse(ctx, *report)
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

var ErrNoStorage = errors.New("the instance has no object storage of its own")

func (c *Client) Storage(ctx context.Context) (instanceapi.InstanceStorage, error) {
	res, err := c.api.GetStorageWithResponse(ctx)
	if err != nil {
		return instanceapi.InstanceStorage{}, fmt.Errorf("get the instance's storage: %w", err)
	}
	switch {
	case res.StatusCode() == http.StatusOK && res.JSON200 != nil:
		return *res.JSON200, nil
	case res.StatusCode() == http.StatusNotFound:
		return instanceapi.InstanceStorage{}, ErrNoStorage
	}
	return instanceapi.InstanceStorage{}, failure(
		res.HTTPResponse,
		res.ApplicationproblemJSON401,
		res.ApplicationproblemJSON429,
	)
}

func (c *Client) Commands(ctx context.Context) (instanceapi.CommandList, error) {
	res, err := c.api.ListCommandsWithResponse(ctx)
	if err != nil {
		return instanceapi.CommandList{}, fmt.Errorf("list commands: %w", err)
	}
	if res.StatusCode() == http.StatusOK && res.JSON200 != nil {
		return *res.JSON200, nil
	}
	return instanceapi.CommandList{}, failure(
		res.HTTPResponse,
		res.ApplicationproblemJSON401,
		res.ApplicationproblemJSON429,
	)
}

func (c *Client) CompleteCommand(
	ctx context.Context,
	id uuid.UUID,
	result instanceapi.CommandResult,
) error {
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
			RetryAfter: RetryAfter(res.Header.Get("Retry-After"), time.Now()),
		}
	case http.StatusServiceUnavailable:
		return &RetryLaterError{
			Reason:     "unavailable for now",
			RetryAfter: RetryAfter(res.Header.Get("Retry-After"), time.Now()),
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

const (
	defaultRetryAfter = time.Minute
	maxRetryAfter     = time.Hour
)

func RetryAfter(h string, now time.Time) time.Duration {
	h = strings.TrimSpace(h)
	if secs, err := strconv.ParseInt(h, 10, 64); err == nil {
		if secs <= 0 {
			return defaultRetryAfter
		}
		if secs > int64(maxRetryAfter/time.Second) {
			return maxRetryAfter
		}
		return time.Duration(secs) * time.Second
	}
	at, err := http.ParseTime(h)
	if err != nil {
		return defaultRetryAfter
	}
	return min(max(at.Sub(now), time.Second), maxRetryAfter)
}
