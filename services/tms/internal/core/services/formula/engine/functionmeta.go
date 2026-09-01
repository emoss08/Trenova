package engine

type FunctionSpec struct {
	Name        string `json:"name"`
	Signature   string `json:"signature"`
	Description string `json:"description"`
	Example     string `json:"example"`
	Category    string `json:"category"`

	fn    func(...any) (any, error)
	types []any
}

const (
	FunctionCategoryMath        = "math"
	FunctionCategoryRounding    = "rounding"
	FunctionCategoryAggregate   = "aggregate"
	FunctionCategoryConditional = "conditional"
	FunctionCategoryRateTable   = "rateTable"
)

func FunctionSpecs() []FunctionSpec {
	specs := make([]FunctionSpec, len(functionSpecs))
	copy(specs, functionSpecs)
	return specs
}

var functionSpecs = []FunctionSpec{
	{
		Name:        "round",
		Signature:   "round(x, decimals?)",
		Description: "Rounds a number to the given decimal places (default 0, between -12 and 12).",
		Example:     "round(3.14159, 2) // 3.14",
		Category:    FunctionCategoryRounding,
		fn:          roundFn,
		types: []any{
			new(func(float64) float64),
			new(func(float64, int) float64),
		},
	},
	{
		Name:        "ceil",
		Signature:   "ceil(x)",
		Description: "Returns the smallest integer greater than or equal to x.",
		Example:     "ceil(3.1) // 4",
		Category:    FunctionCategoryRounding,
		fn:          ceilFn,
		types:       []any{new(func(float64) float64)},
	},
	{
		Name:        "floor",
		Signature:   "floor(x)",
		Description: "Returns the largest integer less than or equal to x.",
		Example:     "floor(3.9) // 3",
		Category:    FunctionCategoryRounding,
		fn:          floorFn,
		types:       []any{new(func(float64) float64)},
	},
	{
		Name:        "abs",
		Signature:   "abs(x)",
		Description: "Returns the absolute value of a number.",
		Example:     "abs(-5) // 5",
		Category:    FunctionCategoryMath,
		fn:          absFn,
		types:       []any{new(func(float64) float64)},
	},
	{
		Name:        "min",
		Signature:   "min(a, b)",
		Description: "Returns the smaller of two numbers.",
		Example:     "min(baseRate, 500) // caps at 500",
		Category:    FunctionCategoryMath,
		fn:          minFn,
		types:       []any{new(func(float64, float64) float64)},
	},
	{
		Name:        "max",
		Signature:   "max(a, b)",
		Description: "Returns the larger of two numbers.",
		Example:     "max(baseRate * totalDistance, 250) // floor of 250",
		Category:    FunctionCategoryMath,
		fn:          maxFn,
		types:       []any{new(func(float64, float64) float64)},
	},
	{
		Name:        "sum",
		Signature:   "sum(...values)",
		Description: "Returns the sum of all numeric arguments.",
		Example:     "sum(100, 50, 25) // 175",
		Category:    FunctionCategoryAggregate,
		fn:          sumFn,
		types:       []any{new(func(...float64) float64)},
	},
	{
		Name:        "avg",
		Signature:   "avg(...values)",
		Description: "Returns the average of all numeric arguments.",
		Example:     "avg(100, 200, 300) // 200",
		Category:    FunctionCategoryAggregate,
		fn:          avgFn,
		types:       []any{new(func(...float64) float64)},
	},
	{
		Name:        "coalesce",
		Signature:   "coalesce(...values)",
		Description: "Returns the first non-null argument. Zero and empty string count as values.",
		Example:     "coalesce(customRate, baseRate, 0)",
		Category:    FunctionCategoryConditional,
		fn:          coalesceFn,
		types:       []any{new(func(...any) any)},
	},
	{
		Name:        "clamp",
		Signature:   "clamp(value, min, max)",
		Description: "Constrains a value between a minimum and maximum bound.",
		Example:     "clamp(rate, 100, 1000)",
		Category:    FunctionCategoryMath,
		fn:          clampFn,
		types:       []any{new(func(float64, float64, float64) float64)},
	},
	{
		Name:        "pow",
		Signature:   "pow(base, exponent)",
		Description: "Returns base raised to the power of exponent.",
		Example:     "pow(1.05, years)",
		Category:    FunctionCategoryMath,
		fn:          powFn,
		types:       []any{new(func(float64, float64) float64)},
	},
	{
		Name:        "sqrt",
		Signature:   "sqrt(x)",
		Description: "Returns the square root of a non-negative number.",
		Example:     "sqrt(16) // 4",
		Category:    FunctionCategoryMath,
		fn:          sqrtFn,
		types:       []any{new(func(float64) float64)},
	},
	{
		Name:        "lookup",
		Signature:   "lookup(table, key)",
		Description: "Returns the value for key in the active single-axis rate matrix named table. Errors when the table or entry is missing.",
		Example:     `baseRate * (1 + lookup("fuel_surcharge", fuelPrice))`,
		Category:    FunctionCategoryRateTable,
	},
	{
		Name:        "lookupOr",
		Signature:   "lookupOr(table, key, fallback)",
		Description: "Like lookup, but returns fallback when the table has no entry or band for the key. A missing table still errors, so a deleted matrix cannot silently reprice.",
		Example:     `lookupOr("lane_rate", laneCode, 0)`,
		Category:    FunctionCategoryRateTable,
	},
	{
		Name:        "lookup2",
		Signature:   "lookup2(table, rowKey, colKey)",
		Description: "Returns the value at the row/column intersection of the active two-axis rate matrix named table. Each axis matches by its own mode (exact key or quantity band). Errors when the table or intersection is missing.",
		Example:     `totalWeight * lookup2("class_rates", destination.state, totalWeight)`,
		Category:    FunctionCategoryRateTable,
	},
	{
		Name:        "lookup2Or",
		Signature:   "lookup2Or(table, rowKey, colKey, fallback)",
		Description: "Like lookup2, but returns fallback when the table has no cell at the row/column intersection. A missing table still errors, so a deleted matrix cannot silently reprice.",
		Example:     `lookup2Or("zone_weight_rates", origin.zip, totalWeight, 0)`,
		Category:    FunctionCategoryRateTable,
	},
}
