package triageshipmentservice

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	lookupLimit           = 3
	maxReferenceRunes     = shipment.MaxExternalReferenceLength
	maxWeightPounds       = 10_000_000
	kilogramsToPounds     = 2.20462
	placeSeparator        = " · "
	instructionsMaxRunes  = 2000
	serviceTypeCandidates = 2
)

var weightPattern = regexp.MustCompile(`(?i)([\d,.]+)\s*(kg|kgs|kilograms?|lb|lbs|pounds?)?`)

type resolvedPlace struct {
	id       pulid.ID
	location *time.Location
}

type loadBuild struct {
	req      *inboundmessageports.CreateTriageLoadRequest
	fields   map[inboundmessage.LoadFieldKey]inboundmessageports.TriageLoadFieldValue
	multiErr *errortypes.MultiError
	now      time.Time
}

func (b *loadBuild) value(key inboundmessage.LoadFieldKey) (string, string, bool) {
	field, ok := b.fields[key]
	if !ok || strings.TrimSpace(field.Value) == "" {
		return "", "", false
	}

	return strings.TrimSpace(field.Value), fmt.Sprintf("fields[%d].value", field.Index), true
}

func (b *loadBuild) fail(key inboundmessage.LoadFieldKey, code errortypes.ErrorCode, message string, args ...any) {
	path := "fields"
	if field, ok := b.fields[key]; ok {
		path = fmt.Sprintf("fields[%d].value", field.Index)
	}
	b.multiErr.Add(path, code, message, args...)
}

func (s *Service) CreateLoad(
	ctx context.Context,
	req *inboundmessageports.CreateTriageLoadRequest,
) (*inboundmessageports.CreatedTriageLoad, error) {
	build := &loadBuild{
		req:      req,
		fields:   make(map[inboundmessage.LoadFieldKey]inboundmessageports.TriageLoadFieldValue, len(req.Fields)),
		multiErr: errortypes.NewMultiError(),
		now:      time.Unix(s.now(), 0),
	}
	for _, field := range req.Fields {
		build.fields[field.Key] = field
	}

	entity, err := s.buildShipment(ctx, build)
	if err != nil {
		return nil, err
	}
	if build.multiErr.HasErrors() {
		return nil, build.multiErr
	}

	created, err := s.creator.Create(ctx, entity, req.Actor)
	if err != nil {
		return nil, err
	}

	if instructions, _, ok := build.value(inboundmessage.LoadFieldInstructions); ok {
		if _, commentErr := s.comments.CreateSystem(ctx, &shipmentports.CreateSystemShipmentCommentRequest{
			TenantInfo: req.TenantInfo,
			ShipmentID: created.ID,
			Comment:    stringutils.TruncateRunes(instructions, instructionsMaxRunes),
			Type:       shipment.CommentTypePickupInstruction,
			Visibility: shipment.CommentVisibilityOperations,
		}); commentErr != nil {
			s.l.Warn("created a load from Triage but could not add its instructions",
				zap.String("shipmentId", created.ID.String()), zap.Error(commentErr))
		}
	}

	return &inboundmessageports.CreatedTriageLoad{ShipmentID: created.ID, ProNumber: created.ProNumber}, nil
}

func (s *Service) buildShipment(ctx context.Context, b *loadBuild) (*shipment.Shipment, error) {
	tenantInfo := b.req.TenantInfo
	customerID, err := s.resolveCustomer(ctx, b)
	if err != nil {
		return nil, err
	}

	entity := &shipment.Shipment{
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		CustomerID:       customerID,
		Status:           shipment.StatusNew,
		EntryMethod:      shipment.EntryMethodManual,
		FreightTerms:     shipment.FreightTermsPrepaid,
		RatingUnit:       1,
		IdempotencyKey:   b.req.IdempotencyKey,
		SourceDocumentID: sourceDocument(b.req.SourceDocumentID),
	}
	if b.req.Actor != nil {
		entity.EnteredByID = b.req.Actor.UserID
	}

	if customerID.IsNotNil() {
		if err = s.applyDefaults(ctx, b, entity); err != nil {
			return nil, err
		}
	}

	origin, err := s.resolvePlace(ctx, b, inboundmessage.LoadFieldOrigin, "Say where it picks up")
	if err != nil {
		return nil, err
	}
	destination, err := s.resolvePlace(ctx, b, inboundmessage.LoadFieldDestination, "Say where it delivers")
	if err != nil {
		return nil, err
	}

	pickup := b.window(inboundmessage.LoadFieldPickup, origin, "Say when it picks up")
	delivery := b.window(inboundmessage.LoadFieldDelivery, destination, "Say when it delivers")

	if err = s.applyEquipment(ctx, b, entity); err != nil {
		return nil, err
	}
	b.applyWeight(entity)
	b.applyRate(entity)
	b.applyReference(entity)
	if err = s.applyCommodity(ctx, b, entity); err != nil {
		return nil, err
	}

	if origin.id.IsNotNil() && destination.id.IsNotNil() && pickup != nil && delivery != nil {
		entity.Moves = []*shipment.ShipmentMove{{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Status:         shipment.MoveStatusNew,
			Loaded:         true,
			Stops: []*shipment.Stop{
				newStop(entity, origin.id, shipment.StopTypePickup, 0, pickup),
				newStop(entity, destination.id, shipment.StopTypeDelivery, 1, delivery),
			},
		}}
	}

	return entity, nil
}

func sourceDocument(id pulid.ID) string {
	if id.IsNil() {
		return ""
	}

	return id.String()
}

func newStop(
	entity *shipment.Shipment,
	locationID pulid.ID,
	stopType shipment.StopType,
	sequence int64,
	window *timeutils.Window,
) *shipment.Stop {
	stop := &shipment.Stop{
		OrganizationID:       entity.OrganizationID,
		BusinessUnitID:       entity.BusinessUnitID,
		LocationID:           locationID,
		Status:               shipment.StopStatusNew,
		Type:                 stopType,
		ScheduleType:         shipment.StopScheduleTypeOpen,
		Sequence:             sequence,
		ScheduledWindowStart: window.Start,
	}
	if window.End > window.Start {
		end := window.End
		stop.ScheduledWindowEnd = &end
	}

	return stop
}

func (s *Service) resolveCustomer(ctx context.Context, b *loadBuild) (pulid.ID, error) {
	text, _, ok := b.value(inboundmessage.LoadFieldCustomer)
	if !ok {
		if b.req.CustomerID.IsNotNil() {
			return b.req.CustomerID, nil
		}
		b.fail(inboundmessage.LoadFieldCustomer, errortypes.ErrRequired, "Say whose load it is")

		return pulid.Nil, nil
	}

	matches, err := s.lookup.FindCustomers(ctx, &shipmentports.TriageLookupQuery{
		TenantInfo: b.req.TenantInfo,
		Text:       text,
		Limit:      lookupLimit,
	})
	if err != nil {
		return pulid.Nil, err
	}

	match, ok := pickRecord(matches)
	if !ok {
		b.fail(inboundmessage.LoadFieldCustomer, errortypes.ErrInvalid, unresolved("customer", text, matches))

		return pulid.Nil, nil
	}

	return match.ID, nil
}

func pickRecord(matches []shipmentports.TriageRecordMatch) (shipmentports.TriageRecordMatch, bool) {
	switch {
	case len(matches) == 0:
		return shipmentports.TriageRecordMatch{}, false
	case matches[0].Exact && (len(matches) == 1 || !matches[1].Exact):
		return matches[0], true
	case len(matches) == 1:
		return matches[0], true
	default:
		return shipmentports.TriageRecordMatch{}, false
	}
}

func unresolved(kind, text string, matches []shipmentports.TriageRecordMatch) string {
	if len(matches) == 0 {
		return fmt.Sprintf("No %s matches %q. Add it first, or write it as it is on file.", kind, text)
	}

	labels := make([]string, 0, len(matches))
	for _, match := range matches {
		labels = append(labels, match.Label)
	}

	return fmt.Sprintf("%q could be %s. Write the one you mean.", text, strings.Join(labels, ", "))
}

func (s *Service) applyDefaults(ctx context.Context, b *loadBuild, entity *shipment.Shipment) error {
	defaults, err := s.lookup.CustomerDefaults(ctx, b.req.TenantInfo, entity.CustomerID)
	if err != nil {
		return err
	}
	if defaults != nil {
		entity.ServiceTypeID = defaults.ServiceTypeID
		entity.ShipmentTypeID = defaults.ShipmentTypeID
		entity.FormulaTemplateID = defaults.FormulaTemplateID
		entity.TractorTypeID = defaults.TractorTypeID
		entity.TrailerTypeID = defaults.TrailerTypeID

		return nil
	}

	serviceTypes, err := s.lookup.ActiveServiceTypes(ctx, b.req.TenantInfo, serviceTypeCandidates)
	if err != nil {
		return err
	}
	if len(serviceTypes) != 1 {
		b.fail(inboundmessage.LoadFieldCustomer, errortypes.ErrInvalid,
			"This customer has no earlier load to take the service type from. "+
				"Create this one from the shipment form.")

		return nil
	}
	entity.ServiceTypeID = serviceTypes[0]

	return nil
}

func splitPlace(text string) (name, city, state string) {
	if before, after, found := strings.Cut(text, placeSeparator); found {
		name = strings.TrimSpace(before)
		text = after
	}

	parts := strings.Split(text, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	switch {
	case len(parts) >= 3:
		return strings.TrimSpace(name + " " + strings.Join(parts[:len(parts)-2], ", ")),
			parts[len(parts)-2], parts[len(parts)-1]
	case len(parts) == 2:
		return name, parts[0], parts[1]
	case name != "":
		return name, parts[0], ""
	default:
		return parts[0], "", ""
	}
}

func (s *Service) resolvePlace(
	ctx context.Context,
	b *loadBuild,
	key inboundmessage.LoadFieldKey,
	missing string,
) (resolvedPlace, error) {
	text, _, ok := b.value(key)
	if !ok {
		b.fail(key, errortypes.ErrRequired, missing)

		return resolvedPlace{}, nil
	}

	name, city, state := splitPlace(text)
	matches, err := s.lookup.FindLocations(ctx, &shipmentports.TriageLocationQuery{
		TenantInfo: b.req.TenantInfo,
		Name:       name,
		City:       city,
		State:      state,
		Limit:      lookupLimit,
	})
	if err != nil {
		return resolvedPlace{}, err
	}
	if len(matches) == 0 && name != "" && city != "" {
		matches, err = s.lookup.FindLocations(ctx, &shipmentports.TriageLocationQuery{
			TenantInfo: b.req.TenantInfo,
			City:       city,
			State:      state,
			Limit:      lookupLimit,
		})
		if err != nil {
			return resolvedPlace{}, err
		}
	}

	records := make([]shipmentports.TriageRecordMatch, 0, len(matches))
	for _, match := range matches {
		records = append(records, shipmentports.TriageRecordMatch{
			ID: match.ID, Label: match.Label, Exact: match.Exact,
		})
	}
	picked, ok := pickRecord(records)
	if !ok {
		b.fail(key, errortypes.ErrInvalid, unresolved("location", text, records))

		return resolvedPlace{}, nil
	}

	zone := ""
	for _, match := range matches {
		if match.ID == picked.ID {
			zone = match.Timezone
		}
	}
	location, _ := timeutils.ResolveZone(zone)

	return resolvedPlace{id: picked.ID, location: location}, nil
}

func (b *loadBuild) window(
	key inboundmessage.LoadFieldKey,
	place resolvedPlace,
	missing string,
) *timeutils.Window {
	text, _, ok := b.value(key)
	if !ok {
		b.fail(key, errortypes.ErrRequired, missing)

		return nil
	}

	location := place.location
	if location == nil {
		location = time.UTC
	}
	window, ok := timeutils.ParseWindow(text, location, b.now)
	if !ok {
		b.fail(key, errortypes.ErrInvalid, "Write a date and time, such as Oct 7, 08:00–14:00")

		return nil
	}

	return &window
}

func (s *Service) applyEquipment(ctx context.Context, b *loadBuild, entity *shipment.Shipment) error {
	text, _, ok := b.value(inboundmessage.LoadFieldEquipment)
	if !ok {
		return nil
	}

	matches, err := s.lookup.FindTrailerTypes(ctx, &shipmentports.TriageLookupQuery{
		TenantInfo: b.req.TenantInfo,
		Text:       text,
		Limit:      lookupLimit,
	})
	if err != nil {
		return err
	}

	match, ok := pickRecord(matches)
	if !ok {
		b.fail(inboundmessage.LoadFieldEquipment, errortypes.ErrInvalid,
			unresolved("trailer type", text, matches))

		return nil
	}
	entity.TrailerTypeID = match.ID

	return nil
}

func (b *loadBuild) applyWeight(entity *shipment.Shipment) {
	text, _, ok := b.value(inboundmessage.LoadFieldWeight)
	if !ok {
		return
	}

	pounds, ok := parsePounds(text)
	if !ok {
		b.fail(inboundmessage.LoadFieldWeight, errortypes.ErrInvalid, "Write the weight in pounds, such as 42,000 lb")

		return
	}
	entity.Weight = &pounds
}

func parsePounds(text string) (int64, bool) {
	match := weightPattern.FindStringSubmatch(text)
	if match == nil {
		return 0, false
	}

	value, err := strconv.ParseFloat(strings.ReplaceAll(match[1], ",", ""), 64)
	if err != nil || value <= 0 {
		return 0, false
	}
	if unit := strings.ToLower(match[2]); strings.HasPrefix(unit, "k") {
		value *= kilogramsToPounds
	}

	pounds := int64(value + 0.5)
	if pounds <= 0 || pounds > maxWeightPounds {
		return 0, false
	}

	return pounds, true
}

func (b *loadBuild) applyRate(entity *shipment.Shipment) {
	text, _, ok := b.value(inboundmessage.LoadFieldRate)
	if !ok {
		return
	}

	rate, err := decimalutils.ParseMoneyText(firstAmount(text))
	if err != nil || !rate.Valid || rate.Decimal.IsNegative() {
		b.fail(inboundmessage.LoadFieldRate, errortypes.ErrInvalid, "Write the rate as an amount, such as $1,240.00")

		return
	}
	entity.BaseRate = rate
}

var amountPattern = regexp.MustCompile(`\$?\s*[\d,]+(?:\.\d+)?`)

func firstAmount(text string) string {
	if match := amountPattern.FindString(text); match != "" {
		return match
	}

	return text
}

func (b *loadBuild) applyReference(entity *shipment.Shipment) {
	text, _, ok := b.value(inboundmessage.LoadFieldReference)
	if !ok {
		return
	}
	if len([]rune(text)) > maxReferenceRunes {
		b.fail(inboundmessage.LoadFieldReference, errortypes.ErrInvalid,
			"Keep the reference under {0} characters", maxReferenceRunes)

		return
	}
	entity.ExternalReference = text
}

func (s *Service) applyCommodity(ctx context.Context, b *loadBuild, entity *shipment.Shipment) error {
	text, _, ok := b.value(inboundmessage.LoadFieldCommodity)
	if !ok {
		return nil
	}

	matches, err := s.lookup.FindCommodities(ctx, &shipmentports.TriageLookupQuery{
		TenantInfo: b.req.TenantInfo,
		Text:       text,
		Limit:      lookupLimit,
	})
	if err != nil {
		return err
	}

	match, ok := pickRecord(matches)
	if !ok {
		b.fail(inboundmessage.LoadFieldCommodity, errortypes.ErrInvalid, unresolved("commodity", text, matches))

		return nil
	}

	commodity := &shipment.ShipmentCommodity{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		CommodityID:    match.ID,
		Pieces:         1,
	}
	if entity.Weight != nil {
		commodity.Weight = *entity.Weight
	}
	entity.Commodities = []*shipment.ShipmentCommodity{commodity}

	return nil
}
