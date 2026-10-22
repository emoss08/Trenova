package tariff

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratezoneports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/ports/servicetypeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/pkg/rules"
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
)

type TablesParams struct {
	fx.In

	Matrices       ratematrixports.RateMatrixRepository
	Zones          ratezoneports.RateZoneRepository
	States         usstateports.UsStateRepository
	EquipmentTypes equipmenttypeports.EquipmentTypeRepository
	ServiceTypes   servicetypeports.ServiceTypeRepository
}

type Tables struct {
	matrices       ratematrixports.RateMatrixRepository
	zones          ratezoneports.RateZoneRepository
	states         usstateports.UsStateRepository
	equipmentTypes equipmenttypeports.EquipmentTypeRepository
	serviceTypes   servicetypeports.ServiceTypeRepository
}

//nolint:gocritic // fx param structs are passed by value
func NewTables(p TablesParams) *Tables {
	return &Tables{
		matrices:       p.Matrices,
		zones:          p.Zones,
		states:         p.States,
		equipmentTypes: p.EquipmentTypes,
		serviceTypes:   p.ServiceTypes,
	}
}

type tableState struct {
	stamp string
	entry *ratetablecache.Entry
}

func (t *Tables) load(ctx context.Context, tenant pagination.TenantInfo) (*tableState, error) {
	stamp, entry, err := ratetablecache.Tables(
		ctx,
		tenant.OrgID,
		tenant.BuID,
		func(ctx context.Context) (string, error) {
			return t.matrices.GetLookupStamp(ctx, tenant)
		},
		func(ctx context.Context, _ string) (*ratetablecache.Entry, error) {
			return t.build(ctx, tenant)
		},
	)
	if err != nil {
		return nil, err
	}
	return &tableState{stamp: stamp, entry: entry}, nil
}

func (t *Tables) Stamp(ctx context.Context, tenant pagination.TenantInfo) (string, error) {
	state, err := t.load(ctx, tenant)
	if err != nil {
		return "", err
	}
	return state.stamp, nil
}

func (t *Tables) Shapes(
	ctx context.Context,
	tenant pagination.TenantInfo,
	distanceUnit string,
) ([]rules.TariffTable, error) {
	state, err := t.load(ctx, tenant)
	if err != nil {
		return nil, err
	}
	tables := make([]rules.TariffTable, len(state.entry.Tables))
	for i := range state.entry.Tables {
		tables[i] = state.entry.Tables[i]
		tables[i].Axes = withDistanceUnit(tables[i].Axes, distanceUnit)
		if tables[i].Loaded != nil {
			tables[i].Loaded = [][]string{}
		}
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

func (t *Tables) build(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*ratetablecache.Entry, error) {
	matrices, err := t.matrices.ListTables(ctx, tenant)
	if err != nil {
		return nil, err
	}

	entry := &ratetablecache.Entry{
		Tables: make([]rules.TariffTable, 0, len(matrices)),
		Index:  make([]ratetablecache.Table, 0, len(matrices)),
	}
	whole := make([]pulid.ID, 0, len(matrices))
	wholeAt := make(map[pulid.ID]int, len(matrices))

	for _, matrix := range matrices {
		dimensions := matrix.OrderedDimensions()
		if len(dimensions) == 0 {
			continue
		}
		table := rules.TariffTable{
			Code:      matrix.Code,
			Name:      matrix.Name,
			ValueType: matrix.ValueType,
			Axes:      make([]rules.TariffAxis, 0, len(dimensions)),
		}
		index := ratetablecache.Table{
			MatrixID: matrix.ID,
			Code:     matrix.Code,
			Axes:     make([]ratetablecache.Axis, 0, len(dimensions)),
		}
		exact := false
		for _, dimension := range dimensions {
			table.Axes = append(table.Axes, axisShape(dimension))
			axis, axisErr := t.axisIndex(ctx, tenant, matrix.ID, dimension)
			if axisErr != nil {
				return nil, axisErr
			}
			exact = exact || axis.Exact
			index.Axes = append(index.Axes, axis)
		}
		if exact {
			table.Loaded = [][]string{}
		} else {
			table.Cells = []rules.TariffCell{}
			wholeAt[matrix.ID] = len(entry.Tables)
			whole = append(whole, matrix.ID)
		}
		entry.Tables = append(entry.Tables, table)
		entry.Index = append(entry.Index, index)
	}

	if len(whole) == 0 {
		return entry, nil
	}
	cells, err := t.matrices.ListTableCells(ctx, &ratematrixports.ListTableCellsRequest{
		TenantInfo:    tenant,
		RateMatrixIDs: whole,
	})
	if err != nil {
		return nil, err
	}
	for _, cell := range cells {
		at, ok := wholeAt[cell.RateMatrixID]
		if !ok {
			continue
		}
		entry.Tables[at].Cells = append(
			entry.Tables[at].Cells,
			encodeCell(&entry.Index[at], cell),
		)
	}
	return entry, nil
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
	matrixID pulid.ID,
	dimension *ratematrix.RateMatrixDimension,
) (ratetablecache.Axis, error) {
	axis := ratetablecache.Axis{
		Position:      dimension.Position,
		Kind:          string(dimension.Kind),
		Exact:         dimension.MatchMode == ratematrix.MatchModeExact,
		Normalization: string(dimension.KeyNormalization),
	}
	if !axis.Exact || !translatedKind(dimension.Kind) {
		return axis, nil
	}

	stored, err := t.matrices.ListAxisKeys(ctx, &ratematrixports.ListAxisKeysRequest{
		TenantInfo:   tenant,
		RateMatrixID: matrixID,
		Position:     dimension.Position,
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
	for _, key := range stored {
		name, ok := names[referenceID(dimension.Kind, key)]
		if !ok {
			continue
		}
		axis.Forward[key] = name
		normalized := normalize(&axis, name)
		axis.Reverse[normalized] = append(axis.Reverse[normalized], key)
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

func encodeCell(index *ratetablecache.Table, cell *ratematrix.RateMatrixCell) rules.TariffCell {
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
		if axis.Exact {
			out.At = append(
				out.At,
				rules.TariffCellPosition{Key: forward(axis, cell.KeyAt(axis.Position))},
			)
			continue
		}
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
	return out
}

func cellCombo(index *ratetablecache.Table, cell *rules.TariffCell) []string {
	combo := make([]string, 0, len(index.Axes))
	for i := range index.Axes {
		if index.Axes[i].Exact && i < len(cell.At) {
			combo = append(combo, normalize(&index.Axes[i], cell.At[i].Key))
		}
	}
	return combo
}

func (t *Tables) LoadCells(
	ctx context.Context,
	tenant pagination.TenantInfo,
	tariffCtx *rules.TariffContext,
	needs []rules.TariffNeed,
) error {
	state, err := t.load(ctx, tenant)
	if err != nil {
		return err
	}
	for table, combos := range groupNeeds(needs) {
		index, _, ok := state.entry.Table(table)
		if !ok {
			continue
		}
		at := tableAt(tariffCtx, table)
		if at < 0 {
			continue
		}
		cells, loadErr := t.cellsFor(ctx, tenant, state.stamp, index, combos)
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

func (t *Tables) LoadWhole(
	ctx context.Context,
	tenant pagination.TenantInfo,
	tariffCtx *rules.TariffContext,
	tables []string,
) error {
	state, err := t.load(ctx, tenant)
	if err != nil {
		return err
	}
	ids := make([]pulid.ID, 0, len(tables))
	indexes := make(map[pulid.ID]*ratetablecache.Table, len(tables))
	for _, code := range tables {
		index, _, ok := state.entry.Table(code)
		if !ok || tableAt(tariffCtx, code) < 0 {
			continue
		}
		ids = append(ids, index.MatrixID)
		indexes[index.MatrixID] = index
	}
	if len(ids) == 0 {
		return nil
	}
	cells, err := t.matrices.ListTableCells(ctx, &ratematrixports.ListTableCellsRequest{
		TenantInfo:    tenant,
		RateMatrixIDs: ids,
	})
	if err != nil {
		return err
	}
	byTable := make(map[string][]rules.TariffCell, len(ids))
	for _, cell := range cells {
		index, ok := indexes[cell.RateMatrixID]
		if !ok {
			continue
		}
		byTable[index.Code] = append(byTable[index.Code], encodeCell(index, cell))
	}
	for _, index := range indexes {
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
	stamp string,
	index *ratetablecache.Table,
	combos [][]string,
) ([]rules.TariffCell, error) {
	out := make([]rules.TariffCell, 0, len(combos))
	missing := make([][]string, 0, len(combos))
	for _, combo := range combos {
		if cached, ok := ratetablecache.Cells(
			tenant.OrgID,
			tenant.BuID,
			stamp,
			index.Code,
			combo,
		); ok {
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
			ratetablecache.StoreCells(tenant.OrgID, tenant.BuID, stamp, index.Code, combo, nil)
		}
		return out, nil
	}
	cells, err := t.matrices.LoadTableCells(ctx, &ratematrixports.LoadTableCellsRequest{
		TenantInfo:   tenant,
		RateMatrixID: index.MatrixID,
		Keys:         keys,
	})
	if err != nil {
		return nil, err
	}

	wanted := make(map[string][]rules.TariffCell, len(missing))
	for _, combo := range missing {
		wanted[strings.Join(combo, "\x1f")] = []rules.TariffCell{}
	}
	for _, cell := range cells {
		encoded := encodeCell(index, cell)
		key := strings.Join(cellCombo(index, &encoded), "\x1f")
		if bucket, ok := wanted[key]; ok {
			wanted[key] = append(bucket, encoded)
		}
	}
	for _, combo := range missing {
		found := wanted[strings.Join(combo, "\x1f")]
		ratetablecache.StoreCells(tenant.OrgID, tenant.BuID, stamp, index.Code, combo, found)
		out = append(out, found...)
	}
	return out, nil
}

func storedKeys(index *ratetablecache.Table, combos [][]string) map[int16][]string {
	keys := make(map[int16][]string, len(index.Axes))
	seen := make(map[int16]map[string]struct{}, len(index.Axes))
	exact := 0
	for i := range index.Axes {
		axis := &index.Axes[i]
		if !axis.Exact {
			continue
		}
		seen[axis.Position] = make(map[string]struct{}, len(combos))
		for _, combo := range combos {
			if exact >= len(combo) {
				continue
			}
			for _, stored := range reverse(axis, combo[exact]) {
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
		exact++
	}
	return keys
}

func groupNeeds(needs []rules.TariffNeed) map[string][][]string {
	grouped := make(map[string][][]string, len(needs))
	seen := make(map[string]struct{}, len(needs))
	for _, need := range needs {
		key := need.Table + "\x1e" + strings.Join(need.Keys, "\x1f")
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
