package triagecorrectionservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const MaxHintsPerSender = 3

type HintProviderParams struct {
	fx.In

	Logger *zap.Logger
	Hints  triageports.SenderHintRepository
	Cache  triageports.TriageCache `optional:"true"`
}

type HintProvider struct {
	l     *zap.Logger
	hints triageports.SenderHintRepository
	cache triageports.TriageCache
}

func NewHintProvider(p HintProviderParams) *HintProvider {
	return &HintProvider{
		l:     p.Logger.Named("service.triage-sender-hints"),
		hints: p.Hints,
		cache: p.Cache,
	}
}

var _ inboundmessageports.SenderHintProvider = (*HintProvider)(nil)

func (p *HintProvider) HintsFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	senderDomain string,
) ([]*triage.SenderHint, error) {
	domain := strings.ToLower(strings.TrimSpace(senderDomain))
	if domain == "" {
		return []*triage.SenderHint{}, nil
	}

	if p.cache != nil {
		cached, hit, err := p.cache.SenderHints(ctx, tenantInfo, domain)
		if err != nil {
			p.l.Warn("sender hint cache unavailable", zap.Error(err))
		}
		if hit {
			return cached, nil
		}
	}

	hints, err := p.hints.ListByDomain(ctx, triageports.ListSenderHintsRequest{
		TenantInfo:   tenantInfo,
		SenderDomain: domain,
		Limit:        MaxHintsPerSender,
	})
	if err != nil {
		return nil, err
	}

	if p.cache != nil {
		if err = p.cache.SetSenderHints(ctx, tenantInfo, domain, hints); err != nil {
			p.l.Warn("could not cache sender hints", zap.Error(err))
		}
	}

	return hints, nil
}
