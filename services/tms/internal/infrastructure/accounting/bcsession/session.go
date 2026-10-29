package bcsession

import (
	"errors"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/businesscentral"
)

var (
	ErrNoRedirect = errors.New(
		"business central needs app.webBaseUrl or accounting.businessCentral.redirectUrl " +
			"to build its callback",
	)
)

var ErrNotConfigured = errors.New("business central is not configured on this instance")

type Session struct {
	apiOpts    []businesscentral.Option
	httpClient *http.Client
	locks      *BatchLocks
	setups     *PostingSetups
	now        func() time.Time
}

type SessionParams struct {
	APIOpts    []businesscentral.Option
	HTTPClient *http.Client
	Locks      *BatchLocks
	Setups     *PostingSetups
	Now        func() time.Time
}

func NewSession(p SessionParams) *Session {
	return &Session{
		apiOpts:    p.APIOpts,
		httpClient: p.HTTPClient,
		locks:      p.Locks,
		setups:     p.Setups,
		now:        p.Now,
	}
}

func (c *Session) IntegrationType() integration.Type {
	return integration.TypeBusinessCentral
}

func (c *Session) Options() []businesscentral.Option {
	opts := make([]businesscentral.Option, 0, len(c.apiOpts)+1)
	opts = append(opts, businesscentral.WithHTTPClient(c.httpClient))
	return append(opts, c.apiOpts...)
}

func (c *Session) Client(
	auth accountingconnectorports.AccountingDocumentAuth,
) (*businesscentral.Client, error) {
	ref, err := businesscentral.ParseCompanyRef(auth.RealmID)
	if err != nil {
		return nil, err
	}
	return businesscentral.New(ref, auth.AccessToken, c.Options()...)
}

func (c *Session) Clock() time.Time {
	return c.now()
}

func (c *Session) Linker() *businesscentral.Linker {
	return businesscentral.NewLinker(c.apiOpts...)
}

func (c *Session) ClassifyError(err error) accountingsync.ErrorCategory {
	return ClassifyError(err)
}
