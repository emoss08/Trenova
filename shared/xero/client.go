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
	PeriodLockDate        wireTime `json:"PeriodLockDate"`
	EndOfYearLockDate     wireTime `json:"EndOfYearLockDate"`
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

type call struct {
	endpoint      string
	method        string
	path          string
	query         url.Values
	key           string
	modifiedSince *time.Time
	body          any
	out           any
	expected      []int
}

func New(tenantID, accessToken string, opts ...Option) (*Client, error) {
	tenant := strings.TrimSpace(tenantID)
	if tenant == "" {
		return nil, ErrTenantIDRequired
	}
	token := strings.TrimSpace(accessToken)
	if token == "" {
		return nil, ErrAccessTokenRequired
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
		ErrorDecoder: decodeAPIError,
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
	if err := c.do(ctx, &call{
		endpoint: "organisation",
		method:   http.MethodGet,
		path:     accountingPath("Organisation"),
		out:      &out,
	}); err != nil {
		return nil, err
	}
	if len(out.Organisations) == 0 {
		return nil, ErrUnexpectedPayload
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
		PeriodLockDate:        org.PeriodLockDate.ptr(),
		EndOfYearLockDate:     org.EndOfYearLockDate.ptr(),
		IsDemoCompany:         org.IsDemoCompany,
	}, nil
}

func (c *Client) OrganisationActions(ctx context.Context) ([]OrganisationAction, error) {
	var out actionsEnvelope
	if err := c.do(ctx, &call{
		endpoint: "organisation-actions",
		method:   http.MethodGet,
		path:     accountingPath("Organisation", "Actions"),
		out:      &out,
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

func (c *Client) do(ctx context.Context, req *call) error {
	headers := requestHeaders{idempotencyKey: req.key}
	if req.modifiedSince != nil && !req.modifiedSince.IsZero() {
		headers.ifModifiedSince = formatHTTPDate(*req.modifiedSince)
	}
	expected := req.expected
	if len(expected) == 0 {
		expected = []int{http.StatusOK}
	}

	_, err := c.transport.Do(withRequestHeaders(ctx, headers), &restx.Request{
		Endpoint:       req.endpoint,
		Method:         req.method,
		Path:           req.path,
		Query:          req.query,
		Body:           req.body,
		Out:            req.out,
		ExpectedStatus: expected,
	})
	if err != nil {
		redactAPIError(err, c.transport.Redact)
	}
	return err
}

func redactAPIError(err error, redact func(string) string) {
	apiErr, ok := apiErrorOf(err)
	if !ok {
		return
	}
	apiErr.Message = redact(apiErr.Message)
	for idx := range apiErr.ValidationMessages {
		apiErr.ValidationMessages[idx] = redact(apiErr.ValidationMessages[idx])
	}
}

func accountingPath(parts ...string) string {
	segments := make([]string, 0, len(parts)+1)
	segments = append(segments, accountingRoot)
	for _, part := range parts {
		segments = append(segments, url.PathEscape(part))
	}
	return "/" + strings.Join(segments, "/")
}

func idempotencyKey(key string) (string, error) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" || len(trimmed) > MaxIdempotencyKeyLength {
		return "", ErrIdempotencyKey
	}
	for idx := range len(trimmed) {
		if trimmed[idx] < ' ' || trimmed[idx] > '~' {
			return "", ErrIdempotencyKey
		}
	}
	return trimmed, nil
}

func guid(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", ErrIDRequired
	}
	if len(value) != guidLength {
		return "", ErrInvalidID
	}
	if _, err := uuid.Parse(value); err != nil {
		return "", ErrInvalidID
	}
	return strings.ToLower(value), nil
}

func guids(ids []string, limit int) ([]string, error) {
	out := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, raw := range ids {
		id, err := guid(raw)
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
		return nil, ErrTooManyIDs
	}
	return out, nil
}

func pageQuery(page int) (url.Values, error) {
	if page < 1 {
		return nil, ErrInvalidPage
	}
	return url.Values{
		"page":     {strconv.Itoa(page)},
		"pageSize": {strconv.Itoa(MaxPageSize)},
	}, nil
}

func textWithin(value string, limit int) bool {
	return utf8.RuneCountInString(value) <= limit
}

func currencyCode(raw string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if code == "" {
		return "", nil
	}
	if len(code) != 3 {
		return "", ErrCurrencyCodeInvalid
	}
	for _, r := range code {
		if !unicode.IsUpper(r) || r > unicode.MaxASCII {
			return "", ErrCurrencyCodeInvalid
		}
	}
	return code, nil
}
