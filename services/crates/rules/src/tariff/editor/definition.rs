//! `tariff.definition`: where the thing under the cursor is declared, in this file, an
//! organisation library or the standard library.
//!
//! A target names its module (`null` for this file, else `std.ltl` or `org.x`), the range of
//! the whole declaration and the range of its name (`selectionRange`), both in UTF-16 units of
//! that module's source. Names resolve through the checker's references (an imported name goes
//! to the library that declares it, a module alias to its `use`); fields of declared record
//! types, variants of declared enums, type names in annotations, named arguments and `use`
//! paths resolve through the program's declarations; a property's `vary` name goes to its
//! `vary`, and a test's `chosen("…")` to the `best` or `worst` with that label. The schema's
//! roots, fields and enums have no source, so they have no target.

use super::choices::{choice_named, chosen_at};
use super::cursor::{Node, Scopes, locate, token_at};
use super::occurrences::{input_target_at, local_at, split_qualified};
use super::session::Session;
use crate::tariff::check::describe::snippet;
use crate::tariff::check::{Callee, SymbolKind};
use crate::tariff::syntax::ast::{ExprKind, ItemKind, PatternKind, StmtKind};
use crate::tariff::syntax::program::ModuleId;
use crate::tariff::syntax::token::TokenKind;
use crate::tariff::types::Type;
use trenova_module_kit::span::Span;

/// One place a definition is.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Target {
    /// The module it is in.
    pub module: ModuleId,
    /// The whole declaration.
    pub range: Span,
    /// The declared name.
    pub selection: Span,
}

/// Where the thing at byte `offset` of the root file is declared.
#[must_use]
pub fn definition(session: &Session<'_>, offset: u32) -> Vec<Target> {
    let file = session.file();
    let Some(index) = token_at(&file.tokens, offset) else {
        return Vec::new();
    };
    let Some(token) = file.tokens.get(index) else {
        return Vec::new();
    };
    if token.kind == TokenKind::Text {
        let scopes = Scopes::of(session.source(), file);
        session.tick(scopes.exprs.len());
        return chosen_choice(session, &scopes, offset)
            .into_iter()
            .collect();
    }
    if !super::cursor::is_word(token.kind) || token.kind == TokenKind::Unit {
        return Vec::new();
    }
    let span = token.span;
    if let Some(target) = module_path(session, span) {
        return vec![target];
    }
    if let Some((_, id)) = session
        .symbol_at(offset)
        .or_else(|| input_target_at(session, span).map(|id| (span, id)))
    {
        let Some(symbol) = session.symbol(id) else {
            return Vec::new();
        };
        if symbol.kind == SymbolKind::Root {
            return Vec::new();
        }
        let (Some(module), Some(name)) = (symbol.module, symbol.span) else {
            return Vec::new();
        };
        return vec![declaration(session, module, name)];
    }
    let scopes = Scopes::of(session.source(), file);
    session.tick(scopes.exprs.len());
    if let Some(local) = local_at(session, &scopes, span) {
        let declared = session
            .declared_at(ModuleId::ROOT, local.span)
            .and_then(|id| session.symbol(id))
            .and_then(|symbol| Some((symbol.module?, symbol.span?)));
        let (module, name) = declared.unwrap_or((ModuleId::ROOT, local.span));
        return vec![declaration(session, module, name)];
    }
    member(session, &scopes, span)
        .or_else(|| variant(session, &scopes, span))
        .or_else(|| argument(session, &scopes, span))
        .or_else(|| written_type(session, span))
        .into_iter()
        .collect()
}

/// The target of a name declared at `name` in `module`: the declaration around it.
#[must_use]
pub fn declaration(session: &Session<'_>, module: ModuleId, name: Span) -> Target {
    let Some(found) = session.program.module(module) else {
        return Target {
            module,
            range: name,
            selection: name,
        };
    };
    let located = locate(&found.source, &found.file, name.start);
    session.tick(located.visited);
    let mut range = name;
    for node in located.chain.iter().rev() {
        let span = match node {
            Node::Param(param) if param.name.span == name => Some(param.span),
            Node::Stmt(stmt) if declares(stmt, name) => Some(stmt.span),
            Node::Item(item) if item.span.start <= name.start && name.end <= item.span.end => {
                Some(item.span)
            }
            _ => None,
        };
        if let Some(span) = span {
            range = span;
            break;
        }
    }
    Target {
        module,
        range,
        selection: name,
    }
}

/// Whether a statement declares the name at `name`.
fn declares(stmt: &crate::tariff::syntax::ast::Stmt, name: Span) -> bool {
    match &stmt.kind {
        StmtKind::Let(binding) | StmtKind::Var(binding) => binding.name.span == name,
        StmtKind::For(statement) => statement.binding.span == name,
        StmtKind::Require {
            condition: crate::tariff::syntax::ast::Condition::Let { name: bound, .. },
            ..
        } => bound.span == name,
        StmtKind::If(statement) => statement.branches.iter().any(|(condition, _)| {
            matches!(condition, crate::tariff::syntax::ast::Condition::Let { name: bound, .. } if bound.span == name)
        }),
        StmtKind::Vary { target, .. } => {
            super::cursor::vary_name(target).is_some_and(|bound| bound.span == name)
        }
        _ => false,
    }
}

/// The choice a test's `chosen("…")` names: the whole `best` or `worst`, its label selected.
fn chosen_choice(session: &Session<'_>, scopes: &Scopes<'_>, offset: u32) -> Option<Target> {
    let (_, label) = chosen_at(session, scopes, offset)?;
    let choose = choice_named(session.file(), &label)?;
    Some(Target {
        module: ModuleId::ROOT,
        range: choose.span,
        selection: choose.label.span,
    })
}

/// A field of a declared record type, read with `.`.
fn member(session: &Session<'_>, scopes: &Scopes<'_>, span: Span) -> Option<Target> {
    let field = scopes.find_last(
        |expr| matches!(&expr.kind, ExprKind::Field { name, .. } if name.span == span),
    )?;
    let ExprKind::Field { target, name, .. } = &field.kind else {
        return None;
    };
    let ty = session.analysis.type_of(ModuleId::ROOT, target.id)?;
    let Type::Record(record) = ty.unwrapped() else {
        return None;
    };
    field_target(session, record.name.as_deref()?, &name.text)
}

/// The declaration of field `field` of the record type `qualified`.
fn field_target(session: &Session<'_>, qualified: &str, field: &str) -> Option<Target> {
    let (module, type_name) = split_qualified(session, qualified)?;
    let found = session.program.module(module)?;
    let symbol = found.symbols.get(type_name)?;
    let item = found.file.items.get(symbol.decl.item())?;
    let ItemKind::Type(decl) = &item.kind else {
        return None;
    };
    let declared = decl
        .fields
        .iter()
        .find(|candidate| candidate.name.text == field)?;
    Some(Target {
        module,
        range: declared.name.span.cover(declared.ty.span),
        selection: declared.name.span,
    })
}

/// A variant of a declared enum, as a value or a pattern, or the enum of a qualified one.
fn variant(session: &Session<'_>, scopes: &Scopes<'_>, span: Span) -> Option<Target> {
    let analysis = session.analysis.module(ModuleId::ROOT)?;
    let (enumeration, written, on_enum) = if let Some(path) = scopes.find_last(|expr| {
        matches!(&expr.kind, ExprKind::Path { head, member }
            if head.span == span || member.as_ref().is_some_and(|m| m.span == span))
    }) {
        let ExprKind::Path { head, member } = &path.kind else {
            return None;
        };
        let on_enum = head.span == span && member.is_some();
        let variant = member.as_ref().unwrap_or(head).text.clone();
        (analysis.variants.get(&path.id)?.clone(), variant, on_enum)
    } else {
        let located = locate(session.source(), session.file(), span.start);
        let pattern = located.pattern()?;
        let PatternKind::Variant { variant, .. } = &pattern.kind else {
            return None;
        };
        if variant.span != span {
            return None;
        }
        (
            analysis.pattern_variants.get(&pattern.span.start)?.clone(),
            variant.text.clone(),
            false,
        )
    };
    let (module, enum_name) = split_qualified(session, &enumeration.name)?;
    let found = session.program.module(module)?;
    let symbol = found.symbols.get(enum_name)?;
    let item = found.file.items.get(symbol.decl.item())?;
    let ItemKind::Enum(decl) = &item.kind else {
        return None;
    };
    if on_enum {
        return Some(Target {
            module,
            range: item.span,
            selection: decl.name.span,
        });
    }
    let declared = decl
        .variants
        .iter()
        .find(|candidate| candidate.name.text == written)?;
    Some(Target {
        module,
        range: declared.name.span,
        selection: declared.name.span,
    })
}

/// A named argument: the parameter it names.
fn argument(session: &Session<'_>, scopes: &Scopes<'_>, span: Span) -> Option<Target> {
    let call = scopes.find_last(|expr| {
        matches!(&expr.kind, ExprKind::Call { args, .. }
            if args.iter().any(|arg| arg.name.as_ref().is_some_and(|name| name.span == span)))
    })?;
    let Callee::Function { module, name } = super::complete::callee_of(session, call)? else {
        return None;
    };
    let decl = super::describe::function_decl(session, module, &name)?;
    let written = snippet(session.source(), span);
    let param = decl
        .params
        .iter()
        .find(|param| param.name.text == written)?;
    Some(Target {
        module,
        range: param.span,
        selection: param.name.span,
    })
}

/// A type name written in an annotation.
fn written_type(session: &Session<'_>, span: Span) -> Option<Target> {
    let located = locate(session.source(), session.file(), span.start);
    located.ty()?;
    let word = snippet(session.source(), span);
    let id = session.global(ModuleId::ROOT, word)?;
    let symbol = session.symbol(id)?;
    Some(declaration(session, symbol.module?, symbol.span?))
}

/// A `use` path: the module's file.
fn module_path(session: &Session<'_>, span: Span) -> Option<Target> {
    let using = session
        .file()
        .items
        .iter()
        .find_map(|item| match &item.kind {
            ItemKind::Use(using) if using.path.iter().any(|part| part.span == span) => Some(using),
            _ => None,
        })?;
    let module = session.program.module_named(&using.dotted())?;
    let selection = module
        .file
        .header
        .as_ref()
        .map_or(Span::default(), |header| header.name.span);
    let end = u32::try_from(module.source.len()).unwrap_or(u32::MAX);
    Some(Target {
        module: module.id,
        range: Span::new(0, end),
        selection,
    })
}
