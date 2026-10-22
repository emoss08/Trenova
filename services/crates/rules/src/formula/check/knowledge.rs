//! What the checker knows about names: the record's fields and the template's inputs, as the
//! caller sends them.
//!
//! `variables` is the Go `DescribeSchema` variable list as published: dotted paths such as
//! `customer.name`, with lowercase JSON schema types. A field of an array's items is named with
//! `[]` after the array, `stops[].weight`; those type `.weight` and `#.weight` inside a predicate
//! over `stops`, and without them an item's fields are not known. `inputs` are the template's
//! declared inputs (`Number`, `String` or `Boolean`, any case).

use super::types::Type;
use crate::formula::catalog::RATING_TIME;
use serde::Deserialize;
use std::collections::HashMap;
use std::collections::HashSet;

/// The schema a formula is checked against.
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Schema {
    /// The record's fields, as `DescribeSchema` publishes them.
    #[serde(default)]
    pub variables: Vec<SchemaVariable>,
    /// The template's declared inputs.
    #[serde(default)]
    pub inputs: Vec<SchemaInput>,
    /// Word operators the schema publishes beyond `startsWith`, `endsWith`, `contains` and
    /// `matches`.
    #[serde(default)]
    pub operators: Vec<String>,
}

/// One field of the record.
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SchemaVariable {
    /// The dotted path it is read by; `list[].field` for a field of a list's items.
    pub name: String,
    /// Its JSON schema type: `number`, `integer`, `string`, `boolean`, `datetime`, `array`,
    /// `object`.
    #[serde(rename = "type", default)]
    pub kind: Option<String>,
    /// What it holds.
    #[serde(default)]
    pub description: Option<String>,
    /// The reference pane's grouping.
    #[serde(default)]
    pub category: Option<String>,
    /// It is empty on some records.
    #[serde(default)]
    pub nullable: Option<bool>,
    /// It is worked out from other fields.
    #[serde(default)]
    pub computed: Option<bool>,
    /// The values it may hold, when it is an enumeration.
    #[serde(default, rename = "enum")]
    pub values: Option<Vec<String>>,
}

/// One input the template declares.
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SchemaInput {
    /// Its name.
    pub name: String,
    /// `Number`, `String` or `Boolean`, in any case.
    #[serde(rename = "type", default)]
    pub kind: Option<String>,
}

/// A rate table the formula may read.
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct RateTable {
    /// The name `lookup` takes.
    pub name: String,
    /// How many axes it has: 1 or 2.
    pub axes: u8,
    /// Its first axis is in quantity bands, which `lookupInterp` and `deficitWeight` need.
    #[serde(default)]
    pub banded: bool,
}

/// A name the checker knows.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(super) struct Doc {
    pub name: String,
    pub ty: Type,
    pub nullable: bool,
    pub input: bool,
    /// The value a guard falls back to: `0`, `""`, `false`, `ratingTime`.
    pub zero: &'static str,
}

/// The fields of the record, or of a list's items.
#[derive(Debug, Clone, Default)]
pub(super) struct Namespace {
    docs: Vec<Doc>,
    by_name: HashMap<String, usize>,
    prefixes: HashSet<String>,
    roots: HashSet<String>,
}

impl Namespace {
    fn insert(&mut self, doc: Doc) {
        let mut cut = 0;
        while let Some(dot) = doc.name.get(cut..).and_then(|rest| rest.find('.')) {
            let prefix = doc.name.get(..cut + dot).unwrap_or_default().to_owned();
            if cut == 0 {
                self.roots.insert(prefix.clone());
            }
            self.prefixes.insert(prefix);
            cut += dot + 1;
        }
        if let Some(&index) = self.by_name.get(&doc.name) {
            if let Some(slot) = self.docs.get_mut(index) {
                *slot = doc;
            }
            return;
        }
        self.by_name.insert(doc.name.clone(), self.docs.len());
        self.docs.push(doc);
    }

    pub fn doc(&self, name: &str) -> Option<&Doc> {
        self.by_name
            .get(name)
            .and_then(|index| self.docs.get(*index))
    }

    /// Whether a path is an object other fields are under, such as `customer`.
    pub fn is_prefix(&self, path: &str) -> bool {
        self.prefixes.contains(path)
    }

    /// Whether a name is the first part of dotted fields.
    pub fn is_root(&self, name: &str) -> bool {
        self.roots.contains(name)
    }

    /// Every name, in the order the schema lists them.
    pub fn names(&self) -> impl Iterator<Item = &str> {
        self.docs.iter().map(|doc| doc.name.as_str())
    }

    pub fn is_empty(&self) -> bool {
        self.docs.is_empty()
    }
}

/// Everything the checker knows about names.
#[derive(Debug, Clone, Default)]
pub(super) struct Knowledge {
    pub top: Namespace,
    pub elements: HashMap<String, Namespace>,
}

fn zero_for(raw: &str) -> &'static str {
    match raw.to_ascii_lowercase().as_str() {
        "boolean" | "bool" => "false",
        "string" | "text" => "\"\"",
        "date" | "datetime" | "time" | "timestamp" => RATING_TIME,
        "array" | "list" => "[]",
        "object" | "map" => "{}",
        _ => "0",
    }
}

impl Knowledge {
    pub fn new(schema: &Schema) -> Self {
        let mut knowledge = Self::default();
        for variable in &schema.variables {
            let raw = variable.kind.as_deref().unwrap_or_default();
            let mut doc = Doc {
                name: variable.name.clone(),
                ty: Type::from_schema(raw),
                nullable: variable.nullable.unwrap_or(false),
                input: false,
                zero: zero_for(raw),
            };
            if let Some(at) = variable.name.rfind("[].") {
                let list = variable.name.get(..at).unwrap_or_default().to_owned();
                variable
                    .name
                    .get(at + 3..)
                    .unwrap_or_default()
                    .clone_into(&mut doc.name);
                if !doc.name.is_empty() {
                    knowledge.elements.entry(list).or_default().insert(doc);
                }
                continue;
            }
            knowledge.top.insert(doc);
        }
        for input in &schema.inputs {
            if input.name.is_empty() {
                continue;
            }
            let raw = input.kind.as_deref().unwrap_or("Any");
            knowledge.top.insert(Doc {
                name: input.name.clone(),
                ty: Type::from_schema(raw),
                nullable: false,
                input: true,
                zero: zero_for(raw),
            });
        }
        knowledge
    }

    /// The fields of a list's items, when the schema publishes them and has any.
    pub fn element(&self, list: &str) -> Option<&Namespace> {
        self.elements
            .get(list)
            .filter(|namespace| !namespace.is_empty())
    }
}
