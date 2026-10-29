package fieldgate

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

// Access answers which of a resource's fields a person may be shown,
// by the same sensitivity model the report compiler applies, so the worker
// a model reads through worker_get is the worker the person could open.
//
// Confidential fields are never sent to a model, whatever the person's
// role: a date of birth or a licence number in a chat transcript is a
// leak with nobody to blame. Restricted fields follow the person's role
// ceiling; Internal ones are shown to anyone who may read the record.
type Access struct {
	Permissions permissionports.PermissionEngine
	registry    *permissioncatalog.Registry
	threads     conversationports.ThreadOwnerRepository
}

func NewAccess(permissions permissionports.PermissionEngine) Access {
	return Access{Permissions: permissions, registry: permissioncatalog.NewRegistry()}
}

func (a Access) WithThreads(threads conversationports.ThreadOwnerRepository) Access {
	a.threads = threads

	return a
}

// ceiling is the most sensitive tier the actor may be shown on a resource,
// worked out by fieldsensitivity.AgentCeiling, which the GraphQL read path
// applies too, so a tool and an operation never disagree on it.
func (a Access) ceiling(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	resource permission.Resource,
) permission.FieldSensitivity {
	return fieldsensitivity.AgentCeiling(
		ctx, a.Permissions, params.Actor, params.DataAccessCeiling, resource,
	)
}

// groups is the protected field groups a tool may show computed figures of:
// those the actor holds, where the agent's data access reaches them, by the
// same rule the GraphQL read path applies.
func (a Access) Groups(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) protectedfield.GroupSet {
	held := fieldsensitivity.VisibleGroups(
		ctx,
		func(ctx context.Context, resource permission.Resource) (bool, error) {
			return a.MayRead(ctx, params, resource), nil
		},
	)

	return fieldsensitivity.AgentVisibleGroups(
		ctx,
		held,
		func(ctx context.Context, resource permission.Resource) permission.FieldSensitivity {
			return a.ceiling(ctx, params, resource)
		},
	)
}

// visible reports whether one field of a resource may be shown under a
// ceiling. Confidential is refused outright.
func (a Access) visible(
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
func (a Access) MayRead(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	resource permission.Resource,
) bool {
	actor := params.Actor
	if actor == nil || a.Permissions == nil {
		return false
	}

	result, err := a.Permissions.Check(ctx, &permissionports.PermissionCheckRequest{
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

func (a Access) MayReadRecord(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	resource permission.Resource,
	recordID string,
) bool {
	actor := params.Actor
	if actor == nil || a.Permissions == nil || resource == "" {
		return false
	}

	req := &permissionports.PermissionCheckRequest{
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

	result, err := a.Permissions.Check(ctx, req)

	return err == nil && result != nil && result.Allowed
}

func PersonOf(actor *permissionports.RequestActor) pulid.ID {
	if !actor.IsUser() {
		return pulid.Nil
	}

	return actor.UserID
}

func (a Access) ReadableDocuments(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	docs []*document.Document,
	mayReadRecord func(permission.Resource, string) bool,
) (map[pulid.ID]bool, error) {
	person := PersonOf(params.Actor)
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

func (a Access) conversationOwners(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
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

	owners, err := a.threads.ThreadOwners(ctx, conversationports.ThreadOwnersRequest{
		TenantInfo: querytool.TenantOf(params),
		ThreadIDs:  ids,
	})
	if err != nil {
		return nil, fmt.Errorf("read who owns the conversations documents are attached to: %w", err)
	}

	return owners, nil
}

func (a Access) recordTextVisible(
	resource permission.Resource,
	ceiling permission.FieldSensitivity,
) bool {
	definition, ok := a.registry.Get(resource.String())
	if !ok || definition.DefaultSensitivity == permission.SensitivityConfidential {
		return false
	}

	return ceiling.CanAccess(definition.DefaultSensitivity)
}

func (a Access) ForRetrieval(params *agenttoolports.QueryToolParams) *retrievalAccess {
	return &retrievalAccess{
		access:    a,
		params:    params,
		resources: make(map[permission.Resource]bool, 4),
		records:   make(map[string]bool, 16),
		ceilings:  make(map[permission.Resource]permission.FieldSensitivity, 4),
	}
}

type retrievalAccess struct {
	access    Access
	params    *agenttoolports.QueryToolParams
	resources map[permission.Resource]bool
	records   map[string]bool
	ceilings  map[permission.Resource]permission.FieldSensitivity
}

var _ retrievalports.RetrievalAccess = (*retrievalAccess)(nil)

func (r *retrievalAccess) MayReadResource(
	ctx context.Context,
	resource permission.Resource,
) bool {
	allowed, seen := r.resources[resource]
	if !seen {
		allowed = r.access.MayRead(ctx, r.params, resource)
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
		allowed = r.access.MayReadRecord(ctx, r.params, resource, recordID)
		r.records[key] = allowed
	}

	return allowed
}

func (r *retrievalAccess) ReadableDocuments(
	ctx context.Context,
	docs []*document.Document,
) (map[pulid.ID]bool, error) {
	return r.access.ReadableDocuments(ctx, r.params, docs,
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

type Gate struct {
	access   Access
	resource permission.Resource
	ceiling  permission.FieldSensitivity
	agent    permission.FieldSensitivity
	resolve  func() permission.FieldSensitivity
	withheld []string
}

func (a Access) Gate(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	resource permission.Resource,
) *Gate {
	gate := a.gateAt(resource, a.ceiling(ctx, params, resource))
	gate.agent = agentLimit(params)

	return gate
}

func agentLimit(params *agenttoolports.QueryToolParams) permission.FieldSensitivity {
	granted := fieldsensitivity.AgentDataAccess(params.DataAccessCeiling)
	if granted == "" && (params.Actor == nil || !params.Actor.IsUser()) {
		return permission.SensitivityInternal
	}

	return granted
}

func (a Access) gateAt(
	resource permission.Resource,
	ceiling permission.FieldSensitivity,
) *Gate {
	if a.registry == nil {
		a.registry = permissioncatalog.NewRegistry()
	}

	return &Gate{access: a, resource: resource, ceiling: ceiling}
}

func (a Access) deferredGate(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	resource permission.Resource,
) *Gate {
	gate := a.gateAt(resource, "")
	gate.agent = agentLimit(params)
	gate.resolve = func() permission.FieldSensitivity {
		return a.ceiling(ctx, params, resource)
	}

	return gate
}

func (g *Gate) level() permission.FieldSensitivity {
	if g.resolve != nil {
		g.ceiling = g.resolve()
		g.resolve = nil
	}

	return g.ceiling
}

func (g *Gate) Shows(field string) bool {
	return g.access.visible(g.resource, field, g.level())
}

func (g *Gate) Show(field, name string) bool {
	if g.Shows(field) {
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

func (g *Gate) HeldByAgent(field string) bool {
	return g.agent != "" &&
		!g.agent.CanAccess(g.access.registry.GetFieldSensitivity(g.resource.String(), field))
}

func (g *Gate) Withheld() []string {
	if len(g.withheld) == 0 {
		return nil
	}

	return slices.Clone(g.withheld)
}

// WithheldNote says what a withheld field means for the next call: no other
// lookup will show it either, so fetching the rows one by one only spends
// steps on the same gap.
const WithheldNote = "The fields in withheldByAccess are withheld at your access level " +
	"on every lookup, a single record's included. Do not look them up again row by row; " +
	"say they are withheld."

type Outcome struct {
	querytool.SearchOutcome

	Withheld     []string `json:"withheldByAccess,omitempty"`
	WithheldNote string   `json:"withheldNote,omitempty"`

	tainted []agentrecord.RecordRef
}

func Result(outcome *querytool.SearchOutcome, gate *Gate) *Outcome {
	result := &Outcome{SearchOutcome: *outcome}
	if gate != nil {
		result.Withheld = gate.Withheld()
	}
	if len(result.Withheld) > 0 {
		result.WithheldNote = WithheldNote
	}

	return result
}

// CutPage cuts the page and keeps what the gate withheld. Every marked record
// stays marked: a row the cut left out was still read.
func (o *Outcome) CutPage(rows int) (any, bool) {
	cut, ok := o.SearchOutcome.Cut(rows)
	if !ok {
		return nil, false
	}
	result := *o
	result.SearchOutcome = cut

	return &result, true
}

func (o *Outcome) WithTaint(refs []agentrecord.RecordRef) *Outcome {
	o.tainted = refs

	return o
}

func (o *Outcome) TaintedRecords() []agentrecord.RecordRef { return o.tainted }

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

type NestedRedactor struct {
	gates map[permission.Resource]*Gate
}

func (a Access) Redactor(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) *NestedRedactor {
	gates := make(map[permission.Resource]*Gate, len(nestedRules))
	for _, rule := range nestedRules {
		if !rule.reduces() {
			gates[rule.resource] = a.deferredGate(ctx, params, rule.resource)
		}
	}

	return &NestedRedactor{gates: gates}
}

func (r *NestedRedactor) WorkerName(w *worker.Worker) string {
	if w == nil {
		return ""
	}
	gate := r.gates[workerRule.resource]
	first := gate.Show("firstName", workerRule.fieldName("firstName"))
	last := gate.Show("lastName", workerRule.fieldName("lastName"))
	if !first || !last {
		return ""
	}

	return querytool.WorkerName(w)
}

func (r *NestedRedactor) Withhold(result any) (*gatedDocument, error) {
	return r.document(result, true)
}

func (r *NestedRedactor) Annotate(result any) (*gatedDocument, error) {
	return r.document(result, false)
}

func (r *NestedRedactor) document(result any, walk bool) (*gatedDocument, error) {
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

func (r *NestedRedactor) withheld() []string {
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

func (r *NestedRedactor) walk(node any) {
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

func (r *NestedRedactor) apply(rule nestedRule, record map[string]any) {
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

func (r *NestedRedactor) gateFields(rule nestedRule, node any) {
	gate := r.gates[rule.resource]
	switch typed := node.(type) {
	case map[string]any:
		for key, child := range typed {
			if !gate.Show(key, rule.fieldName(key)) {
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

func (r *NestedRedactor) gateChild(rule nestedRule, child any) {
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
	tainted []agentrecord.RecordRef
}

func (d *gatedDocument) MarshalJSON() ([]byte, error) {
	return sonic.Marshal(d.tree)
}

func (d *gatedDocument) TaintedRecords() []agentrecord.RecordRef { return d.tainted }
