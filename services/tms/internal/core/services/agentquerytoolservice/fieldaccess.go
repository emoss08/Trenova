package agentquerytoolservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/fieldsensitivity"
	"github.com/emoss08/trenova/shared/jsonutils"
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
	return fieldsensitivity.PersonCeiling(
		ctx, a.permissions, params.Actor.UserID, params.OrganizationID, resource,
	)
}

// visible reports whether one field of a resource may be shown under a
// ceiling. Confidential is refused outright.
func (a fieldAccess) visible(
	resource permission.Resource,
	field string,
	ceiling permission.FieldSensitivity,
) bool {
	return fieldsensitivity.Visible(a.registry, resource, field, ceiling)
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
	resolve  func() permission.FieldSensitivity
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

func (a fieldAccess) deferredGate(
	ctx context.Context,
	params *serviceports.QueryToolParams,
	resource permission.Resource,
) *fieldGate {
	gate := a.gateAt(resource, "")
	gate.resolve = func() permission.FieldSensitivity {
		return a.ceiling(ctx, params, resource)
	}

	return gate
}

func (g *fieldGate) level() permission.FieldSensitivity {
	if g.resolve != nil {
		g.ceiling = g.resolve()
		g.resolve = nil
	}

	return g.ceiling
}

func (g *fieldGate) shows(field string) bool {
	return g.access.visible(g.resource, field, g.level())
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

func gatedResult(outcome *searchOutcome, gate *fieldGate) *gatedOutcome {
	result := &gatedOutcome{searchOutcome: *outcome}
	if gate != nil {
		result.Withheld = gate.Withheld()
	}

	return result
}

func (o *gatedOutcome) withTaint(refs []agent.RecordRef) *gatedOutcome {
	o.tainted = refs

	return o
}

func (o *gatedOutcome) TaintedRecords() []agent.RecordRef { return o.tainted }

type nestedRule struct {
	idPrefix string
	resource permission.Resource
	label    string
}

func (r nestedRule) reduces() bool { return r.resource == "" }

func (r nestedRule) fieldName(field string) string { return r.label + "." + field }

var nestedRules = []nestedRule{
	{idPrefix: tenant.UserIDPrefix},
	{idPrefix: tenant.OrganizationIDPrefix},
	{idPrefix: tenant.BusinessUnitIDPrefix},
	{
		idPrefix: agent.SubjectWorker.IDPrefix(),
		resource: permission.ResourceWorker,
		label:    permission.ResourceWorker.String(),
	},
	{
		idPrefix: shipment.CarrierAssignmentIDPrefix,
		resource: permission.ResourceShipmentMove,
		label:    "shipmentMove",
	},
}

var nestedRulesByPrefix = func() map[string]nestedRule {
	out := make(map[string]nestedRule, len(nestedRules))
	for _, rule := range nestedRules {
		out[rule.idPrefix] = rule
	}

	return out
}()

const (
	nestedIDKey   = "id"
	nestedNameKey = "name"
)

var workerRule = nestedRulesByPrefix[agent.SubjectWorker.IDPrefix()]

func nestedRuleOf(record map[string]any) (nestedRule, bool) {
	id, ok := record[nestedIDKey].(string)
	if !ok {
		return nestedRule{}, false
	}
	rule, found := nestedRulesByPrefix[pulid.ID(id).Prefix()]

	return rule, found
}

type nestedRedactor struct {
	gates map[permission.Resource]*fieldGate
}

func (a fieldAccess) redactor(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) *nestedRedactor {
	gates := make(map[permission.Resource]*fieldGate, len(nestedRules))
	for _, rule := range nestedRules {
		if !rule.reduces() {
			gates[rule.resource] = a.deferredGate(ctx, params, rule.resource)
		}
	}

	return &nestedRedactor{gates: gates}
}

func (r *nestedRedactor) workerName(w *worker.Worker) string {
	if w == nil {
		return ""
	}
	gate := r.gates[workerRule.resource]
	first := gate.show("firstName", workerRule.fieldName("firstName"))
	last := gate.show("lastName", workerRule.fieldName("lastName"))
	if !first || !last {
		return ""
	}

	return workerName(w)
}

func (r *nestedRedactor) withhold(result any) (*gatedDocument, error) {
	return r.document(result, true)
}

func (r *nestedRedactor) annotate(result any) (*gatedDocument, error) {
	return r.document(result, false)
}

func (r *nestedRedactor) document(result any, walk bool) (*gatedDocument, error) {
	tree, err := jsonutils.ToJSON(result)
	if err != nil {
		return nil, fmt.Errorf("encode the result to withhold what access does not reach: %w", err)
	}

	if walk {
		r.walk(tree)
	}
	if withheld := r.withheld(); len(withheld) > 0 {
		tree[withheldByAccessKey] = withheld
	}

	gated := &gatedDocument{tree: tree}
	if carrier, ok := result.(agent.TaintCarrier); ok {
		gated.tainted = carrier.TaintedRecords()
	}

	return gated, nil
}

func (r *nestedRedactor) withheld() []string {
	size := 0
	for _, gate := range r.gates {
		size += len(gate.withheld)
	}
	if size == 0 {
		return nil
	}

	out := make([]string, 0, size)
	for _, gate := range r.gates {
		out = append(out, gate.withheld...)
	}
	slices.Sort(out)

	return slices.Compact(out)
}

func (r *nestedRedactor) walk(node any) {
	switch typed := node.(type) {
	case map[string]any:
		if rule, ok := nestedRuleOf(typed); ok {
			r.apply(rule, typed)
			return
		}
		for _, child := range typed {
			r.walk(child)
		}
	case []any:
		for _, child := range typed {
			r.walk(child)
		}
	}
}

func (r *nestedRedactor) apply(rule nestedRule, record map[string]any) {
	if rule.reduces() {
		for key := range record {
			if key != nestedIDKey && key != nestedNameKey {
				delete(record, key)
			}
		}
		return
	}

	r.gateFields(rule, record)
}

func (r *nestedRedactor) gateFields(rule nestedRule, node any) {
	gate := r.gates[rule.resource]
	switch typed := node.(type) {
	case map[string]any:
		for key, child := range typed {
			if !gate.show(key, rule.fieldName(key)) {
				delete(typed, key)
				continue
			}
			r.gateChild(rule, child)
		}
	case []any:
		for _, child := range typed {
			r.gateChild(rule, child)
		}
	}
}

func (r *nestedRedactor) gateChild(rule nestedRule, child any) {
	if record, ok := child.(map[string]any); ok {
		if nested, found := nestedRuleOf(record); found {
			r.apply(nested, record)
			return
		}
	}

	r.gateFields(rule, child)
}

const withheldByAccessKey = "withheldByAccess"

type gatedDocument struct {
	tree    map[string]any
	tainted []agent.RecordRef
}

func (d *gatedDocument) MarshalJSON() ([]byte, error) {
	return sonic.Marshal(d.tree)
}

func (d *gatedDocument) TaintedRecords() []agent.RecordRef { return d.tainted }
