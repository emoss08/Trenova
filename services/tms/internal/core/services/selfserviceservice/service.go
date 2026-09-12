// Package selfserviceservice owns what a driver can do to their own record
// from the portal and what the office has to answer: the policies they sign,
// and the changes they ask to make.
//
// Two decisions carry the package. An acknowledgement copies what was signed
// — the version label and the document checksum — because the policy row
// will move on and the signature must not; so changing the words under a
// signed policy without changing its version is refused. And a change request
// stores before-and-after pairs rather than a copy of the wanted record, so a
// manager sees exactly what is being asked and an approval applies exactly
// that.
package selfserviceservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// WorkerUpdater is the one write this service makes to a worker: applying an
// approved change. It is a port rather than the worker service so the test can
// see what was written.
type WorkerUpdater interface {
	Update(
		ctx context.Context,
		entity *worker.Worker,
		actor *services.RequestActor,
	) (*worker.Worker, error)
}

// WorkerReader is the one read this service makes of the roster.
type WorkerReader interface {
	GetByID(ctx context.Context, req repositories.GetWorkerByIDRequest) (*worker.Worker, error)
}

// DocumentChecksums is the slice of the document store an acknowledgement
// needs: the checksum of what was signed, so the signature can be tied to the
// bytes and not just the row.
type DocumentChecksums interface {
	Checksum(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		documentID pulid.ID,
	) (string, error)
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	Repo          repositories.SelfServiceRepository
	WorkerRepo    repositories.WorkerRepository
	WorkerUpdater WorkerUpdater
	Checksums     DocumentChecksums `optional:"true"`
	AuditService  services.AuditService
	DriverNotify  *drivernotificationservice.Service `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	repo          repositories.SelfServiceRepository
	workerRepo    WorkerReader
	workerUpdater WorkerUpdater
	checksums     DocumentChecksums
	auditService  services.AuditService
	driverNotify  *drivernotificationservice.Service
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.self-service"),
		repo:          p.Repo,
		workerRepo:    p.WorkerRepo,
		workerUpdater: p.WorkerUpdater,
		checksums:     p.Checksums,
		auditService:  p.AuditService,
		driverNotify:  p.DriverNotify,
	}
}

// Deps is the constructor shape tests use to swap in fakes.
type Deps struct {
	Logger        *zap.Logger
	Repo          repositories.SelfServiceRepository
	WorkerRepo    WorkerReader
	WorkerUpdater WorkerUpdater
	Checksums     DocumentChecksums
	AuditService  services.AuditService
}

func NewWithDeps(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		l:             logger.Named("service.self-service"),
		repo:          d.Repo,
		workerRepo:    d.WorkerRepo,
		workerUpdater: d.WorkerUpdater,
		checksums:     d.Checksums,
		auditService:  d.AuditService,
	}
}

type auditParams struct {
	resource   permission.Resource
	resourceID string
	operation  permission.Operation
	userID     pulid.ID
	tenantInfo pagination.TenantInfo
	current    any
	previous   any
	comment    string
}

func (s *Service) audit(p auditParams) {
	if s.auditService == nil || p.userID.IsNil() {
		return
	}
	params := &services.LogActionParams{
		Resource:       p.resource,
		ResourceID:     p.resourceID,
		Operation:      p.operation,
		UserID:         p.userID,
		CurrentState:   jsonutils.MustToJSON(p.current),
		OrganizationID: p.tenantInfo.OrgID,
		BusinessUnitID: p.tenantInfo.BuID,
	}
	opts := []services.LogOption{auditservice.WithComment(p.comment)}
	if p.previous != nil {
		params.PreviousState = jsonutils.MustToJSON(p.previous)
		opts = append(opts, auditservice.WithDiff(p.previous, p.current))
	}
	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}
}

func (s *Service) ListPolicies(
	ctx context.Context,
	req *repositories.ListWorkerPoliciesRequest,
) ([]*worker.WorkerPolicy, error) {
	return s.repo.ListPolicies(ctx, req)
}

func (s *Service) GetPolicy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.WorkerPolicy, error) {
	return s.repo.GetPolicyByID(ctx, &repositories.GetWorkerPolicyByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}

// PolicyRequest is a policy being written or rewritten.
type PolicyRequest struct {
	Entity     *worker.WorkerPolicy
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
}

func (s *Service) CreatePolicy(
	ctx context.Context,
	req *PolicyRequest,
) (*worker.WorkerPolicy, error) {
	entity := req.Entity
	entity.OrganizationID = req.TenantInfo.OrgID
	entity.BusinessUnitID = req.TenantInfo.BuID
	entity.CreatedByID = req.UserID
	entity.Normalise()
	if entity.EffectiveFrom <= 0 {
		entity.EffectiveFrom = timeutils.NowUnix()
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.CreatePolicy(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(auditParams{
		resource:   permission.ResourceWorkerPolicy,
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    created,
		comment:    "Policy published",
	})

	return created, nil
}

// UpdatePolicy rewrites a policy. The words behind a signed version cannot
// change under the same label: every existing acknowledgement records the
// label it was given for, and rewriting the text beneath it would turn each
// one into a signature on words nobody saw. A new label is a new thing to
// sign, and everybody bound by it goes back to outstanding.
func (s *Service) UpdatePolicy(
	ctx context.Context,
	req *PolicyRequest,
) (*worker.WorkerPolicy, error) {
	entity := req.Entity
	entity.OrganizationID = req.TenantInfo.OrgID
	entity.BusinessUnitID = req.TenantInfo.BuID
	entity.Normalise()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	original, err := s.repo.GetPolicyByID(ctx, &repositories.GetWorkerPolicyByIDRequest{
		ID:         entity.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if entity.ContentChanged(original) && entity.VersionLabel == original.VersionLabel {
		signed, cErr := s.repo.CountAcknowledgements(
			ctx,
			req.TenantInfo,
			original.ID,
			original.VersionLabel,
		)
		if cErr != nil {
			return nil, cErr
		}
		if signed > 0 {
			return nil, errortypes.NewValidationError(
				"versionLabel",
				errortypes.ErrInvalidOperation,
				"{0} person(s) have signed version {1} — give the new text a new version so their signatures stay on what they read", signed, original.VersionLabel,
			)
		}
	}

	entity.CreatedByID = original.CreatedByID
	entity.CreatedAt = original.CreatedAt

	updated, err := s.repo.UpdatePolicy(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(auditParams{
		resource:   permission.ResourceWorkerPolicy,
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    updated,
		previous:   original,
		comment:    "Policy updated",
	})

	return updated, nil
}

// PolicyCompliance is who has signed what is in force and who has not,
// derived on read: the roster and the acknowledgements both move.
type PolicyCompliance struct {
	Policy      *worker.WorkerPolicy
	Rows        []repositories.PolicyComplianceRow
	Signed      int
	Outstanding int
}

func (s *Service) PolicyCompliance(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	policyID pulid.ID,
) (*PolicyCompliance, error) {
	policy, err := s.GetPolicy(ctx, tenantInfo, policyID)
	if err != nil {
		return nil, err
	}

	rows, err := s.repo.PolicyCompliance(ctx, &repositories.PolicyComplianceRequest{
		TenantInfo:   tenantInfo,
		PolicyID:     policy.ID,
		VersionLabel: policy.VersionLabel,
		AppliesTo:    policy.AppliesTo,
	})
	if err != nil {
		return nil, err
	}

	out := &PolicyCompliance{Policy: policy, Rows: rows}
	for _, row := range rows {
		if row.AcknowledgedAt > 0 {
			out.Signed++
		} else {
			out.Outstanding++
		}
	}

	return out, nil
}

// PolicyStanding is one policy from one worker's side: the policy, and their
// signature on the version in force if they have given one.
type PolicyStanding struct {
	Policy          *worker.WorkerPolicy
	Acknowledgement *worker.WorkerPolicyAcknowledgement
}

// Outstanding reports whether the worker still owes a signature.
func (p PolicyStanding) Outstanding() bool { return p.Acknowledgement == nil }

// PoliciesForWorker is every policy in force that binds the worker, with
// where they stand on it. A policy for employees is not shown to a
// contractor at all: being asked to sign a handbook that does not apply to
// you is worse than not seeing it.
func (s *Service) PoliciesForWorker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	wrk *worker.Worker,
) ([]PolicyStanding, error) {
	policies, err := s.repo.ListPolicies(ctx, &repositories.ListWorkerPoliciesRequest{
		TenantInfo: tenantInfo,
		ActiveOnly: true,
		AsOf:       timeutils.NowUnix(),
	})
	if err != nil {
		return nil, err
	}

	acks, err := s.repo.ListAcknowledgements(ctx, &repositories.ListPolicyAcknowledgementsRequest{
		TenantInfo: tenantInfo,
		WorkerID:   wrk.ID,
	})
	if err != nil {
		return nil, err
	}
	signed := make(map[string]*worker.WorkerPolicyAcknowledgement, len(acks))
	for _, ack := range acks {
		signed[ack.PolicyID.String()+"@"+ack.VersionLabel] = ack
	}

	out := make([]PolicyStanding, 0, len(policies))
	for _, policy := range policies {
		if !policy.AppliesTo.Covers(wrk.Type) {
			continue
		}
		out = append(out, PolicyStanding{
			Policy:          policy,
			Acknowledgement: signed[policy.ID.String()+"@"+policy.VersionLabel],
		})
	}

	return out, nil
}

// AcknowledgeRequest is one worker signing one policy.
type AcknowledgeRequest struct {
	PolicyID      pulid.ID
	Worker        *worker.Worker
	SignatureName string
	IP            string
	UserAgent     string
	TenantInfo    pagination.TenantInfo
}

// Acknowledge records the signature. The typed name has to be given when the
// policy asks for one, and it has to be the worker's own name — a signature
// is a claim that a specific person agreed, and "ok" is not a person.
func (s *Service) Acknowledge(
	ctx context.Context,
	req *AcknowledgeRequest,
) (*worker.WorkerPolicyAcknowledgement, error) {
	policy, err := s.GetPolicy(ctx, req.TenantInfo, req.PolicyID)
	if err != nil {
		return nil, err
	}
	if policy.Status != domaintypes.StatusActive {
		return nil, errortypes.NewValidationError(
			"policyId",
			errortypes.ErrInvalidOperation,
			"That policy has been retired",
		)
	}
	if !policy.AppliesTo.Covers(req.Worker.Type) {
		return nil, errortypes.NewValidationError(
			"policyId",
			errortypes.ErrInvalidOperation,
			"That policy does not apply to you",
		)
	}

	signature := strings.TrimSpace(req.SignatureName)
	if policy.RequiresSignature {
		if signature == "" {
			return nil, errortypes.NewValidationError(
				"signatureName",
				errortypes.ErrRequired,
				"Type your full name to sign",
			)
		}
		if !namesMatch(signature, req.Worker.FirstName, req.Worker.LastName) {
			return nil, errortypes.NewValidationError(
				"signatureName",
				errortypes.ErrInvalid,
				"Sign as {0} {1} — the name on your record", req.Worker.FirstName, req.Worker.LastName,
			)
		}
	}

	existing, err := s.repo.ListAcknowledgements(
		ctx,
		&repositories.ListPolicyAcknowledgementsRequest{
			TenantInfo:   req.TenantInfo,
			PolicyID:     policy.ID,
			WorkerID:     req.Worker.ID,
			VersionLabel: policy.VersionLabel,
			Limit:        1,
		},
	)
	if err != nil {
		return nil, err
	}
	// Signing the same words twice adds nothing, and a second row would be the
	// one an audit asks about.
	if len(existing) > 0 {
		return existing[0], nil
	}

	entity := &worker.WorkerPolicyAcknowledgement{
		OrganizationID:     req.TenantInfo.OrgID,
		BusinessUnitID:     req.TenantInfo.BuID,
		PolicyID:           policy.ID,
		WorkerID:           req.Worker.ID,
		VersionLabel:       policy.VersionLabel,
		AcknowledgedAt:     timeutils.NowUnix(),
		SignatureName:      signature,
		SignatureIP:        truncate(req.IP, 64),
		SignatureUserAgent: truncate(req.UserAgent, 255),
	}
	if !policy.DocumentID.IsNil() && s.checksums != nil {
		checksum, cErr := s.checksums.Checksum(ctx, req.TenantInfo, policy.DocumentID)
		if cErr != nil {
			return nil, cErr
		}
		entity.DocumentChecksum = checksum
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.CreateAcknowledgement(ctx, entity)
	if err != nil {
		return nil, err
	}

	return created, nil
}

// namesMatch is a forgiving comparison: case and surrounding space do not
// make a different person, but a different name does.
func namesMatch(signature, first, last string) bool {
	normalise := func(value string) string {
		return strings.Join(strings.Fields(strings.ToLower(value)), " ")
	}
	return normalise(signature) == normalise(first+" "+last)
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func (s *Service) ListAcknowledgements(
	ctx context.Context,
	req *repositories.ListPolicyAcknowledgementsRequest,
) ([]*worker.WorkerPolicyAcknowledgement, error) {
	return s.repo.ListAcknowledgements(ctx, req)
}

// publishNotice tells everybody a policy binds that there is something to
// sign. It is best-effort: a notification that could not be sent is a policy
// still waiting on the portal, not a policy nobody can sign.
func (s *Service) notifyDriver(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	eventType string,
	context documenttemplate.DriverNotificationContext,
	related map[string]any,
) {
	if s.driverNotify == nil {
		return
	}
	s.driverNotify.Notify(ctx, &drivernotificationservice.DriverNotification{
		TenantInfo:      tenantInfo,
		WorkerID:        workerID,
		EventType:       eventType,
		Priority:        notification.PriorityHigh,
		Context:         context,
		Link:            "/dash/profile",
		RelatedEntities: related,
	})
}
