package xerosession

import (
	"context"
	"errors"
	"net/http"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/emoss08/trenova/shared/xeroerrors"
)

var ErrNotConfigured = errors.New("xero is not configured on this instance")

var ErrNoRedirect = errors.New(
	"xero needs app.webBaseUrl or accounting.xero.redirectUrl to build its callback",
)

type Session struct {
	apiOpts    []xero.Option
	httpClient *http.Client
	Codes      *AccountCodes
}

func NewSession(apiOpts []xero.Option, httpClient *http.Client, codes *AccountCodes) *Session {
	return &Session{apiOpts: apiOpts, httpClient: httpClient, Codes: codes}
}

func (c *Session) IntegrationType() integration.Type {
	return integration.TypeXero
}

func (c *Session) Options() []xero.Option {
	opts := make([]xero.Option, 0, len(c.apiOpts)+1)
	opts = append(opts, xero.WithHTTPClient(c.httpClient))
	return append(opts, c.apiOpts...)
}

func (c *Session) Client(
	auth accountingconnectorports.AccountingDocumentAuth,
) (*xero.Client, error) {
	return xero.New(auth.RealmID, auth.AccessToken, c.Options()...)
}

func (c *Session) ClassifyError(err error) accountingsync.ErrorCategory {
	return ClassifyError(err)
}

func ClassifyError(err error) accountingsync.ErrorCategory {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrNotConfigured), errors.Is(err, ErrNoRedirect):
		return accountingsync.ErrorCategoryConfiguration
	case xeroerrors.IsInvalidGrant(err):
		return accountingsync.ErrorCategoryRevoked
	case xeroerrors.IsRateLimited(err):
		return accountingsync.ErrorCategoryRateLimited
	case xeroerrors.IsInvalidClient(err), xeroerrors.IsAuth(err), xeroerrors.IsForbidden(err):
		return accountingsync.ErrorCategoryUnauthorized
	case xeroerrors.IsTransient(err), errors.Is(err, context.DeadlineExceeded):
		return accountingsync.ErrorCategoryTransient
	default:
		return accountingsync.ErrorCategoryUnknown
	}
}
