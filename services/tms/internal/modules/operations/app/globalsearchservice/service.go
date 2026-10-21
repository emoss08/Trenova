package globalsearchservice

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/searchfilter"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/types/search"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Config       *config.Config
	SearchRepo   customerports.SearchRepository
	ShipmentRead shipmentports.ShipmentSearchCardReader
	Permissions  permissionports.PermissionEngine
	States       usstateports.UsStateRepository `optional:"true"`
	Visibility   kernelports.RecordVisibility
	Balances     kernelports.CustomerOpenBalanceReader `optional:"true"`
	RecordLabels captureports.CaptureRecordFinder      `optional:"true"`
	Views        permissionports.ReferenceViews
}

type Service struct {
	logger       *zap.Logger
	config       *config.SearchConfig
	searchRepo   customerports.SearchRepository
	shipmentRead shipmentports.ShipmentSearchCardReader
	permissions  permissionports.PermissionEngine
	visibility   kernelports.RecordVisibility
	states       usstateports.UsStateRepository
	balances     kernelports.CustomerOpenBalanceReader
	recordLabels captureports.CaptureRecordFinder
	views        permissionports.ReferenceViews
}

var _ kernelports.GlobalSearchService = (*Service)(nil)

type entitySearchDefinition struct {
	entityType search.EntityType
	label      string
	index      string
	resource   permission.Resource
	attributes []string
}

const (
	maxSearchLimit = 25
	fieldStatus    = "status"
	fieldCity      = "city"
)

func New(p Params) kernelports.GlobalSearchService {
	return &Service{
		logger:       p.Logger.Named("service.global-search"),
		config:       p.Config.GetSearchConfig(),
		searchRepo:   p.SearchRepo,
		shipmentRead: p.ShipmentRead,
		permissions:  p.Permissions,
		visibility:   p.Visibility,
		states:       p.States,
		balances:     p.Balances,
		recordLabels: p.RecordLabels,
		views:        p.Views,
	}
}

type indexResult struct {
	definition entitySearchDefinition
	hits       []*kernelports.GlobalSearchHit
	documents  []map[string]any
	total      int
}

func (s *Service) Search(
	ctx context.Context,
	req *kernelports.GlobalSearchRequest,
) (*kernelports.GlobalSearchResult, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" && len(req.EntityTypes) != 1 {
		return &kernelports.GlobalSearchResult{
			Query:  "",
			Groups: []*kernelports.GlobalSearchGroup{},
		}, nil
	}
	if !s.searchRepo.Enabled() {
		return &kernelports.GlobalSearchResult{
			Query:  query,
			Groups: []*kernelports.GlobalSearchGroup{},
		}, nil
	}

	access, err := s.resolveAccess(ctx, req)
	if err != nil {
		return nil, err
	}

	limit := limitForRequest(req.Limit, s.config.GetDefaultLimit())
	results := s.searchIndexes(ctx, req, access.definitions, query, limit)
	s.restrictToVisible(ctx, req, access, results)

	enricher := &hitEnricher{
		service: s,
		req:     req,
		access:  access,
		limit:   limit,
		results: results,
	}
	enricher.run(ctx)

	groups := make([]*kernelports.GlobalSearchGroup, 0, len(results))
	for _, result := range results {
		if result == nil || len(result.hits) == 0 {
			continue
		}
		groups = append(groups, &kernelports.GlobalSearchGroup{
			EntityType: result.definition.entityType,
			Label:      result.definition.label,
			Total:      max(result.total, len(result.hits)),
			Hits:       result.hits,
		})
	}

	return &kernelports.GlobalSearchResult{
		Query:  query,
		Groups: groups,
	}, nil
}

func (s *Service) searchIndexes(
	ctx context.Context,
	req *kernelports.GlobalSearchRequest,
	definitions []entitySearchDefinition,
	query string,
	limit int,
) []*indexResult {
	results := make([]*indexResult, len(definitions))
	filter := tenantFilter(req)

	searchPool := pool.New().WithContext(ctx).WithMaxGoroutines(max(1, len(definitions)))
	for idx, definition := range definitions {
		results[idx] = &indexResult{definition: definition}
		if definition.index == "" {
			continue
		}

		searchPool.Go(func(c context.Context) error {
			page, searchErr := s.searchRepo.SearchPage(c, customerports.SearchRequest{
				Index:      definition.index,
				Query:      query,
				Limit:      limit,
				Filter:     filter,
				Attributes: definition.attributes,
			})
			if searchErr != nil {
				s.logger.Warn(
					"global search index query failed",
					zap.String("index", definition.index),
					zap.String("entityType", string(definition.entityType)),
					zap.Error(searchErr),
				)
				return nil
			}
			if page == nil {
				return nil
			}

			results[idx].documents = page.Documents
			results[idx].hits = s.mapHits(definition, page.Documents)
			results[idx].total = page.EstimatedTotal - (len(page.Documents) - len(results[idx].hits))
			return nil
		})
	}
	_ = searchPool.Wait()

	return results
}

type searchAccess struct {
	definitions []entitySearchDefinition
	readable    map[permission.Resource]bool
	narrowed    map[permission.Resource]bool
	linked      permissionports.LinkedFields
}

func (a *searchAccess) canRead(resource permission.Resource) bool {
	return a.readable[resource]
}

// resolveAccess checks, in one batch, every index the request may search and
// every kind of record a hit may describe in its context line. A hit never
// names a record the person could not open themselves.
func (s *Service) resolveAccess(
	ctx context.Context,
	req *kernelports.GlobalSearchRequest,
) (*searchAccess, error) {
	definitions := s.definitions()
	if len(req.EntityTypes) > 0 {
		filtered := make([]entitySearchDefinition, 0, len(definitions))
		for _, definition := range definitions {
			if slices.Contains(req.EntityTypes, definition.entityType) {
				filtered = append(filtered, definition)
			}
		}
		definitions = filtered
	}

	resources := searchResources

	checks := make([]permissionports.ResourceOperationCheck, 0, len(resources))
	for _, resource := range resources {
		checks = append(checks, permissionports.ResourceOperationCheck{
			Resource:  resource.String(),
			Operation: permission.OpRead,
		})
	}

	result, err := s.permissions.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		PrincipalType:  req.Principal.Type,
		PrincipalID:    req.Principal.ID,
		UserID:         req.Principal.UserID,
		APIKeyID:       req.Principal.APIKeyID,
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		Checks:         checks,
	})
	if err != nil {
		return nil, fmt.Errorf("check search permissions: %w", err)
	}

	readable := make(map[permission.Resource]bool, len(resources))
	narrowed := make(map[permission.Resource]bool, len(resources))
	for idx, resource := range resources {
		if idx >= len(result.Results) {
			continue
		}
		readable[resource] = result.Results[idx].Allowed
		narrowed[resource] = result.Results[idx].DataScope.IsNarrow()
	}

	allowed := make([]entitySearchDefinition, 0, len(definitions))
	for _, definition := range definitions {
		if readable[definition.resource] {
			allowed = append(allowed, definition)
		}
	}

	access := &searchAccess{definitions: allowed, readable: readable, narrowed: narrowed}
	access.linked = s.linkedFields(req, readable)

	return access, nil
}

// searchResources is every kind the batch asks about: the indexes, the
// records a hit may describe in its context line, and every kind its
// metadata is classified under.
var searchResources = func() []permission.Resource {
	resources := []permission.Resource{
		permission.ResourceShipment,
		permission.ResourceCustomer,
		permission.ResourceWorker,
		permission.ResourceDocument,
		permission.ResourceAccountsReceivable,
		permission.ResourceTractor,
		permission.ResourceTrailer,
		permission.ResourceCarrier,
		permission.ResourceOrder,
		permission.ResourceInvoice,
	}
	for _, kind := range classifiedKinds() {
		if !slices.Contains(resources, kind) {
			resources = append(resources, kind)
		}
	}

	return resources
}()

// linkedFields answers for the records a hit names besides its own, through
// the reference view port and sharing the batch's answers for each kind's
// read.
func (s *Service) linkedFields(
	req *kernelports.GlobalSearchRequest,
	readable map[permission.Resource]bool,
) permissionports.LinkedFields {
	check := func(_ context.Context, kind permission.Resource) (bool, error) {
		return readable[kind], nil
	}
	if s.views == nil {
		return readsOnly{check: check}
	}

	return s.views.Person(&permissionports.RequestActor{
		PrincipalType:  req.Principal.Type,
		PrincipalID:    req.Principal.ID,
		UserID:         req.Principal.UserID,
		APIKeyID:       req.Principal.APIKeyID,
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
	}, check)
}

// readsOnly is the answer without reference views: a linked record's fields
// need its kind's read.
type readsOnly struct {
	check func(ctx context.Context, kind permission.Resource) (bool, error)
}

func (r readsOnly) Reads(ctx context.Context, kind permission.Resource) bool {
	allowed, err := r.check(ctx, kind)
	return err == nil && allowed
}

func (r readsOnly) Decide(
	ctx context.Context,
	kind permission.Resource,
	_ string,
) permissionports.LinkedFieldDecision {
	if r.Reads(ctx, kind) {
		return permissionports.LinkedFieldVisible
	}
	return permissionports.LinkedFieldWithheldKind
}

func (s *Service) definitions() []entitySearchDefinition {
	return []entitySearchDefinition{
		{
			entityType: search.EntityTypeShipment,
			label:      "Shipments",
			index:      s.config.Meilisearch.Indexes.Shipments,
			resource:   permission.ResourceShipment,
			attributes: []string{"id", "pro_number", "bol", fieldStatus},
		},
		{
			entityType: search.EntityTypeCustomer,
			label:      "Customers",
			index:      s.config.Meilisearch.Indexes.Customers,
			resource:   permission.ResourceCustomer,
			attributes: []string{"id", "name", "code", fieldCity, "state_id", fieldStatus},
		},
		{
			entityType: search.EntityTypeWorker,
			label:      "Workers",
			index:      s.config.Meilisearch.Indexes.Workers,
			resource:   permission.ResourceWorker,
			attributes: []string{
				"id",
				"first_name",
				"last_name",
				"type",
				"driver_type",
				fieldStatus,
				"external_id",
				fieldCity,
				"state_id",
				"available_for_dispatch",
			},
		},
		{
			entityType: search.EntityTypeDocument,
			label:      "Documents",
			index:      s.config.Meilisearch.Indexes.Documents,
			resource:   permission.ResourceDocument,
			attributes: []string{
				"id",
				"original_name",
				"file_name",
				"resource_type",
				"resource_id",
				fieldStatus,
				"detected_kind",
				"created_at",
			},
		},
	}
}

func (s *Service) mapHits(
	definition entitySearchDefinition,
	documents []map[string]any,
) []*kernelports.GlobalSearchHit {
	hits := make([]*kernelports.GlobalSearchHit, 0, len(documents))
	for _, document := range documents {
		hit := mapHit(definition.entityType, document)
		if hit == nil || hit.Href == "" {
			continue
		}
		hits = append(hits, hit)
	}
	return hits
}

func mapHit(entityType search.EntityType, document map[string]any) *kernelports.GlobalSearchHit {
	id := stringValue(document, "id")
	if id == "" {
		return nil
	}

	switch entityType {
	case search.EntityTypeShipment:
		return shipmentHit(id, document)
	case search.EntityTypeCustomer:
		return customerHit(id, document)
	case search.EntityTypeWorker:
		return workerHit(id, document)
	case search.EntityTypeDocument:
		return documentHit(id, document)
	default:
		return nil
	}
}

func shipmentHit(id string, document map[string]any) *kernelports.GlobalSearchHit {
	title := stringutils.FirstNonEmpty(
		stringValue(document, "pro_number"),
		stringValue(document, "bol"),
		id,
	)
	metadata := map[string]string{
		fieldStatus: stringutils.FirstNonEmpty(stringValue(document, fieldStatus), "Unknown"),
	}
	putIfPresent(metadata, "proNumber", stringValue(document, "pro_number"))
	putIfPresent(metadata, "bol", stringValue(document, "bol"))

	return &kernelports.GlobalSearchHit{
		ID:         id,
		EntityType: search.EntityTypeShipment,
		Title:      title,
		Href:       recordHref("shipment", id),
		Metadata:   metadata,
	}
}

func customerHit(id string, document map[string]any) *kernelports.GlobalSearchHit {
	title := stringutils.FirstNonEmpty(
		stringValue(document, "name"),
		stringValue(document, "code"),
		id,
	)
	subtitle := stringutils.FirstNonEmpty(
		stringValue(document, "code"),
		stringValue(document, fieldCity),
	)
	metadata := map[string]string{
		fieldStatus: stringutils.FirstNonEmpty(stringValue(document, fieldStatus), "Unknown"),
	}
	putIfPresent(metadata, "code", stringValue(document, "code"))
	putIfPresent(metadata, fieldCity, stringValue(document, fieldCity))

	return &kernelports.GlobalSearchHit{
		ID:         id,
		EntityType: search.EntityTypeCustomer,
		Title:      title,
		Subtitle:   subtitle,
		Href:       recordHref("customer", id),
		Metadata:   metadata,
	}
}

func workerHit(id string, document map[string]any) *kernelports.GlobalSearchHit {
	title := strings.TrimSpace(
		stringValue(document, "first_name") + " " + stringValue(document, "last_name"),
	)
	title = stringutils.FirstNonEmpty(title, id)
	subtitle := stringutils.FirstNonEmpty(
		stringValue(document, "type"),
		stringValue(document, fieldStatus),
	)
	metadata := map[string]string{
		fieldStatus: stringutils.FirstNonEmpty(stringValue(document, fieldStatus), "Unknown"),
	}
	putIfPresent(metadata, "externalId", stringValue(document, "external_id"))
	putIfPresent(metadata, "workerType", stringValue(document, "type"))
	putIfPresent(metadata, "driverType", stringValue(document, "driver_type"))
	putIfPresent(metadata, fieldCity, stringValue(document, fieldCity))
	putIfPresent(metadata, "availableForDispatch", stringValue(document, "available_for_dispatch"))

	return &kernelports.GlobalSearchHit{
		ID:         id,
		EntityType: search.EntityTypeWorker,
		Title:      title,
		Subtitle:   subtitle,
		Href:       recordHref("worker", id),
		Metadata:   metadata,
	}
}

func documentHit(id string, document map[string]any) *kernelports.GlobalSearchHit {
	title := stringutils.FirstNonEmpty(
		stringValue(document, "original_name"),
		stringValue(document, "file_name"),
		id,
	)
	subtitle := stringutils.FirstNonEmpty(
		stringValue(document, "resource_type"),
		stringValue(document, fieldStatus),
	)
	href := documentHref(document)
	if href == "" {
		return nil
	}
	metadata := map[string]string{
		"resourceType": stringValue(document, "resource_type"),
	}
	putIfPresent(metadata, "resourceId", stringValue(document, "resource_id"))
	putIfPresent(metadata, fieldStatus, stringValue(document, fieldStatus))
	putIfPresent(metadata, "detectedKind", stringValue(document, "detected_kind"))
	putIfPresent(metadata, "receivedAt", stringValue(document, "created_at"))

	return &kernelports.GlobalSearchHit{
		ID:         id,
		EntityType: search.EntityTypeDocument,
		Title:      title,
		Subtitle:   subtitle,
		Href:       href,
		Metadata:   metadata,
	}
}

func putIfPresent(metadata map[string]string, key, value string) {
	if value != "" {
		metadata[key] = value
	}
}

// documentHref opens the record a document is attached to, where the app's
// own links open it.
func documentHref(document map[string]any) string {
	return recordHref(stringValue(document, "resource_type"), stringValue(document, "resource_id"))
}

// recordHref is where one record opens, from the registry the web app's own
// links are built from; "" for a kind of record with no page of its own.
func recordHref(entity, id string) string {
	path, _ := productguide.RecordPath(entity, id)

	return path
}

func tenantFilter(req *kernelports.GlobalSearchRequest) string {
	return searchfilter.And(
		searchfilter.Eq("organization_id", req.TenantInfo.OrgID.String()),
		searchfilter.Eq("business_unit_id", req.TenantInfo.BuID.String()),
	)
}

func limitForRequest(requested, fallback int) int {
	if requested <= 0 {
		requested = fallback
	}
	if requested > maxSearchLimit {
		return maxSearchLimit
	}
	return requested
}

func stringValue(document map[string]any, key string) string {
	value, ok := document[key]
	if !ok || value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return stringutils.DecodeByteString(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return fmt.Sprint(typed)
	}
}
