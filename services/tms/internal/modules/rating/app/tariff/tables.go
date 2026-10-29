package tariff

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixcell"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixversion"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratezoneports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/ports/servicetypeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const (
	unitWeightBreak = "lb"
	unitLinearFeet  = "ft"

	prefixZone    = "ZN:"
	prefixState   = "ST:"
	prefixZip5    = "Z5:"
	prefixZip3    = "Z3:"
	prefixCountry = "CT:"

	comboSeparator = "\x1f"
)

type TablesParams struct {
	fx.In

	Versions       ratematrixports.RateMatrixVersionRepository
	Zones          ratezoneports.RateZoneRepository
	States         usstateports.UsStateRepository
	EquipmentTypes equipmenttypeports.EquipmentTypeRepository
	ServiceTypes   servicetypeports.ServiceTypeRepository
}

type Tables struct {
	versions       ratematrixports.RateMatrixVersionRepository
	zones          ratezoneports.RateZoneRepository
	states         usstateports.UsStateRepository
	equipmentTypes equipmenttypeports.EquipmentTypeRepository
	serviceTypes   servicetypeports.ServiceTypeRepository
}

//nolint:gocritic // fx param structs are passed by value
func NewTables(p TablesParams) *Tables {
	return &Tables{
		versions:       p.Versions,
		zones:          p.Zones,
		states:         p.States,
		equipmentTypes: p.EquipmentTypes,
		serviceTypes:   p.ServiceTypes,
	}
}

type tableSet struct {
	stamp  string
	byCode map[string]*ratetablecache.Version
	order  []string
}

func (s *tableSet) table(code string) (*ratetablecache.Version, bool) {
	version, ok := s.byCode[code]
	return version, ok
}

func (t *Tables) served(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (string, *ratetablecache.Served, error) {
	return ratetablecache.ServedTables(
		ctx,
		tenant.OrgID,
		tenant.BuID,
		func(ctx context.Context) (string, error) {
			return t.versions.ServedStamp(ctx, tenant)
		},
		func(ctx context.Context, _ string) (*ratetablecache.Served, error) {
			return t.buildServed(ctx, tenant)
		},
	)
}

func (t *Tables) set(
	ctx context.Context,
	tenant pagination.TenantInfo,
	pins []tarifftypes.TablePin,
) (*tableSet, error) {
	stamp, served, err := t.served(ctx, tenant)
	if err != nil {
		return nil, err
	}
	set := &tableSet{
		stamp:  stamp,
		byCode: make(map[string]*ratetablecache.Version, len(served.Versions)+len(pins)),
		order:  make([]string, 0, len(served.Versions)+len(pins)),
	}
	for _, version := range served.Versions {
		set.add(version)
	}
	if len(pins) == 0 {
		return set, nil
	}

	pinned, err := t.pinnedVersions(ctx, tenant, pins)
	if err != nil {
		return nil, err
	}
	for _, version := range pinned {
		set.add(version)
	}
	slices.Sort(set.order)
	return set, nil
}

func (s *tableSet) add(version *ratetablecache.Version) {
	code := version.Index.Code
	if _, exists := s.byCode[code]; !exists {
		s.order = append(s.order, code)
	}
	s.byCode[code] = version
}

func (t *Tables) pinnedVersions(
	ctx context.Context,
	tenant pagination.TenantInfo,
	pins []tarifftypes.TablePin,
) ([]*ratetablecache.Version, error) {
	keys := make([]ratetablecache.VersionKey, 0, len(pins))
	for _, pin := range pins {
		if pin.RateMatrixID.IsNil() || pin.VersionNumber <= 0 {
			continue
		}
		keys = append(keys, ratetablecache.VersionKey{
			MatrixID: pin.RateMatrixID,
			Version:  pin.VersionNumber,
		})
	}
	found, err := t.versionEntries(ctx, tenant, keys)
	if err != nil {
		return nil, err
	}
	out := make([]*ratetablecache.Version, 0, len(keys))
	for _, key := range keys {
		if version, ok := found[key]; ok {
			out = append(out, version)
		}
	}
	return out, nil
}

func (t *Tables) versionEntries(
	ctx context.Context,
	tenant pagination.TenantInfo,
	keys []ratetablecache.VersionKey,
) (map[ratetablecache.VersionKey]*ratetablecache.Version, error) {
	return ratetablecache.Versions(
		ctx,
		tenant.OrgID,
		tenant.BuID,
		keys,
		func(ctx context.Context, missing []ratetablecache.VersionKey) ([]*ratetablecache.Version, error) {
			domainKeys := make([]ratematrixversion.VersionKey, 0, len(missing))
			for _, key := range missing {
				domainKeys = append(domainKeys, ratematrixversion.VersionKey{
					RateMatrixID:  key.MatrixID,
					VersionNumber: key.Version,
				})
			}
			versions, err := t.versions.GetByKeys(ctx, &ratematrixports.GetVersionsByKeysRequest{
				TenantInfo: tenant,
				Keys:       domainKeys,
			})
			if err != nil {
				return nil, err
			}
			return t.buildVersions(ctx, tenant, versions)
		},
	)
}

func (t *Tables) Stamp(ctx context.Context, tenant pagination.TenantInfo) (string, error) {
	stamp, _, err := t.served(ctx, tenant)
	return stamp, err
}

func (t *Tables) Shapes(
	ctx context.Context,
	tenant pagination.TenantInfo,
	distanceUnit string,
	pins []tarifftypes.TablePin,
) ([]rules.TariffTable, error) {
	set, err := t.set(ctx, tenant, pins)
	if err != nil {
		return nil, err
	}
	tables := make([]rules.TariffTable, 0, len(set.order))
	for _, code := range set.order {
		version := set.byCode[code]
		table := version.Shape
		table.Axes = withDistanceUnit(table.Axes, distanceUnit)
		if table.Loaded != nil {
			table.Loaded = [][]string{}
		}
		tables = append(tables, table)
	}
	return tables, nil
}

func withDistanceUnit(axes []rules.TariffAxis, unit string) []rules.TariffAxis {
	out := make([]rules.TariffAxis, len(axes))
	copy(out, axes)
	for i := range out {
		if out[i].Kind == string(ratematrix.DimensionKindDistance) &&
			out[i].Match == string(ratematrix.MatchModeRange) {
			out[i].Unit = unit
		}
	}
	return out
}

func (t *Tables) buildServed(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*ratetablecache.Served, error) {
	served, err := t.versions.ListServed(ctx, tenant)
	if err != nil {
		return nil, err
	}
	keys := make([]ratetablecache.VersionKey, 0, len(served))
	byKey := make(map[ratetablecache.VersionKey]*ratematrixversion.RateMatrixVersion, len(served))
	for _, version := range served {
		key := cacheKey(version.Key())
		keys = append(keys, key)
		byKey[key] = version
	}
	entries, err := ratetablecache.Versions(
		ctx,
		tenant.OrgID,
		tenant.BuID,
		keys,
		func(ctx context.Context, missing []ratetablecache.VersionKey) ([]*ratetablecache.Version, error) {
			versions := make([]*ratematrixversion.RateMatrixVersion, 0, len(missing))
			for _, key := range missing {
				versions = append(versions, byKey[key])
			}
			return t.buildVersions(ctx, tenant, versions)
		},
	)
	if err != nil {
		return nil, err
	}
	out := &ratetablecache.Served{Versions: make([]*ratetablecache.Version, 0, len(keys))}
	for _, key := range keys {
		if entry, ok := entries[key]; ok {
			out.Versions = append(out.Versions, entry)
		}
	}
	return out, nil
}

func cacheKey(key ratematrixversion.VersionKey) ratetablecache.VersionKey {
	return ratetablecache.VersionKey{MatrixID: key.RateMatrixID, Version: key.VersionNumber}
}

func domainKey(key ratetablecache.VersionKey) ratematrixversion.VersionKey {
	return ratematrixversion.VersionKey{RateMatrixID: key.MatrixID, VersionNumber: key.Version}
}

func (t *Tables) buildVersions(
	ctx context.Context,
	tenant pagination.TenantInfo,
	versions []*ratematrixversion.RateMatrixVersion,
) ([]*ratetablecache.Version, error) {
	out := make([]*ratetablecache.Version, 0, len(versions))
	whole := make([]ratematrixversion.VersionKey, 0, len(versions))
	wholeAt := make(map[ratematrixversion.VersionKey]int, len(versions))

	for _, version := range versions {
		if version == nil {
			continue
		}
		dimensions := version.OrderedDimensions()
		if len(dimensions) == 0 || len(dimensions) > ratematrix.MaxDimensions {
			continue
		}
		entry := &ratetablecache.Version{
			Shape: rules.TariffTable{
				Code:      version.Code,
				Name:      version.Name,
				Version:   version.VersionNumber,
				MatrixID:  version.RateMatrixID,
				ValueType: version.ValueType,
				Axes:      make([]rules.TariffAxis, 0, len(dimensions)),
			},
			Index: ratetablecache.Table{
				Key:  cacheKey(version.Key()),
				Code: version.Code,
				Axes: make([]ratetablecache.Axis, 0, len(dimensions)),
			},
		}
		keyed := false
		for _, dimension := range dimensions {
			entry.Shape.Axes = append(entry.Shape.Axes, axisShape(dimension))
			axis, err := t.axisIndex(ctx, tenant, version.Key(), dimension)
			if err != nil {
				return nil, err
			}
			keyed = keyed || axis.Keyed()
			entry.Index.Axes = append(entry.Index.Axes, axis)
		}
		if keyed {
			entry.Shape.Loaded = [][]string{}
		} else {
			entry.Shape.Cells = []rules.TariffCell{}
			wholeAt[version.Key()] = len(out)
			whole = append(whole, version.Key())
		}
		out = append(out, entry)
	}

	if len(whole) == 0 {
		return out, nil
	}
	cells, err := t.versions.ListTableCells(ctx, &ratematrixports.ListTableCellsRequest{
		TenantInfo: tenant,
		Keys:       whole,
	})
	if err != nil {
		return nil, err
	}
	for _, cell := range cells {
		at, ok := wholeAt[ratematrixversion.VersionKey{
			RateMatrixID:  cell.RateMatrixID,
			VersionNumber: cell.VersionNumber,
		}]
		if !ok {
			continue
		}
		out[at].Shape.Cells = append(out[at].Shape.Cells, encodeCell(&out[at].Index, cell))
	}
	return out, nil
}

func axisShape(dimension *ratematrix.RateMatrixDimension) rules.TariffAxis {
	axis := rules.TariffAxis{
		Match: string(dimension.MatchMode),
		Kind:  string(dimension.Kind),
		Label: dimension.Label,
	}
	if dimension.MatchMode == ratematrix.MatchModeExact {
		axis.Normalization = string(dimension.KeyNormalization)
		return axis
	}
	if dimension.IsPostalRange() {
		return axis
	}
	axis.Overflow = string(dimension.RangeOverflow)
	switch dimension.Kind { //nolint:exhaustive // the other range axes are counts
	case ratematrix.DimensionKindWeightBreak:
		axis.Unit = unitWeightBreak
	case ratematrix.DimensionKindLinearFeet:
		axis.Unit = unitLinearFeet
	}
	return axis
}

func (t *Tables) axisIndex(
	ctx context.Context,
	tenant pagination.TenantInfo,
	key ratematrixversion.VersionKey,
	dimension *ratematrix.RateMatrixDimension,
) (ratetablecache.Axis, error) {
	axis := ratetablecache.Axis{
		Position:      dimension.Position,
		Kind:          string(dimension.Kind),
		Exact:         dimension.MatchMode == ratematrix.MatchModeExact,
		Postal:        dimension.IsPostalRange(),
		Normalization: string(dimension.KeyNormalization),
	}
	if axis.Postal {
		return t.postalIndex(ctx, tenant, key, axis)
	}
	if !axis.Exact || !translatedKind(dimension.Kind) {
		return axis, nil
	}

	stored, err := t.versions.ListAxisKeys(ctx, &ratematrixports.ListAxisKeysRequest{
		TenantInfo: tenant,
		Key:        key,
		Position:   dimension.Position,
	})
	if err != nil {
		return axis, err
	}
	names, err := t.referenceNames(ctx, tenant, dimension.Kind, stored)
	if err != nil {
		return axis, err
	}

	axis.Forward = make(map[string]string, len(stored))
	axis.Reverse = make(map[string][]string, len(stored))
	for _, storedKey := range stored {
		name, ok := names[referenceID(dimension.Kind, storedKey)]
		if !ok {
			continue
		}
		axis.Forward[storedKey] = name
		normalized := normalize(&axis, name)
		axis.Reverse[normalized] = append(axis.Reverse[normalized], storedKey)
	}
	return axis, nil
}

func (t *Tables) postalIndex(
	ctx context.Context,
	tenant pagination.TenantInfo,
	key ratematrixversion.VersionKey,
	axis ratetablecache.Axis,
) (ratetablecache.Axis, error) {
	ranges, err := t.versions.ListAxisRanges(ctx, &ratematrixports.ListAxisKeysRequest{
		TenantInfo: tenant,
		Key:        key,
		Position:   axis.Position,
	})
	if err != nil {
		return axis, err
	}
	index := ratematrixcell.NewPostalIndex(ranges)
	axis.Ranges = func(lookup string) []string {
		found := index.Find(lookup)
		starts := make([]string, 0, len(found))
		for _, postal := range found {
			starts = append(starts, postal.From)
		}
		return starts
	}
	return axis, nil
}

func translatedKind(kind ratematrix.DimensionKind) bool {
	switch kind { //nolint:exhaustive // only these axes store record IDs
	case ratematrix.DimensionKindZone,
		ratematrix.DimensionKindState,
		ratematrix.DimensionKindEquipmentType,
		ratematrix.DimensionKindServiceType:
		return true
	default:
		return false
	}
}

func referenceID(kind ratematrix.DimensionKind, key string) pulid.ID {
	trimmed := strings.TrimSpace(key)
	switch kind { //nolint:exhaustive // only zones and states carry a lane key prefix
	case ratematrix.DimensionKindZone:
		trimmed = strings.TrimPrefix(trimmed, prefixZone)
	case ratematrix.DimensionKindState:
		trimmed = strings.TrimPrefix(trimmed, prefixState)
	}
	return pulid.ID(trimmed)
}

func (t *Tables) referenceNames(
	ctx context.Context,
	tenant pagination.TenantInfo,
	kind ratematrix.DimensionKind,
	stored []string,
) (map[pulid.ID]string, error) {
	ids := make([]pulid.ID, 0, len(stored))
	for _, key := range stored {
		if id := referenceID(kind, key); id.IsNotNil() {
			ids = append(ids, id)
		}
	}
	names := make(map[pulid.ID]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}

	switch kind { //nolint:exhaustive // translatedKind admits only these
	case ratematrix.DimensionKindZone:
		return t.zones.GetCodesByIDs(ctx, tenant, ids)
	case ratematrix.DimensionKindState:
		states, err := t.states.GetByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, state := range states {
			names[state.ID] = state.Abbreviation
		}
	case ratematrix.DimensionKindEquipmentType:
		types, err := t.equipmentTypes.GetByIDs(
			ctx,
			equipmenttypeports.GetEquipmentTypesByIDsRequest{
				TenantInfo:       tenant,
				EquipmentTypeIDs: ids,
			},
		)
		if err != nil {
			return nil, err
		}
		for _, item := range types {
			names[item.ID] = item.Code
		}
	case ratematrix.DimensionKindServiceType:
		types, err := t.serviceTypes.GetByIDs(ctx, servicetypeports.GetServiceTypesByIDsRequest{
			TenantInfo:     tenant,
			ServiceTypeIDs: ids,
		})
		if err != nil {
			return nil, err
		}
		for _, item := range types {
			names[item.ID] = item.Code
		}
	}
	return names, nil
}

func normalize(axis *ratetablecache.Axis, key string) string {
	if axis.Postal {
		return ratematrixcell.NormalizePostal(key)
	}
	if axis.Kind == string(ratematrix.DimensionKindFreightClass) {
		if class, ok := commodity.ParseFreightClass(key); ok {
			return string(class)
		}
	}
	return ratematrix.KeyNormalization(axis.Normalization).Apply(key)
}

func forward(axis *ratetablecache.Axis, stored string) string {
	if axis.Forward != nil {
		if name, ok := axis.Forward[stored]; ok {
			return name
		}
		return stored
	}
	kind := ratematrix.DimensionKind(axis.Kind)
	switch kind { //nolint:exhaustive // other keys are stored as read
	case ratematrix.DimensionKindZip5:
		return strings.TrimPrefix(stored, prefixZip5)
	case ratematrix.DimensionKindZip3:
		return strings.TrimPrefix(stored, prefixZip3)
	case ratematrix.DimensionKindCountry:
		return strings.TrimPrefix(stored, prefixCountry)
	case ratematrix.DimensionKindFreightClass:
		if class, ok := commodity.ParseFreightClass(stored); ok {
			return string(class)
		}
	}
	return stored
}

func reverse(axis *ratetablecache.Axis, key string) []string {
	if axis.Postal {
		if axis.Ranges == nil {
			return nil
		}
		return axis.Ranges(key)
	}
	if axis.Reverse != nil {
		return axis.Reverse[key]
	}
	kind := ratematrix.DimensionKind(axis.Kind)
	switch kind { //nolint:exhaustive // other keys are stored as read
	case ratematrix.DimensionKindZip5:
		return []string{key, prefixZip5 + key}
	case ratematrix.DimensionKindZip3:
		return []string{key, prefixZip3 + key}
	case ratematrix.DimensionKindCountry:
		return []string{key, prefixCountry + key}
	case ratematrix.DimensionKindFreightClass:
		class := commodity.FreightClass(key)
		return []string{key, class.Number()}
	}
	return []string{key}
}

func encodeCell(index *ratetablecache.Table, cell *ratematrixcell.RateMatrixCell) rules.TariffCell {
	out := rules.TariffCell{
		At:      make([]rules.TariffCellPosition, 0, len(index.Axes)),
		Value:   cell.Value,
		Deficit: cell.DeficitEligible,
	}
	if cell.MinCharge.Valid {
		minimum := cell.MinCharge.Decimal
		out.Minimum = &minimum
	}
	for i := range index.Axes {
		axis := &index.Axes[i]
		switch {
		case axis.Postal:
			postal := cell.PostalRangeAt(axis.Position)
			out.At = append(out.At, rules.TariffCellPosition{From: postal.From, To: postal.To})
		case axis.Exact:
			out.At = append(
				out.At,
				rules.TariffCellPosition{Key: forward(axis, cell.KeyAt(axis.Position))},
			)
		default:
			low, high := cell.BoundsAt(axis.Position)
			position := rules.TariffCellPosition{}
			if low.Valid {
				value := low.Decimal
				position.Min = &value
			}
			if high.Valid {
				value := high.Decimal
				position.Max = &value
			}
			out.At = append(out.At, position)
		}
	}
	return out
}

func storedCombo(index *ratetablecache.Table, cell *rules.TariffCell) []string {
	combo := make([]string, 0, len(index.Axes))
	for i := range index.Axes {
		axis := &index.Axes[i]
		if i >= len(cell.At) {
			continue
		}
		switch {
		case axis.Postal:
			combo = append(combo, cell.At[i].From)
		case axis.Exact:
			combo = append(combo, normalize(axis, cell.At[i].Key))
		}
	}
	return combo
}

func lookupCombos(index *ratetablecache.Table, combo []string) [][]string {
	out := [][]string{make([]string, 0, len(combo))}
	keyed := 0
	for i := range index.Axes {
		axis := &index.Axes[i]
		if !axis.Keyed() {
			continue
		}
		if keyed >= len(combo) {
			return nil
		}
		key := combo[keyed]
		keyed++
		choices := []string{key}
		if axis.Postal {
			choices = reverse(axis, key)
		}
		next := make([][]string, 0, len(out)*len(choices))
		for _, prefix := range out {
			for _, choice := range choices {
				extended := append(slices.Clone(prefix), choice)
				next = append(next, extended)
			}
		}
		out = next
	}
	return out
}

func (t *Tables) LoadCells(
	ctx context.Context,
	tenant pagination.TenantInfo,
	tariffCtx *rules.TariffContext,
	needs []rules.TariffNeed,
) error {
	grouped := groupNeeds(needs)
	indexes, err := t.contextIndexes(ctx, tenant, tariffCtx, mapKeys(grouped))
	if err != nil {
		return err
	}
	for table, combos := range grouped {
		index, ok := indexes[table]
		if !ok {
			continue
		}
		at := tableAt(tariffCtx, table)
		if at < 0 {
			continue
		}
		cells, loadErr := t.cellsFor(ctx, tenant, index, combos)
		if loadErr != nil {
			return loadErr
		}
		target := &tariffCtx.Tables[at]
		merged := make([]rules.TariffCell, 0, len(target.Cells)+len(cells))
		merged = append(merged, target.Cells...)
		merged = append(merged, cells...)
		target.Cells = merged
		loaded := make([][]string, 0, len(target.Loaded)+len(combos))
		loaded = append(loaded, target.Loaded...)
		loaded = append(loaded, combos...)
		target.Loaded = loaded
	}
	return nil
}

func mapKeys(grouped map[string][][]string) []string {
	out := make([]string, 0, len(grouped))
	for code := range grouped {
		out = append(out, code)
	}
	return out
}

func (t *Tables) contextIndexes(
	ctx context.Context,
	tenant pagination.TenantInfo,
	tariffCtx *rules.TariffContext,
	codes []string,
) (map[string]*ratetablecache.Table, error) {
	keys := make([]ratetablecache.VersionKey, 0, len(codes))
	byKey := make(map[ratetablecache.VersionKey]string, len(codes))
	for _, code := range codes {
		at := tableAt(tariffCtx, code)
		if at < 0 {
			continue
		}
		table := &tariffCtx.Tables[at]
		if table.MatrixID.IsNil() || table.Version <= 0 {
			continue
		}
		key := ratetablecache.VersionKey{MatrixID: table.MatrixID, Version: table.Version}
		keys = append(keys, key)
		byKey[key] = code
	}
	found, err := t.versionEntries(ctx, tenant, keys)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*ratetablecache.Table, len(found))
	for key, version := range found {
		out[byKey[key]] = &version.Index
	}
	return out, nil
}

func (t *Tables) LoadWhole(
	ctx context.Context,
	tenant pagination.TenantInfo,
	tariffCtx *rules.TariffContext,
	tables []string,
) error {
	indexes, err := t.contextIndexes(ctx, tenant, tariffCtx, tables)
	if err != nil {
		return err
	}
	keys := make([]ratematrixversion.VersionKey, 0, len(indexes))
	byKey := make(map[ratematrixversion.VersionKey]*ratetablecache.Table, len(indexes))
	for _, index := range indexes {
		key := domainKey(index.Key)
		keys = append(keys, key)
		byKey[key] = index
	}
	if len(keys) == 0 {
		return nil
	}
	cells, err := t.versions.ListTableCells(ctx, &ratematrixports.ListTableCellsRequest{
		TenantInfo: tenant,
		Keys:       keys,
	})
	if err != nil {
		return err
	}
	byTable := make(map[string][]rules.TariffCell, len(keys))
	for _, cell := range cells {
		index, ok := byKey[ratematrixversion.VersionKey{
			RateMatrixID:  cell.RateMatrixID,
			VersionNumber: cell.VersionNumber,
		}]
		if !ok {
			continue
		}
		byTable[index.Code] = append(byTable[index.Code], encodeCell(index, cell))
	}
	for _, index := range byKey {
		target := &tariffCtx.Tables[tableAt(tariffCtx, index.Code)]
		target.Cells = byTable[index.Code]
		if target.Cells == nil {
			target.Cells = []rules.TariffCell{}
		}
		target.Loaded = nil
	}
	return nil
}

func (t *Tables) cellsFor(
	ctx context.Context,
	tenant pagination.TenantInfo,
	index *ratetablecache.Table,
	combos [][]string,
) ([]rules.TariffCell, error) {
	out := make([]rules.TariffCell, 0, len(combos))
	missing := make([][]string, 0, len(combos))
	for _, combo := range combos {
		if cached, ok := ratetablecache.Cells(tenant.OrgID, tenant.BuID, index.Key, combo); ok {
			out = append(out, cached...)
			continue
		}
		missing = append(missing, combo)
	}
	if len(missing) == 0 {
		return out, nil
	}

	keys := storedKeys(index, missing)
	if keys == nil {
		for _, combo := range missing {
			ratetablecache.StoreCells(tenant.OrgID, tenant.BuID, index.Key, combo, nil)
		}
		return out, nil
	}
	cells, err := t.versions.LoadTableCells(ctx, &ratematrixports.LoadTableCellsRequest{
		TenantInfo: tenant,
		Key:        domainKey(index.Key),
		Keys:       keys,
	})
	if err != nil {
		return nil, err
	}

	byStored := make(map[string][]rules.TariffCell, len(cells))
	for _, cell := range cells {
		encoded := encodeCell(index, cell)
		key := strings.Join(storedCombo(index, &encoded), comboSeparator)
		byStored[key] = append(byStored[key], encoded)
	}
	for _, combo := range missing {
		found := make([]rules.TariffCell, 0, 4)
		for _, stored := range lookupCombos(index, combo) {
			found = append(found, byStored[strings.Join(stored, comboSeparator)]...)
		}
		ratetablecache.StoreCells(tenant.OrgID, tenant.BuID, index.Key, combo, found)
		out = append(out, found...)
	}
	return out, nil
}

func storedKeys(index *ratetablecache.Table, combos [][]string) map[int16][]string {
	keys := make(map[int16][]string, len(index.Axes))
	seen := make(map[int16]map[string]struct{}, len(index.Axes))
	keyed := 0
	for i := range index.Axes {
		axis := &index.Axes[i]
		if !axis.Keyed() {
			continue
		}
		seen[axis.Position] = make(map[string]struct{}, len(combos))
		for _, combo := range combos {
			if keyed >= len(combo) {
				continue
			}
			for _, stored := range reverse(axis, combo[keyed]) {
				if _, dup := seen[axis.Position][stored]; dup {
					continue
				}
				seen[axis.Position][stored] = struct{}{}
				keys[axis.Position] = append(keys[axis.Position], stored)
			}
		}
		if len(keys[axis.Position]) == 0 {
			return nil
		}
		keyed++
	}
	return keys
}

func groupNeeds(needs []rules.TariffNeed) map[string][][]string {
	grouped := make(map[string][][]string, len(needs))
	seen := make(map[string]struct{}, len(needs))
	for _, need := range needs {
		key := need.Table + "\x1e" + strings.Join(need.Keys, comboSeparator)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		grouped[need.Table] = append(grouped[need.Table], need.Keys)
	}
	return grouped
}

func tableAt(tariffCtx *rules.TariffContext, code string) int {
	for i := range tariffCtx.Tables {
		if tariffCtx.Tables[i].Code == code {
			return i
		}
	}
	return -1
}
