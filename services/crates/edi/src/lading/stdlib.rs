//! The standard library's catalog: every function's name, how it is called, what it returns
//! and its doc. The checker types calls by these names, the evaluator runs them, and the editor
//! completes and describes them from here.

use trenova_module_kit::message;

/// What a function is called on.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Receiver {
    /// Text, coded or not.
    Text,
    /// A number.
    Number,
    /// Money.
    Money,
    /// A measure.
    Measure,
    /// A date.
    Date,
    /// A date-time.
    DateTime,
    /// A list.
    List,
    /// A segment or a list of segments.
    X12,
    /// Nothing: a plain function.
    None,
}

/// One function of the standard library.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Function {
    /// Its name.
    pub name: &'static str,
    /// What it is called on, for a method.
    pub receiver: Receiver,
    /// How it is called, as the editor shows it.
    pub signature: &'static str,
    /// What it does, as a catalog message.
    pub doc: &'static str,
}

const fn f(
    name: &'static str,
    receiver: Receiver,
    signature: &'static str,
    doc: &'static str,
) -> Function {
    Function {
        name,
        receiver,
        signature,
        doc,
    }
}

/// Every function.
pub const FUNCTIONS: &[Function] = &[
    f(
        "trim",
        Receiver::Text,
        "trim(text: Text) -> Text",
        message!("The text without spaces at either end"),
    ),
    f(
        "upper",
        Receiver::Text,
        "upper(text: Text) -> Text",
        message!("The text in capitals"),
    ),
    f(
        "lower",
        Receiver::Text,
        "lower(text: Text) -> Text",
        message!("The text in small letters"),
    ),
    f(
        "titleCase",
        Receiver::Text,
        "titleCase(text: Text) -> Text",
        message!("Each word capitalised: DALLAS becomes Dallas"),
    ),
    f(
        "take",
        Receiver::Text,
        "take(text: Text, n: Int) -> Text",
        message!("The first n characters, or the first n items of a list"),
    ),
    f(
        "drop",
        Receiver::Text,
        "drop(text: Text, n: Int) -> Text",
        message!("Everything after the first n characters, or items of a list"),
    ),
    f(
        "truncate",
        Receiver::Text,
        "truncate(text: Text, n: Int) -> Text",
        message!("The text cut to at most n characters"),
    ),
    f(
        "pad",
        Receiver::Text,
        "pad(text: Text, n: Int, with: Text = \" \", left: Bool = false) -> Text",
        message!("The text filled to n characters"),
    ),
    f(
        "replace",
        Receiver::Text,
        "replace(text: Text, old: Text, new: Text) -> Text",
        message!("The text with every old replaced by new"),
    ),
    f(
        "split",
        Receiver::Text,
        "split(text: Text, separator: Text) -> [Text]",
        message!("The pieces between each separator"),
    ),
    f(
        "digits",
        Receiver::Text,
        "digits(text: Text) -> Text",
        message!("Only the digits"),
    ),
    f(
        "letters",
        Receiver::Text,
        "letters(text: Text) -> Text",
        message!("Only the letters"),
    ),
    f(
        "clean",
        Receiver::Text,
        "clean(text: Text) -> Text",
        message!("The text without characters X12 cannot carry or the partner's separators"),
    ),
    f(
        "startsWith",
        Receiver::Text,
        "startsWith(text: Text, prefix: Text) -> Bool",
        message!("Whether the text starts with the prefix"),
    ),
    f(
        "endsWith",
        Receiver::Text,
        "endsWith(text: Text, suffix: Text) -> Bool",
        message!("Whether the text ends with the suffix"),
    ),
    f(
        "contains",
        Receiver::Text,
        "contains(text: Text, part: Text) -> Bool",
        message!("Whether the text contains the part"),
    ),
    f(
        "matches",
        Receiver::Text,
        "matches(text: Text, pattern: Text) -> Bool",
        message!("Whether the whole text matches the pattern, written as r\"...\""),
    ),
    f(
        "length",
        Receiver::Text,
        "length(text: Text) -> Int",
        message!("How many characters the text has"),
    ),
    f(
        "isBlank",
        Receiver::Text,
        "isBlank(text: Text) -> Bool",
        message!("Whether the text is empty or only spaces"),
    ),
    f(
        "zip5",
        Receiver::Text,
        "zip5(postal: Text) -> Text?",
        message!("The first five digits of a US ZIP code"),
    ),
    f(
        "zip9",
        Receiver::Text,
        "zip9(postal: Text) -> Text?",
        message!("A nine-digit ZIP code written 12345-6789"),
    ),
    f(
        "stateCode",
        Receiver::Text,
        "stateCode(text: Text) -> Text?",
        message!("A US or Canadian state or province, by name or code, as its two-letter code"),
    ),
    f(
        "countryCode",
        Receiver::Text,
        "countryCode(text: Text, iso: Int = 2) -> Text?",
        message!("A country, by name or code, as its ISO code of two or three letters"),
    ),
    f(
        "phone",
        Receiver::Text,
        "phone(text: Text) -> Text?",
        message!("A phone number in E.164 form (+15551234567)"),
    ),
    f(
        "checkDigit",
        Receiver::Text,
        "checkDigit(text: Text, scheme: Text) -> Bool",
        message!("Whether the number's check digit is right (scheme \"mod10\" or \"mod7\")"),
    ),
    f(
        "isDate",
        Receiver::Text,
        "isDate(text: Text, pattern: Text) -> Bool",
        message!("Whether the text is a date in the pattern"),
    ),
    f(
        "isNumber",
        Receiver::Text,
        "isNumber(text: Text) -> Bool",
        message!("Whether the text is a number"),
    ),
    f(
        "date",
        Receiver::Text,
        "date(text: Text, pattern: Text) -> Date?",
        message!("The date written in the pattern (yyyy, yy, MM, dd), or of a date-time"),
    ),
    f(
        "time",
        Receiver::Text,
        "time(text: Text, pattern: Text) -> Time?",
        message!("The time written in the pattern (HH, mm, ss), or of a date-time"),
    ),
    f(
        "dateTime",
        Receiver::Text,
        "dateTime(text: Text, pattern: Text, zone: Zone) -> DateTime?",
        message!("The date and time written in the pattern, in the zone"),
    ),
    f(
        "int",
        Receiver::None,
        "int(value) -> Int?",
        message!("A whole number from text or a decimal (rounded half up)"),
    ),
    f(
        "decimal",
        Receiver::None,
        "decimal(value) -> Decimal?",
        message!("A decimal from text or a whole number"),
    ),
    f(
        "text",
        Receiver::None,
        "text(value) -> Text",
        message!("Any value written as text"),
    ),
    f(
        "join",
        Receiver::List,
        "join(items: [Text], separator: Text) -> Text",
        message!("The items joined with the separator"),
    ),
    f(
        "round",
        Receiver::Number,
        "round(value, places: Int = 0)",
        message!("The value rounded half up to the places"),
    ),
    f(
        "floor",
        Receiver::Number,
        "floor(value)",
        message!("The value rounded down"),
    ),
    f(
        "ceil",
        Receiver::Number,
        "ceil(value)",
        message!("The value rounded up"),
    ),
    f(
        "abs",
        Receiver::Number,
        "abs(value)",
        message!("The value without its sign"),
    ),
    f(
        "min",
        Receiver::Number,
        "min(a, b)",
        message!("The smaller of two values, or a list's smallest by a function"),
    ),
    f(
        "max",
        Receiver::Number,
        "max(a, b)",
        message!("The larger of two values, or a list's largest by a function"),
    ),
    f(
        "money",
        Receiver::None,
        "money(amount: Decimal, currency: Text) -> Money",
        message!("An amount in a currency"),
    ),
    f(
        "amount",
        Receiver::Money,
        "amount(money: Money) -> Decimal",
        message!("The amount without its currency"),
    ),
    f(
        "currency",
        Receiver::Money,
        "currency(money: Money) -> Text",
        message!("The currency's code"),
    ),
    f(
        "weight",
        Receiver::None,
        "weight(value: Decimal, unitCode: Text) -> Weight?",
        message!("A weight from a number and an X12 weight unit code (L, K, E)"),
    ),
    f(
        "length",
        Receiver::None,
        "length(value: Decimal, unitCode: Text) -> Length?",
        message!("A length from a number and an X12 unit code (F, I, M)"),
    ),
    f(
        "volume",
        Receiver::None,
        "volume(value: Decimal, unitCode: Text) -> Volume?",
        message!("A volume from a number and an X12 unit code (E cubic feet, X cubic meters)"),
    ),
    f(
        "temperature",
        Receiver::None,
        "temperature(value: Decimal, scale: Text) -> Temperature?",
        message!("A temperature from a number and its scale (F or C)"),
    ),
    f(
        "in",
        Receiver::Measure,
        "in(measure, unit) -> the measure",
        message!("The measure in another unit: weight.in(kg)"),
    ),
    f(
        "value",
        Receiver::Measure,
        "value(measure, unit) -> Decimal",
        message!("The number of a measure in a unit: weight.value(lb)"),
    ),
    f(
        "at",
        Receiver::None,
        "at(date: Date?, time: Time?, zone: Zone) -> DateTime?",
        message!(
            "The instant a local date and time stand for in a zone; midnight when the time is none"
        ),
    ),
    f(
        "date",
        Receiver::DateTime,
        "date(at: DateTime) -> Date",
        message!("The local date"),
    ),
    f(
        "time",
        Receiver::DateTime,
        "time(at: DateTime) -> Time",
        message!("The local time"),
    ),
    f(
        "zone",
        Receiver::DateTime,
        "zone(at: DateTime) -> Zone",
        message!("The zone it is in"),
    ),
    f(
        "inZone",
        Receiver::DateTime,
        "inZone(at: DateTime, zone: Zone) -> DateTime",
        message!("The same instant in another zone"),
    ),
    f(
        "zoneCode",
        Receiver::DateTime,
        "zoneCode(at: DateTime) -> Text",
        message!("The X12 time code for its zone (CT, ET, MT, PT, ...), daylight time or not"),
    ),
    f(
        "zoneFromCode",
        Receiver::None,
        "zoneFromCode(code: Text) -> Zone?",
        message!("The zone an X12 time code stands for"),
    ),
    f(
        "addDays",
        Receiver::Date,
        "addDays(date: Date, days: Int) -> Date",
        message!("The date days later"),
    ),
    f(
        "addBusinessDays",
        Receiver::Date,
        "addBusinessDays(date: Date, days: Int) -> Date",
        message!("The date that many weekdays later"),
    ),
    f(
        "now",
        Receiver::None,
        "now() -> DateTime",
        message!("The time the run started"),
    ),
    f(
        "count",
        Receiver::List,
        "count(items, where?) -> Int",
        message!("How many items, or how many match"),
    ),
    f(
        "first",
        Receiver::List,
        "first(items) -> item?",
        message!("The first item"),
    ),
    f(
        "last",
        Receiver::List,
        "last(items) -> item?",
        message!("The last item"),
    ),
    f(
        "isEmpty",
        Receiver::List,
        "isEmpty(items) -> Bool",
        message!("Whether there are no items"),
    ),
    f(
        "find",
        Receiver::List,
        "find(items, f) -> item?",
        message!("The first item f is true for"),
    ),
    f(
        "filter",
        Receiver::List,
        "filter(items, f) -> [item]",
        message!("The items f is true for"),
    ),
    f(
        "map",
        Receiver::List,
        "map(items, f) -> [result]",
        message!("f of each item"),
    ),
    f(
        "any",
        Receiver::List,
        "any(items, f) -> Bool",
        message!("Whether f is true for any item"),
    ),
    f(
        "all",
        Receiver::List,
        "all(items, f) -> Bool",
        message!("Whether f is true for every item"),
    ),
    f(
        "sum",
        Receiver::List,
        "sum(items, f?) -> number",
        message!("The total of the items, or of f of each"),
    ),
    f(
        "sortBy",
        Receiver::List,
        "sortBy(items, f) -> [item]",
        message!("The items in order of f"),
    ),
    f(
        "distinct",
        Receiver::List,
        "distinct(items) -> [item]",
        message!("The items without repeats"),
    ),
    f(
        "flatten",
        Receiver::List,
        "flatten(lists) -> [item]",
        message!("The lists' items in one list"),
    ),
    f(
        "get",
        Receiver::List,
        "get(items, index: Int) -> item?",
        message!("The item at a position, from 0"),
    ),
    f(
        "refs",
        Receiver::X12,
        "refs(segments: [Segment]) -> [Text: Text]",
        message!("L11 or N9 segments as qualifier to reference number"),
    ),
    f(
        "address",
        Receiver::X12,
        "address(loop: Loop) -> Address?",
        message!("The N3 and N4 of a name loop as an address"),
    ),
    f(
        "party",
        Receiver::X12,
        "party(loop: Loop) -> Party?",
        message!("The N1, N3 and N4 of a name loop as a party"),
    ),
    f(
        "elements",
        Receiver::X12,
        "elements(segment: Segment) -> [Text]",
        message!("Every element of a segment as text"),
    ),
];

/// The functions with a name.
pub fn named(name: &str) -> impl Iterator<Item = &'static Function> + '_ {
    FUNCTIONS
        .iter()
        .filter(move |function| function.name == name)
}

/// Whether a name is a standard function.
#[must_use]
pub fn exists(name: &str) -> bool {
    FUNCTIONS.iter().any(|function| function.name == name)
}
