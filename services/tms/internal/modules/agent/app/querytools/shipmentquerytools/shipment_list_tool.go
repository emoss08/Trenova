package shipmentquerytools

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/shipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	DefaultSearchLimit = 10
	maxSearchLimit     = 25
)

func (t *shipmentListTool) partySources(parties *shipmentParties) []partySource {
	sources := make([]partySource, 0, 5)
	if t.parties.Customers != nil {
		sources = append(sources, partySource{into: &parties.customers, lookup: t.customerParties})
	}
	if t.parties.Locations != nil {
		sources = append(sources, partySource{into: &parties.locations, lookup: t.locationParties})
	}
	if t.parties.workers != nil {
		sources = append(sources, partySource{into: &parties.workers, lookup: t.workerParties})
	}
	if t.parties.Commodities != nil {
		sources = append(sources, partySource{
			into:   &parties.commodities,
			lookup: t.commodityParties,
		})
	}
	if t.parties.Types != nil {
		sources = append(sources, partySource{into: &parties.types, lookup: t.typeParties})
	}
	if t.parties.Orders != nil {
		sources = append(sources, partySource{into: &parties.orders, lookup: t.orderParties})
	}

	return sources
}

func (t *shipmentListTool) resolveParties(
	ctx context.Context,
	tenant pagination.TenantInfo,
	query string,
) (*shipmentParties, error) {
	parties := &shipmentParties{}
	if t.parties.none() {
		return parties, nil
	}

	sources := t.partySources(parties)
	for _, term := range partyTerms(query) {
		filter := &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: partyMatchLimit},
			Query:      term,
		}
		named := false
		for _, source := range sources {
			found, err := source.lookup(ctx, filter, term)
			if err != nil {
				return nil, err
			}
			*source.into = append(*source.into, found...)
			named = named || len(found) > 0
		}
		if named {
			parties.terms = append(parties.terms, term)
		}
		if term == query && named {
			break
		}
	}

	return parties, nil
}

func (t *shipmentListTool) customerParties(
	ctx context.Context,
	filter *pagination.QueryOptions,
	term string,
) ([]partyMatch, error) {
	found, err := t.parties.Customers.List(ctx, &customerports.ListCustomerRequest{Filter: filter})
	if err != nil {
		return nil, err
	}

	matches := make([]partyMatch, 0, len(found.Items))
	for _, item := range found.Items {
		matches = append(matches, partyMatch{term: term, id: item.ID, name: item.Name})
	}

	return matches, nil
}

func (t *shipmentListTool) locationParties(
	ctx context.Context,
	filter *pagination.QueryOptions,
	term string,
) ([]partyMatch, error) {
	found, err := t.parties.Locations.List(ctx, &locationports.ListLocationRequest{Filter: filter})
	if err != nil {
		return nil, err
	}

	matches := make([]partyMatch, 0, len(found.Items))
	for _, item := range found.Items {
		matches = append(matches, partyMatch{
			term: term, id: item.ID, name: item.Name + " (" + item.City + ")",
		})
	}

	return matches, nil
}

func (t *shipmentListTool) workerParties(
	ctx context.Context,
	filter *pagination.QueryOptions,
	term string,
) ([]partyMatch, error) {
	found, err := t.parties.workers.List(ctx, &workerports.ListWorkersRequest{Filter: filter})
	if err != nil {
		return nil, err
	}

	matches := make([]partyMatch, 0, len(found.Items))
	for _, item := range found.Items {
		matches = append(
			matches,
			partyMatch{term: term, id: item.ID, name: querytool.WorkerName(item)},
		)
	}

	return matches, nil
}

func (t *shipmentListTool) commodityParties(
	ctx context.Context,
	filter *pagination.QueryOptions,
	term string,
) ([]partyMatch, error) {
	found, err := t.parties.Commodities.List(
		ctx,
		&commodityports.ListCommodityRequest{Filter: filter},
	)
	if err != nil {
		return nil, err
	}

	matches := make([]partyMatch, 0, len(found.Items))
	for _, item := range found.Items {
		matches = append(matches, partyMatch{term: term, id: item.ID, name: item.Name})
	}

	return matches, nil
}

func (t *shipmentListTool) typeParties(
	ctx context.Context,
	filter *pagination.QueryOptions,
	term string,
) ([]partyMatch, error) {
	found, err := t.parties.Types.List(
		ctx,
		&shipmenttypeports.ListShipmentTypesRequest{Filter: filter},
	)
	if err != nil {
		return nil, err
	}

	matches := make([]partyMatch, 0, len(found.Items))
	for _, item := range found.Items {
		matches = append(matches, partyMatch{term: term, id: item.ID, name: item.Code})
	}

	return matches, nil
}

func (t *shipmentListTool) orderParties(
	ctx context.Context,
	filter *pagination.QueryOptions,
	term string,
) ([]partyMatch, error) {
	found, err := t.parties.Orders.MatchOrderReferences(
		ctx,
		&shipmentports.MatchOrderReferencesRequest{
			Query: filter.Query,
			Limit: filter.Pagination.Limit,
		},
	)
	if err != nil {
		return nil, err
	}

	matches := make([]partyMatch, 0, len(found))
	for idx := range found {
		matches = append(matches, partyMatch{
			term: term,
			id:   found[idx].OrderID,
			name: orderReferenceName(&found[idx]),
		})
	}

	return matches, nil
}

type Detail string

const (
	DetailSummary Detail = "summary"
	DetailFull    Detail = "full"
)

func AccessHolder(params *agenttoolports.QueryToolParams) string {
	if params.Actor != nil && params.Actor.IsAgent() {
		return "this agent's unattended access"
	}

	return "the access of the person you are working for"
}

const (
	defaultShipmentWordLimit = 10
	paramOnHold              = "onHold"
	paramLate                = "late"
	paramHoldReasonCodes     = "holdReasonCodes"
	maxHoldReasonCodes       = 10
)

type shipmentListTool struct {
	repo         shipmentports.ShipmentRepository
	parties      PartySources
	holds        shipmentports.ActiveHoldReader
	access       fieldgate.Access
	list         *listtool.Tool
	filterFields agenttoolschema.EnumSource[string]
	columns      agenttoolschema.EnumSource[string]
}

func ProvideShipmentListTool(
	repo shipmentports.ShipmentRepository,
	customers customerports.CustomerRepository,
	locations locationports.LocationRepository,
	workers workerports.WorkerRepository,
	commodities commodityports.CommodityRepository,
	types shipmenttypeports.ShipmentTypeRepository,
	orders shipmentports.OrderReferenceMatcher,
	holds shipmentports.ActiveHoldReader,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return NewShipmentListTool(repo, PartySources{
		Customers:   customers,
		Locations:   locations,
		workers:     workers,
		Commodities: commodities,
		Types:       types,
		Orders:      orders,
	}).WithHolds(holds, permissions)
}

func NewShipmentListTool(
	repo shipmentports.ShipmentRepository,
	parties PartySources,
) *shipmentListTool {
	spec := ListSpec(repo)
	list := listtool.Build(&spec)
	fields := make([]string, 0, len(spec.Fields))
	for _, name := range list.Fields.Names() {
		if name != querytool.ParamStatus {
			fields = append(fields, name)
		}
	}

	return &shipmentListTool{
		repo:         repo,
		parties:      parties,
		list:         list,
		filterFields: agenttoolschema.Derived(querytool.ShipmentListName+".filterField", fields),
		columns:      list.Columns,
	}
}

func (t *shipmentListTool) WithHolds(
	holds shipmentports.ActiveHoldReader,
	permissions permissionports.PermissionEngine,
) *shipmentListTool {
	t.holds = holds
	t.access = fieldgate.NewAccess(permissions)

	return t
}

func (t *shipmentListTool) Name() string { return querytool.ShipmentListName }

func (t *shipmentListTool) Description() string {
	return "Find shipments (loads) by the words people use for them, by status, by holds, or " +
		"by filters on dates, charges and transfer status. Words match a pro number, BOL, " +
		"the customer's PO or reference number, a customer's name or code, a stop's place or " +
		"city, the driver or what it carries; several words each narrow, so \"sunbelt " +
		"chicago\" finds Sunbelt's loads through Chicago, and a word that names several " +
		"things narrows to any of them. With no words and no filters it lists the newest. " +
		"Each row has the first pickup, the final stop, who drives each move, what it carries " +
		"and the holds in force, each with its note, when it started and the holdId " +
		"shipment_release_hold takes, so onHold lists held loads and why in one call, and late " +
		"lists the loads behind schedule by their stops, whatever their status. For " +
		"more, pass every load you need to shipment_get at once."
}

func (t *shipmentListTool) SearchTerms() []string {
	return []string{"pro number", "bol number", "po number", "reference number", "held loads"}
}

func (t *shipmentListTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			querytool.ParamQuery: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "Words to match: a pro number, BOL, the customer's " +
					"PO or reference number (the number alone, such as 77Q-4182, not \"PO " +
					"77Q-4182\"), customer, stop place or city, driver or commodity. Omit to " +
					"list by status, holds and filters alone.",
			},
			querytool.ParamStatus: map[string]any{
				toolschema.KeyType:  toolschema.TypeArray,
				toolschema.KeyItems: agenttoolschema.Enum("", listShipmentStatuses),
				toolschema.KeyDescription: "Only shipments in these statuses, all in one call; " +
					"omit for every status. Not picked up yet is New, PartiallyAssigned and " +
					"Assigned; on the road is InTransit and Delayed. For a load the person " +
					"calls late or running behind, pass late rather than Delayed, which only " +
					"some late loads carry. There is no Delivered: a delivered load is " +
					"Completed, ReadyToInvoice or Invoiced.",
			},
			paramLate: map[string]any{
				toolschema.KeyType: toolschema.TypeBoolean,
				toolschema.KeyDescription: "true lists only shipments running late now: a stop " +
					"not yet reached whose window has closed, the loads behind schedule. " +
					"Delayed is a status a person sets, and a late load may not carry it; " +
					"this reads the stops. Combine with status, words and filters.",
			},
			paramOnHold: map[string]any{
				toolschema.KeyType: toolschema.TypeBoolean,
				toolschema.KeyDescription: "true lists only shipments with a hold in force, " +
					"false only those with none; omit for both.",
			},
			paramHoldReasonCodes: map[string]any{
				toolschema.KeyType:     toolschema.TypeArray,
				toolschema.KeyItems:    map[string]any{toolschema.KeyType: toolschema.TypeString},
				toolschema.KeyMaxItems: maxHoldReasonCodes,
				toolschema.KeyDescription: "Only shipments held for one of these hold reason " +
					"codes, such as CREDIT_HOLD, from a row's reasonCode or " +
					"masterdata_list_hold_reasons; implies onHold.",
			},
			listtool.ParamFilters:       t.filtersSchema(),
			listtool.ParamSortBy:        t.list.SortSchema(),
			listtool.ParamSortDirection: listtool.SortDirectionSchema(),
			querytool.ParamLimit: toolschema.Clamped(map[string]any{
				toolschema.KeyType:    toolschema.TypeInteger,
				toolschema.KeyMinimum: 1,
				toolschema.KeyMaximum: listtool.MaxLimit,
				toolschema.KeyDescription: fmt.Sprintf("How many rows to return: %d when you "+
					"give words, %d otherwise, at most %d.",
					defaultShipmentWordLimit, listtool.DefaultLimit, listtool.MaxLimit),
			}),
			querytool.ParamAfter:           querytool.CursorPageSchema(listtool.DefaultLimit, listtool.MaxLimit)[querytool.ParamAfter],
			agenttoolports.ListFieldsParam: listtool.FieldsSchema(t.columns),
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *shipmentListTool) filtersSchema() map[string]any {
	schema := listtool.FiltersSchema(t.filterFields, t.list.Operators)
	schema[toolschema.KeyDescription] = "Conditions beside status, combined with AND: billingTransferStatus " +
		"(" + strings.Join(
		billingTransferStates,
		"|",
	) + "; isnull means never transferred), " +
		"freightTerms (" + strings.Join(
		freightTerms,
		"|",
	) + "), customerId (from " +
		"customer_list, never guessed), proNumber, bol, the dates actualShipDate, " +
		"actualDeliveryDate, billedAt, canceledAt and createdAt, and the numbers " +
		"totalChargeAmount and weight. Give value for one comparison, values for in and " +
		"notin, days for nextndays and lastndays, and nothing for isnull, isnotnull, today, " +
		"yesterday and tomorrow." + listtool.DateRule

	return schema
}

func (t *shipmentListTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), querytool.ReadSpec{
		Resource: permission.ResourceShipment,
	})
}

func (t *shipmentListTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	statuses, err := StatusFilter(params.Params)
	if err != nil {
		return nil, err
	}

	held, err := shipmentHoldFilterOf(params.Params)
	if err != nil {
		return nil, err
	}

	var lateAt int64
	if querytool.OptionalBool(params.Params, paramLate) {
		lateAt = querytool.ClockFor(params).Now
	}

	readsHolds := t.readsHolds(ctx, params)
	if held.set() && !readsHolds {
		return nil, &graphqlreadports.ReadRefusedError{
			Resource: permission.ResourceShipmentHold,
			Message:  "listing shipments by their holds needs read access to shipment holds",
		}
	}

	var result any
	query := querytool.OptionalString(params.Params, querytool.ParamQuery)
	if query == "" && !held.set() && lateAt == 0 {
		result, err = t.list.Query(ctx, withStatusFilter(params, statuses))
	} else {
		result, err = t.searchWords(ctx, &wordSearch{
			params:   params,
			query:    query,
			statuses: statuses,
			held:     held,
			lateAt:   lateAt,
		})
	}
	if err != nil {
		return nil, err
	}

	return t.withActiveHolds(ctx, params, result, readsHolds)
}

func (t *shipmentListTool) readsHolds(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) bool {
	return t.holds != nil && t.access.MayRead(ctx, params, permission.ResourceShipmentHold)
}

type shipmentHoldFilter struct {
	active *bool
	codes  []string
}

func shipmentHoldFilterOf(params map[string]any) (shipmentHoldFilter, error) {
	filter := shipmentHoldFilter{
		active: querytool.OptionalBoolPointer(params, paramOnHold),
		codes:  sliceutils.DedupeStrings(querytool.OptionalStrings(params, paramHoldReasonCodes)),
	}
	if len(filter.codes) > maxHoldReasonCodes {
		return shipmentHoldFilter{}, fmt.Errorf(
			"parameter %q takes at most %d codes", paramHoldReasonCodes, maxHoldReasonCodes)
	}
	if len(filter.codes) > 0 && filter.active != nil && !*filter.active {
		return shipmentHoldFilter{}, fmt.Errorf(
			"parameter %q lists held shipments, so it cannot go with %q false; leave %q out",
			paramHoldReasonCodes, paramOnHold, paramOnHold)
	}

	return filter, nil
}

func (f shipmentHoldFilter) set() bool {
	return f.active != nil || len(f.codes) > 0
}

func (f shipmentHoldFilter) apply(options *shipmentports.ShipmentOptions) {
	options.ActiveHold = f.active
	options.HoldReasonCodes = f.codes
}

func (f shipmentHoldFilter) describe(criteria *filtercatalog.Criteria) {
	switch {
	case len(f.codes) > 0:
		criteria.Field("with a hold in force for", stringutils.JoinAlternatives(f.codes))
	case f.active != nil && *f.active:
		criteria.Field("with a hold in force", "for any reason")
	case f.active != nil:
		criteria.Field("with no hold", "in force")
	}
}

type shipmentRowHold struct {
	HoldID     string                 `json:"holdId"`
	Reason     string                 `json:"reason"`
	ReasonCode string                 `json:"reasonCode,omitempty"`
	Severity   string                 `json:"severity,omitempty"`
	Blocks     string                 `json:"blocks,omitempty"`
	Note       string                 `json:"note,omitempty"`
	StartedAt  querytool.OptionalDate `json:"startedAt"`
}

func rowHoldOf(hold *shipmenthold.ShipmentHold) shipmentRowHold {
	row := shipmentRowHold{
		HoldID:     hold.ID.String(),
		Reason:     string(hold.Type),
		ReasonCode: hold.ReasonCode,
		Severity:   string(hold.Severity),
		Note:       HoldNote(hold),
		StartedAt:  querytool.RecordedDate(hold.StartedAt),
	}
	switch {
	case hold.HoldReason != nil && strings.TrimSpace(hold.HoldReason.Label) != "":
		row.Reason = strings.TrimSpace(hold.HoldReason.Label)
	case hold.ReasonCode != "":
		row.Reason = hold.ReasonCode
	}

	blocks := make([]string, 0, 3)
	if hold.BlocksDispatch {
		blocks = append(blocks, "dispatch")
	}
	if hold.BlocksDelivery {
		blocks = append(blocks, "delivery")
	}
	if hold.BlocksBilling {
		blocks = append(blocks, "billing")
	}
	row.Blocks = strings.Join(blocks, ", ")

	return row
}

func (t *shipmentListTool) withActiveHolds(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	result any,
	readsHolds bool,
) (any, error) {
	outcome, ok := result.(querytool.SearchOutcome)
	if !ok || outcome.Count == 0 {
		return result, nil
	}
	if !readsHolds {
		if t.holds != nil {
			outcome.Note = strings.TrimSpace(outcome.Note + " " + fmt.Sprintf(
				"Holds are left out of these rows: %s does not include reading shipment "+
					"holds, so a row without activeHolds may still be held.",
				AccessHolder(params)))
		}
		return outcome, nil
	}

	ids := shipmentRowIDs(outcome.Items)
	if len(ids) == 0 {
		return outcome, nil
	}
	holds, err := t.holds.ActiveHolds(ctx, &shipmentports.ActiveHoldsRequest{ShipmentIDs: ids})
	if err != nil {
		return nil, fmt.Errorf("read the holds on these shipments: %w", err)
	}

	byShipment := make(map[string][]shipmentRowHold, len(holds))
	for _, hold := range holds {
		if hold != nil {
			key := hold.ShipmentID.String()
			byShipment[key] = append(byShipment[key], rowHoldOf(hold))
		}
	}
	outcome.Items = withRowHolds(outcome.Items, byShipment)

	return outcome, nil
}

func shipmentRowIDs(items any) []pulid.ID {
	var rows []Row
	switch typed := items.(type) {
	case []Row:
		rows = typed
	case []any:
		rows = make([]Row, 0, len(typed))
		for _, item := range typed {
			if row, ok := item.(Row); ok {
				rows = append(rows, row)
			}
		}
	}

	ids := make([]pulid.ID, 0, len(rows))
	for idx := range rows {
		if id, err := pulid.Parse(rows[idx].ID); err == nil && id.IsNotNil() {
			ids = append(ids, id)
		}
	}

	return ids
}

func withRowHolds(items any, holds map[string][]shipmentRowHold) any {
	switch typed := items.(type) {
	case []Row:
		for idx := range typed {
			typed[idx].ActiveHolds = holds[typed[idx].ID]
		}
		return typed
	case []any:
		for idx, item := range typed {
			if row, ok := item.(Row); ok {
				row.ActiveHolds = holds[row.ID]
				typed[idx] = row
			}
		}
		return typed
	default:
		return items
	}
}

func withStatusFilter(
	params *agenttoolports.QueryToolParams,
	statuses []shipment.Status,
) *agenttoolports.QueryToolParams {
	listed := make(map[string]any, len(params.Params)+1)
	for key, value := range params.Params {
		if key != querytool.ParamStatus {
			listed[key] = value
		}
	}
	if len(statuses) > 0 {
		filters, _ := listed[listtool.ParamFilters].([]any)
		values := make([]any, 0, len(statuses))
		for _, status := range statuses {
			values = append(values, string(status))
		}
		listed[listtool.ParamFilters] = append(slices.Clone(filters), map[string]any{
			querytool.ParamField:   querytool.ParamStatus,
			listtool.ParamOperator: "in",
			listtool.ParamValues:   values,
		})
	}

	copied := *params
	copied.Params = listed

	return &copied
}

type wordSearch struct {
	params   *agenttoolports.QueryToolParams
	query    string
	statuses []shipment.Status
	held     shipmentHoldFilter
	// lateAt keeps the shipments running late at that instant, when set.
	lateAt  int64
	window  querytool.CursorWindow
	filters []domaintypes.FieldFilter
	sorting []domaintypes.SortField
}

func (s *wordSearch) options() shipmentports.ShipmentOptions {
	options := shipmentports.ShipmentOptions{
		Statuses:           s.statuses,
		IncludeCustomer:    true,
		IncludeRoute:       true,
		IncludeCommodities: true,
	}
	s.held.apply(&options)
	options.LateAt = s.lateAt

	return options
}

func (t *shipmentListTool) searchWords(
	ctx context.Context,
	search *wordSearch,
) (any, error) {
	params := search.params
	criteria := filtercatalog.NewCriteria("shipments").At(querytool.ClockFor(params))
	filters, err := t.list.BuildFilters(params.Params, criteria)
	if err != nil {
		return nil, err
	}
	sorting, err := t.list.BuildSort(params.Params)
	if err != nil {
		return nil, err
	}

	defaultLimit := defaultShipmentWordLimit
	if search.query == "" {
		defaultLimit = listtool.DefaultLimit
	}
	window, err := querytool.ReadCursorWindow(
		t.Name(),
		params.Params,
		defaultLimit,
		listtool.MaxLimit,
	)
	if err != nil {
		return nil, err
	}

	search.window = window
	search.filters = filters
	search.sorting = sorting
	found, err := t.search(ctx, search, criteria)
	if err != nil {
		return nil, err
	}
	if len(found.rows) > 0 || len(search.statuses) == 0 || !search.window.First() ||
		search.query == "" {
		return t.wordOutcome(ctx, search, found)
	}

	anyStatus := *search
	anyStatus.statuses = nil
	relaxed, err := t.search(
		ctx,
		&anyStatus,
		filtercatalog.NewCriteria("shipments").At(querytool.ClockFor(params)),
	)
	if err != nil {
		return nil, err
	}
	if len(relaxed.rows) == 0 {
		return querytool.SearchResult(found.criteria, found.rows, 0), nil
	}

	outcome := querytool.SearchResult(relaxed.criteria, relaxed.rows, len(relaxed.rows))
	outcome.Note = fmt.Sprintf(
		"None of the shipments these words match is %s. These are the ones they match in any "+
			"status; read each row's status rather than searching status by status.",
		statusNames(search.statuses))

	return outcome, nil
}

func (t *shipmentListTool) wordOutcome(
	ctx context.Context,
	search *wordSearch,
	found *shipmentSearch,
) (any, error) {
	rows := found.rows
	full := len(rows) >= search.window.Limit
	var sampled []string
	if full && search.window.First() {
		var err error
		if rows, sampled, err = t.withEachStatus(ctx, search, rows); err != nil {
			return nil, err
		}
	}

	outcome := querytool.SearchResult(found.criteria, rows, len(rows))
	switch {
	case full && found.byParty:
		outcome.HasMore = true
		outcome.Note = fmt.Sprintf(
			"These are the first %d of more shipments the words match. If the one meant is "+
				"not here, search again with a status, a date or a pro number rather than "+
				"asking the person to choose from this page.", search.window.Limit)
	case found.hasMore:
		outcome.CursorPaged(&search.window, true, found.nextCursor)
		options := search.options()
		options.IncludeCustomer = false
		options.IncludeRoute = false
		options.IncludeCommodities = false
		if total, exact, err := countShipments(ctx, t.repo, &pagination.QueryOptions{
			TenantInfo:   querytool.TenantOf(search.params),
			Query:        search.query,
			FieldFilters: search.filters,
		}, options); err == nil &&
			total > len(rows) {
			outcome.Total = &total
			outcome.TotalEstimated = !exact
		}
	}
	if !found.byParty && len(sampled) == 0 {
		outcome.Resume = &querytool.PageResume{Scope: search.window.Scope, Cursors: found.cursors}
	}
	if len(sampled) > 0 {
		outcome.HasMore = true
		outcome.Note = strings.TrimSpace(outcome.Note + " " + fmt.Sprintf(
			"The newest matches filled the page, so the newest match in each other status "+
				"asked for is added after them (%s): read each row's status, as the one "+
				"meant may be one of those.", stringutils.JoinAlternatives(sampled)))
	}

	return outcome, nil
}

func (t *shipmentListTool) withEachStatus(
	ctx context.Context,
	search *wordSearch,
	rows []Row,
) ([]Row, []string, error) {
	if len(search.statuses) < 2 {
		return rows, nil, nil
	}

	shown := make(map[string]struct{}, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for idx := range rows {
		shown[rows[idx].Status] = struct{}{}
		seen[rows[idx].ID] = struct{}{}
	}

	sampled := make([]string, 0, len(search.statuses))
	for _, status := range search.statuses {
		if _, ok := shown[string(status)]; ok {
			continue
		}
		one := *search
		one.statuses = []shipment.Status{status}
		one.window = querytool.FirstCursorWindow(1)
		found, err := t.search(
			ctx,
			&one,
			filtercatalog.NewCriteria("shipments").At(querytool.ClockFor(search.params)),
		)
		if err != nil {
			return nil, nil, err
		}
		for idx := range found.rows {
			if _, dup := seen[found.rows[idx].ID]; dup {
				continue
			}
			seen[found.rows[idx].ID] = struct{}{}
			rows = append(rows, found.rows[idx])
			sampled = append(sampled, string(status))
		}
	}

	return rows, sampled, nil
}

type shipmentSearch struct {
	criteria   *filtercatalog.Criteria
	rows       []Row
	byParty    bool
	hasMore    bool
	nextCursor string
	cursors    []string
}

func (t *shipmentListTool) search(
	ctx context.Context,
	search *wordSearch,
	criteria *filtercatalog.Criteria,
) (*shipmentSearch, error) {
	criteria.Text(search.query)
	criteria.Field(querytool.ParamStatus, statusNames(search.statuses))
	search.held.describe(criteria)
	if search.lateAt > 0 {
		criteria.Field(paramLate, "running late: a stop past its window with no arrival")
	}

	result, err := t.repo.List(ctx, &shipmentports.ListShipmentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo:   querytool.TenantOf(search.params),
			Pagination:   pagination.Info{Limit: search.window.Limit},
			Query:        search.query,
			FieldFilters: search.filters,
			Sort:         search.sorting,
		},
		Cursor:          search.window.Info,
		ShipmentOptions: search.options(),
	})
	if err != nil {
		return nil, querytool.RefuseStaleCursor(err)
	}
	next, err := result.NextCursor()
	if err != nil {
		return nil, fmt.Errorf("encode the next page's cursor: %w", err)
	}

	rows := make([]Row, 0, len(result.Items))
	for _, item := range result.Items {
		rows = append(rows, ToRow(item))
	}

	direct := len(rows)
	if search.query != "" && search.window.First() && len(rows) < search.window.Limit {
		rows, err = t.addPartyMatches(ctx, search, criteria, rows)
		if err != nil {
			return nil, err
		}
	}

	return &shipmentSearch{
		criteria:   criteria,
		rows:       rows,
		byParty:    len(rows) > direct,
		hasMore:    result.HasNextPage,
		nextCursor: next,
		cursors:    result.EdgeCursors,
	}, nil
}

func (t *shipmentListTool) addPartyMatches(
	ctx context.Context,
	search *wordSearch,
	criteria *filtercatalog.Criteria,
	rows []Row,
) ([]Row, error) {
	tenant := querytool.TenantOf(search.params)
	parties, err := t.resolveParties(ctx, tenant, search.query)
	if err != nil || parties.empty() {
		return rows, err
	}

	options := search.options()
	options.PartyGroups = partyGroups(parties, criteria)
	if len(options.PartyGroups) == 0 {
		return rows, nil
	}

	result, err := t.repo.List(ctx, &shipmentports.ListShipmentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo:   tenant,
			Pagination:   pagination.Info{Limit: search.window.Limit},
			FieldFilters: search.filters,
			Sort:         search.sorting,
		},
		ShipmentOptions: options,
	})
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, search.window.Limit)
	for idx := range rows {
		seen[rows[idx].ID] = struct{}{}
	}
	for _, item := range result.Items {
		if len(rows) >= search.window.Limit {
			break
		}
		if _, dup := seen[item.ID.String()]; dup {
			continue
		}
		seen[item.ID.String()] = struct{}{}
		rows = append(rows, ToRow(item))
	}

	return rows, nil
}

// StatusFilter refuses a status outside the set and names the set.
//
// A refusal that lists the alternatives is a correction the model can act on;
// an empty result is one it reports as fact. Several statuses are taken at
// once: asked for the load "that hasn't picked up yet", a model searched New,
// Assigned and PartiallyAssigned in three calls.
func StatusFilter(params map[string]any) ([]shipment.Status, error) {
	named := querytool.OptionalStrings(params, querytool.ParamStatus)
	if len(named) == 0 {
		return nil, nil
	}

	statuses := make([]shipment.Status, 0, len(named))
	for _, status := range named {
		if !slices.Contains(shipmentStatuses, status) {
			return nil, fmt.Errorf(
				"parameter %q does not accept %q; a delivered load is Completed. Use one of: %s",
				querytool.ParamStatus, status, strings.Join(shipmentStatuses, ", "),
			)
		}
		if !slices.Contains(statuses, shipment.Status(status)) {
			statuses = append(statuses, shipment.Status(status))
		}
	}

	return statuses, nil
}

func statusNames(statuses []shipment.Status) string {
	names := make([]string, 0, len(statuses))
	for _, status := range statuses {
		names = append(names, string(status))
	}

	return stringutils.JoinAlternatives(names)
}
