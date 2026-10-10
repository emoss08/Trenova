package agentquerytoolservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
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
	repo     repositories.ShipmentRepository
	comments repositories.ShipmentCommentRepository
	holds    repositories.ShipmentHoldRepository
	access   fieldAccess
}

func newGetShipmentTool(
	repo repositories.ShipmentRepository,
	comments repositories.ShipmentCommentRepository,
	holds repositories.ShipmentHoldRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getShipmentTool{
		repo:     repo,
		comments: comments,
		holds:    holds,
		access:   newFieldAccess(permissions),
	}
}

func (t *getShipmentTool) Name() string { return "get_shipment" }

func (t *getShipmentTool) Description() string {
	return "Retrieve one shipment (load) by its id, with its stops, the driver and equipment " +
		"assigned to each move, the holds on it now and its newest comments. It returns a " +
		"summary with each stop's window in the stop's local time; pass detail full only " +
		"when you need a field the summary leaves out. Each hold carries the holdId " +
		"update_shipment_hold and release_shipment_hold take. Use search_shipments first " +
		"when you only have a pro number or customer name."
}

func (t *getShipmentTool) SearchTerms() []string {
	return []string{"assigned driver", "who is driving", "shipment details"}
}

func (t *getShipmentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentId": agenttoolschema.OfResource(map[string]any{
				"type": "string",
				"description": "The shipment's id, from search_shipments or list_shipments, " +
					"the page you are on, or this run's subject.",
			}, permission.ResourceShipment),
			paramDetail: agenttoolschema.Enum(
				"How much to return. Omit it for the summary: the customer, the rating, "+
					"the moves with their stops, assignment and carrier, the commodities, "+
					"charges, holds and comments. full returns every stored field.",
				shipmentDetailLevels,
			),
		},
		"required":             []string{"shipmentId"},
		"additionalProperties": false,
	}
}

func (t *getShipmentTool) Policy() serviceports.ToolPolicy {
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
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	shipmentID, err := requirePulid(params.Params, "shipmentId")
	if err != nil {
		return nil, err
	}

	detail, err := shipmentDetailOf(params.Params)
	if err != nil {
		return nil, err
	}

	tenant := pagination.TenantInfo{
		OrgID:  params.OrganizationID,
		BuID:   params.BusinessUnitID,
		UserID: params.Actor.UserID,
	}
	entity, err := t.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenant,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
			IncludeCustomer:       true,
		},
	})
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
	params *serviceports.QueryToolParams,
	tenant pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*shipment.ShipmentHold, error) {
	if t.holds == nil || !t.access.mayRead(ctx, params, permission.ResourceShipmentHold) {
		return nil, nil
	}

	page, err := t.holds.ListByShipmentID(ctx, &repositories.ListShipmentHoldsRequest{
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
	params *serviceports.QueryToolParams,
	tenant pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*shipment.ShipmentComment, error) {
	if t.comments == nil ||
		!t.access.mayRead(ctx, params, permission.ResourceShipmentComment) {
		return nil, nil
	}

	page, err := t.comments.ListByShipmentID(ctx, &repositories.ListShipmentCommentsRequest{
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

type searchShipmentsTool struct {
	repo    repositories.ShipmentRepository
	parties shipmentPartySources
}

func provideSearchShipmentsTool(
	repo repositories.ShipmentRepository,
	customers repositories.CustomerRepository,
	locations repositories.LocationRepository,
	workers repositories.WorkerRepository,
	commodities repositories.CommodityRepository,
	types repositories.ShipmentTypeRepository,
) serviceports.AgentQueryTool {
	return newSearchShipmentsTool(repo, shipmentPartySources{
		customers:   customers,
		locations:   locations,
		workers:     workers,
		commodities: commodities,
		types:       types,
	})
}

func newSearchShipmentsTool(
	repo repositories.ShipmentRepository,
	parties shipmentPartySources,
) serviceports.AgentQueryTool {
	return &searchShipmentsTool{repo: repo, parties: parties}
}

func (t *searchShipmentsTool) Name() string { return "search_shipments" }

func (t *searchShipmentsTool) Description() string {
	return "Find shipments by the words people use for them: a pro number or BOL, a " +
		"customer's name or code, a stop's place or city, the driver, or what it carries. Words naming a customer, a place or a " +
		"driver narrow to loads matching each, so \"sunbelt chicago\" finds Sunbelt's " +
		"loads through Chicago and \"jane chicago\" Jane's load there. " +
		"Narrow by status too. Call it with no query to see the most recent shipments. " +
		"Each row carries its first pickup, final stop and who is driving each move " +
		"(or that it needs a driver); open one with get_shipment only for more than that."
}

func (t *searchShipmentsTool) SearchTerms() []string {
	return []string{
		"pro number", "bol number", "reference number", "customer's load", "load going to",
		"load from", "shipments for a customer", "shipments through a city",
	}
}

func (t *searchShipmentsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type": "string",
				"description": "Optional words to match: a pro number, BOL, customer, or " +
					"stop location or city. Omit to list shipments unfiltered.",
			},
			paramStatus: map[string]any{
				"type":  "array",
				"items": agenttoolschema.Enum("", listShipmentStatuses),
				"description": "Only shipments in these statuses, all in one call; omit for " +
					"every status. Not picked up yet is New, PartiallyAssigned and Assigned; " +
					"on the road is InTransit and Delayed. A load the person calls late or " +
					"running behind is Delayed. There is no Delivered: a delivered load is " +
					"Completed, ReadyToInvoice or Invoiced.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "How many results to return, at most 25",
			},
		},
		"additionalProperties": false,
	}
}

func (t *searchShipmentsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceShipment,
	})
}

func (t *searchShipmentsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	// Optional for the same reason as on search_worker: the repository applies a
	// text search only when the term is non-empty, so requiring one here refused
	// a call the layer beneath would have served.
	query := optionalString(params.Params, "query")

	// A model asked for "all shipments" will happily request a limit of 10000,
	// which would blow the context window and the query budget at once.
	limit := optionalInt(params.Params, "limit", defaultSearchLimit)
	if limit <= 0 || limit > maxSearchLimit {
		limit = defaultSearchLimit
	}

	// Unvalidated, this passed whatever the model sent straight to the
	// repository, which matched nothing and returned an empty page the model
	// reported as "there are no shipments in that state". The schema used to
	// name "Delivered" here, which is not one of the statuses, so the tool was
	// instructing its caller to produce exactly that.
	statuses, err := shipmentStatusFilter(params.Params)
	if err != nil {
		return nil, err
	}

	found, err := t.search(ctx, params, query, statuses, limit)
	if err != nil {
		return nil, err
	}
	criteria, rows := found.criteria, found.rows
	if len(rows) > 0 || len(statuses) == 0 || query == "" {
		full := len(rows) >= limit
		var sampled []string
		if full {
			if rows, sampled, err = t.withEachStatus(ctx, params, query, statuses, rows); err != nil {
				return nil, err
			}
		}
		outcome := searchResult(criteria, rows, len(rows))
		if full && found.byParty {
			outcome.HasMore = true
			outcome.Note = fmt.Sprintf(
				"These are the first %d of more shipments the words match. If the one meant is "+
					"not here, search again with a status, a date or a pro number rather than "+
					"asking the person to choose from this page.", limit)
		}
		if len(sampled) > 0 {
			outcome.HasMore = true
			outcome.Note = strings.TrimSpace(outcome.Note + " " + fmt.Sprintf(
				"The newest matches filled the page, so the newest match in each other status "+
					"asked for is added after them (%s): read each row's status, as the one "+
					"meant may be one of those.", stringutils.JoinAlternatives(sampled)))
		}
		if full && !found.byParty {
			outcome.HasMore = true
			if total, countErr := countShipments(ctx, t.repo, &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{
					OrgID:  params.OrganizationID,
					BuID:   params.BusinessUnitID,
					UserID: params.Actor.UserID,
				},
				Query: query,
			}, repositories.ShipmentOptions{Statuses: statuses}); countErr == nil &&
				total > len(rows) {
				outcome.Total = &total
			}
		}

		return outcome, nil
	}

	relaxed, err := t.search(ctx, params, query, nil, limit)
	if err != nil {
		return nil, err
	}
	if len(relaxed.rows) == 0 {
		return searchResult(criteria, rows, 0), nil
	}

	outcome := searchResult(relaxed.criteria, relaxed.rows, len(relaxed.rows))
	outcome.Note = fmt.Sprintf(
		"None of the shipments these words match is %s. These are the ones they match in any "+
			"status; read each row's status rather than searching status by status.",
		statusNames(statuses))

	return outcome, nil
}

// withEachStatus adds, to a page the newest matches filled, the newest match
// in each status asked for that the page does not show. Asked for "the reefer
// to LA running behind" across five statuses, Haiku was shown ten new loads
// to Los Angeles and never the delayed one, and asked which load was meant.
func (t *searchShipmentsTool) withEachStatus(
	ctx context.Context,
	params *serviceports.QueryToolParams,
	query string,
	statuses []shipment.Status,
	rows []shipmentRow,
) ([]shipmentRow, []string, error) {
	if len(statuses) < 2 {
		return rows, nil, nil
	}

	shown := make(map[string]struct{}, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for idx := range rows {
		shown[rows[idx].Status] = struct{}{}
		seen[rows[idx].ID] = struct{}{}
	}

	sampled := make([]string, 0, len(statuses))
	for _, status := range statuses {
		if _, ok := shown[string(status)]; ok {
			continue
		}
		found, err := t.search(ctx, params, query, []shipment.Status{status}, 1)
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
	criteria *filtercatalog.Criteria
	rows     []shipmentRow
	byParty  bool
}

func (t *searchShipmentsTool) search(
	ctx context.Context,
	params *serviceports.QueryToolParams,
	query string,
	statuses []shipment.Status,
	limit int,
) (*shipmentSearch, error) {
	criteria := filtercatalog.NewCriteria("shipments").At(clockFor(params))
	criteria.Text(query)
	criteria.Field("status", statusNames(statuses))

	result, err := t.repo.List(ctx, &repositories.ListShipmentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID:  params.OrganizationID,
				BuID:   params.BusinessUnitID,
				UserID: params.Actor.UserID,
			},
			Pagination: pagination.Info{Limit: limit},
			Query:      query,
		},
		ShipmentOptions: repositories.ShipmentOptions{
			Statuses:        statuses,
			IncludeCustomer: true,
			IncludeRoute:    true,
		},
	})
	if err != nil {
		return nil, err
	}

	rows := make([]shipmentRow, 0, len(result.Items))
	for _, item := range result.Items {
		rows = append(rows, toShipmentRow(item))
	}

	direct := len(rows)
	if query != "" && len(rows) < limit {
		rows, err = t.addPartyMatches(ctx, &partySearch{
			params:   params,
			query:    query,
			statuses: statuses,
			limit:    limit,
			criteria: criteria,
		}, rows)
		if err != nil {
			return nil, err
		}
	}

	return &shipmentSearch{criteria: criteria, rows: rows, byParty: len(rows) > direct}, nil
}

type partySearch struct {
	params   *serviceports.QueryToolParams
	query    string
	statuses []shipment.Status
	limit    int
	criteria *filtercatalog.Criteria
}

func (t *searchShipmentsTool) addPartyMatches(
	ctx context.Context,
	search *partySearch,
	rows []shipmentRow,
) ([]shipmentRow, error) {
	tenant := pagination.TenantInfo{
		OrgID:  search.params.OrganizationID,
		BuID:   search.params.BusinessUnitID,
		UserID: search.params.Actor.UserID,
	}

	parties, err := t.resolveParties(ctx, tenant, search.query)
	if err != nil || parties.empty() {
		return rows, err
	}

	lookups := make([]repositories.ShipmentOptions, 0, 5)
	if parties.allFromDifferentWords() || parties.onlyPlacesFromDifferentWords() {
		places := partiesByTerm(parties.locations)
		combined := repositories.ShipmentOptions{
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
			search.criteria.Field("customer", partyNames(parties.customers))
		}
		for _, place := range places {
			search.criteria.Field("with a stop at", partyNames(place))
		}
		if len(parties.workers) > 0 {
			search.criteria.Field("driven by", partyNames(parties.workers))
		}
		if len(parties.commodities) > 0 {
			search.criteria.Field("carrying", partyNames(parties.commodities))
		}
		if len(parties.types) > 0 {
			search.criteria.Field("of type", partyNames(parties.types))
		}
	} else {
		if len(parties.customers) > 0 {
			lookups = append(lookups, repositories.ShipmentOptions{CustomerIDs: partyIDs(parties.customers)})
			search.criteria.Field("or customer", partyNames(parties.customers))
		}
		if len(parties.locations) > 0 {
			lookups = append(lookups, repositories.ShipmentOptions{
				StopLocationIDs: partyIDs(parties.locations),
			})
			search.criteria.Field("or a stop at", partyNames(parties.locations))
		}
		if len(parties.workers) > 0 {
			lookups = append(lookups, repositories.ShipmentOptions{WorkerIDs: partyIDs(parties.workers)})
			search.criteria.Field("or driven by", partyNames(parties.workers))
		}
		if len(parties.commodities) > 0 {
			lookups = append(lookups, repositories.ShipmentOptions{
				CommodityIDs: partyIDs(parties.commodities),
			})
			search.criteria.Field("or carrying", partyNames(parties.commodities))
		}
		if len(parties.types) > 0 {
			lookups = append(lookups, repositories.ShipmentOptions{
				ShipmentTypeIDs: partyIDs(parties.types),
			})
			search.criteria.Field("or of type", partyNames(parties.types))
		}
	}

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
		result, listErr := t.repo.List(ctx, &repositories.ListShipmentsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: tenant,
				Pagination: pagination.Info{Limit: search.limit},
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
