//! Type lowering: a [`TypeExpr`] as written becomes the [`Type`] the checker reasons with.
//!
//! The built-in names (`Bool`, `Int`, `Number`, `Percent`, `Money`, `Text`, `Date`,
//! `DateTime`, the dimensions, `Count` for `Int`, `List<T>`) are fixed; everything else is a
//! record or enum the [`TypeScope`] knows (the schema's `Stop`, `Commodity` and enums, the
//! module's own declarations, imported ones). A quotient is a rate or a density:
//! `Money/cwt` is `Money/Weight` shown per hundredweight, `lb/ft3` is `Weight/Volume` shown in
//! pounds per cubic foot. Records written inline are structural.

use super::describe::{default_unit, quote, single};
use super::fixes::replace;
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::diagnostic::nearest;
use crate::tariff::docs::Doc;
use crate::tariff::syntax::ast::{FieldType, Name, TypeExpr, TypeExprKind};
use crate::tariff::types::{FieldDef, RecordType, Type};
use crate::tariff::units::{Base, Dim, Unit, simple};
use std::sync::Arc;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

/// The built-in type names, with what each is, for completion and the catalog.
pub const BUILTIN_TYPES: &[(&str, &str)] = &[
    ("Bool", message!("`true` or `false`.")),
    (
        "Int",
        message!("A whole number: counts, sequences, indexes."),
    ),
    (
        "Number",
        message!("An exact decimal of up to 28 significant digits. An `Int` is also a `Number`."),
    ),
    (
        "Percent",
        message!("A percent such as `18%`, which multiplies as 0.18."),
    ),
    (
        "Money",
        message!("Money in the rating currency, written `$450`."),
    ),
    ("Text", message!("Unicode text, written in double quotes.")),
    ("Date", message!("A calendar date, written `@2026-10-22`.")),
    (
        "DateTime",
        message!("An instant with its UTC offset, written `@2026-10-22T08:00-05:00`."),
    ),
    (
        "Time",
        message!(
            "A time of day on a clock, with no date and no offset, written `@17:30`; `at.time` is a date-time's."
        ),
    ),
    (
        "Weight",
        message!("A weight: `lb`, `cwt`, `kg`, `t`, `ton`."),
    ),
    ("Distance", message!("A road distance: `mi`, `km`.")),
    (
        "Length",
        message!("A linear length or dimension: `in`, `ft`, `m`."),
    ),
    ("Volume", message!("A volume: `ft3`, `m3`, `gal`, `l`.")),
    ("Duration", message!("Elapsed time: `min`, `h`, `d`.")),
    (
        "Temperature",
        message!("A temperature in `°F` or `°C`, for comparison and difference only."),
    ),
    (
        "Count",
        message!("A count of things: another name for `Int`."),
    ),
    (
        "List",
        message!("A list of values of one type: `List<Stop>`."),
    ),
];

/// Resolves the type names a module can see besides the built-ins.
pub trait TypeScope {
    /// The record or enum type named `name`.
    fn named_type(&self, name: &str) -> Option<Type>;
    /// Every record and enum name, for "did you mean".
    fn type_names(&self) -> Vec<String>;
}

/// The built-in type a name names, without arguments.
#[must_use]
pub fn builtin(name: &str) -> Option<Type> {
    Some(match name {
        "Bool" => Type::Bool,
        "Int" | "Count" => Type::Int,
        "Number" => Type::Number,
        "Percent" => Type::Percent,
        "Money" => Type::money(),
        "Text" => Type::Text,
        "Date" => Type::Date,
        "DateTime" => Type::DateTime,
        "Time" => Type::Time,
        other => Type::of(Base::from_name(other)?),
    })
}

/// Lowers type expressions, reporting what is wrong with them.
#[allow(missing_debug_implementations)]
pub struct Lowerer<'a> {
    /// The names besides the built-ins.
    pub scope: &'a dyn TypeScope,
    /// Whether the type is written in the standard library, whose field docs are translated.
    pub std: bool,
    /// Where findings go.
    pub diagnostics: &'a mut Vec<Diagnostic>,
}

impl Lowerer<'_> {
    /// The type `ty` names. A part that cannot be lowered is [`Type::Error`], reported once.
    pub fn lower(&mut self, ty: &TypeExpr) -> Type {
        match &ty.kind {
            TypeExprKind::Missing => Type::Error,
            TypeExprKind::Optional(inner) => Type::optional(self.lower(inner)),
            TypeExprKind::Record(fields) => Type::Record(Arc::new(RecordType {
                name: None,
                fields: self.fields(fields),
            })),
            TypeExprKind::Named { name, args } => self.named(ty, name, args),
            TypeExprKind::Quotient {
                numerator,
                denominator,
            } => self.quotient(ty, numerator, denominator),
        }
    }

    /// The fields of a record type, each lowered, a repeated name reported.
    pub fn fields(&mut self, fields: &[FieldType]) -> Vec<FieldDef> {
        let mut out: Vec<FieldDef> = Vec::with_capacity(fields.len());
        let mut seen: Vec<&Name> = Vec::new();
        for field in fields {
            let lowered = self.lower(&field.ty);
            if let Some(first) = seen.iter().find(|name| name.text == field.name.text) {
                self.diagnostics.push(
                    Diagnostic::error(
                        "duplicate",
                        field.name.span,
                        Message::with(
                            message!("The record already has a field {0}"),
                            [quote(&field.name.text)],
                        ),
                    )
                    .with_related(first.span, Message::new(message!("The first one is here"))),
                );
                continue;
            }
            seen.push(&field.name);
            out.push(FieldDef {
                name: field.name.text.clone(),
                ty: lowered,
                doc: Doc::of(self.std, field.docs.as_ref()),
            });
        }
        out
    }

    fn named(&mut self, ty: &TypeExpr, name: &Name, args: &[TypeExpr]) -> Type {
        if name.text.is_empty() {
            return Type::Error;
        }
        if name.text == "List" {
            return match args {
                [item] => Type::list(self.lower(item)),
                [] => {
                    self.diagnostics.push(Diagnostic::error(
                        "type-arguments",
                        ty.span,
                        Message::with(
                            message!("A list type names what it holds: write {0}"),
                            [quote("List<Text>")],
                        ),
                    ));
                    Type::Error
                }
                [first, ..] => {
                    let item = self.lower(first);
                    self.diagnostics.push(Diagnostic::error(
                        "type-arguments",
                        ty.span,
                        Message::new(message!("A list holds one type of value: `List<T>`")),
                    ));
                    Type::list(item)
                }
            };
        }
        let found = builtin(&name.text).or_else(|| self.scope.named_type(&name.text));
        if !args.is_empty() {
            self.diagnostics.push(Diagnostic::error(
                "type-arguments",
                ty.span,
                Message::with(message!("{0} takes no type arguments"), [quote(&name.text)]),
            ));
        }
        if let Some(found) = found {
            return found;
        }
        if let Some(unit) = simple(&name.text) {
            self.unit_as_type(name, unit);
            return Type::Error;
        }
        self.unknown(name, &[]);
        Type::Error
    }

    fn unit_as_type(&mut self, name: &Name, unit: &crate::tariff::units::Simple) {
        let dimension = unit.base.map_or("Int", Base::name);
        self.diagnostics.push(
            Diagnostic::error(
                "unit-type",
                name.span,
                Message::with(
                    message!("{0} is a unit, not a type. A value in {0} is {1}: write {2}"),
                    [
                        quote(&name.text),
                        Arg::from(match unit.base {
                            Some(base) => super::describe::base_words(base),
                            None => Message::new(message!("a whole number")),
                        }),
                        quote(dimension),
                    ],
                ),
            )
            .with_fix(replace(
                Message::with(message!("Change to {0}"), [quote(dimension)]),
                name.span,
                dimension,
            )),
        );
    }

    /// Reports a type name nothing declares, with the closest known name as a fix.
    fn unknown(&mut self, name: &Name, extra: &[&str]) {
        let mut candidates: Vec<String> = BUILTIN_TYPES
            .iter()
            .map(|(name, _)| (*name).to_owned())
            .collect();
        candidates.extend(self.scope.type_names());
        candidates.extend(extra.iter().map(|text| (*text).to_owned()));
        let suggestion = nearest(&name.text, candidates.iter().map(String::as_str));
        let diagnostic = match suggestion {
            Some(suggestion) => Diagnostic::error(
                "unknown-type",
                name.span,
                Message::with(
                    message!("{0} isn't a type. Did you mean {1}?"),
                    [quote(&name.text), quote(suggestion)],
                ),
            )
            .with_fix(replace(
                Message::with(message!("Change to {0}"), [quote(suggestion)]),
                name.span,
                suggestion,
            )),
            None => Diagnostic::error(
                "unknown-type",
                name.span,
                Message::with(message!("{0} isn't a type"), [quote(&name.text)]),
            ),
        };
        self.diagnostics.push(diagnostic);
    }

    fn quotient(&mut self, ty: &TypeExpr, numerator: &Name, denominator: &Name) -> Type {
        let (Some(top), Some(bottom)) = (self.side(numerator), self.side(denominator)) else {
            return Type::Error;
        };
        if top.dim.has_temperature() || bottom.dim.has_temperature() {
            self.diagnostics.push(Diagnostic::error(
                "quotient",
                ty.span,
                Message::new(message!(
                    "A temperature can only be compared or subtracted, so it can't be part of a rate"
                )),
            ));
            return Type::Error;
        }
        let Some(dim) = top.dim.per(bottom.dim) else {
            return Type::Error;
        };
        if dim.is_none() || top.dim.is_none() {
            let written = format!("{}/{}", numerator.text, denominator.text);
            let mut diagnostic = Diagnostic::error(
                "quotient",
                ty.span,
                Message::with(
                    message!(
                        "{0} isn't a rate: a rate is money or a quantity per a unit or a dimension, such as {1}"
                    ),
                    [quote(&written), quote("Money/cwt")],
                ),
            );
            if dim.is_none() {
                diagnostic = diagnostic.with_fix(replace(
                    Message::with(message!("Change to {0}"), [quote("Number")]),
                    ty.span,
                    "Number",
                ));
            }
            self.diagnostics.push(diagnostic);
            return Type::Error;
        }
        match (top.unit, bottom.unit) {
            (Some(top), Some(bottom)) => Unit::parse(&format!("{top}/{bottom}"))
                .map_or_else(|_| Type::quantity(dim), |unit| Type::quantity_in(&unit)),
            _ => Type::quantity(dim),
        }
    }

    /// One side of a quotient: a dimension (`Money`, `Weight`) or a unit (`cwt`), with the unit
    /// text a display unit is built from.
    fn side(&mut self, name: &Name) -> Option<Side> {
        if name.text.is_empty() {
            return None;
        }
        if name.text == "Money" {
            return Some(Side {
                dim: Dim::money(),
                unit: Some(String::from("$")),
            });
        }
        if let Some(base) = Base::from_name(&name.text) {
            return Some(Side {
                dim: Dim::of(base),
                unit: None,
            });
        }
        if let Some(unit) = simple(&name.text) {
            return Some(Side {
                dim: unit.base.map_or(Dim::NONE, Dim::of),
                unit: Some(unit.symbol.to_owned()),
            });
        }
        let dims = [
            "Money", "Weight", "Distance", "Length", "Volume", "Duration", "lb", "cwt", "kg", "mi",
            "km", "ft", "ft3", "gal", "pallet", "piece", "stop",
        ];
        if builtin(&name.text).is_some() || self.scope.named_type(&name.text).is_some() {
            self.diagnostics.push(Diagnostic::error(
                "quotient",
                name.span,
                Message::with(
                    message!("{0} can't be part of a rate: only money, a dimension such as `Weight`, or a unit such as `cwt` can"),
                    [quote(&name.text)],
                ),
            ));
            return None;
        }
        self.unknown(name, &dims);
        None
    }
}

struct Side {
    dim: Dim,
    unit: Option<String>,
}

/// The display unit of a dimension that is one base, for building a rate's unit from a
/// dimension: `lb` for `Weight`.
#[must_use]
pub fn dimension_unit(dim: Dim) -> Option<&'static str> {
    single(dim).map(default_unit)
}
