package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ratequoteservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// draftReader is the one read the intake tool makes on document intelligence.
type draftReader interface {
	GetShipmentDraft(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*documentshipmentdraft.DocumentShipmentDraft, error)
}

// getShipmentDraftTool reads what document intelligence extracted from a
// customer's document, so an agent can turn it into a shipment without
// re-reading the PDF. The draft is the import assistant's output; this is the
// hand-off from that assistant to whichever agent creates the record.
type getShipmentDraftTool struct {
	drafts draftReader
}

func newGetShipmentDraftTool(drafts draftReader) serviceports.AgentQueryTool {
	return &getShipmentDraftTool{drafts: drafts}
}

func (t *getShipmentDraftTool) Name() string { return "get_shipment_draft" }

func (t *getShipmentDraftTool) Description() string {
	return "Read the shipment draft document intelligence extracted from an uploaded " +
		"document: the fields it found (BOL, customer, rate, pieces, weight, dates), " +
		"each with a confidence, the stops it found with their addresses and dates, " +
		"and what it could not read. Use it before create_shipment so the record is " +
		"built from the document rather than from memory. Every id the draft names " +
		"still has to be resolved with list_customers and list_locations; the draft " +
		"carries names and addresses, not ids."
}

func (t *getShipmentDraftTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"documentId": map[string]any{
				"type":        "string",
				"description": "The id of the uploaded document the draft was read from.",
			},
		},
		"required":             []string{"documentId"},
		"additionalProperties": false,
	}
}

func (t *getShipmentDraftTool) PermissionResource() permission.Resource {
	return permission.ResourceDocument
}

// draftData is the typed view of what document intelligence stores. The
// stored map's keys are the ones the extraction job writes; decoding through
// JSON keeps this in step with it without a second schema.
type draftData struct {
	ReviewStatus  string                `json:"reviewStatus"`
	MissingFields []string              `json:"missingFields"`
	Fields        map[string]draftField `json:"fields"`
	Stops         []draftStop           `json:"stops"`
	Conflicts     []draftConflict       `json:"conflicts"`
}

type draftField struct {
	Label          string  `json:"label,omitempty"`
	Value          any     `json:"value"`
	Confidence     float64 `json:"confidence"`
	ReviewRequired bool    `json:"reviewRequired,omitempty"`
	Conflict       bool    `json:"conflict,omitempty"`
}

type draftStop struct {
	Sequence            int     `json:"sequence"`
	Role                string  `json:"role"`
	Name                string  `json:"name,omitempty"`
	AddressLine1        string  `json:"addressLine1,omitempty"`
	AddressLine2        string  `json:"addressLine2,omitempty"`
	City                string  `json:"city,omitempty"`
	State               string  `json:"state,omitempty"`
	PostalCode          string  `json:"postalCode,omitempty"`
	Date                string  `json:"date,omitempty"`
	TimeWindow          string  `json:"timeWindow,omitempty"`
	AppointmentRequired bool    `json:"appointmentRequired,omitempty"`
	Confidence          float64 `json:"confidence"`
	ReviewRequired      bool    `json:"reviewRequired,omitempty"`
}

type draftConflict struct {
	Key    string `json:"key"`
	Label  string `json:"label,omitempty"`
	Values []any  `json:"values"`
}

type draftView struct {
	DocumentID         string                `json:"documentId"`
	Status             string                `json:"status"`
	ReviewStatus       string                `json:"reviewStatus,omitempty"`
	DocumentKind       string                `json:"documentKind,omitempty"`
	Confidence         float64               `json:"confidence"`
	AttachedShipmentID string                `json:"attachedShipmentId,omitempty"`
	FailureMessage     string                `json:"failureMessage,omitempty"`
	MissingFields      []string              `json:"missingFields"`
	Fields             map[string]draftField `json:"fields"`
	Stops              []draftStop           `json:"stops"`
	Conflicts          []draftConflict       `json:"conflicts,omitempty"`
	Note               string                `json:"note,omitempty"`
}

func (t *getShipmentDraftTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	documentID, err := requirePulid(params.Params, "documentId")
	if err != nil {
		return nil, err
	}

	draft, err := t.drafts.GetShipmentDraft(ctx, documentID, tenantOf(params))
	if err != nil {
		return nil, err
	}

	return draftViewOf(draft)
}

func draftViewOf(draft *documentshipmentdraft.DocumentShipmentDraft) (*draftView, error) {
	view := &draftView{
		DocumentID:     draft.DocumentID.String(),
		Status:         string(draft.Status),
		DocumentKind:   draft.DocumentKind,
		Confidence:     draft.Confidence,
		FailureMessage: draft.FailureMessage,
		MissingFields:  []string{},
		Fields:         map[string]draftField{},
		Stops:          []draftStop{},
	}
	if draft.AttachedShipmentID != nil && draft.AttachedShipmentID.IsNotNil() {
		view.AttachedShipmentID = draft.AttachedShipmentID.String()
		view.Note = "This draft was already turned into a shipment; do not create it again."
	}

	if len(draft.DraftData) > 0 {
		var data draftData
		if err := jsonutils.Convert(draft.DraftData, &data); err != nil {
			return nil, fmt.Errorf("draft data: %w", err)
		}
		view.ReviewStatus = data.ReviewStatus
		if data.MissingFields != nil {
			view.MissingFields = data.MissingFields
		}
		if data.Fields != nil {
			view.Fields = data.Fields
		}
		if data.Stops != nil {
			view.Stops = data.Stops
		}
		view.Conflicts = data.Conflicts
	}

	if view.Note == "" {
		switch {
		case draft.Status == documentshipmentdraft.StatusPending:
			view.Note = "Extraction is still running; ask again shortly."
		case draft.Status == documentshipmentdraft.StatusFailed,
			draft.Status == documentshipmentdraft.StatusUnavailable:
			view.Note = "Nothing usable was extracted from this document."
		case view.ReviewStatus == draftReviewNeeded:
			view.Note = "Some fields are low confidence or conflicting; confirm them " +
				"against the document or ask a person before creating the shipment."
		}
	}

	return view, nil
}

// shipmentQuoter and carrierShopper are the two questions the rating tools
// ask: what would we charge, and what would a carrier charge us.
type shipmentQuoter interface {
	Quote(ctx context.Context, req *ratequoteservice.QuoteRequest) (*serviceports.RatedShipment, error)
}

type carrierShopper interface {
	Shop(ctx context.Context, req *ratequoteservice.ShopRequest) (*serviceports.ShopResult, error)
}

type locationReader interface {
	GetByIDs(ctx context.Context, req repositories.GetLocationsByIDsRequest) ([]*location.Location, error)
}

// draftReviewNeeded is the review status document intelligence writes when
// a field is low confidence or two readings disagree.
const draftReviewNeeded = "NeedsReview"

const (
	maxQuoteStops   = 12
	maxShopCarriers = 25
	defaultShopMax  = 5
	maxShopOptions  = 20
)

// quoteShipmentTool prices a shipment that does not exist yet from the
// organization's rate agreements, the way the billing panel does before a
// load is saved. Nothing is written: the quote is not persisted and no
// shipment is created, so an agent can price a document three ways before
// proposing one.
type quoteShipmentTool struct {
	quotes    shipmentQuoter
	locations locationReader
}

func newQuoteShipmentTool(quotes shipmentQuoter, locations locationReader) serviceports.AgentQueryTool {
	return &quoteShipmentTool{quotes: quotes, locations: locations}
}

func (t *quoteShipmentTool) Name() string { return "quote_shipment" }

func (t *quoteShipmentTool) Description() string {
	return "Price a shipment that has not been saved, from the customer's rate " +
		"agreements and the organization's rating rules. Give the customer, the " +
		"service type and the stops in order, each with a location id from " +
		"list_locations and a date. Returns the linehaul amount, the currency, which " +
		"agreement and rule priced it, and whether a formula had to stand in because " +
		"no agreement covered the lane. Nothing is saved: quote as many variations " +
		"as you need before create_shipment."
}

func (t *quoteShipmentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"customerId":     map[string]any{"type": "string", "description": "The customer being billed."},
			"serviceTypeId":  map[string]any{"type": "string", "description": "The service type, from list_service_types."},
			"shipmentTypeId": map[string]any{"type": "string", "description": "The shipment type, from list_shipment_types."},
			"stops": map[string]any{
				"type":        "array",
				"description": "The stops in travel order: at least a pickup and a delivery.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"locationId": map[string]any{"type": "string", "description": "A location id from list_locations."},
						"type": map[string]any{
							"type": "string",
							"enum": []string{"Pickup", "Delivery", "SplitPickup", "SplitDelivery"},
						},
						"date": map[string]any{
							"type":        "string",
							"description": "The scheduled date, YYYY-MM-DD, or an RFC 3339 time.",
						},
					},
					"required":             []string{"locationId", "type"},
					"additionalProperties": false,
				},
			},
			"pieces":     map[string]any{"type": "integer", "description": "Piece count, when known."},
			"weight":     map[string]any{"type": "integer", "description": "Weight in pounds, when known."},
			"ratingUnit": map[string]any{"type": "integer", "description": "Rating units; defaults to 1."},
			"asOf": map[string]any{
				"type":        "string",
				"description": "Rate as of this date, YYYY-MM-DD. Defaults to the first stop's date, then today.",
			},
		},
		"required":             []string{"customerId", "serviceTypeId", "stops"},
		"additionalProperties": false,
	}
}

func (t *quoteShipmentTool) PermissionResource() permission.Resource {
	return permission.ResourceRateQuote
}

type quoteStopParam struct {
	LocationID string `json:"locationId"`
	Type       string `json:"type"`
	Date       string `json:"date"`
}

type quoteView struct {
	Outcome           string           `json:"outcome"`
	Amount            decimal.Decimal  `json:"amount"`
	Currency          string           `json:"currency"`
	BaseRate          *decimal.Decimal `json:"baseRate,omitempty"`
	AgreementID       string           `json:"agreementId,omitempty"`
	RuleID            string           `json:"ruleId,omitempty"`
	FormulaTemplateID string           `json:"formulaTemplateId,omitempty"`
	BillToCustomerID  string           `json:"billToCustomerId,omitempty"`
	Lane              []string         `json:"lane"`
	Note              string           `json:"note,omitempty"`
}

func (t *quoteShipmentTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	customerID, err := requirePulid(params.Params, "customerId")
	if err != nil {
		return nil, err
	}
	serviceTypeID, err := requirePulid(params.Params, "serviceTypeId")
	if err != nil {
		return nil, err
	}

	var stopParams []quoteStopParam
	if err = decodeParam(params.Params, "stops", &stopParams); err != nil {
		return nil, err
	}
	if len(stopParams) < 2 || len(stopParams) > maxQuoteStops {
		return nil, fmt.Errorf("a quote needs between 2 and %d stops", maxQuoteStops)
	}

	clk := clockFor(params)
	tenantInfo := tenantOf(params)
	entity, lane, err := t.hypotheticalShipment(ctx, tenantInfo, stopParams, clk)
	if err != nil {
		return nil, err
	}
	entity.CustomerID = customerID
	entity.ServiceTypeID = serviceTypeID
	if shipmentTypeID, ok, pErr := optionalPulid(params.Params, "shipmentTypeId"); pErr != nil {
		return nil, pErr
	} else if ok {
		entity.ShipmentTypeID = shipmentTypeID
	}
	if pieces := optionalInt(params.Params, "pieces", 0); pieces > 0 {
		value := int64(pieces)
		entity.Pieces = &value
	}
	if weight := optionalInt(params.Params, "weight", 0); weight > 0 {
		value := int64(weight)
		entity.Weight = &value
	}
	entity.RatingUnit = int64(optionalInt(params.Params, "ratingUnit", 1))
	if entity.RatingUnit <= 0 {
		entity.RatingUnit = 1
	}

	asOf := entity.Moves[0].Stops[0].ScheduledWindowStart
	if raw := optionalString(params.Params, "asOf"); raw != "" {
		if asOf, err = coerceDateValue("asOf", raw, clk); err != nil {
			return nil, err
		}
	}
	if asOf == 0 {
		asOf = clk.instant()
	}

	rated, err := t.quotes.Quote(ctx, &ratequoteservice.QuoteRequest{
		TenantInfo: tenantInfo,
		Shipment:   entity,
		AsOf:       asOf,
		Persist:    false,
	})
	if err != nil {
		return nil, err
	}

	return quoteViewOf(rated, lane), nil
}

// hypotheticalShipment builds the unsaved shipment the rate engine reads: one
// move whose stops carry their locations, because the lane is resolved from
// the locations' cities, states and postal codes.
func (t *quoteShipmentTool) hypotheticalShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	stopParams []quoteStopParam,
	clk clock,
) (*shipment.Shipment, []string, error) {
	ids := make([]pulid.ID, 0, len(stopParams))
	for idx, stop := range stopParams {
		id, err := pulid.Parse(stop.LocationID)
		if err != nil {
			return nil, nil, fmt.Errorf("stops[%d].locationId is not a location id", idx)
		}
		ids = append(ids, id)
	}

	locations, err := t.locations.GetByIDs(ctx, repositories.GetLocationsByIDsRequest{
		TenantInfo:  tenantInfo,
		LocationIDs: ids,
	})
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[pulid.ID]*location.Location, len(locations))
	for _, loc := range locations {
		byID[loc.ID] = loc
	}

	move := &shipment.ShipmentMove{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Status:         shipment.MoveStatusNew,
		Loaded:         true,
		Sequence:       0,
	}
	lane := make([]string, 0, len(stopParams))
	for idx, stop := range stopParams {
		loc, ok := byID[ids[idx]]
		if !ok {
			return nil, nil, fmt.Errorf("stops[%d].locationId %s is not a location in this organization", idx, ids[idx])
		}
		stopType := shipment.StopType(stop.Type)
		if !stopType.IsValid() {
			return nil, nil, fmt.Errorf("stops[%d].type %q is not a stop type", idx, stop.Type)
		}

		var scheduled int64
		if strings.TrimSpace(stop.Date) != "" {
			if scheduled, err = coerceDateValue(fmt.Sprintf("stops[%d].date", idx), stop.Date, clk); err != nil {
				return nil, nil, err
			}
		}

		move.Stops = append(move.Stops, &shipment.Stop{
			OrganizationID:       tenantInfo.OrgID,
			BusinessUnitID:       tenantInfo.BuID,
			LocationID:           loc.ID,
			Location:             loc,
			Status:               shipment.StopStatusNew,
			Type:                 stopType,
			ScheduleType:         shipment.StopScheduleTypeOpen,
			Sequence:             int64(idx),
			ScheduledWindowStart: scheduled,
		})
		lane = append(lane, placeLabel(loc))
	}

	return &shipment.Shipment{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Status:         shipment.StatusNew,
		RatingUnit:     1,
		Moves:          []*shipment.ShipmentMove{move},
	}, lane, nil
}

func placeLabel(loc *location.Location) string {
	parts := make([]string, 0, 3)
	if loc.City != "" {
		parts = append(parts, loc.City)
	}
	if loc.State != nil && loc.State.Abbreviation != "" {
		parts = append(parts, loc.State.Abbreviation)
	}
	if loc.PostalCode != "" {
		parts = append(parts, loc.PostalCode)
	}
	if len(parts) == 0 {
		return loc.Name
	}

	return strings.Join(parts, ", ")
}

func quoteViewOf(rated *serviceports.RatedShipment, lane []string) *quoteView {
	view := &quoteView{
		Outcome:  string(rated.Outcome),
		Amount:   rated.Amount,
		Currency: rated.Currency,
		Lane:     lane,
	}
	if rated.BaseRate.Valid {
		base := rated.BaseRate.Decimal
		view.BaseRate = &base
	}
	if rated.AgreementID != nil {
		view.AgreementID = rated.AgreementID.String()
	}
	if rated.RuleID != nil {
		view.RuleID = rated.RuleID.String()
	}
	if rated.FormulaTemplateID != nil {
		view.FormulaTemplateID = rated.FormulaTemplateID.String()
	}
	if rated.BillToCustomerID != nil {
		view.BillToCustomerID = rated.BillToCustomerID.String()
	}

	switch {
	case !rated.Outcome.Priced():
		view.Note = "No agreement or formula could price this lane; the amount is not a rate. " +
			"Say so rather than quoting a number, or ask a person for a manual rate."
	case rated.FormulaTemplateID != nil && rated.AgreementID == nil:
		view.Note = "No agreement covered the lane, so a formula priced it. Treat the amount " +
			"as a system estimate rather than a contracted rate."
	}

	return view
}

// shopCarriersTool asks what several carriers would charge to haul a saved
// shipment, ranked by the strategy the organization chose, so an agent can
// tender to the right one rather than the first one.
type shopCarriersTool struct {
	shopper carrierShopper
}

func newShopCarriersTool(shopper carrierShopper) serviceports.AgentQueryTool {
	return &shopCarriersTool{shopper: shopper}
}

func (t *shopCarriersTool) Name() string { return "shop_carriers" }

func (t *shopCarriersTool) Description() string {
	return "Price a saved shipment against carriers and rank them: what each would " +
		"charge under its contract, the margin against what the customer is being " +
		"billed, and where the routing guide ranks it. Leave the carrier list empty to " +
		"use the shipment's routing guide. Read this before tendering; nothing is " +
		"tendered or written by it."
}

func (t *shopCarriersTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentId": map[string]any{"type": "string", "description": "The saved shipment to price."},
			"strategy": map[string]any{
				"type": "string",
				"enum": []string{
					string(serviceports.ShopStrategyLeastCost),
					string(serviceports.ShopStrategyBestMargin),
					string(serviceports.ShopStrategyGuideRank),
					string(serviceports.ShopStrategyFastestAccept),
				},
				"description": "How to rank the options. Defaults to the organization's own choice.",
			},
			"carrierIds": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "An explicit shortlist of carrier ids. Empty means the routing guide's candidates.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "How many options to return; defaults to 5, at most 20.",
			},
		},
		"required":             []string{"shipmentId"},
		"additionalProperties": false,
	}
}

func (t *shopCarriersTool) PermissionResource() permission.Resource {
	return permission.ResourceRateQuote
}

type shopOptionView struct {
	Rank            int             `json:"rank"`
	CarrierID       string          `json:"carrierId"`
	CarrierName     string          `json:"carrierName"`
	Outcome         string          `json:"outcome"`
	Priced          bool            `json:"priced"`
	Cost            decimal.Decimal `json:"cost"`
	Currency        string          `json:"currency"`
	MarginAmount    decimal.Decimal `json:"marginAmount"`
	MarginPercent   decimal.Decimal `json:"marginPercent"`
	BelowFloor      bool            `json:"belowFloor"`
	GuideRank       int16           `json:"guideRank,omitempty"`
	OfferTTLSeconds int32           `json:"offerTtlSeconds,omitempty"`
	Note            string          `json:"note,omitempty"`
}

type shopView struct {
	Strategy       string           `json:"strategy"`
	RoutingGuideID string           `json:"routingGuideId,omitempty"`
	SellTotal      *decimal.Decimal `json:"sellTotal,omitempty"`
	Options        []shopOptionView `json:"options"`
	Warnings       []string         `json:"warnings,omitempty"`
	Note           string           `json:"note,omitempty"`
}

func (t *shopCarriersTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	shipmentID, err := requirePulid(params.Params, "shipmentId")
	if err != nil {
		return nil, err
	}

	limit := optionalInt(params.Params, "limit", defaultShopMax)
	if limit <= 0 || limit > maxShopOptions {
		limit = defaultShopMax
	}

	req := &ratequoteservice.ShopRequest{
		TenantInfo: tenantOf(params),
		ShipmentID: shipmentID,
		Strategy:   serviceports.ShopStrategy(optionalString(params.Params, "strategy")),
		Limit:      limit,
		Persist:    false,
	}
	if raw, ok := params.Params["carrierIds"]; ok && raw != nil {
		var carrierIDs []string
		if err = decodeParam(params.Params, "carrierIds", &carrierIDs); err != nil {
			return nil, err
		}
		if len(carrierIDs) > maxShopCarriers {
			return nil, fmt.Errorf("at most %d carriers can be shopped at once", maxShopCarriers)
		}
		for idx, rawID := range carrierIDs {
			id, parseErr := pulid.Parse(rawID)
			if parseErr != nil {
				return nil, fmt.Errorf("carrierIds[%d] is not a carrier id", idx)
			}
			req.CarrierIDs = append(req.CarrierIDs, id)
		}
	}

	result, err := t.shopper.Shop(ctx, req)
	if err != nil {
		return nil, err
	}

	return shopViewOf(result), nil
}

func shopViewOf(result *serviceports.ShopResult) *shopView {
	view := &shopView{
		Strategy: string(result.Strategy),
		Options:  make([]shopOptionView, 0, len(result.Options)),
		Warnings: result.Warnings,
	}
	if result.RoutingGuideID != nil {
		view.RoutingGuideID = result.RoutingGuideID.String()
	}
	if result.SellTotal.Valid {
		sell := result.SellTotal.Decimal
		view.SellTotal = &sell
	}
	for _, option := range result.Options {
		if option == nil {
			continue
		}
		view.Options = append(view.Options, shopOptionView{
			Rank:            option.Rank,
			CarrierID:       option.CarrierID.String(),
			CarrierName:     option.CarrierName,
			Outcome:         string(option.Outcome),
			Priced:          option.Priced(),
			Cost:            option.Cost,
			Currency:        option.Currency,
			MarginAmount:    option.Margin.Amount,
			MarginPercent:   option.Margin.Percent,
			BelowFloor:      option.Margin.BelowFloor,
			GuideRank:       option.GuideRank,
			OfferTTLSeconds: option.OfferTTLSeconds,
			Note:            option.Note,
		})
	}
	if len(view.Options) == 0 {
		view.Note = "No carrier could be priced for this shipment. Check the routing guide " +
			"covers the lane, or name carriers explicitly."
	}

	return view
}

func optionalPulid(params map[string]any, key string) (pulid.ID, bool, error) {
	value := optionalString(params, key)
	if value == "" {
		return pulid.Nil, false, nil
	}

	id, err := pulid.Parse(value)
	if err != nil {
		return pulid.Nil, false, fmt.Errorf("parameter %q is not an id", key)
	}

	return id, true, nil
}
