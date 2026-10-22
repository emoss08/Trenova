//! Declared types, read from the tree: what an input's annotation means, so the value a host
//! sends for it can be decoded and checked.
//!
//! This is the part of type lowering the evaluator needs on its own (it runs whether or not the
//! checker ran): built-in types, `List<T>`, `T?`, inline records, the schema's records and
//! enums, records and enums declared in the program (nominal, qualified by module path), and
//! quotients (`Money/cwt` shows per hundredweight; `Money/Weight` and `Weight/Volume` name
//! dimensions; `lb/ft3` names a unit).

use super::interp::link::{Decl, Linked};
use crate::tariff::schema::Schema;
use crate::tariff::syntax::ast::{FieldType, TypeExpr, TypeExprKind};
use crate::tariff::types::{EnumType, FieldDef, RecordType, Type};
use crate::tariff::units::{Base, Dim, Unit};
use std::sync::Arc;

/// How deeply declared types may refer to each other before lowering gives up.
const MAX_DEPTH: usize = 16;

/// Lowers type expressions written in a program.
#[derive(Debug)]
pub struct Lowering<'p, 'a> {
    linked: &'p Linked<'a>,
    schema: &'p Schema,
}

impl<'p, 'a> Lowering<'p, 'a> {
    /// A lowering over a program and the schema.
    #[must_use]
    pub const fn new(linked: &'p Linked<'a>, schema: &'p Schema) -> Self {
        Self { linked, schema }
    }

    /// The type `expr` names in `module`; [`Type::Error`] for what names no type.
    #[must_use]
    pub fn lower(&self, module: usize, expr: &TypeExpr) -> Type {
        self.at(module, expr, 0)
    }

    fn at(&self, module: usize, expr: &TypeExpr, depth: usize) -> Type {
        if depth > MAX_DEPTH {
            return Type::Error;
        }
        match &expr.kind {
            TypeExprKind::Named { name, args } => {
                if name.text == "List" {
                    return args.first().map_or(Type::Error, |item| {
                        Type::list(self.at(module, item, depth + 1))
                    });
                }
                self.named(module, &name.text, depth)
            }
            TypeExprKind::Quotient {
                numerator,
                denominator,
            } => quotient(&numerator.text, &denominator.text),
            TypeExprKind::Record(fields) => Type::Record(Arc::new(RecordType {
                name: None,
                fields: self.fields(module, fields, depth),
            })),
            TypeExprKind::Optional(inner) => Type::optional(self.at(module, inner, depth + 1)),
            TypeExprKind::Missing => Type::Error,
        }
    }

    fn fields(&self, module: usize, fields: &[FieldType], depth: usize) -> Vec<FieldDef> {
        fields
            .iter()
            .map(|field| FieldDef {
                name: field.name.text.clone(),
                ty: self.at(module, &field.ty, depth + 1),
                doc: field.docs.as_ref().map(|doc| doc.text.clone()),
            })
            .collect()
    }

    fn named(&self, module: usize, name: &str, depth: usize) -> Type {
        if let Some(builtin) = builtin(name) {
            return builtin;
        }
        if let Some((at, decl)) = self.linked.resolve(module, name) {
            let qualified =
                |declared: &str| match self.linked.module(at).and_then(|m| m.path.as_deref()) {
                    Some(path) => format!("{path}.{declared}"),
                    None => declared.to_owned(),
                };
            return match decl {
                Decl::Type(declared) => Type::Record(Arc::new(RecordType {
                    name: Some(qualified(&declared.name.text)),
                    fields: self.fields(at, &declared.fields, depth),
                })),
                Decl::Enum(declared) => Type::Enum(Arc::new(EnumType {
                    name: qualified(&declared.name.text),
                    variants: declared
                        .variants
                        .iter()
                        .map(|variant| {
                            (
                                variant.name.text.clone(),
                                variant.docs.as_ref().map(|doc| doc.text.clone()),
                            )
                        })
                        .collect(),
                    doc: None,
                })),
                Decl::Fn(_) | Decl::Const(_) => Type::Error,
            };
        }
        if let Some(record) = self.schema.record(name) {
            return Type::Record(record);
        }
        self.schema
            .enumeration(name)
            .map_or(Type::Error, Type::Enum)
    }
}

fn builtin(name: &str) -> Option<Type> {
    Some(match name {
        "Bool" => Type::Bool,
        "Int" | "Count" => Type::Int,
        "Number" => Type::Number,
        "Percent" => Type::Percent,
        "Money" => Type::money(),
        "Text" => Type::Text,
        "Date" => Type::Date,
        "DateTime" => Type::DateTime,
        other => Type::of(Base::from_name(other)?),
    })
}

/// The dimension one side of a quotient names: a dimension name or a unit.
fn side(text: &str) -> Option<(Dim, Option<Unit>)> {
    if text == "Money" {
        return Some((Dim::money(), None));
    }
    if let Some(base) = Base::from_name(text) {
        return Some((Dim::of(base), None));
    }
    let unit = Unit::parse(text).ok()?;
    Some((unit.dim(), Some(unit)))
}

fn quotient(numerator: &str, denominator: &str) -> Type {
    let (Some((top, top_unit)), Some((bottom, bottom_unit))) = (side(numerator), side(denominator))
    else {
        return Type::Error;
    };
    let Some(dim) = top.per(bottom) else {
        return Type::Error;
    };
    let top_symbol = match (numerator, &top_unit) {
        ("Money", _) => Some(String::from("$")),
        (_, Some(unit)) => Some(unit.to_string()),
        _ => None,
    };
    match (top_symbol, bottom_unit) {
        (Some(top), Some(bottom)) => Unit::parse(&format!("{top}/{bottom}"))
            .map_or_else(|_| Type::quantity(dim), |unit| Type::quantity_in(&unit)),
        _ => Type::quantity(dim),
    }
}
