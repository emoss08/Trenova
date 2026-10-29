package businesscentral

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/businesscentralerrors"
)

const (
	MaxPageSize                     = 1000
	MaxIDsPerRead                   = 50
	MaxExternalDocumentNumberLength = 35
	MaxDocumentNumberLength         = 20
	MaxJournalCodeLength            = 10
	MaxDescriptionLength            = 100
	MaxClientStateLength            = 2048
	SubscriptionLifetime            = 72 * time.Hour
	actionPrefix                    = "Microsoft.NAV."
)

type Client struct {
	core        *Core
	ref         CompanyRef
	envPath     string
	companyPath string
}

type CompanyInformation struct {
	ID                string
	DisplayName       string
	CountryRegionCode string
	CurrencyCode      string
}

type GeneralLedgerSetup struct {
	LocalCurrencyCode string
	AllowPostingFrom  string
	AllowPostingTo    string
}

func (s *GeneralLedgerSetup) AllowsPosting(date string) (bool, error) {
	value, err := InputDate(date, true)
	if err != nil {
		return false, err
	}
	if s.AllowPostingFrom != "" && value < s.AllowPostingFrom {
		return false, nil
	}
	if s.AllowPostingTo != "" && value > s.AllowPostingTo {
		return false, nil
	}
	return true, nil
}

type AccountingPeriod struct {
	StartingDate  string
	Name          string
	NewFiscalYear bool
	Closed        bool
	DateLocked    bool
}

type Currency struct {
	ID          string
	Code        string
	DisplayName string
}

type wireCompanyInformation struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	CountryRegionCode string `json:"countryRegionCode"`
	CurrencyCode      string `json:"currencyCode"`
}

func (w *wireCompanyInformation) info() CompanyInformation {
	return CompanyInformation{
		ID:                strings.ToLower(w.ID),
		DisplayName:       w.DisplayName,
		CountryRegionCode: w.CountryRegionCode,
		CurrencyCode:      strings.ToUpper(strings.TrimSpace(w.CurrencyCode)),
	}
}

type wireGeneralLedgerSetup struct {
	LocalCurrencyCode string `json:"localCurrencyCode"`
	AllowPostingFrom  string `json:"allowPostingFrom"`
	AllowPostingTo    string `json:"allowPostingTo"`
}

func (w *wireGeneralLedgerSetup) setup() GeneralLedgerSetup {
	return GeneralLedgerSetup{
		LocalCurrencyCode: strings.ToUpper(strings.TrimSpace(w.LocalCurrencyCode)),
		AllowPostingFrom:  OutputDate(w.AllowPostingFrom),
		AllowPostingTo:    OutputDate(w.AllowPostingTo),
	}
}

type wireAccountingPeriod struct {
	StartingDate  string `json:"startingDate"`
	Name          string `json:"name"`
	NewFiscalYear bool   `json:"newFiscalYear"`
	Closed        bool   `json:"closed"`
	DateLocked    bool   `json:"dateLocked"`
}

func (w *wireAccountingPeriod) period() AccountingPeriod {
	return AccountingPeriod{
		StartingDate:  OutputDate(w.StartingDate),
		Name:          w.Name,
		NewFiscalYear: w.NewFiscalYear,
		Closed:        w.Closed,
		DateLocked:    w.DateLocked,
	}
}

type wireCurrency struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	DisplayName string `json:"displayName"`
}

func (w *wireCurrency) currency() Currency {
	return Currency{ID: strings.ToLower(w.ID), Code: w.Code, DisplayName: w.DisplayName}
}

func New(ref CompanyRef, accessToken string, opts ...Option) (*Client, error) {
	token := strings.TrimSpace(accessToken)
	if token == "" {
		return nil, businesscentralerrors.ErrAccessTokenRequired
	}
	normalized, err := NewCompanyRef(ref.TenantID, ref.Environment, ref.CompanyID)
	if err != nil {
		return nil, err
	}

	settings := resolveOptions(opts)
	bucket := TenantBucket(
		settings.limiterKeyPrefix,
		normalized.TenantID,
		strings.ToLower(normalized.Environment),
	)
	c, err := newCore(&settings, token, &bucket)
	if err != nil {
		return nil, err
	}
	envPath := environmentPath(normalized.TenantID, normalized.Environment)
	return &Client{
		core:        c,
		ref:         normalized,
		envPath:     envPath,
		companyPath: envPath + "/" + CompaniesEntity + "(" + normalized.CompanyID + ")",
	}, nil
}

func (c *Client) Ref() CompanyRef {
	return c.ref
}

func (c *Client) CompanyInformation(ctx context.Context) (*CompanyInformation, error) {
	return FetchFirst(ctx, c.core, &ListCall{
		Endpoint: "company-information",
		Path:     c.CollectionPath("companyInformation"),
	}, (*wireCompanyInformation).info)
}

func (c *Client) GeneralLedgerSetup(ctx context.Context) (*GeneralLedgerSetup, error) {
	return FetchFirst(ctx, c.core, &ListCall{
		Endpoint: "general-ledger-setup",
		Path:     c.CollectionPath("generalLedgerSetup"),
	}, (*wireGeneralLedgerSetup).setup)
}

func (c *Client) AccountingPeriods(ctx context.Context) ([]AccountingPeriod, error) {
	return Collect(ctx, c.core, &ListCall{
		Endpoint: "accounting-periods",
		Path:     c.CollectionPath("accountingPeriods"),
	}, (*wireAccountingPeriod).period)
}

func (c *Client) Currencies(ctx context.Context) ([]Currency, error) {
	return Collect(ctx, c.core, &ListCall{
		Endpoint: "currencies",
		Path:     c.CollectionPath("currencies"),
	}, (*wireCurrency).currency)
}

func (c *Client) Core() *Core {
	return c.core
}

func (c *Client) EnvPath() string {
	return c.envPath
}

func (c *Client) CollectionPath(entity string) string {
	return c.companyPath + "/" + entity
}

func (c *Client) EntityPath(entity, id string) string {
	return c.companyPath + "/" + entity + "(" + id + ")"
}

func (c *Client) ActionPath(entity, id, action string) string {
	return c.EntityPath(entity, id) + "/" + actionPrefix + action
}

func (c *Client) Invoke(ctx context.Context, endpoint, path string) error {
	_, err := c.core.do(ctx, &Call{
		Endpoint: endpoint,
		Method:   http.MethodPost,
		Path:     path,
		Expected: []int{http.StatusNoContent, http.StatusOK},
	})
	return err
}

func (c *Client) Remove(ctx context.Context, endpoint, path, etag string) error {
	match, err := IfMatch(etag)
	if err != nil {
		return err
	}
	_, err = c.core.do(ctx, &Call{
		Endpoint: endpoint,
		Method:   http.MethodDelete,
		Path:     path,
		IfMatch:  match,
		Expected: []int{http.StatusNoContent, http.StatusOK},
	})
	return err
}

const (
	ActionPost = "post"
)
