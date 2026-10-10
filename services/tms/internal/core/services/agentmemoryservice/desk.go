package agentmemoryservice

import (
	"context"
	"encoding/base64"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
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
// Who may do what is decided here, from the person and the two agent-memory
// permissions their resolver read for them:
//
//   - A person's own ("Just you") memories are theirs to write, change, pause
//     and forget with no permission beyond using the Desk.
//   - A role's memories reach everyone holding the role, and the
//     organization's reach everyone, so writing one, or changing, pausing or
//     forgetting one, needs the agent-memory create or update permission, and
//     for a role also holding that role. Without them a person still reads
//     them and sees them used, but cannot change them.
//
// Agent-scoped memories are administered in AI Control and never listed here.

const deskPageDefault = 50

type roleReader interface {
	GetUserRoleAssignments(
		ctx context.Context,
		userID, orgID pulid.ID,
	) ([]*permission.UserRoleAssignment, error)
	GetRolesWithInheritance(ctx context.Context, roleIDs []pulid.ID) ([]*permission.Role, error)
}

func rolesOrNil(roles repositories.RoleRepository) roleReader {
	if roles == nil {
		return nil
	}

	return roles
}

// heldRoles are the roles a person holds directly, unexpired, with their names.
func (s *Service) heldRoles(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) ([]*permission.Role, error) {
	if s.roles == nil || userID.IsNil() {
		return nil, nil
	}

	assignments, err := s.roles.GetUserRoleAssignments(ctx, userID, tenant.OrgID)
	if err != nil {
		return nil, err
	}

	roles := make([]*permission.Role, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment == nil || assignment.IsExpired() || assignment.Role == nil {
			continue
		}
		roles = append(roles, assignment.Role)
	}
	slices.SortFunc(roles, func(a, b *permission.Role) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	return roles, nil
}

func (s *Service) Reader(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) (agent.MemoryReader, error) {
	reader := agent.MemoryReader{UserID: userID}
	if s.roles == nil || userID.IsNil() {
		return reader, nil
	}

	held, err := s.heldRoles(ctx, tenant, userID)
	if err != nil {
		return reader, err
	}
	if len(held) == 0 {
		return reader, nil
	}

	ids := make([]pulid.ID, 0, len(held))
	for _, role := range held {
		ids = append(ids, role.ID)
	}
	// A role inherits what its parents grant, and a memory kept for the
	// parent role is part of what its holders work under.
	closure, err := s.roles.GetRolesWithInheritance(ctx, ids)
	if err != nil {
		return reader, err
	}
	for _, role := range closure {
		if role != nil && !slices.Contains(ids, role.ID) {
			ids = append(ids, role.ID)
		}
	}
	reader.RoleIDs = ids

	return reader, nil
}

// promptReader is Reader for a prompt or a recall, which go on without the
// person's roles rather than fail when they cannot be read.
func (s *Service) promptReader(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) agent.MemoryReader {
	reader, err := s.Reader(ctx, tenant, userID)
	if err != nil {
		s.l.Warn("agent memory: the person's roles could not be read; reading their own memories only",
			zap.String("user", userID.String()),
			zap.Error(err),
		)
	}

	return reader
}

func (s *Service) SavingMode(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) (agent.MemorySavingMode, error) {
	if userID.IsNil() {
		return agent.MemorySavingAutomatic, nil
	}

	preference, err := s.repo.GetPreference(ctx, repositories.GetAgentMemoryPreferenceRequest{
		TenantInfo: tenant,
		UserID:     userID,
	})
	if err != nil {
		return "", err
	}
	if preference.AsksFirst() {
		return agent.MemorySavingAskFirst, nil
	}

	return agent.MemorySavingAutomatic, nil
}

// deskPerson is the person on the Desk with who they read memories as.
type deskPerson struct {
	*services.DeskMemoryActor

	tenant pagination.TenantInfo
	userID pulid.ID
	reader agent.MemoryReader
}

func (s *Service) person(ctx context.Context, actor *services.DeskMemoryActor) (*deskPerson, error) {
	if actor == nil || !actor.Actor.IsUser() || actor.Actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError("Only a person keeps memories on the Desk")
	}

	tenant := actor.Actor.TenantInfo()
	reader, err := s.Reader(ctx, tenant, actor.Actor.UserID)
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
// conversation it was offered in.
func (p *deskPerson) sees(memory *agent.Memory) bool {
	if memory.Scope == agent.MemoryScopeAgent {
		return memory.Source == agent.MemorySourceReflection && p.offeredTo(memory)
	}
	if memory.Status.IsSuggestion() {
		return memory.Source.OfferedByAgent() && p.offeredTo(memory)
	}

	return p.reader.Reads(memory)
}

func (p *deskPerson) offeredTo(memory *agent.Memory) bool {
	return memory.CreatedByUserID != nil && *memory.CreatedByUserID == p.userID
}

func (p *deskPerson) holds(roleID pulid.ID) bool {
	return roleID.IsNotNil() && slices.Contains(p.reader.RoleIDs, roleID)
}

// mayChange reports whether the person may change, pause or forget a memory
// they see.
func (p *deskPerson) mayChange(memory *agent.Memory) bool {
	switch memory.Scope {
	case agent.MemoryScopeUser:
		return memory.OwnerUserID != nil && *memory.OwnerUserID == p.userID
	case agent.MemoryScopeRole:
		return memory.RoleID != nil && p.holds(*memory.RoleID) && p.MayUpdateShared
	case agent.MemoryScopeOrganization, agent.MemoryScopeAgent:
		return p.MayUpdateShared
	default:
		return false
	}
}

// mayKeepFor reports whether the person may put a memory in front of the
// scope's readers, writing a new one there or moving one there.
func (p *deskPerson) mayKeepFor(scope agent.MemoryScope, roleID pulid.ID) error {
	switch scope {
	case agent.MemoryScopeUser:
		return nil
	case agent.MemoryScopeRole:
		if !p.holds(roleID) {
			return errortypes.NewValidationError(
				"roleId", errortypes.ErrInvalid, "You can only keep a memory for a role you hold",
			)
		}
		if !p.MayCreateShared {
			return errortypes.NewAuthorizationError(
				"Keeping a memory for your team needs permission to create agent memories",
			)
		}

		return nil
	case agent.MemoryScopeOrganization:
		if !p.MayCreateShared {
			return errortypes.NewAuthorizationError(
				"Keeping a memory for the whole organization needs permission to create agent memories",
			)
		}

		return nil
	case agent.MemoryScopeAgent:
		if !p.MayCreateShared {
			return errortypes.NewAuthorizationError(
				"Keeping a memory for everyone who uses this agent needs permission to create agent memories",
			)
		}

		return nil
	default:
		return errortypes.NewValidationError(
			"scope", errortypes.ErrInvalid, "Keep a memory for yourself, a role you hold or the organization",
		)
	}
}

func (p *deskPerson) desk(row *repositories.DeskMemoryRow) *services.DeskMemory {
	return &services.DeskMemory{
		Memory:      row.Memory,
		SourceTitle: row.SourceTitle,
		RoleName:    row.RoleName,
		Editable:    p.mayChange(row.Memory),
	}
}

func checkDeskContent(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", errortypes.NewValidationError("content", errortypes.ErrRequired, "Say what to remember")
	}
	if utf8.RuneCountInString(content) > agent.MaxMemoryContentChars {
		return "", errortypes.NewValidationError(
			"content", errortypes.ErrInvalid,
			"A memory is at most {0} characters", agent.MaxMemoryContentChars,
		)
	}

	return content, nil
}

func (s *Service) ListDesk(
	ctx context.Context,
	req *services.ListDeskMemoriesRequest,
) (*services.DeskMemoryPage, error) {
	person, err := s.person(ctx, req.Actor)
	if err != nil {
		return nil, err
	}
	switch req.Scope {
	case "", agent.MemoryScopeUser, agent.MemoryScopeRole, agent.MemoryScopeOrganization:
	default:
		return nil, errortypes.NewValidationError("scope", errortypes.ErrInvalid, "Filter by a scope the Desk lists")
	}
	after, err := decodeDeskCursor(req.After)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = deskPageDefault
	}

	filter := repositories.DeskMemoryFilter{
		TenantInfo: person.tenant,
		Reader:     person.reader,
		Scope:      req.Scope,
		RoleID:     req.RoleID,
		Query:      req.Query,
	}
	rows, err := s.repo.ListDesk(ctx, repositories.ListDeskMemoriesRequest{
		Filter: filter,
		After:  after,
		Limit:  limit + 1,
	})
	if err != nil {
		return nil, err
	}

	page := &services.DeskMemoryPage{Items: make([]*services.DeskMemory, 0, min(len(rows), limit))}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1].Memory
		page.Next = encodeDeskCursor(last.CreatedAt, last.ID)
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
	counts, err := s.repo.CountDesk(ctx, filter)
	if err != nil {
		return nil, err
	}
	for _, count := range counts {
		page.All += count.Count
		page.Counts = append(page.Counts, services.DeskMemoryCount{
			Scope:  count.Scope,
			RoleID: count.RoleID,
			Count:  count.Count,
		})
	}

	return page, nil
}

func (s *Service) DeskByIDs(
	ctx context.Context,
	actor *services.DeskMemoryActor,
	ids []pulid.ID,
) ([]*services.DeskMemory, error) {
	person, err := s.person(ctx, actor)
	if err != nil {
		return nil, err
	}

	return s.deskByIDs(ctx, person, ids)
}

func (s *Service) deskByIDs(
	ctx context.Context,
	person *deskPerson,
	ids []pulid.ID,
) ([]*services.DeskMemory, error) {
	rows, err := s.repo.GetDesk(ctx, repositories.GetDeskMemoriesRequest{
		TenantInfo: person.tenant,
		IDs:        ids,
	})
	if err != nil {
		return nil, err
	}

	byID := make(map[pulid.ID]*repositories.DeskMemoryRow, len(rows))
	for _, row := range rows {
		if person.sees(row.Memory) {
			byID[row.Memory.ID] = row
		}
	}

	// In the order asked for, which is the order the conversation used them.
	out := make([]*services.DeskMemory, 0, len(byID))
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

func (s *Service) deskLinks(
	ctx context.Context,
	person *deskPerson,
	items []*services.DeskMemory,
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

	replaced, err := s.repo.ListByIDs(ctx, repositories.ListAgentMemoriesByIDsRequest{
		TenantInfo: person.tenant,
		IDs:        replacedIDs,
	})
	if err != nil {
		return err
	}
	replacedByID := make(map[pulid.ID]*agent.Memory, len(replaced))
	for _, memory := range replaced {
		replacedByID[memory.ID] = memory
	}

	replacements, err := s.repo.ListReplacements(
		ctx,
		repositories.ListAgentMemoryReplacementsRequest{
			TenantInfo:  person.tenant,
			ReplacedIDs: ids,
		},
	)
	if err != nil {
		return err
	}
	newest := agent.NewestReplacements(replacements)

	for _, item := range items {
		if item.Memory.Replaces() {
			item.Replaces = person.link(replacedByID[*item.Memory.SupersedesID])
		}
		item.ReplacedBy = person.link(newest[item.Memory.ID])
	}

	return nil
}

func (p *deskPerson) link(memory *agent.Memory) *services.DeskMemoryLink {
	if memory == nil || !p.sees(memory) {
		return nil
	}

	return &services.DeskMemoryLink{
		ID:      memory.ID,
		Content: memory.Content,
		Status:  memory.Status,
	}
}

func (s *Service) deskOne(
	ctx context.Context,
	person *deskPerson,
	id pulid.ID,
) (*services.DeskMemory, error) {
	found, err := s.deskByIDs(ctx, person, []pulid.ID{id})
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, errortypes.NewNotFoundError("Memory not found")
	}

	return found[0], nil
}

func (s *Service) DeskSettings(
	ctx context.Context,
	actor *services.DeskMemoryActor,
) (*services.DeskMemorySettings, error) {
	person, err := s.person(ctx, actor)
	if err != nil {
		return nil, err
	}

	return s.deskSettings(ctx, person)
}

func (s *Service) deskSettings(
	ctx context.Context,
	person *deskPerson,
) (*services.DeskMemorySettings, error) {
	mode, err := s.SavingMode(ctx, person.tenant, person.userID)
	if err != nil {
		return nil, err
	}
	held, err := s.heldRoles(ctx, person.tenant, person.userID)
	if err != nil {
		return nil, err
	}

	settings := &services.DeskMemorySettings{
		SavingMode:               mode,
		Roles:                    make([]services.DeskMemoryRole, 0, len(held)),
		CanShareWithOrganization: person.MayCreateShared,
	}
	for _, role := range held {
		settings.Roles = append(settings.Roles, services.DeskMemoryRole{
			ID:       role.ID,
			Name:     role.Name,
			Writable: person.MayCreateShared,
		})
	}

	return settings, nil
}

func (s *Service) SetSavingMode(
	ctx context.Context,
	actor *services.DeskMemoryActor,
	mode agent.MemorySavingMode,
) (*services.DeskMemorySettings, error) {
	person, err := s.person(ctx, actor)
	if err != nil {
		return nil, err
	}

	entity := &agent.MemoryPreference{
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

func (s *Service) CreateDesk(
	ctx context.Context,
	req *services.CreateDeskMemoryRequest,
) (*services.DeskMemory, error) {
	person, err := s.person(ctx, req.Actor)
	if err != nil {
		return nil, err
	}
	content, err := checkDeskContent(req.Content)
	if err != nil {
		return nil, err
	}
	if err = person.mayKeepFor(req.Scope, req.RoleID); err != nil {
		return nil, err
	}

	// What a person writes down for their agents is a rule to follow, not a
	// fact to weigh.
	created, err := s.Remember(ctx, &services.RememberRequest{
		TenantInfo:  person.tenant,
		Kind:        agent.MemoryKindInstruction,
		Content:     content,
		Scope:       req.Scope,
		OwnerUserID: person.userID,
		RoleID:      req.RoleID,
	}, person.Actor)
	if err != nil {
		return nil, err
	}

	return s.deskOne(ctx, person, created.ID)
}

func (s *Service) ReviseDesk(
	ctx context.Context,
	req *services.ReviseDeskMemoryRequest,
) (*services.DeskMemory, error) {
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
		if content, err = checkDeskContent(req.Content); err != nil {
			return nil, err
		}
	}
	scope, roleID := current.Scope, pulid.Nil
	if current.RoleID != nil {
		roleID = *current.RoleID
	}
	if req.Scope != "" && (req.Scope != current.Scope || req.RoleID != roleID) {
		if req.Scope == agent.MemoryScopeAgent {
			return nil, errortypes.NewValidationError(
				"scope",
				errortypes.ErrInvalid,
				"Keep a memory for yourself, a role you hold or the organization",
			)
		}
		if err = person.mayKeepFor(req.Scope, req.RoleID); err != nil {
			return nil, err
		}
		scope, roleID = req.Scope, req.RoleID
	}
	owner := pulid.Nil
	if current.OwnerUserID != nil {
		owner = *current.OwnerUserID
	}
	if scope == agent.MemoryScopeUser && current.Scope != agent.MemoryScopeUser {
		owner = person.userID
	}

	previous := jsonutils.MustToJSON(current)
	revised, err := s.repo.Revise(ctx, repositories.ReviseAgentMemoryRequest{
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

	s.logChange(revised, previous, person.Actor, permission.OpUpdate, "Agent memory changed on the Desk")
	s.queueForRetrieval(ctx, revised)

	return s.deskOne(ctx, person, revised.ID)
}

// changeable reads a memory the person sees and may change; one they do not
// see is not found, and one they see but may not change is refused.
func (s *Service) changeable(
	ctx context.Context,
	person *deskPerson,
	id pulid.ID,
) (*agent.Memory, error) {
	current, err := s.repo.GetByID(ctx, repositories.GetAgentMemoryByIDRequest{
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
		return nil, errortypes.NewAuthorizationError(
			"Changing a memory your team or organization shares needs permission to update agent memories",
		)
	}

	return current, nil
}

func (s *Service) SetDeskStatus(
	ctx context.Context,
	req *services.SetDeskMemoryStatusRequest,
) (*services.DeskMemory, error) {
	person, err := s.person(ctx, req.Actor)
	if err != nil {
		return nil, err
	}
	switch req.Status {
	case agent.MemoryStatusActive, agent.MemoryStatusPaused, agent.MemoryStatusRetired:
	default:
		return nil, errortypes.NewValidationError(
			"status", errortypes.ErrInvalid, "A memory is resumed, paused or forgotten",
		)
	}
	if _, err = s.changeable(ctx, person, req.ID); err != nil {
		return nil, err
	}

	if _, err = s.SetStatus(ctx, services.SetAgentMemoryStatusRequest{
		ID:         req.ID,
		TenantInfo: person.tenant,
		Status:     req.Status,
	}, person.Actor); err != nil {
		return nil, err
	}

	return s.deskOne(ctx, person, req.ID)
}

// offered reads a suggestion an agent made in the person's own conversation.
// One they turned down can still be accepted, which is the conversation's Undo
// of "Don't save"; it is never turned down twice.
func (s *Service) offered(
	ctx context.Context,
	person *deskPerson,
	id pulid.ID,
	reconsider bool,
) (*agent.Memory, error) {
	current, err := s.repo.GetByID(ctx, repositories.GetAgentMemoryByIDRequest{
		ID:         id,
		TenantInfo: person.tenant,
	})
	if err != nil {
		return nil, err
	}
	if !current.Source.OfferedByAgent() || !person.offeredTo(current) {
		return nil, errortypes.NewNotFoundError("Memory not found")
	}
	waiting := current.Status == agent.MemoryStatusSuggested ||
		(reconsider && current.Status == agent.MemoryStatusDismissed)
	if !waiting {
		return nil, errortypes.NewBusinessError("This memory has already been saved or dismissed")
	}

	return current, nil
}

func (s *Service) ConfirmDesk(
	ctx context.Context,
	req *services.ConfirmDeskMemoryRequest,
) (*services.DeskMemory, error) {
	person, err := s.person(ctx, req.Actor)
	if err != nil {
		return nil, err
	}
	current, err := s.offered(ctx, person, req.ID, true)
	if err != nil {
		return nil, err
	}
	content, err := checkDeskContent(req.Content)
	if err != nil {
		return nil, err
	}
	scope := req.Scope
	if scope == "" {
		scope = current.Scope
	}
	if scope == agent.MemoryScopeAgent && current.Scope != agent.MemoryScopeAgent {
		return nil, errortypes.NewValidationError(
			"scope",
			errortypes.ErrInvalid,
			"Keep a memory for yourself, a role you hold or the organization",
		)
	}
	if err = person.mayKeepFor(scope, req.RoleID); err != nil {
		return nil, err
	}
	if err = s.mayReplace(ctx, person, current); err != nil {
		return nil, err
	}

	confirmed, err := s.repo.ResolveSuggestion(ctx, repositories.ResolveAgentMemorySuggestionRequest{
		ID:           current.ID,
		TenantInfo:   person.tenant,
		Status:       agent.MemoryStatusActive,
		Reconsidered: current.Status == agent.MemoryStatusDismissed,
		Kind:         current.Kind,
		Content:      content,
		Scope:        scope,
		OwnerUserID:  person.userID,
		RoleID:       req.RoleID,
		ByUserID:     person.userID,
		At:           timeutils.NowUnix(),
		Version:      req.Version,
	})
	if err != nil {
		return nil, err
	}

	if current.Tainted {
		s.recordReview(ctx, current, confirmed, person.Actor)
	} else {
		s.logChange(confirmed, jsonutils.MustToJSON(current), person.Actor, permission.OpUpdate,
			"Suggested agent memory saved from the conversation")
	}
	s.queueForRetrieval(ctx, confirmed)
	s.recordReplaced(ctx, confirmed, person.Actor)

	return s.deskOne(ctx, person, confirmed.ID)
}

func (s *Service) mayReplace(ctx context.Context, person *deskPerson, offered *agent.Memory) error {
	if !offered.Replaces() {
		return nil
	}

	replaced, err := s.repo.GetByID(ctx, repositories.GetAgentMemoryByIDRequest{
		ID:         *offered.SupersedesID,
		TenantInfo: person.tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}

		return err
	}
	if !Replaceable(replaced.Status) {
		return nil
	}
	shared := replaced.Scope == agent.MemoryScopeAgent && person.MayUpdateShared
	if !shared && (!person.sees(replaced) || !person.mayChange(replaced)) {
		return errortypes.NewAuthorizationError(
			"This replaces a memory your team or organization shares; saving it needs permission to update agent memories",
		)
	}

	return nil
}

func (s *Service) DismissDesk(
	ctx context.Context,
	req services.DeskMemoryRef,
) (*services.DeskMemory, error) {
	person, err := s.person(ctx, req.Actor)
	if err != nil {
		return nil, err
	}
	current, err := s.offered(ctx, person, req.ID, false)
	if err != nil {
		return nil, err
	}

	dismissed, err := s.repo.ResolveSuggestion(ctx, repositories.ResolveAgentMemorySuggestionRequest{
		ID:         current.ID,
		TenantInfo: person.tenant,
		Status:     agent.MemoryStatusDismissed,
		ByUserID:   person.userID,
		At:         timeutils.NowUnix(),
		Version:    current.Version,
	})
	if err != nil {
		return nil, err
	}

	s.logChange(dismissed, jsonutils.MustToJSON(current), person.Actor, permission.OpUpdate,
		"Suggested agent memory not saved")

	return s.deskOne(ctx, person, dismissed.ID)
}

func encodeDeskCursor(createdAt int64, id pulid.ID) string {
	return base64.RawURLEncoding.EncodeToString(
		[]byte(strconv.FormatInt(createdAt, 10) + ":" + id.String()),
	)
}

func decodeDeskCursor(raw string) (*repositories.DeskMemoryCursor, error) {
	if raw == "" {
		return nil, nil
	}

	invalid := errortypes.NewValidationError("after", errortypes.ErrInvalid, "The page cursor is not valid")
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, invalid
	}
	at, rawID, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return nil, invalid
	}
	createdAt, err := strconv.ParseInt(at, 10, 64)
	if err != nil {
		return nil, invalid
	}
	id, err := pulid.Parse(rawID)
	if err != nil {
		return nil, invalid
	}

	return &repositories.DeskMemoryCursor{CreatedAt: createdAt, ID: id}, nil
}
