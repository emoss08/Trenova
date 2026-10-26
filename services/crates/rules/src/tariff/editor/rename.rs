//! `tariff.rename`: renaming a name everywhere this file uses it, or saying why it can't be.
//!
//! What renames, and what it changes (found by [`super::occurrences`], which references share):
//!
//! - a name the file declares (input, constant, function, parameter, type, enum, `let`, `var`,
//!   loop variable, lambda parameter, pattern or `if let` binding, a property's `vary` name):
//!   its declaration and every use (interpolations, `given input.x` and `vary input.x`, monotone
//!   `expect` targets and imports included), and for a parameter the call sites that name it
//!   (`deficit(actual: …)`);
//! - a field of a record type the file declares: the declaration, every `.field` read from a
//!   value of that type, every record literal of that type and every record pattern on it;
//! - a variant of an enum the file declares: the declaration and every use, bare, qualified or
//!   in a pattern;
//! - a module alias (`use std.fuel`, read as `fuel`): every use and the `use` itself, which
//!   gains (or changes) its `as`.
//!
//! What is refused, with a `Message`: the schema's roots, fields and enums; keywords; names an
//! imported library declares (an import can't rename what it imports: the library declares the
//! name, so the rename belongs in the library); tests, properties and choices (named by their
//! text, `chosen("…")` included); a new name that is not an identifier of the right case for
//! what it names; and a new name that collides with one already visible where the renamed name
//! is declared or used, or that would make a bare variant ambiguous. Without `newName`, the
//! operation answers `prepareRename`: the name's range and text, or why it can't be renamed.

use super::cursor::{Extent, Scopes, token_at};
use super::occurrences::{
    FieldOwner, Subject, VariantOwner, field_at, input_target_at, local_at, local_subject,
    occurrences, variant_at,
};
use super::session::Session;
use crate::span::Span;
use crate::tariff::check::describe::{quote, snippet};
use crate::tariff::check::lower::BUILTIN_TYPES;
use crate::tariff::check::{SymbolId, SymbolKind};
use crate::tariff::syntax::ast::ItemKind;
use crate::tariff::syntax::lexer::{classify, is_keyword};
use crate::tariff::syntax::program::{Decl, ModuleId};
use crate::tariff::syntax::token::TokenKind;
use crate::tariff::types::Type;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

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
    let scopes = Scopes::of(session.source(), session.file());
    session.tick(scopes.exprs.len());
    let (span, _) = subject(session, &scopes, offset)?;
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
    let scopes = Scopes::of(session.source(), session.file());
    session.tick(scopes.exprs.len());
    let (span, subject) = subject(session, &scopes, offset)?;
    let old = snippet(session.source(), span).to_owned();
    if old == new_name {
        return Ok(Vec::new());
    }
    match &subject {
        Subject::Symbol(id) => {
            symbol_name_ok(session, *id, new_name)?;
            symbol_free(session, &scopes, *id, new_name)?;
        }
        Subject::Local(declared) => {
            shaped(
                new_name,
                TokenKind::Name,
                Message::new(message!("a binding")),
            )?;
            name_free(
                session,
                &scopes,
                Free {
                    declared: Some(*declared),
                    global: false,
                    types: false,
                },
                new_name,
            )?;
        }
        Subject::Field(record, _) => {
            shaped(new_name, TokenKind::Name, Message::new(message!("a field")))?;
            field_free(session, record, new_name)?;
        }
        Subject::Variant(enumeration, _) => {
            shaped(
                new_name,
                TokenKind::TypeName,
                Message::new(message!("a variant")),
            )?;
            variant_free(session, enumeration, new_name)?;
        }
    }
    let mut edits: Vec<Edit> = occurrences(session, &scopes, &subject)
        .into_iter()
        .filter(|occurrence| occurrence.module == ModuleId::ROOT)
        .map(|occurrence| Edit {
            span: occurrence.span,
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
fn subject(
    session: &Session<'_>,
    scopes: &Scopes<'_>,
    offset: u32,
) -> Result<(Span, Subject), Message> {
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
    if token.kind == TokenKind::Text
        && (super::choices::label_at(file, offset).is_some()
            || super::choices::chosen_at(session, scopes, offset).is_some())
    {
        return Err(Message::new(message!(
            "A choice is named by its label, which is text: edit it between the quotes, and in every `chosen` that names it"
        )));
    }
    if !super::cursor::is_word(token.kind) || token.kind == TokenKind::Unit {
        return Err(nothing());
    }
    if let Some((span, id)) = session.symbol_at(offset) {
        return symbol_subject(session, span, id);
    }
    if let Some(id) = input_target_at(session, token.span) {
        return symbol_subject(session, token.span, id);
    }
    if let Some(local) = local_at(session, scopes, token.span) {
        return match local_subject(session, local) {
            Subject::Symbol(id) => symbol_subject(session, token.span, id),
            other => Ok((token.span, other)),
        };
    }
    if let Some(field) = field_at(session, scopes, token.span) {
        return field_subject(token.span, field);
    }
    if let Some(variant) = variant_at(session, scopes, token.span) {
        return variant_subject(token.span, variant);
    }
    if session.schema.root(word).is_some() || word == "tables" || word == "bill" {
        return Err(root_refusal(word));
    }
    Err(nothing())
}

fn root_refusal(word: &str) -> Message {
    if word == "bill" {
        return Message::new(message!(
            "`bill` is the bill so far, so it can't be renamed"
        ));
    }
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

/// A field renames when the root declares its record type.
fn field_subject(span: Span, owner: FieldOwner) -> Result<(Span, Subject), Message> {
    match owner {
        FieldOwner::Root(record, field) => Ok((span, Subject::Field(record, field))),
        FieldOwner::Library(record, field) => Err(Message::with(
            message!("{0} is a field of {1}, which a library declares; rename it there"),
            [quote(&field), quote(&record)],
        )),
        FieldOwner::Schema(field) => Err(Message::with(
            message!("{0} is a field of the record a tariff rates, so it can't be renamed"),
            [quote(&field)],
        )),
    }
}

/// A variant renames when the root declares its enum.
fn variant_subject(span: Span, owner: VariantOwner) -> Result<(Span, Subject), Message> {
    match owner {
        VariantOwner::Root(enumeration, variant) => {
            Ok((span, Subject::Variant(enumeration, variant)))
        }
        VariantOwner::Library(enumeration, variant)
        | VariantOwner::Schema(enumeration, variant) => Err(Message::with(
            message!(
                "{0} is a variant of {1}, which this file doesn't declare, so it can't be renamed here"
            ),
            [quote(&variant), quote(&enumeration)],
        )),
    }
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

/// What a renamed name must not collide with.
#[derive(Debug, Clone, Copy)]
struct Free {
    /// Where the renamed name is declared in the root file, when it is.
    declared: Option<Span>,
    /// It is a file-level name, visible everywhere its body kinds see.
    global: bool,
    /// It names a type or an enum.
    types: bool,
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
    let free = Free {
        declared: symbol.span,
        global: !matches!(
            symbol.kind,
            SymbolKind::Let
                | SymbolKind::Var
                | SymbolKind::Param
                | SymbolKind::LoopVariable
                | SymbolKind::LambdaParam
                | SymbolKind::Binding
                | SymbolKind::Vary
        ),
        types: matches!(symbol.kind, SymbolKind::Type | SymbolKind::Enum),
    };
    name_free(session, scopes, free, name)
}

/// Whether `name` is free for a name declared as `free` describes.
fn name_free(
    session: &Session<'_>,
    scopes: &Scopes<'_>,
    free: Free,
    name: &str,
) -> Result<(), Message> {
    if free.types {
        if BUILTIN_TYPES.iter().any(|(builtin, _)| *builtin == name)
            || name == "List"
            || session.schema.record(name).is_some()
            || session.schema.enumeration(name).is_some()
        {
            return Err(taken(name, Message::new(message!("a built-in type"))));
        }
    } else if session.schema.root(name).is_some() || name == "tables" {
        return Err(taken(name, Message::new(message!("a schema root"))));
    } else if name == "bill" && session.kind() == crate::tariff::syntax::ast::FileKind::Tariff {
        return Err(taken(name, Message::new(message!("the bill so far"))));
    }
    if session.program.resolve(ModuleId::ROOT, name).is_some() {
        return Err(taken(name, Message::new(message!("declared in this file"))));
    }
    if free.types {
        return Ok(());
    }
    let own = free.declared.and_then(|span| scopes.declared_at(span));
    let closed = closed_items(session);
    for other in scopes.locals.iter().filter(|local| local.name == name) {
        let clash = match own {
            _ if free.global => true,
            Some(own) => overlaps(own.extent, other.extent, &closed),
            None => true,
        };
        if clash {
            return Err(taken(name, Message::new(message!("a name in scope"))));
        }
    }
    Ok(())
}

/// The spans of the file's functions, tests and properties, which see none of the file's top-level names.
fn closed_items(session: &Session<'_>) -> Vec<Span> {
    session
        .file()
        .items
        .iter()
        .filter(|item| {
            matches!(
                item.kind,
                ItemKind::Fn(_) | ItemKind::Test(_) | ItemKind::Property(_)
            )
        })
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
