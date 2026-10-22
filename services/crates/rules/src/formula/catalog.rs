//! Every function and word operator the formula language has.
//!
//! This is the single list of names: the parser asks it which calls take a predicate, the
//! checker takes each function's arity, parameter kinds and result type from it, the evaluator
//! implements every entry that is not an operator, and `formula.catalog` publishes it for the Go
//! schema endpoint (`GET /formula-templates/schema`) to republish unchanged next to a record's
//! variables.
//!
//! The entries are split the way the old engine published them: `functions` are the ones the
//! Formula Studio's reference lists (rounding, aggregates, rate tables, text), `builtins` the
//! list, predicate, bit and date functions. Both carry a description and an example now, since
//! the editor shows them in hover cards and signature help. The disabled names are functions
//! formulas must not call, each with the reason and, where there is one, the replacement;
//! reserved names are the ones a template input may not take.

use crate::message;
use crate::message::Arg;
use crate::message::Message;

/// What a function gives back, as the checker types it.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Returns {
    /// A number.
    Number,
    /// A whole number.
    Integer,
    /// Text.
    String,
    /// True or false.
    Boolean,
    /// A date.
    Date,
    /// A duration.
    Duration,
    /// A list.
    Array,
    /// A record.
    Object,
    /// Anything, depending on the arguments.
    Any,
}

impl Returns {
    /// The type's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Number => "Number",
            Self::Integer => "Integer",
            Self::String => "String",
            Self::Boolean => "Boolean",
            Self::Date => "Date",
            Self::Duration => "Duration",
            Self::Array => "Array",
            Self::Object => "Object",
            Self::Any => "Any",
        }
    }
}

/// The kind of value a parameter takes, as the checker holds an argument to it.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Param {
    /// Anything.
    Any,
    /// A number.
    Number,
    /// A whole number.
    Integer,
    /// Text.
    Text,
    /// A list.
    List,
    /// A record.
    Record,
    /// Text, a list or a record: something with a length.
    Sized,
    /// A number or a list of numbers, flattened one level.
    Aggregate,
    /// Text or a number, converted.
    Convertible,
    /// A rate table's name, written as a string literal.
    Table,
    /// A predicate, worked out per item.
    Predicate,
}

/// The reference pane's grouping.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Category {
    /// Arithmetic.
    Math,
    /// Rounding.
    Rounding,
    /// Sums, averages and the like.
    Aggregate,
    /// Choosing between values.
    Conditional,
    /// Rate table lookups.
    RateTable,
    /// Text.
    String,
    /// Lists and records.
    List,
    /// Functions that take a predicate.
    Predicate,
    /// Bit operations on whole numbers.
    Bits,
    /// Dates and durations.
    Date,
}

impl Category {
    /// The category's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Math => "math",
            Self::Rounding => "rounding",
            Self::Aggregate => "aggregate",
            Self::Conditional => "conditional",
            Self::RateTable => "rateTable",
            Self::String => "string",
            Self::List => "list",
            Self::Predicate => "predicate",
            Self::Bits => "bits",
            Self::Date => "date",
        }
    }
}

/// One function or word operator.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Function {
    /// The name it is called by.
    pub name: &'static str,
    /// How it is written, with its parameters.
    pub signature: &'static str,
    /// What it does: an English message template.
    pub description: &'static str,
    /// An example: an English message template.
    pub example: &'static str,
    /// The reference pane's grouping.
    pub category: Category,
    /// Written infix (`text startsWith "x"`) or as a slice, not called.
    pub operator: bool,
    /// The fewest arguments it takes.
    pub min_args: u8,
    /// The most arguments it takes; `None` for any number.
    pub max_args: Option<u8>,
    /// What it gives back.
    pub returns: Returns,
    /// Every argument is a number.
    pub numeric_args: bool,
    /// Its second argument is a predicate, worked out per item.
    pub predicate: bool,
    /// The kind each argument takes, in order; when it takes any number of arguments the last
    /// kind repeats.
    pub params: &'static [Param],
    /// Listed among the documented `functions` rather than the `builtins`.
    pub documented: bool,
}

impl Function {
    /// The kind of the argument at `index`.
    #[must_use]
    pub fn param(&self, index: usize) -> Param {
        self.params
            .get(index)
            .or_else(|| {
                if self.max_args.is_none() {
                    self.params.last()
                } else {
                    None
                }
            })
            .copied()
            .unwrap_or(Param::Any)
    }

    /// Whether it takes `count` arguments.
    #[must_use]
    pub fn takes(&self, count: usize) -> bool {
        count >= usize::from(self.min_args)
            && self.max_args.is_none_or(|max| count <= usize::from(max))
    }

    /// It reads a rate table.
    #[must_use]
    pub fn reads_table(&self) -> bool {
        self.category == Category::RateTable
    }
}

/// Why a function is turned off.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum DisabledReason {
    /// It would give a different answer each run.
    Nondeterministic,
    /// It serialises or parses JSON.
    Serialization,
    /// It builds text.
    TextBuilding,
}

impl DisabledReason {
    /// The reason's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Nondeterministic => "nondeterministic",
            Self::Serialization => "serialization",
            Self::TextBuilding => "textBuilding",
        }
    }
}

/// A function formulas must not call.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Disabled {
    /// Its name.
    pub name: &'static str,
    /// Why it is off.
    pub reason: DisabledReason,
    /// The name to use instead, when there is one.
    pub replacement: Option<&'static str>,
}

impl Disabled {
    /// What the checker says when a formula calls it.
    #[must_use]
    pub fn message(&self) -> Message {
        let called = format!("`{}()`", self.name);
        match self.reason {
            DisabledReason::Nondeterministic => Message::with(
                message!(
                    "{0} isn't available in formulas: a formula must price a shipment the same way every time it runs. Use {1}, the date the contract terms are read at"
                ),
                [
                    Arg::from(called),
                    Arg::from(format!("`{}`", self.replacement.unwrap_or(RATING_TIME))),
                ],
            ),
            DisabledReason::TextBuilding => Message::with(
                message!(
                    "{0} isn't available in formulas: formulas compare codes and do not build text"
                ),
                [Arg::from(format!("`{}`", self.name))],
            ),
            DisabledReason::Serialization => Message::with(
                message!(
                    "{0} isn't available in formulas: formulas read the shipment's fields directly and do not convert them to or from JSON"
                ),
                [Arg::from(format!("`{}`", self.name))],
            ),
        }
    }
}

/// The variable that stands in for "now": the date the contract terms are read at.
pub const RATING_TIME: &str = "ratingTime";

const N: Param = Param::Number;
const I: Param = Param::Integer;
const T: Param = Param::Text;
const L: Param = Param::List;
const A: Param = Param::Any;
const P: Param = Param::Predicate;

#[allow(clippy::too_many_arguments)]
const fn entry(
    name: &'static str,
    signature: &'static str,
    description: &'static str,
    example: &'static str,
    category: Category,
    (min_args, max_args): (u8, Option<u8>),
    returns: Returns,
    params: &'static [Param],
) -> Function {
    Function {
        name,
        signature,
        description,
        example,
        category,
        operator: false,
        min_args,
        max_args,
        returns,
        numeric_args: false,
        predicate: false,
        params,
        documented: true,
    }
}

const fn numeric(function: Function) -> Function {
    Function {
        numeric_args: true,
        ..function
    }
}

const fn operator(function: Function) -> Function {
    Function {
        operator: true,
        ..function
    }
}

const fn builtin(function: Function) -> Function {
    Function {
        documented: false,
        ..function
    }
}

const fn predicate(function: Function) -> Function {
    Function {
        predicate: true,
        documented: false,
        ..function
    }
}

const fn exactly(count: u8) -> (u8, Option<u8>) {
    (count, Some(count))
}

const fn between(min: u8, max: u8) -> (u8, Option<u8>) {
    (min, Some(max))
}

const fn at_least(min: u8) -> (u8, Option<u8>) {
    (min, None)
}

use Category as C;
use Returns as R;

/// Every function and word operator, documented ones first, in the order the reference lists
/// them.
pub const FUNCTIONS: &[Function] = &[
    numeric(entry(
        "round",
        "round(x, decimals?)",
        message!("Rounds a number to the given decimal places (default 0, between -12 and 12)."),
        message!("round(3.14159, 2) // 3.14"),
        C::Rounding,
        between(1, 2),
        R::Number,
        &[N, N],
    )),
    numeric(entry(
        "roundUp",
        "roundUp(x, decimals?)",
        message!(
            "Rounds a number up (toward positive infinity) at the given decimal places; roundUp(x, 2) is the cent-level ceiling a tariff means by \"rounded up to the nearest cent\"."
        ),
        message!("roundUp(2.001, 2) // 2.01"),
        C::Rounding,
        between(1, 2),
        R::Number,
        &[N, N],
    )),
    numeric(entry(
        "roundDown",
        "roundDown(x, decimals?)",
        message!("Rounds a number down (toward negative infinity) at the given decimal places."),
        message!("roundDown(2.999, 2) // 2.99"),
        C::Rounding,
        between(1, 2),
        R::Number,
        &[N, N],
    )),
    numeric(entry(
        "roundHalfEven",
        "roundHalfEven(x, decimals?)",
        message!(
            "Banker's rounding: exact halves go to the even neighbour, so a large book of charges does not drift upward."
        ),
        message!("roundHalfEven(2.665, 2) // 2.66"),
        C::Rounding,
        between(1, 2),
        R::Number,
        &[N, N],
    )),
    numeric(entry(
        "roundTo",
        "roundTo(x, increment)",
        message!(
            "Rounds to the nearest multiple of increment: roundTo(x, 5) prices to the nearest $5, roundTo(x, 0.25) to the nearest quarter."
        ),
        message!("roundTo(123.4, 5) // 125"),
        C::Rounding,
        exactly(2),
        R::Number,
        &[N, N],
    )),
    numeric(entry(
        "ceil",
        "ceil(x)",
        message!("Returns the smallest integer greater than or equal to x."),
        message!("ceil(3.1) // 4"),
        C::Rounding,
        exactly(1),
        R::Number,
        &[N],
    )),
    numeric(entry(
        "floor",
        "floor(x)",
        message!("Returns the largest integer less than or equal to x."),
        message!("floor(3.9) // 3"),
        C::Rounding,
        exactly(1),
        R::Number,
        &[N],
    )),
    numeric(entry(
        "abs",
        "abs(x)",
        message!("Returns the absolute value of a number."),
        message!("abs(-5) // 5"),
        C::Math,
        exactly(1),
        R::Number,
        &[N],
    )),
    entry(
        "min",
        "min(...values)",
        message!(
            "Returns the smallest of one or more numbers; a list counts as its numbers, and the first of equal values wins."
        ),
        message!("min(baseRate, 500) // caps at 500"),
        C::Math,
        at_least(1),
        R::Number,
        &[Param::Aggregate],
    ),
    entry(
        "max",
        "max(...values)",
        message!(
            "Returns the largest of one or more numbers; a list counts as its numbers, and the first of equal values wins."
        ),
        message!("max(baseRate * totalDistance, 250, minimumCharge) // highest floor wins"),
        C::Math,
        at_least(1),
        R::Number,
        &[Param::Aggregate],
    ),
    entry(
        "sum",
        "sum(...values)",
        message!(
            "Returns the sum of numbers and lists of numbers, so it works over a mapped collection such as map(stops, .weight)."
        ),
        message!("sum(map(commodities, .weight)) // total commodity weight"),
        C::Aggregate,
        at_least(1),
        R::Number,
        &[Param::Aggregate],
    ),
    entry(
        "avg",
        "avg(...values)",
        message!("Returns the average of numbers and lists of numbers; every element counts once."),
        message!("avg(map(commodities, .density)) // mean pounds per cubic foot"),
        C::Aggregate,
        at_least(1),
        R::Number,
        &[Param::Aggregate],
    ),
    entry(
        "coalesce",
        "coalesce(...values)",
        message!("Returns the first non-null argument. Zero and empty string count as values."),
        message!("coalesce(customRate, baseRate, 0)"),
        C::Conditional,
        at_least(1),
        R::Any,
        &[A],
    ),
    numeric(entry(
        "clamp",
        "clamp(value, min, max)",
        message!("Constrains a value between a minimum and maximum bound."),
        message!("clamp(rate, 100, 1000)"),
        C::Math,
        exactly(3),
        R::Number,
        &[N, N, N],
    )),
    numeric(entry(
        "pow",
        "pow(base, exponent)",
        message!(
            "Returns base raised to the power of exponent: exact for a whole exponent, correct to the last digit otherwise."
        ),
        message!("pow(1.05, years)"),
        C::Math,
        exactly(2),
        R::Number,
        &[N, N],
    )),
    numeric(entry(
        "sqrt",
        "sqrt(x)",
        message!("Returns the square root of a non-negative number."),
        message!("sqrt(16) // 4"),
        C::Math,
        exactly(1),
        R::Number,
        &[N],
    )),
    entry(
        "lookup",
        "lookup(table, key)",
        message!(
            "Returns the value for key in the active single-axis rate matrix named table. Errors when the table or entry is missing."
        ),
        message!("baseRate * (1 + lookup(\"fuel_surcharge\", fuelPrice))"),
        C::RateTable,
        exactly(2),
        R::Number,
        &[Param::Table, A],
    ),
    entry(
        "lookupOr",
        "lookupOr(table, key, fallback)",
        message!(
            "Like lookup, but returns fallback when the table has no entry or band for the key. A missing table still errors, so a deleted matrix cannot silently reprice."
        ),
        message!("lookupOr(\"lane_rate\", laneCode, 0)"),
        C::RateTable,
        exactly(3),
        R::Number,
        &[Param::Table, A, N],
    ),
    entry(
        "lookup2",
        "lookup2(table, rowKey, colKey)",
        message!(
            "Returns the value at the row/column intersection of the active two-axis rate matrix named table. Each axis matches by its own mode (exact key or quantity band). Errors when the table or intersection is missing."
        ),
        message!("totalWeight * lookup2(\"class_rates\", destination.state, totalWeight)"),
        C::RateTable,
        exactly(3),
        R::Number,
        &[Param::Table, A, A],
    ),
    entry(
        "lookup2Or",
        "lookup2Or(table, rowKey, colKey, fallback)",
        message!(
            "Like lookup2, but returns fallback when the table has no cell at the row/column intersection. A missing table still errors, so a deleted matrix cannot silently reprice."
        ),
        message!("lookup2Or(\"zone_weight_rates\", origin.zip, totalWeight, 0)"),
        C::RateTable,
        exactly(4),
        R::Number,
        &[Param::Table, A, A, N],
    ),
    entry(
        "lookupInterp",
        "lookupInterp(table, key)",
        message!(
            "Reads a banded single-axis table as a curve: between two band floors the value is interpolated linearly between those bands' values. Below the first floor or past the last, the edge band's value is used."
        ),
        message!("baseRate * (1 + lookupInterp(\"fuel_curve\", fuelPrice))"),
        C::RateTable,
        exactly(2),
        R::Number,
        &[Param::Table, N],
    ),
    entry(
        "deficitWeight",
        "deficitWeight(table, weight)",
        message!(
            "The weight to bill under a per-unit break table: the actual weight, unless rating the next break's minimum at the next break's rate is cheaper. Pair it with lookup on the same table."
        ),
        message!(
            "deficitWeight(\"cwt\", totalWeight) / 100 * lookup(\"cwt\", deficitWeight(\"cwt\", totalWeight))"
        ),
        C::RateTable,
        exactly(2),
        R::Number,
        &[Param::Table, N],
    ),
    operator(entry(
        "startsWith",
        "text startsWith \"prefix\"",
        message!("True when the text begins with the prefix. Case-sensitive."),
        message!("origin.zip startsWith \"7\" ? 1.10 : 1.00"),
        C::String,
        exactly(2),
        R::Boolean,
        &[T, T],
    )),
    operator(entry(
        "endsWith",
        "text endsWith \"suffix\"",
        message!("True when the text ends with the suffix. Case-sensitive."),
        message!("customer.code endsWith \"-EXP\" ? 75 : 0"),
        C::String,
        exactly(2),
        R::Boolean,
        &[T, T],
    )),
    operator(entry(
        "contains",
        "text contains \"part\"",
        message!(
            "True when the text includes the part anywhere. Case-sensitive; wrap both sides in lower() to ignore case."
        ),
        message!("lower(customer.name) contains \"hospital\" ? 50 : 0"),
        C::String,
        exactly(2),
        R::Boolean,
        &[T, T],
    )),
    operator(entry(
        "matches",
        "text matches \"pattern\"",
        message!(
            "True when the text matches the regular expression. Anchor with ^ and $ to match the whole value."
        ),
        message!("origin.zip matches \"^(75|76)\" ? 25 : 0"),
        C::String,
        exactly(2),
        R::Boolean,
        &[T, T],
    )),
    operator(entry(
        "[start:end]",
        "text[start:end]",
        message!(
            "Slices characters from start up to but not including end. Either bound may be left out; negative bounds count from the end."
        ),
        message!("origin.zip[0:3] == \"750\" ? 1.05 : 1.00"),
        C::String,
        between(1, 3),
        R::Any,
        &[A, I, I],
    )),
    entry(
        "upper",
        "upper(text)",
        message!("Converts text to upper case."),
        message!("upper(origin.state) == \"TX\""),
        C::String,
        exactly(1),
        R::String,
        &[T],
    ),
    entry(
        "lower",
        "lower(text)",
        message!("Converts text to lower case."),
        message!("lower(customer.name) contains \"clinic\""),
        C::String,
        exactly(1),
        R::String,
        &[T],
    ),
    entry(
        "trim",
        "trim(text)",
        message!("Removes leading and trailing whitespace."),
        message!("trim(customer.code) == \"ACME\""),
        C::String,
        between(1, 2),
        R::String,
        &[T, T],
    ),
    entry(
        "trimPrefix",
        "trimPrefix(text, \"prefix\")",
        message!("Removes the prefix when the text begins with it."),
        message!("trimPrefix(customer.code, \"C-\")"),
        C::String,
        between(1, 2),
        R::String,
        &[T, T],
    ),
    entry(
        "trimSuffix",
        "trimSuffix(text, \"suffix\")",
        message!("Removes the suffix when the text ends with it."),
        message!("trimSuffix(origin.zip, \"-0000\")"),
        C::String,
        between(1, 2),
        R::String,
        &[T, T],
    ),
    entry(
        "hasPrefix",
        "hasPrefix(text, \"prefix\")",
        message!("Function form of startsWith."),
        message!("hasPrefix(origin.zip, \"7\")"),
        C::String,
        exactly(2),
        R::Boolean,
        &[T, T],
    ),
    entry(
        "hasSuffix",
        "hasSuffix(text, \"suffix\")",
        message!("Function form of endsWith."),
        message!("hasSuffix(customer.code, \"-EXP\")"),
        C::String,
        exactly(2),
        R::Boolean,
        &[T, T],
    ),
    entry(
        "indexOf",
        "indexOf(text, \"part\")",
        message!("Position of the first occurrence of part, or -1 when absent."),
        message!("indexOf(customer.code, \"-\") > 0"),
        C::String,
        exactly(2),
        R::Integer,
        &[T, T],
    ),
    entry(
        "replace",
        "replace(text, \"old\", \"new\")",
        message!("Replaces every occurrence of old with new."),
        message!("replace(origin.zip, \"-\", \"\")"),
        C::String,
        between(3, 4),
        R::String,
        &[T, T, T, I],
    ),
    entry(
        "split",
        "split(text, \"separator\")",
        message!("Splits text into a list on the separator."),
        message!("split(customer.code, \"-\")[0]"),
        C::String,
        between(2, 3),
        R::Array,
        &[T, T, I],
    ),
    entry(
        "len",
        "len(value)",
        message!("Number of characters in text or items in a list."),
        message!("len(origin.zip) == 5"),
        C::String,
        exactly(1),
        R::Integer,
        &[Param::Sized],
    ),
    entry(
        "string",
        "string(value)",
        message!("Converts a number or boolean to text."),
        message!("string(totalPieces) + \" pcs\""),
        C::String,
        exactly(1),
        R::String,
        &[A],
    ),
    entry(
        "float",
        "float(value)",
        message!("Converts numeric text or an integer to a number."),
        message!("float(trim(customer.creditTier))"),
        C::String,
        exactly(1),
        R::Number,
        &[Param::Convertible],
    ),
    entry(
        "int",
        "int(value)",
        message!("Converts numeric text or a number to a whole number, truncating toward zero."),
        message!("int(origin.zip[0:1])"),
        C::String,
        exactly(1),
        R::Integer,
        &[Param::Convertible],
    ),
    predicate(entry(
        "all",
        "all(list, predicate)",
        message!("True when the predicate holds for every item; true for an empty list."),
        message!("all(stops, .status == \"Completed\")"),
        C::Predicate,
        exactly(2),
        R::Boolean,
        &[L, P],
    )),
    predicate(entry(
        "none",
        "none(list, predicate)",
        message!("True when the predicate holds for no item."),
        message!("none(commodities, .hazmat) ? 0 : hazmatFee"),
        C::Predicate,
        exactly(2),
        R::Boolean,
        &[L, P],
    )),
    predicate(entry(
        "any",
        "any(list, predicate)",
        message!("True when the predicate holds for at least one item."),
        message!("any(commodities, .freightClass == \"85\") ? 25 : 0"),
        C::Predicate,
        exactly(2),
        R::Boolean,
        &[L, P],
    )),
    predicate(entry(
        "one",
        "one(list, predicate)",
        message!("True when the predicate holds for exactly one item."),
        message!("one(stops, .type == \"Delivery\") ? 0 : 50"),
        C::Predicate,
        exactly(2),
        R::Boolean,
        &[L, P],
    )),
    predicate(entry(
        "filter",
        "filter(list, predicate)",
        message!("The items the predicate holds for, in order."),
        message!("len(filter(stops, .type == \"Delivery\"))"),
        C::Predicate,
        exactly(2),
        R::Array,
        &[L, P],
    )),
    predicate(entry(
        "map",
        "map(list, predicate)",
        message!("The predicate's value for each item, in order."),
        message!("sum(map(commodities, .weight ?? 0))"),
        C::Predicate,
        exactly(2),
        R::Array,
        &[L, P],
    )),
    predicate(entry(
        "find",
        "find(list, predicate)",
        message!("The first item the predicate holds for, or empty when none does."),
        message!("find(stops, .type == \"Delivery\")?.status"),
        C::Predicate,
        exactly(2),
        R::Any,
        &[L, P],
    )),
    predicate(entry(
        "findIndex",
        "findIndex(list, predicate)",
        message!("The position of the first item the predicate holds for, or -1."),
        message!("findIndex(stops, .type == \"Delivery\")"),
        C::Predicate,
        exactly(2),
        R::Integer,
        &[L, P],
    )),
    predicate(entry(
        "findLast",
        "findLast(list, predicate)",
        message!("The last item the predicate holds for, or empty when none does."),
        message!("findLast(stops, .type == \"Pickup\")?.status"),
        C::Predicate,
        exactly(2),
        R::Any,
        &[L, P],
    )),
    predicate(entry(
        "findLastIndex",
        "findLastIndex(list, predicate)",
        message!("The position of the last item the predicate holds for, or -1."),
        message!("findLastIndex(stops, .type == \"Pickup\")"),
        C::Predicate,
        exactly(2),
        R::Integer,
        &[L, P],
    )),
    predicate(entry(
        "count",
        "count(list, predicate?)",
        message!(
            "How many items the predicate holds for; without a predicate, how many items are true."
        ),
        message!("count(stops, .type == \"Delivery\") * stopFee"),
        C::Predicate,
        between(1, 2),
        R::Integer,
        &[L, P],
    )),
    predicate(entry(
        "groupBy",
        "groupBy(list, predicate)",
        message!("A record of lists: the items grouped by the predicate's value."),
        message!("len(keys(groupBy(stops, .type)))"),
        C::Predicate,
        exactly(2),
        R::Object,
        &[L, P],
    )),
    predicate(entry(
        "sortBy",
        "sortBy(list, predicate, order?)",
        message!("The items ordered by the predicate's value, \"asc\" (the default) or \"desc\"."),
        message!("first(sortBy(commodities, .weight, \"desc\")).weight"),
        C::Predicate,
        between(2, 3),
        R::Array,
        &[L, P, T],
    )),
    predicate(entry(
        "reduce",
        "reduce(list, predicate, initial?)",
        message!(
            "Folds the list into one value: the predicate sees the running value as #acc and the item as #; initial is the first #acc (the first item when left out)."
        ),
        message!("reduce(map(stops, .weight ?? 0), #acc + #, 0)"),
        C::Predicate,
        between(2, 3),
        R::Any,
        &[L, P, A],
    )),
    builtin(entry(
        "type",
        "type(value)",
        message!(
            "The name of a value's kind: \"int\", \"float\", \"string\", \"bool\", \"array\", \"map\", \"time.Time\", \"time.Duration\" or \"nil\"."
        ),
        message!("type(fuelPrice) == \"nil\" ? 0 : fuelPrice"),
        C::String,
        exactly(1),
        R::String,
        &[A],
    )),
    builtin(entry(
        "splitAfter",
        "splitAfter(text, \"separator\", n?)",
        message!(
            "Splits text after each separator, keeping it on the piece before; n caps the number of pieces."
        ),
        message!("splitAfter(customer.code, \"-\")[0]"),
        C::String,
        between(2, 3),
        R::Array,
        &[T, T, I],
    )),
    builtin(entry(
        "join",
        "join(list, \"separator\"?)",
        message!(
            "Joins a list of text into one text, with the separator (none by default) between items."
        ),
        message!("join([origin.state, destination.state], \"-\") == \"TX-OK\""),
        C::String,
        between(1, 2),
        R::String,
        &[L, T],
    )),
    builtin(entry(
        "lastIndexOf",
        "lastIndexOf(text, \"part\")",
        message!("Position of the last occurrence of part, or -1 when absent."),
        message!("lastIndexOf(customer.code, \"-\")"),
        C::String,
        exactly(2),
        R::Integer,
        &[T, T],
    )),
    builtin(entry(
        "mean",
        "mean(...values)",
        message!("The average of numbers and lists of numbers; the same as avg."),
        message!("mean(map(commodities, .weight ?? 0))"),
        C::Aggregate,
        at_least(1),
        R::Number,
        &[Param::Aggregate],
    )),
    builtin(entry(
        "median",
        "median(...values)",
        message!(
            "The middle of numbers and lists of numbers once sorted; the average of the two middle values for an even count."
        ),
        message!("median(map(commodities, .weight ?? 0))"),
        C::Aggregate,
        at_least(1),
        R::Number,
        &[Param::Aggregate],
    )),
    builtin(entry(
        "duration",
        "duration(\"1h30m\")",
        message!(
            "A length of time from text such as \"90m\" or \"1h30m\", in the units ns, us, ms, s, m and h. Add it to or subtract it from a date."
        ),
        message!("deliveryDate - pickupDate > duration(\"48h\") ? 150 : 0"),
        C::Date,
        exactly(1),
        R::Duration,
        &[T],
    )),
    builtin(entry(
        "date",
        "date(text)",
        message!(
            "A date from text: RFC 3339 (\"2026-01-02T15:04:05-05:00\"), \"2026-01-02T15:04:05\", \"2026-01-02T15:04\", \"2026-01-02 15:04\" or \"2026-01-02\". Text without an offset is UTC."
        ),
        message!("ratingTime >= date(\"2026-07-01\") ? 1.05 : 1.00"),
        C::Date,
        exactly(1),
        R::Date,
        &[T],
    )),
    builtin(entry(
        "first",
        "first(list)",
        message!("The first item of a list, or empty when it has none."),
        message!("first(stops)?.type"),
        C::List,
        exactly(1),
        R::Any,
        &[L],
    )),
    builtin(entry(
        "last",
        "last(list)",
        message!("The last item of a list, or empty when it has none."),
        message!("last(stops)?.status"),
        C::List,
        exactly(1),
        R::Any,
        &[L],
    )),
    builtin(entry(
        "get",
        "get(value, key)",
        message!(
            "The item at a position of a list or the value at a key of a record, or empty when there is none, where value[key] would fail."
        ),
        message!("get(stops, 5) == nil ? 0 : 75"),
        C::List,
        exactly(2),
        R::Any,
        &[A, A],
    )),
    builtin(entry(
        "take",
        "take(list, n)",
        message!("The first n items of a list."),
        message!("sum(take(map(commodities, .weight ?? 0), 3))"),
        C::List,
        exactly(2),
        R::Array,
        &[L, I],
    )),
    builtin(entry(
        "keys",
        "keys(record)",
        message!("A record's keys, sorted."),
        message!("len(keys(groupBy(stops, .type)))"),
        C::List,
        exactly(1),
        R::Array,
        &[Param::Record],
    )),
    builtin(entry(
        "values",
        "values(record)",
        message!("A record's values, in the order of its sorted keys."),
        message!("len(values(groupBy(stops, .type)))"),
        C::List,
        exactly(1),
        R::Array,
        &[Param::Record],
    )),
    builtin(entry(
        "toPairs",
        "toPairs(record)",
        message!("A record as a list of [key, value] pairs, sorted by key."),
        message!("len(toPairs(groupBy(stops, .type)))"),
        C::List,
        exactly(1),
        R::Array,
        &[Param::Record],
    )),
    builtin(entry(
        "fromPairs",
        "fromPairs(list)",
        message!("A record from a list of [key, value] pairs."),
        message!("fromPairs([[\"TX\", 1.1], [\"OK\", 1.0]])[origin.state] ?? 1"),
        C::List,
        exactly(1),
        R::Object,
        &[L],
    )),
    builtin(entry(
        "reverse",
        "reverse(list)",
        message!("A list in reverse order."),
        message!("first(reverse(stops))?.type"),
        C::List,
        exactly(1),
        R::Array,
        &[L],
    )),
    builtin(entry(
        "uniq",
        "uniq(list)",
        message!("A list without its repeated items, each kept where it first appears."),
        message!("len(uniq(map(stops, .type)))"),
        C::List,
        exactly(1),
        R::Array,
        &[L],
    )),
    builtin(entry(
        "concat",
        "concat(...lists)",
        message!("One list of the items of every list given, in order."),
        message!("len(concat(stops, commodities))"),
        C::List,
        at_least(1),
        R::Array,
        &[L],
    )),
    builtin(entry(
        "flatten",
        "flatten(list)",
        message!("A list of lists as one list, every level flattened."),
        message!("sum(flatten([[1, 2], [3]])) // 6"),
        C::List,
        exactly(1),
        R::Array,
        &[L],
    )),
    builtin(entry(
        "sort",
        "sort(list, \"asc\"?)",
        message!("A list of numbers or text in order, \"asc\" (the default) or \"desc\"."),
        message!("first(sort(map(commodities, .weight ?? 0), \"desc\"))"),
        C::List,
        between(1, 2),
        R::Array,
        &[L, T],
    )),
    builtin(entry(
        "bitand",
        "bitand(a, b)",
        message!("The bits set in both whole numbers."),
        message!("bitand(pickupDayOfWeek, 1) == 1"),
        C::Bits,
        exactly(2),
        R::Integer,
        &[I, I],
    )),
    builtin(entry(
        "bitor",
        "bitor(a, b)",
        message!("The bits set in either whole number."),
        message!("bitor(4, 1) // 5"),
        C::Bits,
        exactly(2),
        R::Integer,
        &[I, I],
    )),
    builtin(entry(
        "bitxor",
        "bitxor(a, b)",
        message!("The bits set in exactly one of the whole numbers."),
        message!("bitxor(5, 1) // 4"),
        C::Bits,
        exactly(2),
        R::Integer,
        &[I, I],
    )),
    builtin(entry(
        "bitnand",
        "bitnand(a, b)",
        message!("The bits set in a and not in b."),
        message!("bitnand(7, 2) // 5"),
        C::Bits,
        exactly(2),
        R::Integer,
        &[I, I],
    )),
    builtin(entry(
        "bitshl",
        "bitshl(a, b)",
        message!("A whole number's bits shifted left by b places."),
        message!("bitshl(1, 3) // 8"),
        C::Bits,
        exactly(2),
        R::Integer,
        &[I, I],
    )),
    builtin(entry(
        "bitshr",
        "bitshr(a, b)",
        message!("A whole number's bits shifted right by b places, keeping its sign."),
        message!("bitshr(-8, 1) // -4"),
        C::Bits,
        exactly(2),
        R::Integer,
        &[I, I],
    )),
    builtin(entry(
        "bitushr",
        "bitushr(a, b)",
        message!("A whole number's 64 bits shifted right by b places, filling with zeros."),
        message!("bitushr(8, 1) // 4"),
        C::Bits,
        exactly(2),
        R::Integer,
        &[I, I],
    )),
    builtin(entry(
        "bitnot",
        "bitnot(a)",
        message!("A whole number with every bit flipped: -a - 1."),
        message!("bitnot(1) // -2"),
        C::Bits,
        exactly(1),
        R::Integer,
        &[I],
    )),
];

/// The functions formulas must not call, by name.
pub const DISABLED: &[Disabled] = &[
    Disabled {
        name: "fromJSON",
        reason: DisabledReason::Serialization,
        replacement: None,
    },
    Disabled {
        name: "now",
        reason: DisabledReason::Nondeterministic,
        replacement: Some(RATING_TIME),
    },
    Disabled {
        name: "repeat",
        reason: DisabledReason::TextBuilding,
        replacement: None,
    },
    Disabled {
        name: "toJSON",
        reason: DisabledReason::Serialization,
        replacement: None,
    },
];

/// The methods a date has, each giving a whole number.
pub const DATE_METHODS: &[&str] = &[
    "Year", "Month", "Day", "Hour", "Minute", "Second", "Weekday", "YearDay", "Unix",
];

/// The methods a duration has, each giving a number.
pub const DURATION_METHODS: &[&str] = &["Hours", "Minutes", "Seconds"];

/// The function or word operator with this name.
#[must_use]
pub fn function(name: &str) -> Option<&'static Function> {
    FUNCTIONS.iter().find(|function| function.name == name)
}

/// The callable function with this name: not an operator.
#[must_use]
pub fn callable(name: &str) -> Option<&'static Function> {
    function(name).filter(|function| !function.operator)
}

/// The disabled function with this name.
#[must_use]
pub fn disabled(name: &str) -> Option<&'static Disabled> {
    DISABLED.iter().find(|disabled| disabled.name == name)
}

/// The word operators every formula has: the operator entries written as one word.
pub fn word_operators() -> impl Iterator<Item = &'static str> {
    FUNCTIONS
        .iter()
        .filter(|function| function.operator && !function.name.starts_with('['))
        .map(|function| function.name)
}

/// Every name a template input may not take, sorted: the keywords and every function, builtin
/// and word operator, since an input by one of those names would hide it or could not be read.
#[must_use]
pub fn reserved() -> Vec<&'static str> {
    let mut names: Vec<&'static str> = crate::formula::syntax::lexer::KEYWORDS
        .iter()
        .copied()
        .chain(
            FUNCTIONS
                .iter()
                .filter(|function| !function.name.starts_with('['))
                .map(|function| function.name),
        )
        .collect();
    names.sort_unstable();
    names.dedup();
    names
}

/// Whether a template input may not take this name.
#[must_use]
pub fn is_reserved(name: &str) -> bool {
    crate::formula::syntax::lexer::KEYWORDS.contains(&name)
        || function(name).is_some_and(|function| !function.name.starts_with('['))
}
