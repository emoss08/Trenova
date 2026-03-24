package globalsearchservice

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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
	SearchRepo   repoports.SearchRepository
	ShipmentRepo repoports.ShipmentRepository
	Permissions  serviceports.PermissionEngine
}

type Service struct {
	logger       *zap.Logger
	config       *config.SearchConfig
	searchRepo   repoports.SearchRepository
	shipmentRepo repoports.ShipmentRepository
	permissions  serviceports.PermissionEngine
}

var _ serviceports.GlobalSearchService = (*Service)(nil)

type entitySearchDefinition struct {
	entityType search.EntityType
	label      string
	index      string
	resource   permission.Resource
}

const maxSearchLimit = 25

func New(p Params) serviceports.GlobalSearchService {
	return &Service{
		logger:       p.Logger.Named("service.global-search"),
		config:       p.Config.GetSearchConfig(),
		searchRepo:   p.SearchRepo,
		shipmentRepo: p.ShipmentRepo,
		permissions:  p.Permissions,
	}
}

func (s *Service) Search(
	ctx context.Context,
	req *serviceports.GlobalSearchRequest,
) (*serviceports.GlobalSearchResult, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return &serviceports.GlobalSearchResult{
			Query:  "",
			Groups: []*serviceports.GlobalSearchGroup{},
		}, nil
	}
	if !s.searchRepo.Enabled() {
		return &serviceports.GlobalSearchResult{
			Query:  query,
			Groups: []*serviceports.GlobalSearchGroup{},
		}, nil
	}

	allowed, err := s.allowedDefinitions(ctx, req)
	if err != nil {
		return nil, err
	}

	groups := make([]*serviceports.GlobalSearchGroup, 0, len(allowed))
	results := make([]*serviceports.GlobalSearchGroup, len(allowed))
	limit := limitForRequest(req.Limit, s.config.GetDefaultLimit())
	filter := tenantFilter(req)

	searchPool := pool.New().WithContext(ctx).WithMaxGoroutines(len(allowed))
	for idx, definition := range allowed {
		if definition.index == "" {
			continue
		}

		searchPool.Go(func(c context.Context) error {
			documents, searchErr := s.searchRepo.Search(c, repoports.SearchRequest{
				Index:  definition.index,
				Query:  query,
				Limit:  limit,
				Filter: filter,
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

			hits := s.mapHits(definition, documents)
			if len(hits) == 0 {
				return nil
			}

			results[idx] = &serviceports.GlobalSearchGroup{
				EntityType: definition.entityType,
				Label:      definition.label,
				Hits:       hits,
			}
			return nil
		})
	}
	_ = searchPool.Wait()

	for _, result := range results {
		if result != nil {
			groups = append(groups, result)
		}
	}

	s.enrichShipmentHits(ctx, req.TenantInfo, groups)

	return &serviceports.GlobalSearchResult{
		Query:  query,
		Groups: groups,
	}, nil
}

func (s *Service) enrichShipmentHits(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	groups []*serviceports.GlobalSearchGroup,
) {
	var shipmentHits []*serviceports.GlobalSearchHit
	for _, group := range groups {
		if group.EntityType == search.EntityTypeShipment {
			shipmentHits = append(shipmentHits, group.Hits...)
		}
	}

	if len(shipmentHits) == 0 {
		return
	}

	ids := make([]pulid.ID, 0, len(shipmentHits))
	for _, hit := range shipmentHits {
		parsed, parseErr := pulid.Parse(hit.ID)
		if parseErr == nil {
			ids = append(ids, parsed)
		}
	}

	if len(ids) == 0 {
		return
	}

	shipments, err := s.shipmentRepo.GetByIDs(ctx, &repoports.GetShipmentsByIDsRequest{
		TenantInfo:  tenantInfo,
		ShipmentIDs: ids,
	})
	if err != nil {
		s.logger.Warn("failed to enrich shipment search hits", zap.Error(err))
		return
	}

	byID := make(map[string]*shipment.Shipment, len(shipments))
	for _, sp := range shipments {
		byID[sp.ID.String()] = sp
	}

	for _, hit := range shipmentHits {
		sp, ok := byID[hit.ID]
		if !ok {
			continue
		}

		if hit.Metadata == nil {
			hit.Metadata = make(map[string]string)
		}

		hit.Metadata["proNumber"] = sp.ProNumber
		hit.Metadata["status"] = string(sp.Status)

		if sp.BOL != "" {
			hit.Metadata["bol"] = sp.BOL
		}

		if sp.Customer != nil {
			hit.Metadata["customerName"] = sp.Customer.Name
			hit.Metadata["customerCode"] = sp.Customer.Code
		}

		if sp.ServiceType != nil {
			hit.Metadata["serviceTypeCode"] = sp.ServiceType.Code
		}
	}
}

func (s *Service) allowedDefinitions(
	ctx context.Context,
	req *serviceports.GlobalSearchRequest,
) ([]entitySearchDefinition, error) {
	definitions := s.definitions()
	if len(req.EntityTypes) > 0 {
		filtered := definitions[:0]
		for _, definition := range definitions {
			if slices.Contains(req.EntityTypes, definition.entityType) {
				filtered = append(filtered, definition)
			}
		}
		definitions = filtered
	}

	checks := make([]serviceports.ResourceOperationCheck, 0, len(definitions))
	for _, definition := range definitions {
		checks = append(checks, serviceports.ResourceOperationCheck{
			Resource:  definition.resource.String(),
			Operation: permission.OpRead,
		})
	}

	result, err := s.permissions.CheckBatch(ctx, &serviceports.BatchPermissionCheckRequest{
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

	allowed := make([]entitySearchDefinition, 0, len(definitions))
	for idx, definition := range definitions {
		if idx < len(result.Results) && result.Results[idx].Allowed {
			allowed = append(allowed, definition)
		}
	}

	return allowed, nil
}

func (s *Service) definitions() []entitySearchDefinition {
	return []entitySearchDefinition{
		{
			entityType: search.EntityTypeShipment,
			label:      "Shipments",
			index:      s.config.Meilisearch.Indexes.Shipments,
			resource:   permission.ResourceShipment,
		},
		{
			entityType: search.EntityTypeCustomer,
			label:      "Customers",
			index:      s.config.Meilisearch.Indexes.Customers,
			resource:   permission.ResourceCustomer,
		},
		{
			entityType: search.EntityTypeWorker,
			label:      "Workers",
			index:      s.config.Meilisearch.Indexes.Workers,
			resource:   permission.ResourceWorker,
		},
		{
			entityType: search.EntityTypeDocument,
			label:      "Documents",
			index:      s.config.Meilisearch.Indexes.Documents,
			resource:   permission.ResourceDocument,
		},
	}
}

func (s *Service) mapHits(
	definition entitySearchDefinition,
	documents []map[string]any,
) []*serviceports.GlobalSearchHit {
	hits := make([]*serviceports.GlobalSearchHit, 0, len(documents))
	for _, document := range documents {
		hit := mapHit(definition.entityType, document)
		if hit == nil || hit.Href == "" {
			continue
		}
		hits = append(hits, hit)
	}
	return hits
}

func mapHit(entityType search.EntityType, document map[string]any) *serviceports.GlobalSearchHit {
	id := stringValue(document, "id")
	if id == "" {
		return nil
	}

	switch entityType {
	case search.EntityTypeShipment:
		title := stringutils.FirstNonEmpty(
			stringValue(document, "pro_number"),
			stringValue(document, "bol"),
			id,
		)
		return &serviceports.GlobalSearchHit{
			ID:         id,
			EntityType: entityType,
			Title:      title,
			Href: fmt.Sprintf(
				"/shipment-management/shipments?panelEntityId=%s&panelType=edit",
				id,
			),
			Metadata: map[string]string{
				"status": stringutils.FirstNonEmpty(stringValue(document, "status"), "Unknown"),
			},
		}
	case search.EntityTypeCustomer:
		title := stringutils.FirstNonEmpty(
			stringValue(document, "name"),
			stringValue(document, "code"),
			id,
		)
		subtitle := stringutils.FirstNonEmpty(
			stringValue(document, "code"),
			stringValue(document, "city"),
		)
		return &serviceports.GlobalSearchHit{
			ID:         id,
			EntityType: entityType,
			Title:      title,
			Subtitle:   subtitle,
			Href: fmt.Sprintf(
				"/billing/configuration-files/customers?panelEntityId=%s&panelType=edit",
				id,
			),
			Metadata: map[string]string{
				"status": stringutils.FirstNonEmpty(stringValue(document, "status"), "Unknown"),
			},
		}
	case search.EntityTypeWorker:
		title := strings.TrimSpace(
			stringValue(document, "first_name") + " " + stringValue(document, "last_name"),
		)
		title = stringutils.FirstNonEmpty(title, id)
		subtitle := stringutils.FirstNonEmpty(
			stringValue(document, "type"),
			stringValue(document, "status"),
		)
		return &serviceports.GlobalSearchHit{
			ID:         id,
			EntityType: entityType,
			Title:      title,
			Subtitle:   subtitle,
			Href:       fmt.Sprintf("/dispatch/workers?panelEntityId=%s&panelType=edit", id),
			Metadata: map[string]string{
				"status": stringutils.FirstNonEmpty(stringValue(document, "status"), "Unknown"),
			},
		}
	case search.EntityTypeDocument:
		title := stringutils.FirstNonEmpty(
			stringValue(document, "original_name"),
			stringValue(document, "file_name"),
			id,
		)
		subtitle := stringutils.FirstNonEmpty(
			stringValue(document, "resource_type"),
			stringValue(document, "status"),
		)
		href := documentHref(document)
		if href == "" {
			return nil
		}
		return &serviceports.GlobalSearchHit{
			ID:         id,
			EntityType: entityType,
			Title:      title,
			Subtitle:   subtitle,
			Href:       href,
			Metadata: map[string]string{
				"resourceType": stringValue(document, "resource_type"),
			},
		}
	default:
		return nil
	}
}

func documentHref(document map[string]any) string {
	resourceType := stringValue(document, "resource_type")
	resourceID := stringValue(document, "resource_id")
	if resourceID == "" {
		return ""
	}

	switch resourceType {
	case "shipment":
		return fmt.Sprintf(
			"/shipment-management/shipments?panelEntityId=%s&panelType=edit",
			resourceID,
		)
	case "worker":
		return fmt.Sprintf("/dispatch/workers?panelEntityId=%s&panelType=edit", resourceID)
	case "customer":
		return fmt.Sprintf(
			"/billing/configuration-files/customers?panelEntityId=%s&panelType=edit",
			resourceID,
		)
	default:
		return ""
	}
}

func tenantFilter(req *serviceports.GlobalSearchRequest) string {
	return fmt.Sprintf(
		"organization_id = %q AND business_unit_id = %q",
		req.TenantInfo.OrgID.String(),
		req.TenantInfo.BuID.String(),
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
	case json.RawMessage:
		return stringutils.DecodeByteString(typed)
	case []byte:
		return stringutils.DecodeByteString(typed)
	default:
		return fmt.Sprint(typed)
	}
}
