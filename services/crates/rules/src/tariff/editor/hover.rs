//! `tariff.hover`: what the thing under the cursor is.
//!
//! The contents are a `signature` (Tariff code), a `type`, a `unit` for quantities and a `doc`
//! (the author's doc comment as markdown, or a built-in's catalog text as a `Message`), plus,
//! when the last preview's trace is given, a binding's `value`. Covered: names of every kind
//! (with their declaration), fields of records, properties and methods, functions (a named
//! argument shows its parameter's doc), types, enums and variants, units (their dimension and
//! exact factor to SI), charge codes, rate tables, modules, keywords, and literals (`1_200 lb` is
//! `544.31 kg`, `@2026-10-22` falls on a `Thu`).

use super::choices::{CHOSEN, Label, choice_named, chosen_at, label_at};
use super::cursor::{Body, Scopes, token_at};
use super::describe::{
    constraint_doc, flat, keyword_doc, method_signature, si, symbol_signature, type_text, unit_doc,
};
use super::names::{Parameter, parameters};
use super::occurrences::{Subject, local_at, local_subject};
use super::session::Session;
use super::values::{Values, hover_json};
use super::vary::{varied_at, vary_type};
use crate::message;
use crate::message::{Arg, Message};
use crate::span::{Range, Span};
use crate::tariff::check::describe::{quote, snippet, unit_text};
use crate::tariff::check::lower::BUILTIN_TYPES;
use crate::tariff::check::methods;
use crate::tariff::check::{
    REQUESTED, REQUESTED_DOC, REQUESTED_SIGNATURE, SymbolId, SymbolKind, TABLE_METHODS,
};
use crate::tariff::docs::Doc;
use crate::tariff::eval::time::Date;
use crate::tariff::schema::TableShape;
use crate::tariff::syntax::ast::{Choose, Expect, Expr, ExprKind, ItemKind, PatternKind, StmtKind};
use crate::tariff::syntax::program::ModuleId;
use crate::tariff::syntax::token::TokenKind;
use crate::tariff::types::Type;
use crate::tariff::units::{Unit, convert, simple};
use rust_decimal::{Decimal, RoundingStrategy};
use serde_json::Value as Json;

/// What hover shows.
#[derive(Debug, Clone, PartialEq)]
pub struct Hover {
    /// The text hovered.
    pub range: Span,
    /// What it is, as Tariff code.
    pub signature: Option<String>,
    /// Its type.
    pub ty: Option<String>,
    /// The unit a quantity is written in.
    pub unit: Option<String>,
    /// Its documentation.
    pub doc: Option<Doc>,
    /// The value it had in the last preview: `{"value", "display", "times"}`.
    pub value: Option<Json>,
    /// For an input with a `where` clause, the values it accepts: `from $0.50/mi to $15/mi`.
    pub constraint: Option<Message>,
}

impl Hover {
    fn new(range: Span) -> Self {
        Self {
            range,
            signature: None,
            ty: None,
            unit: None,
            doc: None,
            value: None,
            constraint: None,
        }
    }

    fn typed(mut self, ty: Option<&Type>) -> Self {
        if let Some(ty) = ty.filter(|ty| !matches!(ty, Type::Error | Type::Never)) {
            self.ty = Some(type_text(ty));
            if let Type::Quantity(quantity) = ty.unwrapped() {
                self.unit = unit_text(quantity);
            }
        }
        self
    }
}

/// What the session's root file has at byte `offset`.
#[must_use]
pub fn hover(session: &Session<'_>, offset: u32, values: &Values) -> Option<Hover> {
    let file = session.file();
    let index = token_at(&file.tokens, offset)?;
    let token = file.tokens.get(index)?.clone();
    let scopes = Scopes::of(session.source(), file);
    session.tick(scopes.exprs.len());
    let site = Hovered {
        session,
        scopes: &scopes,
        values,
        offset,
    };
    let after_dot = super::cursor::previous(&file.tokens, index)
        .and_then(|before| file.tokens.get(before))
        .is_some_and(|before| matches!(snippet(session.source(), before.span), "." | "?."));
    match token.kind {
        TokenKind::Number | TokenKind::Money | TokenKind::Percent | TokenKind::Date => {
            site.literal(token.span)
        }
        TokenKind::Unit => site.unit(token.span),
        TokenKind::Keyword if !after_dot => {
            site.keyword(token.span).or_else(|| site.name(token.span))
        }
        TokenKind::Name | TokenKind::TypeName | TokenKind::Code | TokenKind::Keyword => {
            site.name(token.span)
        }
        TokenKind::Text => site.label(),
        _ => None,
    }
}

struct Hovered<'s, 'c> {
    session: &'s Session<'c>,
    scopes: &'s Scopes<'s>,
    values: &'s Values,
    offset: u32,
}

impl Hovered<'_, '_> {
    fn text(&self, span: Span) -> &str {
        snippet(self.session.source(), span)
    }

    fn keyword(&self, span: Span) -> Option<Hover> {
        let word = self.text(span);
        if word == "tables" {
            return None;
        }
        let doc = keyword_doc(word)?;
        let mut hover = Hover::new(span);
        hover.signature = Some(word.to_owned());
        hover.doc = Some(Doc::Message(Message::new(doc)));
        Some(hover)
    }

    fn unit(&self, span: Span) -> Option<Hover> {
        let word = self.text(span);
        let unit = simple(word)?;
        let mut hover = Hover::new(span);
        hover.signature = Some(word.to_owned());
        hover.ty = Some(match unit.base {
            Some(base) => base.name().to_owned(),
            None if word == "%" => String::from("Percent"),
            None => String::from("Count"),
        });
        hover.unit = Some(word.to_owned());
        hover.doc = Some(Doc::Message(unit_doc(unit)));
        Some(hover)
    }

    /// A literal: its type, and for a quantity its value in SI units.
    fn literal(&self, span: Span) -> Option<Hover> {
        let expr = self.scopes.find_last(|expr| {
            expr.span.start == span.start
                && matches!(
                    expr.kind,
                    ExprKind::Number { .. }
                        | ExprKind::Money { .. }
                        | ExprKind::Percent(_)
                        | ExprKind::Date(_)
                        | ExprKind::DateTime(_)
                )
        })?;
        let range = Span::new(expr.span.start, expr.span.end.max(span.end));
        let written = self.text(range).to_owned();
        let mut hover =
            Hover::new(range).typed(self.session.analysis.type_of(ModuleId::ROOT, expr.id));
        hover.signature = Some(written.clone());
        hover.doc = literal_doc(expr, &written).map(Doc::Message);
        Some(hover)
    }

    fn name(&self, span: Span) -> Option<Hover> {
        if let Some((at, id)) = self.session.symbol_at(self.offset) {
            return Some(self.symbol(at, id));
        }
        if let Some(local) = local_at(self.session, self.scopes, span) {
            return Some(match local_subject(self.session, local) {
                Subject::Symbol(id) => self.symbol(span, id),
                _ => self.local(span, &local.name, local.span),
            });
        }
        self.member(span)
            .or_else(|| self.variant(span))
            .or_else(|| self.argument(span))
            .or_else(|| self.code(span))
            .or_else(|| self.written_type(span))
            .or_else(|| self.module_path(span))
            .or_else(|| self.given(span))
            .or_else(|| self.tables(span))
            .or_else(|| self.keyword(span))
    }

    fn symbol(&self, at: Span, id: SymbolId) -> Hover {
        let session = self.session;
        let mut hover = Hover::new(at);
        let Some(symbol) = session.symbol(id) else {
            return hover;
        };
        hover.signature = Some(symbol_signature(session, id));
        let value_like = !matches!(
            symbol.kind,
            SymbolKind::Type | SymbolKind::Enum | SymbolKind::Module | SymbolKind::Test
        );
        if value_like {
            hover = hover.typed(symbol.ty.as_ref());
            hover.signature = Some(symbol_signature(session, id));
        }
        hover.doc = match symbol.kind {
            SymbolKind::Param => self.param_doc_of(id),
            _ => symbol.doc.clone(),
        };
        if symbol.kind == SymbolKind::Input {
            hover.constraint = self.constraint_of(symbol);
        }
        if symbol.module == Some(ModuleId::ROOT)
            && matches!(
                symbol.kind,
                SymbolKind::Let | SymbolKind::Var | SymbolKind::Binding
            )
        {
            hover.value = self.value_of(id);
        }
        hover
    }

    /// What an input's `where` clause accepts, in words, from the checker's folded constraint.
    fn constraint_of(&self, symbol: &crate::tariff::check::Symbol) -> Option<Message> {
        if symbol.module != Some(ModuleId::ROOT) {
            return None;
        }
        let span = symbol.span?;
        let input = self
            .session
            .analysis
            .inputs
            .iter()
            .find(|input| input.span == span)?;
        Some(constraint_doc(input.constraint.as_ref()?, &input.ty))
    }

    /// A local the checker has no symbol for: a property's `vary` name with the type of the
    /// values it takes, or a name in a part of the file the checker did not reach.
    fn local(&self, span: Span, name: &str, declared: Span) -> Hover {
        let mut hover = Hover::new(span);
        let Some(varied) = varied_at(self.session.file(), declared) else {
            hover.signature = Some(name.to_owned());
            return hover;
        };
        let ty = varied
            .values
            .and_then(|values| vary_type(self.session, values));
        hover = hover.typed(ty.as_ref());
        hover.signature = Some(match &hover.ty {
            Some(ty) => format!("vary {name}: {ty}"),
            None => format!("vary {name}"),
        });
        hover.doc = keyword_doc("vary").map(|doc| Doc::Message(Message::new(doc)));
        hover
    }

    /// A choice's or option's label, or the label a test's `chosen("…")` names.
    fn label(&self) -> Option<Hover> {
        let session = self.session;
        let file = session.file();
        if let Some(found) = label_at(file, self.offset) {
            return Some(match found {
                Label::Choice(choose) => choice_hover(choose.label.span, choose),
                Label::OptionOf(choose, option) => {
                    let mut hover = Hover::new(option.label.span);
                    hover.signature = Some(format!("option \"{}\"", option.label.text));
                    hover.doc = Some(Doc::Message(Message::with(
                        message!("One of the methods {0} chooses between."),
                        [quote(&format!(
                            "{} \"{}\"",
                            choose.kind.keyword(),
                            choose.label.text
                        ))],
                    )));
                    hover
                }
            });
        }
        let (span, label) = chosen_at(session, self.scopes, self.offset)?;
        if let Some(choose) = choice_named(file, &label) {
            let mut hover = choice_hover(span, choose);
            hover.ty = Some(String::from("Text?"));
            return Some(hover);
        }
        let mut hover = Hover::new(span);
        hover.signature = Some(format!("{CHOSEN}(\"{label}\")"));
        hover.ty = Some(String::from("Text?"));
        hover.doc = Some(Doc::Message(Message::with(
            message!("No `best` or `worst` in this file is labelled {0}."),
            [quote(&format!("\"{label}\""))],
        )));
        Some(hover)
    }

    /// A parameter's doc: the sentences of its function's doc that name it.
    fn param_doc_of(&self, id: SymbolId) -> Option<Doc> {
        let symbol = self.session.symbol(id)?;
        let module = symbol.module?;
        let span = symbol.span?;
        let found = self.session.program.module(module)?;
        found.file.items.iter().find_map(|item| match &item.kind {
            ItemKind::Fn(decl) if decl.params.iter().any(|param| param.name.span == span) => {
                Doc::of(found.std, item.docs.as_ref())?.param(&symbol.name)
            }
            _ => None,
        })
    }

    /// The value a binding had in the last preview.
    fn value_of(&self, id: SymbolId) -> Option<Json> {
        if self.values.is_empty() {
            return None;
        }
        let symbol = self.session.symbol(id)?;
        let span = symbol.span?;
        let utf16 = &self.session.root().utf16;
        let declared = utf16.to_utf16(span.start);
        if symbol.kind == SymbolKind::Var {
            let local = self.scopes.declared_at(span)?;
            let extent = match local.extent {
                super::cursor::Extent::Span(extent) => utf16.range(extent),
                super::cursor::Extent::TopLevel { from } => Range {
                    from: utf16.to_utf16(from),
                    to: u32::MAX,
                },
            };
            let (last, times) = self.values.last_in(&symbol.name, declared, extent)?;
            return Some(hover_json(last, times));
        }
        let bindings: Vec<_> = self.values.bindings(&symbol.name, declared).collect();
        let last = bindings.last()?;
        Some(hover_json(last, bindings.len()))
    }

    /// A field, property, method, table or table method after `.`.
    fn member(&self, span: Span) -> Option<Hover> {
        let field = self.scopes.find_last(
            |expr| matches!(&expr.kind, ExprKind::Field { name, .. } if name.span == span),
        )?;
        let ExprKind::Field { target, name, .. } = &field.kind else {
            return None;
        };
        let session = self.session;
        if let ExprKind::Name(root) = &target.kind
            && root.text == "tables"
        {
            let table = session
                .context
                .tables
                .iter()
                .find(|table| table.code == name.text)?;
            return Some(table_hover(span, table));
        }
        if let ExprKind::Field {
            target: root,
            name: code,
            ..
        } = &target.kind
            && matches!(&root.kind, ExprKind::Name(root) if root.text == "tables")
        {
            let (method, doc) = TABLE_METHODS
                .iter()
                .find(|(method, _)| *method == name.text)?;
            let mut hover = Hover::new(span);
            hover.signature = Some(format!("tables.{}.{method}(…)", code.text));
            hover.doc = Some(Doc::Message(Message::new(doc)));
            return Some(hover);
        }
        let receiver = session
            .analysis
            .type_of(ModuleId::ROOT, target.id)?
            .unwrapped()
            .clone();
        if let Type::Record(record) = &receiver
            && record.name.as_deref() == Some("Bill")
            && matches!(&target.kind, ExprKind::Name(root) if root.text == "bill")
            && name.text == "charge"
        {
            let mut hover = Hover::new(span);
            hover.signature = Some(String::from("bill.charge(line: CODE | Text) -> Money?"));
            hover.doc = Some(Doc::message(crate::tariff::schema::BILL_CHARGE_DOC));
            return Some(hover);
        }
        if let Type::Record(record) = &receiver
            && let Some(found) = record.field(&name.text)
        {
            let mut hover = Hover::new(span).typed(Some(&found.ty));
            hover.signature = Some(format!("{}: {}", found.name, type_text(&found.ty)));
            hover.doc.clone_from(&found.doc);
            return Some(hover);
        }
        if name.text == REQUESTED && session.schema.root("shipment") == Some(&receiver) {
            let mut hover = Hover::new(span);
            hover.signature = Some(String::from(REQUESTED_SIGNATURE));
            hover.doc = Some(Doc::Message(Message::new(REQUESTED_DOC)));
            return Some(hover);
        }
        let method = methods::find(&receiver, &name.text)?;
        let mut hover = Hover::new(span);
        hover.signature = Some(method_signature(method, &receiver).label);
        if method.property {
            hover = hover.typed(session.analysis.type_of(ModuleId::ROOT, field.id));
            hover.signature = Some(method_signature(method, &receiver).label);
        }
        hover.doc = Some(Doc::Message(Message::new(method.doc)));
        Some(hover)
    }

    /// An enum variant, written bare or qualified, as a value or a pattern; or the enum of a
    /// qualified variant.
    fn variant(&self, span: Span) -> Option<Hover> {
        let analysis = self.session.analysis.module(ModuleId::ROOT)?;
        if let Some(path) = self.scopes.find_last(|expr| {
            matches!(&expr.kind, ExprKind::Path { head, member }
                if head.span == span || member.as_ref().is_some_and(|m| m.span == span))
        }) && let ExprKind::Path { head, member } = &path.kind
        {
            let enumeration = analysis.variants.get(&path.id)?;
            let on_head = head.span == span && member.is_some();
            let mut hover = Hover::new(span);
            let enum_type = Type::Enum(enumeration.clone());
            if on_head {
                hover.signature = Some(enum_signature(&enum_type, enumeration));
                hover.doc.clone_from(&enumeration.doc);
                return Some(hover);
            }
            let variant = member.as_ref().unwrap_or(head);
            return Some(variant_hover(span, &enum_type, enumeration, &variant.text));
        }
        let pattern = self.pattern_variant(span)?;
        let enumeration = analysis.pattern_variants.get(&pattern.0)?;
        let enum_type = Type::Enum(enumeration.clone());
        Some(variant_hover(span, &enum_type, enumeration, &pattern.1))
    }

    /// The span start and name of a variant pattern whose variant is written at `span`.
    fn pattern_variant(&self, span: Span) -> Option<(u32, String)> {
        let located = super::cursor::locate(self.session.source(), self.session.file(), span.start);
        self.session.tick(located.visited);
        let pattern = located.pattern()?;
        match &pattern.kind {
            PatternKind::Variant { variant, .. } if variant.span == span => {
                Some((pattern.span.start, variant.text.clone()))
            }
            _ => None,
        }
    }

    /// A named argument: the parameter it names.
    fn argument(&self, span: Span) -> Option<Hover> {
        let call = self.scopes.find_last(|expr| {
            matches!(&expr.kind, ExprKind::Call { args, .. }
                if args.iter().any(|arg| arg.name.as_ref().is_some_and(|name| name.span == span)))
        })?;
        let callee = super::complete::callee_of(self.session, call)?;
        let name = self.text(span);
        let param: Parameter = parameters(self.session, call, &callee)
            .into_iter()
            .find(|param| param.name == name)?;
        let mut hover = Hover::new(span).typed(param.ty.as_ref());
        hover.signature = Some(match &param.ty {
            Some(ty) => format!("{}: {}", param.name, type_text(ty)),
            None => param.name.clone(),
        });
        hover.doc = param.doc;
        Some(hover)
    }

    /// A charge code: `as FSC`, `charge(FSC)`.
    fn code(&self, span: Span) -> Option<Hover> {
        let word = self.text(span);
        let codes = self.session.context.codes.as_ref()?;
        let code = codes.iter().find(|code| code.code == word)?;
        let mut hover = Hover::new(span);
        hover.signature = Some(code.code.clone());
        let description = if code.description.is_empty() {
            None
        } else {
            Some(code.description.clone())
        };
        hover.doc = Some(match (description, code.active) {
            (Some(description), true) => Doc::Text(description),
            (Some(description), false) => Doc::Message(Message::with(
                message!("{0} (inactive: it can't be billed)"),
                [Arg::from(description)],
            )),
            (None, true) => Doc::Message(Message::new(message!("An accessorial charge code."))),
            (None, false) => Doc::Message(Message::new(message!(
                "An inactive accessorial charge code: it can't be billed."
            ))),
        });
        Some(hover)
    }

    /// A type name in an annotation.
    fn written_type(&self, span: Span) -> Option<Hover> {
        let word = self.text(span);
        let session = self.session;
        let located = super::cursor::locate(session.source(), session.file(), span.start);
        located.ty()?;
        if let Some((_, doc)) = BUILTIN_TYPES.iter().find(|(name, _)| *name == word) {
            let mut hover = Hover::new(span);
            hover.signature = Some(word.to_owned());
            hover.doc = Some(Doc::Message(Message::new(doc)));
            return Some(hover);
        }
        if let Some(id) = session.global(ModuleId::ROOT, word) {
            let mut hover = self.symbol(span, id);
            hover.range = span;
            return Some(hover);
        }
        if let Some(record) = session.schema.record(word) {
            let mut hover = Hover::new(span);
            let fields: Vec<String> = record
                .fields
                .iter()
                .map(|field| format!("{}: {}", field.name, type_text(&field.ty)))
                .collect();
            hover.signature = Some(format!("type {word} = {{ {} }}", fields.join(", ")));
            return Some(hover);
        }
        let enumeration = session.schema.enumeration(word)?;
        let mut hover = Hover::new(span);
        hover.signature = Some(enum_signature(
            &Type::Enum(enumeration.clone()),
            &enumeration,
        ));
        hover.doc.clone_from(&enumeration.doc);
        Some(hover)
    }

    /// A part of a `use` path: the module so far.
    fn module_path(&self, span: Span) -> Option<Hover> {
        let session = self.session;
        let item = session.file().items.iter().find(|item| {
            matches!(&item.kind, ItemKind::Use(using) if using.path.iter().any(|part| part.span == span))
        })?;
        let ItemKind::Use(using) = &item.kind else {
            return None;
        };
        let upto = using.path.iter().position(|part| part.span == span)?;
        let path: Vec<&str> = using
            .path
            .iter()
            .take(upto + 1)
            .map(|part| part.text.as_str())
            .collect();
        let path = path.join(".");
        let mut hover = Hover::new(span);
        hover.signature = Some(format!("use {path}"));
        hover.doc = match path.as_str() {
            "std" => Some(Doc::Message(Message::new(message!(
                "The standard library, shipped with the module."
            )))),
            "org" => Some(Doc::Message(Message::new(message!(
                "Your organisation's libraries."
            )))),
            _ => session
                .program
                .module_named(&path)
                .and_then(|module| Doc::of(module.std, module.file.docs.as_ref())),
        };
        Some(hover)
    }

    /// A part of a `given` or `vary` path, or of a monotone `expect`'s target: the record field
    /// it names, or the input after `input.`.
    fn given(&self, span: Span) -> Option<Hover> {
        let located = super::cursor::locate(self.session.source(), self.session.file(), span.start);
        self.session.tick(located.visited);
        let stmt = located.stmt()?;
        let (StmtKind::Given { target, .. }
        | StmtKind::Vary { target, .. }
        | StmtKind::Expect(Expect::Monotone { target, .. })) = &stmt.kind
        else {
            return None;
        };
        let position = target.iter().position(|part| part.span == span)?;
        let (head, rest) = target.split_first()?;
        if let ("tables", [code]) = (head.text.as_str(), rest)
            && code.span == span
        {
            let table = self
                .session
                .context
                .tables
                .iter()
                .find(|table| table.code == code.text)?;
            return Some(table_hover(span, table));
        }
        if head.text == "input" {
            let name = rest.first().filter(|name| name.span == span)?;
            let id = self.session.global(ModuleId::ROOT, &name.text)?;
            return Some(self.symbol(span, id));
        }
        let mut ty = self.session.schema.root(&head.text)?.clone();
        let mut doc = None;
        for part in rest.iter().take(position) {
            let Type::Record(record) = ty.unwrapped().clone() else {
                return None;
            };
            let field = record.field(&part.text)?;
            doc.clone_from(&field.doc);
            ty = field.ty.clone();
        }
        let mut hover = Hover::new(span).typed(Some(&ty));
        let written: Vec<&str> = target
            .iter()
            .take(position + 1)
            .map(|part| part.text.as_str())
            .collect();
        hover.signature = Some(format!("{}: {}", written.join("."), type_text(&ty)));
        hover.doc = doc;
        Some(hover)
    }

    /// `tables` itself.
    fn tables(&self, span: Span) -> Option<Hover> {
        if self.text(span) != "tables" {
            return None;
        }
        let located = super::cursor::locate(self.session.source(), self.session.file(), span.start);
        if !matches!(located.body, Body::Top | Body::Test) {
            return None;
        }
        let mut hover = Hover::new(span);
        hover.signature = Some(String::from("tables"));
        let codes: Vec<&str> = self
            .session
            .context
            .tables
            .iter()
            .map(|table| table.code.as_str())
            .collect();
        hover.doc = Some(Doc::Message(Message::with(
            message!("The organisation's active rate matrices, by code: {0}"),
            [Arg::from(codes.join(", "))],
        )));
        Some(hover)
    }
}

/// A choice: its keyword and label, and the options it chooses between.
fn choice_hover(span: Span, choose: &Choose) -> Hover {
    let mut hover = Hover::new(span);
    hover.signature = Some(format!(
        "{} \"{}\"",
        choose.kind.keyword(),
        choose.label.text
    ));
    let options: Vec<String> = choose
        .options
        .iter()
        .map(|option| format!("\"{}\"", option.label.text))
        .collect();
    let template = match choose.kind {
        crate::tariff::syntax::ast::ChooseKind::Best => {
            message!("Bills the option with the lowest subtotal of {0}.")
        }
        crate::tariff::syntax::ast::ChooseKind::Worst => {
            message!("Bills the option with the highest subtotal of {0}.")
        }
    };
    hover.doc = Some(Doc::Message(Message::with(
        template,
        [Arg::from(options.join(", "))],
    )));
    hover
}

fn enum_signature(ty: &Type, enumeration: &crate::tariff::types::EnumType) -> String {
    let variants: Vec<&str> = enumeration
        .variants
        .iter()
        .map(|(name, _)| name.as_str())
        .collect();
    format!("enum {ty} {{ {} }}", variants.join(", "))
}

fn variant_hover(
    span: Span,
    ty: &Type,
    enumeration: &crate::tariff::types::EnumType,
    variant: &str,
) -> Hover {
    let mut hover = Hover::new(span);
    hover.signature = Some(format!("{ty}.{variant}"));
    hover.ty = Some(ty.to_string());
    let doc = enumeration
        .variants
        .iter()
        .find(|(name, _)| name == variant)
        .and_then(|(_, doc)| doc.clone())
        .or_else(|| enumeration.doc.clone());
    hover.doc = doc;
    hover
}

fn table_hover(span: Span, table: &TableShape) -> Hover {
    let mut hover = Hover::new(span);
    let axes: Vec<String> = table
        .axes
        .iter()
        .map(|axis| {
            let label = if axis.label.is_empty() {
                axis.kind.as_str()
            } else {
                axis.label.as_str()
            };
            match &axis.unit {
                Some(unit) => format!("{label}: {} ({}, {unit})", axis.kind, axis.match_mode),
                None => format!("{label}: {} ({})", axis.kind, axis.match_mode),
            }
        })
        .collect();
    hover.signature = Some(format!("tables.{}[{}]", table.code, axes.join(", ")));
    hover.ty = Some(format!("{}?", flat(&table.value_type)));
    hover.doc = (!table.name.is_empty()).then(|| Doc::Text(table.name.clone()));
    hover
}

/// What a literal is worth in plainer terms: a quantity in SI units, a percent as a factor, a
/// date's weekday.
fn literal_doc(expr: &Expr, written: &str) -> Option<Message> {
    match &expr.kind {
        ExprKind::Number {
            value,
            unit: Some(unit),
            ..
        } => {
            let from = Unit::parse(&unit.text()).ok()?;
            let base = from.factors().first()?.0.base?;
            if from.factors().len() != 1 || base == crate::tariff::units::Base::Temperature {
                return None;
            }
            let target = Unit::parse(si_unit(base)).ok()?;
            if from == target {
                return None;
            }
            let converted = convert(*value, &from, &target).ok()?;
            Some(Message::with(
                message!("`{0}` is `{1} {2}`"),
                [
                    Arg::from(written.to_owned()),
                    Arg::from(shown(converted)),
                    Arg::from(si_unit(base)),
                ],
            ))
        }
        ExprKind::Percent(value) => Some(Message::with(
            message!("`{0}` multiplies as {1}"),
            [
                Arg::from(written.to_owned()),
                Arg::from(
                    value
                        .checked_div(Decimal::ONE_HUNDRED)
                        .unwrap_or_default()
                        .normalize()
                        .to_string(),
                ),
            ],
        )),
        ExprKind::Date(date)
        | ExprKind::DateTime(crate::tariff::syntax::ast::DateTimeLit { date, .. }) => {
            let day = Date::from_civil(
                i64::from(date.year),
                i64::from(date.month),
                i64::from(date.day),
            )?;
            Some(Message::with(
                message!("`{0}` falls on a `{1}`"),
                [Arg::from(written.to_owned()), Arg::from(day.weekday())],
            ))
        }
        _ => None,
    }
}

/// The SI unit a literal of a base dimension is shown in: road distances in kilometres (a
/// distance has no metre), the rest in their SI unit.
const fn si_unit(base: crate::tariff::units::Base) -> &'static str {
    match base {
        crate::tariff::units::Base::Distance => "km",
        other => si(other),
    }
}

/// A converted value as hover writes it: two decimals, more for a value below one.
fn shown(value: Decimal) -> String {
    let places = if value.abs() >= Decimal::ONE { 2 } else { 6 };
    value
        .round_dp_with_strategy(places, RoundingStrategy::MidpointAwayFromZero)
        .normalize()
        .to_string()
}
