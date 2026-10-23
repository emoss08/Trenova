//! A file's inputs as the hosts see them: name, doc, type (display text and a structured
//! descriptor for building forms), whether a value is required, and the default as a wire
//! value, and its folded `where` constraint; and the validation of the input values a rate
//! agreement rule and its agreement set, against their types and constraints.
//!
//! Descriptors: `{"kind":"money"}`, `{"kind":"quantity","dimension":"Money/Distance","unit":
//! "$/mi"}`, `{"kind":"list","item":…}`, `{"kind":"optional","inner":…}`, `{"kind":"record",
//! "name":"Break","fields":[{"name","doc","type"}]}`, `{"kind":"enum","name","variants":[…]}`,
//! and `int`, `number`, `percent`, `text`, `bool`, `date`, `dateTime`, `time`. An enum's
//! descriptor also says whether its variants are `ordered` (`FreightClass`, `Weekday`), so a form
//! can offer a range of them. A field's `doc` is a
//! [`crate::tariff::docs::Doc`]: a `Message` for the schema's and the standard library's
//! fields, the author's text otherwise.
//!
//! Values are checked with the evaluator's wire codec ([`crate::tariff::wire::decode`]), which
//! reads the reference's "Wire values" exactly, so the checker and the evaluator agree on every
//! value by construction.

use super::consts::wire;
use super::decls::State;
use super::describe::{quote, unit_text};
use super::{InputInfo, ModuleAnalysis};
use crate::message;
use crate::message::Message;
use crate::tariff::diagnostic::nearest;
use crate::tariff::syntax::ast::ItemKind;
use crate::tariff::syntax::program::{Decl, Module};
use crate::tariff::types::Type;
use crate::tariff::units::Dim;
use serde_json::{Map, Value as Json, json};

/// The root's inputs, in declaration order.
#[must_use]
pub fn describe(root: &Module, state: &State, _: &[ModuleAnalysis]) -> Vec<InputInfo> {
    let mut out: Vec<InputInfo> = Vec::new();
    for (name, symbol) in &root.symbols {
        let Decl::Input(item) = symbol.decl else {
            continue;
        };
        let Some(item) = root.file.items.get(item) else {
            continue;
        };
        let ItemKind::Input(input) = &item.kind else {
            continue;
        };
        let ty = state
            .symbols
            .iter()
            .find(|known| known.module == Some(root.id) && known.span == Some(symbol.span))
            .and_then(|known| known.ty.clone())
            .unwrap_or(Type::Error);
        out.push(InputInfo {
            name: name.clone(),
            doc: item.docs.as_ref().map(|doc| doc.text.clone()),
            descriptor: descriptor(&ty),
            required: input.default.is_none(),
            default: state.defaults.get(&(root.id.0, name.clone())).map(wire),
            constraint: state
                .constraints
                .get(&(root.id.0, name.clone()))
                .map(super::decls::FoldedConstraint::to_wire),
            ty,
            span: symbol.span,
        });
    }
    out.sort_by_key(|input| input.span.start);
    out
}

/// A type as a form descriptor.
#[must_use]
pub fn descriptor(ty: &Type) -> Json {
    match ty {
        Type::Bool => json!({ "kind": "bool" }),
        Type::Int => json!({ "kind": "int" }),
        Type::Number => json!({ "kind": "number" }),
        Type::Percent => json!({ "kind": "percent" }),
        Type::Text => json!({ "kind": "text" }),
        Type::Date => json!({ "kind": "date" }),
        Type::DateTime => json!({ "kind": "dateTime" }),
        Type::Time => json!({ "kind": "time" }),
        Type::Quantity(quantity) if quantity.dim == Dim::money() => json!({ "kind": "money" }),
        Type::Quantity(quantity) => json!({
            "kind": "quantity",
            "dimension": quantity.dim.to_string(),
            "unit": unit_text(quantity),
        }),
        Type::List(item) => json!({ "kind": "list", "item": descriptor(item) }),
        Type::Optional(inner) => json!({ "kind": "optional", "inner": descriptor(inner) }),
        Type::Record(record) => json!({
            "kind": "record",
            "name": record.name.as_ref().map(|_| ty.to_string()),
            "fields": record.fields.iter().map(|field| json!({
                "name": field.name,
                "doc": field.doc,
                "type": descriptor(&field.ty),
            })).collect::<Vec<_>>(),
        }),
        Type::Enum(enumeration) => json!({
            "kind": "enum",
            "name": ty.to_string(),
            "variants": enumeration.variants.iter().map(|(name, _)| name.clone()).collect::<Vec<_>>(),
            "ordered": enumeration.ordered,
        }),
        Type::Never | Type::Error => json!({ "kind": "unknown" }),
    }
}

/// A problem with a contract's input value, at a path such as `rule.rate` or
/// `agreement.breaks[2].rate`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ValueFinding {
    /// Where the value is.
    pub path: String,
    /// What is wrong.
    pub message: Message,
}

/// Checks the values a rule and an agreement set against the inputs: unknown names, values
/// that do not fit, and required inputs neither sets.
#[must_use]
pub fn validate(
    inputs: &[InputInfo],
    rule: Option<&Map<String, Json>>,
    agreement: Option<&Map<String, Json>>,
) -> Vec<ValueFinding> {
    let mut out = Vec::new();
    for (scope, values) in [("rule", rule), ("agreement", agreement)] {
        let Some(values) = values else { continue };
        for (name, value) in values {
            let path = format!("{scope}.{name}");
            if let Some(input) = inputs.iter().find(|input| input.name == *name) {
                check_value(input, value, &path, &mut out);
            } else {
                let suggestion = nearest(name, inputs.iter().map(|input| input.name.as_str()));
                out.push(ValueFinding {
                    path,
                    message: match suggestion {
                        Some(close) => Message::with(
                            message!("{0} isn't an input of this tariff. Did you mean {1}?"),
                            [quote(name), quote(close)],
                        ),
                        None => Message::with(
                            message!("{0} isn't an input of this tariff"),
                            [quote(name)],
                        ),
                    },
                });
            }
        }
    }
    for input in inputs.iter().filter(|input| input.required) {
        let set = |values: Option<&Map<String, Json>>| {
            values.is_some_and(|values| values.contains_key(&input.name))
        };
        if !set(rule) && !set(agreement) {
            out.push(ValueFinding {
                path: format!("rule.{}", input.name),
                message: Message::with(
                    message!("{0} has no default, so the rule or the agreement must set it"),
                    [quote(&input.name)],
                ),
            });
        }
    }
    out
}

/// Checks one wire value against its input's type with the evaluator's own codec, so a value
/// the checker accepts is one the evaluator decodes, then against the input's constraint with
/// the evaluator's own test ([`crate::tariff::constraint::breach`]). A list item outside the
/// constraint is named by its index: `rule.drops[2]`.
fn check_value(input: &InputInfo, value: &Json, path: &str, out: &mut Vec<ValueFinding>) {
    let decoded = match crate::tariff::wire::decode(value, &input.ty, path) {
        Ok(decoded) => decoded,
        Err(error) => {
            out.push(ValueFinding {
                path: error.path,
                message: error.message,
            });
            return;
        }
    };
    if let Some(constraint) = &input.constraint
        && let Some(breach) = crate::tariff::constraint::breach(constraint, &decoded)
    {
        out.push(ValueFinding {
            path: format!("{path}{}", breach.suffix()),
            message: breach.message(constraint, &input.name),
        });
    }
}

#[cfg(test)]
mod tests {
    use super::descriptor;
    use crate::tariff::schema::{Schema, Settings};
    use crate::tariff::types::Type;
    use serde_json::Value as Json;

    #[test]
    fn a_schema_record_describes_its_fields_with_translated_docs() {
        let schema = Schema::new(&Settings::default());
        let stop = schema.record("Stop").map_or(Type::Error, Type::Record);
        let described = descriptor(&stop);
        let fields = described
            .get("fields")
            .and_then(Json::as_array)
            .cloned()
            .unwrap_or_default();
        assert!(!fields.is_empty());
        for field in &fields {
            assert!(
                field
                    .pointer("/doc/message")
                    .and_then(Json::as_str)
                    .is_some(),
                "{field}"
            );
        }
    }
}
