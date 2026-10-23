//! `tariff.signature`: the signature of the call the cursor is in, and which parameter it is
//! at.
//!
//! The call is found from the tokens (the innermost `(` not closed before the cursor, past any
//! list, record or index inside the arguments), so a call still being typed has help. Covered:
//! user and library functions (with the doc comment, and each parameter's doc from the
//! sentences naming it), module functions, built-in methods (a list method's lambda typed by
//! the list's items: `where(f: Stop => Bool)`), conversions, the test record constructors,
//! `charge(…)` and `chosen(…)` in tests, rate table methods and a charge's `(quantity:, rate:)`
//! options. The active parameter follows a named argument by its name and a positional one by
//! its place.

use super::choices::{CHOSEN, CHOSEN_DOC, is_chosen};
use super::complete::{call_with_open, callee_of};
use super::cursor::{Bracket, Scopes, enclosing, previous};
use super::describe::{
    Signature, function_doc, function_signature, method_signature, signature as build, type_text,
};
use super::names::receiver_type;
use super::session::Session;
use crate::message;
use crate::message::Message;
use crate::tariff::check::{CONSTRUCTORS, Callee, REQUESTED_DOC, TABLE_METHODS, location_record};
use crate::tariff::docs::{Doc, collapse};
use crate::tariff::syntax::token::TokenKind;
use crate::tariff::types::FieldDef;

/// The most brackets signature help looks out through to find a call.
const MAX_NESTING: usize = 16;

/// One signature.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Help {
    /// The signature and where each parameter is in it.
    pub signature: Signature,
    /// What it does.
    pub doc: Option<Doc>,
    /// Each parameter's doc, in order.
    pub params: Vec<Option<Doc>>,
}

/// The answer.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct SignatureHelp {
    /// The signatures (one, or none outside a call).
    pub signatures: Vec<Help>,
    /// The parameter the cursor is at, when it is at one.
    pub active_parameter: Option<usize>,
}

/// The signature help at byte `offset` of the session's root file.
#[must_use]
pub fn signature(session: &Session<'_>, offset: u32) -> SignatureHelp {
    let file = session.file();
    let tokens = &file.tokens;
    if super::cursor::in_literal(session.source(), tokens, offset) {
        return SignatureHelp::default();
    }
    let scopes = Scopes::of(session.source(), file);
    session.tick(scopes.exprs.len());
    let mut at = offset;
    for _ in 0..MAX_NESTING {
        let Some(bracket) = enclosing(session.source(), tokens, at) else {
            break;
        };
        let Some(open) = tokens.get(bracket.open) else {
            break;
        };
        if bracket.kind == '('
            && let Some(help) = call_help(session, &scopes, &bracket)
        {
            return help;
        }
        if bracket.kind == '{' && !is_record(&scopes, open.span.start) {
            break;
        }
        at = open.span.start;
    }
    SignatureHelp::default()
}

fn is_record(scopes: &Scopes<'_>, start: u32) -> bool {
    scopes
        .find(|expr| {
            expr.span.start == start
                && matches!(expr.kind, crate::tariff::syntax::ast::ExprKind::Record(_))
        })
        .is_some()
}

/// The help for the call (or charge options) a `(` opens.
fn call_help(
    session: &Session<'_>,
    scopes: &Scopes<'_>,
    bracket: &Bracket,
) -> Option<SignatureHelp> {
    let tokens = &session.file().tokens;
    let open = tokens.get(bracket.open)?;
    let label_before = previous(tokens, bracket.open)
        .and_then(|index| tokens.get(index))
        .is_some_and(|token| token.kind == TokenKind::Text);
    let help = if label_before && is_charge(session, open.span.start) {
        charge_options()
    } else {
        let call = call_with_open(scopes, open.span.start)?;
        if is_chosen(session, call) {
            chosen_help()
        } else {
            let callee = callee_of(session, call)?;
            callee_help(session, call, &callee)?
        }
    };
    let active = active(&help.signature, bracket);
    Some(SignatureHelp {
        signatures: vec![help],
        active_parameter: active,
    })
}

/// Whether the `(` at `open` is a charge statement's options.
fn is_charge(session: &Session<'_>, open: u32) -> bool {
    let located = super::cursor::locate(session.source(), session.file(), open);
    session.tick(located.visited);
    located.stmt().is_some_and(|stmt| {
        matches!(&stmt.kind, crate::tariff::syntax::ast::StmtKind::Charge(charge) if charge.label.span.end <= open)
    })
}

/// The parameter a bracket's part is: by name when the part is named, else by place.
fn active(signature: &Signature, bracket: &Bracket) -> Option<usize> {
    if let Some(name) = bracket.current_name() {
        return signature.params.iter().position(|param| param.name == name);
    }
    let named_before = bracket
        .named
        .iter()
        .filter(|(part, _)| *part < bracket.part)
        .count();
    let positional = bracket.part.saturating_sub(named_before);
    if named_before > 0 {
        let given: Vec<&str> = bracket
            .named
            .iter()
            .map(|(_, name)| name.as_str())
            .collect();
        return signature
            .params
            .iter()
            .position(|param| !given.contains(&param.name.as_str()));
    }
    (positional < signature.params.len()).then_some(positional)
}

fn charge_options() -> Help {
    let signature = build(
        "charge \"label\" ",
        &[
            (String::from("quantity"), String::from("Quantity"), None),
            (String::from("rate"), String::from("Money/unit"), None),
            (String::from("minimum"), String::from("Money"), None),
            (String::from("maximum"), String::from("Money"), None),
        ],
        "",
    );
    Help {
        signature,
        doc: Some(Doc::Message(Message::new(message!(
            "A charge billed as `rate × quantity`, both printed on the invoice line; give `= amount` instead to bill an amount."
        )))),
        params: vec![
            Some(Doc::Message(Message::new(message!(
                "How much is billed: the invoice line's quantity, in the rate's unit."
            )))),
            Some(Doc::Message(Message::new(message!(
                "The price per unit of the quantity: the invoice line's unit price."
            )))),
            Some(Doc::Message(Message::new(message!(
                "The least the line bills: a smaller amount is raised to it, and the line says the minimum applied."
            )))),
            Some(Doc::Message(Message::new(message!(
                "The most the line bills: a larger amount is lowered to it, and the line says the maximum applied."
            )))),
        ],
    }
}

/// A test's `chosen("Label")`.
fn chosen_help() -> Help {
    Help {
        signature: build(
            CHOSEN,
            &[(String::from("choice"), String::from("Text"), None)],
            "Text?",
        ),
        doc: Some(Doc::Message(Message::new(CHOSEN_DOC))),
        params: vec![Some(Doc::Message(Message::new(message!(
            "The choice's label, as written after `best` or `worst`."
        ))))],
    }
}

/// A test constructor's fields, a stop's location fields after its own, all optional.
fn constructor_help(session: &Session<'_>, constructor: &str) -> Option<Help> {
    let (_, record) = CONSTRUCTORS.iter().find(|(name, _)| *name == constructor)?;
    let shape = session.schema.record(record)?;
    let location = location_record(&shape);
    let all: Vec<&FieldDef> = shape
        .fields
        .iter()
        .chain(location.iter().flat_map(|location| location.fields.iter()))
        .collect();
    let fields: Vec<(String, String, Option<String>)> = all
        .iter()
        .map(|field| (field.name.clone(), type_text(&field.ty), None))
        .collect();
    let mut signature = build(constructor, &fields, record);
    for param in &mut signature.params {
        param.optional = true;
    }
    let params = all.iter().map(|field| field.doc.clone()).collect();
    Some(Help {
        signature,
        doc: Some(Doc::Message(Message::with(
            message!(
                "Builds a `{0}` for a test: every field is named and optional, and the sample record's values fill the rest."
            ),
            [crate::message::Arg::from(*record)],
        ))),
        params,
    })
}

fn callee_help(
    session: &Session<'_>,
    call: &crate::tariff::syntax::ast::Expr,
    callee: &Callee,
) -> Option<Help> {
    match callee {
        Callee::Function { module, name } => {
            let decl = super::describe::function_decl(session, *module, name)?;
            let doc = function_doc(session, *module, name);
            let params = decl
                .params
                .iter()
                .map(|param| {
                    let doc = doc.as_ref()?;
                    doc.param(&param.name.text)
                        .filter(|own| own.text() != collapse(doc.text()))
                })
                .collect();
            Some(Help {
                signature: function_signature(session, *module, decl),
                doc,
                params,
            })
        }
        Callee::Method(method) => {
            let receiver = receiver_type(session, call)?;
            let signature = method_signature(method, &receiver);
            let params = vec![None; signature.params.len()];
            Some(Help {
                signature,
                doc: Some(Doc::Message(Message::new(method.doc))),
                params,
            })
        }
        Callee::Constructor(constructor) => constructor_help(session, constructor),
        Callee::Requested => Some(Help {
            signature: build(
                "requested",
                &[(String::from("code"), String::from("CODE"), None)],
                "Bool",
            ),
            doc: Some(Doc::Message(Message::new(REQUESTED_DOC))),
            params: vec![None],
        }),
        Callee::ChargeLookup => Some(Help {
            signature: build(
                "charge",
                &[(String::from("line"), String::from("CODE | Text"), None)],
                "Money?",
            ),
            doc: Some(Doc::Message(Message::new(message!(
                "The amount the bill's lines with this code or label add up to, or `none` when there are none."
            )))),
            params: vec![None],
        }),
        Callee::BillCharge => Some(Help {
            signature: build(
                "bill.charge",
                &[(String::from("line"), String::from("CODE | Text"), None)],
                "Money?",
            ),
            doc: Some(Doc::Message(Message::new(
                crate::tariff::schema::BILL_CHARGE_DOC,
            ))),
            params: vec![None],
        }),
        Callee::Conversion(conversion) => {
            let (param, ty, ret, doc) = if *conversion == "Number" {
                (
                    "i",
                    "Int",
                    "Number",
                    message!("`Number(i)`: an `Int` as a `Number`."),
                )
            } else {
                (
                    "text",
                    "Text",
                    "Int?",
                    message!("`Int.parse(text)`: the whole number a text spells, or `none`."),
                )
            };
            Some(Help {
                signature: build(conversion, &[(param.to_owned(), ty.to_owned(), None)], ret),
                doc: Some(Doc::Message(Message::new(doc))),
                params: vec![None],
            })
        }
        Callee::Table { code, method } => table_help(session, code, method),
    }
}

fn table_help(session: &Session<'_>, code: &str, method: &str) -> Option<Help> {
    let table = session
        .context
        .tables
        .iter()
        .find(|table| table.code == code)?;
    let (_, doc) = TABLE_METHODS.iter().find(|(name, _)| *name == method)?;
    let schema = &session.schema;
    let params: Vec<(String, String, Option<String>)> = if method == "cell" {
        table
            .axes
            .iter()
            .enumerate()
            .map(|(index, axis)| {
                let name = if axis.label.is_empty() {
                    format!("key{}", index + 1)
                } else {
                    axis.label.clone()
                };
                (name, type_text(&axis.key_type(schema)), None)
            })
            .collect()
    } else {
        let key = table.axes.first().map_or_else(
            || String::from("?"),
            |axis| type_text(&axis.key_type(schema)),
        );
        vec![(String::from("x"), key, None)]
    };
    let value = table.value_type.clone();
    let key = params
        .first()
        .map_or("?", |(_, ty, _)| ty.as_str())
        .to_owned();
    let ret = match method {
        "cell" => format!("{{ value: {value}, minimum: Money?, deficit: Bool }}?"),
        "band" => format!("{{ min: {key}, max: {key}?, value: {value} }}?"),
        _ => format!("{value}?"),
    };
    let count = params.len();
    Some(Help {
        signature: build(&format!("tables.{code}.{method}"), &params, &ret),
        doc: Some(Doc::Message(Message::new(doc))),
        params: vec![None; count],
    })
}
