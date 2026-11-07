package auditintegrity

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/zap"
)

const (
	SealLag         = 10 * time.Minute
	sealLookback    = 72 * time.Hour
	maxSealSpan     = 24 * time.Hour
)

type Sealer struct {
	entries auditports.AuditRepository
	seals   auditports.SealRepository
	keyring *Keyring
	metrics *metrics.Registry
	now     func() time.Time
	l       *zap.Logger
}

type SealerParams struct {
	Entries auditports.AuditRepository
	Seals   auditports.SealRepository
	Keyring *Keyring
	Metrics *metrics.Registry
	Now     func() time.Time
	Logger  *zap.Logger
}

func NewSealer(p *SealerParams) *Sealer {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Sealer{
		entries: p.Entries,
		seals:   p.Seals,
		keyring: p.Keyring,
		metrics: p.Metrics,
		now:     now,
		l:       p.Logger.Named("audit.sealer"),
	}
}

type SealResult struct {
	Tenants int
	Seals   int
	Failed  int
}

func Cutoff(now time.Time) int64 {
	return now.Add(-SealLag).UTC().Truncate(time.Hour).Unix()
}

func (s *Sealer) SealAll(ctx context.Context, heartbeat func(string)) (*SealResult, error) {
	cutoff := Cutoff(s.now())
	from := cutoff - int64(sealLookback.Seconds())
	tenants, err := s.entries.TenantsWithEntries(ctx, from, cutoff)
	if err != nil {
		return nil, err
	}

	result := &SealResult{Tenants: len(tenants)}
	for _, tenant := range tenants {
		if heartbeat != nil {
			heartbeat(tenant.OrganizationID.String())
		}
		sealed, sealErr := s.SealTenant(ctx, tenant, from, cutoff)
		result.Seals += sealed
		if sealErr != nil {
			result.Failed++
			s.l.Error("audit entries could not be sealed",
				zap.String("organizationId", tenant.OrganizationID.String()),
				zap.String("businessUnitId", tenant.BusinessUnitID.String()),
				zap.Error(sealErr),
			)
		}
	}
	return result, nil
}

func (s *Sealer) SealTenant(
	ctx context.Context,
	tenant auditports.TenantKey,
	windowStart, cutoff int64,
) (int, error) {
	sealed := 0
	for {
		seal, err := s.seals.AppendSeal(ctx, tenant, func(head *audit.ChainHead) (*audit.Seal, error) {
			start := head.SealedThrough
			if start == 0 {
				start = windowStart
			}
			if start >= cutoff {
				return nil, nil
			}
			end := min(cutoff, start+int64(maxSealSpan.Seconds()))

			digest, err := s.seals.Digest(ctx, tenant, start, end)
			if err != nil {
				return nil, err
			}

			next := &audit.Seal{
				OrganizationID: tenant.OrganizationID,
				BusinessUnitID: tenant.BusinessUnitID,
				Seq:            head.LastSeq + 1,
				FromTS:         start,
				ToTS:           end,
				RowCount:       digest.RowCount,
				ClassDigests:   digest.ClassDigests,
				PrevHash:       head.LastHash,
				SealedAt:       s.now().Unix(),
			}
			s.keyring.Sign(next)
			return next, nil
		})
		if err != nil {
			s.metrics.Audit.RecordSeal(false)
			return sealed, fmt.Errorf("seal audit entries: %w", err)
		}
		if seal == nil {
			return sealed, nil
		}
		s.metrics.Audit.RecordSeal(true)
		sealed++
		if seal.ToTS >= cutoff {
			return sealed, nil
		}
	}
}
