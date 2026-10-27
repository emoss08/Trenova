//! `tariff.catalog`: everything an author can name before declaring anything, with its docs:
//! the standard library's modules and declarations, the built-in types, every property and
//! method, the units, the keywords, rate table methods and the schema. The editor's
//! completion and hover and Nova's prompt read it; it is static data, built per call. Every doc
//! in it is a `Message` the hosts translate ([`crate::tariff::docs`]).

use super::describe::{default_unit, words};
use super::inputs::descriptor;
use super::lower::BUILTIN_TYPES;
use super::methods::METHODS;
use super::tables::TABLE_METHODS;
use super::{Analysis, SymbolKind};
use crate::tariff::context::Context;
use crate::tariff::docs::Doc;
use crate::tariff::schema::{BILL_CHARGE_DOC, BILL_DOC, Schema, Settings, TABLES_DOC, bill_type};
use crate::tariff::stdlib;
use crate::tariff::syntax::ast::{FileKind, ItemKind};
use crate::tariff::syntax::lexer::{CONTEXTUAL, KEYWORDS};
use crate::tariff::syntax::program::load;
use crate::tariff::types::Type;
use crate::tariff::units::SIMPLE_UNITS;
use serde_json::{Value as Json, json};

/// The names that mean something only in a test, after the rating: `total`, `lines`,
/// `reviews` and a condition's `applies`. With the lexer's contextual words ([`CONTEXTUAL`]) and
/// the words that start a construct ([`CONSTRUCT_WORDS`]) they are what the catalog lists as
/// `contextual`: ordinary names everywhere else, so a parameter may be called `to` and a binding
/// `option`.
pub const TEST_NAMES: &[&str] = &["total", "lines", "reviews", "applies"];

/// The words the parser reads as a construct only where one can start, and as names everywhere
/// else: `best`, `worst` and `option` (choices), `review`, `property`, `vary` and `assume`
/// (properties), and `nondecreasing` and `nonincreasing` (a property's monotone `expect`).
pub const CONSTRUCT_WORDS: &[&str] = &[
    "best",
    "worst",
    "option",
    "review",
    "property",
    "vary",
    "assume",
    "nondecreasing",
    "nonincreasing",
];

/// The catalog as JSON.
#[must_use]
pub fn catalog() -> Json {
    let schema = Schema::new(&Settings::default());
    json!({
        "modules": stdlib::MODULES.iter().map(|(path, source)| module(path, source)).collect::<Vec<_>>(),
        "types": BUILTIN_TYPES.iter().map(|(name, doc)| json!({ "name": name, "doc": Doc::message(doc) })).collect::<Vec<_>>(),
        "methods": METHODS.iter().map(|method| json!({
            "receiver": method.receiver.as_str(),
            "name": method.name,
            "kind": if method.property { "property" } else { "method" },
            "signature": method.signature(),
            "params": method.params.iter().map(|param| json!({
                "name": param.name,
                "type": param.ty,
                "default": param.default,
            })).collect::<Vec<_>>(),
            "returns": method.returns,
            "doc": Doc::message(method.doc),
        })).chain(std::iter::once(json!({
            "receiver": "Shipment",
            "name": super::REQUESTED,
            "kind": "method",
            "signature": super::REQUESTED_SIGNATURE,
            "params": [{"name": "code", "type": "CODE", "default": null}],
            "returns": "Bool",
            "doc": Doc::message(super::REQUESTED_DOC),
        }))).collect::<Vec<_>>(),
        "tableMethods": TABLE_METHODS.iter().map(|(name, doc)| json!({ "name": name, "doc": Doc::message(doc) })).collect::<Vec<_>>(),
        "units": SIMPLE_UNITS.iter().filter(|unit| unit.symbol != "$").map(|unit| json!({
            "unit": unit.symbol,
            "dimension": unit.dimension(),
            "factor": unit.factor().normalize().to_string(),
            "si": unit.base.map(|base| match base {
                crate::tariff::units::Base::Weight => "kg",
                crate::tariff::units::Base::Distance | crate::tariff::units::Base::Length => "m",
                crate::tariff::units::Base::Volume => "m3",
                crate::tariff::units::Base::Duration => "s",
                other => default_unit(other),
            }),
        })).collect::<Vec<_>>(),
        "keywords": KEYWORDS,
        "contextual": CONTEXTUAL.iter().chain(CONSTRUCT_WORDS).chain(TEST_NAMES).collect::<Vec<_>>(),
        "schema": schema_json(&schema),
    })
}

fn schema_json(schema: &Schema) -> Json {
    json!({
        "roots": schema.roots.iter().map(|(name, ty, doc)| json!({
            "name": name,
            "doc": doc,
            "type": ty.to_string(),
            "fields": fields(ty),
        })).chain([
            json!({
                "name": "tables",
                "doc": Doc::message(TABLES_DOC),
                "type": "tables",
                "fields": [],
            }),
            json!({
                "name": "bill",
                "doc": Doc::message(BILL_DOC),
                "type": "Bill",
                "fields": fields(&Type::Record(bill_type())),
                "methods": [{
                    "name": "charge",
                    "signature": "charge(line: CODE | Text) -> Money?",
                    "doc": Doc::message(BILL_CHARGE_DOC),
                }],
            }),
        ]).collect::<Vec<_>>(),
        "records": schema.records.iter().map(|record| json!({
            "name": record.name,
            "fields": fields(&Type::Record(record.clone())),
        })).collect::<Vec<_>>(),
        "enums": schema.enums.iter().map(|enumeration| json!({
            "name": enumeration.name,
            "variants": enumeration.variants.iter().map(|(name, _)| name.clone()).collect::<Vec<_>>(),
            "ordered": enumeration.ordered,
        })).collect::<Vec<_>>(),
    })
}

/// A record type's fields, with their types (as written and as descriptors), words and docs,
/// nested records expanded.
fn fields(ty: &Type) -> Json {
    let Type::Record(record) = ty.unwrapped() else {
        return json!([]);
    };
    Json::Array(
        record
            .fields
            .iter()
            .map(|field| {
                let nested = match field.ty.unwrapped() {
                    Type::Record(inner) if inner.name.is_none() => fields(&field.ty),
                    _ => json!([]),
                };
                json!({
                    "name": field.name,
                    "type": field.ty.to_string(),
                    "descriptor": descriptor(&field.ty),
                    "words": words(&field.ty),
                    "doc": field.doc,
                    "fields": nested,
                })
            })
            .collect(),
    )
}

/// One standard library module: its doc and each public declaration's kind, signature and doc.
fn module(path: &str, source: &str) -> Json {
    let context = Context::default();
    let program = load(source, Some(FileKind::Library), &context);
    let analysis: Analysis = super::check(&program, &context);
    let Some(root) = program.root() else {
        return json!({ "path": path, "doc": null, "declarations": [] });
    };
    let mut declarations = Vec::new();
    for item in &root.file.items {
        if item.private.is_some() {
            continue;
        }
        let doc = Doc::of(root.std, item.docs.as_ref());
        let entry = match &item.kind {
            ItemKind::Fn(decl) => {
                let symbol = find(&analysis, &decl.name.text, SymbolKind::Function);
                let ret = symbol
                    .and_then(|s| s.ty.clone())
                    .map_or_else(String::new, |t| t.to_string());
                let params: Vec<String> = decl
                    .params
                    .iter()
                    .map(|param| {
                        let ty = super::describe::snippet(&root.source, param.ty.span).to_owned();
                        match &param.default {
                            Some(default) => format!(
                                "{}: {ty} = {}",
                                param.name.text,
                                super::describe::snippet(&root.source, default.span)
                            ),
                            None => format!("{}: {ty}", param.name.text),
                        }
                    })
                    .collect();
                json!({ "kind": "fn", "name": decl.name.text, "signature": format!("fn {}({}) -> {ret}", decl.name.text, params.join(", ")), "doc": doc })
            }
            ItemKind::Const(decl) => {
                let ty = find(&analysis, &decl.name.text, SymbolKind::Const)
                    .and_then(|s| s.ty.clone())
                    .map_or_else(String::new, |t| t.to_string());
                json!({ "kind": "const", "name": decl.name.text, "signature": format!("const {}: {ty}", decl.name.text), "doc": doc })
            }
            ItemKind::Type(decl) => {
                let ty =
                    find(&analysis, &decl.name.text, SymbolKind::Type).and_then(|s| s.ty.clone());
                let body = match &ty {
                    Some(Type::Record(record)) => record
                        .fields
                        .iter()
                        .map(|f| format!("{}: {}", f.name, f.ty))
                        .collect::<Vec<_>>()
                        .join(", "),
                    _ => String::new(),
                };
                json!({ "kind": "type", "name": decl.name.text, "signature": format!("type {} = {{ {body} }}", decl.name.text), "doc": doc, "fields": ty.as_ref().map(fields) })
            }
            ItemKind::Enum(decl) => {
                let variants: Vec<String> =
                    decl.variants.iter().map(|v| v.name.text.clone()).collect();
                json!({ "kind": "enum", "name": decl.name.text, "signature": format!("enum {} {{ {} }}", decl.name.text, variants.join(", ")), "doc": doc, "variants": variants })
            }
            _ => continue,
        };
        declarations.push(entry);
    }
    json!({
        "path": path,
        "doc": Doc::of(root.std, root.file.docs.as_ref()),
        "declarations": declarations,
    })
}

fn find<'a>(analysis: &'a Analysis, name: &str, kind: SymbolKind) -> Option<&'a super::Symbol> {
    analysis.symbols.iter().find(|symbol| {
        symbol.name == name && symbol.kind == kind && symbol.module == Some(super::ModuleId::ROOT)
    })
}

#[cfg(test)]
mod tests {
    use super::{catalog, module};
    use serde_json::Value as Json;

    fn find<'a>(list: &'a Json, key: &str, value: &str) -> Option<&'a Json> {
        list.as_array()?
            .iter()
            .find(|item| item.get(key).and_then(Json::as_str) == Some(value))
    }

    #[test]
    fn the_catalog_lists_the_bill_after_the_roots() {
        let out = catalog();
        let roots = out
            .get("schema")
            .and_then(|schema| schema.get("roots"))
            .cloned()
            .unwrap_or(Json::Null);
        let bill = find(&roots, "name", "bill").cloned().unwrap_or(Json::Null);
        assert_eq!(bill.get("type").and_then(Json::as_str), Some("Bill"));
        assert!(find(bill.get("fields").unwrap_or(&Json::Null), "name", "freight").is_some());
        assert!(find(bill.get("methods").unwrap_or(&Json::Null), "name", "charge").is_some());
    }

    #[test]
    fn the_catalog_holds_every_part() {
        let out = catalog();
        let modules = out
            .get("modules")
            .and_then(Json::as_array)
            .map_or(0, Vec::len);
        assert_eq!(modules, crate::tariff::stdlib::MODULES.len());
        let methods = out.get("methods").cloned().unwrap_or(Json::Null);
        let round_up = methods
            .as_array()
            .and_then(|list| {
                list.iter().find(|m| {
                    m.get("name").and_then(Json::as_str) == Some("roundUp")
                        && m.get("receiver").and_then(Json::as_str) == Some("Money")
                })
            })
            .and_then(|m| m.get("signature"))
            .and_then(Json::as_str);
        assert_eq!(round_up, Some("roundUp(to: Money) -> Money"));
        let units = out.get("units").cloned().unwrap_or(Json::Null);
        let pound = find(&units, "unit", "lb").cloned().unwrap_or(Json::Null);
        assert_eq!(
            pound.get("factor").and_then(Json::as_str),
            Some("0.45359237")
        );
        assert_eq!(
            pound.get("dimension").and_then(Json::as_str),
            Some("Weight")
        );
        let pallet = find(&units, "unit", "pallet")
            .cloned()
            .unwrap_or(Json::Null);
        assert_eq!(
            pallet.get("dimension").and_then(Json::as_str),
            Some("Count")
        );
        let keywords = out
            .get("keywords")
            .and_then(Json::as_array)
            .cloned()
            .unwrap_or_default();
        assert!(keywords.contains(&Json::from("charge")));
        let contextual = out
            .get("contextual")
            .and_then(Json::as_array)
            .cloned()
            .unwrap_or_default();
        for word in [
            "as",
            "cents",
            "to",
            "up",
            "down",
            "half",
            "even",
            "on",
            "freight",
            "best",
            "worst",
            "option",
            "review",
            "property",
            "vary",
            "assume",
            "nondecreasing",
            "nonincreasing",
            "total",
            "lines",
            "reviews",
            "applies",
        ] {
            assert!(
                contextual.contains(&Json::from(word)),
                "{word} is contextual"
            );
            assert!(
                !keywords.contains(&Json::from(word)),
                "{word} is not reserved"
            );
        }
        let schema = out.get("schema").cloned().unwrap_or(Json::Null);
        let roots = schema.get("roots").cloned().unwrap_or(Json::Null);
        assert!(find(&roots, "name", "shipment").is_some());
        assert!(find(&roots, "name", "tables").is_some());
        let stop = find(schema.get("records").unwrap_or(&Json::Null), "name", "Stop")
            .cloned()
            .unwrap_or(Json::Null);
        let dwell = find(stop.get("fields").unwrap_or(&Json::Null), "name", "dwell")
            .cloned()
            .unwrap_or(Json::Null);
        assert_eq!(dwell.get("type").and_then(Json::as_str), Some("Duration?"));
        let types = out.get("types").cloned().unwrap_or(Json::Null);
        assert!(find(&types, "name", "Money").is_some());
    }

    #[test]
    fn the_catalog_marks_the_ordered_enums_and_lists_times_and_tiers() {
        let out = catalog();
        let schema = out.get("schema").cloned().unwrap_or(Json::Null);
        let enums = schema.get("enums").cloned().unwrap_or(Json::Null);
        let ordered = |name: &str| {
            find(&enums, "name", name)
                .and_then(|found| found.get("ordered").and_then(Json::as_bool))
        };
        assert_eq!(ordered("FreightClass"), Some(true));
        assert_eq!(ordered("Weekday"), Some(true));
        assert_eq!(ordered("StopKind"), Some(false));
        let types = out.get("types").cloned().unwrap_or(Json::Null);
        assert!(find(&types, "name", "Time").is_some());
        let methods = out.get("methods").cloned().unwrap_or(Json::Null);
        let receivers: Vec<&str> = methods
            .as_array()
            .map(|list| {
                list.iter()
                    .filter_map(|method| method.get("receiver").and_then(Json::as_str))
                    .collect()
            })
            .unwrap_or_default();
        assert!(receivers.contains(&"Time") && receivers.contains(&"FreightClass"));
        assert!(
            find(
                out.get("modules").unwrap_or(&Json::Null),
                "path",
                "std.tiers"
            )
            .is_some()
        );
    }

    /// Every `doc` under `value`, with its JSON path.
    fn docs(value: &Json, path: &str, out: &mut Vec<(String, Json)>) {
        match value {
            Json::Object(map) => {
                for (key, inner) in map {
                    let at = format!("{path}.{key}");
                    if key == "doc" {
                        out.push((at.clone(), inner.clone()));
                    }
                    docs(inner, &at, out);
                }
            }
            Json::Array(list) => {
                for (index, inner) in list.iter().enumerate() {
                    docs(inner, &format!("{path}[{index}]"), out);
                }
            }
            _ => {}
        }
    }

    #[test]
    fn every_standard_library_and_schema_doc_is_a_message() {
        let out = catalog();
        let mut found = Vec::new();
        for part in ["modules", "schema", "types", "methods", "tableMethods"] {
            docs(out.get(part).unwrap_or(&Json::Null), part, &mut found);
        }
        let strings: Vec<&String> = found
            .iter()
            .filter(|(_, doc)| {
                !doc.is_null() && doc.get("message").and_then(Json::as_str).is_none()
            })
            .map(|(path, _)| path)
            .collect();
        assert!(strings.is_empty(), "docs sent as text: {strings:?}");
        assert!(found.len() > 150, "{} docs", found.len());
        let ltl = find(out.get("modules").unwrap_or(&Json::Null), "path", "std.ltl")
            .cloned()
            .unwrap_or(Json::Null);
        assert_eq!(
            ltl.pointer("/doc/message").and_then(Json::as_str),
            Some(
                "Less-than-truckload pricing: weight breaks, deficit weight rating, billed weight and the NMFC density scale."
            )
        );
    }

    #[test]
    fn a_library_module_lists_its_public_declarations() {
        let source = "library \"org.x\"\n\n/// Weight breaks.\n\n/// One break.\ntype Break = { from: Weight, rate: Money/cwt }\nenum Zone { A, B }\nconst FLOOR: Money = $50\n/// Billed weight.\nfn billed(weight: Weight, to: Weight = 100 lb) -> Weight = weight.roundUp(to: to)\nprivate fn hidden() -> Int = 1\n";
        let out = module("org.x", source);
        assert_eq!(
            out.get("doc").and_then(Json::as_str),
            Some("Weight breaks.")
        );
        let declarations = out.get("declarations").cloned().unwrap_or(Json::Null);
        let signatures: Vec<&str> = declarations
            .as_array()
            .map(|list| {
                list.iter()
                    .filter_map(|d| d.get("signature").and_then(Json::as_str))
                    .collect()
            })
            .unwrap_or_default();
        assert_eq!(
            signatures,
            [
                "type Break = { from: Weight, rate: Money/cwt }",
                "enum Zone { A, B }",
                "const FLOOR: Money",
                "fn billed(weight: Weight, to: Weight = 100 lb) -> Weight",
            ]
        );
        let billed = find(&declarations, "name", "billed")
            .cloned()
            .unwrap_or(Json::Null);
        assert_eq!(
            billed.get("doc").and_then(Json::as_str),
            Some("Billed weight.")
        );
    }
}
