package factoringservice

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	remittanceResource      = permission.ResourceFactoringRemittance
	remittanceRealtimeEvent = "updated"
)

type RemittanceParams struct {
	fx.In

	DB          kernelports.DBConnection
	Remittances factoringports.RemittanceRepository
	Submissions factoringports.SubmissionRepository
	Ledger      factoringports.LedgerService
	Factors     factoringcompanyports.FactoringCompanyRepository
	Documents   documentports.InvoiceDocumentService
	Content     documentports.DocumentContentService  `optional:"true"`
	Completion  agentports.CompletionService          `optional:"true"`
	Starter     factoringports.RemittanceStageStarter `optional:"true"`
	Events      agentports.AgentEventPublisher        `optional:"true"`
	Audit       auditports.AuditService
	Realtime    agentports.RealtimeService `optional:"true"`
	Logger      *zap.Logger
}

type RemittanceService struct {
	db          kernelports.DBConnection
	remittances factoringports.RemittanceRepository
	submissions factoringports.SubmissionRepository
	ledger      factoringports.LedgerService
	factors     factoringcompanyports.FactoringCompanyRepository
	documents   documentports.InvoiceDocumentService
	content     documentports.DocumentContentService
	completion  agentports.CompletionService
	starter     factoringports.RemittanceStageStarter
	events      agentports.AgentEventPublisher
	rec         recorder
	l           *zap.Logger
	now         func() int64
}

var _ factoringports.RemittanceService = (*RemittanceService)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func NewRemittanceService(p RemittanceParams) factoringports.RemittanceService {
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
		events:      p.Events,
		rec:         recorder{audit: p.Audit, realtime: p.Realtime, l: l},
		l:           l,
		now:         timeutils.NowUnix,
	}
}

func remittanceTenant(rem *factoring.Remittance) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: rem.OrganizationID, BuID: rem.BusinessUnitID}
}

func (s *RemittanceService) Get(
	ctx context.Context,
	req factoringports.GetRemittanceRequest,
) (*factoring.Remittance, error) {
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
	req *factoringports.ListRemittanceConnectionRequest,
) (*pagination.CursorListResult[*factoring.Remittance], error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	return s.remittances.ListConnection(ctx, req)
}

func (s *RemittanceService) ListLineConnection(
	ctx context.Context,
	req *factoringports.ListRemittanceLineConnectionRequest,
) (*pagination.CursorListResult[*factoring.RemittanceLine], error) {
	if err := requireRequest(req != nil); err != nil {
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
	req *factoringports.CreateRemittanceRequest,
	actor *permissionports.RequestActor,
) (*factoring.Remittance, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireActor(actor, "Importing a remittance"); err != nil {
		return nil, err
	}
	return s.create(ctx, &factoring.Remittance{
		OrganizationID:     req.TenantInfo.OrgID,
		BusinessUnitID:     req.TenantInfo.BuID,
		FactoringCompanyID: req.FactoringCompanyID,
		Source:             factoring.RemittanceSourceUpload,
		DefaultKind:        req.DefaultKind,
		RemittedOn:         req.RemittedOn,
		CreatedByID:        actor.ExecutorUserID(),
	}, actor)
}

func (s *RemittanceService) create(
	ctx context.Context,
	rem *factoring.Remittance,
	actor *permissionports.RequestActor,
) (*factoring.Remittance, error) {
	rem.Status = factoring.RemittanceStatusPending
	if rem.DefaultKind == "" {
		rem.DefaultKind = factoring.RemittanceLineFunding
	}
	if rem.Source == "" {
		rem.Source = factoring.RemittanceSourceUpload
	}
	if err := s.requireRemittedOn(rem.RemittedOn); err != nil {
		return nil, err
	}
	multiErr := errortypes.NewMultiError()
	rem.Validate(multiErr)
	if err := validationOf(multiErr); err != nil {
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
	case *at > s.now()+maxFutureMovementSeconds:
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
	req *factoringports.RemittanceVersionRequest,
	actor *permissionports.RequestActor,
) (*factoring.Remittance, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireActor(actor, "Matching a remittance"); err != nil {
		return nil, err
	}
	var before, updated *factoring.Remittance
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
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
		factoring.MatchRemittance(rem.Lines, candidates)
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
	req *factoringports.UpdateRemittanceLineRequest,
	actor *permissionports.RequestActor,
) (*factoring.Remittance, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireActor(actor, "Resolving a remittance line"); err != nil {
		return nil, err
	}
	if err := validateLineRequest(req); err != nil {
		return nil, err
	}
	var before, updated *factoring.Remittance
	var comment string
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
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
		if err = s.remittances.UpdateLines(txCtx, []*factoring.RemittanceLine{line}); err != nil {
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

func validateLineRequest(req *factoringports.UpdateRemittanceLineRequest) error {
	multiErr := errortypes.NewMultiError()
	if req.LineID.IsNil() {
		multiErr.Add("lineId", errortypes.ErrRequired, "Choose the line")
	}
	if !req.Action.IsValid() {
		multiErr.Add("action", errortypes.ErrInvalid, "Choose what to do with the line")
	}
	switch req.Action {
	case factoringports.RemittanceLineActionMatch:
		if req.ItemID.IsNil() {
			multiErr.Add("itemId", errortypes.ErrRequired, "Choose the invoice the line pays")
		}
	case factoringports.RemittanceLineActionIgnore:
		if strings.TrimSpace(req.Reason) == "" {
			multiErr.Add("reason", errortypes.ErrRequired, "Say why this line is not booked")
		}
	case factoringports.RemittanceLineActionSetKind:
		if !req.Kind.IsValid() {
			multiErr.Add("kind", errortypes.ErrInvalid, "Choose what the line records")
		}
	case factoringports.RemittanceLineActionUnmatch, factoringports.RemittanceLineActionRestore:
	}
	return validationOf(multiErr)
}

func (s *RemittanceService) resolveLine(
	ctx context.Context,
	rem *factoring.Remittance,
	line *factoring.RemittanceLine,
	req *factoringports.UpdateRemittanceLineRequest,
) (string, error) {
	row := "Row " + strconv.Itoa(line.RowNumber) + " of remittance " + rem.FileName
	switch req.Action {
	case factoringports.RemittanceLineActionMatch:
		item, err := s.matchByPerson(ctx, rem, line, req.ItemID)
		if err != nil {
			return "", err
		}
		return row + " matched to invoice " + item.InvoiceNumber, nil
	case factoringports.RemittanceLineActionUnmatch:
		if line.MatchStatus != factoring.RemittanceLineMatched {
			return "", errortypes.NewBusinessError(
				"Row {0} is not matched",
				line.RowNumber,
			)
		}
		line.Suggest(line.Suggestions, "")
		return row + " unmatched", nil
	case factoringports.RemittanceLineActionIgnore:
		if err := line.Ignore(req.Reason); err != nil {
			return "", err
		}
		return row + " ignored: " + line.IgnoredReason, nil
	case factoringports.RemittanceLineActionRestore:
		if line.MatchStatus != factoring.RemittanceLineIgnored {
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
	case factoringports.RemittanceLineActionSetKind:
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
	rem *factoring.Remittance,
	line *factoring.RemittanceLine,
	itemID pulid.ID,
) (*factoring.SubmissionItem, error) {
	if line.MatchStatus == factoring.RemittanceLineError {
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
		&factoringports.ListRemittanceCandidatesRequest{
			TenantInfo:         remittanceTenant(rem),
			FactoringCompanyID: rem.FactoringCompanyID,
			ItemIDs:            []pulid.ID{itemID},
			Limit:              1,
		},
	)
	if err != nil {
		return nil, err
	}
	var candidate *factoring.RemittanceCandidate
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
		if other.MatchStatus == factoring.RemittanceLineMatched ||
			other.MatchStatus == factoring.RemittanceLineApplied {
			return nil, errortypes.NewValidationError(
				"itemId",
				errortypes.ErrInvalid,
				"Row {0} already matched invoice {1}",
				other.RowNumber,
				candidate.Item.InvoiceNumber,
			)
		}
	}
	score, _ := factoring.ScoreRemittanceLine(line, candidate)
	line.MatchTo(
		candidate.Item,
		factoring.RemittanceMatchedByPerson,
		score,
		factoring.MatchReasonChosenByPerson,
	)
	return candidate.Item, nil
}

func (s *RemittanceService) suggestFor(
	ctx context.Context,
	rem *factoring.Remittance,
	line *factoring.RemittanceLine,
) error {
	if !line.Matchable() {
		return nil
	}
	if !line.Kind.Books() {
		line.Suggest(nil, factoring.MatchReasonNotBookedFromRemittance)
		return nil
	}
	candidates, err := s.candidatesFor(ctx, rem, []*factoring.RemittanceLine{line})
	if err != nil {
		return err
	}
	line.Suggest(factoring.RankRemittanceLine(line, candidates), "")
	return nil
}

func (s *RemittanceService) Discard(
	ctx context.Context,
	req *factoringports.DiscardRemittanceRequest,
	actor *permissionports.RequestActor,
) (*factoring.Remittance, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireActor(actor, "Discarding a remittance"); err != nil {
		return nil, err
	}
	var before, updated *factoring.Remittance
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
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
) (*factoring.Remittance, error) {
	if id.IsNil() {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrRequired,
			"Choose the remittance",
		)
	}
	rem, err := s.remittances.Get(ctx, factoringports.GetRemittanceRequest{
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
	rem *factoring.Remittance,
) (*factoring.Remittance, error) {
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
	rem *factoring.Remittance,
	lines []*factoring.RemittanceLine,
) ([]*factoring.RemittanceCandidate, error) {
	invoices := make([]string, 0, len(lines)*2)
	amounts := make([]int64, 0, len(lines)*3)
	seenInvoice := make(map[string]struct{}, len(lines)*2)
	seenAmount := make(map[int64]struct{}, len(lines)*3)
	addInvoice := func(raw string) {
		value := factoring.NormalizeRemittanceReference(raw)
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
		return []*factoring.RemittanceCandidate{}, nil
	}
	return s.remittances.ListCandidates(ctx, &factoringports.ListRemittanceCandidatesRequest{
		TenantInfo:         remittanceTenant(rem),
		FactoringCompanyID: rem.FactoringCompanyID,
		Invoices:           invoices,
		Amounts:            amounts,
	})
}

func (s *RemittanceService) record(
	ctx context.Context,
	actor *permissionports.RequestActor,
	before, updated *factoring.Remittance,
	op permission.Operation,
	comment string,
) {
	kernelports.AfterCommit(ctx, func(committedCtx context.Context) {
		entry := &auditEntry{
			resource: remittanceResource,
			entity:   withoutLines(updated),
			op:       op,
			comment:  comment,
		}
		if before != nil {
			entry.previous = withoutLines(before)
		}
		s.rec.log(actor, entry)
		s.rec.publish(committedCtx, remittanceResource, withoutLines(updated), actor,
			remittanceRealtimeEvent)
	})
}

func (s *RemittanceService) announceStaged(ctx context.Context, rem *factoring.Remittance) {
	if rem.Status != factoring.RemittanceStatusParsed {
		return
	}
	agentports.PublishAgentEvent(ctx, s.events, agentports.AgentEvent{
		Kind:       agent.EventFactoringRemittanceStaged,
		SubjectID:  rem.ID,
		TenantInfo: remittanceTenant(rem),
		Related:    []pulid.ID{rem.FactoringCompanyID},
		Detail:     "Remittance " + remittanceName(rem) + " read: " + countsPhrase(rem),
	})
}

func withoutLines(rem *factoring.Remittance) *factoring.Remittance {
	if rem == nil {
		return nil
	}
	out := *rem
	out.Lines = nil
	out.FactoringCompany = nil
	return &out
}

func cloneRemittance(rem *factoring.Remittance) *factoring.Remittance {
	out := *rem
	out.Lines = nil
	return &out
}

func remittanceName(rem *factoring.Remittance) string {
	if name := strings.TrimSpace(rem.FileName); name != "" {
		return name
	}
	return rem.ID.String()
}

func countsPhrase(rem *factoring.Remittance) string {
	return strconv.Itoa(rem.LineCount) + " lines, " +
		strconv.Itoa(rem.MatchedCount) + " matched, " +
		strconv.Itoa(rem.ExceptionCount) + " to resolve"
}

func kindPhrase(kind factoring.RemittanceLineKind) string {
	switch kind {
	case factoring.RemittanceLineFunding:
		return "a funding"
	case factoring.RemittanceLineRejection:
		return "a rejection"
	case factoring.RemittanceLineReserveRelease:
		return "a reserve release"
	case factoring.RemittanceLineChargeback:
		return "a chargeback"
	case factoring.RemittanceLineOther:
		return "an adjustment"
	}
	return string(kind)
}
