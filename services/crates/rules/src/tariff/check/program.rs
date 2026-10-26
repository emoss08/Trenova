//! Checking a whole program: every module in dependency order (declarations first, then
//! bodies), the call graph across modules, and the root file's rules (what each kind of file
//! may hold, the order of total clauses, unused inputs and imports).

use super::checker::{Body, Checker, Global, Globals, Want};
use super::decls::{Declarer, State};
use super::describe::quote;
use super::scope::LocalKind;
use super::stmt::later_names;
use super::{Analysis, ImportInfo, ModuleAnalysis, ModuleId, Program, Symbol, SymbolKind};
use crate::span::Span;
use crate::tariff::context::Context;
use crate::tariff::diagnostic::{Diagnostic, Severity};
use crate::tariff::docs::Doc;
use crate::tariff::schema::{BILL_DOC, Schema, TABLES_DOC, bill_type};
use crate::tariff::syntax::ast::{
    FileKind, FnBody, FnDecl, ItemKind, Stmt, StmtKind, TextPart, TotalClause,
};
use crate::tariff::syntax::program::Decl;
use crate::tariff::types::Type;
use std::collections::BTreeMap;
use trenova_module_kit::message;
use trenova_module_kit::message::Message;

/// Checks a program; see [`super`].
#[must_use]
pub fn check(program: &Program, context: &Context) -> Analysis {
    let schema = Schema::new(&context.settings);
    let mut state = State::default();
    for (root, ty, doc) in &schema.roots {
        state.symbols.push(Symbol {
            name: (*root).to_owned(),
            kind: SymbolKind::Root,
            module: None,
            span: None,
            ty: Some(ty.clone()),
            doc: Some(doc.clone()),
        });
    }
    state.symbols.push(Symbol {
        name: String::from("tables"),
        kind: SymbolKind::Root,
        module: None,
        span: None,
        ty: None,
        doc: Some(Doc::message(TABLES_DOC)),
    });
    state.symbols.push(Symbol {
        name: String::from("bill"),
        kind: SymbolKind::Root,
        module: None,
        span: None,
        ty: Some(Type::Record(bill_type())),
        doc: Some(Doc::message(BILL_DOC)),
    });
    let mut modules: Vec<ModuleAnalysis> = program
        .modules
        .iter()
        .map(|_| ModuleAnalysis::default())
        .collect();
    for module in program.in_dependency_order() {
        let index = usize::try_from(module.id.0).unwrap_or(usize::MAX);
        let Some(out) = modules.get_mut(index) else {
            continue;
        };
        let declarer = Declarer {
            program,
            module,
            schema: &schema,
            context,
        };
        declarer.declare(&mut state, out);
        bodies(&declarer, &mut state, out);
    }
    recursion(program, &state, &mut modules);
    for module in &mut modules {
        module
            .diagnostics
            .sort_by_key(|diagnostic| (diagnostic.span.start, diagnostic.span.end));
        module
            .diagnostics
            .dedup_by(|a, b| a.span == b.span && a.code == b.code && a.message == b.message);
    }
    let inputs = program
        .root()
        .map(|root| super::inputs::describe(root, &state, &modules))
        .unwrap_or_default();
    let imports = program
        .in_dependency_order()
        .filter(|module| module.id != ModuleId::ROOT)
        .filter_map(|module| {
            let path = module.path.clone()?;
            let version = module.version.clone().or_else(|| {
                path.starts_with("std.")
                    .then(|| env!("CARGO_PKG_VERSION").to_owned())
            });
            Some(ImportInfo { path, version })
        })
        .collect();
    let mut analysis = Analysis {
        modules,
        symbols: std::mem::take(&mut state.symbols),
        references: std::mem::take(&mut state.references),
        inputs,
        imports,
        ok: false,
    };
    analysis.ok = !analysis.has_errors()
        && !program
            .diagnostics
            .iter()
            .any(|(_, diagnostic)| diagnostic.severity == Severity::Error);
    analysis
}

/// Checks a function's body (and its parameters' defaults); returns the type it gives.
pub fn function_body(
    declarer: &Declarer<'_>,
    state: &mut State,
    globals: &Globals,
    out: &mut ModuleAnalysis,
    decl: &FnDecl,
    ret: Option<&Type>,
) -> Type {
    let name = decl.name.text.clone();
    let info = match globals.values.get(&name) {
        Some(Global::Fn(info)) => Some(info.clone()),
        _ => None,
    };
    for (param, info_param) in decl
        .params
        .iter()
        .zip(info.iter().flat_map(|info| info.params.iter()))
    {
        if let Some(default) = &param.default {
            let want = Want::Annotation {
                name: param.name.text.clone(),
            };
            let _checked =
                declarer.constant_value(state, globals, out, default, Some(&info_param.ty), &want);
        }
    }
    let body = Body::Function {
        name: name.clone(),
        ret: ret.cloned().unwrap_or(Type::Error),
    };
    let file_lets = file_lets(declarer.module);
    let (result, calls) = declarer.run(state, globals, out, body, |checker| {
        checker.file_lets = file_lets;
        checker.scopes.push_barrier();
        if let Some(info) = &info {
            for (param, info_param) in decl.params.iter().zip(&info.params) {
                checker.declare(&param.name, LocalKind::Param, info_param.ty.clone());
            }
        }
        let result = match &decl.body {
            FnBody::Expr(value) => match ret {
                Some(ret) => {
                    checker.expect(value, ret, &Want::Return { name: name.clone() });
                    ret.clone()
                }
                None => match checker.expr(value, None) {
                    Type::Never => Type::Error,
                    other => other,
                },
            },
            FnBody::Block(block) => {
                checker.block(block);
                if ret.is_some_and(|ret| !matches!(ret, Type::Error)) && !checker.block_exits(block) {
                    checker.report(Diagnostic::error(
                        "missing-return",
                        decl.name.span,
                        Message::with(
                            message!("{0} doesn't return on every path: end each path with `return` (or `fail`)"),
                            [quote(&name)],
                        ),
                    ));
                }
                ret.cloned().unwrap_or(Type::Error)
            }
        };
        unused_params(checker);
        result
    });
    state
        .calls
        .entry((declarer.module.id.0, name))
        .or_default()
        .extend(calls);
    result
}

/// The names a file's top level declares with `let` or `var`.
fn file_lets(module: &crate::tariff::syntax::program::Module) -> Vec<String> {
    module
        .file
        .items
        .iter()
        .filter_map(|item| match &item.kind {
            ItemKind::Stmt(Stmt {
                kind: StmtKind::Let(binding) | StmtKind::Var(binding),
                ..
            }) => Some(binding.name.text.clone()),
            _ => None,
        })
        .collect()
}

/// Parameters are not reported unused: a library function's signature is its interface.
fn unused_params(_: &mut Checker<'_>) {}

/// Checks the bodies of one module: its file-level statements, functions and tests, then the
/// rules of its file kind.
fn bodies(declarer: &Declarer<'_>, state: &mut State, out: &mut ModuleAnalysis) {
    let index = usize::try_from(declarer.module.id.0).unwrap_or(usize::MAX);
    let globals = std::mem::take(
        state
            .globals
            .get_mut(index)
            .unwrap_or(&mut Globals::default()),
    );
    let module = declarer.module;
    let kind = module
        .file
        .header
        .as_ref()
        .map_or(FileKind::Tariff, |header| header.kind);
    for (name, symbol) in &module.symbols {
        if let (Decl::Fn(item), false) = (
            symbol.decl,
            state.inferred.contains(&(module.id.0, name.clone())),
        ) && let Some(ItemKind::Fn(decl)) = module.file.items.get(item).map(|item| &item.kind)
        {
            let ret = match globals.values.get(name) {
                Some(Global::Fn(info)) => info.ret.clone(),
                _ => Type::Error,
            };
            function_body(declarer, state, &globals, out, decl, Some(&ret));
        }
    }
    let stmts: Vec<&Stmt> = module
        .file
        .items
        .iter()
        .filter_map(|item| match &item.kind {
            ItemKind::Stmt(stmt) => Some(stmt),
            _ => None,
        })
        .collect();
    let billed = billed(&stmts);
    let header = module
        .file
        .header
        .as_ref()
        .map_or(Span::default(), |header| header.span);
    let owned: Vec<Stmt> = stmts.iter().map(|stmt| (*stmt).clone()).collect();
    let file_lets: Vec<String> = later_names(&owned)
        .into_iter()
        .map(|(name, _)| name)
        .collect();
    let (used, _) = declarer.run(state, &globals, out, Body::Top, |checker| {
        checker.billed.clone_from(&billed);
        checker.file_lets.clone_from(&file_lets);
        checker.scopes.push_barrier();
        checker.scopes.push(later_names(&owned));
        if kind != FileKind::Library {
            top_level(checker, &stmts, kind, header);
        }
        checker.pop_scope();
        std::mem::take(&mut checker.used)
    });
    let mut used = used;
    for item in &module.file.items {
        let ItemKind::Test(test) = &item.kind else {
            continue;
        };
        let ((), _) = declarer.run(state, &globals, out, Body::Test, |checker| {
            checker.billed.clone_from(&billed);
            checker.file_lets.clone_from(&file_lets);
            checker.expects_refusal = test.body.stmts.iter().any(|stmt| {
                matches!(
                    stmt.kind,
                    StmtKind::Expect(crate::tariff::syntax::ast::Expect::Fails(_))
                )
            });
            checker.scopes.push_barrier();
            checker.block(&test.body);
            used.extend(std::mem::take(&mut checker.used));
        });
    }
    used.extend(super::property::properties(
        declarer, state, &globals, out, &billed, &file_lets,
    ));
    used.extend(names_read_by_declarations(module));
    unused_globals(declarer, &globals, &used, out);
    super::bounds::check(module, state, &globals, out);
    super::options::check(module, out);
    super::codes::check(module, declarer.context, out);
    super::tables::live(module, declarer.context, out);
    if let Some(slot) = state.globals.get_mut(index) {
        *slot = globals;
    }
}

/// Imported names and module aliases read by functions, constants and defaults (checked in
/// their own passes), so they are not reported unused.
fn names_read_by_declarations(module: &crate::tariff::syntax::program::Module) -> Vec<String> {
    let mut out = Vec::new();
    for item in &module.file.items {
        match &item.kind {
            ItemKind::Fn(decl) => {
                for param in &decl.params {
                    if let Some(default) = &param.default {
                        super::decls::names_in(default, &mut out);
                    }
                    type_words(&param.ty, &mut out);
                }
                if let Some(ret) = &decl.ret {
                    type_words(ret, &mut out);
                }
                match &decl.body {
                    FnBody::Expr(value) => super::decls::names_in(value, &mut out),
                    FnBody::Block(block) => super::visit::block(block, &mut |_| {}, &mut |expr| {
                        if let crate::tariff::syntax::ast::ExprKind::Name(name) = &expr.kind {
                            out.push(name.text.clone());
                        }
                    }),
                }
            }
            ItemKind::Const(decl) => {
                super::decls::names_in(&decl.value, &mut out);
                if let Some(ty) = &decl.ty {
                    type_words(ty, &mut out);
                }
            }
            ItemKind::Input(decl) => {
                if let Some(default) = &decl.default {
                    super::decls::names_in(default, &mut out);
                }
                type_words(&decl.ty, &mut out);
            }
            ItemKind::Type(decl) => {
                for field in &decl.fields {
                    type_words(&field.ty, &mut out);
                }
            }
            ItemKind::Stmt(stmt) => stmt_types(stmt, &mut out),
            _ => {}
        }
    }
    out
}

/// The type names written in annotations inside statements, so an imported type used only in
/// `let x: Break = …` counts as used.
fn stmt_types(stmt: &Stmt, out: &mut Vec<String>) {
    super::visit::stmt(
        stmt,
        &mut |inner| {
            if let StmtKind::Let(binding) | StmtKind::Var(binding) = &inner.kind
                && let Some(ty) = &binding.ty
            {
                type_words(ty, out);
            }
        },
        &mut |_| {},
    );
}

fn type_words(ty: &crate::tariff::syntax::ast::TypeExpr, out: &mut Vec<String>) {
    use crate::tariff::syntax::ast::TypeExprKind;
    match &ty.kind {
        TypeExprKind::Named { name, args } => {
            out.push(name.text.clone());
            for arg in args {
                type_words(arg, out);
            }
        }
        TypeExprKind::Record(fields) => {
            for field in fields {
                type_words(&field.ty, out);
            }
        }
        TypeExprKind::Optional(inner) => type_words(inner, out),
        _ => {}
    }
}

/// Reports inputs, imported names and module aliases nothing reads.
fn unused_globals(
    declarer: &Declarer<'_>,
    globals: &Globals,
    used: &std::collections::BTreeSet<String>,
    out: &mut ModuleAnalysis,
) {
    let module = declarer.module;
    for (name, symbol) in &module.symbols {
        if let Decl::Input(_) = symbol.decl
            && !used.contains(name)
        {
            out.diagnostics.push(Diagnostic::warning(
                "unused",
                symbol.span,
                Message::with(message!("The input {0} is never used"), [quote(name)]),
            ));
        }
    }
    for (name, import) in &module.imports {
        if !used.contains(name)
            && (globals.values.contains_key(name) || globals.types.contains_key(name))
        {
            out.diagnostics.push(Diagnostic::warning(
                "unused",
                import.span,
                Message::with(message!("{0} is imported and never used"), [quote(name)]),
            ));
        }
    }
    for (name, alias) in &module.aliases {
        if !used.contains(name) {
            out.diagnostics.push(Diagnostic::warning(
                "unused",
                alias.span,
                Message::with(
                    message!("The module {0} is imported and never used"),
                    [quote(name)],
                ),
            ));
        }
    }
}

/// The labels and codes a file's charges bill and the labels of its choices, for its tests'
/// `charge(…)` and `chosen(…)`.
fn billed(stmts: &[&Stmt]) -> super::checker::Billed {
    let mut out = super::checker::Billed::default();
    let mut valued = Vec::new();
    for stmt in stmts {
        super::visit::stmt(
            stmt,
            &mut |inner| {
                if let StmtKind::Choose(choose) = &inner.kind {
                    out.choices.push(choose.label.text.clone());
                }
                if let StmtKind::Charge(charge) = &inner.kind {
                    match charge.label.literal() {
                        Some(label) => out.labels.push(label),
                        None => out.any_label = true,
                    }
                    if let Some(code) = &charge.code {
                        out.codes.push(code.text.clone());
                    }
                }
            },
            &mut |expr| {
                if let crate::tariff::syntax::ast::ExprKind::Choose(choose) = &expr.kind {
                    valued.push(choose.label.text.clone());
                }
            },
        );
    }
    out.choices.extend(valued);
    out.sites = super::bill::sites(stmts);
    out
}

/// A file's top-level statements, with the rules of its kind: total clauses last and in order,
/// a tariff billing something, a condition ending in its `Bool`.
fn top_level(checker: &mut Checker<'_>, stmts: &[&Stmt], kind: FileKind, header: Span) {
    let last = stmts.len().saturating_sub(1);
    let mut totals: Vec<(u8, Span)> = Vec::new();
    for (index, stmt) in stmts.iter().enumerate() {
        if let StmtKind::Total(clause) = &stmt.kind {
            let rank = match clause {
                TotalClause::Minimum(bound) if bound.on_freight() => 0,
                TotalClause::Maximum(bound) if bound.on_freight() => 1,
                TotalClause::Minimum(_) => 2,
                TotalClause::Maximum(_) => 3,
                TotalClause::Round { .. } => 4,
            };
            total_order(checker, &totals, rank, stmt.span);
            totals.push((rank, stmt.span));
        } else if let Some((_, first)) = totals.first()
            && kind == FileKind::Tariff
        {
            checker.report(
                Diagnostic::error(
                    "total-order",
                    stmt.span,
                    Message::new(message!(
                        "Total clauses come last: move this above `minimum`, `maximum` and `round`"
                    )),
                )
                .with_related(*first, Message::new(message!("The first total clause"))),
            );
        }
        if kind == FileKind::Condition && index == last {
            condition_result(checker, stmt);
            continue;
        }
        checker.stmt(stmt);
    }
    if kind == FileKind::Condition && stmts.is_empty() {
        checker.report(Diagnostic::error(
            "condition-result",
            header,
            Message::new(message!(
                "A condition ends with the `Bool` expression that decides it"
            )),
        ));
    }
    if kind == FileKind::Tariff && checker.billed.labels.is_empty() && !checker.billed.any_label {
        checker.report(Diagnostic::warning(
            "no-charge",
            header,
            Message::new(message!(
                "This tariff bills no charge, so its bill is always empty"
            )),
        ));
    }
    minimum_over_maximum(checker, stmts);
}

fn total_order(checker: &mut Checker<'_>, seen: &[(u8, Span)], rank: u8, span: Span) {
    if let Some((_, earlier)) = seen.iter().find(|(other, _)| *other == rank) {
        checker.report(
            Diagnostic::error(
                "total-order",
                span,
                Message::new(message!("Each total clause can be written once")),
            )
            .with_related(
                *earlier,
                Message::new(message!("It is already written here")),
            ),
        );
    } else if let Some((_, later)) = seen.iter().find(|(other, _)| *other > rank) {
        checker.report(
            Diagnostic::error(
                "total-order",
                span,
                Message::new(message!(
                    "Total clauses go in the order `minimum … on freight`, `maximum … on freight`, `minimum`, `maximum`, `round`"
                )),
            )
            .with_related(
                *later,
                Message::new(message!("This one should come after it")),
            ),
        );
    }
}

/// When a minimum and a maximum fold, the minimum may not be above the maximum: the freight
/// clauses against each other, the whole bill's against each other, and a freight minimum
/// against the whole bill's maximum (accessorials only add to the total, so freight above the
/// bill's maximum can't be kept).
fn minimum_over_maximum(checker: &mut Checker<'_>, stmts: &[&Stmt]) {
    type Clause<'s> = Option<(&'s crate::tariff::syntax::ast::Expr, Span)>;
    let (mut freight_low, mut freight_high, mut low, mut high): (Clause, Clause, Clause, Clause) =
        (None, None, None, None);
    for stmt in stmts {
        match &stmt.kind {
            StmtKind::Total(TotalClause::Minimum(bound)) if bound.on_freight() => {
                freight_low = Some((&bound.amount, stmt.span));
            }
            StmtKind::Total(TotalClause::Maximum(bound)) if bound.on_freight() => {
                freight_high = Some((&bound.amount, stmt.span));
            }
            StmtKind::Total(TotalClause::Minimum(bound)) => low = Some((&bound.amount, stmt.span)),
            StmtKind::Total(TotalClause::Maximum(bound)) => {
                high = Some((&bound.amount, stmt.span));
            }
            _ => {}
        }
    }
    let pairs = [
        (
            freight_low,
            freight_high,
            message!(
                "The freight minimum is above the freight maximum, so no freight total satisfies both"
            ),
        ),
        (
            low,
            high,
            message!("The minimum is above the maximum, so no total satisfies both"),
        ),
        (
            freight_low,
            high,
            message!("The freight minimum is above the bill's maximum, so no bill satisfies both"),
        ),
    ];
    for (minimum, maximum, template) in pairs {
        let (Some((low, low_span)), Some((high, high_span))) = (minimum, maximum) else {
            continue;
        };
        if above(checker, low, high) {
            checker.report(
                Diagnostic::error("total-order", low_span, Message::new(template))
                    .with_related(high_span, Message::new(message!("The maximum"))),
            );
        }
    }
}

/// Whether `low` and `high` both fold and `low` is the larger.
pub(super) fn above(
    checker: &Checker<'_>,
    low: &crate::tariff::syntax::ast::Expr,
    high: &crate::tariff::syntax::ast::Expr,
) -> bool {
    let fold = |expr: &crate::tariff::syntax::ast::Expr| -> Option<super::consts::Value> {
        let none = |_: &crate::tariff::syntax::ast::Expr| None;
        let type_of = |id| checker.recorded(id);
        super::consts::Folder {
            constant: &none,
            type_of: &type_of,
        }
        .fold(expr)
        .ok()
    };
    let (Some(low_value), Some(high_value)) = (fold(low), fold(high)) else {
        return false;
    };
    super::consts::compare(&low_value, &high_value) == Some(std::cmp::Ordering::Greater)
}

/// A condition's last statement: the `Bool` it decides.
fn condition_result(checker: &mut Checker<'_>, stmt: &Stmt) {
    checker.statements.push(stmt.span);
    if let StmtKind::Expr(value) = &stmt.kind {
        checker.expect(value, &Type::Bool, &Want::Condition);
    } else {
        checker.stmt(stmt);
        checker.report(Diagnostic::error(
            "condition-result",
            stmt.span,
            Message::new(message!("A condition ends with the `Bool` expression that decides it, such as `shipment.hazmat and lane.interstate`")),
        ));
    }
    checker.statements.pop();
}

/// Reports every cycle in the call graph: functions that call themselves, directly or through
/// others, could run forever.
fn recursion(program: &Program, state: &State, modules: &mut [ModuleAnalysis]) {
    let mut color: BTreeMap<(u32, String), u8> = BTreeMap::new();
    let mut reported: Vec<Vec<(u32, String)>> = Vec::new();
    let nodes: Vec<(u32, String)> = state.calls.keys().cloned().collect();
    for node in nodes {
        let mut stack: Vec<(u32, String)> = Vec::new();
        visit(&node, state, &mut color, &mut stack, &mut reported, 0);
    }
    for cycle in reported {
        let Some((module, name)) = cycle.first() else {
            continue;
        };
        let Some(span) = program
            .decl(ModuleId(*module), name)
            .map(|symbol| symbol.span)
        else {
            continue;
        };
        let mut path: Vec<String> = cycle.iter().map(|(_, name)| name.clone()).collect();
        path.push(name.clone());
        if let Some(out) = modules.get_mut(usize::try_from(*module).unwrap_or(usize::MAX)) {
            out.diagnostics.push(Diagnostic::error(
                "recursion",
                span,
                Message::with(
                    message!("{0} calls itself through {1}: a function can't call itself, directly or through others, so every rating ends"),
                    [quote(name), quote(&path.join(" → "))],
                ),
            ));
        }
    }
}

fn visit(
    node: &(u32, String),
    state: &State,
    color: &mut BTreeMap<(u32, String), u8>,
    stack: &mut Vec<(u32, String)>,
    reported: &mut Vec<Vec<(u32, String)>>,
    depth: usize,
) {
    if depth > 256 {
        return;
    }
    match color.get(node) {
        Some(2) => return,
        Some(1) => {
            if let Some(start) = stack.iter().position(|entry| entry == node) {
                let cycle: Vec<(u32, String)> = stack.get(start..).unwrap_or_default().to_vec();
                let mut key = cycle.clone();
                key.sort();
                if !reported.iter().any(|known| {
                    let mut other = known.clone();
                    other.sort();
                    other == key
                }) {
                    reported.push(cycle);
                }
            }
            return;
        }
        _ => {}
    }
    color.insert(node.clone(), 1);
    stack.push(node.clone());
    if let Some(calls) = state.calls.get(node) {
        for (module, name, _) in calls {
            visit(
                &(module.0, name.clone()),
                state,
                color,
                stack,
                reported,
                depth + 1,
            );
        }
    }
    stack.pop();
    color.insert(node.clone(), 2);
}

/// The text of a charge label as written, `{…}` parts included.
#[must_use]
pub fn label_text(source: &str, label: &crate::tariff::syntax::ast::Text) -> String {
    if let Some(literal) = label.literal() {
        return literal;
    }
    let inner = Span::new(
        label.span.start.saturating_add(1),
        label.span.end.saturating_sub(1),
    );
    let _ = label
        .parts
        .iter()
        .any(|part| matches!(part, TextPart::Expr(_)));
    super::describe::snippet(source, inner).to_owned()
}
