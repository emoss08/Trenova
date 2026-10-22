//! `tariff.rename`: renaming a name everywhere this file uses it, or saying why it can't be.
//!
//! What renames, and what it changes:
//!
//! - a name the file declares (input, constant, function, parameter, type, enum, `let`, `var`,
//!   loop variable, lambda parameter, pattern or `if let` binding): its declaration and every
//!   reference the checker recorded (interpolations, `given input.x` and imports included), and
//!   for a parameter the call sites that name it (`deficit(actual: …)`);
//! - a field of a record type the file declares: the declaration, every `.field` read from a
//!   value of that type, every record literal of that type and every record pattern on it;
//! - a variant of an enum the file declares: the declaration and every use, bare, qualified or
//!   in a pattern;
//! - a module alias (`use std.fuel`, read as `fuel`): every use and the `use` itself, which
//!   gains (or changes) its `as`.
//!
//! What is refused, with a `Message`: the schema's roots, fields and enums; keywords; names an
//! imported library declares (an import can't rename what it imports: the library declares the
//! name, so the rename belongs in the library); tests (named by their text); a new name that is
//! not an identifier of the right case for what it names; and a new name that collides with one
//! already visible where the renamed name is declared or used, or that would make a bare variant
//! ambiguous. Without `newName`, the operation answers `prepareRename`: the name's range and
//! text, or why it can't be renamed.

use super::cursor::{Extent, Scopes, token_at};
use super::session::Session;
use crate::message;
use crate::message::{Arg, Message};
use crate::span::Span;
use crate::tariff::check::describe::{quote, snippet};
use crate::tariff::check::lower::BUILTIN_TYPES;
use crate::tariff::check::{Callee, SymbolId, SymbolKind};
use crate::tariff::syntax::ast::{ExprKind, ItemKind, Pattern, PatternKind};
use crate::tariff::syntax::lexer::{classify, is_keyword};
use crate::tariff::syntax::program::{Decl, ModuleId};
use crate::tariff::syntax::token::TokenKind;
use crate::tariff::types::Type;
use std::collections::BTreeSet;

/// What the cursor is on, when it can be renamed.
#[derive(Debug, Clone, PartialEq, Eq)]
enum Subject {
    /// A symbol the root file declares.
    Symbol(SymbolId),
    /// A field of a record type the root file declares: the type's qualified name and the
    /// field.
    Field(String, String),
    /// A variant of an enum the root file declares.
    Variant(String, String),
}

/// A name that can be renamed: where it is and its text.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Prepared {
    /// The name at the cursor.
    pub range: Span,
    /// Its text, to start the new name from.
    pub placeholder: String,
}

/// One replacement.
#[derive(Debug, Clone, PartialEq, Eq, PartialOrd, Ord)]
pub struct Edit {
    /// The text replaced.
    pub span: Span,
    /// What it is replaced with.
    pub insert: String,
}

/// Whether the name at byte `offset` can be renamed.
///
/// # Errors
///
/// Why it can't be, as a message for the author.
pub fn prepare(session: &Session<'_>, offset: u32) -> Result<Prepared, Message> {
    let (span, _) = subject(session, offset)?;
    Ok(Prepared {
        range: span,
        placeholder: snippet(session.source(), span).to_owned(),
    })
}

/// The edits that rename the name at byte `offset` to `new_name`.
///
/// # Errors
///
/// Why it can't be renamed, or why `new_name` can't be its name.
pub fn rename(session: &Session<'_>, offset: u32, new_name: &str) -> Result<Vec<Edit>, Message> {
    let (span, subject) = subject(session, offset)?;
    let old = snippet(session.source(), span).to_owned();
    if old == new_name {
        return Ok(Vec::new());
    }
    let scopes = Scopes::of(session.source(), session.file());
    session.tick(scopes.exprs.len());
    let spans = match &subject {
        Subject::Symbol(id) => {
            symbol_name_ok(session, *id, new_name)?;
            symbol_free(session, &scopes, *id, new_name)?;
            symbol_spans(session, &scopes, *id)
        }
        Subject::Field(record, field) => {
            shaped(new_name, TokenKind::Name, Message::new(message!("a field")))?;
            field_free(session, record, new_name)?;
            field_spans(session, &scopes, record, field)
        }
        Subject::Variant(enumeration, variant) => {
            shaped(
                new_name,
                TokenKind::TypeName,
                Message::new(message!("a variant")),
            )?;
            variant_free(session, enumeration, new_name)?;
            variant_spans(session, &scopes, enumeration, variant)
        }
    };
    let mut edits: Vec<Edit> = spans
        .into_iter()
        .map(|span| Edit {
            span,
            insert: new_name.to_owned(),
        })
        .collect();
    if let Subject::Symbol(id) = subject
        && let Some(alias) = alias_edit(session, id, new_name)
    {
        let declared = session.symbol(id).and_then(|symbol| symbol.span);
        edits.retain(|edit| Some(edit.span) != declared);
        edits.push(alias);
    }
    edits.sort();
    edits.dedup();
    Ok(edits)
}

/// What the cursor is on, and the name's span.
fn subject(session: &Session<'_>, offset: u32) -> Result<(Span, Subject), Message> {
    let nothing = || Message::new(message!("There is no name here to rename"));
    let file = session.file();
    let index = token_at(&file.tokens, offset).ok_or_else(nothing)?;
    let token = file.tokens.get(index).ok_or_else(nothing)?;
    let word = snippet(session.source(), token.span);
    if token.kind == TokenKind::Keyword && is_keyword(word) {
        return Err(Message::with(
            message!("{0} is a keyword, so it can't be renamed"),
            [quote(word)],
        ));
    }
    if !super::cursor::is_word(token.kind) || token.kind == TokenKind::Unit {
        return Err(nothing());
    }
    if let Some((span, id)) = session.symbol_at(offset) {
        return symbol_subject(session, span, id);
    }
    let scopes = Scopes::of(session.source(), file);
    if let Some(found) = field_subject(session, &scopes, token.span)
        .or_else(|| variant_subject(session, &scopes, token.span))
    {
        return found;
    }
    if session.schema.root(word).is_some() || word == "tables" {
        return Err(root_refusal(word));
    }
    Err(nothing())
}

fn root_refusal(word: &str) -> Message {
    Message::with(
        message!("{0} is part of the record a tariff rates, so it can't be renamed"),
        [quote(word)],
    )
}

fn symbol_subject(
    session: &Session<'_>,
    span: Span,
    id: SymbolId,
) -> Result<(Span, Subject), Message> {
    let symbol = session
        .symbol(id)
        .ok_or_else(|| Message::new(message!("There is no name here to rename")))?;
    if symbol.kind == SymbolKind::Root {
        return Err(root_refusal(&symbol.name));
    }
    if symbol.kind == SymbolKind::Test {
        return Err(Message::new(message!(
            "A test is named by its text: edit the text between the quotes"
        )));
    }
    let module = symbol.module.unwrap_or(ModuleId::ROOT);
    if module != ModuleId::ROOT {
        let path = session.module_path(module).unwrap_or_default();
        let message = if path.starts_with("std.") {
            Message::with(
                message!("{0} is declared in the standard library ({1}), so it can't be renamed"),
                [quote(&symbol.name), quote(&path)],
            )
        } else {
            Message::with(
                message!(
                    "{0} is declared in the library {1}; rename it there, then update this file's import"
                ),
                [quote(&symbol.name), quote(&path)],
            )
        };
        return Err(message);
    }
    Ok((span, Subject::Symbol(id)))
}

/// A field of a record type the root declares, at its declaration or a use.
fn field_subject(
    session: &Session<'_>,
    scopes: &Scopes<'_>,
    span: Span,
) -> Option<Result<(Span, Subject), Message>> {
    for item in &session.file().items {
        if let ItemKind::Type(decl) = &item.kind
            && let Some(field) = decl.fields.iter().find(|field| field.name.span == span)
        {
            return Some(Ok((
                span,
                Subject::Field(decl.name.text.clone(), field.name.text.clone()),
            )));
        }
    }
    let types = |id| session.analysis.type_of(ModuleId::ROOT, id).cloned();
    if let Some(field) = scopes
        .find_last(|expr| matches!(&expr.kind, ExprKind::Field { name, .. } if name.span == span))
        && let ExprKind::Field { target, name, .. } = &field.kind
        && let Some(Type::Record(record)) = types(target.id).as_ref().map(Type::unwrapped)
        && record.name.is_none()
    {
        return Some(Err(Message::with(
            message!("{0} is a field of the record a tariff rates, so it can't be renamed"),
            [quote(&name.text)],
        )));
    }
    let record = scopes.exprs.iter().find_map(|expr| match &expr.kind {
        ExprKind::Field { target, name, .. } if name.span == span => {
            record_name(types(target.id).as_ref()).map(|record| (record, name.text.clone()))
        }
        ExprKind::Record(fields) => {
            fields
                .iter()
                .find(|init| init.name.span == span)
                .and_then(|init| {
                    record_name(types(expr.id).as_ref())
                        .map(|record| (record, init.name.text.clone()))
                })
        }
        _ => None,
    });
    let (record, field) = record?;
    if record.contains('.') {
        return Some(Err(Message::with(
            message!("{0} is a field of {1}, which a library declares; rename it there"),
            [quote(&field), quote(&record)],
        )));
    }
    if session.schema.record(&record).is_some() || !session.root().symbols.contains_key(&record) {
        return Some(Err(Message::with(
            message!("{0} is a field of the record a tariff rates, so it can't be renamed"),
            [quote(&field)],
        )));
    }
    Some(Ok((span, Subject::Field(record, field))))
}

/// The declared name of a record type, through an optional.
fn record_name(ty: Option<&Type>) -> Option<String> {
    match ty?.unwrapped() {
        Type::Record(record) => record.name.clone(),
        _ => None,
    }
}

/// A variant of an enum the root declares, at its declaration or a use.
fn variant_subject(
    session: &Session<'_>,
    scopes: &Scopes<'_>,
    span: Span,
) -> Option<Result<(Span, Subject), Message>> {
    for item in &session.file().items {
        if let ItemKind::Enum(decl) = &item.kind
            && let Some(variant) = decl
                .variants
                .iter()
                .find(|variant| variant.name.span == span)
        {
            return Some(Ok((
                span,
                Subject::Variant(decl.name.text.clone(), variant.name.text.clone()),
            )));
        }
    }
    let analysis = session.analysis.module(ModuleId::ROOT)?;
    let found = scopes.exprs.iter().find_map(|expr| match &expr.kind {
        ExprKind::Path { head, member } => {
            let written = member.as_ref().unwrap_or(head);
            (written.span == span)
                .then(|| {
                    analysis
                        .variants
                        .get(&expr.id)
                        .map(|e| (e.name.clone(), written.text.clone()))
                })
                .flatten()
        }
        _ => None,
    });
    let found = found.or_else(|| {
        let located = super::cursor::locate(session.source(), session.file(), span.start);
        let pattern = located.pattern()?;
        match &pattern.kind {
            PatternKind::Variant { variant, .. } if variant.span == span => analysis
                .pattern_variants
                .get(&pattern.span.start)
                .map(|e| (e.name.clone(), variant.text.clone())),
            _ => None,
        }
    });
    let (enumeration, variant) = found?;
    if enumeration.contains('.') || !session.root().symbols.contains_key(&enumeration) {
        return Some(Err(Message::with(
            message!(
                "{0} is a variant of {1}, which this file doesn't declare, so it can't be renamed here"
            ),
            [quote(&variant), quote(&enumeration)],
        )));
    }
    Some(Ok((span, Subject::Variant(enumeration, variant))))
}

/// Whether `name` is an identifier of the class `kind`, explaining why not.
fn shaped(name: &str, kind: TokenKind, what: Message) -> Result<(), Message> {
    let identifier = name.chars().next().is_some_and(|c| c.is_ascii_alphabetic())
        && name.chars().all(|c| c.is_ascii_alphanumeric() || c == '_');
    if !identifier {
        return Err(Message::with(
            message!("{0} isn't a name: a name is a letter followed by letters, digits and `_`"),
            [quote(name)],
        ));
    }
    if is_keyword(name) {
        return Err(Message::with(
            message!("{0} is a keyword, so it can't be a name"),
            [quote(name)],
        ));
    }
    let found = classify(name);
    if found == kind || (kind == TokenKind::Code && found == TokenKind::Name) {
        return Ok(());
    }
    let template = match kind {
        TokenKind::TypeName => message!(
            "{0} can't name {1}: types, enums and variants are PascalCase, like `ExtraDrop`"
        ),
        TokenKind::Code => message!(
            "{0} can't name {1}: a constant is camelCase or UPPERCASE, like `dieselBase` or `DIESEL_BASE`"
        ),
        _ => message!("{0} can't name {1}: names are camelCase, like `extraDropFee`"),
    };
    Err(Message::with(template, [quote(name), Arg::from(what)]))
}

fn symbol_name_ok(session: &Session<'_>, id: SymbolId, name: &str) -> Result<(), Message> {
    let Some(symbol) = session.symbol(id) else {
        return Ok(());
    };
    let (kind, what) = match symbol.kind {
        SymbolKind::Type => (TokenKind::TypeName, message!("a type")),
        SymbolKind::Enum => (TokenKind::TypeName, message!("an enum")),
        SymbolKind::Const => (TokenKind::Code, message!("a constant")),
        SymbolKind::Function => (TokenKind::Name, message!("a function")),
        SymbolKind::Module => (TokenKind::Name, message!("a module")),
        SymbolKind::Input => (TokenKind::Name, message!("an input")),
        _ => (TokenKind::Name, message!("a binding")),
    };
    shaped(name, kind, Message::new(what))
}

fn taken(name: &str, what: Message) -> Message {
    Message::with(
        message!("{0} is already {1} here; pick another name"),
        [quote(name), Arg::from(what)],
    )
}

/// Whether `name` is free for the symbol: nothing visible where it is declared or used goes by
/// it.
fn symbol_free(
    session: &Session<'_>,
    scopes: &Scopes<'_>,
    id: SymbolId,
    name: &str,
) -> Result<(), Message> {
    let Some(symbol) = session.symbol(id) else {
        return Ok(());
    };
    let types = matches!(symbol.kind, SymbolKind::Type | SymbolKind::Enum);
    if types {
        if BUILTIN_TYPES.iter().any(|(builtin, _)| *builtin == name)
            || name == "List"
            || session.schema.record(name).is_some()
            || session.schema.enumeration(name).is_some()
        {
            return Err(taken(name, Message::new(message!("a built-in type"))));
        }
    } else if session.schema.root(name).is_some() || name == "tables" {
        return Err(taken(name, Message::new(message!("a schema root"))));
    }
    if session.program.resolve(ModuleId::ROOT, name).is_some() {
        return Err(taken(name, Message::new(message!("declared in this file"))));
    }
    if types {
        return Ok(());
    }
    let global = !matches!(
        symbol.kind,
        SymbolKind::Let
            | SymbolKind::Var
            | SymbolKind::Param
            | SymbolKind::LoopVariable
            | SymbolKind::LambdaParam
            | SymbolKind::Binding
    );
    let own = symbol.span.and_then(|span| scopes.declared_at(span));
    let closed = closed_items(session);
    for other in scopes.locals.iter().filter(|local| local.name == name) {
        let clash = match own {
            _ if global => true,
            Some(own) => overlaps(own.extent, other.extent, &closed),
            None => true,
        };
        if clash {
            return Err(taken(name, Message::new(message!("a name in scope"))));
        }
    }
    Ok(())
}

/// The spans of the file's functions and tests, which see none of the file's top-level names.
fn closed_items(session: &Session<'_>) -> Vec<Span> {
    session
        .file()
        .items
        .iter()
        .filter(|item| matches!(item.kind, ItemKind::Fn(_) | ItemKind::Test(_)))
        .map(|item| item.span)
        .collect()
}

/// Whether two names' extents meet, so one would shadow or hide the other.
fn overlaps(first: Extent, second: Extent, closed: &[Span]) -> bool {
    let inside_closed = |span: Span| {
        closed
            .iter()
            .any(|item| item.start <= span.start && span.end <= item.end)
    };
    match (first, second) {
        (Extent::Span(a), Extent::Span(b)) => a.start <= b.end && b.start <= a.end,
        (Extent::TopLevel { .. }, Extent::TopLevel { .. }) => true,
        (Extent::TopLevel { from }, Extent::Span(span))
        | (Extent::Span(span), Extent::TopLevel { from }) => {
            span.end >= from && !inside_closed(span)
        }
    }
}

/// Every place a symbol's name is written in the root file.
fn symbol_spans(session: &Session<'_>, scopes: &Scopes<'_>, id: SymbolId) -> Vec<Span> {
    let mut spans: BTreeSet<Span> = session
        .references_to(id)
        .map(|reference| reference.span)
        .collect();
    let Some(symbol) = session.symbol(id) else {
        return Vec::new();
    };
    if let Some(span) = symbol.span {
        spans.insert(span);
    }
    if matches!(symbol.kind, SymbolKind::Type | SymbolKind::Enum) {
        spans.extend(
            scopes
                .types
                .iter()
                .filter(|name| name.text == symbol.name)
                .map(|name| name.span),
        );
    }
    if symbol.kind == SymbolKind::Param
        && let Some(function) = owning_function(session, symbol.span)
    {
        for expr in &scopes.exprs {
            let ExprKind::Call { args, .. } = &expr.kind else {
                continue;
            };
            let calls_it = matches!(
                session.analysis.module(ModuleId::ROOT).and_then(|analysis| analysis.callees.get(&expr.id)),
                Some(Callee::Function { module, name }) if *module == ModuleId::ROOT && *name == function
            );
            if !calls_it {
                continue;
            }
            for arg in args {
                if let Some(name) = &arg.name
                    && name.text == symbol.name
                {
                    spans.insert(name.span);
                }
            }
        }
    }
    spans.into_iter().collect()
}

/// The root function a parameter named at `span` belongs to.
fn owning_function(session: &Session<'_>, span: Option<Span>) -> Option<String> {
    let span = span?;
    session
        .file()
        .items
        .iter()
        .find_map(|item| match &item.kind {
            ItemKind::Fn(decl) if decl.params.iter().any(|param| param.name.span == span) => {
                Some(decl.name.text.clone())
            }
            _ => None,
        })
}

/// The `use` edit for a module alias: change its `as`, or add one after the path.
fn alias_edit(session: &Session<'_>, id: SymbolId, name: &str) -> Option<Edit> {
    let symbol = session.symbol(id)?;
    if symbol.kind != SymbolKind::Module {
        return None;
    }
    let span = symbol.span?;
    session.file().items.iter().find_map(|item| {
        let ItemKind::Use(using) = &item.kind else {
            return None;
        };
        if let Some(alias) = &using.alias
            && alias.span == span
        {
            return Some(Edit {
                span,
                insert: name.to_owned(),
            });
        }
        let last = using.path.last()?;
        (last.span == span && using.names.is_none()).then(|| Edit {
            span: Span::new(last.span.end, last.span.end),
            insert: format!(" as {name}"),
        })
    })
}

/// Whether the record type already has a field named `name`.
fn field_free(session: &Session<'_>, record: &str, name: &str) -> Result<(), Message> {
    let has = session
        .root()
        .symbols
        .get(record)
        .and_then(|symbol| match symbol.decl {
            Decl::Type(item) => session.file().items.get(item),
            _ => None,
        });
    if let Some(item) = has
        && let ItemKind::Type(decl) = &item.kind
        && decl.fields.iter().any(|field| field.name.text == name)
    {
        return Err(Message::with(
            message!("{0} already has a field {1}"),
            [quote(record), quote(name)],
        ));
    }
    Ok(())
}

/// Every place a field of a record type is written: its declaration, `.field` reads, record
/// literals and record patterns of that type.
fn field_spans(session: &Session<'_>, scopes: &Scopes<'_>, record: &str, field: &str) -> Vec<Span> {
    let mut spans = BTreeSet::new();
    for item in &session.file().items {
        if let ItemKind::Type(decl) = &item.kind
            && decl.name.text == record
        {
            spans.extend(
                decl.fields
                    .iter()
                    .filter(|f| f.name.text == field)
                    .map(|f| f.name.span),
            );
        }
    }
    let types = |id| session.analysis.type_of(ModuleId::ROOT, id).cloned();
    for expr in &scopes.exprs {
        match &expr.kind {
            ExprKind::Field { target, name, .. } if name.text == field => {
                if record_name(types(target.id).as_ref()).as_deref() == Some(record) {
                    spans.insert(name.span);
                }
            }
            ExprKind::Record(fields)
                if record_name(types(expr.id).as_ref()).as_deref() == Some(record) =>
            {
                spans.extend(
                    fields
                        .iter()
                        .filter(|init| init.name.text == field)
                        .map(|init| init.name.span),
                );
            }
            ExprKind::Match { scrutinee, arms } => {
                let ty = types(scrutinee.id);
                for arm in arms {
                    pattern_fields(&arm.pattern, ty.as_ref(), record, field, &mut spans);
                }
            }
            _ => {}
        }
    }
    match_statements(session, &mut |scrutinee, pattern| {
        let ty = types(scrutinee);
        pattern_fields(pattern, ty.as_ref(), record, field, &mut spans);
    });
    spans.into_iter().collect()
}

/// Calls `visit` with the scrutinee and each arm's pattern of every `match` statement.
fn match_statements(
    session: &Session<'_>,
    visit: &mut impl FnMut(crate::tariff::syntax::ast::NodeId, &Pattern),
) {
    fn stmt(
        statement: &crate::tariff::syntax::ast::Stmt,
        visit: &mut impl FnMut(crate::tariff::syntax::ast::NodeId, &Pattern),
    ) {
        use crate::tariff::syntax::ast::{ArmBody, StmtKind};
        match &statement.kind {
            StmtKind::Match(found) => {
                for arm in &found.arms {
                    visit(found.scrutinee.id, &arm.pattern);
                    if let ArmBody::Block(block) = &arm.body {
                        for inner in &block.stmts {
                            stmt(inner, visit);
                        }
                    }
                }
            }
            StmtKind::If(found) => {
                for block in found
                    .branches
                    .iter()
                    .map(|(_, block)| block)
                    .chain(found.otherwise.iter())
                {
                    for inner in &block.stmts {
                        stmt(inner, visit);
                    }
                }
            }
            StmtKind::For(found) => {
                for inner in &found.body.stmts {
                    stmt(inner, visit);
                }
            }
            StmtKind::Item(item) => {
                if let ItemKind::Fn(decl) = &item.kind
                    && let crate::tariff::syntax::ast::FnBody::Block(block) = &decl.body
                {
                    for inner in &block.stmts {
                        stmt(inner, visit);
                    }
                }
            }
            _ => {}
        }
    }
    for item in &session.file().items {
        match &item.kind {
            ItemKind::Stmt(statement) => stmt(statement, visit),
            ItemKind::Fn(decl) => {
                if let crate::tariff::syntax::ast::FnBody::Block(block) = &decl.body {
                    for inner in &block.stmts {
                        stmt(inner, visit);
                    }
                }
            }
            ItemKind::Test(test) => {
                for inner in &test.body.stmts {
                    stmt(inner, visit);
                }
            }
            _ => {}
        }
    }
}

/// The field names a record pattern on a value of type `ty` writes for `record.field`, nested
/// record patterns followed through their fields' types.
fn pattern_fields(
    pattern: &Pattern,
    ty: Option<&Type>,
    record: &str,
    field: &str,
    out: &mut BTreeSet<Span>,
) {
    match &pattern.kind {
        PatternKind::Record(fields) => {
            let shape = match ty.map(Type::unwrapped) {
                Some(Type::Record(shape)) => Some(shape.clone()),
                _ => None,
            };
            for (name, inner) in fields {
                if shape.as_ref().and_then(|shape| shape.name.as_deref()) == Some(record)
                    && name.text == field
                {
                    out.insert(name.span);
                }
                let inner_ty = shape
                    .as_ref()
                    .and_then(|shape| shape.field(&name.text))
                    .map(|f| f.ty.clone());
                pattern_fields(inner, inner_ty.as_ref(), record, field, out);
            }
        }
        PatternKind::Or(alternatives) => {
            for inner in alternatives {
                pattern_fields(inner, ty, record, field, out);
            }
        }
        _ => {}
    }
}

/// Whether `name` is free for a variant: not already a variant of the enum, nor of another
/// enum the file can name (which would make bare uses ambiguous).
fn variant_free(session: &Session<'_>, enumeration: &str, name: &str) -> Result<(), Message> {
    let own = session
        .global(ModuleId::ROOT, enumeration)
        .and_then(|id| session.symbol(id))
        .and_then(|symbol| symbol.ty.clone());
    if let Some(Type::Enum(own)) = &own
        && own.has(name)
    {
        return Err(Message::with(
            message!("{0} already has a variant {1}"),
            [quote(enumeration), quote(name)],
        ));
    }
    let others = session
        .schema
        .enums
        .iter()
        .map(|e| (e.name.clone(), e.has(name)));
    let declared =
        session.root().symbols.iter().filter(|(other, symbol)| {
            *other != enumeration && matches!(symbol.decl, Decl::Enum(_))
        });
    let mut clash: Vec<String> = others
        .filter(|(_, has)| *has)
        .map(|(name, _)| name)
        .collect();
    for (other, _) in declared {
        if let Some(Type::Enum(found)) = session
            .global(ModuleId::ROOT, other)
            .and_then(|id| session.symbol(id))
            .and_then(|symbol| symbol.ty.clone())
            && found.has(name)
        {
            clash.push(other.clone());
        }
    }
    if let Some(other) = clash.first() {
        return Err(Message::with(
            message!(
                "{0} is also a variant of {1}, so a bare {0} would be ambiguous; pick another name"
            ),
            [quote(name), quote(other)],
        ));
    }
    Ok(())
}

/// Every place a variant of an enum the root declares is written.
fn variant_spans(
    session: &Session<'_>,
    scopes: &Scopes<'_>,
    enumeration: &str,
    variant: &str,
) -> Vec<Span> {
    let mut spans = BTreeSet::new();
    for item in &session.file().items {
        if let ItemKind::Enum(decl) = &item.kind
            && decl.name.text == enumeration
        {
            spans.extend(
                decl.variants
                    .iter()
                    .filter(|v| v.name.text == variant)
                    .map(|v| v.name.span),
            );
        }
    }
    let Some(analysis) = session.analysis.module(ModuleId::ROOT) else {
        return spans.into_iter().collect();
    };
    for expr in &scopes.exprs {
        if let ExprKind::Path { head, member } = &expr.kind
            && analysis
                .variants
                .get(&expr.id)
                .is_some_and(|found| found.name == enumeration)
        {
            let written = member.as_ref().unwrap_or(head);
            if written.text == variant {
                spans.insert(written.span);
            }
        }
    }
    match_statements(session, &mut |_, pattern| {
        pattern_variants(pattern, analysis, enumeration, variant, &mut spans);
    });
    for expr in &scopes.exprs {
        if let ExprKind::Match { arms, .. } = &expr.kind {
            for arm in arms {
                pattern_variants(&arm.pattern, analysis, enumeration, variant, &mut spans);
            }
        }
    }
    spans.into_iter().collect()
}

fn pattern_variants(
    pattern: &Pattern,
    analysis: &crate::tariff::check::ModuleAnalysis,
    enumeration: &str,
    variant: &str,
    out: &mut BTreeSet<Span>,
) {
    match &pattern.kind {
        PatternKind::Variant {
            variant: written, ..
        } => {
            if written.text == variant
                && analysis
                    .pattern_variants
                    .get(&pattern.span.start)
                    .is_some_and(|found| found.name == enumeration)
            {
                out.insert(written.span);
            }
        }
        PatternKind::Record(fields) => {
            for (_, inner) in fields {
                pattern_variants(inner, analysis, enumeration, variant, out);
            }
        }
        PatternKind::Or(alternatives) => {
            for inner in alternatives {
                pattern_variants(inner, analysis, enumeration, variant, out);
            }
        }
        _ => {}
    }
}
