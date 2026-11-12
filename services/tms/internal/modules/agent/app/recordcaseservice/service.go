package recordcaseservice

import (
	"context"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/caseownerports"
	"github.com/emoss08/trenova/shared/stringutils"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/app/deskrecords"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskattentionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordcaseports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgzone"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenttrackingports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxSignals         = 200
	decidedLookback    = 24 * 3600
	readingRetryAfter  = 120
	maxHandOffNote     = 1200
	readingKeptSeconds = 7 * 24 * 3600
	maxMessageExhibits = 2
	maxConditions      = 3
)

var openStatuses = []watchtower.Status{
	watchtower.StatusOpen,
	watchtower.StatusAcknowledged,
	watchtower.StatusInProgress,
}

// caseStatuses are what a case reads: what is open, and what was put away,
// which is still true of the record and comes back.
var caseStatuses = append(slices.Clone(openStatuses), watchtower.StatusSnoozed)

type Params struct {
	fx.In

	Logger      *zap.Logger
	Watchtower  watchtowerports.WatchtowerService
	Exceptions  watchtowerports.ExceptionService
	Labels      *deskrecords.Labeler
	Repo        recordcaseports.RecordCaseRepository
	Proposals   agentproposalports.AgentProposalRepository
	Definitions agentdefinitionports.AgentDefinitionRepository
	Runs        agentruntimeports.AgentRunService
	RunRecords  agentruntimeports.AgentRunRepository
	Readings    recordcaseports.ReadingStarter
	Tracking    shipmenttrackingports.ShipmentTrackingReader
	Invoices    invoiceports.InvoiceRepository
	Disputes    invoiceports.InvoiceDisputeRepository
	Orgs        tenantports.OrganizationRepository
	Shadow      *agentshadow.Resolver
	Owners      caseownerports.Service
}

type Service struct {
	l           *zap.Logger
	watchtower  watchtowerports.WatchtowerService
	exceptions  watchtowerports.ExceptionService
	labels      *deskrecords.Labeler
	repo        recordcaseports.RecordCaseRepository
	proposals   agentproposalports.AgentProposalRepository
	definitions agentdefinitionports.AgentDefinitionRepository
	runs        agentruntimeports.AgentRunService
	runRecords  agentruntimeports.AgentRunRepository
	readings    recordcaseports.ReadingStarter
	orgs        orgzone.OrganizationReader
	shadow      *agentshadow.Resolver
	owners      caseownerports.Service
	subjects    map[recordkind.Kind]subjectReader
	stands      map[recordkind.Kind]standReader
	now         func() int64
}

func New(p Params) *Service { //nolint:gocritic // fx params are passed by value
	return &Service{
		l:           p.Logger.Named("service.record-case"),
		watchtower:  p.Watchtower,
		exceptions:  p.Exceptions,
		labels:      p.Labels,
		repo:        p.Repo,
		proposals:   p.Proposals,
		definitions: p.Definitions,
		runs:        p.Runs,
		runRecords:  p.RunRecords,
		readings:    p.Readings,
		orgs:        p.Orgs,
		shadow:      p.Shadow,
		owners:      p.Owners,
		subjects: map[recordkind.Kind]subjectReader{
			recordkind.Kind(permission.ResourceShipment): shipmentSubject(p.Tracking),
			recordkind.Kind(permission.ResourceInvoice): invoiceSubject(
				p.Invoices,
				p.Disputes,
			),
			recordkind.Kind(permission.ResourceInvoiceDispute): disputeSubject(
				p.Invoices,
				p.Disputes,
			),
		},
		stands: map[recordkind.Kind]standReader{
			recordkind.Kind(permission.ResourceInvoice): invoiceStand(p.Disputes),
		},
		now: timeutils.NowUnix,
	}
}

func AsService(s *Service) recordcaseports.RecordCaseService { return s }

func AsComposer(s *Service) *Composer { return &Composer{service: s} }

type composed struct {
	record    *deskattentionports.Record
	items     []*watchtower.Item
	away      *recordcase.Away
	subject   *subject
	options   []*recordcase.Option
	chosen    *recordcase.Option
	decision  *recordcase.Decision
	drafting  bool
	draftNote string
	standing  *caseownerports.Standing
	handOff   *recordcase.HandOff
	exhibits  []recordcase.Exhibit
	stream    string
	zone      *time.Location
	now       int64
}

func (s *Service) Case(
	ctx context.Context,
	req *recordcaseports.CaseRequest,
	actor *permissionports.RequestActor,
) (*recordcase.Case, error) {
	if stood := s.standsFor(ctx, req); stood != nil {
		found, err := s.caseOf(ctx, stood, actor)
		if err == nil ||
			(!errortypes.IsNotFoundError(err) && !errortypes.IsAuthorizationError(err)) {
			return found, err
		}
	}

	return s.caseOf(ctx, req, actor)
}

func (s *Service) standsFor(
	ctx context.Context,
	req *recordcaseports.CaseRequest,
) *recordcaseports.CaseRequest {
	stand, ok := s.stands[req.Kind]
	if !ok {
		return nil
	}
	kind, id, err := stand(ctx, req.TenantInfo, req.ID)
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			s.l.Warn("could not look for the record a case stands on", zap.Error(err))
		}

		return nil
	}
	if id.IsNil() {
		return nil
	}

	return &recordcaseports.CaseRequest{
		TenantInfo: req.TenantInfo,
		Kind:       kind,
		ID:         id,
		Timezone:   req.Timezone,
	}
}

func (s *Service) caseOf(
	ctx context.Context,
	req *recordcaseports.CaseRequest,
	actor *permissionports.RequestActor,
) (*recordcase.Case, error) {
	built, err := s.compose(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	fingerprint := fingerprintOf(built)
	reading, err := s.repo.Reading(ctx, recordcaseports.ReadingRequest{
		TenantInfo:  req.TenantInfo,
		RecordID:    req.ID,
		Fingerprint: fingerprint,
	})
	if err != nil {
		return nil, err
	}
	status := recordcase.ReadingStatusUnavailable
	if len(built.items) > 0 {
		status = s.readingStatus(ctx, req, actor, built, reading, fingerprint)
	}
	if !reading.Current(fingerprint) || reading.Status != recordcase.ReadingStatusReady {
		reading = nil
	}

	return assemble(built, reading, status), nil
}

func (s *Service) readingStatus(
	ctx context.Context,
	req *recordcaseports.CaseRequest,
	actor *permissionports.RequestActor,
	built *composed,
	reading *recordcase.Reading,
	fingerprint string,
) recordcase.ReadingStatus {
	if reading.Current(fingerprint) && reading.Status != recordcase.ReadingStatusWriting {
		return reading.Status
	}
	if reading.Current(fingerprint) && reading.RequestedAt > built.now-readingRetryAfter {
		return recordcase.ReadingStatusWriting
	}

	claimed, err := s.repo.ClaimReading(ctx, &recordcaseports.ClaimReadingRequest{
		TenantInfo:   req.TenantInfo,
		RecordKind:   req.Kind,
		RecordID:     req.ID,
		Fingerprint:  fingerprint,
		Now:          built.now,
		StaleBefore:  built.now - readingRetryAfter,
		ForgetBefore: built.now - readingKeptSeconds,
	})
	if err != nil {
		s.l.Warn("a case reading could not be claimed", zap.Error(err))

		return recordcase.ReadingStatusUnavailable
	}
	if !claimed {
		return recordcase.ReadingStatusWriting
	}
	if err = s.readings.StartReading(ctx, &recordcaseports.WriteReadingRequest{
		TenantInfo:  req.TenantInfo,
		RecordKind:  req.Kind,
		RecordID:    req.ID,
		Fingerprint: fingerprint,
		Timezone:    built.zone.String(),
		UserID:      actor.UserID,
	}); err != nil {
		s.l.Warn("a case reading could not be started", zap.Error(err))

		return recordcase.ReadingStatusUnavailable
	}

	return recordcase.ReadingStatusWriting
}

func (s *Service) compose(
	ctx context.Context,
	req *recordcaseports.CaseRequest,
	actor *permissionports.RequestActor,
) (*composed, error) {
	if !req.Kind.Known() || req.Kind.Resource() == "" {
		return nil, errortypes.NewNotFoundError("That record has no case")
	}
	if prefix, ok := req.Kind.IDPrefix(); !ok || req.ID.Prefix() != prefix {
		return nil, errortypes.NewNotFoundError("That record has no case")
	}

	record := &deskattentionports.Record{Kind: req.Kind, ID: req.ID}
	if err := s.labels.Label(
		ctx,
		req.TenantInfo,
		actor,
		[]*deskattentionports.Record{record},
	); err != nil {
		return nil, err
	}
	if !record.Readable {
		return nil, errortypes.NewAuthorizationError("You may not read this record")
	}
	if record.Label == "" {
		return nil, errortypes.NewNotFoundError("That record was not found")
	}

	page, err := s.watchtower.List(ctx, watchtowerports.ListWatchtowerItemsRequest{
		TenantInfo:     req.TenantInfo,
		UnresolvedOnly: true,
		Statuses:       caseStatuses,
		AboutIDs:       []pulid.ID{req.ID},
		OrderByDue:     true,
		First:          maxSignals,
	}, actor)
	if err != nil {
		return nil, err
	}

	standing, err := s.standing(ctx, req, actor, len(page.Items) > 0)
	if err != nil {
		return nil, err
	}

	built := &composed{
		record:   record,
		items:    page.Items,
		away:     awayOf(page.Items),
		standing: standing,
		zone:     s.zone(ctx, req),
		now:      s.now(),
	}

	if built.options, err = s.repo.Options(ctx, recordcaseports.RecordRequest{
		TenantInfo: req.TenantInfo,
		RecordID:   req.ID,
	}); err != nil {
		return nil, err
	}
	built.chosen = recordcase.Chosen(built.options)

	if err = s.decide(ctx, req, built); err != nil {
		return nil, err
	}
	if built.decision != nil && !offeredBy(built.options, built.decision.AgentDefinitionID) {
		built.options, built.chosen = nil, nil
	}

	read := &subjectRead{
		tenant: req.TenantInfo,
		id:     req.ID,
		zone:   built.zone,
		site:   s.site(ctx, req.TenantInfo),
		now:    built.now,
		items:  built.items,
		chosen: built.chosen,
		done:   built.decision != nil && built.decision.Phase == recordcase.PhaseDone,
		stopped: built.decision != nil && (built.decision.Phase == recordcase.PhaseFailed ||
			built.decision.Phase == recordcase.PhaseRunning ||
			built.decision.Phase == recordcase.PhaseDeclined),
	}
	reader, known := s.subjects[req.Kind]
	if !known {
		reader = plainSubject
	}
	if built.subject, err = reader(ctx, read); err != nil {
		return nil, err
	}
	if built.subject == nil {
		built.subject = &subject{}
	}
	built.exhibits = exhibitsOf(built)
	built.handOff = s.handOffOf(ctx, req, built)

	return built, nil
}

// site is the organization's zone: the clock a stop's window is kept on when
// its location names none, which is the one the tools that move a window
// read and write it in.
func (s *Service) site(ctx context.Context, tenant pagination.TenantInfo) *time.Location {
	loc, err := orgzone.Location(ctx, s.orgs, tenant)
	if err != nil || loc == nil {
		return nil
	}

	return loc
}

func (s *Service) zone(ctx context.Context, req *recordcaseports.CaseRequest) *time.Location {
	if req.Timezone != "" {
		if loc, err := time.LoadLocation(req.Timezone); err == nil {
			return loc
		}
	}
	loc, err := orgzone.Location(ctx, s.orgs, req.TenantInfo)
	if err != nil || loc == nil {
		return time.UTC
	}

	return loc
}

// offeredBy reports whether the options on a record are the deciding agent's
// own. The options one agent offered are no account of what another agent's
// plan would do, so a case whose decision is another agent's shows none.
func offeredBy(options []*recordcase.Option, agentID pulid.ID) bool {
	for _, option := range options {
		if !option.AgentDefinitionID.IsNil() && !agentID.IsNil() &&
			option.AgentDefinitionID != agentID {
			return false
		}
	}

	return true
}

/*
standing is how the reader stands to the case's owner. A case someone else
owns is refused to a reader who neither covers for the owner nor sees every
case. One with something open and no owner yet is routed to whoever its
record names, so a record that gained a driver or a biller after its first
signal still finds its owner.
*/
func (s *Service) standing(
	ctx context.Context,
	req *recordcaseports.CaseRequest,
	actor *permissionports.RequestActor,
	open bool,
) (*caseownerports.Standing, error) {
	record := caseownerports.RecordRequest{TenantInfo: req.TenantInfo, RecordID: req.ID}
	standing, err := s.owners.Standing(ctx, record, actor)
	if err != nil {
		return nil, err
	}
	if standing.Owner == nil && open {
		s.owners.Route(ctx, caseownerports.CandidatesRequest{
			TenantInfo: req.TenantInfo,
			Kind:       req.Kind,
			RecordID:   req.ID,
		})
		if standing, err = s.owners.Standing(ctx, record, actor); err != nil {
			return nil, err
		}
	}
	if !standing.CanSee {
		return nil, errortypes.NewAuthorizationError("This case is owned by someone else")
	}

	return standing, nil
}

var errNotTheOwner = errortypes.NewAuthorizationError(
	"This case is owned by someone else. Its owner decides it, or someone who assigns cases " +
		"can hand it to you.",
)

func (s *Service) AssignCase(
	ctx context.Context,
	req *recordcaseports.AssignCaseRequest,
	actor *permissionports.RequestActor,
) error {
	built, err := s.compose(ctx, &recordcaseports.CaseRequest{
		TenantInfo: req.TenantInfo,
		Kind:       req.Kind,
		ID:         req.ID,
	}, actor)
	if err != nil {
		return err
	}

	assign := &caseownerports.AssignRequest{
		TenantInfo: req.TenantInfo,
		Kind:       req.Kind,
		RecordID:   req.ID,
		OwnerID:    req.OwnerID,
		Label:      built.record.Label,
	}
	if req.OwnerID.IsNil() || req.OwnerID == actor.UserID {
		_, err = s.owners.Claim(ctx, assign, actor)

		return err
	}
	_, err = s.owners.Assign(ctx, assign, actor)

	return err
}

func (s *Service) ReleaseCase(
	ctx context.Context,
	req *recordcaseports.CaseRequest,
	actor *permissionports.RequestActor,
) error {
	if _, err := s.compose(ctx, req, actor); err != nil {
		return err
	}

	return s.owners.Release(ctx, &caseownerports.ReleaseRequest{
		TenantInfo: req.TenantInfo,
		Kind:       req.Kind,
		RecordID:   req.ID,
	}, actor)
}

/*
handOffOf is the condition an agent can be asked to work: the first one open
about the record, when it names a subject an agent run can start on. A case
with only that, raised by the Watchtower and worked by nobody, is a reading
and nothing to do; this is what lets a person put an agent on it. When an
agent had it and finished without proposing anything, what it said is kept,
so the person reads why and hands it on or accepts the risk.
*/
func (s *Service) handOffOf(
	ctx context.Context,
	req *recordcaseports.CaseRequest,
	built *composed,
) *recordcase.HandOff {
	primary := primaryCondition(built)
	if primary == nil || primary.SubjectType == "" || primary.SubjectID.IsNil() {
		return nil
	}

	out := &recordcase.HandOff{ItemID: primary.ID, Kind: primary.SourceKind}
	if !primary.HasAgent() {
		return out
	}
	out.AgentDefinitionID = primary.AgentDefinitionID

	run, err := s.runRecords.GetByID(ctx, agentruntimeports.GetAgentRunByIDRequest{
		ID:         primary.AgentRunID,
		TenantInfo: &req.TenantInfo,
	})
	if err != nil {
		return out
	}
	if primary.AgentWorkFrom(run).Active() {
		out.Working = true
		built.stream = run.WorkflowID

		return out
	}
	out.Note = stringutils.Ellipsize(
		stringutils.FirstNonEmptyTrimmed(run.ErrorMessage, run.Summary),
		maxHandOffNote,
	)

	return out
}
