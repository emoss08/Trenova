//! Where a name is written: the occurrence-finding `tariff.references` and `tariff.rename`
//! share.
//!
//! A [`Subject`] is what a name names: a checker symbol, a field of a declared record type, a
//! variant of an enum, or a local the checker has no symbol for (one in a part of the file it
//! did not reach, such as a property's `vary` name). [`occurrences`] finds every place it is
//! written, each classed as its declaration, a read or a write:
//!
//! - a symbol: its declaration (in the module that declares it), every reference the checker
//!   recorded in the root file, the type names in annotations for a type or enum, the call
//!   sites that name a parameter, the `input.name` targets of `given`, `vary` and a monotone
//!   `expect`, and the names the checker did not reach that resolve to it by scope;
//! - a field: its declaration, `.field` reads, record literals and record patterns of the type;
//! - a variant: its declaration, bare and qualified uses, and patterns;
//! - a local: its declaration and the names in its extent that resolve to it.
//!
//! A write is a `var` assignment's target, or the input a `given` or `vary` sets. The schema
//! has no source, so a schema field has no occurrences and a schema variant no declaration.

use super::cursor::{Scoped, Scopes, body_at, locate, statements};
use super::session::Session;
use trenova_module_kit::span::Span;
use crate::tariff::check::{Callee, ModuleAnalysis, SymbolId, SymbolKind};
use crate::tariff::syntax::ast::{
    Expect, ExprKind, File, ItemKind, Name, NodeId, Pattern, PatternKind, StmtKind,
};
use crate::tariff::syntax::program::ModuleId;
use crate::tariff::types::Type;
use std::collections::BTreeSet;

/// Where a name is written: its module and span.
type Place = (ModuleId, Span);

/// A subject's declaration, when it has one in a source, and every place it is written.
type Found = (Option<Place>, BTreeSet<Place>);

/// What a name names.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Subject {
    /// A symbol the checker declared.
    Symbol(SymbolId),
    /// A field of a declared record type: the type's qualified name and the field.
    Field(String, String),
    /// A variant: its enum's qualified name and the variant.
    Variant(String, String),
    /// A local name of the root file the checker has no symbol for, by its declaration.
    Local(Span),
}

/// How an occurrence uses its name.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub enum Use {
    /// Where the name is declared.
    Declaration,
    /// Where it is read.
    Read,
    /// Where it is set: a `var` assigned, an input a `given` or `vary` sets.
    Write,
}

impl Use {
    /// The wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Declaration => "declaration",
            Self::Read => "read",
            Self::Write => "write",
        }
    }
}

/// One place a name is written.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub struct Occurrence {
    /// The module it is in.
    pub module: ModuleId,
    /// The name as written.
    pub span: Span,
    /// How it is used.
    pub kind: Use,
}

/// Who declares a field written at a position.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FieldOwner {
    /// A record type the root file declares: its name and the field.
    Root(String, String),
    /// A record type a library declares: its qualified name and the field.
    Library(String, String),
    /// The record a tariff rates: the field.
    Schema(String),
}

/// Who declares a variant written at a position.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum VariantOwner {
    /// An enum the root file declares: its name and the variant.
    Root(String, String),
    /// An enum a library declares: its qualified name and the variant.
    Library(String, String),
    /// A schema enum: its name and the variant.
    Schema(String, String),
}

/// Every place `subject` is written, the root file's in source order after a declaration in
/// another module.
#[must_use]
pub fn occurrences(
    session: &Session<'_>,
    scopes: &Scopes<'_>,
    subject: &Subject,
) -> Vec<Occurrence> {
    let (declaration, spans) = match subject {
        Subject::Symbol(id) => (
            session
                .symbol(*id)
                .and_then(|symbol| Some((symbol.module?, symbol.span?))),
            symbol_spans(session, scopes, *id),
        ),
        Subject::Field(record, field) => field_spans(session, scopes, record, field),
        Subject::Variant(enumeration, variant) => {
            variant_spans(session, scopes, enumeration, variant)
        }
        Subject::Local(span) => (
            Some((ModuleId::ROOT, *span)),
            local_spans(session, scopes, *span)
                .into_iter()
                .map(|span| (ModuleId::ROOT, span))
                .collect(),
        ),
    };
    let writes = writes(session.file());
    session.tick(spans.len());
    let mut found: Vec<Occurrence> = spans
        .into_iter()
        .chain(declaration)
        .map(|(module, span)| Occurrence {
            module,
            span,
            kind: if Some((module, span)) == declaration {
                Use::Declaration
            } else if module == ModuleId::ROOT && writes.contains(&span) {
                Use::Write
            } else {
                Use::Read
            },
        })
        .collect();
    found.sort_by_key(|occurrence| {
        (
            occurrence.module == ModuleId::ROOT,
            occurrence.module,
            occurrence.span.start,
            occurrence.span.end,
        )
    });
    found.dedup_by_key(|occurrence| (occurrence.module, occurrence.span));
    found
}

/// The spans a statement sets: an assignment's target, and the input of a `given input.x` or
/// `vary input.x`.
fn writes(file: &File) -> BTreeSet<Span> {
    let mut out = BTreeSet::new();
    statements(file, &mut |stmt| match &stmt.kind {
        StmtKind::Assign { target, .. } => {
            out.insert(target.span);
        }
        StmtKind::Given { target, .. } | StmtKind::Vary { target, .. } => {
            if let [head, name] = target.as_slice()
                && head.text == "input"
            {
                out.insert(name.span);
            }
        }
        _ => {}
    });
    out
}

/// Every name the root file writes outside expressions: assignment targets, `given`, `vary`
/// and monotone `expect` targets, each with whether it is a target path's last part after
/// `input.`.
fn written_names(file: &File) -> Vec<(&Name, bool)> {
    let mut out = Vec::new();
    statements(file, &mut |stmt| {
        let target = match &stmt.kind {
            StmtKind::Assign { target, .. } => {
                out.push((target, false));
                return;
            }
            StmtKind::Given { target, .. }
            | StmtKind::Vary { target, .. }
            | StmtKind::Expect(Expect::Monotone { target, .. }) => target,
            _ => return,
        };
        match target.as_slice() {
            [head, name] if head.text == "input" => out.push((name, true)),
            [name] if !matches!(stmt.kind, StmtKind::Given { .. }) => out.push((name, false)),
            _ => {}
        }
    });
    out
}

/// The local a name written at `at` resolves to by scope: the innermost visible one, declared
/// before it, with that name.
#[must_use]
pub fn resolve_local<'s>(
    file: &File,
    scopes: &'s Scopes<'_>,
    name: &str,
    at: Span,
) -> Option<&'s Scoped> {
    let body = body_at(file, at.start);
    scopes
        .locals
        .iter()
        .filter(|local| local.name == name && local.span != at && local.visible(at.start, body))
        .max_by_key(|local| local.span.start)
}

/// The local at `span`: declared there, or a use there (an expression, an assignment target or
/// a monotone `expect` target) that resolves to it.
#[must_use]
pub fn local_at<'s>(
    session: &Session<'_>,
    scopes: &'s Scopes<'_>,
    span: Span,
) -> Option<&'s Scoped> {
    if let Some(local) = scopes.declared_at(span) {
        return Some(local);
    }
    let file = session.file();
    let name = scopes
        .find_last(|expr| matches!(&expr.kind, ExprKind::Name(name) if name.span == span))
        .and_then(|expr| match &expr.kind {
            ExprKind::Name(name) => Some(name.text.clone()),
            _ => None,
        })
        .or_else(|| {
            written_names(file)
                .into_iter()
                .find(|(name, after_input)| !after_input && name.span == span)
                .map(|(name, _)| name.text.clone())
        })?;
    resolve_local(file, scopes, &name, span)
}

/// The input an `input.name` target (of a `given`, a `vary` or a monotone `expect`) names
/// at `span`.
#[must_use]
pub fn input_target_at(session: &Session<'_>, span: Span) -> Option<SymbolId> {
    let (name, _) = written_names(session.file())
        .into_iter()
        .find(|(name, after_input)| *after_input && name.span == span)?;
    let id = session.global(ModuleId::ROOT, &name.text)?;
    (session.symbol(id)?.kind == SymbolKind::Input).then_some(id)
}

/// The subject of a local: its checker symbol when it has one, else the local itself.
#[must_use]
pub fn local_subject(session: &Session<'_>, local: &Scoped) -> Subject {
    session
        .declared_at(ModuleId::ROOT, local.span)
        .map_or(Subject::Local(local.span), Subject::Symbol)
}

/// The names written in the root file that resolve by scope to the local declared at `decl`,
/// where the checker recorded no other meaning.
fn local_spans(session: &Session<'_>, scopes: &Scopes<'_>, decl: Span) -> BTreeSet<Span> {
    let file = session.file();
    let Some(local) = scopes.declared_at(decl) else {
        return BTreeSet::new();
    };
    let own = session.declared_at(ModuleId::ROOT, decl);
    let names = scopes
        .exprs
        .iter()
        .filter_map(|expr| match &expr.kind {
            ExprKind::Name(name) => Some(name),
            _ => None,
        })
        .chain(
            written_names(file)
                .into_iter()
                .filter(|(_, after_input)| !after_input)
                .map(|(name, _)| name),
        );
    session.tick(scopes.exprs.len());
    let mut out = BTreeSet::new();
    for name in names {
        if name.text != local.name || !local.covers(name.span) {
            continue;
        }
        let checked = session.reference_at(name.span);
        if checked.is_some() && checked != own {
            continue;
        }
        if resolve_local(file, scopes, &name.text, name.span)
            .is_some_and(|found| found.span == decl)
        {
            out.insert(name.span);
        }
    }
    out
}

/// Every place a symbol's name is written, its declaration included.
fn symbol_spans(session: &Session<'_>, scopes: &Scopes<'_>, id: SymbolId) -> BTreeSet<Place> {
    let mut spans: BTreeSet<Place> = session
        .references_to(id)
        .map(|reference| (ModuleId::ROOT, reference.span))
        .collect();
    let Some(symbol) = session.symbol(id) else {
        return spans;
    };
    if let (Some(module), Some(span)) = (symbol.module, symbol.span) {
        spans.insert((module, span));
    }
    let mut root = BTreeSet::new();
    if matches!(symbol.kind, SymbolKind::Type | SymbolKind::Enum) {
        root.extend(
            scopes
                .types
                .iter()
                .filter(|name| name.text == symbol.name)
                .filter(|name| session.global(ModuleId::ROOT, &name.text) == Some(id))
                .map(|name| name.span),
        );
    }
    if symbol.kind == SymbolKind::Param {
        root.extend(named_arguments(session, scopes, id));
    }
    if symbol.module == Some(ModuleId::ROOT)
        && let Some(span) = symbol.span
        && scopes.declared_at(span).is_some()
    {
        root.extend(local_spans(session, scopes, span));
    } else {
        root.extend(global_uses(session, scopes, id));
    }
    spans.extend(root.into_iter().map(|span| (ModuleId::ROOT, span)));
    spans
}

/// The names of the root file that mean the global symbol `id` where the checker recorded no
/// reference: `input.name` targets, and plain names no local hides in a body that sees it.
fn global_uses(session: &Session<'_>, scopes: &Scopes<'_>, id: SymbolId) -> BTreeSet<Span> {
    let mut out = BTreeSet::new();
    let Some(symbol) = session.symbol(id) else {
        return out;
    };
    let file = session.file();
    if symbol.kind == SymbolKind::Input {
        for (name, after_input) in written_names(file) {
            if after_input && session.global(ModuleId::ROOT, &name.text) == Some(id) {
                out.insert(name.span);
            }
        }
    }
    let sees = |at: u32| {
        let body = body_at(file, at);
        match symbol.kind {
            SymbolKind::Input => body == super::cursor::Body::Top,
            SymbolKind::Const => body != super::cursor::Body::Declaration,
            SymbolKind::Function => !matches!(
                body,
                super::cursor::Body::Declaration | super::cursor::Body::Constant
            ),
            _ => false,
        }
    };
    session.tick(scopes.exprs.len());
    for expr in &scopes.exprs {
        let ExprKind::Name(name) = &expr.kind else {
            continue;
        };
        if name.text != symbol.name
            || !sees(name.span.start)
            || session.reference_at(name.span).is_some()
            || resolve_local(file, scopes, &name.text, name.span).is_some()
            || session.global(ModuleId::ROOT, &name.text) != Some(id)
        {
            continue;
        }
        out.insert(name.span);
    }
    out
}

/// The names of the arguments that name the parameter `id` at calls of its function.
fn named_arguments(session: &Session<'_>, scopes: &Scopes<'_>, id: SymbolId) -> BTreeSet<Span> {
    let mut out = BTreeSet::new();
    let Some(symbol) = session.symbol(id) else {
        return out;
    };
    let Some(function) = owning_function(session, symbol.span) else {
        return out;
    };
    let callees = session
        .analysis
        .module(ModuleId::ROOT)
        .map(|analysis| &analysis.callees);
    for expr in &scopes.exprs {
        let ExprKind::Call { args, .. } = &expr.kind else {
            continue;
        };
        let calls_it = matches!(
            callees.and_then(|callees| callees.get(&expr.id)),
            Some(Callee::Function { module, name }) if *module == ModuleId::ROOT && *name == function
        );
        if !calls_it {
            continue;
        }
        out.extend(
            args.iter()
                .filter_map(|arg| arg.name.as_ref())
                .filter(|name| name.text == symbol.name)
                .map(|name| name.span),
        );
    }
    out
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

/// The module a declared type's qualified name (`org.fees.Break`, `Break`) is in, and the
/// name there.
#[must_use]
pub fn split_qualified<'n>(
    session: &Session<'_>,
    qualified: &'n str,
) -> Option<(ModuleId, &'n str)> {
    match qualified.rsplit_once('.') {
        Some((path, name)) => Some((session.program.module_named(path)?.id, name)),
        None => Some((ModuleId::ROOT, qualified)),
    }
}

/// The declared name of a record type, through an optional.
#[must_use]
pub fn record_name(ty: Option<&Type>) -> Option<String> {
    match ty?.unwrapped() {
        Type::Record(record) => record.name.clone(),
        _ => None,
    }
}

/// The field written at `span` and who declares its record type: a field of a root `type`
/// declaration, a `.field` read or a record literal's field.
#[must_use]
pub fn field_at(session: &Session<'_>, scopes: &Scopes<'_>, span: Span) -> Option<FieldOwner> {
    for item in &session.file().items {
        if let ItemKind::Type(decl) = &item.kind
            && let Some(field) = decl.fields.iter().find(|field| field.name.span == span)
        {
            return Some(FieldOwner::Root(
                decl.name.text.clone(),
                field.name.text.clone(),
            ));
        }
    }
    let types = |id| session.analysis.type_of(ModuleId::ROOT, id).cloned();
    if let Some(field) = scopes
        .find_last(|expr| matches!(&expr.kind, ExprKind::Field { name, .. } if name.span == span))
        && let ExprKind::Field { target, name, .. } = &field.kind
        && let Some(Type::Record(record)) = types(target.id).as_ref().map(Type::unwrapped)
        && record.name.is_none()
    {
        return Some(FieldOwner::Schema(name.text.clone()));
    }
    let (record, field) = scopes.exprs.iter().find_map(|expr| match &expr.kind {
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
    })?;
    if record.contains('.') {
        return Some(FieldOwner::Library(record, field));
    }
    if session.schema.record(&record).is_some() || !session.root().symbols.contains_key(&record) {
        return Some(FieldOwner::Schema(field));
    }
    Some(FieldOwner::Root(record, field))
}

/// The variant written at `span` (a declaration, a bare or qualified use, or a pattern) and
/// who declares its enum.
#[must_use]
pub fn variant_at(session: &Session<'_>, scopes: &Scopes<'_>, span: Span) -> Option<VariantOwner> {
    for item in &session.file().items {
        if let ItemKind::Enum(decl) = &item.kind
            && let Some(variant) = decl
                .variants
                .iter()
                .find(|variant| variant.name.span == span)
        {
            return Some(VariantOwner::Root(
                decl.name.text.clone(),
                variant.name.text.clone(),
            ));
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
        let located = locate(session.source(), session.file(), span.start);
        session.tick(located.visited);
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
    Some(if enumeration.contains('.') {
        VariantOwner::Library(enumeration, variant)
    } else if session.root().symbols.contains_key(&enumeration)
        && session.schema.enumeration(&enumeration).is_none()
    {
        VariantOwner::Root(enumeration, variant)
    } else {
        VariantOwner::Schema(enumeration, variant)
    })
}

/// Calls `visit` with the scrutinee and each arm's pattern of every `match` in the root file,
/// statements and expressions alike.
fn match_patterns(
    session: &Session<'_>,
    scopes: &Scopes<'_>,
    visit: &mut impl FnMut(NodeId, &Pattern),
) {
    statements(session.file(), &mut |stmt| {
        if let StmtKind::Match(found) = &stmt.kind {
            for arm in &found.arms {
                visit(found.scrutinee.id, &arm.pattern);
            }
        }
    });
    for expr in &scopes.exprs {
        if let ExprKind::Match { scrutinee, arms } = &expr.kind {
            for arm in arms {
                visit(scrutinee.id, &arm.pattern);
            }
        }
    }
}

/// Where a declared record type's field is declared, in the module that declares the type.
fn field_declaration(session: &Session<'_>, record: &str, field: &str) -> Option<Place> {
    let (module, name) = split_qualified(session, record)?;
    let found = session.program.module(module)?;
    let symbol = found.symbols.get(name)?;
    let ItemKind::Type(decl) = &found.file.items.get(symbol.decl.item())?.kind else {
        return None;
    };
    decl.fields
        .iter()
        .find(|candidate| candidate.name.text == field)
        .map(|declared| (module, declared.name.span))
}

/// Every place a field of a record type is written: its declaration, `.field` reads, record
/// literals and record patterns of that type.
fn field_spans(session: &Session<'_>, scopes: &Scopes<'_>, record: &str, field: &str) -> Found {
    let declaration = field_declaration(session, record, field);
    let mut spans = BTreeSet::new();
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
            _ => {}
        }
    }
    match_patterns(session, scopes, &mut |scrutinee, pattern| {
        let ty = types(scrutinee);
        pattern_fields(pattern, ty.as_ref(), record, field, &mut spans);
    });
    (
        declaration,
        spans
            .into_iter()
            .map(|span| (ModuleId::ROOT, span))
            .collect(),
    )
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
        PatternKind::Wildcard
        | PatternKind::None
        | PatternKind::Literal(_)
        | PatternKind::Range { .. }
        | PatternKind::Variant { .. }
        | PatternKind::Binding(_)
        | PatternKind::Missing => {}
    }
}

/// Where a declared enum's variant is declared, in the module that declares the enum.
fn variant_declaration(session: &Session<'_>, enumeration: &str, variant: &str) -> Option<Place> {
    let (module, name) = split_qualified(session, enumeration)?;
    let found = session.program.module(module)?;
    let symbol = found.symbols.get(name)?;
    let ItemKind::Enum(decl) = &found.file.items.get(symbol.decl.item())?.kind else {
        return None;
    };
    decl.variants
        .iter()
        .find(|candidate| candidate.name.text == variant)
        .map(|declared| (module, declared.name.span))
}

/// Every place a variant is written: its declaration (for a declared enum) and every use,
/// bare, qualified or in a pattern.
fn variant_spans(
    session: &Session<'_>,
    scopes: &Scopes<'_>,
    enumeration: &str,
    variant: &str,
) -> Found {
    let declaration = if session.schema.enumeration(enumeration).is_some() {
        None
    } else {
        variant_declaration(session, enumeration, variant)
    };
    let mut spans = BTreeSet::new();
    let Some(analysis) = session.analysis.module(ModuleId::ROOT) else {
        return (declaration, BTreeSet::new());
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
    match_patterns(session, scopes, &mut |_, pattern| {
        pattern_variants(pattern, analysis, enumeration, variant, &mut spans);
    });
    (
        declaration,
        spans
            .into_iter()
            .map(|span| (ModuleId::ROOT, span))
            .collect(),
    )
}

fn pattern_variants(
    pattern: &Pattern,
    analysis: &ModuleAnalysis,
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
        PatternKind::Wildcard
        | PatternKind::None
        | PatternKind::Literal(_)
        | PatternKind::Range { .. }
        | PatternKind::Binding(_)
        | PatternKind::Missing => {}
    }
}
