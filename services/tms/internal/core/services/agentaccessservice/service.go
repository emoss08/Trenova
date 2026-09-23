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
	Tools           services.AgentToolRegistry
	QueryTools      services.AgentQueryToolRegistry
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
	tools       services.AgentToolRegistry
	queryTools  services.AgentQueryToolRegistry
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
		tools:       p.Tools,
		queryTools:  p.QueryTools,
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

	roleIDs, err := validateIDs(idList{
		field:     "roleIds",
		ids:       req.RoleIDs,
		empty:     "Role cannot be empty",
		duplicate: "Role is listed more than once",
	})
	if err != nil {
		return nil, err
	}
	if !req.Mode.IsValid() {
		return nil, errortypes.NewValidationError(
			"accessMode", errortypes.ErrInvalid, "Access must be Everyone or Roles",
		)
	}

	var (
		definition   *agentdefinition.Definition
		roles        []*permission.Role
		previousMode agentdefinition.AccessMode
		change       repositories.GrantChange
	)
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		var txErr error
		definition, txErr = s.definitions.GetByID(txCtx, repositories.GetAgentDefinitionByIDRequest{
			ID:         req.AgentID,
			TenantInfo: req.TenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		if txErr = validateAgentAccess(definition, req.Mode, roleIDs); txErr != nil {
			return txErr
		}

		roles, txErr = s.tenantRoles(txCtx, req.TenantInfo, roleIDs)
		if txErr != nil {
			return txErr
		}

		previousMode = definition.EffectiveAccessMode()
		if definition.AccessMode != req.Mode {
			txErr = s.definitions.SetAccessMode(txCtx, repositories.SetAgentDefinitionAccessModeRequest{
				ID:         definition.ID,
				TenantInfo: req.TenantInfo,
				Mode:       req.Mode,
			})
			if txErr != nil {
				return txErr
			}
		}

		change, txErr = s.grants.ReplaceForAgent(txCtx, repositories.ReplaceAgentGrantsRequest{
			TenantInfo: req.TenantInfo,
			AgentID:    definition.ID,
			RoleIDs:    roleIDs,
			GrantedBy:  actor.UserIDOrNil(),
		})

		return txErr
	})
	if err != nil {
		return nil, err
	}

	definition.AccessMode = req.Mode
	s.invalidate(ctx, change.Added, change.Removed)
	if previousMode != req.Mode || change.Changed() {
		s.logAgentAudit(agentAuditEntry{
			definition:   definition,
			previousMode: previousMode,
			previousIDs:  change.Previous,
			currentIDs:   roleIDs,
			idsKey:       "roleIds",
			actor:        actor,
		})
	}
	s.logRoleGrantAudits(roleGrantAudit{
		tenant:  req.TenantInfo,
		agent:   definition,
		added:   change.Added,
		removed: change.Removed,
		actor:   actor,
	})

	return &services.AgentAccess{Agent: definition, Roles: roles}, nil
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
		result, err := s.permissions.Check(ctx, actor.PermissionCheck(resource, permission.OpUpdate))
		if err != nil {
			return fmt.Errorf("check %s permission: %w", resource, err)
		}
		if result == nil || !result.Allowed {
			return errortypes.NewAuthorizationError(
				"You don't have permission to perform this action: {0} {1}",
				resource, permission.OpUpdate,
			)
		}
	}

	return nil
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
			"A system agent is open to everyone who can use the assistant and is granted to no role")
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
			multiErr.Add(field, errortypes.ErrInvalid, "This agent does not exist in this organization")
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
