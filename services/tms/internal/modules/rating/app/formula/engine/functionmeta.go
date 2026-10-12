package engine

type FunctionSpec struct {
	Name        string `json:"name"`
	Signature   string `json:"signature"`
	Description string `json:"description"`
	Example     string `json:"example"`
	Category    string `json:"category"`
	Operator    bool   `json:"operator"`

	fn    func(...any) (any, error)
	types []any
}

// Executable reports whether Trenova supplies the implementation. Specs
// without one document something expr already ships, so the reference pane
// and linter know it exists without the engine registering it twice.
func (s FunctionSpec) Executable() bool {
	return s.fn != nil
}

const (
	FunctionCategoryMath        = "math"
	FunctionCategoryRounding    = "rounding"
	FunctionCategoryAggregate   = "aggregate"
	FunctionCategoryConditional = "conditional"
	FunctionCategoryRateTable   = "rateTable"
	FunctionCategoryString      = "string"
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
		Name:        "roundUp",
		Signature:   "roundUp(x, decimals?)",
		Description: "Rounds a number up (toward positive infinity) at the given decimal places; roundUp(x, 2) is the cent-level ceiling a tariff means by \"rounded up to the nearest cent\".",
		Example:     "roundUp(2.001, 2) // 2.01",
		Category:    FunctionCategoryRounding,
		fn:          roundUpFn,
		types: []any{
			new(func(float64) float64),
			new(func(float64, int) float64),
		},
	},
	{
		Name:        "roundDown",
		Signature:   "roundDown(x, decimals?)",
		Description: "Rounds a number down (toward negative infinity) at the given decimal places.",
		Example:     "roundDown(2.999, 2) // 2.99",
		Category:    FunctionCategoryRounding,
		fn:          roundDownFn,
		types: []any{
			new(func(float64) float64),
			new(func(float64, int) float64),
		},
	},
	{
		Name:        "roundHalfEven",
		Signature:   "roundHalfEven(x, decimals?)",
		Description: "Banker's rounding: exact halves go to the even neighbour, so a large book of charges does not drift upward.",
		Example:     "roundHalfEven(2.665, 2) // 2.66",
		Category:    FunctionCategoryRounding,
		fn:          roundHalfEvenFn,
		types: []any{
			new(func(float64) float64),
			new(func(float64, int) float64),
		},
	},
	{
		Name:        "roundTo",
		Signature:   "roundTo(x, increment)",
		Description: "Rounds to the nearest multiple of increment: roundTo(x, 5) prices to the nearest $5, roundTo(x, 0.25) to the nearest quarter.",
		Example:     "roundTo(123.4, 5) // 125",
		Category:    FunctionCategoryRounding,
		fn:          roundToFn,
		types:       []any{new(func(float64, float64) float64)},
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
		Signature:   "min(...values)",
		Description: "Returns the smallest of one or more numbers.",
		Example:     "min(baseRate, 500) // caps at 500",
		Category:    FunctionCategoryMath,
		fn:          minFn,
		types:       []any{new(func(...float64) float64)},
	},
	{
		Name:        "max",
		Signature:   "max(...values)",
		Description: "Returns the largest of one or more numbers.",
		Example:     "max(baseRate * totalDistance, 250, minimumCharge) // highest floor wins",
		Category:    FunctionCategoryMath,
		fn:          maxFn,
		types:       []any{new(func(...float64) float64)},
	},
	{
		Name:        "sum",
		Signature:   "sum(...values)",
		Description: "Returns the sum of numbers and lists of numbers, so it works over a mapped collection such as map(stops, .weight).",
		Example:     "sum(map(commodities, .weight)) // total commodity weight",
		Category:    FunctionCategoryAggregate,
		fn:          sumFn,
		types:       []any{new(func(...any) float64)},
	},
	{
		Name:        "avg",
		Signature:   "avg(...values)",
		Description: "Returns the average of numbers and lists of numbers; every element counts once.",
		Example:     "avg(map(commodities, .density)) // mean pounds per cubic foot",
		Category:    FunctionCategoryAggregate,
		fn:          avgFn,
		types:       []any{new(func(...any) float64)},
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
	{
		Name:        "lookupInterp",
		Signature:   "lookupInterp(table, key)",
		Description: "Reads a banded single-axis table as a curve: between two band floors the value is interpolated linearly between those bands' values. Below the first floor or past the last, the edge band's value is used.",
		Example:     `baseRate * (1 + lookupInterp("fuel_curve", fuelPrice))`,
		Category:    FunctionCategoryRateTable,
	},
	{
		Name:        "deficitWeight",
		Signature:   "deficitWeight(table, weight)",
		Description: "The weight to bill under a per-unit break table: the actual weight, unless rating the next break's minimum at the next break's rate is cheaper. Pair it with lookup on the same table.",
		Example:     `deficitWeight("cwt", totalWeight) / 100 * lookup("cwt", deficitWeight("cwt", totalWeight))`,
		Category:    FunctionCategoryRateTable,
	},
	{
		Name:        "startsWith",
		Signature:   `text startsWith "prefix"`,
		Description: "True when the text begins with the prefix. Case-sensitive.",
		Example:     `origin.zip startsWith "7" ? 1.10 : 1.00`,
		Category:    FunctionCategoryString,
		Operator:    true,
	},
	{
		Name:        "endsWith",
		Signature:   `text endsWith "suffix"`,
		Description: "True when the text ends with the suffix. Case-sensitive.",
		Example:     `customer.code endsWith "-EXP" ? 75 : 0`,
		Category:    FunctionCategoryString,
		Operator:    true,
	},
	{
		Name:        "contains",
		Signature:   `text contains "part"`,
		Description: "True when the text includes the part anywhere. Case-sensitive; wrap both sides in lower() to ignore case.",
		Example:     `lower(customer.name) contains "hospital" ? 50 : 0`,
		Category:    FunctionCategoryString,
		Operator:    true,
	},
	{
		Name:        "matches",
		Signature:   `text matches "pattern"`,
		Description: "True when the text matches the regular expression. Anchor with ^ and $ to match the whole value.",
		Example:     `origin.zip matches "^(75|76)" ? 25 : 0`,
		Category:    FunctionCategoryString,
		Operator:    true,
	},
	{
		Name:        "[start:end]",
		Signature:   "text[start:end]",
		Description: "Slices characters from start up to but not including end. Either bound may be left out; negative bounds count from the end.",
		Example:     `origin.zip[0:3] == "750" ? 1.05 : 1.00`,
		Category:    FunctionCategoryString,
		Operator:    true,
	},
	{
		Name:        "upper",
		Signature:   "upper(text)",
		Description: "Converts text to upper case.",
		Example:     `upper(origin.state) == "TX"`,
		Category:    FunctionCategoryString,
	},
	{
		Name:        "lower",
		Signature:   "lower(text)",
		Description: "Converts text to lower case.",
		Example:     `lower(customer.name) contains "clinic"`,
		Category:    FunctionCategoryString,
	},
	{
		Name:        "trim",
		Signature:   "trim(text)",
		Description: "Removes leading and trailing whitespace.",
		Example:     `trim(customer.code) == "ACME"`,
		Category:    FunctionCategoryString,
	},
	{
		Name:        "trimPrefix",
		Signature:   `trimPrefix(text, "prefix")`,
		Description: "Removes the prefix when the text begins with it.",
		Example:     `trimPrefix(customer.code, "C-")`,
		Category:    FunctionCategoryString,
	},
	{
		Name:        "trimSuffix",
		Signature:   `trimSuffix(text, "suffix")`,
		Description: "Removes the suffix when the text ends with it.",
		Example:     `trimSuffix(origin.zip, "-0000")`,
		Category:    FunctionCategoryString,
	},
	{
		Name:        "hasPrefix",
		Signature:   `hasPrefix(text, "prefix")`,
		Description: "Function form of startsWith.",
		Example:     `hasPrefix(origin.zip, "7")`,
		Category:    FunctionCategoryString,
	},
	{
		Name:        "hasSuffix",
		Signature:   `hasSuffix(text, "suffix")`,
		Description: "Function form of endsWith.",
		Example:     `hasSuffix(customer.code, "-EXP")`,
		Category:    FunctionCategoryString,
	},
	{
		Name:        "indexOf",
		Signature:   `indexOf(text, "part")`,
		Description: "Position of the first occurrence of part, or -1 when absent.",
		Example:     `indexOf(customer.code, "-") > 0`,
		Category:    FunctionCategoryString,
	},
	{
		Name:        "replace",
		Signature:   `replace(text, "old", "new")`,
		Description: "Replaces every occurrence of old with new.",
		Example:     `replace(origin.zip, "-", "")`,
		Category:    FunctionCategoryString,
	},
	{
		Name:        "split",
		Signature:   `split(text, "separator")`,
		Description: "Splits text into a list on the separator.",
		Example:     `split(customer.code, "-")[0]`,
		Category:    FunctionCategoryString,
	},
	{
		Name:        "len",
		Signature:   "len(value)",
		Description: "Number of characters in text or items in a list.",
		Example:     `len(origin.zip) == 5`,
		Category:    FunctionCategoryString,
	},
	{
		Name:        "string",
		Signature:   "string(value)",
		Description: "Converts a number or boolean to text.",
		Example:     `string(totalPieces) + " pcs"`,
		Category:    FunctionCategoryString,
	},
	{
		Name:        "float",
		Signature:   "float(value)",
		Description: "Converts numeric text or an integer to a number.",
		Example:     `float(trim(customer.creditTier))`,
		Category:    FunctionCategoryString,
	},
	{
		Name:        "int",
		Signature:   "int(value)",
		Description: "Converts numeric text or a number to a whole number, truncating toward zero.",
		Example:     `int(origin.zip[0:1])`,
		Category:    FunctionCategoryString,
	},
}
