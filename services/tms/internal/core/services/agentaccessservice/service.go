package agentaccessservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxGrantsPerWrite = 100

type toolPolicies interface {
	Get(name string) (services.ToolPolicy, bool)
}

func policyLookup(catalog *agenttoolpolicy.Catalog) toolPolicies {
	if catalog == nil {
		return nil
	}

	return catalog
}

type Params struct {
	fx.In

	Logger          *zap.Logger
	DB              ports.DBConnection
	Definitions     repositories.AgentDefinitionRepository
	Grants          repositories.RoleAgentGrantRepository
	Roles           repositories.RoleRepository
	PermissionCache repositories.PermissionCacheRepository
	Permissions     services.PermissionEngine
	AuditService    services.AuditService
	Runtime         services.AgentRuntime
	Policies        *agenttoolpolicy.Catalog `optional:"true"`
	Registry        *permission.Registry
}

type Service struct {
	l           *zap.Logger
	db          ports.DBConnection
	definitions repositories.AgentDefinitionRepository
	grants      repositories.RoleAgentGrantRepository
	roles       repositories.RoleRepository
	cache       repositories.PermissionCacheRepository
	permissions services.PermissionEngine
	audit       services.AuditService
	runtime     services.AgentRuntime
	policies    toolPolicies
	sensitive   SensitiveToolRule
}

//nolint:gocritic // dependency injection
func New(p Params) services.AgentAccessService {
	return &Service{
		l:           p.Logger.Named("service.agentaccess"),
		db:          p.DB,
		definitions: p.Definitions,
		grants:      p.Grants,
		roles:       p.Roles,
		cache:       p.PermissionCache,
		permissions: p.Permissions,
		audit:       p.AuditService,
		runtime:     p.Runtime,
		policies:    policyLookup(p.Policies),
		sensitive:   DefaultSensitiveRule(p.Registry),
	}
}

func (s *Service) SetAgentAccess(
	ctx context.Context,
	req *services.SetAgentAccessRequest,
	actor *services.RequestActor,
) (*services.AgentAccess, error) {
	if err := s.authorize(ctx, actor, req.TenantInfo,
		permission.ResourceAgentDefinition, permission.ResourceRole); err != nil {
		return nil, err
	}

	roleIDs, err := validateAccessWrite(services.AgentAccessWrite{
		Mode:    req.Mode,
		RoleIDs: req.RoleIDs,
	})
	if err != nil {
		return nil, err
	}

	var written *accessWrite
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		definition, txErr := s.definitions.GetByID(txCtx, repositories.GetAgentDefinitionByIDRequest{
			ID:         req.AgentID,
			TenantInfo: req.TenantInfo,
		})
		if txErr != nil {
			return txErr
		}

		written, txErr = s.applyAccess(txCtx, accessChange{
			tenant:     req.TenantInfo,
			definition: definition,
			mode:       req.Mode,
			roleIDs:    roleIDs,
			actor:      actor,
		})

		return txErr
	})
	if err != nil {
		return nil, err
	}

	s.settleAccess(ctx, req.TenantInfo, written, actor)

	return &services.AgentAccess{Agent: written.definition, Roles: written.roles}, nil
}

// SaveWithAccess saves an agent and who may use it in one transaction, so an
// agent meant for some roles is never, even for a moment, open to everyone,
// and a refusal of either leaves neither. The save is the caller's and is
// authorized by the caller; changing who may use the agent is authorized
// here, and needs permission to update roles. Access that already reads as
// asked needs nothing more, so a person who may not update roles can still
// save an agent whose access they did not touch.
func (s *Service) SaveWithAccess(
	ctx context.Context,
	req *services.SaveAgentWithAccessRequest,
	actor *services.RequestActor,
) (*agentdefinition.Definition, error) {
	if req.Save == nil {
		return nil, errortypes.NewBusinessError("Nothing was given to save")
	}
	if err := s.authorize(ctx, actor, req.TenantInfo); err != nil {
		return nil, err
	}

	roleIDs, err := validateAccessWrite(req.Access)
	if err != nil {
		return nil, err
	}

	mayGrant, err := s.allowed(ctx, actor, permission.ResourceRole)
	if err != nil {
		return nil, err
	}

	var written *accessWrite
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		saved, txErr := req.Save(txCtx)
		if txErr != nil {
			return txErr
		}

		written, txErr = s.applyAccess(txCtx, accessChange{
			tenant:     req.TenantInfo,
			definition: saved,
			mode:       req.Access.Mode,
			roleIDs:    roleIDs,
			actor:      actor,
		})
		if txErr != nil {
			return txErr
		}
		if written.changed() && !mayGrant {
			return errAccessNeedsRoles()
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	s.settleAccess(ctx, req.TenantInfo, written, actor)

	return written.definition, nil
}

func errAccessNeedsRoles() error {
	return errortypes.NewAuthorizationError(
		"Changing who can use an agent needs permission to update roles as well as agents. " +
			"Save it without changing who can use it, or ask someone who can update roles.",
	)
}

// accessChange is who may use an agent, as one write asks it.
type accessChange struct {
	tenant     pagination.TenantInfo
	definition *agentdefinition.Definition
	mode       agentdefinition.AccessMode
	roleIDs    []pulid.ID
	actor      *services.RequestActor
}

// accessWrite is what setting who may use an agent changed.
type accessWrite struct {
	definition   *agentdefinition.Definition
	roles        []*permission.Role
	roleIDs      []pulid.ID
	previousMode agentdefinition.AccessMode
	change       repositories.GrantChange
}

func (w *accessWrite) changed() bool {
	return w.previousMode != w.definition.EffectiveAccessMode() || w.change.Changed()
}

// applyAccess sets the agent's access mode and replaces the roles granted it,
// inside the caller's transaction. The definition comes back with the mode
// it now has.
func (s *Service) applyAccess(ctx context.Context, req accessChange) (*accessWrite, error) {
	definition := req.definition
	if err := validateAgentAccess(definition, req.mode, req.roleIDs); err != nil {
		return nil, err
	}

	roles, err := s.tenantRoles(ctx, req.tenant, req.roleIDs)
	if err != nil {
		return nil, err
	}

	previousMode := definition.EffectiveAccessMode()
	if definition.AccessMode != req.mode {
		if err = s.definitions.SetAccessMode(ctx, repositories.SetAgentDefinitionAccessModeRequest{
			ID:         definition.ID,
			TenantInfo: req.tenant,
			Mode:       req.mode,
		}); err != nil {
			return nil, err
		}
	}

	change, err := s.grants.ReplaceForAgent(ctx, repositories.ReplaceAgentGrantsRequest{
		TenantInfo: req.tenant,
		AgentID:    definition.ID,
		RoleIDs:    req.roleIDs,
		GrantedBy:  req.actor.UserIDOrNil(),
	})
	if err != nil {
		return nil, err
	}

	definition.AccessMode = req.mode

	return &accessWrite{
		definition:   definition,
		roles:        roles,
		roleIDs:      req.roleIDs,
		previousMode: previousMode,
		change:       change,
	}, nil
}

// settleAccess does what follows a committed access write: the permission
// caches of the roles whose grants moved are dropped, and the change is
// audited on the agent and on each role.
func (s *Service) settleAccess(
	ctx context.Context,
	tenant pagination.TenantInfo,
	written *accessWrite,
	actor *services.RequestActor,
) {
	s.invalidate(ctx, written.change.Added, written.change.Removed)
	if written.changed() {
		s.logAgentAudit(agentAuditEntry{
			definition:   written.definition,
			previousMode: written.previousMode,
			previousIDs:  written.change.Previous,
			currentIDs:   written.roleIDs,
			idsKey:       "roleIds",
			actor:        actor,
		})
	}
	s.logRoleGrantAudits(roleGrantAudit{
		tenant:  tenant,
		agent:   written.definition,
		added:   written.change.Added,
		removed: written.change.Removed,
		actor:   actor,
	})
}

func validateAccessWrite(access services.AgentAccessWrite) ([]pulid.ID, error) {
	roleIDs, err := validateIDs(idList{
		field:     "roleIds",
		ids:       access.RoleIDs,
		empty:     "Role cannot be empty",
		duplicate: "Role is listed more than once",
	})
	if err != nil {
		return nil, err
	}
	if !access.Mode.IsValid() {
		return nil, errortypes.NewValidationError(
			"accessMode", errortypes.ErrInvalid, "Access must be Everyone or Roles",
		)
	}

	return roleIDs, nil
}

func (s *Service) SetRoleAgents(
	ctx context.Context,
	req *services.SetRoleAgentsRequest,
	actor *services.RequestActor,
) (*services.RoleAgents, error) {
	if err := s.authorize(ctx, actor, req.TenantInfo, permission.ResourceRole); err != nil {
		return nil, err
	}

	agentIDs, err := validateIDs(idList{
		field:     "agentIds",
		ids:       req.AgentIDs,
		empty:     "Agent cannot be empty",
		duplicate: "Agent is listed more than once",
	})
	if err != nil {
		return nil, err
	}

	var (
		role   *permission.Role
		agents []*agentdefinition.Definition
		change repositories.GrantChange
	)
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		roles, txErr := s.grants.ListGrantableRoles(txCtx, repositories.ListGrantableRolesRequest{
			TenantInfo: req.TenantInfo,
			RoleIDs:    []pulid.ID{req.RoleID},
		})
		if txErr != nil {
			return txErr
		}
		if len(roles) == 0 {
			return errortypes.NewNotFoundError("Role not found within your organization")
		}
		role = roles[0]

		agents, txErr = s.tenantAgents(txCtx, req.TenantInfo, agentIDs)
		if txErr != nil {
			return txErr
		}

		change, txErr = s.grants.ReplaceForRole(txCtx, repositories.ReplaceRoleGrantsRequest{
			TenantInfo: req.TenantInfo,
			RoleID:     role.ID,
			AgentIDs:   agentIDs,
			GrantedBy:  actor.UserIDOrNil(),
		})

		return txErr
	})
	if err != nil {
		return nil, err
	}

	if change.Changed() {
		s.invalidate(ctx, []pulid.ID{role.ID})
		s.logRoleAudit(role, change, agentIDs, actor)
		s.logAgentGrantAudits(req.TenantInfo, role, change, actor)
	}

	return &services.RoleAgents{Role: role, Agents: agents}, nil
}

func (s *Service) authorize(
	ctx context.Context,
	actor *services.RequestActor,
	tenant pagination.TenantInfo,
	resources ...permission.Resource,
) error {
	if actor == nil || !actor.IsUser() {
		return errortypes.NewAuthorizationError("Only a person can change who may use an agent")
	}
	if tenant.OrgID != actor.OrganizationID || tenant.BuID != actor.BusinessUnitID {
		return errortypes.NewAuthorizationError(
			"Access to an agent can only be changed within your own organization",
		)
	}

	for _, resource := range resources {
		allowed, err := s.allowed(ctx, actor, resource)
		if err != nil {
			return err
		}
		if !allowed {
			return errortypes.NewAuthorizationError(
				"You don't have permission to perform this action: {0} {1}",
				resource, permission.OpUpdate,
			)
		}
	}

	return nil
}

// allowed reports whether the actor may update the resource.
func (s *Service) allowed(
	ctx context.Context,
	actor *services.RequestActor,
	resource permission.Resource,
) (bool, error) {
	result, err := s.permissions.Check(ctx, actor.PermissionCheck(resource, permission.OpUpdate))
	if err != nil {
		return false, fmt.Errorf("check %s permission: %w", resource, err)
	}

	return result != nil && result.Allowed, nil
}

type idList struct {
	field     string
	ids       []pulid.ID
	empty     string
	duplicate string
}

func validateIDs(list idList) ([]pulid.ID, error) {
	multiErr := errortypes.NewMultiError()
	if len(list.ids) > maxGrantsPerWrite {
		multiErr.Add(list.field, errortypes.ErrInvalid, "At most 100 can be granted at once")
	}

	seen := make(map[pulid.ID]struct{}, len(list.ids))
	out := make([]pulid.ID, 0, len(list.ids))
	for idx, id := range list.ids {
		entry := fmt.Sprintf("%s[%d]", list.field, idx)
		if id.IsNil() {
			multiErr.Add(entry, errortypes.ErrRequired, list.empty)
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			multiErr.Add(entry, errortypes.ErrDuplicate, list.duplicate)
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return out, nil
}

func validateAgentAccess(
	definition *agentdefinition.Definition,
	mode agentdefinition.AccessMode,
	roleIDs []pulid.ID,
) error {
	multiErr := errortypes.NewMultiError()
	if refusal := definition.AccessRefusal(mode); refusal != "" {
		multiErr.Add("accessMode", errortypes.ErrInvalid, refusal)
	}
	if definition.IsSystem() && len(roleIDs) > 0 {
		multiErr.Add("roleIds", errortypes.ErrInvalid,
			"A system agent is open to everyone who can use the assistant and is granted "+
				"to no role")
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) tenantRoles(
	ctx context.Context,
	tenant pagination.TenantInfo,
	roleIDs []pulid.ID,
) ([]*permission.Role, error) {
	if len(roleIDs) == 0 {
		return []*permission.Role{}, nil
	}

	roles, err := s.grants.ListGrantableRoles(ctx, repositories.ListGrantableRolesRequest{
		TenantInfo: tenant,
		RoleIDs:    roleIDs,
	})
	if err != nil {
		return nil, err
	}

	found := make(map[pulid.ID]struct{}, len(roles))
	for _, role := range roles {
		found[role.ID] = struct{}{}
	}

	multiErr := errortypes.NewMultiError()
	for idx, id := range roleIDs {
		if _, ok := found[id]; !ok {
			multiErr.Add(fmt.Sprintf("roleIds[%d]", idx), errortypes.ErrInvalid,
				"This role does not exist in this organization")
		}
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return roles, nil
}

func (s *Service) tenantAgents(
	ctx context.Context,
	tenant pagination.TenantInfo,
	agentIDs []pulid.ID,
) ([]*agentdefinition.Definition, error) {
	if len(agentIDs) == 0 {
		return []*agentdefinition.Definition{}, nil
	}

	agents, err := s.definitions.ListByIDs(ctx, repositories.ListAgentDefinitionsByIDsRequest{
		IDs:        agentIDs,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	byID := make(map[pulid.ID]*agentdefinition.Definition, len(agents))
	for _, agent := range agents {
		byID[agent.ID] = agent
	}

	multiErr := errortypes.NewMultiError()
	ordered := make([]*agentdefinition.Definition, 0, len(agentIDs))
	for idx, id := range agentIDs {
		field := fmt.Sprintf("agentIds[%d]", idx)
		agent, ok := byID[id]
		switch {
		case !ok:
			multiErr.Add(
				field,
				errortypes.ErrInvalid,
				"This agent does not exist in this organization",
			)
		case agent.IsSystem():
			multiErr.Add(field, errortypes.ErrInvalid,
				"{0} is a system agent and is open to everyone who can use the assistant",
				agent.Name)
		default:
			ordered = append(ordered, agent)
		}
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return ordered, nil
}

func (s *Service) invalidate(ctx context.Context, groups ...[]pulid.ID) {
	if s.cache == nil {
		return
	}

	seen := make(map[pulid.ID]struct{})
	for _, roleIDs := range groups {
		for _, roleID := range roleIDs {
			if _, done := seen[roleID]; done {
				continue
			}
			seen[roleID] = struct{}{}
			if err := s.cache.InvalidateByRole(ctx, roleID, s.roles); err != nil {
				s.l.Warn("failed to invalidate permission cache after an agent grant change",
					zap.String("roleID", roleID.String()),
					zap.Error(err),
				)
			}
		}
	}
}

type agentAuditEntry struct {
	definition   *agentdefinition.Definition
	previousMode agentdefinition.AccessMode
	previousIDs  []pulid.ID
	currentIDs   []pulid.ID
	idsKey       string
	actor        *services.RequestActor
}

func (s *Service) logAgentAudit(entry agentAuditEntry) {
	auditActor := entry.actor.AuditActor()
	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceAgentDefinition,
		ResourceID:     entry.definition.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		OrganizationID: entry.definition.OrganizationID,
		BusinessUnitID: entry.definition.BusinessUnitID,
		PreviousState: map[string]any{
			"accessMode": entry.previousMode,
			entry.idsKey: idStrings(entry.previousIDs),
		},
		CurrentState: map[string]any{
			"accessMode": entry.definition.AccessMode,
			entry.idsKey: idStrings(entry.currentIDs),
		},
		Critical: true,
	}, auditservice.WithComment("Agent access changed")); err != nil {
		s.l.Error("failed to log agent access audit", zap.Error(err))
	}
}

type roleGrantAudit struct {
	tenant  pagination.TenantInfo
	agent   *agentdefinition.Definition
	added   []pulid.ID
	removed []pulid.ID
	actor   *services.RequestActor
}

func (s *Service) logRoleGrantAudits(entry roleGrantAudit) {
	for _, roleID := range entry.added {
		s.logRoleGrant(entry, roleID, true)
	}
	for _, roleID := range entry.removed {
		s.logRoleGrant(entry, roleID, false)
	}
}

func (s *Service) logRoleGrant(entry roleGrantAudit, roleID pulid.ID, granted bool) {
	comment := "Agent access removed"
	if granted {
		comment = "Agent access granted"
	}

	auditActor := entry.actor.AuditActor()
	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceRole,
		ResourceID:     roleID.String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		OrganizationID: entry.tenant.OrgID,
		BusinessUnitID: entry.tenant.BuID,
		CurrentState: map[string]any{
			"agentDefinitionId": entry.agent.ID.String(),
			"agentName":         entry.agent.Name,
			"granted":           granted,
		},
		Critical: true,
	}, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log role agent grant audit", zap.Error(err))
	}
}

func (s *Service) logRoleAudit(
	role *permission.Role,
	change repositories.GrantChange,
	agentIDs []pulid.ID,
	actor *services.RequestActor,
) {
	auditActor := actor.AuditActor()
	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceRole,
		ResourceID:     role.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		OrganizationID: role.OrganizationID,
		BusinessUnitID: role.BusinessUnitID,
		PreviousState:  map[string]any{"agentIds": idStrings(change.Previous)},
		CurrentState:   map[string]any{"agentIds": idStrings(agentIDs)},
		Critical:       true,
	}, auditservice.WithComment("Role agent access changed")); err != nil {
		s.l.Error("failed to log role agent access audit", zap.Error(err))
	}
}

func (s *Service) logAgentGrantAudits(
	tenant pagination.TenantInfo,
	role *permission.Role,
	change repositories.GrantChange,
	actor *services.RequestActor,
) {
	changed := sliceutils.Dedupe(append(append([]pulid.ID{}, change.Added...), change.Removed...))
	auditActor := actor.AuditActor()
	for _, agentID := range changed {
		granted := slices.Contains(change.Added, agentID)
		comment := "Role no longer granted this agent"
		if granted {
			comment = "Role granted this agent"
		}
		if err := s.audit.LogAction(&services.LogActionParams{
			Resource:       permission.ResourceAgentDefinition,
			ResourceID:     agentID.String(),
			Operation:      permission.OpUpdate,
			UserID:         auditActor.UserID,
			PrincipalType:  auditActor.PrincipalType,
			PrincipalID:    auditActor.PrincipalID,
			APIKeyID:       auditActor.APIKeyID,
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			CurrentState: map[string]any{
				"roleId":   role.ID.String(),
				"roleName": role.Name,
				"granted":  granted,
			},
			Critical: true,
		}, auditservice.WithComment(comment)); err != nil {
			s.l.Error("failed to log agent grant audit", zap.Error(err))
		}
	}
}

func idStrings(ids []pulid.ID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}

	return out
}
