package resolver

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/shopspring/decimal"
)

var (
	ErrNilPointer = errors.New("nil pointer")
	ErrNotStruct  = errors.New("not a struct")
)

func RegisterDefaultComputed(r *Resolver) {
	r.RegisterComputed("computeTotalDistance", computeTotalDistance)
	r.RegisterComputed("computeTotalStops", computeTotalStops)
	r.RegisterComputed("computeHasHazmat", computeHasHazmat)
	r.RegisterComputed("computeRequiresTemperatureControl", computeRequiresTemperatureControl)
	r.RegisterComputed("computeTemperatureDifferential", computeTemperatureDifferential)
	r.RegisterComputed("computeTotalWeight", computeTotalWeight)
	r.RegisterComputed("computeTotalPieces", computeTotalPieces)
	r.RegisterComputed("computeTotalLinearFeet", computeTotalLinearFeet)
	r.RegisterComputed("computeFreightChargeAmount", computeFreightChargeAmount)
	r.RegisterComputed("computeOtherChargeAmount", computeOtherChargeAmount)
	r.RegisterComputed("computeCurrentTotalCharge", computeCurrentTotalCharge)
}

func computeTotalDistance(entity any) (any, error) {
	moves, err := getFieldSlice(entity, "Moves")
	if err != nil {
		return 0.0, err
	}

	var total float64
	for _, move := range moves {
		if distance, distanceErr := getFieldFloat64(move, "Distance"); distanceErr == nil {
			total += distance
		}
	}

	return total, nil
}

func computeTotalStops(entity any) (any, error) {
	moves, err := getFieldSlice(entity, "Moves")
	if err != nil {
		return 0, err
	}

	var total int
	for _, move := range moves {
		stops, stopsErr := getFieldSlice(move, "Stops")
		if stopsErr == nil {
			total += len(stops)
		}
	}

	return total, nil
}

func computeHasHazmat(entity any) (any, error) {
	commodities, err := getFieldSlice(entity, "Commodities")
	if err != nil {
		return false, err
	}

	for _, sc := range commodities {
		commodity, commErr := getFieldValue(sc, "Commodity")
		if commErr != nil || commodity == nil {
			continue
		}

		hazmat, hazmatErr := getFieldValue(commodity, "HazardousMaterial")
		if hazmatErr == nil && hazmat != nil && !isNilInterface(hazmat) {
			return true, nil
		}
	}

	return false, nil
}

func computeRequiresTemperatureControl(entity any) (any, error) {
	tempMin, errMin := getFieldValue(entity, "TemperatureMin")
	tempMax, errMax := getFieldValue(entity, "TemperatureMax")

	hasMin := errMin == nil && tempMin != nil && !isNilInterface(tempMin)
	hasMax := errMax == nil && tempMax != nil && !isNilInterface(tempMax)

	return hasMin || hasMax, nil
}

func computeTemperatureDifferential(entity any) (any, error) {
	tempMin, errMin := getFieldInt16(entity, "TemperatureMin")
	tempMax, errMax := getFieldInt16(entity, "TemperatureMax")

	if errMin != nil || errMax != nil {
		return 0.0, errMin
	}

	return float64(tempMax - tempMin), nil
}

func computeTotalWeight(entity any) (any, error) {
	weight, err := getFieldInt64(entity, "Weight")
	if err == nil && weight > 0 {
		return float64(weight), nil
	}

	commodities, err := getFieldSlice(entity, "Commodities")
	if err != nil {
		return 0.0, err
	}

	var total int64
	for _, sc := range commodities {
		if w, wErr := getFieldInt64(sc, "Weight"); wErr == nil {
			total += w
		}
	}

	return float64(total), nil
}

func computeTotalPieces(entity any) (any, error) {
	pieces, err := getFieldInt64(entity, "Pieces")
	if err == nil && pieces > 0 {
		return pieces, nil
	}

	commodities, err := getFieldSlice(entity, "Commodities")
	if err != nil {
		return int64(0), err
	}

	var total int64
	for _, sc := range commodities {
		if p, pErr := getFieldInt64(sc, "Pieces"); pErr == nil {
			total += p
		}
	}

	return total, nil
}

func computeTotalLinearFeet(entity any) (any, error) {
	commodities, err := getFieldSlice(entity, "Commodities")
	if err != nil {
		if errors.Is(err, ErrNotStruct) {
			return 0.0, nil
		}
		return 0.0, err
	}

	var total float64
	for _, sc := range commodities {
		pieces, pErr := getFieldInt64(sc, "Pieces")
		if pErr != nil || pieces == 0 {
			continue
		}

		commodity, commErr := getFieldValue(sc, "Commodity")
		if commErr != nil || commodity == nil || isNilInterface(commodity) {
			continue
		}

		linearFeetPerUnit, lfErr := getFieldFloat64(commodity, "LinearFeetPerUnit")
		if lfErr != nil {
			continue
		}

		total += float64(pieces) * linearFeetPerUnit
	}

	return total, nil
}

func getFieldValue(entity any, fieldName string) (any, error) {
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, ErrNilPointer
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, ErrNotStruct
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return nil, fmt.Errorf("field %s not found", fieldName)
	}

	return field.Interface(), nil
}

func getFieldSlice(entity any, fieldName string) ([]any, error) {
	val, err := getFieldValue(entity, fieldName)
	if err != nil {
		return nil, err
	}

	v := reflect.ValueOf(val)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("field %s is not a slice", fieldName)
	}

	result := make([]any, v.Len())
	for i := 0; i < v.Len(); i++ {
		result[i] = v.Index(i).Interface()
	}

	return result, nil
}

func getFieldFloat64(entity any, fieldName string) (float64, error) {
	val, err := getFieldValue(entity, fieldName)
	if err != nil {
		return 0, err
	}

	switch v := val.(type) {
	case float64:
		return v, nil
	case *float64:
		if v == nil {
			return 0, nil
		}
		return *v, nil
	case float32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", val)
	}
}

func getFieldInt64(entity any, fieldName string) (int64, error) {
	val, err := getFieldValue(entity, fieldName)
	if err != nil {
		return 0, err
	}

	switch v := val.(type) {
	case int64:
		return v, nil
	case *int64:
		if v == nil {
			return 0, ErrNilPointer
		}
		return *v, nil
	case int:
		return int64(v), nil
	case *int:
		if v == nil {
			return 0, ErrNilPointer
		}
		return int64(*v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", val)
	}
}

func getFieldInt16(entity any, fieldName string) (int16, error) {
	val, err := getFieldValue(entity, fieldName)
	if err != nil {
		return 0, err
	}

	switch v := val.(type) {
	case int16:
		return v, nil
	case *int16:
		if v == nil {
			return 0, ErrNilPointer
		}
		return *v, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int16", val)
	}
}

func isNilInterface(i any) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	return v.Kind() == reflect.Pointer && v.IsNil()
}

func computeFreightChargeAmount(entity any) (any, error) {
	return getFieldDecimal(entity, "FreightChargeAmount")
}

func computeOtherChargeAmount(entity any) (any, error) {
	switch typed := entity.(type) {
	case *shipment.Shipment:
		return calculateShipmentOtherChargeAmount(typed), nil
	case shipment.Shipment:
		return calculateShipmentOtherChargeAmount(&typed), nil
	}

	return getFieldDecimal(entity, "OtherChargeAmount")
}

func computeCurrentTotalCharge(entity any) (any, error) {
	return getFieldDecimal(entity, "TotalChargeAmount")
}

func calculateShipmentOtherChargeAmount(entity *shipment.Shipment) float64 {
	if entity == nil {
		return 0
	}

	baseCharge := decimal.Zero
	if entity.FreightChargeAmount.Valid {
		baseCharge = entity.FreightChargeAmount.Decimal
	}

	total := decimal.Zero
	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}

		switch charge.Method {
		case accessorialcharge.MethodFlat:
			unit := max(charge.Unit, 1)
			total = total.Add(charge.Amount.Mul(decimal.NewFromInt32(int32(unit))))
		case accessorialcharge.MethodPerUnit:
			if charge.Unit < 1 {
				continue
			}
			total = total.Add(charge.Amount.Mul(decimal.NewFromInt32(int32(charge.Unit))))
		case accessorialcharge.MethodPercentage:
			total = total.Add(baseCharge.Mul(charge.Amount.Div(decimal.NewFromInt(100))))
		}
	}

	return total.InexactFloat64()
}

func getFieldDecimal(entity any, fieldName string) (float64, error) {
	val, err := getFieldValue(entity, fieldName)
	if err != nil {
		return 0, err
	}

	if val == nil || isNilInterface(val) {
		return 0, nil
	}

	v := reflect.ValueOf(val)

	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return 0, nil
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		if isNullableInvalid(v) {
			return 0, nil
		}

		if f, ok := tryDecimalFieldFloat(v); ok {
			return f, nil
		}

		if f, ok := tryInexactFloat64Method(v); ok {
			return f, nil
		}
	}

	switch typedVal := val.(type) {
	case float64:
		return typedVal, nil
	case *float64:
		if typedVal == nil {
			return 0, nil
		}
		return *typedVal, nil
	}

	return 0, nil
}

func isNullableInvalid(v reflect.Value) bool {
	validField := v.FieldByName("Valid")
	if !validField.IsValid() || validField.Kind() != reflect.Bool {
		return false
	}
	return !validField.Bool()
}

func tryDecimalFieldFloat(v reflect.Value) (float64, bool) {
	decimalField := v.FieldByName("Decimal")
	if !decimalField.IsValid() {
		return 0, false
	}

	method := decimalField.MethodByName("InexactFloat64")
	if !method.IsValid() {
		return 0, false
	}

	results := method.Call(nil)
	if len(results) == 0 {
		return 0, false
	}

	return results[0].Float(), true
}

func tryInexactFloat64Method(v reflect.Value) (float64, bool) {
	method := v.MethodByName("InexactFloat64")
	if !method.IsValid() {
		return 0, false
	}

	results := method.Call(nil)
	if len(results) == 0 {
		return 0, false
	}

	return results[0].Float(), true
}
