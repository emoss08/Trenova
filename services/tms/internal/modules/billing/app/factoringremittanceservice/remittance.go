package factoringremittanceservice

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/app/factoringaction"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringremittanceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	remittanceResource      = permission.ResourceFactoringRemittance
	remittanceRealtimeEvent = "updated"
)

type RemittanceParams struct {
	fx.In

	DB          kernelports.Transactor
	Remittances factoringremittanceports.RemittanceRepository
	Submissions factoringsubmissionports.SubmissionRepository
	Ledger      factoringledgerports.LedgerService
	Factors     factoringcompanyports.FactoringCompanyRepository
	Documents   documentports.InvoiceDocumentService
	Content     documentports.DocumentContentService            `optional:"true"`
	Completion  llmports.CompletionService                      `optional:"true"`
	Starter     factoringremittanceports.RemittanceStageStarter `optional:"true"`
	Events      agenteventports.AgentEventPublisher             `optional:"true"`
	Audit       auditports.AuditService
	Realtime    realtimeports.RealtimeService `optional:"true"`
	SheetLimits rateimport.Limits             `optional:"true"`
	Logger      *zap.Logger
}

type RemittanceService struct {
	db          kernelports.Transactor
	remittances factoringremittanceports.RemittanceRepository
	submissions factoringsubmissionports.SubmissionRepository
	ledger      factoringledgerports.LedgerService
	factors     factoringcompanyports.FactoringCompanyRepository
	documents   documentports.InvoiceDocumentService
	content     documentports.DocumentContentService
	completion  llmports.CompletionService
	starter     factoringremittanceports.RemittanceStageStarter
	events      agenteventports.AgentEventPublisher
	rec         factoringaction.Recorder
	sheetLimits rateimport.Limits
	l           *zap.Logger
	now         func() int64
}

var _ factoringremittanceports.RemittanceService = (*RemittanceService)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func NewRemittanceService(p RemittanceParams) factoringremittanceports.RemittanceService {
	l := p.Logger.Named("service.factoring-remittance")
	return &RemittanceService{
		db:          p.DB,
		remittances: p.Remittances,
		submissions: p.Submissions,
		ledger:      p.Ledger,
		factors:     p.Factors,
		documents:   p.Documents,
		content:     p.Content,
		completion:  p.Completion,
		starter:     p.Starter,
		sheetLimits: p.SheetLimits,
		events:      p.Events,
		rec:         factoringaction.Recorder{Audit: p.Audit, Realtime: p.Realtime, Logger: l},
		l:           l,
		now:         timeutils.NowUnix,
	}
}

func remittanceTenant(rem *factoringremittance.Remittance) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: rem.OrganizationID, BuID: rem.BusinessUnitID}
}

func (s *RemittanceService) Get(
	ctx context.Context,
	req factoringremittanceports.GetRemittanceRequest,
) (*factoringremittance.Remittance, error) {
	if req.ID.IsNil() {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrRequired,
			"Choose the remittance",
		)
	}
	req.ForUpdate = false
	return s.remittances.Get(ctx, req)
}

func (s *RemittanceService) ListConnection(
	ctx context.Context,
	req *factoringremittanceports.ListRemittanceConnectionRequest,
) (*pagination.CursorListResult[*factoringremittance.Remittance], error) {
	if err := factoringaction.RequireRequest(req != nil); err != nil {
		return nil, err
	}
	return s.remittances.ListConnection(ctx, req)
}

func (s *RemittanceService) ListLineConnection(
	ctx context.Context,
	req *factoringremittanceports.ListRemittanceLineConnectionRequest,
) (*pagination.CursorListResult[*factoringremittance.RemittanceLine], error) {
	if err := factoringaction.RequireRequest(req != nil); err != nil {
		return nil, err
	}
	if req.RemittanceID.IsNil() {
		return nil, errortypes.NewValidationError(
			"remittanceId",
			errortypes.ErrRequired,
			"Choose the remittance",
		)
	}
	return s.remittances.ListLineConnection(ctx, req)
}

func (s *RemittanceService) Create(
	ctx context.Context,
	req *factoringremittanceports.CreateRemittanceRequest,
	actor *permissionports.RequestActor,
) (*factoringremittance.Remittance, error) {
	if err := factoringaction.RequireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := factoringaction.RequireActor(actor, "Importing a remittance"); err != nil {
		return nil, err
	}
	return s.create(ctx, &factoringremittance.Remittance{
		OrganizationID:     req.TenantInfo.OrgID,
		BusinessUnitID:     req.TenantInfo.BuID,
		FactoringCompanyID: req.FactoringCompanyID,
		Source:             factoringremittance.RemittanceSourceUpload,
		DefaultKind:        req.DefaultKind,
		RemittedOn:         req.RemittedOn,
		CreatedByID:        actor.ExecutorUserID(),
	}, actor)
}

func (s *RemittanceService) create(
	ctx context.Context,
	rem *factoringremittance.Remittance,
	actor *permissionports.RequestActor,
) (*factoringremittance.Remittance, error) {
	rem.Status = factoringremittance.RemittanceStatusPending
	if rem.DefaultKind == "" {
		rem.DefaultKind = factoringremittance.RemittanceLineFunding
	}
	if rem.Source == "" {
		rem.Source = factoringremittance.RemittanceSourceUpload
	}
	if err := s.requireRemittedOn(rem.RemittedOn); err != nil {
		return nil, err
	}
	multiErr := errortypes.NewMultiError()
	rem.Validate(multiErr)
	if err := factoringaction.ValidationOf(multiErr); err != nil {
		return nil, err
	}
	factor, err := s.factors.GetByID(ctx, factoringcompanyports.GetFactoringCompanyByIDRequest{
		ID:         rem.FactoringCompanyID,
		TenantInfo: remittanceTenant(rem),
	})
	if err != nil {
		return nil, err
	}
	if !factor.IsActive() {
		return nil, errortypes.NewBusinessError(
			"{0} is inactive; reactivate the factor before importing its remittances",
			factor.Name,
		)
	}
	created, err := s.remittances.Create(ctx, rem)
	if err != nil {
		return nil, err
	}
	s.record(ctx, actor, nil, created, permission.OpCreate,
		"Remittance opened for "+factor.Name)
	return created, nil
}

func (s *RemittanceService) requireRemittedOn(at *int64) error {
	if at == nil {
		return nil
	}
	switch {
	case *at <= 0:
		return errortypes.NewValidationError(
			"remittedOn",
			errortypes.ErrInvalid,
			"Give the date the factor sent the report",
		)
	case *at > s.now()+factoringaction.MaxFutureMovementSeconds:
		return errortypes.NewValidationError(
			"remittedOn",
			errortypes.ErrInvalid,
			"The date cannot be in the future",
		)
	default:
		return nil
	}
}

func (s *RemittanceService) Rematch(
	ctx context.Context,
	req *factoringremittanceports.RemittanceVersionRequest,
	actor *permissionports.RequestActor,
) (*factoringremittance.Remittance, error) {
	if err := factoringaction.RequireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := factoringaction.RequireActor(actor, "Matching a remittance"); err != nil {
		return nil, err
	}
	var before, updated *factoringremittance.Remittance
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		rem, err := s.lock(txCtx, req.TenantInfo, req.ID, req.Version)
		if err != nil {
			return err
		}
		if err = rem.RequireEditable(); err != nil {
			return err
		}
		before = cloneRemittance(rem)
		candidates, err := s.candidatesFor(txCtx, rem, rem.Lines)
		if err != nil {
			return err
		}
		factoringremittance.MatchRemittance(rem.Lines, candidates)
		rem.Recount()
		if err = s.remittances.UpdateLines(txCtx, rem.Lines); err != nil {
			return err
		}
		updated, err = s.save(txCtx, rem)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.record(ctx, actor, before, updated, permission.OpUpdate,
		"Remittance "+updated.FileName+" matched again: "+countsPhrase(updated))
	return updated, nil
}

func (s *RemittanceService) UpdateLine(
	ctx context.Context,
	req *factoringremittanceports.UpdateRemittanceLineRequest,
	actor *permissionports.RequestActor,
) (*factoringremittance.Remittance, error) {
	if err := factoringaction.RequireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := factoringaction.RequireActor(actor, "Resolving a remittance line"); err != nil {
		return nil, err
	}
	if err := validateLineRequest(req); err != nil {
		return nil, err
	}
	var before, updated *factoringremittance.Remittance
	var comment string
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		rem, err := s.lock(txCtx, req.TenantInfo, req.RemittanceID, req.Version)
		if err != nil {
			return err
		}
		line, err := rem.EditableLine(req.LineID)
		if err != nil {
			return err
		}
		before = cloneRemittance(rem)
		if comment, err = s.resolveLine(txCtx, rem, line, req); err != nil {
			return err
		}
		rem.SettleIfDone()
		if err = s.remittances.UpdateLines(
			txCtx,
			[]*factoringremittance.RemittanceLine{line},
		); err != nil {
			return err
		}
		updated, err = s.save(txCtx, rem)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.record(ctx, actor, before, updated, permission.OpUpdate, comment)
	return updated, nil
}

func (s *RemittanceService) PlanUpdateLine(
	ctx context.Context,
	req *factoringremittanceports.UpdateRemittanceLineRequest,
	actor *permissionports.RequestActor,
) (*factoringremittanceports.RemittanceLinePlan, error) {
	if err := factoringaction.RequireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := factoringaction.RequireActor(actor, "Resolving a remittance line"); err != nil {
		return nil, err
	}
	if err := validateLineRequest(req); err != nil {
		return nil, err
	}
	rem, err := s.remittances.Get(ctx, factoringremittanceports.GetRemittanceRequest{
		TenantInfo:   req.TenantInfo,
		ID:           req.RemittanceID,
		IncludeLines: true,
	})
	if err != nil {
		return nil, err
	}
	if err = requireVersion(rem, req.Version); err != nil {
		return nil, err
	}
	line, err := rem.EditableLine(req.LineID)
	if err != nil {
		return nil, err
	}
	plan := &factoringremittanceports.RemittanceLinePlan{
		Before:     cloneRemittance(rem),
		LineBefore: new(*line),
	}
	if plan.Comment, err = s.resolveLine(ctx, rem, line, req); err != nil {
		return nil, err
	}
	rem.SettleIfDone()
	plan.After = cloneRemittance(rem)
	plan.LineAfter = line
	return plan, nil
}

func validateLineRequest(req *factoringremittanceports.UpdateRemittanceLineRequest) error {
	multiErr := errortypes.NewMultiError()
	if req.LineID.IsNil() {
		multiErr.Add("lineId", errortypes.ErrRequired, "Choose the line")
	}
	if !req.Action.IsValid() {
		multiErr.Add("action", errortypes.ErrInvalid, "Choose what to do with the line")
	}
	switch req.Action {
	case factoringremittanceports.RemittanceLineActionMatch:
		if req.ItemID.IsNil() {
			multiErr.Add("itemId", errortypes.ErrRequired, "Choose the invoice the line pays")
		}
	case factoringremittanceports.RemittanceLineActionIgnore:
		if strings.TrimSpace(req.Reason) == "" {
			multiErr.Add("reason", errortypes.ErrRequired, "Say why this line is not booked")
		}
	case factoringremittanceports.RemittanceLineActionSetKind:
		if !req.Kind.IsValid() {
			multiErr.Add("kind", errortypes.ErrInvalid, "Choose what the line records")
		}
	case factoringremittanceports.RemittanceLineActionUnmatch,
		factoringremittanceports.RemittanceLineActionRestore:
	}
	return factoringaction.ValidationOf(multiErr)
}

func (s *RemittanceService) resolveLine(
	ctx context.Context,
	rem *factoringremittance.Remittance,
	line *factoringremittance.RemittanceLine,
	req *factoringremittanceports.UpdateRemittanceLineRequest,
) (string, error) {
	row := "Row " + strconv.Itoa(line.RowNumber) + " of remittance " + rem.FileName
	switch req.Action {
	case factoringremittanceports.RemittanceLineActionMatch:
		item, err := s.matchByPerson(ctx, rem, line, req.ItemID)
		if err != nil {
			return "", err
		}
		return row + " matched to invoice " + item.InvoiceNumber, nil
	case factoringremittanceports.RemittanceLineActionUnmatch:
		if line.MatchStatus != factoringremittance.RemittanceLineMatched {
			return "", errortypes.NewBusinessError(
				"Row {0} is not matched",
				line.RowNumber,
			)
		}
		line.Suggest(line.Suggestions, "")
		return row + " unmatched", nil
	case factoringremittanceports.RemittanceLineActionIgnore:
		if err := line.Ignore(req.Reason); err != nil {
			return "", err
		}
		return row + " ignored: " + line.IgnoredReason, nil
	case factoringremittanceports.RemittanceLineActionRestore:
		if line.MatchStatus != factoringremittance.RemittanceLineIgnored {
			return "", errortypes.NewBusinessError(
				"Row {0} is not ignored",
				line.RowNumber,
			)
		}
		line.Reopen()
		if err := s.suggestFor(ctx, rem, line); err != nil {
			return "", err
		}
		return row + " restored", nil
	case factoringremittanceports.RemittanceLineActionSetKind:
		if err := line.SetKind(req.Kind); err != nil {
			return "", err
		}
		if err := s.suggestFor(ctx, rem, line); err != nil {
			return "", err
		}
		return row + " now records " + kindPhrase(line.Kind), nil
	}
	return "", errortypes.NewValidationError(
		"action",
		errortypes.ErrInvalid,
		"Choose what to do with the line",
	)
}

func (s *RemittanceService) matchByPerson(
	ctx context.Context,
	rem *factoringremittance.Remittance,
	line *factoringremittance.RemittanceLine,
	itemID pulid.ID,
) (*factoringsubmission.SubmissionItem, error) {
	if line.MatchStatus == factoringremittance.RemittanceLineError {
		return nil, errortypes.NewBusinessError(
			"Row {0} could not be read; ignore it and book it by hand",
			line.RowNumber,
		)
	}
	if !line.Kind.Books() {
		return nil, errortypes.NewBusinessError(
			"Row {0} is not a funding, rejection, reserve release or chargeback; set what it records first",
			line.RowNumber,
		)
	}
	candidates, err := s.remittances.ListCandidates(
		ctx,
		&factoringremittanceports.ListRemittanceCandidatesRequest{
			TenantInfo:         remittanceTenant(rem),
			FactoringCompanyID: rem.FactoringCompanyID,
			ItemIDs:            []pulid.ID{itemID},
			Limit:              1,
		},
	)
	if err != nil {
		return nil, err
	}
	var candidate *factoringremittance.RemittanceCandidate
	for _, c := range candidates {
		if c.Item.ID == itemID {
			candidate = c
		}
	}
	if candidate == nil {
		return nil, errortypes.NewValidationError(
			"itemId",
			errortypes.ErrInvalid,
			"That invoice is not on an open schedule sent to this factor",
		)
	}
	if !line.Kind.Accepts(candidate) {
		return nil, errortypes.NewValidationError(
			"itemId",
			errortypes.ErrInvalid,
			"Invoice {0} is {1}; it cannot take {2}",
			candidate.Item.InvoiceNumber,
			strings.ToLower(string(candidate.Item.Status)),
			kindPhrase(line.Kind),
		)
	}
	for _, other := range rem.Lines {
		if other == nil || other.ID == line.ID || other.Kind != line.Kind ||
			other.SubmissionItemID != itemID {
			continue
		}
		if other.MatchStatus == factoringremittance.RemittanceLineMatched ||
			other.MatchStatus == factoringremittance.RemittanceLineApplied {
			return nil, errortypes.NewValidationError(
				"itemId",
				errortypes.ErrInvalid,
				"Row {0} already matched invoice {1}",
				other.RowNumber,
				candidate.Item.InvoiceNumber,
			)
		}
	}
	score, _ := factoringremittance.ScoreRemittanceLine(line, candidate)
	line.MatchTo(
		candidate.Item,
		factoringremittance.RemittanceMatchedByPerson,
		score,
		factoringremittance.MatchReasonChosenByPerson,
	)
	return candidate.Item, nil
}

func (s *RemittanceService) suggestFor(
	ctx context.Context,
	rem *factoringremittance.Remittance,
	line *factoringremittance.RemittanceLine,
) error {
	if !line.Matchable() {
		return nil
	}
	if !line.Kind.Books() {
		line.Suggest(nil, factoringremittance.MatchReasonNotBookedFromRemittance)
		return nil
	}
	candidates, err := s.candidatesFor(ctx, rem, []*factoringremittance.RemittanceLine{line})
	if err != nil {
		return err
	}
	line.Suggest(factoringremittance.RankRemittanceLine(line, candidates), "")
	return nil
}

func (s *RemittanceService) PlanDiscard(
	ctx context.Context,
	req *factoringremittanceports.DiscardRemittanceRequest,
	actor *permissionports.RequestActor,
) (*factoringremittanceports.RemittanceDiscardPlan, error) {
	if err := factoringaction.RequireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := factoringaction.RequireActor(actor, "Discarding a remittance"); err != nil {
		return nil, err
	}
	rem, err := s.remittances.Get(ctx, factoringremittanceports.GetRemittanceRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	})
	if err != nil {
		return nil, err
	}
	if err = requireVersion(rem, req.Version); err != nil {
		return nil, err
	}
	plan := &factoringremittanceports.RemittanceDiscardPlan{Before: cloneRemittance(rem)}
	if err = rem.Discard(s.now(), req.Reason); err != nil {
		return nil, err
	}
	plan.After = rem
	return plan, nil
}

func (s *RemittanceService) Discard(
	ctx context.Context,
	req *factoringremittanceports.DiscardRemittanceRequest,
	actor *permissionports.RequestActor,
) (*factoringremittance.Remittance, error) {
	if err := factoringaction.RequireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := factoringaction.RequireActor(actor, "Discarding a remittance"); err != nil {
		return nil, err
	}
	var before, updated *factoringremittance.Remittance
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		rem, err := s.lock(txCtx, req.TenantInfo, req.ID, req.Version)
		if err != nil {
			return err
		}
		before = cloneRemittance(rem)
		if err = rem.Discard(s.now(), req.Reason); err != nil {
			return err
		}
		updated, err = s.save(txCtx, rem)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.record(ctx, actor, before, updated, permission.OpCancel,
		"Remittance "+updated.FileName+" discarded: "+updated.DiscardReason)
	return updated, nil
}

func (s *RemittanceService) lock(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	version int64,
) (*factoringremittance.Remittance, error) {
	if id.IsNil() {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrRequired,
			"Choose the remittance",
		)
	}
	rem, err := s.remittances.Get(ctx, factoringremittanceports.GetRemittanceRequest{
		TenantInfo:   tenantInfo,
		ID:           id,
		IncludeLines: true,
		ForUpdate:    true,
	})
	if err != nil {
		return nil, err
	}
	if err = requireVersion(rem, version); err != nil {
		return nil, err
	}
	return rem, nil
}

func (s *RemittanceService) save(
	ctx context.Context,
	rem *factoringremittance.Remittance,
) (*factoringremittance.Remittance, error) {
	lines := rem.Lines
	updated, err := s.remittances.Update(ctx, rem)
	if err != nil {
		return nil, err
	}
	updated.Lines = lines
	return updated, nil
}

func (s *RemittanceService) candidatesFor(
	ctx context.Context,
	rem *factoringremittance.Remittance,
	lines []*factoringremittance.RemittanceLine,
) ([]*factoringremittance.RemittanceCandidate, error) {
	invoices := make([]string, 0, len(lines)*2)
	amounts := make([]int64, 0, len(lines)*3)
	seenInvoice := make(map[string]struct{}, len(lines)*2)
	seenAmount := make(map[int64]struct{}, len(lines)*3)
	addInvoice := func(raw string) {
		value := factoringremittance.NormalizeRemittanceReference(raw)
		if value == "" {
			return
		}
		if _, dup := seenInvoice[value]; !dup {
			seenInvoice[value] = struct{}{}
			invoices = append(invoices, value)
		}
	}
	addAmount := func(value *int64) {
		if value == nil || *value == 0 {
			return
		}
		amount := *value
		if amount < 0 {
			amount = -amount
		}
		if _, dup := seenAmount[amount]; !dup {
			seenAmount[amount] = struct{}{}
			amounts = append(amounts, amount)
		}
	}
	for _, line := range lines {
		if line == nil || !line.Matchable() || !line.Kind.Books() {
			continue
		}
		addInvoice(line.InvoiceNumber)
		addInvoice(line.FactorReference)
		addAmount(line.AdvanceMinor)
		addAmount(line.FaceMinor)
		addAmount(line.CashMinor)
		addAmount(line.ReserveMinor)
	}
	if len(invoices) == 0 && len(amounts) == 0 {
		return []*factoringremittance.RemittanceCandidate{}, nil
	}
	return s.remittances.ListCandidates(
		ctx,
		&factoringremittanceports.ListRemittanceCandidatesRequest{
			TenantInfo:         remittanceTenant(rem),
			FactoringCompanyID: rem.FactoringCompanyID,
			Invoices:           invoices,
			Amounts:            amounts,
		},
	)
}

func (s *RemittanceService) record(
	ctx context.Context,
	actor *permissionports.RequestActor,
	before, updated *factoringremittance.Remittance,
	op permission.Operation,
	comment string,
) {
	kernelports.AfterCommit(ctx, func(committedCtx context.Context) {
		entry := &factoringaction.AuditEntry{
			Resource: remittanceResource,
			Entity:   withoutLines(updated),
			Op:       op,
			Comment:  comment,
		}
		if before != nil {
			entry.Previous = withoutLines(before)
		}
		s.rec.Log(actor, entry)
		s.rec.Publish(committedCtx, remittanceResource, withoutLines(updated), actor,
			remittanceRealtimeEvent)
	})
}

func (s *RemittanceService) announceStaged(
	ctx context.Context,
	rem *factoringremittance.Remittance,
) {
	if rem.Status != factoringremittance.RemittanceStatusParsed {
		return
	}
	agenteventports.PublishAgentEvent(ctx, s.events, agenteventports.AgentEvent{
		Kind:       agenttrigger.EventFactoringRemittanceStaged,
		SubjectID:  rem.ID,
		TenantInfo: remittanceTenant(rem),
		Related:    []pulid.ID{rem.FactoringCompanyID},
		Detail:     "Remittance " + remittanceName(rem) + " read: " + countsPhrase(rem),
	})
}

func withoutLines(rem *factoringremittance.Remittance) *factoringremittance.Remittance {
	if rem == nil {
		return nil
	}
	out := *rem
	out.Lines = nil
	out.FactoringCompany = nil
	return &out
}

func cloneRemittance(rem *factoringremittance.Remittance) *factoringremittance.Remittance {
	out := *rem
	out.Lines = nil
	return &out
}

func remittanceName(rem *factoringremittance.Remittance) string {
	if name := strings.TrimSpace(rem.FileName); name != "" {
		return name
	}
	return rem.ID.String()
}

func countsPhrase(rem *factoringremittance.Remittance) string {
	return strconv.Itoa(rem.LineCount) + " lines, " +
		strconv.Itoa(rem.MatchedCount) + " matched, " +
		strconv.Itoa(rem.ExceptionCount) + " to resolve"
}

func kindPhrase(kind factoringremittance.RemittanceLineKind) string {
	switch kind {
	case factoringremittance.RemittanceLineFunding:
		return "a funding"
	case factoringremittance.RemittanceLineRejection:
		return "a rejection"
	case factoringremittance.RemittanceLineReserveRelease:
		return "a reserve release"
	case factoringremittance.RemittanceLineChargeback:
		return "a chargeback"
	case factoringremittance.RemittanceLineOther:
		return "an adjustment"
	}
	return string(kind)
}
