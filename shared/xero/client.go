package xero

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/restx"
	"github.com/emoss08/trenova/shared/xeroerrors"
	"github.com/google/uuid"
)

const (
	MaxIDsPerRead           = 100
	MaxPageSize             = 1000
	MaxIdempotencyKeyLength = 128
	tenantHeader            = "xero-tenant-id"
	authorizationHeader     = "Authorization"
	actionAllowed           = "ALLOWED"
	guidLength              = 36
)

type Client struct {
	transport *restx.Client
	tenantID  string
}

type Organisation struct {
	OrganisationID        string
	Name                  string
	LegalName             string
	ShortCode             string
	CountryCode           string
	BaseCurrency          string
	OrganisationType      string
	SalesTaxBasis         string
	Edition               string
	FinancialYearEndDay   int
	FinancialYearEndMonth time.Month
	PeriodLockDate        *time.Time
	EndOfYearLockDate     *time.Time
	IsDemoCompany         bool
}

type OrganisationAction struct {
	Name   string
	Status string
}

func (a OrganisationAction) Allowed() bool {
	return strings.EqualFold(a.Status, actionAllowed)
}

type wireOrganisation struct {
	OrganisationID        string   `json:"OrganisationID"`
	Name                  string   `json:"Name"`
	LegalName             string   `json:"LegalName"`
	ShortCode             string   `json:"ShortCode"`
	CountryCode           string   `json:"CountryCode"`
	BaseCurrency          string   `json:"BaseCurrency"`
	OrganisationType      string   `json:"OrganisationType"`
	SalesTaxBasis         string   `json:"SalesTaxBasis"`
	Edition               string   `json:"Edition"`
	FinancialYearEndDay   int      `json:"FinancialYearEndDay"`
	FinancialYearEndMonth int      `json:"FinancialYearEndMonth"`
	PeriodLockDate        WireTime `json:"PeriodLockDate"`
	EndOfYearLockDate     WireTime `json:"EndOfYearLockDate"`
	IsDemoCompany         bool     `json:"IsDemoCompany"`
}

type organisationEnvelope struct {
	Organisations []wireOrganisation `json:"Organisations"`
}

type actionsEnvelope struct {
	Actions []struct {
		Name   string `json:"Name"`
		Status string `json:"Status"`
	} `json:"Actions"`
}

type Call struct {
	Endpoint      string
	Method        string
	Path          string
	Query         url.Values
	Key           string
	ModifiedSince *time.Time
	Body          any
	Out           any
	Expected      []int
}

func New(tenantID, accessToken string, opts ...Option) (*Client, error) {
	tenant := strings.TrimSpace(tenantID)
	if tenant == "" {
		return nil, xeroerrors.ErrTenantIDRequired
	}
	token := strings.TrimSpace(accessToken)
	if token == "" {
		return nil, xeroerrors.ErrAccessTokenRequired
	}

	settings := resolveOptions(opts)
	cfg := restx.Config{
		BaseURL:    settings.baseURL,
		Timeout:    settings.timeout,
		UserAgent:  settings.userAgent,
		HTTPClient: headerClient(settings.httpClient),
		Headers: map[string]string{
			authorizationHeader: "Bearer " + token,
			tenantHeader:        tenant,
		},
		Retry:        settings.retry,
		Observer:     settings.observer,
		ErrorDecoder: xeroerrors.DecodeAPIError,
	}
	if settings.limiter != nil {
		bucket := TenantBucket(settings.limiterKeyPrefix, tenant)
		cfg.Limiter = settings.limiter
		cfg.BucketFor = func(string) (restx.Bucket, bool) { return bucket, true }
	}

	transport, err := restx.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("xero: configure transport: %w", err)
	}

	return &Client{transport: transport, tenantID: tenant}, nil
}

func (c *Client) TenantID() string {
	return c.tenantID
}

func (c *Client) Organisation(ctx context.Context) (*Organisation, error) {
	var out organisationEnvelope
	if err := c.Do(ctx, &Call{
		Endpoint: "organisation",
		Method:   http.MethodGet,
		Path:     AccountingPath("Organisation"),
		Out:      &out,
	}); err != nil {
		return nil, err
	}
	if len(out.Organisations) == 0 {
		return nil, xeroerrors.ErrUnexpectedPayload
	}

	org := &out.Organisations[0]
	return &Organisation{
		OrganisationID:        org.OrganisationID,
		Name:                  org.Name,
		LegalName:             org.LegalName,
		ShortCode:             org.ShortCode,
		CountryCode:           org.CountryCode,
		BaseCurrency:          strings.ToUpper(strings.TrimSpace(org.BaseCurrency)),
		OrganisationType:      org.OrganisationType,
		SalesTaxBasis:         org.SalesTaxBasis,
		Edition:               org.Edition,
		FinancialYearEndDay:   org.FinancialYearEndDay,
		FinancialYearEndMonth: time.Month(org.FinancialYearEndMonth),
		PeriodLockDate:        org.PeriodLockDate.Ptr(),
		EndOfYearLockDate:     org.EndOfYearLockDate.Ptr(),
		IsDemoCompany:         org.IsDemoCompany,
	}, nil
}

func (c *Client) OrganisationActions(ctx context.Context) ([]OrganisationAction, error) {
	var out actionsEnvelope
	if err := c.Do(ctx, &Call{
		Endpoint: "organisation-actions",
		Method:   http.MethodGet,
		Path:     AccountingPath("Organisation", "Actions"),
		Out:      &out,
	}); err != nil {
		return nil, err
	}

	actions := make([]OrganisationAction, 0, len(out.Actions))
	for idx := range out.Actions {
		actions = append(actions, OrganisationAction{
			Name:   out.Actions[idx].Name,
			Status: out.Actions[idx].Status,
		})
	}
	return actions, nil
}

func (c *Client) Do(ctx context.Context, req *Call) error {
	headers := requestHeaders{idempotencyKey: req.Key}
	if req.ModifiedSince != nil && !req.ModifiedSince.IsZero() {
		headers.ifModifiedSince = formatHTTPDate(*req.ModifiedSince)
	}
	expected := req.Expected
	if len(expected) == 0 {
		expected = []int{http.StatusOK}
	}

	_, err := c.transport.Do(withRequestHeaders(ctx, headers), &restx.Request{
		Endpoint:       req.Endpoint,
		Method:         req.Method,
		Path:           req.Path,
		Query:          req.Query,
		Body:           req.Body,
		Out:            req.Out,
		ExpectedStatus: expected,
	})
	if err != nil {
		redactAPIError(err, c.transport.Redact)
	}
	return err
}

func redactAPIError(err error, redact func(string) string) {
	apiErr, ok := xeroerrors.APIErrorOf(err)
	if !ok {
		return
	}
	apiErr.Message = redact(apiErr.Message)
	for idx := range apiErr.ValidationMessages {
		apiErr.ValidationMessages[idx] = redact(apiErr.ValidationMessages[idx])
	}
}

func AccountingPath(parts ...string) string {
	segments := make([]string, 0, len(parts)+1)
	segments = append(segments, accountingRoot)
	for _, part := range parts {
		segments = append(segments, url.PathEscape(part))
	}
	return "/" + strings.Join(segments, "/")
}

func IdempotencyKey(key string) (string, error) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" || len(trimmed) > MaxIdempotencyKeyLength {
		return "", xeroerrors.ErrIdempotencyKey
	}
	for idx := range len(trimmed) {
		if trimmed[idx] < ' ' || trimmed[idx] > '~' {
			return "", xeroerrors.ErrIdempotencyKey
		}
	}
	return trimmed, nil
}

func GUID(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", xeroerrors.ErrIDRequired
	}
	if len(value) != guidLength {
		return "", xeroerrors.ErrInvalidID
	}
	if _, err := uuid.Parse(value); err != nil {
		return "", xeroerrors.ErrInvalidID
	}
	return strings.ToLower(value), nil
}

func GUIDs(ids []string, limit int) ([]string, error) {
	out := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, raw := range ids {
		id, err := GUID(raw)
		if err != nil {
			return nil, err
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) > limit {
		return nil, xeroerrors.ErrTooManyIDs
	}
	return out, nil
}

func PageQuery(page int) (url.Values, error) {
	if page < 1 {
		return nil, xeroerrors.ErrInvalidPage
	}
	return url.Values{
		"page":     {strconv.Itoa(page)},
		"pageSize": {strconv.Itoa(MaxPageSize)},
	}, nil
}

func TextWithin(value string, limit int) bool {
	return utf8.RuneCountInString(value) <= limit
}

func CurrencyCode(raw string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if code == "" {
		return "", nil
	}
	if len(code) != 3 {
		return "", xeroerrors.ErrCurrencyCodeInvalid
	}
	for _, r := range code {
		if !unicode.IsUpper(r) || r > unicode.MaxASCII {
			return "", xeroerrors.ErrCurrencyCodeInvalid
		}
	}
	return code, nil
}
