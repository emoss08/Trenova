package agentquerytoolservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
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
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	defaultSearchLimit  = 10
	maxSearchLimit      = 25
	maxShipmentComments = 20
	maxShipmentHolds    = 25
	paramDetail         = "detail"
)

type shipmentDetail string

const (
	shipmentDetailSummary shipmentDetail = "summary"
	shipmentDetailFull    shipmentDetail = "full"
)

type getShipmentTool struct {
	repo     shipmentports.ShipmentRepository
	comments shipmentports.ShipmentCommentRepository
	holds    shipmentports.ShipmentHoldRepository
	access   fieldAccess
}

func newGetShipmentTool(
	repo shipmentports.ShipmentRepository,
	comments shipmentports.ShipmentCommentRepository,
	holds shipmentports.ShipmentHoldRepository,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &getShipmentTool{
		repo:     repo,
		comments: comments,
		holds:    holds,
		access:   newFieldAccess(permissions),
	}
}

func (t *getShipmentTool) Name() string { return "shipment_get" }

func (t *getShipmentTool) Description() string {
	return "Read one shipment (load) by its id or PRO number. It shows the customer and " +
		"rating, each move's stops with their windows in the stop's local time, who drives " +
		"it and with what equipment, the holds on it now and its newest comments. Pass " +
		"detail full only " +
		"for a field the summary leaves out. Each hold carries the holdId " +
		"shipment_update_hold and shipment_release_hold take. Find it with shipment_list " +
		"when you have only a customer, a place or a BOL."
}

func (t *getShipmentTool) SearchTerms() []string {
	return []string{"assigned driver", "who is driving", "shipment details"}
}

func (t *getShipmentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			paramShipmentID: shipmentRefProperty(),
			paramProNumber:  proNumberProperty(),
			paramDetail: agenttoolschema.Enum(
				"How much to return. Omit it for the summary: the customer, the rating, "+
					"the moves with their stops, assignment and carrier, the commodities, "+
					"charges, holds and comments. full returns every stored field.",
				shipmentDetailLevels,
			),
		},
		"additionalProperties": false,
	}
}

func (t *getShipmentTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceShipment,
		reads:    agent.ExternalReadMarked,
		source:   agent.TaintSourceRecordNote,
		rationale: "Reads a shipment with its newest comments, some of which a driver, a " +
			"trading partner or another system outside the organization wrote; nothing " +
			"changes and nothing is sent.",
	})
}

func (t *getShipmentTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	detail, err := shipmentDetailOf(params.Params)
	if err != nil {
		return nil, err
	}

	tenant := tenantOf(params)
	entity, err := loadShipment(ctx, t.repo, tenant, params.Params)
	if err != nil {
		return nil, err
	}

	comments, err := t.recentComments(ctx, params, tenant, entity.ID)
	if err != nil {
		return nil, err
	}

	holds, err := t.activeHolds(ctx, params, tenant, entity.ID)
	if err != nil {
		return nil, err
	}

	activity := newShipmentActivity(comments, holds)
	people := t.access.redactor(ctx, params)

	if detail == shipmentDetailFull {
		return people.withhold(newShipmentView(entity, activity))
	}

	return people.annotate(summarizeShipment(&shipmentSummaryInput{
		entity:   entity,
		activity: activity,
		timezone: params.Timezone,
		people:   people,
	}))
}

func shipmentDetailOf(params map[string]any) (shipmentDetail, error) {
	value, err := validEnum(params, paramDetail, shipmentDetailLevels.AsStrings())
	if err != nil {
		return "", err
	}
	if value == "" {
		return shipmentDetailSummary, nil
	}

	return shipmentDetail(value), nil
}

func (t *getShipmentTool) activeHolds(
	ctx context.Context,
	params *agentports.QueryToolParams,
	tenant pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*shipment.ShipmentHold, error) {
	if t.holds == nil || !t.access.mayRead(ctx, params, permission.ResourceShipmentHold) {
		return nil, nil
	}

	page, err := t.holds.ListByShipmentID(ctx, &shipmentports.ListShipmentHoldsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: maxShipmentHolds},
		},
		ShipmentID: shipmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("read the shipment's holds: %w", err)
	}
	if page == nil {
		return nil, nil
	}

	active := make([]*shipment.ShipmentHold, 0, len(page.Items))
	for _, hold := range page.Items {
		if hold != nil && hold.ReleasedAt == nil {
			active = append(active, hold)
		}
	}

	return active, nil
}

func (t *getShipmentTool) recentComments(
	ctx context.Context,
	params *agentports.QueryToolParams,
	tenant pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*shipment.ShipmentComment, error) {
	if t.comments == nil ||
		!t.access.mayRead(ctx, params, permission.ResourceShipmentComment) {
		return nil, nil
	}

	page, err := t.comments.ListByShipmentID(ctx, &shipmentports.ListShipmentCommentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: maxShipmentComments},
		},
		Cursor:     pagination.CursorInfo{Limit: maxShipmentComments},
		ShipmentID: shipmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("read the shipment's comments: %w", err)
	}
	if page == nil {
		return nil, nil
	}

	return page.Items, nil
}

const (
	shipmentListName         = "shipment_list"
	defaultShipmentWordLimit = 10
)

type shipmentListTool struct {
	repo         shipmentports.ShipmentRepository
	parties      shipmentPartySources
	list         *listTool
	filterFields agenttoolschema.EnumSource[string]
}

func provideShipmentListTool(
	repo shipmentports.ShipmentRepository,
	customers customerports.CustomerRepository,
	locations locationports.LocationRepository,
	workers workerports.WorkerRepository,
	commodities commodityports.CommodityRepository,
	types shipmenttypeports.ShipmentTypeRepository,
) agentports.AgentQueryTool {
	return newShipmentListTool(repo, shipmentPartySources{
		customers:   customers,
		locations:   locations,
		workers:     workers,
		commodities: commodities,
		types:       types,
	})
}

func newShipmentListTool(
	repo shipmentports.ShipmentRepository,
	parties shipmentPartySources,
) *shipmentListTool {
	spec := shipmentListSpec(repo)
	list := buildListTool(&spec)
	fields := make([]string, 0, len(spec.fields))
	for _, name := range list.fields.Names() {
		if name != paramStatus {
			fields = append(fields, name)
		}
	}

	return &shipmentListTool{
		repo:         repo,
		parties:      parties,
		list:         list,
		filterFields: agenttoolschema.Derived(shipmentListName+".filterField", fields),
	}
}

func (t *shipmentListTool) Name() string { return shipmentListName }

func (t *shipmentListTool) Description() string {
	return "Find shipments (loads) by the words people use for them, by status, or by " +
		"filters on dates, charges and transfer status. Words match a pro number or BOL, a " +
		"customer's name or code, a stop's place or city, the driver or what it carries; " +
		"several words each narrow, so \"sunbelt chicago\" finds Sunbelt's loads through " +
		"Chicago. With no words and no filters it lists the newest. Each row has the first " +
		"pickup, the final stop and who drives each move; open one with shipment_get only " +
		"for more."
}

func (t *shipmentListTool) SearchTerms() []string {
	return []string{"pro number", "bol number", "reference number"}
}

func (t *shipmentListTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramQuery: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "Words to match: a pro number, BOL, customer, stop " +
					"place or city, driver or commodity. Omit to list by status and filters alone.",
			},
			paramStatus: map[string]any{
				toolschema.KeyType:  toolschema.TypeArray,
				toolschema.KeyItems: agenttoolschema.Enum("", listShipmentStatuses),
				toolschema.KeyDescription: "Only shipments in these statuses, all in one call; " +
					"omit for every status. Not picked up yet is New, PartiallyAssigned and " +
					"Assigned; on the road is InTransit and Delayed. A load the person calls " +
					"late or running behind is Delayed. There is no Delivered: a delivered load " +
					"is Completed, ReadyToInvoice or Invoiced.",
			},
			paramFilters:       t.filtersSchema(),
			paramSortBy:        t.list.sortSchema(),
			paramSortDirection: sortDirectionSchema(),
			paramLimit: toolschema.Clamped(map[string]any{
				toolschema.KeyType:    toolschema.TypeInteger,
				toolschema.KeyMinimum: 1,
				toolschema.KeyMaximum: maxListLimit,
				toolschema.KeyDescription: fmt.Sprintf("How many rows to return: %d when you "+
					"give words, %d otherwise, at most %d.",
					defaultShipmentWordLimit, defaultListLimit, maxListLimit),
			}),
			paramAfter: cursorPageSchema(defaultListLimit, maxListLimit)[paramAfter],
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *shipmentListTool) filtersSchema() map[string]any {
	schema := filtersSchema(t.filterFields, t.list.operators)
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
		"yesterday and tomorrow." + listDateRule

	return schema
}

func (t *shipmentListTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceShipment,
	})
}

func (t *shipmentListTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	statuses, err := shipmentStatusFilter(params.Params)
	if err != nil {
		return nil, err
	}

	query := optionalString(params.Params, "query")
	if query == "" {
		return t.list.Query(ctx, withStatusFilter(params, statuses))
	}

	return t.searchWords(ctx, params, query, statuses)
}

func withStatusFilter(
	params *agentports.QueryToolParams,
	statuses []shipment.Status,
) *agentports.QueryToolParams {
	listed := make(map[string]any, len(params.Params)+1)
	for key, value := range params.Params {
		if key != paramStatus {
			listed[key] = value
		}
	}
	if len(statuses) > 0 {
		filters, _ := listed[paramFilters].([]any)
		values := make([]any, 0, len(statuses))
		for _, status := range statuses {
			values = append(values, string(status))
		}
		listed[paramFilters] = append(slices.Clone(filters), map[string]any{
			paramField:    paramStatus,
			paramOperator: "in",
			paramValues:   values,
		})
	}

	copied := *params
	copied.Params = listed

	return &copied
}

type wordSearch struct {
	params   *agentports.QueryToolParams
	query    string
	statuses []shipment.Status
	window   cursorWindow
	filters  []domaintypes.FieldFilter
	sorting  []domaintypes.SortField
}

func (t *shipmentListTool) searchWords(
	ctx context.Context,
	params *agentports.QueryToolParams,
	query string,
	statuses []shipment.Status,
) (any, error) {
	criteria := filtercatalog.NewCriteria("shipments").At(clockFor(params))
	filters, err := t.list.buildFilters(params.Params, criteria)
	if err != nil {
		return nil, err
	}
	sorting, err := t.list.buildSort(params.Params)
	if err != nil {
		return nil, err
	}

	window, err := readCursorWindow(
		t.Name(),
		params.Params,
		defaultShipmentWordLimit,
		maxListLimit,
	)
	if err != nil {
		return nil, err
	}

	search := &wordSearch{
		params:   params,
		query:    query,
		statuses: statuses,
		window:   window,
		filters:  filters,
		sorting:  sorting,
	}
	found, err := t.search(ctx, search, criteria)
	if err != nil {
		return nil, err
	}
	if len(found.rows) > 0 || len(statuses) == 0 || !search.window.first() {
		return t.wordOutcome(ctx, search, found)
	}

	relaxed, err := t.search(ctx, &wordSearch{
		params:  params,
		query:   query,
		window:  search.window,
		filters: filters,
		sorting: sorting,
	}, filtercatalog.NewCriteria("shipments").At(clockFor(params)))
	if err != nil {
		return nil, err
	}
	if len(relaxed.rows) == 0 {
		return searchResult(found.criteria, found.rows, 0), nil
	}

	outcome := searchResult(relaxed.criteria, relaxed.rows, len(relaxed.rows))
	outcome.Note = fmt.Sprintf(
		"None of the shipments these words match is %s. These are the ones they match in any "+
			"status; read each row's status rather than searching status by status.",
		statusNames(statuses))

	return outcome, nil
}

func (t *shipmentListTool) wordOutcome(
	ctx context.Context,
	search *wordSearch,
	found *shipmentSearch,
) (any, error) {
	rows := found.rows
	full := len(rows) >= search.window.limit
	var sampled []string
	if full && search.window.first() {
		var err error
		if rows, sampled, err = t.withEachStatus(ctx, search, rows); err != nil {
			return nil, err
		}
	}

	outcome := searchResult(found.criteria, rows, len(rows))
	switch {
	case full && found.byParty:
		outcome.HasMore = true
		outcome.Note = fmt.Sprintf(
			"These are the first %d of more shipments the words match. If the one meant is "+
				"not here, search again with a status, a date or a pro number rather than "+
				"asking the person to choose from this page.", search.window.limit)
	case found.hasMore:
		outcome.cursorPaged(&search.window, true, found.nextCursor)
		if total, err := countShipments(ctx, t.repo, &pagination.QueryOptions{
			TenantInfo:   tenantOf(search.params),
			Query:        search.query,
			FieldFilters: search.filters,
		}, shipmentports.ShipmentOptions{Statuses: search.statuses}); err == nil &&
			total > len(rows) {
			outcome.Total = &total
		}
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
	rows []shipmentRow,
) ([]shipmentRow, []string, error) {
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
		one.window = firstCursorWindow(1)
		found, err := t.search(
			ctx,
			&one,
			filtercatalog.NewCriteria("shipments").At(clockFor(search.params)),
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
	rows       []shipmentRow
	byParty    bool
	hasMore    bool
	nextCursor string
}

func (t *shipmentListTool) search(
	ctx context.Context,
	search *wordSearch,
	criteria *filtercatalog.Criteria,
) (*shipmentSearch, error) {
	criteria.Text(search.query)
	criteria.Field(paramStatus, statusNames(search.statuses))

	result, err := t.repo.List(ctx, &shipmentports.ListShipmentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo:   tenantOf(search.params),
			Pagination:   pagination.Info{Limit: search.window.limit},
			Query:        search.query,
			FieldFilters: search.filters,
			Sort:         search.sorting,
		},
		Cursor: search.window.info,
		ShipmentOptions: shipmentports.ShipmentOptions{
			Statuses:        search.statuses,
			IncludeCustomer: true,
			IncludeRoute:    true,
		},
	})
	if err != nil {
		return nil, refuseStaleCursor(err)
	}
	next, err := result.NextCursor()
	if err != nil {
		return nil, fmt.Errorf("encode the next page's cursor: %w", err)
	}

	rows := make([]shipmentRow, 0, len(result.Items))
	for _, item := range result.Items {
		rows = append(rows, toShipmentRow(item))
	}

	direct := len(rows)
	if search.window.first() && len(rows) < search.window.limit {
		rows, err = t.addPartyMatches(ctx, &partySearch{
			params:   search.params,
			query:    search.query,
			statuses: search.statuses,
			limit:    search.window.limit,
			filters:  search.filters,
			sorting:  search.sorting,
			criteria: criteria,
		}, rows)
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
	}, nil
}

type partySearch struct {
	params   *agentports.QueryToolParams
	query    string
	statuses []shipment.Status
	limit    int
	filters  []domaintypes.FieldFilter
	sorting  []domaintypes.SortField
	criteria *filtercatalog.Criteria
}

func (t *shipmentListTool) addPartyMatches(
	ctx context.Context,
	search *partySearch,
	rows []shipmentRow,
) ([]shipmentRow, error) {
	tenant := tenantOf(search.params)
	parties, err := t.resolveParties(ctx, tenant, search.query)
	if err != nil || parties.empty() {
		return rows, err
	}

	return t.appendPartyRows(ctx, search, tenant, partyLookups(parties, search.criteria), rows)
}

func partyLookups(
	parties *shipmentParties,
	criteria *filtercatalog.Criteria,
) []shipmentports.ShipmentOptions {
	lookups := make([]shipmentports.ShipmentOptions, 0, 5)
	if parties.allFromDifferentWords() || parties.onlyPlacesFromDifferentWords() {
		places := partiesByTerm(parties.locations)
		combined := shipmentports.ShipmentOptions{
			CustomerIDs:     partyIDs(parties.customers),
			WorkerIDs:       partyIDs(parties.workers),
			CommodityIDs:    partyIDs(parties.commodities),
			ShipmentTypeIDs: partyIDs(parties.types),
		}
		if len(places) > 1 {
			combined.StopEachOf = make([][]pulid.ID, 0, len(places))
			for _, place := range places {
				combined.StopEachOf = append(combined.StopEachOf, partyIDs(place))
			}
		} else {
			combined.StopLocationIDs = partyIDs(parties.locations)
		}
		lookups = append(lookups, combined)
		if len(parties.customers) > 0 {
			criteria.Field("customer", partyNames(parties.customers))
		}
		for _, place := range places {
			criteria.Field("with a stop at", partyNames(place))
		}
		if len(parties.workers) > 0 {
			criteria.Field("driven by", partyNames(parties.workers))
		}
		if len(parties.commodities) > 0 {
			criteria.Field("carrying", partyNames(parties.commodities))
		}
		if len(parties.types) > 0 {
			criteria.Field("of type", partyNames(parties.types))
		}
	} else {
		if len(parties.customers) > 0 {
			lookups = append(
				lookups,
				shipmentports.ShipmentOptions{CustomerIDs: partyIDs(parties.customers)},
			)
			criteria.Field("or customer", partyNames(parties.customers))
		}
		if len(parties.locations) > 0 {
			lookups = append(lookups, shipmentports.ShipmentOptions{
				StopLocationIDs: partyIDs(parties.locations),
			})
			criteria.Field("or a stop at", partyNames(parties.locations))
		}
		if len(parties.workers) > 0 {
			lookups = append(
				lookups,
				shipmentports.ShipmentOptions{WorkerIDs: partyIDs(parties.workers)},
			)
			criteria.Field("or driven by", partyNames(parties.workers))
		}
		if len(parties.commodities) > 0 {
			lookups = append(lookups, shipmentports.ShipmentOptions{
				CommodityIDs: partyIDs(parties.commodities),
			})
			criteria.Field("or carrying", partyNames(parties.commodities))
		}
		if len(parties.types) > 0 {
			lookups = append(lookups, shipmentports.ShipmentOptions{
				ShipmentTypeIDs: partyIDs(parties.types),
			})
			criteria.Field("or of type", partyNames(parties.types))
		}
	}

	return lookups
}

func (t *shipmentListTool) appendPartyRows(
	ctx context.Context,
	search *partySearch,
	tenant pagination.TenantInfo,
	lookups []shipmentports.ShipmentOptions,
	rows []shipmentRow,
) ([]shipmentRow, error) {
	seen := make(map[string]struct{}, search.limit)
	for idx := range rows {
		seen[rows[idx].ID] = struct{}{}
	}
	for _, options := range lookups {
		if len(rows) >= search.limit {
			break
		}
		options.Statuses = search.statuses
		options.IncludeCustomer = true
		options.IncludeRoute = true
		result, listErr := t.repo.List(ctx, &shipmentports.ListShipmentsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo:   tenant,
				Pagination:   pagination.Info{Limit: search.limit},
				FieldFilters: search.filters,
				Sort:         search.sorting,
			},
			ShipmentOptions: options,
		})
		if listErr != nil {
			return nil, listErr
		}
		for _, item := range result.Items {
			if len(rows) >= search.limit {
				break
			}
			if _, dup := seen[item.ID.String()]; dup {
				continue
			}
			seen[item.ID.String()] = struct{}{}
			rows = append(rows, toShipmentRow(item))
		}
	}

	return rows, nil
}

// shipmentStatusFilter refuses a status outside the set and names the set.
//
// A refusal that lists the alternatives is a correction the model can act on;
// an empty result is one it reports as fact. Several statuses are taken at
// once: asked for the load "that hasn't picked up yet", a model searched New,
// Assigned and PartiallyAssigned in three calls.
func shipmentStatusFilter(params map[string]any) ([]shipment.Status, error) {
	named := optionalStrings(params, paramStatus)
	if len(named) == 0 {
		return nil, nil
	}

	statuses := make([]shipment.Status, 0, len(named))
	for _, status := range named {
		if !slices.Contains(shipmentStatuses, status) {
			return nil, fmt.Errorf(
				"parameter %q does not accept %q; a delivered load is Completed. Use one of: %s",
				paramStatus, status, strings.Join(shipmentStatuses, ", "),
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
