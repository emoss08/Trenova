package globalsearchservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/types/search"
	"github.com/shopspring/decimal"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/zap"
)

const partyExpansionLimit = 3

var labelledResources = []permission.Resource{
	permission.ResourceShipment,
	permission.ResourceCustomer,
	permission.ResourceWorker,
	permission.ResourceTractor,
	permission.ResourceTrailer,
	permission.ResourceCarrier,
}

type hitEnricher struct {
	service *Service
	req     *kernelports.GlobalSearchRequest
	access  *searchAccess
	limit   int
	results []*indexResult
}

type enrichment struct {
	links    []*shipmentports.ShipmentPartyLink
	cards    map[pulid.ID]*shipmentports.ShipmentSearchCard
	states   map[pulid.ID]string
	balances map[pulid.ID]*kernelports.CustomerOpenBalance
	labels   map[string]map[pulid.ID]*captureports.CaptureRecordLabel
}

func (e *hitEnricher) result(entityType search.EntityType) *indexResult {
	for _, result := range e.results {
		if result != nil && result.definition.entityType == entityType {
			return result
		}
	}
	return nil
}

func (e *hitEnricher) run(ctx context.Context) {
	shipments := e.result(search.EntityTypeShipment)
	customers := e.result(search.EntityTypeCustomer)
	workers := e.result(search.EntityTypeWorker)
	documents := e.result(search.EntityTypeDocument)

	found := &enrichment{}
	tasks := pool.New().WithContext(ctx)

	tasks.Go(func(c context.Context) error {
		found.links, found.cards = e.loadShipments(c, shipments, customers, workers)
		return nil
	})
	tasks.Go(func(c context.Context) error {
		found.states = e.loadStates(c, customers, workers)
		return nil
	})
	tasks.Go(func(c context.Context) error {
		found.balances = e.loadBalances(c, customers)
		return nil
	})
	tasks.Go(func(c context.Context) error {
		found.labels = e.loadLabels(c, documents)
		return nil
	})
	_ = tasks.Wait()

	e.applyShipments(found, shipments, workers)
	e.applyCustomers(found, customers)
	e.applyWorkers(found, workers)
	e.applyDocuments(found, documents)
}

func hitIDs(result *indexResult, limit int) []pulid.ID {
	if result == nil {
		return nil
	}
	ids := make([]pulid.ID, 0, min(len(result.hits), limit))
	for _, hit := range result.hits {
		if len(ids) == limit {
			break
		}
		if parsed, err := pulid.Parse(hit.ID); err == nil {
			ids = append(ids, parsed)
		}
	}
	return ids
}

// loadShipments reads the search card of every shipment a result line will
// name: the shipment hits themselves, the shipments of the customers found
// and the load each worker found is on now. A search for a customer or a
// driver also finds the freight they move, which no shipment index field
// would match.
func (e *hitEnricher) loadShipments(
	ctx context.Context,
	shipments, customers, workers *indexResult,
) ([]*shipmentports.ShipmentPartyLink, map[pulid.ID]*shipmentports.ShipmentSearchCard) {
	if !e.access.canRead(permission.ResourceShipment) || e.service.shipmentRead == nil {
		return nil, nil
	}

	var links []*shipmentports.ShipmentPartyLink
	linkRequest := &shipmentports.ListShipmentPartyLinksRequest{
		TenantInfo: e.req.TenantInfo,
		WorkerIDs:  hitIDs(workers, maxSearchLimit),
	}
	if shipments != nil {
		linkRequest.CustomerIDs = hitIDs(customers, partyExpansionLimit)
		linkRequest.CustomerLimit = e.limit
	}
	if len(linkRequest.CustomerIDs) > 0 || len(linkRequest.WorkerIDs) > 0 {
		found, err := e.service.shipmentRead.ListPartyLinks(ctx, linkRequest)
		if err != nil {
			e.service.logger.Warn("failed to find shipments for search parties", zap.Error(err))
		} else {
			links = found
		}
	}

	ids := hitIDs(shipments, maxSearchLimit)
	for _, link := range links {
		ids = append(ids, link.ShipmentID)
	}
	if len(ids) == 0 {
		return links, nil
	}

	cards, err := e.service.shipmentRead.ListSearchCards(ctx, &shipmentports.ListShipmentSearchCardsRequest{
		TenantInfo:  e.req.TenantInfo,
		ShipmentIDs: uniqueIDs(ids),
	})
	if err != nil {
		e.service.logger.Warn("failed to read shipment search cards", zap.Error(err))
		return links, nil
	}

	byID := make(map[pulid.ID]*shipmentports.ShipmentSearchCard, len(cards))
	for _, card := range cards {
		byID[card.ShipmentID] = card
	}
	return links, byID
}

func (e *hitEnricher) loadStates(
	ctx context.Context,
	results ...*indexResult,
) map[pulid.ID]string {
	if e.service.states == nil {
		return nil
	}

	ids := make([]pulid.ID, 0)
	for _, result := range results {
		if result == nil {
			continue
		}
		for _, document := range result.documents {
			if parsed, err := pulid.Parse(stringValue(document, "state_id")); err == nil {
				ids = append(ids, parsed)
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}

	states, err := e.service.states.GetByIDs(ctx, uniqueIDs(ids))
	if err != nil {
		e.service.logger.Warn("failed to read states for search hits", zap.Error(err))
		return nil
	}

	byID := make(map[pulid.ID]string, len(states))
	for _, state := range states {
		byID[state.ID] = state.Abbreviation
	}
	return byID
}

func (e *hitEnricher) loadBalances(
	ctx context.Context,
	customers *indexResult,
) map[pulid.ID]*kernelports.CustomerOpenBalance {
	if e.service.balances == nil || !e.access.canRead(permission.ResourceAccountsReceivable) {
		return nil
	}
	ids := hitIDs(customers, maxSearchLimit)
	if len(ids) == 0 {
		return nil
	}

	balances, err := e.service.balances.ListCustomerOpenBalances(
		ctx,
		&kernelports.ListCustomerOpenBalancesRequest{
			TenantInfo:  e.req.TenantInfo,
			CustomerIDs: ids,
		},
	)
	if err != nil {
		e.service.logger.Warn("failed to read customer balances for search", zap.Error(err))
		return nil
	}

	byID := make(map[pulid.ID]*kernelports.CustomerOpenBalance, len(balances))
	for _, balance := range balances {
		byID[balance.CustomerID] = balance
	}
	return byID
}

func (e *hitEnricher) loadLabels(
	ctx context.Context,
	documents *indexResult,
) map[string]map[pulid.ID]*captureports.CaptureRecordLabel {
	if e.service.recordLabels == nil || documents == nil {
		return nil
	}

	idsByKind := make(map[string][]pulid.ID)
	for _, hit := range documents.hits {
		kind := hit.Metadata["resourceType"]
		parsed, err := pulid.Parse(hit.Metadata["resourceId"])
		if err != nil || !e.canLabel(kind) {
			continue
		}
		idsByKind[kind] = append(idsByKind[kind], parsed)
	}
	if len(idsByKind) == 0 {
		return nil
	}

	labels := make(map[string]map[pulid.ID]*captureports.CaptureRecordLabel, len(idsByKind))
	for kind, ids := range idsByKind {
		found, err := e.service.recordLabels.Labels(ctx, &captureports.ListCaptureRecordLabelsRequest{
			TenantInfo:   e.req.TenantInfo,
			ResourceType: kind,
			IDs:          uniqueIDs(ids),
		})
		if err != nil {
			e.service.logger.Warn(
				"failed to label document records for search",
				zap.String("resourceType", kind),
				zap.Error(err),
			)
			continue
		}
		byID := make(map[pulid.ID]*captureports.CaptureRecordLabel, len(found))
		for _, label := range found {
			byID[label.ID] = label
		}
		labels[kind] = byID
	}
	return labels
}

func (e *hitEnricher) canLabel(kind string) bool {
	for _, resource := range labelledResources {
		if resource.String() == kind {
			return e.access.canRead(resource)
		}
	}
	return false
}

func (e *hitEnricher) applyShipments(
	found *enrichment,
	shipments, workers *indexResult,
) {
	if shipments != nil {
		seen := make(map[string]struct{}, len(shipments.hits))
		for _, hit := range shipments.hits {
			seen[hit.ID] = struct{}{}
		}
		added := 0
		for _, link := range found.links {
			if len(shipments.hits) >= e.limit {
				break
			}
			id := link.ShipmentID.String()
			if _, ok := seen[id]; ok || link.CustomerID.IsNil() && link.WorkerID.IsNil() {
				continue
			}
			card, ok := found.cards[link.ShipmentID]
			if !ok {
				continue
			}
			seen[id] = struct{}{}
			shipments.hits = append(shipments.hits, &kernelports.GlobalSearchHit{
				ID:         id,
				EntityType: search.EntityTypeShipment,
				Title:      card.ProNumber,
				Href:       recordHref("shipment", id),
				Metadata:   map[string]string{},
			})
			added++
		}
		shipments.total += added

		for _, hit := range shipments.hits {
			parsed, err := pulid.Parse(hit.ID)
			if err != nil {
				continue
			}
			if card, ok := found.cards[parsed]; ok {
				applyShipmentCard(hit, card)
			}
		}
	}

	if workers == nil {
		return
	}
	loads := make(map[pulid.ID]*shipmentports.ShipmentSearchCard)
	for _, link := range found.links {
		if link.WorkerID.IsNil() {
			continue
		}
		if card, ok := found.cards[link.ShipmentID]; ok {
			loads[link.WorkerID] = card
		}
	}
	for _, hit := range workers.hits {
		parsed, err := pulid.Parse(hit.ID)
		if err != nil {
			continue
		}
		if card, ok := loads[parsed]; ok {
			hit.Metadata["currentLoadId"] = card.ShipmentID.String()
			putIfPresent(hit.Metadata, "currentLoad", card.ProNumber)
			putIfPresent(hit.Metadata, "currentLoadStatus", string(card.Status))
		}
	}
}

func applyShipmentCard(hit *kernelports.GlobalSearchHit, card *shipmentports.ShipmentSearchCard) {
	if card.ProNumber != "" {
		hit.Title = card.ProNumber
	}
	hit.Metadata["status"] = string(card.Status)
	putIfPresent(hit.Metadata, "proNumber", card.ProNumber)
	putIfPresent(hit.Metadata, "bol", card.BOL)
	putIfPresent(hit.Metadata, "customerName", card.CustomerName)
	putIfPresent(hit.Metadata, "customerCode", card.CustomerCode)
	if !card.CustomerID.IsNil() {
		hit.Metadata["customerId"] = card.CustomerID.String()
	}
	putIfPresent(hit.Metadata, "serviceTypeCode", card.ServiceTypeCode)
	putIfPresent(hit.Metadata, "origin", place(card.OriginCity, card.OriginState))
	putIfPresent(hit.Metadata, "destination", place(card.DestinationCity, card.DestinationState))
	putIfPresent(hit.Metadata, "driverName", strings.TrimSpace(card.DriverName))
	if !card.DriverID.IsNil() {
		hit.Metadata["driverId"] = card.DriverID.String()
	}
}

func (e *hitEnricher) applyCustomers(found *enrichment, customers *indexResult) {
	if customers == nil {
		return
	}
	for idx, hit := range customers.hits {
		if document := documentFor(customers, hit.ID, idx); document != nil {
			applyPlace(hit, document, found.states)
		}
		parsed, err := pulid.Parse(hit.ID)
		if err != nil {
			continue
		}
		balance, ok := found.balances[parsed]
		if !ok {
			if found.balances != nil {
				hit.Metadata["openBalance"] = "0"
			}
			continue
		}
		hit.Metadata["openBalance"] = minorToDecimal(balance.OpenMinor)
		hit.Metadata["overdueBalance"] = minorToDecimal(balance.OverdueMinor)
		putIfPresent(hit.Metadata, "currencyCode", balance.CurrencyCode)
	}
}

func (e *hitEnricher) applyWorkers(found *enrichment, workers *indexResult) {
	if workers == nil {
		return
	}
	for idx, hit := range workers.hits {
		if document := documentFor(workers, hit.ID, idx); document != nil {
			applyPlace(hit, document, found.states)
		}
	}
}

func (e *hitEnricher) applyDocuments(found *enrichment, documents *indexResult) {
	if documents == nil {
		return
	}
	for _, hit := range documents.hits {
		kind := hit.Metadata["resourceType"]
		parsed, err := pulid.Parse(hit.Metadata["resourceId"])
		if err != nil {
			continue
		}
		label, ok := found.labels[kind][parsed]
		if !ok {
			continue
		}
		putIfPresent(hit.Metadata, "attachedLabel", label.Title)
	}
}

// documentFor finds the index row a hit was mapped from. Hits keep the rows'
// order, so the row at the same position is tried first.
func documentFor(result *indexResult, id string, position int) map[string]any {
	if position < len(result.documents) && stringValue(result.documents[position], "id") == id {
		return result.documents[position]
	}
	for _, document := range result.documents {
		if stringValue(document, "id") == id {
			return document
		}
	}
	return nil
}

func applyPlace(
	hit *kernelports.GlobalSearchHit,
	document map[string]any,
	states map[pulid.ID]string,
) {
	city := stringValue(document, "city")
	state := ""
	if parsed, err := pulid.Parse(stringValue(document, "state_id")); err == nil {
		state = states[parsed]
	}
	putIfPresent(hit.Metadata, "city", place(city, state))
}

func place(city, state string) string {
	city = strings.TrimSpace(city)
	state = strings.TrimSpace(state)
	switch {
	case city != "" && state != "":
		return city + ", " + state
	case city != "":
		return city
	default:
		return state
	}
}

func minorToDecimal(minor int64) string {
	return decimal.New(minor, -2).StringFixed(2)
}

func uniqueIDs(ids []pulid.ID) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(ids))
	unique := make([]pulid.ID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}
