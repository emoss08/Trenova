package aiauditverify

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aimetrics"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiauditheartbeat"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiauditkeyring"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const (
	verifyPageSize       = 1000
	mismatchNoticeLimit  = 50
	mismatchNoticeWindow = 7 * 24 * time.Hour
	mismatchNoticeSource = "aiauditservice.Verify"
	mismatchKeyKind      = "kind"
	mismatchKeyStatus    = "status"
)

// PermittedNotifier tells everyone who holds a permission about one thing.
type PermittedNotifier interface {
	NotifyPermitted(
		ctx context.Context,
		req notificationservice.NotifyPermittedRequest,
	) (int, error)
}

// VerifyResult is what walking one tenant's chain found.
type VerifyResult struct {
	TenantInfo  pagination.TenantInfo      `json:"tenantInfo"`
	Status      aiaudit.VerificationStatus `json:"status"`
	VerifiedSeq int64                      `json:"verifiedSeq"`
	FailedSeq   *int64                     `json:"failedSeq,omitempty"`
	Detail      string                     `json:"detail,omitempty"`
	Rows        int64                      `json:"rows"`
}

// Verifier walks a tenant's chain from its oldest retained row and checks
// every link: the seq has no gap, each row names the hash before it, each
// hash recomputes under the key it names, each seal matches the row it ends
// on, and the head names the last row.
type Verifier struct {
	ledger   aiauditports.AIAuditRepository
	keyring  *aiauditkeyring.Keyring
	notifier PermittedNotifier
	audit    auditports.AuditService
	realtime realtimeports.RealtimeService
	metrics  *aimetrics.AIAudit
	now      func() time.Time
	l        *zap.Logger
}

type Params struct {
	Ledger   aiauditports.AIAuditRepository
	Keyring  *aiauditkeyring.Keyring
	Notifier PermittedNotifier
	Audit    auditports.AuditService
	Realtime realtimeports.RealtimeService
	Metrics  *aimetrics.AIAudit
	Now      func() time.Time
	Logger   *zap.Logger
}

func New(p *Params) *Verifier {
	now := p.Now
	if now == nil {
		now = time.Now
	}

	return &Verifier{
		ledger:   p.Ledger,
		keyring:  p.Keyring,
		notifier: p.Notifier,
		audit:    p.Audit,
		realtime: p.Realtime,
		metrics:  p.Metrics,
		now:      now,
		l:        p.Logger.Named("aiaudit.verifier"),
	}
}

// Tenants are the tenants with a trail to verify.
func (v *Verifier) Tenants(ctx context.Context) ([]pagination.TenantInfo, error) {
	heads, err := v.ledger.ListChainHeads(ctx)
	if err != nil {
		return nil, err
	}

	tenants := make([]pagination.TenantInfo, 0, len(heads))
	for _, head := range heads {
		tenants = append(tenants, pagination.TenantInfo{
			OrgID: head.OrganizationID,
			BuID:  head.BusinessUnitID,
		})
	}

	return tenants, nil
}

// VerifyTenant walks one tenant's chain, records what it found on the chain
// head, and raises the alarm when the chain does not hold.
func (v *Verifier) VerifyTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	heartbeat aiauditheartbeat.Heartbeat,
) (*VerifyResult, error) {
	result, err := v.walk(ctx, tenantInfo, heartbeat)
	if err != nil {
		return nil, err
	}

	if _, err = v.ledger.RecordVerification(ctx, &aiauditports.RecordAIAuditVerificationRequest{
		TenantInfo:  tenantInfo,
		Status:      result.Status,
		VerifiedSeq: result.VerifiedSeq,
		FailedSeq:   result.FailedSeq,
		Detail:      result.Detail,
		At:          v.now().Unix(),
	}); err != nil {
		return nil, err
	}
	v.metrics.RecordVerification(string(result.Status))
	v.publish(ctx, tenantInfo)

	if !result.Status.Holds() {
		v.alarm(ctx, result)
	}

	return result, nil
}

func (v *Verifier) walk(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	heartbeat aiauditheartbeat.Heartbeat,
) (*VerifyResult, error) {
	result := &VerifyResult{TenantInfo: tenantInfo, Status: aiaudit.VerificationVerified}

	head, err := v.ledger.GetChainHead(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if head.LastSeq == 0 {
		return result, nil
	}

	first, found, err := v.ledger.FirstSeq(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !found {
		return v.verifyFullyPruned(ctx, tenantInfo, head, result)
	}

	cursor, ended, err := v.walkRange(ctx, result, first, head.LastSeq, heartbeat)
	if err != nil {
		return nil, err
	}
	if ended {
		return result, nil
	}

	if result.VerifiedSeq != head.LastSeq || cursor.expectedPrev != head.LastHash {
		return fail(result, head.LastSeq, "the chain head does not name the chain's last row"), nil
	}
	result.Status = cursor.origin.Status()

	return result, nil
}

func (v *Verifier) VerifyThrough(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	throughSeq int64,
) (*aiauditports.TrailCheck, error) {
	ctx = dbscope.WithValidTenant(ctx, dbscope.Tenant{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	})
	first, found, err := v.ledger.FirstSeq(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !found || throughSeq < first {
		return &aiauditports.TrailCheck{Status: aiaudit.VerificationVerified}, nil
	}
	result := &VerifyResult{TenantInfo: tenantInfo, Status: aiaudit.VerificationVerified}
	cursor, ended, err := v.walkRange(ctx, result, first, throughSeq, nil)
	if err != nil {
		return nil, err
	}
	check := &aiauditports.TrailCheck{
		Status:   result.Status,
		FirstSeq: first,
		LastSeq:  result.VerifiedSeq,
		LastHash: cursor.expectedPrev,
		Links:    result.Rows,
	}
	if !ended && result.VerifiedSeq != throughSeq {
		check.Status = aiaudit.VerificationMismatch
	}
	if check.Status != aiaudit.VerificationVerified {
		v.l.Warn("an archive's AI audit chain does not verify at its source",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.String("status", string(check.Status)),
			zap.String("detail", result.Detail))
	}

	return check, nil
}

func (v *Verifier) walkRange(
	ctx context.Context,
	result *VerifyResult,
	first, last int64,
	heartbeat aiauditheartbeat.Heartbeat,
) (*chainCursor, bool, error) {
	tenantInfo := result.TenantInfo
	expectedPrev, anchorErr := v.anchor(ctx, tenantInfo, first)
	if anchorErr != "" {
		fail(result, first, anchorErr)
		return &chainCursor{}, true, nil
	}

	cursor := &chainCursor{expectedPrev: expectedPrev, expectedSeq: first}
	after := first - 1
	for after < last {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}

		rows, listErr := v.ledger.ListChainRange(ctx, aiauditports.ListAIAuditChainRangeRequest{
			TenantInfo: tenantInfo,
			AfterSeq:   after,
			Limit:      verifyPageSize,
		})
		if listErr != nil {
			return nil, false, listErr
		}
		if len(rows) == 0 {
			fail(result, cursor.expectedSeq, "rows are missing from the end of the chain")
			return cursor, true, nil
		}

		seals, sealErr := v.pageSeals(ctx, tenantInfo, rows[0].Seq, rows[len(rows)-1].Seq)
		if sealErr != nil {
			return nil, false, sealErr
		}

		if v.verifyPage(result, cursor, rows, seals, last) {
			return cursor, true, nil
		}

		after = rows[len(rows)-1].Seq
		aiauditheartbeat.Beat(heartbeat, tenantInfo.OrgID.String(), after)
	}

	return cursor, false, nil
}

type chainCursor struct {
	expectedPrev string
	expectedSeq  int64
	signedFrom   int64
	origin       aiaudit.Origin
}

func (v *Verifier) verifyPage(
	result *VerifyResult,
	cursor *chainCursor,
	rows []*aiaudit.AIAuditEvent,
	seals map[int64]*aiaudit.AIAuditSeal,
	lastSeq int64,
) (ended bool) {
	for _, row := range rows {
		if row.Seq > lastSeq {
			break
		}
		if v.rowEndsChain(result, cursor, row, seals) {
			return true
		}

		if cursor.signedFrom == 0 && row.Keyed() {
			cursor.signedFrom = row.Seq
		}
		cursor.origin = cursor.origin.Weaker(row.Origin())
		cursor.expectedPrev = row.Hash
		cursor.expectedSeq = row.Seq + 1
		result.VerifiedSeq = row.Seq
		result.Rows++
	}

	return false
}

func (v *Verifier) rowEndsChain(
	result *VerifyResult,
	cursor *chainCursor,
	row *aiaudit.AIAuditEvent,
	seals map[int64]*aiaudit.AIAuditSeal,
) bool {
	if row.Seq != cursor.expectedSeq {
		fail(result, cursor.expectedSeq, "row "+strconv.FormatInt(cursor.expectedSeq, 10)+
			" is missing from the chain")

		return true
	}
	if cursor.signedFrom != 0 && !row.Keyed() {
		fail(result, row.Seq, "the row is unsigned though row "+
			strconv.FormatInt(cursor.signedFrom, 10)+" before it was signed")

		return true
	}
	if detail := v.checkRow(row, cursor.expectedPrev); detail != "" {
		if detail == keyMissingDetail {
			markKeyMissing(result, row)

			return true
		}
		fail(result, row.Seq, detail)

		return true
	}
	if seal, sealed := seals[row.Seq]; sealed && seal.HeadHash != row.Hash {
		fail(result, row.Seq, "the seal ending at this row records another hash")

		return true
	}

	return false
}

func markKeyMissing(result *VerifyResult, row *aiaudit.AIAuditEvent) {
	result.Status = aiaudit.VerificationKeyMissing
	seq := row.Seq
	result.FailedSeq = &seq
	result.Detail = "row " + strconv.FormatInt(row.Seq, 10) +
		" is signed with key " + strconv.Quote(row.HashKeyID) +
		", which is not configured"
}

const keyMissingDetail = "key missing"

func (v *Verifier) checkRow(row *aiaudit.AIAuditEvent, expectedPrev string) string {
	if row.PrevHash != expectedPrev {
		return "the row does not name the hash of the row before it"
	}

	key, missing := v.keyring.KeyFor(row)
	if missing {
		return keyMissingDetail
	}
	ok, err := aiaudit.VerifyHash(row, key)
	if err != nil {
		return "the row's hash cannot be recomputed: " + err.Error()
	}
	if !ok {
		return "the row's content no longer matches its hash"
	}

	return ""
}

// anchor is the hash the oldest retained row must name: the genesis hash for
// a chain never pruned, else the head of the seal the prune cut at.
func (v *Verifier) anchor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	first int64,
) (headHash, problem string) {
	if first == 1 {
		return aiaudit.GenesisHash, ""
	}

	seal, err := v.ledger.GetSealEndingAt(ctx, tenantInfo, first-1)
	if err != nil {
		return "", "the seal before the oldest row cannot be read: " + err.Error()
	}
	if seal == nil {
		return "", "the chain was cut where no seal ends"
	}

	return seal.HeadHash, ""
}

func (v *Verifier) verifyFullyPruned(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	head *aiaudit.AIAuditChainHead,
	result *VerifyResult,
) (*VerifyResult, error) {
	seal, err := v.ledger.GetSealEndingAt(ctx, tenantInfo, head.LastSeq)
	if err != nil {
		return nil, err
	}
	if seal == nil || seal.HeadHash != head.LastHash {
		return fail(result, head.LastSeq, "every row is gone and no seal accounts for them"), nil
	}
	result.VerifiedSeq = head.LastSeq

	return result, nil
}

func (v *Verifier) pageSeals(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	from, to int64,
) (map[int64]*aiaudit.AIAuditSeal, error) {
	seals, err := v.ledger.ListSeals(ctx, aiauditports.ListAIAuditSealsRequest{
		TenantInfo: tenantInfo,
		FromSeq:    from,
		Limit:      int(to-from) + 1,
	})
	if err != nil {
		return nil, err
	}

	bySeq := make(map[int64]*aiaudit.AIAuditSeal, len(seals))
	for _, seal := range seals {
		if seal.ToSeq <= to {
			bySeq[seal.ToSeq] = seal
		}
	}

	return bySeq, nil
}

func fail(result *VerifyResult, seq int64, detail string) *VerifyResult {
	result.Status = aiaudit.VerificationMismatch
	failed := seq
	result.FailedSeq = &failed
	result.Detail = detail

	return result
}

func (v *Verifier) publish(ctx context.Context, tenantInfo pagination.TenantInfo) {
	if v.realtime == nil {
		return
	}
	if err := v.realtime.PublishResourceInvalidation(ctx,
		&realtimeports.PublishResourceInvalidationRequest{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Resource:       aiauditports.AIAuditChainResource,
			Action:         "verified",
			ActorType:      permissionports.PrincipalTypeSystem,
			ActorID:        permissionports.SystemPrincipalID,
		}); err != nil {
		v.l.Warn("failed to announce an AI audit verification", zap.Error(err))
	}
}

// alarm records a failed verification in the audit log as critical and tells
// everyone who may read the trail. A notice about the same failure is sent
// once a week at most.
func (v *Verifier) alarm(ctx context.Context, result *VerifyResult) {
	tenantInfo := result.TenantInfo
	failedSeq := int64(0)
	if result.FailedSeq != nil {
		failedSeq = *result.FailedSeq
	}

	v.l.Error("the AI audit chain failed verification",
		zap.String("organizationId", tenantInfo.OrgID.String()),
		zap.String("businessUnitId", tenantInfo.BuID.String()),
		zap.String("status", string(result.Status)),
		zap.Int64("failedSeq", failedSeq),
		zap.String("detail", result.Detail),
	)

	if v.audit != nil {
		actor := permissionports.SystemAuditActor()
		if err := v.audit.LogAction(ctx, &auditports.LogActionParams{
			Resource:       permission.ResourceAIAuditTrail,
			ResourceID:     tenantInfo.OrgID.String(),
			Operation:      permission.OpRead,
			PrincipalType:  actor.PrincipalType,
			PrincipalID:    actor.PrincipalID,
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Critical:       true,
			CurrentState: map[string]any{
				"event":           "chain_verification_failed",
				mismatchKeyStatus: string(result.Status),
				"failedSeq":       failedSeq,
				"detail":          result.Detail,
			},
		}, auditservice.WithComment("The AI audit trail failed verification")); err != nil {
			v.l.Error("failed to audit an AI audit chain mismatch", zap.Error(err))
		}
	}

	if v.notifier == nil {
		return
	}

	now := v.now()
	correlation := fmt.Sprintf("%s:%s:%s:%d",
		tenantInfo.OrgID, tenantInfo.BuID, result.Status, failedSeq)
	if _, err := v.notifier.NotifyPermitted(ctx, notificationservice.NotifyPermittedRequest{
		Tenant:      pagination.TenantInfo{OrgID: tenantInfo.OrgID, BuID: tenantInfo.BuID},
		Resource:    permission.ResourceAIAuditTrail,
		Operation:   permission.OpRead,
		Limit:       mismatchNoticeLimit,
		Now:         now.Unix(),
		DedupeSince: now.Add(-mismatchNoticeWindow).Unix(),
		Notification: notification.Notification{
			EventType: aiauditports.AIAuditChainMismatchEvent,
			Priority:  notification.PriorityCritical,
			Title:     "The AI audit trail failed verification",
			Message: "A row of the AI audit trail no longer matches its chain. " +
				"Open the audit trail in AI Control to see where.",
			Source:        mismatchNoticeSource,
			CorrelationID: &correlation,
			Data: map[string]any{
				mismatchKeyKind:   aiauditports.AIAuditChainMismatchEvent,
				mismatchKeyStatus: string(result.Status),
				"failedSeq":       failedSeq,
			},
		},
	}); err != nil {
		v.l.Error("failed to tell anyone the AI audit chain failed verification", zap.Error(err))
	}
}

// ChainStatus is a tenant's chain as a reader sees it.
func (v *Verifier) ChainStatus(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*aiauditports.AIAuditChainStatus, error) {
	head, err := v.ledger.GetChainHead(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	first, found, err := v.ledger.FirstSeq(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	status := &aiauditports.AIAuditChainStatus{
		OrganizationID:         tenantInfo.OrgID,
		BusinessUnitID:         tenantInfo.BuID,
		Signed:                 v.keyring.Signed(),
		ActiveKeyID:            v.keyring.ActiveKeyID(),
		LastSeq:                head.LastSeq,
		LastHash:               head.LastHash,
		SealedThroughSeq:       head.LastSeq,
		LastVerifiedSeq:        head.LastVerifiedSeq,
		LastVerifiedAt:         head.LastVerifiedAt,
		LastVerificationStatus: head.LastVerificationStatus,
		FailedSeq:              head.LastVerificationFailedSeq,
		Detail:                 head.LastVerificationDetail,
	}
	if found {
		status.FirstSeq = first
	}

	return status, nil
}
