package shipmentrepository

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type copyDisposition int

const (
	copiedFromSource copyDisposition = iota
	setByCopy
	intentionallyDropped
)

type fieldPlan struct {
	disposition copyDisposition
	reason      string
}

var shipmentCopyPlan = map[string]fieldPlan{
	"ID":        {setByCopy, "a copy is a new row and gets a fresh identifier"},
	"Status":    {setByCopy, "every copy re-enters the lifecycle at New"},
	"ProNumber": {setByCopy, "the caller supplies the pro number through ShipmentCopySpec"},
	"BOL":       {setByCopy, "derived from the source BOL with a copy suffix"},
	"EnteredByID": {
		setByCopy,
		"attributed to the user who requested the duplicate, not the original enterer",
	},
	"Moves":             {setByCopy, "the move/stop graph is rebuilt with new identifiers"},
	"AdditionalCharges": {setByCopy, "charges are rebuilt against the new shipment id"},
	"Commodities":       {setByCopy, "commodities are rebuilt against the new shipment id"},

	"BusinessUnitID":    {copiedFromSource, "a copy stays inside the source tenant"},
	"OrganizationID":    {copiedFromSource, "a copy stays inside the source tenant"},
	"ServiceTypeID":     {copiedFromSource, "service classification carries over"},
	"ShipmentTypeID":    {copiedFromSource, "service classification carries over"},
	"CustomerID":        {copiedFromSource, "a copy bills the same customer"},
	"TractorTypeID":     {copiedFromSource, "equipment requirements carry over"},
	"TrailerTypeID":     {copiedFromSource, "equipment requirements carry over"},
	"FormulaTemplateID": {copiedFromSource, "the copy is rated by the same formula template"},
	"OtherChargeAmount": {
		copiedFromSource,
		"quoted charges carry over until the copy is re-rated",
	},
	"FreightChargeAmount": {
		copiedFromSource,
		"quoted charges carry over until the copy is re-rated",
	},
	"TotalChargeAmount": {
		copiedFromSource,
		"quoted charges carry over until the copy is re-rated",
	},
	"Pieces":         {copiedFromSource, "freight description carries over"},
	"Weight":         {copiedFromSource, "freight description carries over"},
	"TemperatureMin": {copiedFromSource, "handling requirements carry over"},
	"TemperatureMax": {copiedFromSource, "handling requirements carry over"},
	"RatingUnit":     {copiedFromSource, "the rating basis carries over"},

	"OwnerID":      {intentionallyDropped, "ownership is assigned by dispatch, never inherited"},
	"CanceledByID": {intentionallyDropped, "cancellation state belongs to the original"},
	"CanceledAt":   {intentionallyDropped, "cancellation state belongs to the original"},
	"CancelReason": {intentionallyDropped, "cancellation state belongs to the original"},
	"ConsolidationGroupID": {
		intentionallyDropped,
		"consolidation membership is earned per shipment",
	},
	"OrderID":      {intentionallyDropped, "the copy receives its own order from BuildAutoOrder"},
	"TenderStatus": {intentionallyDropped, "a copy has never been tendered"},
	"EntryMethod":  {intentionallyDropped, "left zero so the column default applies"},
	"BaseRate":     {intentionallyDropped, "recomputed when the copy is rated"},
	"EnvelopeLengthFeet": {
		intentionallyDropped,
		"dimensional envelope is recomputed from the copy's own stops",
	},
	"EnvelopeWidthFeet": {
		intentionallyDropped,
		"dimensional envelope is recomputed from the copy's own stops",
	},
	"EnvelopeHeightFeet": {
		intentionallyDropped,
		"dimensional envelope is recomputed from the copy's own stops",
	},
	"EnvelopeOverallHeightFeet": {
		intentionallyDropped,
		"dimensional envelope is recomputed from the copy's own stops",
	},
	"ActualDeliveryDate":     {intentionallyDropped, "actuals belong to the original execution"},
	"ActualShipDate":         {intentionallyDropped, "actuals belong to the original execution"},
	"BillingTransferStatus":  {intentionallyDropped, "the billing lifecycle restarts for a copy"},
	"TransferredToBillingAt": {intentionallyDropped, "the billing lifecycle restarts for a copy"},
	"MarkedReadyToBillAt":    {intentionallyDropped, "the billing lifecycle restarts for a copy"},
	"BilledAt":               {intentionallyDropped, "the billing lifecycle restarts for a copy"},
	"FuelSurchargeLocked":    {intentionallyDropped, "rate locks do not survive duplication"},
	"RateLocked":             {intentionallyDropped, "rate locks do not survive duplication"},
	"AutoRated":              {intentionallyDropped, "the copy has not been rated yet"},
	"AutoRatedAt":            {intentionallyDropped, "the copy has not been rated yet"},
	"RatingDetail": {
		intentionallyDropped,
		"the rating receipt is regenerated on re-rate",
	},
	"RateQuoteID": {
		intentionallyDropped,
		"a quote is consumed by the shipment that used it",
	},
	"RateAgreementID":     {intentionallyDropped, "an agreement prices a shipment exactly once"},
	"RateAgreementRuleID": {intentionallyDropped, "an agreement prices a shipment exactly once"},
	"RateOverrideAmount": {
		intentionallyDropped,
		"a manual override is re-authorized per shipment",
	},
	"RateOverrideReason": {
		intentionallyDropped,
		"a manual override is re-authorized per shipment",
	},
	"RateOverrideByID": {
		intentionallyDropped,
		"a manual override is re-authorized per shipment",
	},
	"RateOverrideAt": {
		intentionallyDropped,
		"a manual override is re-authorized per shipment",
	},
	"SourceDocumentID": {intentionallyDropped, "transient ingest field, not persisted"},
	"SearchVector":     {intentionallyDropped, "scanonly, produced by the database"},
	"Rank":             {intentionallyDropped, "scanonly, produced by the database"},
	"Version":          {intentionallyDropped, "optimistic-lock counter starts at zero on insert"},
	"CreatedAt":        {intentionallyDropped, "stamped by the database default on insert"},
	"UpdatedAt":        {intentionallyDropped, "stamped by the database default on insert"},

	"BusinessUnit":    {intentionallyDropped, "relation, hydrated on read"},
	"Organization":    {intentionallyDropped, "relation, hydrated on read"},
	"ShipmentType":    {intentionallyDropped, "relation, hydrated on read"},
	"ServiceType":     {intentionallyDropped, "relation, hydrated on read"},
	"Customer":        {intentionallyDropped, "relation, hydrated on read"},
	"TractorType":     {intentionallyDropped, "relation, hydrated on read"},
	"TrailerType":     {intentionallyDropped, "relation, hydrated on read"},
	"CanceledBy":      {intentionallyDropped, "relation, hydrated on read"},
	"Owner":           {intentionallyDropped, "relation, hydrated on read"},
	"EnteredBy":       {intentionallyDropped, "relation, hydrated on read"},
	"FormulaTemplate": {intentionallyDropped, "relation, hydrated on read"},
	"Comments":        {intentionallyDropped, "comments belong to the original shipment"},
}

func TestCopyShipmentGraph_EveryFieldIsClassified(t *testing.T) {
	t.Parallel()

	shipmentType := reflect.TypeOf(shipment.Shipment{})
	for i := range shipmentType.NumField() {
		field := shipmentType.Field(i)
		if field.Anonymous || !field.IsExported() {
			continue
		}
		_, ok := shipmentCopyPlan[field.Name]
		assert.Truef(
			t,
			ok,
			"shipment.Shipment.%s is not classified in shipmentCopyPlan. Decide whether "+
				"CopyShipmentGraph should carry it to a duplicate, then record that decision "+
				"in the plan with a reason.",
			field.Name,
		)
	}
}

func TestCopyShipmentGraph_PlanHasNoStaleEntries(t *testing.T) {
	t.Parallel()

	shipmentType := reflect.TypeOf(shipment.Shipment{})
	for name := range shipmentCopyPlan {
		_, ok := shipmentType.FieldByName(name)
		assert.Truef(
			t,
			ok,
			"shipmentCopyPlan lists %q, which no longer exists on shipment.Shipment",
			name,
		)
	}
}

func TestCopyShipmentGraph_HonorsFieldPlan(t *testing.T) {
	t.Parallel()

	source := duplicateSourceFixture()
	fillScalarFields(t, reflect.ValueOf(source).Elem())

	duplicated := CopyShipmentGraph(source, ShipmentCopySpec{
		ProNumber:   "PRO-COPY",
		BOL:         "COPY-BOL",
		RequestedBy: pulid.MustNew("usr_"),
	})

	sourceValue := reflect.ValueOf(source).Elem()
	copyValue := reflect.ValueOf(duplicated).Elem()
	shipmentType := sourceValue.Type()

	for i := range shipmentType.NumField() {
		field := shipmentType.Field(i)
		if field.Anonymous || !field.IsExported() || isRelationField(field) {
			continue
		}
		plan, ok := shipmentCopyPlan[field.Name]
		if !ok {
			continue
		}

		sourceField := sourceValue.Field(i)
		copyField := copyValue.Field(i)

		switch plan.disposition {
		case copiedFromSource:
			assert.Truef(
				t,
				reflect.DeepEqual(sourceField.Interface(), copyField.Interface()),
				"%s is planned as copied (%s) but CopyShipmentGraph did not carry it over",
				field.Name,
				plan.reason,
			)
			if sourceField.Kind() == reflect.Pointer && !sourceField.IsNil() {
				assert.NotEqualf(
					t,
					sourceField.Pointer(),
					copyField.Pointer(),
					"%s shares a pointer with the source; mutating the duplicate would "+
						"corrupt the original",
					field.Name,
				)
			}
		case setByCopy:
			assert.Falsef(
				t,
				copyField.IsZero(),
				"%s is planned as set by the copy (%s) but came out zero",
				field.Name,
				plan.reason,
			)
			assert.Falsef(
				t,
				reflect.DeepEqual(sourceField.Interface(), copyField.Interface()),
				"%s is planned as set by the copy (%s) but matched the source value",
				field.Name,
				plan.reason,
			)
		case intentionallyDropped:
			assert.Truef(
				t,
				copyField.IsZero(),
				"%s is planned as dropped (%s) but CopyShipmentGraph carried a value over",
				field.Name,
				plan.reason,
			)
		}
	}
}

func isRelationField(field reflect.StructField) bool {
	return strings.HasPrefix(field.Tag.Get("bun"), "rel:")
}

func fillScalarFields(t *testing.T, target reflect.Value) {
	t.Helper()

	targetType := target.Type()
	seed := 0
	for i := range targetType.NumField() {
		field := targetType.Field(i)
		if field.Anonymous || !field.IsExported() || isRelationField(field) {
			continue
		}
		seed++
		fillNonZero(t, target.Field(i), seed)
	}
}

func fillNonZero(t *testing.T, target reflect.Value, seed int) {
	t.Helper()

	switch target.Type() {
	case reflect.TypeOf(decimal.NullDecimal{}):
		target.Set(reflect.ValueOf(decimal.NewNullDecimal(decimal.NewFromInt(int64(seed)))))

		return
	case reflect.TypeOf(&shipment.RatingDetail{}):
		target.Set(reflect.ValueOf(&shipment.RatingDetail{
			FormulaTemplateName: "fixture",
			Result:              float64(seed),
			RatedAt:             int64(seed),
		}))

		return
	}

	switch target.Kind() {
	case reflect.String:
		target.SetString(fmt.Sprintf("fixture-%d", seed))
	case reflect.Bool:
		target.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		target.SetInt(int64(seed) + 1)
	case reflect.Float32, reflect.Float64:
		target.SetFloat(float64(seed) + 1.5)
	case reflect.Pointer:
		target.Set(reflect.New(target.Type().Elem()))
		fillNonZero(t, target.Elem(), seed)
	default:
		require.Failf(
			t,
			"unfillable field",
			"fillNonZero has no strategy for %s; extend it so the duplication plan stays enforced",
			target.Type(),
		)
	}
}
