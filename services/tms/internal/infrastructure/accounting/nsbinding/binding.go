package nsbinding

import (
	"errors"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/netsuite"
)

var (
	ErrNoRedirect = errors.New(
		"netsuite needs app.webBaseUrl or accounting.netSuite.redirectUrl to build its callback",
	)
)

var ErrNotConfigured = errors.New("netsuite is not configured on this instance")

type Binding struct {
	apiOpts    []netsuite.Option
	httpClient *http.Client
	books      *CurrencyBooks
	now        func() time.Time
}

type BindingParams struct {
	APIOpts    []netsuite.Option
	HTTPClient *http.Client
	Books      *CurrencyBooks
	Now        func() time.Time
}

func NewBinding(p BindingParams) *Binding {
	return &Binding{
		apiOpts:    p.APIOpts,
		httpClient: p.HTTPClient,
		books:      p.Books,
		now:        p.Now,
	}
}

func (c *Binding) IntegrationType() integration.Type {
	return integration.TypeNetSuite
}

func (c *Binding) Options() []netsuite.Option {
	opts := make([]netsuite.Option, 0, len(c.apiOpts)+1)
	opts = append(opts, netsuite.WithHTTPClient(c.httpClient))
	return append(opts, c.apiOpts...)
}

func (c *Binding) ClassifyError(err error) accountingsync.ErrorCategory {
	return ClassifyError(err)
}
