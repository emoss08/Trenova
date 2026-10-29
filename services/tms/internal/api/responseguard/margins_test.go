package responseguard

import (
	"fmt"
	"go/types"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// marginName is the conservative pattern a JSON field that may carry a margin
// is named by, the same one the GraphQL schema is held to in gqlext.
var marginName = regexp.MustCompile(`(?i)margin|profit`)

// restMargin is one Go type a REST response carries a margin in: a value with
// every margin figure set, and the JSON paths WithholdFields must clear for a
// caller without the margins group. A path reaches into a struct the type
// carries (trace.totals.margin) where that struct is withheld by its carrier.
type restMargin struct {
	build func() permission.FieldWithholder
	paths []string
}

func margin(value string) decimal.NullDecimal {
	return decimal.NewNullDecimal(decimal.RequireFromString(value))
}

var detentionPay = []string{"driverPayAmount", "netMargin"}

func restMargins() []restMargin {
	return []restMargin{
		{
			build: func() permission.FieldWithholder {
				return &detention.DetentionOccurrence{
					DriverPayAmount: margin("75"),
					NetMargin:       margin("150"),
				}
			},
			paths: detentionPay,
		},
		{
			build: func() permission.FieldWithholder {
				return &detentionports.CustomerDetentionStat{
					DriverPayAmount: margin("75"),
					NetMargin:       margin("150"),
				}
			},
			paths: detentionPay,
		},
		{
			build: func() permission.FieldWithholder {
				return &detentionports.FacilityDetentionStat{
					DriverPayAmount: margin("75"),
					NetMargin:       margin("150"),
				}
			},
			paths: detentionPay,
		},
		{
			build: func() permission.FieldWithholder {
				return &detentionservice.PreviewResult{
					DriverPayAmount: margin("75"),
					NetMargin:       margin("150"),
				}
			},
			paths: detentionPay,
		},
		{
			build: func() permission.FieldWithholder {
				return &detentionservice.BacktestBucket{
					DriverPayAmount: margin("75"),
					NetMargin:       margin("150"),
				}
			},
			paths: detentionPay,
		},
		{
			build: func() permission.FieldWithholder {
				stops := 2
				return &detentionservice.BacktestResult{
					ProposedDriverPay:   margin("75"),
					ProposedNetMargin:   margin("150"),
					NegativeMarginStops: &stops,
				}
			},
			paths: []string{"proposedDriverPay", "proposedNetMargin", "negativeMarginStops"},
		},
		{
			build: func() permission.FieldWithholder {
				return &dispatchcontrol.DispatchControl{
					TenderCounterMarginFloorPercent: margin("10"),
				}
			},
			paths: []string{"tenderCounterMarginFloorPercent"},
		},
		{
			build: func() permission.FieldWithholder {
				return &costingcontrol.CostingControl{TargetMarginPercent: margin("15")}
			},
			paths: []string{"targetMarginPercent"},
		},
		{
			build: func() permission.FieldWithholder {
				return &tender.TenderOffer{CounterMargin: margin("8.5")}
			},
			paths: []string{"counterMargin"},
		},
		{
			build: func() permission.FieldWithholder {
				return &rateagreement.RateAgreement{
					MarginFloorPercent:  margin("12"),
					MaxPayPercentOfSell: margin("85"),
				}
			},
			paths: []string{"marginFloorPercent", "maxPayPercentOfSell"},
		},
		{
			build: func() permission.FieldWithholder {
				return &rateagreement.RateAgreementVersion{
					MarginFloorPercent:  margin("12"),
					MaxPayPercentOfSell: margin("85"),
				}
			},
			paths: []string{"marginFloorPercent", "maxPayPercentOfSell"},
		},
		{
			build: func() permission.FieldWithholder {
				return &ratequote.RateQuote{
					CostAmount:    margin("800"),
					MarginAmount:  margin("200"),
					MarginPercent: margin("20"),
					Trace: &ratetypes.Trace{Totals: ratetypes.Totals{
						Total:     decimal.NewFromInt(1000),
						Cost:      margin("800"),
						Margin:    margin("200"),
						MarginPct: margin("20"),
					}},
				}
			},
			paths: []string{
				"costAmount", "marginAmount", "marginPercent",
				"trace.totals.cost", "trace.totals.margin", "trace.totals.marginPercent",
			},
		},
		{
			build: func() permission.FieldWithholder {
				return &ratequoteports.ShopOption{
					Margin: &ratetypes.MarginVerdict{
						Amount:      decimal.NewFromInt(200),
						Percent:     decimal.NewFromInt(20),
						BelowFloor:  true,
						Explanation: "margin of 20% is below the 25% this contract requires",
					},
					Note: "margin of 20% is below the 25% this contract requires",
				}
			},
			paths: []string{"margin", "note"},
		},
		{
			build: func() permission.FieldWithholder {
				return &ratequoteports.ShopResult{SellTotal: margin("1000")}
			},
			paths: []string{"sellTotal"},
		},
	}
}

// carriedMargins are types a REST response reaches a margin in that do not
// withhold it themselves, each with the type that carries and withholds it.
var carriedMargins = map[string]string{
	"github.com/emoss08/trenova/pkg/ratetypes.Totals": "github.com/emoss08/trenova/" +
		"internal/modules/rating/domain/ratequote.RateQuote",
}

// unprotectedRESTMarginNames are the JSON fields REST responses carry whose
// names match marginName and that carry no margin, each with why.
var unprotectedRESTMarginNames = map[string]string{
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol.BillingControl." +
		"enforceMarginFloor": "a switch that turns the margin floor check on, not a figure",
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glbalanceports." +
		"GLIncomeStatement.grossProfitMinor": "a line of the income statement, the posted " +
		"ledger's revenue less its cost of sales beside the very lines it totals; it is " +
		"gated by the general ledger's own read, not by the margins group, which covers " +
		"pricing and operational margins",
}

func typeKey(value any) string {
	t := reflect.TypeOf(value)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.PkgPath() + "." + t.Name()
}

// marginFieldsResponsesReach walks every REST response type and returns each
// JSON field named for a margin, keyed type.field, with one route that
// reaches it.
func marginFieldsResponsesReach(t *testing.T) map[string]string {
	t.Helper()

	found := make(map[string]string)
	seen := make(map[types.Type]struct{})
	var walk func(typ types.Type, via string)
	walk = func(typ types.Type, via string) {
		switch u := types.Unalias(typ).(type) {
		case *types.Pointer:
			walk(u.Elem(), via)
		case *types.Slice:
			walk(u.Elem(), via)
		case *types.Array:
			walk(u.Elem(), via)
		case *types.Map:
			walk(u.Elem(), via)
		case *types.Named:
			st, ok := u.Underlying().(*types.Struct)
			if !ok || u.Obj().Pkg() == nil {
				return
			}
			if _, done := seen[u]; done {
				return
			}
			seen[u] = struct{}{}
			owner := u.Obj().Pkg().Path() + "." + u.Obj().Name()
			for i := range st.NumFields() {
				f := st.Field(i)
				tag := reflect.StructTag(st.Tag(i))
				if tag.Get("json") == "-" || (!f.Exported() && !f.Embedded()) {
					continue
				}
				name, _ := parseTag(tag.Get("json"))
				if name == "" {
					name = f.Name()
				}
				if marginName.MatchString(name) {
					if _, known := found[owner+"."+name]; !known {
						found[owner+"."+name] = via
					}
				}
				walk(f.Type(), via+" > "+u.Obj().Name()+"."+name)
			}
		}
	}

	positions := make([]string, 0, len(responseTypes(t)))
	for position := range responseTypes(t) {
		positions = append(positions, position)
	}
	slices.Sort(positions)
	for _, position := range positions {
		walk(responseTypes(t)[position], shortPosition(position))
	}

	return found
}

// Every JSON field a REST response carries that is named for a margin is
// withheld from a caller without the margins group by the entity that holds
// it, or by the record that carries it, unless it is listed as no margin.
func TestEveryMarginFieldARESTResponseCarriesIsWithheld(t *testing.T) {
	withheld := make(map[string][]string)
	for _, entry := range restMargins() {
		withheld[typeKey(entry.build())] = entry.paths
	}

	found := marginFieldsResponsesReach(t)
	var missing []string
	for key, via := range found {
		if _, listed := unprotectedRESTMarginNames[key]; listed {
			continue
		}
		i := strings.LastIndex(key, ".")
		owner, field := key[:i], key[i+1:]
		if slices.Contains(withheld[owner], field) {
			continue
		}
		if carrier, carried := carriedMargins[owner]; carried &&
			slices.ContainsFunc(withheld[carrier], func(path string) bool {
				return strings.HasSuffix(path, "."+field)
			}) {
			continue
		}
		missing = append(missing, fmt.Sprintf("%s (reached at %s)", key, via))
	}
	slices.Sort(missing)
	require.Empty(t, missing,
		"\nthese REST fields are named for a margin but nothing withholds them: give the "+
			"type WithholdFields clearing the field for a caller without "+
			"permission.FieldGroupMargins and add it to restMargins, or list it in "+
			"unprotectedRESTMarginNames with why it carries no margin:\n"+
			strings.Join(missing, "\n"))

	for key := range unprotectedRESTMarginNames {
		_, ok := found[key]
		assert.Truef(t, ok, "%s is listed in unprotectedRESTMarginNames but no REST "+
			"response carries it; remove it", key)
	}
}

func TestEveryRESTMarginIsClearedWithoutTheMarginsGroupAndKeptWithIt(t *testing.T) {
	t.Parallel()

	for _, entry := range restMargins() {
		key := typeKey(entry.build())

		kept := entry.build()
		kept.WithholdFields(permission.AllFieldGroups())
		cleared := entry.build()
		cleared.WithholdFields(permission.AllFieldGroups() &^
			(permission.FieldGroupSet(0).With(permission.FieldGroupMargins)))

		keptJSON := encode(t, kept)
		clearedJSON := encode(t, cleared)
		for _, path := range entry.paths {
			assert.NotEmptyf(t, valueAt(keptJSON, path),
				"%s.%s was cleared for a caller who holds the margins group", key, path)
			assert.Emptyf(t, valueAt(clearedJSON, path),
				"%s.%s reached a caller without the margins group", key, path)
		}
	}
}

func shopResponse() *ratequoteports.ShopResult {
	verdict := ratetypes.MarginVerdict{
		Amount:      decimal.NewFromInt(200),
		Percent:     decimal.NewFromInt(20),
		Explanation: "margin of 20% is below the 25% this contract requires",
	}
	return &ratequoteports.ShopResult{
		Strategy:  ratequoteports.ShopStrategyLeastCost,
		SellTotal: margin("1000"),
		Options: []*ratequoteports.ShopOption{{
			CarrierName: "Swift Lines",
			Cost:        decimal.NewFromInt(800),
			Margin:      &verdict,
			Note:        verdict.Explanation,
			Quote: &ratequote.RateQuote{
				TotalAmount:  decimal.NewFromInt(800),
				MarginAmount: margin("200"),
			},
		}},
	}
}

func TestAShopResultShowsItsMarginsOnlyToTheMarginsGroup(t *testing.T) {
	t.Parallel()

	body, _ := serve(t, []permission.Resource{permission.ResourceProtectedMargins},
		shopResponse(), guarded)
	assert.Nil(t, body["sellTotal"])
	option := object(t, body, "options", 0)
	assert.Nil(t, option["margin"])
	assert.Empty(t, option["note"])
	assert.Equal(t, "800", option["cost"], "the carrier's cost is what shopping is for")
	assert.Nil(t, object(t, option, "quote")["marginAmount"])

	body, _ = serve(t, nil, shopResponse(), guarded)
	assert.Equal(t, "1000", body["sellTotal"])
	option = object(t, body, "options", 0)
	assert.Equal(t, "200", object(t, option, "margin")["amount"])
	assert.Equal(t, "200", object(t, option, "quote")["marginAmount"])
}

func TestADetentionOccurrenceWithholdsTheDriversPayFromItsTrace(t *testing.T) {
	t.Parallel()

	pay := decimal.NewFromInt(75)
	occurrence := func() *detention.DetentionOccurrence {
		return &detention.DetentionOccurrence{
			BillableAmount:  decimal.NewFromInt(225),
			DriverPayAmount: decimal.NewNullDecimal(pay),
			NetMargin:       margin("150"),
			CalculationTrace: &detention.CalculationTrace{Steps: []detention.TraceStep{
				{Kind: detention.TraceStepDriverPay, Label: "Driver detention pay", Amount: &pay},
			}},
		}
	}

	body, _ := serve(t, []permission.Resource{permission.ResourceProtectedMargins},
		occurrence(), guarded)
	assert.Equal(t, "225", body["billableAmount"])
	assert.Nil(t, body["driverPayAmount"])
	assert.Nil(t, body["netMargin"])
	assert.Nil(t, object(t, body, "calculationTrace", "steps", 0)["amount"])

	body, _ = serve(t, nil, occurrence(), guarded)
	assert.Equal(t, "75", body["driverPayAmount"])
	assert.Equal(t, "75", object(t, body, "calculationTrace", "steps", 0)["amount"])
}

func encode(t *testing.T, value any) map[string]any {
	t.Helper()

	raw, err := sonic.Marshal(value)
	require.NoError(t, err)
	out := map[string]any{}
	require.NoError(t, sonic.Unmarshal(raw, &out))
	return out
}

func valueAt(doc map[string]any, path string) any {
	var current any = doc
	for segment := range strings.SplitSeq(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = object[segment]
	}
	if text, ok := current.(string); ok && text == "" {
		return nil
	}

	return current
}
