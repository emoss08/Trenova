package provision

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/emoss08/trenova/edi-partner-sim/internal/trenova"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	entityCustomer          = "Customer"
	entityServiceType       = "ServiceType"
	entityShipmentType      = "ShipmentType"
	entityFormulaTemplate   = "FormulaTemplate"
	entityLocation          = "Location"
	entityCommodity         = "Commodity"
	entityAccessorialCharge = "AccessorialCharge"
	referenceLimit          = 100
	maxCommodityName        = 100
	postalPrefix            = 5
	preferredServiceType    = "STD"
	preferredShipmentType   = "FTL"
	preferredFormula        = "Flat Rate"
	preferredCategory       = "Customer Drop Point"
)

var ErrUnmappable = errors.New("the partner's code cannot be mapped")

type Mapper struct {
	trenova *trenova.Client
	mu      sync.Mutex
	targets map[string]trenova.SelectOption
	states  map[string]string
}

func NewMapper(client *trenova.Client) *Mapper {
	return &Mapper{
		trenova: client,
		targets: map[string]trenova.SelectOption{},
		states:  map[string]string{},
	}
}

func (m *Mapper) Resolve(
	ctx context.Context,
	setup *Result,
	transferID string,
) ([]trenova.MappingItem, error) {
	preview, err := m.trenova.MappingPreview(ctx, transferID)
	if err != nil {
		return nil, fmt.Errorf("read the transfer's mapping preview: %w", err)
	}
	if len(preview.Unresolved) == 0 {
		return nil, nil
	}
	payload, err := m.trenova.TransferPayload(ctx, transferID)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	items := make([]trenova.MappingItem, 0, len(preview.Unresolved))
	for _, row := range preview.Unresolved {
		target, targetErr := m.target(ctx, setup, payload, &row)
		if targetErr != nil {
			return nil, fmt.Errorf("map %s %s: %w", row.EntityType, row.SourceID, targetErr)
		}
		items = append(items, trenova.MappingItem{
			EntityType:  row.EntityType,
			SourceID:    row.SourceID,
			SourceLabel: row.SourceLabel,
			TargetID:    target.ID,
			TargetLabel: target.Display(),
		})
	}
	return items, nil
}

func (m *Mapper) target(
	ctx context.Context,
	setup *Result,
	payload *trenova.TenderPayload,
	row *trenova.MappingResolution,
) (trenova.SelectOption, error) {
	source := firstText(row.SourceLabel, row.SourceID)
	switch row.EntityType {
	case entityCustomer:
		if setup.CustomerID == "" {
			return trenova.SelectOption{}, fmt.Errorf("%w: provisioning made no customer for %s",
				ErrUnmappable, setup.Partner)
		}
		return trenova.SelectOption{ID: setup.CustomerID, Label: setup.CustomerLabel}, nil
	case entityServiceType:
		return m.reference(ctx, trenova.ResourceServiceType, source, preferredServiceType)
	case entityShipmentType:
		return m.reference(ctx, trenova.ResourceShipmentType, source, preferredShipmentType)
	case entityFormulaTemplate:
		return m.reference(ctx, trenova.ResourceFormulaTemplate, source, preferredFormula)
	case entityLocation:
		stop, ok := payload.Stop(row.SourceID)
		if !ok {
			return trenova.SelectOption{}, fmt.Errorf("%w: no stop in the tender is at %s",
				ErrUnmappable, row.SourceID)
		}
		return m.location(ctx, setup, stop)
	case entityCommodity:
		description := source
		if commodity, ok := payload.Commodity(row.SourceID); ok {
			description = firstText(commodity.CommodityDescription, commodity.CommodityName,
				description)
		}
		return m.commodity(ctx, setup, description, row.SourceID)
	case entityAccessorialCharge:
		code := source
		if charge, ok := payload.Charge(row.SourceID); ok {
			code = firstText(charge.AccessorialCode, code)
		}
		return m.reference(ctx, trenova.ResourceAccessorialCharge, code, "")
	default:
		return trenova.SelectOption{}, fmt.Errorf("%w: the runner does not map %s codes",
			ErrUnmappable, row.EntityType)
	}
}

func (m *Mapper) reference(
	ctx context.Context,
	resource, source, preferred string,
) (trenova.SelectOption, error) {
	key := resource + "\x00" + strings.ToUpper(source)
	if known, ok := m.targets[key]; ok {
		return known, nil
	}
	options, err := m.trenova.SelectOptions(ctx, resource, "", referenceLimit)
	if err != nil {
		return trenova.SelectOption{}, fmt.Errorf("list %s records: %w", resource, err)
	}
	picked := -1
	for _, want := range []string{source, preferred} {
		if want == "" || picked >= 0 {
			continue
		}
		for index := range options {
			if strings.EqualFold(options[index].Label, want) ||
				strings.EqualFold(options[index].MetaString("code"), want) {
				picked = index
				break
			}
		}
	}
	if picked < 0 && preferred != "" && len(options) > 0 {
		picked = 0
	}
	if picked < 0 {
		return trenova.SelectOption{}, fmt.Errorf("%w: Trenova has no %s record for %q",
			ErrUnmappable, resource, source)
	}
	m.targets[key] = options[picked]
	return options[picked], nil
}

func (m *Mapper) location(
	ctx context.Context,
	setup *Result,
	stop *trenova.TenderStop,
) (trenova.SelectOption, error) {
	name := strings.TrimSpace(stop.LocationName)
	if name == "" || stop.LocationAddressLine1 == "" || stop.LocationCity == "" ||
		stop.LocationStateCode == "" || stop.LocationPostalCode == "" {
		return trenova.SelectOption{}, fmt.Errorf("%w: the tender gives %s no full name and "+
			"address to create a location from", ErrUnmappable, stop.LocationID)
	}
	postal := prefix(stop.LocationPostalCode, postalPrefix)
	key := trenova.ResourceLocation + "\x00" + strings.ToUpper(name) + "\x00" + postal
	if known, ok := m.targets[key]; ok {
		return known, nil
	}
	options, err := m.trenova.SelectOptions(ctx, trenova.ResourceLocation, name, searchLimit)
	if err != nil {
		return trenova.SelectOption{}, fmt.Errorf("look for the location %s: %w", name, err)
	}
	for index := range options {
		option := options[index]
		if strings.EqualFold(option.Label, name) &&
			prefix(option.MetaString("postalCode"), postalPrefix) == postal &&
			strings.EqualFold(option.MetaString("city"), stop.LocationCity) {
			m.targets[key] = option
			return option, nil
		}
	}
	state, err := m.state(ctx, stop.LocationStateCode)
	if err != nil {
		return trenova.SelectOption{}, err
	}
	category, err := m.reference(ctx, trenova.ResourceLocationCategory, preferredCategory,
		preferredCategory)
	if err != nil {
		return trenova.SelectOption{}, err
	}
	created, err := m.trenova.CreateLocation(ctx, &trenova.NewLocation{
		Name: name,
		Description: fmt.Sprintf("Tendered by EDI simulator partner %s as %s", setup.Partner,
			stop.LocationID),
		AddressLine1: stop.LocationAddressLine1,
		AddressLine2: stop.LocationAddressLine2,
		City:         stop.LocationCity,
		StateID:      state,
		PostalCode:   stop.LocationPostalCode,
		CategoryID:   category.ID,
	})
	if err != nil {
		return trenova.SelectOption{}, fmt.Errorf("create the location %s: %w", name, err)
	}
	m.targets[key] = *created
	return *created, nil
}

func (m *Mapper) commodity(
	ctx context.Context,
	setup *Result,
	description, sourceID string,
) (trenova.SelectOption, error) {
	name := prefix(strings.TrimSpace(description), maxCommodityName)
	if name == "" {
		return trenova.SelectOption{}, fmt.Errorf("%w: the tender does not describe "+
			"commodity %s", ErrUnmappable, sourceID)
	}
	key := trenova.ResourceCommodity + "\x00" + strings.ToUpper(name)
	if known, ok := m.targets[key]; ok {
		return known, nil
	}
	options, err := m.trenova.SelectOptions(ctx, trenova.ResourceCommodity, name, searchLimit)
	if err != nil {
		return trenova.SelectOption{}, fmt.Errorf("look for the commodity %s: %w", name, err)
	}
	for index := range options {
		if strings.EqualFold(options[index].Label, name) {
			m.targets[key] = options[index]
			return options[index], nil
		}
	}
	created, err := m.trenova.CreateCommodity(ctx, name,
		fmt.Sprintf("%s, tendered by EDI simulator partner %s", name, setup.Partner))
	if err != nil {
		return trenova.SelectOption{}, fmt.Errorf("create the commodity %s: %w", name, err)
	}
	m.targets[key] = *created
	return *created, nil
}

func (m *Mapper) state(ctx context.Context, abbreviation string) (string, error) {
	key := strings.ToUpper(abbreviation)
	if known, ok := m.states[key]; ok {
		return known, nil
	}
	id, err := findState(ctx, m.trenova, key)
	if err != nil {
		return "", err
	}
	m.states[key] = id
	return id, nil
}

func firstText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func prefix(value string, length int) string {
	return stringutils.TruncateRunes(value, length)
}
