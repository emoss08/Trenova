//! How the editor writes what something is: a symbol's declaration as Tariff code, a function's
//! signature with where each parameter is in it, a type short enough for a completion list, and
//! the one-line explanations of keywords and units.

use super::session::Session;
use crate::message;
use crate::message::{Arg, Message};
use crate::tariff::check::describe::{base_noun, snippet, unit_noun};
use crate::tariff::check::methods::Method;
use crate::tariff::check::{SymbolId, SymbolKind};
use crate::tariff::constraint::Constraint;
use crate::tariff::docs::Doc;
use crate::tariff::syntax::ast::{FnDecl, Item, ItemKind};
use crate::tariff::syntax::program::{Decl, ModuleId};
use crate::tariff::types::Type;
use crate::tariff::units::{Base, Simple};
use crate::tariff::wire::decode;

/// The most fields an inline record type shows before `…`.
const SHOWN_FIELDS: usize = 4;

/// A type as an author writes it, with a long inline record cut to its first fields.
#[must_use]
pub fn type_text(ty: &Type) -> String {
    match ty {
        Type::Record(record) if record.name.is_none() && record.fields.len() > SHOWN_FIELDS => {
            let shown: Vec<String> = record
                .fields
                .iter()
                .take(SHOWN_FIELDS)
                .map(|field| format!("{}: {}", field.name, type_text(&field.ty)))
                .collect();
            format!("{{ {}, … }}", shown.join(", "))
        }
        Type::Optional(inner) => format!("{}?", type_text(inner)),
        Type::List(item) => format!("List<{}>", type_text(item)),
        other => other.to_string(),
    }
}

/// Source text on one line: runs of whitespace (newlines included) become one space.
#[must_use]
pub fn flat(text: &str) -> String {
    text.split_whitespace().collect::<Vec<_>>().join(" ")
}

/// A function's signature as signature help shows it, with each parameter's place in it.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Signature {
    /// `deficit(breaks: List<Break>, actual: Weight, allow: Bool = true) -> Rated`.
    pub label: String,
    /// Each parameter: its name, its UTF-16 `[from, to)` within the label and its type.
    pub params: Vec<SignatureParam>,
}

/// One parameter of a [`Signature`].
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SignatureParam {
    /// Its name.
    pub name: String,
    /// Where it is in the label, in UTF-16 code units.
    pub range: (u32, u32),
    /// Its type as written.
    pub ty: String,
    /// Whether it may be left out.
    pub optional: bool,
}

/// Builds a signature label from a name, its parameters (name, type, default) and what it
/// returns.
#[must_use]
pub fn signature(name: &str, params: &[(String, String, Option<String>)], ret: &str) -> Signature {
    let mut label = format!("{name}(");
    let mut out = Vec::new();
    for (index, (param, ty, default)) in params.iter().enumerate() {
        if index > 0 {
            label.push_str(", ");
        }
        let from = utf16_len(&label);
        label.push_str(param);
        label.push_str(": ");
        label.push_str(ty);
        if let Some(default) = default {
            label.push_str(" = ");
            label.push_str(default);
        }
        out.push(SignatureParam {
            name: param.clone(),
            range: (from, utf16_len(&label)),
            ty: ty.clone(),
            optional: default.is_some(),
        });
    }
    label.push(')');
    if !ret.is_empty() {
        label.push_str(" -> ");
        label.push_str(ret);
    }
    Signature { label, params: out }
}

/// The UTF-16 length of a text.
#[must_use]
pub fn utf16_len(text: &str) -> u32 {
    u32::try_from(text.encode_utf16().count()).unwrap_or(u32::MAX)
}

/// The signature of a user or library function, from its declaration (types and defaults as
/// written) and, when it has no `->`, the return type the checker inferred.
#[must_use]
pub fn function_signature(session: &Session<'_>, module: ModuleId, decl: &FnDecl) -> Signature {
    let source = session
        .program
        .module(module)
        .map_or("", |found| found.source.as_str());
    let params: Vec<(String, String, Option<String>)> = decl
        .params
        .iter()
        .map(|param| {
            (
                param.name.text.clone(),
                flat(snippet(source, param.ty.span)),
                param
                    .default
                    .as_ref()
                    .map(|default| flat(snippet(source, default.span))),
            )
        })
        .collect();
    let ret = match &decl.ret {
        Some(ret) => flat(snippet(source, ret.span)),
        None => session
            .declared_at(module, decl.name.span)
            .and_then(|id| session.symbol(id))
            .and_then(|symbol| symbol.ty.as_ref())
            .filter(|ty| !matches!(ty, Type::Error))
            .map(type_text)
            .unwrap_or_default(),
    };
    signature(&decl.name.text, &params, &ret)
}

/// The item that declares a module-level name.
#[must_use]
pub fn declaring_item<'s>(
    session: &'s Session<'_>,
    module: ModuleId,
    name: &str,
) -> Option<&'s Item> {
    let found = session.program.module(module)?;
    let symbol = found.symbols.get(name).or_else(|| found.tests.get(name))?;
    found.file.items.get(symbol.decl.item())
}

/// The function a module declares as `name`.
#[must_use]
pub fn function_decl<'s>(
    session: &'s Session<'_>,
    module: ModuleId,
    name: &str,
) -> Option<&'s FnDecl> {
    let found = session.program.module(module)?;
    let symbol = found.symbols.get(name)?;
    let Decl::Fn(item) = symbol.decl else {
        return None;
    };
    match &found.file.items.get(item)?.kind {
        ItemKind::Fn(decl) => Some(decl),
        _ => None,
    }
}

/// A symbol's declaration as Tariff code: `let miles: Distance`, `input rate: Money/mi =
/// $2.35/mi`, `fn deficit(…) -> Rated`, `type Break = { … }`, `enum Zone { A, B }`.
#[must_use]
pub fn symbol_signature(session: &Session<'_>, id: SymbolId) -> String {
    let Some(symbol) = session.symbol(id) else {
        return String::new();
    };
    let ty = symbol
        .ty
        .as_ref()
        .filter(|ty| !matches!(ty, Type::Error))
        .map(type_text);
    let typed = |prefix: &str| match &ty {
        Some(ty) => format!("{prefix}{}: {ty}", symbol.name),
        None => format!("{prefix}{}", symbol.name),
    };
    match symbol.kind {
        SymbolKind::Root => match symbol.name.as_str() {
            "tables" => String::from("tables"),
            _ => typed(""),
        },
        SymbolKind::Let => typed("let "),
        SymbolKind::Var => typed("var "),
        SymbolKind::Vary => typed("vary "),
        SymbolKind::Param
        | SymbolKind::LoopVariable
        | SymbolKind::LambdaParam
        | SymbolKind::Binding => typed(""),
        SymbolKind::Module => {
            let path = alias_path(session, id).unwrap_or_default();
            if path.rsplit('.').next() == Some(symbol.name.as_str()) {
                format!("use {path}")
            } else {
                format!("use {path} as {}", symbol.name)
            }
        }
        SymbolKind::Test => format!("test \"{}\"", symbol.name),
        SymbolKind::Input
        | SymbolKind::Const
        | SymbolKind::Function
        | SymbolKind::Type
        | SymbolKind::Enum => declaration(session, id).unwrap_or_else(|| typed("")),
    }
}

/// The declaration of a module-level symbol as Tariff code, from its item.
fn declaration(session: &Session<'_>, id: SymbolId) -> Option<String> {
    let symbol = session.symbol(id)?;
    let module = symbol.module?;
    let item = declaring_item(session, module, &symbol.name)?;
    let source = session.program.module(module)?.source.as_str();
    let private = if item.private.is_some() {
        "private "
    } else {
        ""
    };
    let text = match &item.kind {
        ItemKind::Fn(decl) => {
            format!(
                "{private}fn {}",
                function_signature(session, module, decl).label
            )
        }
        ItemKind::Input(input) => {
            let mut text = format!(
                "input {}: {}",
                input.name.text,
                flat(snippet(source, input.ty.span))
            );
            if let Some(default) = &input.default {
                text.push_str(" = ");
                text.push_str(&flat(snippet(source, default.span)));
            }
            if let Some(constraint) = &input.constraint {
                text.push_str(" where ");
                text.push_str(&flat(snippet(source, constraint.span)));
            }
            text
        }
        ItemKind::Const(constant) => {
            let ty = match (&constant.ty, &symbol.ty) {
                (Some(ty), _) => flat(snippet(source, ty.span)),
                (None, Some(ty)) => type_text(ty),
                (None, None) => String::new(),
            };
            format!(
                "{private}const {}: {ty} = {}",
                constant.name.text,
                flat(snippet(source, constant.value.span))
            )
        }
        ItemKind::Type(decl) => {
            let fields: Vec<String> = decl
                .fields
                .iter()
                .map(|field| {
                    format!(
                        "{}: {}",
                        field.name.text,
                        flat(snippet(source, field.ty.span))
                    )
                })
                .collect();
            format!(
                "{private}type {} = {{ {} }}",
                decl.name.text,
                fields.join(", ")
            )
        }
        ItemKind::Enum(decl) => {
            let variants: Vec<&str> = decl
                .variants
                .iter()
                .map(|variant| variant.name.text.as_str())
                .collect();
            format!(
                "{private}enum {} {{ {} }}",
                decl.name.text,
                variants.join(", ")
            )
        }
        ItemKind::Test(test) => format!("test \"{}\"", test.name.text),
        ItemKind::Property(property) => format!("property \"{}\"", property.name.text),
        ItemKind::Use(_) | ItemKind::Stmt(_) => return None,
    };
    Some(text)
}

/// The values an input's folded `where` constraint accepts, in the words the contract-value
/// findings use ([`crate::tariff::constraint::message`]), each value written as interpolation
/// writes it: `from $0.50/mi to $15.00/mi`, `at least 1 and below 10`, `at least $1.00`,
/// `at most $10.00`, `below 5`, `one of [Standard, Guaranteed]`. `ty` is the input's type.
#[must_use]
pub fn constraint_doc(constraint: &Constraint, ty: &Type) -> Message {
    let item = constraint_item(ty);
    let shown = |json: &serde_json::Value| {
        decode(json, &item, "").map_or_else(|_| json.to_string(), |value| value.display())
    };
    match constraint {
        Constraint::Range {
            min,
            max,
            max_inclusive,
        } => {
            let low = min.as_ref().map(|json| Arg::from(shown(json)));
            let high = max.as_ref().map(|json| Arg::from(shown(json)));
            match (low, high) {
                (Some(low), Some(high)) if *max_inclusive => {
                    Message::with(message!("from {0} to {1}"), [low, high])
                }
                (Some(low), Some(high)) => {
                    Message::with(message!("at least {0} and below {1}"), [low, high])
                }
                (Some(low), None) => Message::with(message!("at least {0}"), [low]),
                (None, Some(high)) if *max_inclusive => {
                    Message::with(message!("at most {0}"), [high])
                }
                (None, Some(high)) => Message::with(message!("below {0}"), [high]),
                (None, None) => Message::new(message!("any value")),
            }
        }
        Constraint::OneOf(values) => {
            let listed: Vec<String> = values.iter().map(shown).collect();
            Message::with(
                message!("one of {0}"),
                [Arg::from(format!("[{}]", listed.join(", ")))],
            )
        }
    }
}

/// The type a constraint's values have: an item's for a list input, the inner type for an
/// optional one.
fn constraint_item(ty: &Type) -> Type {
    match ty {
        Type::List(item) | Type::Optional(item) => constraint_item(item),
        other => other.clone(),
    }
}

/// The doc comment of a function's declaration, translated when the module is the standard
/// library's.
#[must_use]
pub fn function_doc(session: &Session<'_>, module: ModuleId, name: &str) -> Option<Doc> {
    let std = session.program.module(module)?.std;
    Doc::of(std, declaring_item(session, module, name)?.docs.as_ref())
}

/// The module path a module alias names: `std.fuel` for `use std.fuel as f`.
fn alias_path(session: &Session<'_>, id: SymbolId) -> Option<String> {
    let symbol = session.symbol(id)?;
    let span = symbol.span?;
    let module = session.program.module(symbol.module?)?;
    module.file.items.iter().find_map(|item| match &item.kind {
        ItemKind::Use(using)
            if using.alias.as_ref().map_or_else(
                || using.path.last().is_some_and(|last| last.span == span),
                |alias| alias.span == span,
            ) =>
        {
            Some(using.dotted())
        }
        _ => None,
    })
}

/// A built-in method's signature, `Self` and `T` replaced by the receiver's type and its items'
/// (`where(f: Stop => Bool) -> List<Stop>` on a `List<Stop>`).
#[must_use]
pub fn method_signature(method: &Method, receiver: &Type) -> Signature {
    let own = type_text(receiver);
    let item = match receiver {
        Type::List(item) => type_text(item),
        _ => String::from("T"),
    };
    let fill = |text: &str| substitute(text, &own, &item);
    if method.property {
        return Signature {
            label: format!("{}: {}", method.name, fill(method.returns)),
            params: Vec::new(),
        };
    }
    let params: Vec<(String, String, Option<String>)> = method
        .params
        .iter()
        .map(|param| {
            (
                param.name.to_owned(),
                fill(param.ty),
                param.default.map(str::to_owned),
            )
        })
        .collect();
    signature(method.name, &params, &fill(method.returns))
}

/// `text` with the whole words `Self` and `T` replaced.
fn substitute(text: &str, own: &str, item: &str) -> String {
    let mut out = String::with_capacity(text.len());
    let mut word = String::new();
    let flush = |word: &mut String, out: &mut String| {
        match word.as_str() {
            "Self" => out.push_str(own),
            "T" => out.push_str(item),
            other => out.push_str(other),
        }
        word.clear();
    };
    for character in text.chars() {
        if character.is_ascii_alphanumeric() || character == '_' {
            word.push(character);
        } else {
            flush(&mut word, &mut out);
            out.push(character);
        }
    }
    flush(&mut word, &mut out);
    out
}

/// The SI unit of a base dimension, for a unit's exact factor.
#[must_use]
pub const fn si(base: Base) -> &'static str {
    match base {
        Base::Money => "$",
        Base::Weight => "kg",
        Base::Distance | Base::Length => "m",
        Base::Volume => "m3",
        Base::Duration => "s",
        Base::Temperature => "°C",
    }
}

/// What a unit is: its dimension and its exact factor to SI.
#[must_use]
pub fn unit_doc(unit: &Simple) -> Message {
    match unit.base {
        Some(Base::Temperature) => Message::with(
            message!(
                "`{0}` is a {1}, a unit of temperature: temperatures compare and subtract, and convert exactly between `°F` and `°C`."
            ),
            [Arg::from(unit.symbol), Arg::from(unit_noun(unit))],
        ),
        Some(Base::Money) => Message::new(message!("`$` is money in the rating currency.")),
        Some(base) => Message::with(
            message!("`{0}` is a {1}, a unit of {2}: 1 {0} = {3} {4}, exactly."),
            [
                Arg::from(unit.symbol),
                Arg::from(unit_noun(unit)),
                Arg::from(base_noun(base)),
                Arg::from(unit.factor().normalize().to_string()),
                Arg::from(si(base)),
            ],
        ),
        None if unit.symbol == "%" => {
            Message::new(message!("`%` is a percent: `18%` multiplies as 0.18."))
        }
        None => Message::with(
            message!(
                "`{0}` labels a count: `3 {0}` is the whole number 3, and `Money/{0}` is money per {1}."
            ),
            [Arg::from(unit.symbol), Arg::from(unit_noun(unit))],
        ),
    }
}

/// What each keyword (and contextual word) does, in one line.
pub const KEYWORD_DOCS: &[(&str, &str)] = &[
    (
        "tariff",
        message!(
            "Starts a tariff file: inputs, bindings, charges, total clauses and tests, producing an itemised bill."
        ),
    ),
    (
        "library",
        message!(
            "Starts a library file: functions, constants, types, enums and tests that tariffs import."
        ),
    ),
    (
        "condition",
        message!(
            "Starts a condition file: inputs, bindings and functions ending in one `Bool` that decides whether an accessorial applies."
        ),
    ),
    (
        "use",
        message!(
            "Imports a module (`use std.fuel`) or names from it (`use std.ltl.{ deficit, Break }`)."
        ),
    ),
    (
        "input",
        message!(
            "Declares a value the contract or template supplies: `input rate: Money/mi = $2.35/mi`."
        ),
    ),
    (
        "let",
        message!("Binds a name to a value that never changes."),
    ),
    (
        "var",
        message!("Binds a name that can be reassigned with `=`, `+=` or `-=` inside its block."),
    ),
    (
        "const",
        message!("Declares a compile-time constant at file level, importable from a library."),
    ),
    (
        "fn",
        message!("Declares a function: annotated parameters and return type, pure and closed."),
    ),
    (
        "type",
        message!("Declares a record type: `type Break = { from: Weight, rate: Money/cwt }`."),
    ),
    (
        "enum",
        message!("Declares an enum: `enum Zone { A, B, C }`."),
    ),
    (
        "private",
        message!("Keeps a library's declaration to itself: other modules can't import it."),
    ),
    (
        "charge",
        message!(
            "Adds a line to the bill: `charge \"Label\" as CODE (quantity: q, rate: r, minimum: lo, maximum: hi) = amount`; in a test, `charge(CODE)` reads a line's amount."
        ),
    ),
    (
        "note",
        message!("Adds a note with no amount to the bill, the trace and the invoice."),
    ),
    (
        "review",
        message!(
            "Bills as usual and asks for the rating to be approved before it is used, with this text as the reason: `review \"Hazmat over 500 miles\"`."
        ),
    ),
    (
        "if",
        message!(
            "Runs a block, or gives a value, only when a condition holds; `if let` binds an optional when present."
        ),
    ),
    (
        "else",
        message!("What runs, or the value given, when the `if` condition does not hold."),
    ),
    (
        "match",
        message!("Picks the first arm whose pattern matches; a `match` must cover every value."),
    ),
    (
        "for",
        message!(
            "Walks a list or a range of `Int`s in order: `for stop in shipment.stops where … { … }`."
        ),
    ),
    (
        "in",
        message!(
            "Tests membership in a list or range, names the list a `for` walks, and gives the values a `vary` takes or the target a monotone `expect` follows."
        ),
    ),
    (
        "where",
        message!(
            "Keeps only the items a condition holds for; after an input, bounds the values a contract may set: `where $0.50/mi..=$15/mi` or `where [Standard, Guaranteed]`."
        ),
    ),
    ("break", message!("Leaves the innermost `for` loop.")),
    (
        "continue",
        message!("Skips to the next iteration of the innermost `for` loop."),
    ),
    (
        "return",
        message!("Gives a function's result and leaves it."),
    ),
    (
        "minimum",
        message!(
            "A total clause: when the total falls short, the difference is billed as a minimum charge adjustment; `minimum m on freight` applies to the freight lines only. As a charge option, `minimum: m` makes the line bill at least `m`."
        ),
    ),
    (
        "maximum",
        message!(
            "A total clause: when the total exceeds it, a negative adjustment brings it down; `maximum m on freight` applies to the freight lines only. As a charge option, `maximum: m` makes the line bill at most `m`."
        ),
    ),
    (
        "round",
        message!(
            "A total clause: how every line and the total are rounded, `round to cents` by default."
        ),
    ),
    (
        "to",
        message!("What a `round` clause rounds to: `cents` or an amount."),
    ),
    ("up", message!("Rounds away from zero.")),
    ("down", message!("Rounds toward zero.")),
    (
        "half",
        message!("With `even`, rounds half to even (banker's rounding)."),
    ),
    (
        "even",
        message!("With `half`, rounds half to even (banker's rounding)."),
    ),
    ("cents", message!("Rounds to the currency's cents.")),
    (
        "on",
        message!(
            "After a `minimum` or `maximum` amount, what the clause bounds: `on freight`, the uncoded lines."
        ),
    ),
    (
        "freight",
        message!(
            "In `minimum m on freight` and `maximum m on freight`: the bill's uncoded lines; `bill.freight` is what they add up to so far."
        ),
    ),
    (
        "require",
        message!(
            "Refuses to rate unless a condition holds, with the reason given after `else`; `require let` binds an optional from there on."
        ),
    ),
    (
        "fail",
        message!("Refuses to rate with a message the user can act on."),
    ),
    (
        "fails",
        message!("Expects the tariff to refuse to rate, optionally with that message."),
    ),
    (
        "test",
        message!("Declares a test, run against the sample record on every change."),
    ),
    (
        "property",
        message!(
            "Declares a property: a rule checked over many shipments, the cases chosen from the `vary` ranges, run on every change like a test."
        ),
    ),
    (
        "vary",
        message!(
            "Makes a field, an input or a new name take many values in a property: `vary shipment.weight in 1 lb..40_000 lb`."
        ),
    ),
    (
        "assume",
        message!(
            "Skips the cases of a property where a condition is false; it is read before the tariff runs, so it can't read the bill."
        ),
    ),
    (
        "nondecreasing",
        message!(
            "In a property, `expect value nondecreasing in target`: the value never falls as the varied target grows."
        ),
    ),
    (
        "nonincreasing",
        message!(
            "In a property, `expect value nonincreasing in target`: the value never rises as the varied target grows."
        ),
    ),
    (
        "best",
        message!(
            "Bills the cheapest of several methods: each `option` runs, and the one with the lowest subtotal joins the bill. `let x = best \"Label\" …` also names that subtotal."
        ),
    ),
    (
        "worst",
        message!(
            "Bills the dearest of several methods: each `option` runs, and the one with the highest subtotal joins the bill. `let x = worst \"Label\" …` also names that subtotal."
        ),
    ),
    (
        "option",
        message!(
            "One method of a `best` or `worst`: its charges are its subtotal, and an option that refuses is out of the running."
        ),
    ),
    (
        "given",
        message!(
            "Sets a field of the sample record or an input for this test, or for every case of a property."
        ),
    ),
    (
        "expect",
        message!(
            "A `Bool` the test needs to hold after the tariff has run; in a property, it must hold in every case, or `expect value nondecreasing in target` must hold as the target grows."
        ),
    ),
    (
        "and",
        message!(
            "True when both sides are true; the right side is read only when the left is true."
        ),
    ),
    (
        "or",
        message!(
            "True when either side is true; the right side is read only when the left is false."
        ),
    ),
    ("not", message!("True when the value is false.")),
    ("true", message!("The `Bool` value true.")),
    ("false", message!("The `Bool` value false.")),
    ("none", message!("The empty value of an optional.")),
    (
        "as",
        message!(
            "Names the accessorial charge code a charge bills, or the name a module is imported as."
        ),
    ),
    ("total", message!("In a test, the bill's total.")),
    (
        "lines",
        message!("In a test, the bill's lines: label, code, amount and kind."),
    ),
    (
        "reviews",
        message!("In a test, the bill's review reasons, in order."),
    ),
];

/// The one-line explanation of a keyword or contextual word.
#[must_use]
pub fn keyword_doc(word: &str) -> Option<&'static str> {
    KEYWORD_DOCS
        .iter()
        .find(|(keyword, _)| *keyword == word)
        .map(|(_, doc)| *doc)
}

#[cfg(test)]
mod tests {
    use super::signature;

    #[test]
    fn a_signature_knows_where_each_parameter_is() {
        let found = signature(
            "f",
            &[
                (String::from("a"), String::from("°F"), None),
                (
                    String::from("b"),
                    String::from("Int"),
                    Some(String::from("1")),
                ),
            ],
            "Int",
        );
        assert_eq!(found.label, "f(a: °F, b: Int = 1) -> Int");
        assert_eq!(found.params[0].range, (2, 7));
        assert_eq!(found.params[1].range, (9, 19));
    }
}
