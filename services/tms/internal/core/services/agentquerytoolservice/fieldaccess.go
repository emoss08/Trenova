package agentquerytoolservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

// fieldAccess answers which of a resource's fields a person may be shown,
// by the same sensitivity model the report compiler applies, so the worker
// a model reads through get_worker is the worker the person could open.
//
// Confidential fields are never sent to a model, whatever the person's
// role: a date of birth or a licence number in a chat transcript is a
// leak with nobody to blame. Restricted fields follow the person's role
// ceiling; Internal ones are shown to anyone who may read the record.
type fieldAccess struct {
	permissions serviceports.PermissionEngine
	registry    *permission.Registry
	threads     repositories.ThreadOwnerRepository
}

func newFieldAccess(permissions serviceports.PermissionEngine) fieldAccess {
	return fieldAccess{permissions: permissions, registry: permission.NewRegistry()}
}

func (a fieldAccess) withThreads(threads repositories.ThreadOwnerRepository) fieldAccess {
	a.threads = threads

	return a
}

// ceiling is the most sensitive tier the actor may be shown on a resource.
// An agent principal reads at its definition's data access, Internal when it
// has none. A person reads at their own role's tier, lowered to the agent's
// data access when an agent is working for them. An authorization that
// cannot be resolved reads at Internal.
func (a fieldAccess) ceiling(
	ctx context.Context,
	params *serviceports.QueryToolParams,
	resource permission.Resource,
) permission.FieldSensitivity {
	granted := agentDataAccess(params.DataAccessCeiling)
	if params.Actor == nil || !params.Actor.IsUser() {
		if granted == "" {
			return permission.SensitivityInternal
		}

		return granted
	}

	person := a.personCeiling(ctx, params, resource)
	if granted != "" && person.Level() > granted.Level() {
		return granted
	}

	return person
}

func agentDataAccess(granted permission.FieldSensitivity) permission.FieldSensitivity {
	switch {
	case granted == "":
		return ""
	case granted.CanAccess(permission.SensitivityRestricted):
		return permission.SensitivityRestricted
	default:
		return permission.SensitivityInternal
	}
}

func (a fieldAccess) personCeiling(
	ctx context.Context,
	params *serviceports.QueryToolParams,
	resource permission.Resource,
) permission.FieldSensitivity {
	if a.permissions == nil {
		return permission.SensitivityInternal
	}

	detail, err := a.permissions.GetResourcePermissions(
		ctx, params.Actor.UserID, params.OrganizationID, resource.String(),
	)
	if err != nil || detail == nil {
		return permission.SensitivityInternal
	}
	if detail.MaxSensitivity.CanAccess(permission.SensitivityConfidential) {
		// Nothing above Restricted reaches a model, however far the role goes.
		return permission.SensitivityRestricted
	}

	return detail.MaxSensitivity
}

// visible reports whether one field of a resource may be shown under a
// ceiling. Confidential is refused outright.
func (a fieldAccess) visible(
	resource permission.Resource,
	field string,
	ceiling permission.FieldSensitivity,
) bool {
	sensitivity := a.registry.GetFieldSensitivity(resource.String(), field)
	if sensitivity == permission.SensitivityConfidential {
		return false
	}

	return ceiling.CanAccess(sensitivity)
}

// mayRead reports whether the actor may read a resource other than the one
// the tool is gated on, for a row that joins two: a tractor's position
// carries its driver's name, and the name is the worker's to show. The
// question goes to the engine as the tool guard's own would, so an agent
// principal, a person and an API key are each answered by their own rules.
func (a fieldAccess) mayRead(
	ctx context.Context,
	params *serviceports.QueryToolParams,
	resource permission.Resource,
) bool {
	actor := params.Actor
	if actor == nil || a.permissions == nil {
		return false
	}

	result, err := a.permissions.Check(ctx, &serviceports.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       resource.String(),
		Operation:      permission.OpRead,
	})

	return err == nil && result != nil && result.Allowed
}

func (a fieldAccess) mayReadRecord(
	ctx context.Context,
	params *serviceports.QueryToolParams,
	resource permission.Resource,
	recordID string,
) bool {
	actor := params.Actor
	if actor == nil || a.permissions == nil || resource == "" {
		return false
	}

	req := &serviceports.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       resource.String(),
		Operation:      permission.OpRead,
	}
	if id, err := pulid.Parse(recordID); err == nil && id.IsNotNil() {
		req.ResourceID = &id
	}

	result, err := a.permissions.Check(ctx, req)

	return err == nil && result != nil && result.Allowed
}

func personOf(actor *serviceports.RequestActor) pulid.ID {
	if !actor.IsUser() {
		return pulid.Nil
	}

	return actor.UserID
}

func (a fieldAccess) readableDocuments(
	ctx context.Context,
	params *serviceports.QueryToolParams,
	docs []*document.Document,
	mayReadRecord func(permission.Resource, string) bool,
) (map[pulid.ID]bool, error) {
	person := personOf(params.Actor)
	owners, err := a.conversationOwners(ctx, params, person, docs)
	if err != nil {
		return nil, err
	}

	readable := make(map[pulid.ID]bool, len(docs))
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		owner := doc.OwnerResource()
		if !a.registry.HasResource(owner.String()) ||
			!doc.VisibleToPerson(owners, person) ||
			!mayReadRecord(owner, doc.ResourceID) {
			continue
		}
		readable[doc.ID] = true
	}

	return readable, nil
}

func (a fieldAccess) conversationOwners(
	ctx context.Context,
	params *serviceports.QueryToolParams,
	person pulid.ID,
	docs []*document.Document,
) (map[pulid.ID]pulid.ID, error) {
	if a.threads == nil || person.IsNil() {
		return map[pulid.ID]pulid.ID{}, nil
	}
	ids := document.ConversationIDs(docs)
	if len(ids) == 0 {
		return map[pulid.ID]pulid.ID{}, nil
	}

	owners, err := a.threads.ThreadOwners(ctx, repositories.ThreadOwnersRequest{
		TenantInfo: tenantOf(params),
		ThreadIDs:  ids,
	})
	if err != nil {
		return nil, fmt.Errorf("read who owns the conversations documents are attached to: %w", err)
	}

	return owners, nil
}

func (a fieldAccess) recordTextVisible(
	resource permission.Resource,
	ceiling permission.FieldSensitivity,
) bool {
	definition, ok := a.registry.Get(resource.String())
	if !ok || definition.DefaultSensitivity == permission.SensitivityConfidential {
		return false
	}

	return ceiling.CanAccess(definition.DefaultSensitivity)
}

func (a fieldAccess) forRetrieval(params *serviceports.QueryToolParams) *retrievalAccess {
	return &retrievalAccess{
		access:    a,
		params:    params,
		resources: make(map[permission.Resource]bool, 4),
		records:   make(map[string]bool, 16),
		ceilings:  make(map[permission.Resource]permission.FieldSensitivity, 4),
	}
}

type retrievalAccess struct {
	access    fieldAccess
	params    *serviceports.QueryToolParams
	resources map[permission.Resource]bool
	records   map[string]bool
	ceilings  map[permission.Resource]permission.FieldSensitivity
}

var _ serviceports.RetrievalAccess = (*retrievalAccess)(nil)

func (r *retrievalAccess) MayReadResource(
	ctx context.Context,
	resource permission.Resource,
) bool {
	allowed, seen := r.resources[resource]
	if !seen {
		allowed = r.access.mayRead(ctx, r.params, resource)
		r.resources[resource] = allowed
	}

	return allowed
}

func (r *retrievalAccess) MayReadRecord(
	ctx context.Context,
	resource permission.Resource,
	recordID string,
) bool {
	if !r.MayReadResource(ctx, resource) {
		return false
	}

	key := resource.String() + ":" + recordID
	allowed, seen := r.records[key]
	if !seen {
		allowed = r.access.mayReadRecord(ctx, r.params, resource, recordID)
		r.records[key] = allowed
	}

	return allowed
}

func (r *retrievalAccess) ReadableDocuments(
	ctx context.Context,
	docs []*document.Document,
) (map[pulid.ID]bool, error) {
	return r.access.readableDocuments(ctx, r.params, docs,
		func(resource permission.Resource, recordID string) bool {
			return r.MayReadRecord(ctx, resource, recordID)
		})
}

func (r *retrievalAccess) ceiling(
	ctx context.Context,
	resource permission.Resource,
) permission.FieldSensitivity {
	ceiling, seen := r.ceilings[resource]
	if !seen {
		ceiling = r.access.ceiling(ctx, r.params, resource)
		r.ceilings[resource] = ceiling
	}

	return ceiling
}

func (r *retrievalAccess) ShowsField(
	ctx context.Context,
	resource permission.Resource,
	field string,
) bool {
	return r.access.visible(resource, field, r.ceiling(ctx, resource))
}

func (r *retrievalAccess) ShowsRecordText(
	ctx context.Context,
	resource permission.Resource,
) bool {
	return r.access.recordTextVisible(resource, r.ceiling(ctx, resource))
}

type fieldGate struct {
	access   fieldAccess
	resource permission.Resource
	ceiling  permission.FieldSensitivity
	withheld []string
}

func (a fieldAccess) gate(
	ctx context.Context,
	params *serviceports.QueryToolParams,
	resource permission.Resource,
) *fieldGate {
	return a.gateAt(resource, a.ceiling(ctx, params, resource))
}

func (a fieldAccess) gateAt(
	resource permission.Resource,
	ceiling permission.FieldSensitivity,
) *fieldGate {
	if a.registry == nil {
		a.registry = permission.NewRegistry()
	}

	return &fieldGate{access: a, resource: resource, ceiling: ceiling}
}

func (g *fieldGate) shows(field string) bool {
	return g.access.visible(g.resource, field, g.ceiling)
}

func (g *fieldGate) show(field, name string) bool {
	if g.shows(field) {
		return true
	}
	if g.access.registry.GetFieldSensitivity(
		g.resource.String(),
		field,
	) != permission.SensitivityConfidential &&
		!slices.Contains(g.withheld, name) {
		g.withheld = append(g.withheld, name)
	}

	return false
}

func (g *fieldGate) Withheld() []string {
	if len(g.withheld) == 0 {
		return nil
	}

	return slices.Clone(g.withheld)
}

type gatedOutcome struct {
	searchOutcome

	Withheld []string `json:"withheldByAccess,omitempty"`

	tainted []agent.RecordRef
}

func gatedResult(outcome searchOutcome, gate *fieldGate) gatedOutcome {
	result := gatedOutcome{searchOutcome: outcome}
	if gate != nil {
		result.Withheld = gate.Withheld()
	}

	return result
}

func (o gatedOutcome) withTaint(refs []agent.RecordRef) gatedOutcome {
	o.tainted = refs

	return o
}

func (o gatedOutcome) TaintedRecords() []agent.RecordRef { return o.tainted }
