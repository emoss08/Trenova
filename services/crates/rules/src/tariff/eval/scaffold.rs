//! `tariff.scaffoldTest`: "Save as test" in the Studio's preview. Rates a tariff against a real
//! shipment's record and writes a `test` block that pins the result:
//!
//! - a `given` for every record field that differs from the sample record (nested records
//!   field by field; stops and commodities through the test constructors, leaving out fields
//!   equal to the defaults, sequences equal to the stop's position and dwells equal to the
//!   departure minus the arrival, and writing a stop's location field by field (`zip3: "752"`); `pickup` and `delivery` only when they are not the stops
//!   they would follow);
//! - a `given tables.Code[k, …] = value` for every cell the rating found by `[…]`, `.cell(…)` or
//!   `.band(x)` (a range axis written as the band's floor), so the test holds the cells it reads
//!   instead of reading the matrix's live data;
//! - a `given input.x` for every input whose value differs from its default;
//! - an `expect charge(CODE)` (or `charge("Label")` for freight) per charge, `expect total`,
//!   and `expect reviews == […]` when the bill asks for review; or `expect fails "…"` when the
//!   rating was refused.
//!
//! Request: `{"source", "context", "record": {…} | null, "inputs": {…}, "name": "…"}`.
//! Response: `{"test": "test \"…\" {\n  given …\n  expect …\n}\n"}`; or `{"test": null,
//! "error": …}` when the tariff does not load or the rating fails; or `{"test": null, "status":
//! "needs", "needs": […]}` when it needs table cells the request did not carry.

use super::bill::LineKind;
use super::budget::Limits;
use super::error::{Code, EvalError};
use super::interp::link::Linked;
use super::interp::{Env, Interp, Setup};
use super::ops::{error_json, gate, needs_json};
use super::record::{decode_record, sample, settle, stop, with_dwell};
use super::run::{Outcome, Request, run_program};
use super::tables::Tables;
use super::trace::Trace;
use super::value::{Record, Value, format_decimal, format_quantity, money_digits};
use trenova_module_kit::span::Utf16Index;
use crate::tariff::check::Analysis;
use crate::tariff::context::Context;
use crate::tariff::schema::{Axis, ENUMS, Schema, TableShape};
use crate::tariff::syntax::ast::FileKind;
use crate::tariff::syntax::program::load;
use crate::tariff::types::Type;
use crate::tariff::wire::decode as decode_wire;
use rust_decimal::Decimal;
use serde::Deserialize;
use serde_json::{Map, Value as Json};
use std::collections::BTreeMap;
use std::fmt::Write as _;
use std::rc::Rc;
use trenova_module_kit::message::Message;
use trenova_module_kit::ops::{OpResult, decode, encode};

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct ScaffoldInput {
    source: String,
    #[serde(default)]
    context: Context,
    #[serde(default)]
    record: Option<Json>,
    #[serde(default)]
    inputs: BTreeMap<String, Json>,
    name: String,
}

/// The longest `given` line written on one line; a longer list breaks one item per line.
const MAX_LINE: usize = 100;

/// The test constructors, by the stop kind they build.
const CONSTRUCTORS: [(&str, &str); 4] = [
    ("Pickup", "pickup"),
    ("Delivery", "delivery"),
    ("SplitPickup", "splitPickup"),
    ("SplitDelivery", "splitDelivery"),
];

/// `tariff.scaffoldTest`.
///
/// # Errors
///
/// When the input does not have the operation's shape.
pub fn op(input: &serde_json::value::RawValue) -> OpResult {
    let input: ScaffoldInput = decode(input)?;
    let mut output = Map::new();
    output.insert(String::from("test"), Json::Null);
    let program = load(&input.source, Some(FileKind::Tariff), &input.context);
    let index = program
        .root()
        .map_or_else(|| Utf16Index::new(""), |root| root.utf16.clone());
    let schema = Schema::new(&input.context.settings);
    let record = match &input.record {
        None | Some(Json::Null) => Ok(sample(&input.context.settings)),
        Some(json) => decode_record(json, &schema).map_err(|error| {
            EvalError::with(
                Code::Record,
                trenova_module_kit::message!("The record does not fit the schema at {0}: {1}"),
                [
                    trenova_module_kit::message::Arg::from(error.path.clone()),
                    trenova_module_kit::message::Arg::from(error.message),
                ],
            )
        }),
    };
    let (analysis, record) = match (gate(&program, &input.context), record) {
        (Err(error), _) | (Ok(_), Err(error)) => {
            output.insert(String::from("error"), error_json(&error, &index));
            return encode(&Json::Object(output));
        }
        (Ok(analysis), Ok(record)) => (analysis, record),
    };
    let tables = Tables::build(&input.context.tables);
    let run = run_program(Request {
        program: &program,
        types: &analysis,
        tables: &tables,
        record: record.clone(),
        inputs: &input.inputs,
        limits: Limits::DEFAULT,
        trace: false,
    });
    let cells = cell_givens(&run.trace, &input.context.tables, &schema);
    let expects = match run.outcome {
        Outcome::Rated { bill, .. } => {
            let mut expects = charge_expects(&bill.lines, bill.total);
            if !bill.reviews.is_empty() {
                let reasons: Vec<String> = bill
                    .reviews
                    .iter()
                    .map(|review| text_literal(&review.text))
                    .collect();
                expects.push(format!("expect reviews == [{}]", reasons.join(", ")));
            }
            expects
        }
        Outcome::Refused(refusal) => vec![format!(
            "expect fails {}",
            text_literal(&refusal.message.text())
        )],
        Outcome::Failed(error) => {
            output.insert(String::from("error"), error_json(&error, &index));
            return encode(&Json::Object(output));
        }
        Outcome::Needs(needs) => {
            output.insert(String::from("status"), Json::String(String::from("needs")));
            output.insert(String::from("needs"), needs_json(&needs));
            return encode(&Json::Object(output));
        }
        Outcome::Condition(_) => {
            let error = EvalError::internal(Message::new(trenova_module_kit::message!(
                "Only a tariff's rating can be saved as a test"
            )));
            output.insert(String::from("error"), error_json(&error, &index));
            return encode(&Json::Object(output));
        }
    };
    let writer = Writer { schema: &schema };
    let mut givens = writer.record_givens(&record, &sample(&input.context.settings));
    givens.extend(cells);
    let linked = Linked::new(&program);
    givens.extend(input_givens(&Inputs {
        linked,
        analysis: &analysis,
        schema: &schema,
        tables: &tables,
        values: &input.inputs,
    }));
    let mut test = format!("test {} {{\n", text_literal(&input.name));
    for line in givens.iter().chain(&expects) {
        let _ = writeln!(test, "  {line}");
    }
    test.push_str("}\n");
    output.insert(String::from("test"), Json::String(test));
    encode(&Json::Object(output))
}

/// A `given tables.Code[k, …] = value` for each cell the rating found, each once, in the order
/// first read. `.interpolate(x)` reads no one cell, so its table is left live.
fn cell_givens(trace: &Trace, shapes: &[TableShape], schema: &Schema) -> Vec<String> {
    let writer = Writer { schema };
    let mut out: Vec<String> = Vec::new();
    for lookup in trace.lookups() {
        let (Some(matched), false) = (&lookup.matched, lookup.pending) else {
            continue;
        };
        let Some(shape) = shapes.iter().find(|shape| shape.code == lookup.table) else {
            continue;
        };
        let Some(value_type) = shape.value() else {
            continue;
        };
        let value = match (lookup.op, &lookup.value) {
            ("index", value) => value.clone(),
            ("cell", Value::Record(cell)) => {
                let bare = matches!(cell.get("minimum"), Some(Value::None))
                    && matches!(cell.get("deficit"), Some(Value::Bool(false)));
                match cell.get("value") {
                    Some(value) if bare => value.clone(),
                    _ => Value::Record(Rc::clone(cell)),
                }
            }
            ("band", Value::Record(band)) => band.get("value").cloned().unwrap_or(Value::None),
            _ => continue,
        };
        let keys: Option<Vec<String>> = shape
            .axes
            .iter()
            .zip(matched)
            .map(|(axis, at)| cell_key(&writer, axis, at))
            .collect();
        let Some(keys) = keys.filter(|keys| keys.len() == shape.axes.len()) else {
            continue;
        };
        let ty = if matches!(value, Value::Record(_)) {
            crate::tariff::check::cell_type(value_type)
        } else {
            value_type
        };
        let line = format!(
            "given tables.{}[{}] = {}",
            shape.code,
            keys.join(", "),
            writer.literal(&value, &ty)
        );
        if !out.contains(&line) {
            out.push(line);
        }
    }
    out
}

/// One axis's key of a found cell as source: an exact key as text (or its `FreightClass`
/// variant), a band as its floor in the axis's unit.
fn cell_key(writer: &Writer<'_>, axis: &Axis, at: &Json) -> Option<String> {
    if let Some(key) = at.get("key").and_then(Json::as_str) {
        return Some(match axis.key_type(writer.schema) {
            Type::Enum(enumeration) => variant_literal(key, &enumeration.name),
            _ => text_literal(key),
        });
    }
    let floor = crate::wire::parse_decimal(at.get("min")?.as_str()?).ok()?;
    let unit = axis.unit.as_deref().or(match axis.kind.as_str() {
        "WeightBreak" => Some("lb"),
        "Distance" => Some("mi"),
        "LinearFeet" => Some("ft"),
        _ => None,
    });
    Some(match (axis.key_type(writer.schema), unit) {
        (Type::Quantity(_), Some(unit)) => format!("{} {unit}", format_decimal(floor)),
        _ => format_decimal(floor),
    })
}

/// One `expect` per charge code or freight label, then the total.
fn charge_expects(lines: &[super::bill::Line], total: Decimal) -> Vec<String> {
    let mut keys: Vec<(bool, String)> = Vec::new();
    for line in lines.iter().filter(|line| line.kind == LineKind::Charge) {
        let key = match (&line.code, &line.label) {
            (Some(code), _) => (true, code.clone()),
            (None, Some(label)) => (false, label.clone()),
            (None, None) => continue,
        };
        if !keys.contains(&key) {
            keys.push(key);
        }
    }
    let mut out = Vec::with_capacity(keys.len() + 1);
    for (coded, key) in keys {
        let sum: Decimal = lines
            .iter()
            .filter(|line| {
                if coded {
                    line.code.as_deref() == Some(key.as_str())
                } else {
                    line.label.as_deref() == Some(key.as_str())
                }
            })
            .map(|line| line.amount)
            .sum();
        let reader = if coded { key } else { text_literal(&key) };
        out.push(format!(
            "expect charge({reader}) == {}",
            money_literal(sum, "")
        ));
    }
    out.push(format!("expect total == {}", money_literal(total, "")));
    out
}

/// What [`input_givens`] reads.
struct Inputs<'i> {
    linked: Linked<'i>,
    analysis: &'i Analysis,
    schema: &'i Schema,
    tables: &'i Tables,
    values: &'i BTreeMap<String, Json>,
}

/// A `given input.x` for each input sent with a value other than its default.
fn input_givens(inputs: &Inputs<'_>) -> Vec<String> {
    let mut interp = Interp::new(Setup {
        linked: &inputs.linked,
        types: inputs.analysis,
        tables: inputs.tables,
        record: Value::None,
        meter: super::budget::Meter::new(Limits::DEFAULT),
        trace: false,
    });
    let writer = Writer {
        schema: inputs.schema,
    };
    let mut out = Vec::new();
    for input in inputs.linked.inputs(0) {
        let name = &input.name.text;
        let (Some(json), Some(ty)) = (inputs.values.get(name), interp.input_type(name)) else {
            continue;
        };
        let Ok(value) = decode_wire(json, &ty, name) else {
            continue;
        };
        let default = input.default.as_ref().and_then(|default| {
            let mut scope = Env::new(0, true);
            interp.eval_in(&mut scope, default).ok()
        });
        if default.is_some_and(|default| default.equals(&value).unwrap_or(false)) {
            continue;
        }
        out.push(writer.given(&format!("input.{name}"), &value, &ty));
    }
    out
}

struct Writer<'s> {
    schema: &'s Schema,
}

/// A value as Tariff source of type `ty`: stops and commodities through the test
/// constructors, variants qualified where two schema enums share them. A property's
/// counterexample writes its `given`s with it.
#[must_use]
pub fn source_literal(schema: &Schema, value: &Value, ty: &Type) -> String {
    Writer { schema }.literal(value, ty)
}

impl Writer<'_> {
    /// Givens for every field of `record` that differs from `base`.
    fn record_givens(&self, record: &Value, base: &Value) -> Vec<String> {
        let mut out = Vec::new();
        let (Value::Record(record), Value::Record(base)) = (record, base) else {
            return out;
        };
        for (root, ty, _) in &self.schema.roots {
            let (Some(value), Some(default)) = (record.get(root), base.get(root)) else {
                continue;
            };
            if *root == "shipment" {
                self.shipment_givens(value, default, ty, &mut out);
            } else {
                self.diff(root, value, default, ty, &mut out);
            }
        }
        out
    }

    fn diff(&self, path: &str, value: &Value, default: &Value, ty: &Type, out: &mut Vec<String>) {
        if value.equals(default).unwrap_or(false) {
            return;
        }
        if let (Value::Record(fields), Value::Record(defaults), Type::Record(record)) =
            (value, default, ty)
        {
            for field in &record.fields {
                let (Some(value), Some(default)) =
                    (fields.get(&field.name), defaults.get(&field.name))
                else {
                    continue;
                };
                self.diff(
                    &format!("{path}.{}", field.name),
                    value,
                    default,
                    &field.ty,
                    out,
                );
            }
            return;
        }
        out.push(self.given(path, value, ty));
    }

    /// `given path = value`, a long list one item per line.
    fn given(&self, path: &str, value: &Value, ty: &Type) -> String {
        let line = format!("given {path} = {}", self.literal(value, ty));
        match (value, ty.unwrapped()) {
            (Value::List(items), Type::List(item)) if line.len() > MAX_LINE => {
                let mut out = format!("given {path} = [\n");
                for (index, entry) in items.iter().enumerate() {
                    let _ = writeln!(out, "    {},", self.item_literal(entry, index, item));
                }
                out.push_str("  ]");
                out
            }
            _ => line,
        }
    }

    fn shipment_givens(&self, value: &Value, default: &Value, ty: &Type, out: &mut Vec<String>) {
        let (Value::Record(shipment), Value::Record(defaults), Type::Record(record)) =
            (value, default, ty)
        else {
            return;
        };
        for field in &record.fields {
            if matches!(field.name.as_str(), "pickup" | "delivery") {
                continue;
            }
            let (Some(value), Some(default)) =
                (shipment.get(&field.name), defaults.get(&field.name))
            else {
                continue;
            };
            self.diff(
                &format!("shipment.{}", field.name),
                value,
                default,
                &field.ty,
                out,
            );
        }
        let mut rebuilt = (**defaults).clone();
        if let Some(stops) = shipment.get("stops") {
            rebuilt.set("stops", without_sequences(stops));
        }
        let wrapped = Value::record(vec![("shipment", Value::Record(Rc::new(rebuilt)))]);
        let settled = settle(&wrapped, false, false);
        let followed = match &settled {
            Value::Record(roots) => roots.get("shipment").cloned(),
            _ => None,
        };
        for field in ["pickup", "delivery"] {
            let actual = shipment.get(field).cloned().unwrap_or(Value::None);
            let expected = match &followed {
                Some(Value::Record(settled)) => settled.get(field).cloned().unwrap_or(Value::None),
                _ => Value::None,
            };
            if !actual.equals(&expected).unwrap_or(false) {
                let stop_type = record
                    .field(field)
                    .map_or(Type::Error, |def| def.ty.clone());
                out.push(format!(
                    "given shipment.{field} = {}",
                    self.stop_literal(&actual, 0, &stop_type)
                ));
            }
        }
    }

    /// A value as Tariff source of type `ty`.
    fn literal(&self, value: &Value, ty: &Type) -> String {
        match (value, ty.unwrapped()) {
            (Value::None, _) => String::from("none"),
            (Value::Bool(flag), _) => flag.to_string(),
            (Value::Int(number) | Value::Number(number), _) => format_decimal(*number),
            (Value::Quantity(quantity), _) => quantity_literal(quantity),
            (Value::Text(text), _) => text_literal(text),
            (Value::Date(date), _) => format!("@{}", date.iso()),
            (Value::DateTime(at), _) => format!("@{}", at.iso()),
            (Value::Time(time), _) => format!("@{}", time.display()),
            (Value::Enum(variant), Type::Enum(enumeration)) => {
                variant_literal(variant, &enumeration.name)
            }
            (Value::Enum(variant), _) => variant.to_string(),
            (Value::List(items), Type::List(item)) => {
                let parts: Vec<String> = items
                    .iter()
                    .enumerate()
                    .map(|(index, entry)| self.item_literal(entry, index, item))
                    .collect();
                format!("[{}]", parts.join(", "))
            }
            (Value::List(items), _) => {
                let parts: Vec<String> = items
                    .iter()
                    .map(|entry| self.literal(entry, &Type::Error))
                    .collect();
                format!("[{}]", parts.join(", "))
            }
            (Value::Record(record), Type::Record(declared)) => {
                let parts: Vec<String> = record
                    .fields
                    .iter()
                    .map(|(name, entry)| {
                        let field_type = declared
                            .field(name)
                            .map_or(Type::Error, |field| field.ty.clone());
                        format!("{name}: {}", self.literal(entry, &field_type))
                    })
                    .collect();
                format!("{{ {} }}", parts.join(", "))
            }
            (Value::Record(record), _) => {
                let parts: Vec<String> = record
                    .fields
                    .iter()
                    .map(|(name, entry)| format!("{name}: {}", self.literal(entry, &Type::Error)))
                    .collect();
                format!("{{ {} }}", parts.join(", "))
            }
            (Value::Table(code), _) => format!("tables.{code}"),
        }
    }

    fn item_literal(&self, value: &Value, index: usize, ty: &Type) -> String {
        match ty {
            Type::Record(record) if record.name.as_deref() == Some("Stop") => {
                self.stop_literal(value, index + 1, ty)
            }
            Type::Record(record) if record.name.as_deref() == Some("Commodity") => {
                self.commodity_literal(value, ty)
            }
            _ => self.literal(value, ty),
        }
    }

    /// A stop as a constructor call, leaving out what the constructor would fill in.
    fn stop_literal(&self, value: &Value, position: usize, ty: &Type) -> String {
        let (Value::Record(fields), Type::Record(declared)) = (value, ty.unwrapped()) else {
            return self.literal(value, ty);
        };
        let kind = match fields.get("kind") {
            Some(Value::Enum(kind)) => kind.to_string(),
            _ => String::from("Pickup"),
        };
        let Some((_, constructor)) = CONSTRUCTORS.iter().find(|(name, _)| *name == kind) else {
            return self.literal(value, ty);
        };
        let Value::Record(defaults) = stop(&kind, None) else {
            return self.literal(value, ty);
        };
        let mut without_dwell = (**fields).clone();
        without_dwell.set("dwell", Value::None);
        let computed = with_dwell(Value::Record(Rc::new(without_dwell)), false);
        let computed_dwell = match &computed {
            Value::Record(record) => record.get("dwell").cloned().unwrap_or(Value::None),
            _ => Value::None,
        };
        let mut args = Vec::new();
        for field in &declared.fields {
            let Some(actual) = fields.get(&field.name) else {
                continue;
            };
            if let (Value::Record(location), Some(Value::Record(blank)), Type::Record(shape)) =
                (actual, defaults.get(&field.name), &field.ty)
                && field.name == "location"
            {
                for part in &shape.fields {
                    let (Some(value), Some(default)) =
                        (location.get(&part.name), blank.get(&part.name))
                    else {
                        continue;
                    };
                    if !value.equals(default).unwrap_or(false) {
                        args.push(format!("{}: {}", part.name, self.literal(value, &part.ty)));
                    }
                }
                continue;
            }
            let skip = match field.name.as_str() {
                "kind" => true,
                "sequence" => {
                    matches!(actual, Value::Int(at) if *at == Decimal::from(position.max(1)))
                }
                "dwell" => actual.equals(&computed_dwell).unwrap_or(false),
                _ => defaults
                    .get(&field.name)
                    .is_some_and(|default| actual.equals(default).unwrap_or(false)),
            };
            if !skip {
                args.push(format!(
                    "{}: {}",
                    field.name,
                    self.literal(actual, &field.ty)
                ));
            }
        }
        format!("{constructor}({})", args.join(", "))
    }

    fn commodity_literal(&self, value: &Value, ty: &Type) -> String {
        let (Value::Record(fields), Type::Record(declared)) = (value, ty.unwrapped()) else {
            return self.literal(value, ty);
        };
        let Value::Record(defaults) = super::record::commodity() else {
            return self.literal(value, ty);
        };
        let args: Vec<String> = declared
            .fields
            .iter()
            .filter_map(|field| {
                let actual = fields.get(&field.name)?;
                let same = defaults
                    .get(&field.name)
                    .is_some_and(|default| actual.equals(default).unwrap_or(false));
                (!same).then(|| format!("{}: {}", field.name, self.literal(actual, &field.ty)))
            })
            .collect();
        format!("commodity({})", args.join(", "))
    }
}

/// The stops with their sequences cleared where they equal their position, as the constructors
/// would leave them before the record settles.
fn without_sequences(stops: &Value) -> Value {
    let Value::List(items) = stops else {
        return stops.clone();
    };
    Value::list(
        items
            .iter()
            .enumerate()
            .map(|(index, item)| match item {
                Value::Record(record)
                    if matches!(record.get("sequence"), Some(Value::Int(at)) if *at == Decimal::from(index + 1)) =>
                {
                    let mut changed: Record = (**record).clone();
                    changed.set("sequence", Value::None);
                    Value::Record(Rc::new(changed))
                }
                other => other.clone(),
            })
            .collect(),
    )
}

/// A variant, qualified by its enum when another schema enum has a variant of that name.
fn variant_literal(variant: &str, enumeration: &str) -> String {
    let shared = ENUMS
        .iter()
        .filter(|(_, variants)| variants.contains(&variant))
        .count()
        > 1;
    if shared {
        let short = enumeration.rsplit('.').next().unwrap_or(enumeration);
        format!("{short}.{variant}")
    } else {
        variant.to_owned()
    }
}

fn money_literal(amount: Decimal, per: &str) -> String {
    let sign = if amount.is_sign_negative() && !amount.is_zero() {
        "-"
    } else {
        ""
    };
    format!("{sign}${}{per}", money_digits(amount.abs()))
}

fn quantity_literal(quantity: &crate::tariff::units::Quantity) -> String {
    let unit = quantity.unit.to_string();
    if let Some(per) = unit.strip_prefix('$') {
        return money_literal(quantity.value, per);
    }
    let shown = format_quantity(quantity);
    if super::value::is_percent(quantity) {
        return shown;
    }
    format!("{} {unit}", format_decimal(quantity.value))
}

/// A text literal: quotes, backslashes, braces and control characters escaped.
#[must_use]
pub fn text_literal(text: &str) -> String {
    let mut out = String::with_capacity(text.len() + 2);
    out.push('"');
    for character in text.chars() {
        match character {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\t' => out.push_str("\\t"),
            '{' => out.push_str("{{"),
            '}' => out.push_str("}}"),
            control if control.is_control() => {
                let _ = write!(out, "\\u{{{:x}}}", u32::from(control));
            }
            other => out.push(other),
        }
    }
    out.push('"');
    out
}
