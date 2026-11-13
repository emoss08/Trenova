package agentmemorydesk

import (
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// The Desk's memory page and the memory cards in a conversation work on the
// memories a person keeps: their own, their roles', and the organization's.
//
// Who may do what is decided here, from the person and the one agent-memory
// permission their resolver read for them:
//
//   - A person's own ("Just you") memories are theirs to write, change, pause
//     and forget with no permission beyond using the Desk.
//   - A role's memories reach everyone holding the role, and the
//     organization's reach everyone, so keeping one at once, or changing,
//     pausing or forgetting one, is agent-memory manage, and for a role also
//     holding that role. Without it a person still writes one, but it is
//     offered: kept as a suggestion an administrator approves in AI Control,
//     listed to the person as offered until then, and theirs to withdraw.
//
// Agent-scoped memories are administered in AI Control and never listed here.

const (
	deskPageDefault    = 50
	deskCursorResource = "desk_memories:"
)

// Core is what the Desk's memory page needs of the memory service: the one
// way a memory is kept, with its dedupe, consolidation, taint and audit, and
// the person's readers. The page decides who may do what; keeping, changing
// and recording stay where every other writer goes through them.
type Core interface {
	Keep(
		ctx context.Context,
		req *agentmemoryports.RememberRequest,
		actor *permissionports.RequestActor,
		manages bool,
	) (*agentmemory.Memory, error)
	SetStatusAs(
		ctx context.Context,
		req *agentmemoryports.SetAgentMemoryStatusRequest,
		actor *permissionports.RequestActor,
		manages bool,
	) (*agentmemory.Memory, error)
	QueueForRetrieval(ctx context.Context, memory *agentmemory.Memory)
	LogChange(
		ctx context.Context,
		entity *agentmemory.Memory,
		previous map[string]any,
		actor *permissionports.RequestActor,
		op permission.Operation,
		comment string,
	)
	RecordReview(
		ctx context.Context,
		before, after *agentmemory.Memory,
		actor *permissionports.RequestActor,
	)
	RecordReplaced(
		ctx context.Context,
		replacement *agentmemory.Memory,
		actor *permissionports.RequestActor,
	)
	HeldRoles(
		ctx context.Context,
		tenant pagination.TenantInfo,
		userID pulid.ID,
	) ([]*rbac.Role, error)
	Reader(
		ctx context.Context,
		tenant pagination.TenantInfo,
		userID pulid.ID,
	) (agentmemory.MemoryReader, error)
	SavingMode(
		ctx context.Context,
		tenant pagination.TenantInfo,
		userID pulid.ID,
	) (agentmemory.MemorySavingMode, error)
}

// Desk is the memory page of the Desk and the memory cards of a
// conversation: the memories a person keeps, and what they may do with them.
type Desk struct {
	core Core
	repo agentmemoryports.AgentMemoryRepository
	l    *zap.Logger
}

func New(core Core, repo agentmemoryports.AgentMemoryRepository, logger *zap.Logger) *Desk {
	return &Desk{core: core, repo: repo, l: logger}
}

// deskPerson is the person on the Desk with who they read memories as.
type deskPerson struct {
	*agentmemoryports.DeskMemoryActor

	tenant pagination.TenantInfo
	userID pulid.ID
	reader agentmemory.MemoryReader
}

func (s *Desk) person(
	ctx context.Context,
	actor *agentmemoryports.DeskMemoryActor,
) (*deskPerson, error) {
	if actor == nil || !actor.Actor.IsUser() || actor.Actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError("Only a person keeps memories on the Desk")
	}

	tenant := actor.Actor.TenantInfo()
	reader, err := s.core.Reader(ctx, tenant, actor.Actor.UserID)
	if err != nil {
		return nil, err
	}

	return &deskPerson{
		DeskMemoryActor: actor,
		tenant:          tenant,
		userID:          actor.Actor.UserID,
		reader:          reader,
	}, nil
}

// sees reports whether the memory is one the person keeps: theirs, their
// roles' or the organization's. A suggestion is seen only by the person whose
// conversation it was offered in, or who offered it themselves.
func (p *deskPerson) sees(memory *agentmemory.Memory) bool {
	if memory.Scope == agentmemory.MemoryScopeAgent {
		return memory.Source == agentmemory.MemorySourceReflection && p.offeredTo(memory)
	}
	if memory.Status.IsSuggestion() {
		return p.offeredTo(memory)
	}

	return p.reader.Reads(memory)
}

// keptForAgents reports whether a memory is one an agent keeps for its own
// work. It is not the person's, so their list leaves it out, but a
// conversation that used one names it, and the person may read what the
// answer they got was built on.
func keptForAgents(memory *agentmemory.Memory) bool {
	return memory.Scope == agentmemory.MemoryScopeAgent && !memory.Status.IsSuggestion()
}

func (p *deskPerson) offeredTo(memory *agentmemory.Memory) bool {
	return memory.CreatedByUserID != nil && *memory.CreatedByUserID == p.userID
}

func (p *deskPerson) holds(roleID pulid.ID) bool {
	return roleID.IsNotNil() && slices.Contains(p.reader.RoleIDs, roleID)
}

// mayChange reports whether the person may change, pause or forget a memory
// they see.
func (p *deskPerson) mayChange(memory *agentmemory.Memory) bool {
	switch memory.Scope {
	case agentmemory.MemoryScopeUser:
		return memory.OwnerUserID != nil && *memory.OwnerUserID == p.userID
	case agentmemory.MemoryScopeRole:
		return memory.RoleID != nil && p.holds(*memory.RoleID) && p.MayManageShared
	case agentmemory.MemoryScopeOrganization, agentmemory.MemoryScopeAgent:
		return p.MayManageShared
	default:
		return false
	}
}

// errOfferInstead is the refusal for moving a kept memory in front of readers
// the person may not keep one for. A new memory for them is offered; a kept
// one is not quietly turned into an offer.
func errOfferInstead() error {
	return errortypes.NewAuthorizationError(
		"Only an administrator keeps a memory for your team or the organization. Keep " +
			"this one for yourself, or write a new one for them and it will be offered to " +
			"an administrator.",
	)
}

// mayKeepFor reports whether the person may put a memory in front of the
// scope's readers, and whether it is kept at once or offered: a role's or the
// organization's is offered unless the person manages shared memory.
func (p *deskPerson) mayKeepFor(scope agentmemory.MemoryScope, roleID pulid.ID) (bool, error) {
	switch scope {
	case agentmemory.MemoryScopeUser:
		return false, nil
	case agentmemory.MemoryScopeRole:
		if !p.holds(roleID) {
			return false, errortypes.NewValidationError(
				"roleId", errortypes.ErrInvalid, "You can only keep a memory for a role you hold",
			)
		}

		return !p.MayManageShared, nil
	case agentmemory.MemoryScopeOrganization:
		return !p.MayManageShared, nil
	case agentmemory.MemoryScopeAgent:
		if !p.MayManageShared {
			return false, agentmemoryports.ErrSharedMemoryManaged()
		}

		return false, nil
	default:
		return false, errortypes.NewValidationError(
			"scope",
			errortypes.ErrInvalid,
			"Keep a memory for yourself, a role you hold or the organization",
		)
	}
}

func (p *deskPerson) desk(row *agentmemoryports.DeskMemoryRow) *agentmemoryports.DeskMemory {
	return &agentmemoryports.DeskMemory{
		Memory:      row.Memory,
		SourceTitle: row.SourceTitle,
		RoleName:    row.RoleName,
		Editable:    p.mayChange(row.Memory),
	}
}

func checkDeskContent(content string, kind agentmemory.MemoryKind) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", errortypes.NewValidationError(
			"content",
			errortypes.ErrRequired,
			"Say what to remember",
		)
	}
	if limit := kind.MaxContentChars(); utf8.RuneCountInString(content) > limit {
		return "", errortypes.NewValidationError(
			"content", errortypes.ErrInvalid,
			"A memory is at most {0} characters", limit,
		)
	}

	return content, nil
}

func (s *Desk) ListDesk(
	ctx context.Context,
	req *agentmemoryports.ListDeskMemoriesRequest,
) (*agentmemoryports.DeskMemoryPage, error) {
	person, err := s.person(ctx, req.Actor)
	if err != nil {
		return nil, err
	}
	if req.Scope != "" && !slices.Contains(deskListedScopes(), req.Scope) {
		return nil, errortypes.NewValidationError(
			"scope",
			errortypes.ErrInvalid,
			"Filter by a scope the Desk lists",
		)
	}
	after, err := decodeDeskCursor(person.cursorBinding(), req.After)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = deskPageDefault
	}

	filter := agentmemoryports.DeskMemoryFilter{
		TenantInfo: person.tenant,
		Reader:     person.reader,
		Scope:      req.Scope,
		RoleID:     req.RoleID,
		Query:      req.Query,
	}
	if req.SubjectID.IsNotNil() {
		if !req.SubjectType.IsValid() {
			return nil, errortypes.NewValidationError(
				"subjectType",
				errortypes.ErrInvalid,
				"Say what kind of record to list memories about",
			)
		}
		filter.Subject = &agentmemoryports.MemorySubjectRef{
			Type: req.SubjectType,
			ID:   req.SubjectID,
		}
	}
	rows, err := s.repo.ListDesk(ctx, &agentmemoryports.ListDeskMemoriesQuery{
		Filter: filter,
		After:  after,
		Limit:  limit + 1,
	})
	if err != nil {
		return nil, err
	}

	page := &agentmemoryports.DeskMemoryPage{
		Items: make([]*agentmemoryports.DeskMemory, 0, min(len(rows), limit)),
	}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1].Memory
		page.Next, err = pagination.EncodePositionCursor(
			person.cursorBinding(),
			pagination.PositionCursor{At: last.CreatedAt, ID: last.ID},
		)
		if err != nil {
			return nil, err
		}
	}
	for _, row := range rows {
		page.Items = append(page.Items, person.desk(row))
	}
	if err = s.deskLinks(ctx, person, page.Items); err != nil {
		return nil, err
	}

	// The counts are of what the search matches across every scope, so the
	// chips say how many each would show.
	filter.Scope = ""
	filter.RoleID = pulid.Nil
	counts, err := s.repo.CountDesk(ctx, &filter)
	if err != nil {
		return nil, err
	}
	for _, count := range counts {
		page.All += count.Count
		page.Counts = append(page.Counts, agentmemoryports.DeskMemoryCount{
			Scope:  count.Scope,
			RoleID: count.RoleID,
			Count:  count.Count,
		})
	}

	return page, nil
}

func (s *Desk) DeskByIDs(
	ctx context.Context,
	actor *agentmemoryports.DeskMemoryActor,
	ids []pulid.ID,
) ([]*agentmemoryports.DeskMemory, error) {
	person, err := s.person(ctx, actor)
	if err != nil {
		return nil, err
	}

	return s.deskByIDs(ctx, person, ids)
}

func (s *Desk) deskByIDs(
	ctx context.Context,
	person *deskPerson,
	ids []pulid.ID,
) ([]*agentmemoryports.DeskMemory, error) {
	rows, err := s.repo.GetDesk(ctx, agentmemoryports.GetDeskMemoriesRequest{
		TenantInfo: person.tenant,
		IDs:        ids,
	})
	if err != nil {
		return nil, err
	}

	byID := make(map[pulid.ID]*agentmemoryports.DeskMemoryRow, len(rows))
	for _, row := range rows {
		if person.sees(row.Memory) || keptForAgents(row.Memory) {
			byID[row.Memory.ID] = row
		}
	}

	// In the order asked for, which is the order the conversation used them.
	out := make([]*agentmemoryports.DeskMemory, 0, len(byID))
	for _, id := range ids {
		if row, ok := byID[id]; ok {
			out = append(out, person.desk(row))
			delete(byID, id)
		}
	}
	if err = s.deskLinks(ctx, person, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Desk) deskLinks(
	ctx context.Context,
	person *deskPerson,
	items []*agentmemoryports.DeskMemory,
) error {
	if len(items) == 0 {
		return nil
	}

	ids := make([]pulid.ID, 0, len(items))
	replacedIDs := make([]pulid.ID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.Memory.ID)
		if item.Memory.Replaces() {
			replacedIDs = append(replacedIDs, *item.Memory.SupersedesID)
		}
	}

	replaced, err := s.repo.ListByIDs(ctx, agentmemoryports.ListAgentMemoriesByIDsRequest{
		TenantInfo: person.tenant,
		IDs:        replacedIDs,
	})
	if err != nil {
		return err
	}
	replacedByID := make(map[pulid.ID]*agentmemory.Memory, len(replaced))
	for _, memory := range replaced {
		replacedByID[memory.ID] = memory
	}

	replacements, err := s.repo.ListReplacements(
		ctx,
		agentmemoryports.ListAgentMemoryReplacementsRequest{
			TenantInfo:  person.tenant,
			ReplacedIDs: ids,
		},
	)
	if err != nil {
		return err
	}
	newest := agentmemory.NewestReplacements(replacements)

	for _, item := range items {
		if item.Memory.Replaces() {
			item.Replaces = person.link(replacedByID[*item.Memory.SupersedesID])
		}
		item.ReplacedBy = person.link(newest[item.Memory.ID])
	}

	return nil
}

func (p *deskPerson) link(memory *agentmemory.Memory) *agentmemoryports.DeskMemoryLink {
	if memory == nil || !p.sees(memory) {
		return nil
	}

	return &agentmemoryports.DeskMemoryLink{
		ID:      memory.ID,
		Content: memory.Content,
		Status:  memory.Status,
	}
}

func (s *Desk) deskOne(
	ctx context.Context,
	person *deskPerson,
	id pulid.ID,
) (*agentmemoryports.DeskMemory, error) {
	found, err := s.deskByIDs(ctx, person, []pulid.ID{id})
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, errortypes.NewNotFoundError("Memory not found")
	}

	return found[0], nil
}

func (s *Desk) DeskSettings(
	ctx context.Context,
	actor *agentmemoryports.DeskMemoryActor,
) (*agentmemoryports.DeskMemorySettings, error) {
	person, err := s.person(ctx, actor)
	if err != nil {
		return nil, err
	}

	return s.deskSettings(ctx, person)
}

func (s *Desk) deskSettings(
	ctx context.Context,
	person *deskPerson,
) (*agentmemoryports.DeskMemorySettings, error) {
	mode, err := s.core.SavingMode(ctx, person.tenant, person.userID)
	if err != nil {
		return nil, err
	}
	held, err := s.core.HeldRoles(ctx, person.tenant, person.userID)
	if err != nil {
		return nil, err
	}

	// Anyone may write a memory for a team they are on or the organization:
	// one who does not manage shared memory offers it rather than keeps it.
	settings := &agentmemoryports.DeskMemorySettings{
		SavingMode:               mode,
		Roles:                    make([]agentmemoryports.DeskMemoryRole, 0, len(held)),
		CanShareWithOrganization: true,
		ManagesShared:            person.MayManageShared,
	}
	for _, role := range held {
		settings.Roles = append(settings.Roles, agentmemoryports.DeskMemoryRole{
			ID:       role.ID,
			Name:     role.Name,
			Writable: true,
		})
	}

	return settings, nil
}

func (s *Desk) SetSavingMode(
	ctx context.Context,
	actor *agentmemoryports.DeskMemoryActor,
	mode agentmemory.MemorySavingMode,
) (*agentmemoryports.DeskMemorySettings, error) {
	person, err := s.person(ctx, actor)
	if err != nil {
		return nil, err
	}

	entity := &agentmemory.MemoryPreference{
		OrganizationID: person.tenant.OrgID,
		BusinessUnitID: person.tenant.BuID,
		UserID:         person.userID,
		SavingMode:     mode,
	}
	me := errortypes.NewMultiError()
	entity.Validate(me)
	if me.HasErrors() {
		return nil, me
	}
	if _, err = s.repo.SavePreference(ctx, entity); err != nil {
		return nil, err
	}

	return s.deskSettings(ctx, person)
}

func (s *Desk) CreateDesk(
	ctx context.Context,
	req *agentmemoryports.CreateDeskMemoryRequest,
) (*agentmemoryports.DeskMemory, error) {
	person, err := s.person(ctx, req.Actor)
	if err != nil {
		return nil, err
	}
	kind := req.Kind
	if kind == "" {
		kind = agentmemory.MemoryKindInstruction
	}
	if !slices.Contains(deskWrittenKinds(), kind) {
		return nil, errortypes.NewValidationError(
			"kind",
			errortypes.ErrInvalid,
			"Write an instruction, a fact or a procedure",
		)
	}
	content, err := checkDeskContent(req.Content, kind)
	if err != nil {
		return nil, err
	}
	offer, err := person.mayKeepFor(req.Scope, req.RoleID)
	if err != nil {
		return nil, err
	}

	// What a person writes down for their agents is a rule to follow unless
	// they say it is a fact to weigh. One for a team or the organization from
	// a person who does not manage shared memory is offered for an
	// administrator to approve.
	created, err := s.core.Keep(ctx, &agentmemoryports.RememberRequest{
		TenantInfo:  person.tenant,
		Kind:        kind,
		Content:     content,
		SubjectType: req.SubjectType,
		SubjectID:   req.SubjectID,
		Scope:       req.Scope,
		OwnerUserID: person.userID,
		RoleID:      req.RoleID,
		Suggest:     offer,
	}, person.Actor, person.MayManageShared)
	if err != nil {
		return nil, err
	}

	return s.deskOne(ctx, person, created.ID)
}

func deskWrittenKinds() []agentmemory.MemoryKind {
	return []agentmemory.MemoryKind{
		agentmemory.MemoryKindInstruction,
		agentmemory.MemoryKindFact,
		agentmemory.MemoryKindProcedure,
	}
}

func (s *Desk) ReviseDesk(
	ctx context.Context,
	req *agentmemoryports.ReviseDeskMemoryRequest,
) (*agentmemoryports.DeskMemory, error) {
	person, err := s.person(ctx, req.Actor)
	if err != nil {
		return nil, err
	}
	current, err := s.changeable(ctx, person, req.ID)
	if err != nil {
		return nil, err
	}

	content := current.Content
	if strings.TrimSpace(req.Content) != "" {
		if content, err = checkDeskContent(req.Content, current.Kind); err != nil {
			return nil, err
		}
	}
	scope, roleID := current.Scope, pulid.Nil
	if current.RoleID != nil {
		roleID = *current.RoleID
	}
	if req.Scope != "" && (req.Scope != current.Scope || req.RoleID != roleID) {
		if req.Scope == agentmemory.MemoryScopeAgent {
			return nil, errortypes.NewValidationError(
				"scope",
				errortypes.ErrInvalid,
				"Keep a memory for yourself, a role you hold or the organization",
			)
		}
		offer, keepErr := person.mayKeepFor(req.Scope, req.RoleID)
		if keepErr != nil {
			return nil, keepErr
		}
		if offer {
			return nil, errOfferInstead()
		}
		scope, roleID = req.Scope, req.RoleID
	}
	owner := pulid.Nil
	if current.OwnerUserID != nil {
		owner = *current.OwnerUserID
	}
	if scope == agentmemory.MemoryScopeUser && current.Scope != agentmemory.MemoryScopeUser {
		owner = person.userID
	}

	previous := jsonutils.MustToJSON(current)
	revised, err := s.repo.Revise(ctx, &agentmemoryports.ReviseAgentMemoryRequest{
		ID:          current.ID,
		TenantInfo:  person.tenant,
		Content:     content,
		Scope:       scope,
		OwnerUserID: owner,
		RoleID:      roleID,
		Version:     req.Version,
	})
	if err != nil {
		return nil, err
	}

	s.core.LogChange(ctx,
		revised,
		previous,
		person.Actor,
		permission.OpUpdate,
		"Agent memory changed on the Desk",
	)
	s.core.QueueForRetrieval(ctx, revised)

	return s.deskOne(ctx, person, revised.ID)
}

// changeable reads a memory the person sees and may change; one they do not
// see is not found, and one they see but may not change is refused.
func (s *Desk) changeable(
	ctx context.Context,
	person *deskPerson,
	id pulid.ID,
) (*agentmemory.Memory, error) {
	current, err := s.repo.GetByID(ctx, agentmemoryports.GetAgentMemoryByIDRequest{
		ID:         id,
		TenantInfo: person.tenant,
	})
	if err != nil {
		return nil, err
	}
	if !person.sees(current) {
		return nil, errortypes.NewNotFoundError("Memory not found")
	}
	if current.Status.IsSuggestion() {
		return nil, errortypes.NewValidationError(
			"status", errortypes.ErrInvalid, "Save or dismiss this suggested memory first",
		)
	}
	if !person.mayChange(current) {
		return nil, agentmemoryports.ErrSharedMemoryManaged()
	}

	return current, nil
}

func (s *Desk) SetDeskStatus(
	ctx context.Context,
	req *agentmemoryports.SetDeskMemoryStatusRequest,
) (*agentmemoryports.DeskMemory, error) {
	person, err := s.person(ctx, req.Actor)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(deskSetStatuses(), req.Status) {
		return nil, errortypes.NewValidationError(
			"status", errortypes.ErrInvalid, "A memory is resumed, paused or forgotten",
		)
	}
	if _, err = s.changeable(ctx, person, req.ID); err != nil {
		return nil, err
	}

	if _, err = s.core.SetStatusAs(ctx, &agentmemoryports.SetAgentMemoryStatusRequest{
		ID:         req.ID,
		TenantInfo: person.tenant,
		Status:     req.Status,
	}, person.Actor, person.MayManageShared); err != nil {
		return nil, err
	}

	return s.deskOne(ctx, person, req.ID)
}

// offered reads a suggestion an agent made in the person's own conversation.
// One they turned down can still be accepted, which is the conversation's Undo
// of "Don't save"; it is never turned down twice.
func (s *Desk) offered(
	ctx context.Context,
	person *deskPerson,
	id pulid.ID,
	reconsider bool,
) (*agentmemory.Memory, error) {
	current, err := s.repo.GetByID(ctx, agentmemoryports.GetAgentMemoryByIDRequest{
		ID:         id,
		TenantInfo: person.tenant,
	})
	if err != nil {
		return nil, err
	}
	if !person.offeredTo(current) {
		return nil, errortypes.NewNotFoundError("Memory not found")
	}
	waiting := current.Status == agentmemory.MemoryStatusSuggested ||
		(reconsider && current.Status == agentmemory.MemoryStatusDismissed)
	if !waiting {
		return nil, errortypes.NewBusinessError("This memory has already been saved or dismissed")
	}

	return current, nil
}

func (s *Desk) ConfirmDesk(
	ctx context.Context,
	req *agentmemoryports.ConfirmDeskMemoryRequest,
) (*agentmemoryports.DeskMemory, error) {
	person, err := s.person(ctx, req.Actor)
	if err != nil {
		return nil, err
	}
	current, err := s.offered(ctx, person, req.ID, true)
	if err != nil {
		return nil, err
	}
	if !current.Source.OfferedByAgent() {
		return nil, errortypes.NewBusinessError(
			"You offered this memory for others; an administrator decides whether it is kept",
		)
	}
	content, err := checkDeskContent(req.Content, current.Kind)
	if err != nil {
		return nil, err
	}
	scope := req.Scope
	if scope == "" {
		scope = current.Scope
	}
	if current.Scope.Shared() && scope == agentmemory.MemoryScopeUser && !person.MayManageShared {
		return s.keepCopy(ctx, person, current, content)
	}
	if scope == agentmemory.MemoryScopeAgent && current.Scope != agentmemory.MemoryScopeAgent {
		return nil, errortypes.NewValidationError(
			"scope",
			errortypes.ErrInvalid,
			"Keep a memory for yourself, a role you hold or the organization",
		)
	}
	offer, err := person.mayKeepFor(scope, req.RoleID)
	if err != nil {
		return nil, err
	}
	if offer {
		return nil, errOfferInstead()
	}
	if err = s.mayReplace(ctx, person, current); err != nil {
		return nil, err
	}

	confirmed, err := s.repo.ResolveSuggestion(
		ctx,
		&agentmemoryports.ResolveAgentMemorySuggestionRequest{
			ID:           current.ID,
			TenantInfo:   person.tenant,
			Status:       agentmemory.MemoryStatusActive,
			Reconsidered: current.Status == agentmemory.MemoryStatusDismissed,
			Kind:         current.Kind,
			Content:      content,
			Scope:        scope,
			OwnerUserID:  person.userID,
			RoleID:       req.RoleID,
			ByUserID:     person.userID,
			At:           timeutils.NowUnix(),
			Version:      req.Version,
		},
	)
	if err != nil {
		return nil, err
	}

	if current.Tainted {
		s.core.RecordReview(ctx, current, confirmed, person.Actor)
	} else {
		s.core.LogChange(
			ctx,
			confirmed,
			jsonutils.MustToJSON(current),
			person.Actor,
			permission.OpUpdate,
			"Suggested agent memory saved from the conversation",
		)
	}
	s.core.QueueForRetrieval(ctx, confirmed)
	s.core.RecordReplaced(ctx, confirmed, person.Actor)

	return s.deskOne(ctx, person, confirmed.ID)
}

func (s *Desk) keepCopy(
	ctx context.Context,
	person *deskPerson,
	offer *agentmemory.Memory,
	content string,
) (*agentmemoryports.DeskMemory, error) {
	created, err := s.core.Keep(ctx, &agentmemoryports.RememberRequest{
		TenantInfo:  person.tenant,
		Kind:        offer.Kind,
		Content:     content,
		SubjectType: offer.SubjectType,
		SubjectID:   pulid.ConvertFromPtr(offer.SubjectID),
		ToolName:    offer.ToolName,
		ExpiresAt:   offer.ExpiresAt,
		Scope:       agentmemory.MemoryScopeUser,
		OwnerUserID: person.userID,
	}, person.Actor, person.MayManageShared)
	if err != nil {
		return nil, err
	}

	return s.deskOne(ctx, person, created.ID)
}

func (s *Desk) mayReplace(
	ctx context.Context,
	person *deskPerson,
	offered *agentmemory.Memory,
) error {
	if !offered.Replaces() {
		return nil
	}

	replaced, err := s.repo.GetByID(ctx, agentmemoryports.GetAgentMemoryByIDRequest{
		ID:         *offered.SupersedesID,
		TenantInfo: person.tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}

		return err
	}
	if !replaced.Status.Replaceable() {
		return nil
	}
	shared := replaced.Scope == agentmemory.MemoryScopeAgent && person.MayManageShared
	if !shared && (!person.sees(replaced) || !person.mayChange(replaced)) {
		return agentmemoryports.ErrSharedMemoryManaged()
	}

	return nil
}

func (s *Desk) DismissDesk(
	ctx context.Context,
	req agentmemoryports.DeskMemoryRef,
) (*agentmemoryports.DeskMemory, error) {
	person, err := s.person(ctx, req.Actor)
	if err != nil {
		return nil, err
	}
	current, err := s.offered(ctx, person, req.ID, false)
	if err != nil {
		return nil, err
	}

	dismissed, err := s.repo.ResolveSuggestion(
		ctx,
		&agentmemoryports.ResolveAgentMemorySuggestionRequest{
			ID:         current.ID,
			TenantInfo: person.tenant,
			Status:     agentmemory.MemoryStatusDismissed,
			ByUserID:   person.userID,
			At:         timeutils.NowUnix(),
			Version:    current.Version,
		},
	)
	if err != nil {
		return nil, err
	}

	s.core.LogChange(
		ctx,
		dismissed,
		jsonutils.MustToJSON(current),
		person.Actor,
		permission.OpUpdate,
		"Suggested agent memory not saved",
	)

	return s.deskOne(ctx, person, dismissed.ID)
}

// deskListedScopes are the audiences the Desk's page lists and filters by:
// a person's own, their roles' and the organization's.
func deskListedScopes() []agentmemory.MemoryScope {
	return []agentmemory.MemoryScope{
		agentmemory.MemoryScopeUser,
		agentmemory.MemoryScopeRole,
		agentmemory.MemoryScopeOrganization,
	}
}

// deskSetStatuses are the states a person puts a kept memory in: resumed,
// paused or forgotten. An offer is accepted or turned down, not set.
func deskSetStatuses() []agentmemory.MemoryStatus {
	return []agentmemory.MemoryStatus{
		agentmemory.MemoryStatusActive,
		agentmemory.MemoryStatusPaused,
		agentmemory.MemoryStatusRetired,
	}
}

func (p *deskPerson) cursorBinding() pagination.CursorBinding {
	return pagination.BindCursor(p.tenant, deskCursorResource+p.userID.String(), nil)
}

func decodeDeskCursor(
	binding pagination.CursorBinding,
	raw string,
) (*agentmemoryports.DeskMemoryCursor, error) {
	if raw == "" {
		//nolint:nilnil // no cursor is the first page, not an error or a missing value
		return nil, nil
	}

	cursor, err := pagination.DecodePositionCursor(binding, raw)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"after",
			errortypes.ErrInvalid,
			"The page cursor is not valid",
		)
	}

	return &agentmemoryports.DeskMemoryCursor{CreatedAt: cursor.At, ID: cursor.ID}, nil
}
