package archiveengine

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const attestBeat = 20 * time.Second

type chainCheck struct {
	verified bool
	firstSeq int64
	lastSeq  int64
	lastHash string
	links    int64
}

type attestation struct {
	done   chan struct{}
	trails *tenantarchive.Trails
	err    error
}

func (e *Engine) startAttesting(
	ctx context.Context,
	tenant dbscope.Tenant,
	facts *archiveports.TrailFacts,
) *attestation {
	out := &attestation{done: make(chan struct{})}
	go func() {
		defer close(out.done)
		out.trails, out.err = e.attest(ctx, tenant, facts)
	}()
	return out
}

func (a *attestation) wait(ctx context.Context, beat func()) (*tenantarchive.Trails, error) {
	ticker := time.NewTicker(attestBeat)
	defer ticker.Stop()
	for {
		select {
		case <-a.done:
			return a.trails, a.err
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if beat != nil {
				beat()
			}
		}
	}
}

func (e *Engine) attest(
	ctx context.Context,
	tenant dbscope.Tenant,
	facts *archiveports.TrailFacts,
) (*tenantarchive.Trails, error) {
	if facts == nil || (facts.Audit == nil && facts.AIAudit == nil) {
		return nil, nil //nolint:nilnil // a tenant without audit trails states none
	}
	info := pagination.TenantInfo{OrgID: tenant.OrganizationID, BuID: tenant.BusinessUnitID}
	out := new(tenantarchive.Trails)
	if facts.Audit != nil {
		check, err := e.auditCheck(ctx, info, facts.Audit)
		if err != nil {
			return nil, fmt.Errorf("verify the tenant's audit seals before archiving them: %w", err)
		}
		out.Audit = e.attestation("audit log", tenant, facts.Audit, check)
	}
	if facts.AIAudit != nil {
		check, err := e.aiCheck(ctx, info, facts.AIAudit)
		if err != nil {
			return nil, fmt.Errorf("verify the tenant's AI audit chain before archiving it: %w", err)
		}
		out.AIAudit = e.attestation("AI audit trail", tenant, facts.AIAudit, check)
	}
	return out, nil
}

func (e *Engine) auditCheck(
	ctx context.Context,
	info pagination.TenantInfo,
	facts *archiveports.ChainFacts,
) (*chainCheck, error) {
	if facts.LastSeq == 0 {
		return &chainCheck{verified: true}, nil
	}
	check, err := e.auditTrail.VerifyThrough(ctx, info, facts.LastSeq)
	if err != nil {
		return nil, err
	}
	return &chainCheck{
		verified: check.Verified(),
		firstSeq: check.FirstSeq,
		lastSeq:  check.LastSeq,
		lastHash: check.LastHash,
		links:    check.Links,
	}, nil
}

func (e *Engine) aiCheck(
	ctx context.Context,
	info pagination.TenantInfo,
	facts *archiveports.ChainFacts,
) (*chainCheck, error) {
	check, err := e.aiTrail.VerifyThrough(ctx, info, facts.LastSeq)
	if err != nil {
		return nil, err
	}
	return &chainCheck{
		verified: check.Verified(),
		firstSeq: check.FirstSeq,
		lastSeq:  check.LastSeq,
		lastHash: check.LastHash,
		links:    check.Links,
	}, nil
}

func (e *Engine) attestation(
	trail string,
	tenant dbscope.Tenant,
	facts *archiveports.ChainFacts,
	check *chainCheck,
) *tenantarchive.TrailAttestation {
	held := facts.Held()
	verified := check.verified
	if held > 0 {
		verified = verified && check.firstSeq == facts.FirstSeq &&
			check.lastSeq == facts.LastSeq && check.lastHash == facts.LastHash &&
			check.links == held
	}
	if !verified {
		e.l.Warn("the archive states that its source could not verify a trail it carries",
			zap.String("trail", trail),
			zap.String("organizationId", tenant.OrganizationID.String()),
			zap.Bool("chainVerified", check.verified),
			zap.Int64("links", facts.Links),
			zap.Int64("walked", check.links),
		)
	}
	return &tenantarchive.TrailAttestation{
		SealedThrough: time.Unix(facts.SealedThrough, 0).UTC(),
		Verified:      verified,
		Links:         facts.Links,
	}
}
