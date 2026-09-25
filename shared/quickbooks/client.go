package quickbooks

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/shared/restx"
)

type Client struct {
	transport *restx.Client
	realmID   string
}

type CompanyInfo struct {
	RealmID     string
	CompanyName string
	LegalName   string
	Country     string
}

type Preferences struct {
	HomeCurrency         string
	MultiCurrencyEnabled bool
	BooksClosedThrough   string
}

type companyInfoEnvelope struct {
	CompanyInfo *struct {
		ID          string `json:"Id"`
		CompanyName string `json:"CompanyName"`
		LegalName   string `json:"LegalName"`
		Country     string `json:"Country"`
	} `json:"CompanyInfo"`
}

type preferencesEnvelope struct {
	Preferences *struct {
		CurrencyPrefs *struct {
			MultiCurrencyEnabled bool `json:"MultiCurrencyEnabled"`
			HomeCurrency         *struct {
				Value string `json:"value"`
			} `json:"HomeCurrency"`
		} `json:"CurrencyPrefs"`
		AccountingInfoPrefs *struct {
			BookCloseDate string `json:"BookCloseDate"`
		} `json:"AccountingInfoPrefs"`
	} `json:"Preferences"`
}

func New(env Environment, realmID, accessToken string, opts ...Option) (*Client, error) {
	realm := strings.TrimSpace(realmID)
	if realm == "" {
		return nil, ErrRealmIDRequired
	}
	token := strings.TrimSpace(accessToken)
	if token == "" {
		return nil, ErrAccessTokenRequired
	}

	settings := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&settings)
		}
	}
	baseURL := env.APIBaseURL()
	if settings.baseURL != "" {
		baseURL = settings.baseURL
	}

	cfg := restx.Config{
		BaseURL:      baseURL,
		Timeout:      settings.timeout,
		UserAgent:    settings.userAgent,
		HTTPClient:   settings.httpClient,
		Headers:      map[string]string{"Authorization": "Bearer " + token},
		QueryParams:  map[string]string{},
		Retry:        settings.retry,
		Observer:     settings.observer,
		ErrorDecoder: decodeAPIError,
	}
	if settings.limiter != nil {
		bucket := RealmBucket(settings.limiterKeyPrefix, realm)
		cfg.Limiter = settings.limiter
		cfg.BucketFor = func(string) (restx.Bucket, bool) { return bucket, true }
	}

	transport, err := restx.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("quickbooks: configure transport: %w", err)
	}

	return &Client{transport: transport, realmID: realm}, nil
}

func (c *Client) RealmID() string {
	return c.realmID
}

func (c *Client) CompanyInfo(ctx context.Context) (*CompanyInfo, error) {
	var out companyInfoEnvelope
	if _, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: "company-info",
		Method:   http.MethodGet,
		Path:     c.companyPath("companyinfo", c.realmID),
		Query:    c.query(),
		Out:      &out,
	}); err != nil {
		return nil, err
	}
	if out.CompanyInfo == nil {
		return nil, ErrUnexpectedPayload
	}

	return &CompanyInfo{
		RealmID:     c.realmID,
		CompanyName: out.CompanyInfo.CompanyName,
		LegalName:   out.CompanyInfo.LegalName,
		Country:     out.CompanyInfo.Country,
	}, nil
}

func (c *Client) Preferences(ctx context.Context) (*Preferences, error) {
	var out preferencesEnvelope
	if _, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: "preferences",
		Method:   http.MethodGet,
		Path:     c.companyPath("preferences"),
		Query:    c.query(),
		Out:      &out,
	}); err != nil {
		return nil, err
	}
	if out.Preferences == nil {
		return nil, ErrUnexpectedPayload
	}

	prefs := &Preferences{}
	if currency := out.Preferences.CurrencyPrefs; currency != nil {
		prefs.MultiCurrencyEnabled = currency.MultiCurrencyEnabled
		if currency.HomeCurrency != nil {
			prefs.HomeCurrency = strings.ToUpper(strings.TrimSpace(currency.HomeCurrency.Value))
		}
	}
	if accounting := out.Preferences.AccountingInfoPrefs; accounting != nil {
		prefs.BooksClosedThrough = strings.TrimSpace(accounting.BookCloseDate)
	}

	return prefs, nil
}

func (c *Client) companyPath(parts ...string) string {
	segments := make([]string, 0, len(parts)+3)
	segments = append(segments, "v3", "company", url.PathEscape(c.realmID))
	for _, part := range parts {
		segments = append(segments, url.PathEscape(part))
	}
	return "/" + strings.Join(segments, "/")
}

func (c *Client) query() url.Values {
	return url.Values{"minorversion": {minorVersion}}
}
