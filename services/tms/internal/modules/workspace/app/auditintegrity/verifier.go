package auditintegrity

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/identity/domain/dataretention"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const (
	verifyPageSize        = 200
	rollingVerifyPerRun   = 200
	secondsPerDay         = int64(86400)
	verificationDetailMax = 1000
)

type Verifier struct {
	seals     auditports.SealRepository
	retention tenantports.DataRetentionRepository
	keyring   *Keyring
	metrics   *metrics.Registry
	now       func() time.Time
	l         *zap.Logger
}

type VerifierParams struct {
	Seals     auditports.SealRepository
	Retention tenantports.DataRetentionRepository
	Keyring   *Keyring
	Metrics   *metrics.Registry
	Now       func() time.Time
	Logger    *zap.Logger
}

func NewVerifier(p *VerifierParams) *Verifier {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Verifier{
		seals:     p.Seals,
		retention: p.Retention,
		keyring:   p.Keyring,
		metrics:   p.Metrics,
		now:       now,
		l:         p.Logger.Named("audit.verifier"),
	}
}

type VerifyResult struct {
	Tenants int
	Broken  int
}

func (v *Verifier) VerifyAll(ctx context.Context, heartbeat func(string)) (*VerifyResult, error) {
	heads, err := v.seals.ListHeads(ctx)
	if err != nil {
		return nil, err
	}
	result := &VerifyResult{Tenants: len(heads)}
	for _, head := range heads {
		if heartbeat != nil {
			heartbeat(head.OrganizationID.String())
		}
		verified, verifyErr := v.VerifyTenant(ctx, head)
		if verifyErr != nil {
			v.l.Error("audit chain could not be verified",
				zap.String("organizationId", head.OrganizationID.String()),
				zap.Error(verifyErr),
			)
			continue
		}
		if verified.LastVerificationStatus != audit.VerificationVerified {
			result.Broken++
		}
	}
	return result, nil
}

func (v *Verifier) VerifyTenant(ctx context.Context, head *audit.ChainHead) (*audit.ChainHead, error) {
	tenant := pagination.TenantInfo{OrgID: head.OrganizationID, BuID: head.BusinessUnitID}
	ctx = dbscope.WithValidTenant(ctx, dbscope.Tenant{
		OrganizationID: head.OrganizationID,
		BusinessUnitID: head.BusinessUnitID,
	})
	policy := v.retentionFor(ctx, tenant)

	status, failedSeq, detail, err := v.walk(ctx, tenant, policy, head.LastVerifiedSeq, head.LastSeq)
	if err != nil {
		return nil, err
	}

	rollingFrom := head.VerifyCursorSeq
	if status == audit.VerificationVerified && rollingFrom < head.LastVerifiedSeq {
		rollingTo := min(head.LastVerifiedSeq, rollingFrom+rollingVerifyPerRun)
		status, failedSeq, detail, err = v.walk(ctx, tenant, policy, rollingFrom, rollingTo)
		if err != nil {
			return nil, err
		}
		head.VerifyCursorSeq = rollingTo
		if rollingTo >= head.LastVerifiedSeq {
			head.VerifyCursorSeq = 0
		}
	}

	now := v.now().Unix()
	head.LastVerifiedAt = &now
	head.LastVerificationStatus = status
	head.LastVerificationDetail = truncate(detail, verificationDetailMax)
	head.LastVerificationFailedSeq = nil
	if status == audit.VerificationVerified {
		head.LastVerifiedSeq = head.LastSeq
	} else {
		head.LastVerificationFailedSeq = &failedSeq
	}
	v.metrics.Audit.RecordVerification(string(status))

	if err = v.seals.RecordVerification(ctx, head); err != nil {
		return nil, err
	}
	return head, nil
}

func (v *Verifier) walk(
	ctx context.Context,
	tenant pagination.TenantInfo,
	policy *dataretention.DataRetention,
	fromSeq, toSeq int64,
) (audit.VerificationStatus, int64, string, error) {
	prevHash := ""
	after := fromSeq
	if fromSeq > 0 {
		anchor, err := v.seals.ListSeals(ctx, tenant, fromSeq-1, 1)
		if err != nil {
			return "", 0, "", err
		}
		if len(anchor) == 1 && anchor[0].Seq == fromSeq {
			prevHash = anchor[0].Hash
		}
	}

	for after < toSeq {
		page, err := v.seals.ListSeals(ctx, tenant, after, verifyPageSize)
		if err != nil {
			return "", 0, "", err
		}
		if len(page) == 0 {
			return audit.VerificationBroken, after + 1,
				fmt.Sprintf("seal %d is missing", after+1), nil
		}
		for _, seal := range page {
			if seal.Seq > toSeq {
				return audit.VerificationVerified, 0, "", nil
			}
			if seal.Seq != after+1 {
				return audit.VerificationBroken, after + 1,
					fmt.Sprintf("seal %d is missing", after+1), nil
			}
			if (after > 0 || fromSeq == 0) && seal.PrevHash != prevHash {
				return audit.VerificationBroken, seal.Seq,
					fmt.Sprintf("seal %d does not follow the seal before it", seal.Seq), nil
			}
			ok, keyMissing := v.keyring.Check(seal)
			if keyMissing {
				return audit.VerificationKeyMissing, seal.Seq,
					fmt.Sprintf("seal %d was signed with key %q, which is not configured", seal.Seq, seal.HashKeyID), nil
			}
			if !ok {
				return audit.VerificationBroken, seal.Seq,
					fmt.Sprintf("seal %d has been altered", seal.Seq), nil
			}
			if detail, intact, checkErr := v.checkEntries(ctx, seal, policy); checkErr != nil {
				return "", 0, "", checkErr
			} else if !intact {
				return audit.VerificationBroken, seal.Seq, detail, nil
			}
			prevHash = seal.Hash
			after = seal.Seq
		}
	}

	return audit.VerificationVerified, 0, "", nil
}

func (v *Verifier) checkEntries(
	ctx context.Context,
	seal *audit.Seal,
	policy *dataretention.DataRetention,
) (string, bool, error) {
	present, err := v.seals.Digest(ctx, auditports.TenantKey{
		OrganizationID: seal.OrganizationID,
		BusinessUnitID: seal.BusinessUnitID,
	}, seal.FromTS, seal.ToTS)
	if err != nil {
		return "", false, err
	}

	now := v.now().Unix()
	for _, class := range audit.RetentionClasses {
		sealed := seal.ClassDigests[string(class)]
		found := present.ClassDigests[string(class)]
		if found.Count == sealed.Count && found.Digest == sealed.Digest {
			continue
		}
		expiredBefore := now - int64(policy.AuditRetentionDays(string(class)))*secondsPerDay
		if seal.ToTS <= expiredBefore && found.Count <= sealed.Count {
			continue
		}
		switch {
		case found.Count > sealed.Count:
			return fmt.Sprintf(
				"%d %s entries were added to hours sealed by seal %d",
				found.Count-sealed.Count, class, seal.Seq,
			), false, nil
		case found.Count < sealed.Count:
			return fmt.Sprintf(
				"%d %s entries sealed by seal %d are missing before their retention ended",
				sealed.Count-found.Count, class, seal.Seq,
			), false, nil
		default:
			return fmt.Sprintf("%s entries sealed by seal %d have been changed", class, seal.Seq), false, nil
		}
	}
	return "", true, nil
}

func (v *Verifier) retentionFor(
	ctx context.Context,
	tenant pagination.TenantInfo,
) *dataretention.DataRetention {
	if v.retention == nil {
		return nil
	}
	policy, err := v.retention.Get(ctx, tenantports.GetDataRetentionRequest{
		OrgID: tenant.OrgID,
		BuID:  tenant.BuID,
	})
	if err != nil {
		return nil
	}
	return policy
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
