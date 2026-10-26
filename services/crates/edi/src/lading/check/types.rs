//! Written types to checker types.

use trenova_module_kit::diagnostic::{Diagnostic, nearest};
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

use crate::lading::schema;
use crate::lading::syntax::ast::{TypeExpr, TypeExprKind};
use crate::lading::types::Type;

/// Every type name a file may write.
const BUILT_IN: &[&str] = &[
    "Text",
    "Int",
    "Decimal",
    "Bool",
    "Date",
    "Time",
    "DateTime",
    "Zone",
    "Duration",
    "Percent",
    "Money",
    "Weight",
    "Length",
    "Volume",
    "Temperature",
    "Json",
    "Code",
    "Segment",
    "Loop",
];

/// The type a written type stands for, and what is wrong with it.
#[must_use]
pub fn resolve(ty: &TypeExpr) -> (Type, Vec<Diagnostic>) {
    let mut diagnostics = Vec::new();
    let resolved = resolve_into(ty, &mut diagnostics);
    (resolved, diagnostics)
}

fn resolve_into(ty: &TypeExpr, diagnostics: &mut Vec<Diagnostic>) -> Type {
    match &ty.kind {
        TypeExprKind::Error => Type::Any,
        TypeExprKind::List(item) => resolve_into(item, diagnostics).list(),
        TypeExprKind::Map(value) => Type::Map(Box::new(resolve_into(value, diagnostics))),
        TypeExprKind::Optional(inner) => resolve_into(inner, diagnostics).optional(),
        TypeExprKind::Named(name, args) => {
            match (name.as_str(), args.as_slice()) {
                ("Code", [element]) => return Type::Code(element.clone()),
                ("Segment", [segment]) => return Type::SegmentId(segment.clone()),
                ("Loop", [id]) => return Type::LoopId(id.clone()),
                ("Code" | "Segment" | "Loop", _) => {
                    diagnostics.push(Diagnostic::error(
                        "type-arguments",
                        ty.span,
                        Message::with(
                            message!("{0} takes one argument: {0}<S502>"),
                            [Arg::from(name.as_str())],
                        ),
                    ));
                    return Type::Any;
                }
                _ => {}
            }
            if !args.is_empty() {
                diagnostics.push(Diagnostic::error(
                    "type-arguments",
                    ty.span,
                    Message::with(
                        message!("{0} takes no arguments"),
                        [Arg::from(name.as_str())],
                    ),
                ));
            }
            if let Some(resolved) = schema::named_type(name) {
                return resolved;
            }
            let known = BUILT_IN
                .iter()
                .copied()
                .chain(schema::ENUMS.iter().map(|item| item.name))
                .chain(schema::LOOKUPS.iter().map(|item| item.name))
                .chain(schema::RECORDS.iter().map(|item| item.name));
            let mut diagnostic = Diagnostic::error(
                "unknown-type",
                ty.span,
                Message::with(
                    message!("There is no type named {0}"),
                    [Arg::from(name.as_str())],
                ),
            );
            if let Some(close) = nearest(name, known) {
                diagnostic = diagnostic.with_fix(super::replace_fix(ty.span, close));
            }
            diagnostics.push(diagnostic);
            Type::Any
        }
    }
}
