//! The declaration phase of one module: its enums and record types, function signatures,
//! constants (typed and folded) and inputs, and the names it imports, gathered into the
//! [`Globals`] its bodies are checked with and the [`Exports`] other modules import from.

use super::checker::{Billed, Body, Checker, Exports, FnInfo, Global, Globals, ParamInfo, Want};
use super::consts::{FoldError, Folder, Value};
use super::describe::quote;
use super::lower::{Lowerer, TypeScope};
use super::scope::Scopes;
use super::{HintKind, ModuleAnalysis, ModuleId, Reference, Symbol, SymbolId, SymbolKind};
use crate::message;
use crate::message::Message;
use crate::tariff::context::Context;
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::docs::Doc;
use crate::tariff::schema::Schema;
use crate::tariff::syntax::ast::{
    Const, EnumDecl, Expr, ExprKind, FnBody, FnDecl, Input, Item, ItemKind, TypeDecl, TypeExpr,
    TypeExprKind,
};
use crate::tariff::syntax::program::{Decl, Module, Program};
use crate::tariff::types::{EnumType, RecordType, Type};
use std::collections::{BTreeMap, BTreeSet};
use std::sync::Arc;

/// The calls one function makes: the callee's module and name, and where the call is.
pub type Calls = Vec<(ModuleId, String, crate::span::Span)>;

/// An input's `where` constraint with its parts folded to values.
#[derive(Debug, Clone, PartialEq)]
pub enum FoldedConstraint {
    /// A range: the minimum is inclusive, the maximum inclusive when `max_inclusive`.
    Range {
        /// The smallest value accepted.
        min: Option<Value>,
        /// The bound above.
        max: Option<Value>,
        /// The maximum is accepted too.
        max_inclusive: bool,
    },
    /// The values accepted.
    OneOf(Vec<Value>),
}

impl FoldedConstraint {
    /// The constraint in wire values, as `tariff.check` reports it.
    #[must_use]
    pub fn to_wire(&self) -> crate::tariff::constraint::Constraint {
        match self {
            Self::Range {
                min,
                max,
                max_inclusive,
            } => crate::tariff::constraint::Constraint::Range {
                min: min.as_ref().map(super::consts::wire),
                max: max.as_ref().map(super::consts::wire),
                max_inclusive: *max_inclusive,
            },
            Self::OneOf(values) => crate::tariff::constraint::Constraint::OneOf(
                values.iter().map(super::consts::wire).collect(),
            ),
        }
    }
}

/// The type a constraint's values have: an item's for a list input, the inner type for an
/// optional one, else the input's own.
#[must_use]
pub fn constraint_item_type(ty: &Type) -> Type {
    match ty {
        Type::List(item) | Type::Optional(item) => constraint_item_type(item),
        other => other.clone(),
    }
}

/// What the whole program has declared so far, shared by every module's check.
#[derive(Debug, Default)]
pub struct State {
    /// Per module: what it exports.
    pub exports: Vec<Exports>,
    /// Per module: its own record and enum types, by name, with their symbols.
    pub types: Vec<BTreeMap<String, (Type, Option<SymbolId>)>>,
    /// Per module: the names it can see.
    pub globals: Vec<Globals>,
    /// Folded constants by module and name.
    pub values: BTreeMap<(u32, String), Result<Value, FoldError>>,
    /// Folded input defaults of each module, by name.
    pub defaults: BTreeMap<(u32, String), Value>,
    /// Folded input `where` constraints of each module, by name: the range ends or the listed
    /// values, each checked against the input's type (an item's type, for a list input).
    pub constraints: BTreeMap<(u32, String), FoldedConstraint>,
    /// Functions whose bodies were checked while inferring their return type.
    pub inferred: BTreeSet<(u32, String)>,
    /// Every symbol.
    pub symbols: Vec<Symbol>,
    /// Every reference.
    pub references: Vec<Reference>,
    /// The call graph: each function's calls.
    pub calls: BTreeMap<(u32, String), Calls>,
}

impl TypeScope for Globals {
    fn named_type(&self, name: &str) -> Option<Type> {
        self.types.get(name).map(|(ty, _)| ty.clone())
    }

    fn type_names(&self) -> Vec<String> {
        self.types.keys().cloned().collect()
    }
}

/// A module's qualified name for its declarations: `std.ltl.Break`, or `Break` in the root.
fn qualified(module: &Module, name: &str) -> String {
    match &module.path {
        Some(path) => format!("{path}.{name}"),
        None => name.to_owned(),
    }
}

/// The names a type expression refers to.
fn type_names(ty: &TypeExpr, out: &mut Vec<String>) {
    match &ty.kind {
        TypeExprKind::Named { name, args } => {
            out.push(name.text.clone());
            for arg in args {
                type_names(arg, out);
            }
        }
        TypeExprKind::Record(fields) => {
            for field in fields {
                type_names(&field.ty, out);
            }
        }
        TypeExprKind::Optional(inner) => type_names(inner, out),
        TypeExprKind::Quotient { .. } | TypeExprKind::Missing => {}
    }
}

/// The context one module is declared in.
pub struct Declarer<'a> {
    /// The program.
    pub program: &'a Program,
    /// The module.
    pub module: &'a Module,
    /// The schema.
    pub schema: &'a Schema,
    /// The request's context.
    pub context: &'a Context,
}

impl Declarer<'_> {
    fn id(&self) -> ModuleId {
        self.module.id
    }

    fn index(&self) -> usize {
        usize::try_from(self.module.id.0).unwrap_or(usize::MAX)
    }

    fn item(&self, decl: Decl) -> Option<&Item> {
        self.module.file.items.get(decl.item())
    }

    /// A doc comment written in this module, as the hosts receive it.
    fn doc(&self, doc: Option<&crate::tariff::syntax::ast::Doc>) -> Option<Doc> {
        Doc::of(self.module.std, doc)
    }

    fn symbol(
        &self,
        state: &mut State,
        name: &str,
        kind: SymbolKind,
        span: crate::span::Span,
        ty: Option<Type>,
        doc: Option<Doc>,
    ) -> SymbolId {
        let id = SymbolId(u32::try_from(state.symbols.len()).unwrap_or(u32::MAX));
        state.symbols.push(Symbol {
            name: name.to_owned(),
            kind,
            module: Some(self.id()),
            span: Some(span),
            ty,
            doc,
        });
        id
    }

    /// Declares the module: types, imports, signatures, constants and inputs.
    pub fn declare(&self, state: &mut State, out: &mut ModuleAnalysis) {
        let index = self.index();
        while state.globals.len() <= index {
            state.globals.push(Globals::default());
            state.exports.push(Exports::default());
            state.types.push(BTreeMap::new());
        }
        let mut globals = Globals::default();
        for record in &self.schema.records {
            if let Some(name) = &record.name {
                globals
                    .types
                    .insert(name.clone(), (Type::Record(Arc::clone(record)), None));
            }
        }
        for enumeration in &self.schema.enums {
            globals.types.insert(
                enumeration.name.clone(),
                (Type::Enum(Arc::clone(enumeration)), None),
            );
            globals.enums.push(Arc::clone(enumeration));
        }
        self.imports(state, &mut globals);
        self.enums(state, &mut globals);
        self.records(state, &mut globals, out);
        self.signatures(state, &mut globals, out);
        self.constants(state, &mut globals, out);
        self.inputs(state, &mut globals, out);
        self.infer_returns(state, &mut globals, out);
        for (name, symbol) in &self.module.tests {
            let doc = self.doc(self.item(symbol.decl).and_then(|item| item.docs.as_ref()));
            self.symbol(state, name, SymbolKind::Test, symbol.span, None, doc);
        }
        if let Some(slot) = state.globals.get_mut(index) {
            *slot = globals;
        }
    }

    fn imports(&self, state: &mut State, globals: &mut Globals) {
        for (local, import) in &self.module.imports {
            let from = usize::try_from(import.module.0).unwrap_or(usize::MAX);
            if let Some(ty) = state
                .types
                .get(from)
                .and_then(|types| types.get(&import.name))
                .cloned()
            {
                if let Some(symbol) = ty.1 {
                    state.references.push(Reference {
                        module: self.id(),
                        span: import.span,
                        symbol,
                    });
                }
                if let Type::Enum(enumeration) = &ty.0 {
                    globals.enums.push(Arc::clone(enumeration));
                }
                globals.types.insert(local.clone(), ty);
                continue;
            }
            if let Some(value) = state
                .exports
                .get(from)
                .and_then(|exports| exports.values.get(&import.name))
                .cloned()
            {
                let symbol = match &value {
                    Global::Fn(info) => Some(info.symbol),
                    Global::Const { symbol, .. } => Some(*symbol),
                    _ => None,
                };
                if let Some(symbol) = symbol {
                    state.references.push(Reference {
                        module: self.id(),
                        span: import.span,
                        symbol,
                    });
                }
                globals.values.insert(local.clone(), value);
            }
        }
        for (alias, target) in &self.module.aliases {
            let doc = self
                .program
                .module(target.module)
                .and_then(|module| Doc::of(module.std, module.file.docs.as_ref()));
            let symbol = self.symbol(state, alias, SymbolKind::Module, target.span, None, doc);
            globals.values.insert(
                alias.clone(),
                Global::Module {
                    module: target.module,
                    symbol,
                },
            );
        }
    }

    fn enums(&self, state: &mut State, globals: &mut Globals) {
        for (name, symbol) in &self.module.symbols {
            let Decl::Enum(_) = symbol.decl else { continue };
            let Some(Item {
                kind: ItemKind::Enum(decl),
                docs,
                ..
            }) = self.item(symbol.decl)
            else {
                continue;
            };
            let enumeration = Arc::new(self.enum_type(decl, self.doc(docs.as_ref())));
            let ty = Type::Enum(Arc::clone(&enumeration));
            let id = self.symbol(
                state,
                name,
                SymbolKind::Enum,
                symbol.span,
                Some(ty.clone()),
                enumeration.doc.clone(),
            );
            globals.types.insert(name.clone(), (ty.clone(), Some(id)));
            globals.enums.push(enumeration);
            if let Some(types) = state.types.get_mut(self.index()) {
                types.insert(name.clone(), (ty, Some(id)));
            }
        }
    }

    fn enum_type(&self, decl: &EnumDecl, doc: Option<Doc>) -> EnumType {
        let mut variants: Vec<(String, Option<Doc>)> = Vec::new();
        for variant in &decl.variants {
            if !variants.iter().any(|(name, _)| *name == variant.name.text) {
                variants.push((variant.name.text.clone(), self.doc(variant.docs.as_ref())));
            }
        }
        EnumType {
            name: qualified(self.module, &decl.name.text),
            variants,
            doc,
        }
    }

    /// Lowers the module's record types, each once every type it names is known; the ones left
    /// refer to each other in a cycle.
    fn records(&self, state: &mut State, globals: &mut Globals, out: &mut ModuleAnalysis) {
        let mut pending: Vec<(&String, &TypeDecl, Option<Doc>, crate::span::Span)> = Vec::new();
        for (name, symbol) in &self.module.symbols {
            if let (
                Decl::Type(_),
                Some(Item {
                    kind: ItemKind::Type(decl),
                    docs,
                    ..
                }),
            ) = (symbol.decl, self.item(symbol.decl))
            {
                pending.push((name, decl, self.doc(docs.as_ref()), symbol.span));
            }
        }
        let local: BTreeSet<String> = pending.iter().map(|(name, ..)| (*name).clone()).collect();
        loop {
            let ready = pending.iter().position(|(_, decl, ..)| {
                let mut names = Vec::new();
                for field in &decl.fields {
                    type_names(&field.ty, &mut names);
                }
                names
                    .iter()
                    .all(|name| !local.contains(name) || globals.types.contains_key(name))
            });
            let Some(position) = ready else { break };
            let (name, decl, doc, span) = pending.remove(position);
            let mut diagnostics = Vec::new();
            let fields = Lowerer {
                scope: globals,
                std: self.module.std,
                diagnostics: &mut diagnostics,
            }
            .fields(&decl.fields);
            out.diagnostics.extend(diagnostics);
            let ty = Type::Record(Arc::new(RecordType {
                name: Some(qualified(self.module, name)),
                fields,
            }));
            let id = self.symbol(state, name, SymbolKind::Type, span, Some(ty.clone()), doc);
            globals.types.insert(name.clone(), (ty.clone(), Some(id)));
            if let Some(types) = state.types.get_mut(self.index()) {
                types.insert(name.clone(), (ty, Some(id)));
            }
        }
        for (name, _, doc, span) in pending {
            out.diagnostics.push(Diagnostic::error(
                "recursive-type",
                span,
                Message::with(
                    message!("{0} contains itself through its fields, so no value of it could ever be written"),
                    [quote(name)],
                ),
            ));
            let id = self.symbol(state, name, SymbolKind::Type, span, Some(Type::Error), doc);
            globals.types.insert(name.clone(), (Type::Error, Some(id)));
        }
    }

    fn lower(&self, globals: &Globals, ty: &TypeExpr, out: &mut ModuleAnalysis) -> Type {
        let mut diagnostics = Vec::new();
        let lowered = Lowerer {
            scope: globals,
            std: self.module.std,
            diagnostics: &mut diagnostics,
        }
        .lower(ty);
        out.diagnostics.extend(diagnostics);
        lowered
    }

    fn signatures(&self, state: &mut State, globals: &mut Globals, out: &mut ModuleAnalysis) {
        for (name, symbol) in &self.module.symbols {
            let Decl::Fn(_) = symbol.decl else { continue };
            let Some(Item {
                kind: ItemKind::Fn(decl),
                docs,
                private,
                ..
            }) = self.item(symbol.decl)
            else {
                continue;
            };
            let params: Vec<ParamInfo> = decl
                .params
                .iter()
                .map(|param| ParamInfo {
                    name: param.name.text.clone(),
                    ty: self.lower(globals, &param.ty, out),
                    default: param.default.is_some(),
                })
                .collect();
            let ret = if let Some(ret) = &decl.ret {
                self.lower(globals, ret, out)
            } else {
                if matches!(decl.body, FnBody::Block(_)) {
                    out.diagnostics.push(Diagnostic::error(
                        "return-type",
                        decl.name.span,
                        Message::with(
                            message!(
                                "{0} has a block body, so it must say what it returns: write {1}"
                            ),
                            [quote(name), quote(&format!("fn {name}(…) -> Type {{ … }}"))],
                        ),
                    ));
                }
                Type::Error
            };
            let doc = self.doc(docs.as_ref());
            let id = self.symbol(
                state,
                name,
                SymbolKind::Function,
                symbol.span,
                Some(ret.clone()),
                doc,
            );
            let info = Arc::new(FnInfo {
                module: self.id(),
                name: name.clone(),
                params,
                ret,
                symbol: id,
                private: private.is_some(),
            });
            globals
                .values
                .insert(name.clone(), Global::Fn(Arc::clone(&info)));
            if let Some(exports) = state.exports.get_mut(self.index()) {
                exports.values.insert(name.clone(), Global::Fn(info));
            }
        }
    }

    /// A checker for a part of this module that is not a body: a constant or a default.
    fn with_checker<R>(
        &self,
        state: &mut State,
        globals: &Globals,
        out: &mut ModuleAnalysis,
        body: Body,
        run: impl FnOnce(&mut Checker<'_>) -> R,
    ) -> R {
        self.run(state, globals, out, body, |checker| {
            checker.scopes.push_barrier();
            run(checker)
        })
        .0
    }

    /// A compile-time value checked and folded; its fold error is reported unless checking
    /// already reported why.
    pub fn constant_value(
        &self,
        state: &mut State,
        globals: &Globals,
        out: &mut ModuleAnalysis,
        value: &Expr,
        expected: Option<&Type>,
        want: &Want,
    ) -> (Type, Result<Value, FoldError>) {
        let before = out.diagnostics.len();
        let ty = self.with_checker(
            state,
            globals,
            out,
            Body::Constant,
            |checker| match expected {
                Some(expected) => {
                    checker.expect(value, expected, want);
                    expected.clone()
                }
                None => checker.expr(value, None),
            },
        );
        let module = self.id();
        let types = &out.types;
        let values = &state.values;
        let lookup = |expr: &Expr| -> Option<Result<Value, FoldError>> {
            let name = match &expr.kind {
                ExprKind::Name(name) => name.text.clone(),
                ExprKind::Field { target, name, .. } => {
                    let ExprKind::Name(alias) = &target.kind else {
                        return None;
                    };
                    let Some(Global::Module { module, .. }) = globals.values.get(&alias.text)
                    else {
                        return None;
                    };
                    return values
                        .get(&(module.0, name.text.clone()))
                        .cloned()
                        .map(|v| v.map_err(|_| FoldError::Upstream));
                }
                _ => return None,
            };
            match globals.values.get(&name)? {
                Global::Const {
                    module: owner,
                    name: declared,
                    ..
                } => values
                    .get(&(owner.0, declared.clone()))
                    .cloned()
                    .map(|v| v.map_err(|_| FoldError::Upstream))
                    .or(Some(Err(FoldError::Upstream))),
                _ => None,
            }
        };
        let type_of = |id: crate::tariff::syntax::ast::NodeId| {
            types
                .get(usize::try_from(id.0).unwrap_or(usize::MAX))
                .cloned()
                .flatten()
        };
        let folded = Folder {
            constant: &lookup,
            type_of: &type_of,
        }
        .fold(value);
        let _ = module;
        if out.diagnostics.len() == before
            && let Err(error) = &folded
            && let Some(diagnostic) = fold_diagnostic(*error)
        {
            out.diagnostics.push(diagnostic);
        }
        (ty, folded)
    }

    fn constants(&self, state: &mut State, globals: &mut Globals, out: &mut ModuleAnalysis) {
        let mut consts: Vec<(&String, &Const, Option<Doc>, bool, crate::span::Span)> = Vec::new();
        for (name, symbol) in &self.module.symbols {
            if let (
                Decl::Const(_),
                Some(Item {
                    kind: ItemKind::Const(decl),
                    docs,
                    private,
                    ..
                }),
            ) = (symbol.decl, self.item(symbol.decl))
            {
                consts.push((
                    name,
                    decl,
                    self.doc(docs.as_ref()),
                    private.is_some(),
                    symbol.span,
                ));
            }
        }
        let local: BTreeSet<String> = consts.iter().map(|(name, ..)| (*name).clone()).collect();
        let library = self
            .module
            .file
            .header
            .as_ref()
            .is_some_and(|h| h.kind == crate::tariff::syntax::ast::FileKind::Library);
        loop {
            let ready = consts.iter().position(|(_, decl, ..)| {
                let mut names = Vec::new();
                names_in(&decl.value, &mut names);
                names
                    .iter()
                    .all(|name| !local.contains(name) || globals.values.contains_key(name))
            });
            let Some(position) = ready else { break };
            let (name, decl, doc, private, span) = consts.remove(position);
            if library && decl.ty.is_none() {
                out.diagnostics.push(Diagnostic::error(
                    "const-type",
                    decl.name.span,
                    Message::with(
                        message!("A library's constant is part of its interface, so {0} needs a type: write {1}"),
                        [quote(name), quote(&format!("const {name}: Type = …"))],
                    ),
                ));
            }
            let declared = decl.ty.as_ref().map(|ty| self.lower(globals, ty, out));
            let want = Want::Annotation { name: name.clone() };
            let (ty, folded) =
                self.constant_value(state, globals, out, &decl.value, declared.as_ref(), &want);
            let ty = match ty {
                Type::Never => Type::Error,
                other => other,
            };
            state.values.insert((self.id().0, name.clone()), folded);
            let id = self.symbol(state, name, SymbolKind::Const, span, Some(ty.clone()), doc);
            let global = Global::Const {
                module: self.id(),
                name: name.clone(),
                ty,
                symbol: id,
                private,
            };
            globals.values.insert(name.clone(), global.clone());
            if let Some(exports) = state.exports.get_mut(self.index()) {
                exports.values.insert(name.clone(), global);
            }
        }
        for (name, decl, doc, private, span) in consts {
            out.diagnostics.push(Diagnostic::error(
                "recursion",
                decl.name.span,
                Message::with(message!("{0} is defined in terms of itself"), [quote(name)]),
            ));
            state
                .values
                .insert((self.id().0, name.clone()), Err(FoldError::Upstream));
            let id = self.symbol(state, name, SymbolKind::Const, span, Some(Type::Error), doc);
            globals.values.insert(
                name.clone(),
                Global::Const {
                    module: self.id(),
                    name: name.clone(),
                    ty: Type::Error,
                    symbol: id,
                    private,
                },
            );
        }
    }

    fn inputs(&self, state: &mut State, globals: &mut Globals, out: &mut ModuleAnalysis) {
        let mut inputs: Vec<(&String, &Input, Option<Doc>, crate::span::Span)> = Vec::new();
        for (name, symbol) in &self.module.symbols {
            if let (
                Decl::Input(_),
                Some(Item {
                    kind: ItemKind::Input(decl),
                    docs,
                    ..
                }),
            ) = (symbol.decl, self.item(symbol.decl))
            {
                inputs.push((name, decl, self.doc(docs.as_ref()), symbol.span));
            }
        }
        inputs.sort_by_key(|(_, _, _, span)| span.start);
        for (name, decl, doc, span) in inputs {
            let ty = self.lower(globals, &decl.ty, out);
            if let Some(default) = &decl.default {
                let want = Want::Annotation { name: name.clone() };
                let before = out.diagnostics.len();
                let (_, folded) =
                    self.constant_value(state, globals, out, default, Some(&ty), &want);
                let clean = out.diagnostics.len() == before && !matches!(ty, Type::Error);
                if let (Ok(value), true) = (folded, clean) {
                    state.defaults.insert((self.id().0, name.clone()), value);
                }
            }
            if let Some(constraint) = &decl.constraint
                && let Some(folded) =
                    self.fold_constraint(state, globals, out, name, &ty, constraint)
            {
                state
                    .constraints
                    .insert((self.id().0, name.clone()), folded);
            }
            let id = self.symbol(state, name, SymbolKind::Input, span, Some(ty.clone()), doc);
            globals
                .values
                .insert(name.clone(), Global::Input { ty, symbol: id });
        }
    }

    /// Folds an input's `where` constraint: a range's ends or a list's items, each a
    /// compile-time value of the input's type (its item type for a `List<T>`, its inner type for
    /// a `T?`). `None` when the constraint is neither a range nor a list, or a part does not fold
    /// (the reasons are reported as they are folded).
    fn fold_constraint(
        &self,
        state: &mut State,
        globals: &mut Globals,
        out: &mut ModuleAnalysis,
        name: &str,
        ty: &Type,
        constraint: &Expr,
    ) -> Option<FoldedConstraint> {
        let item = constraint_item_type(ty);
        let want = Want::Annotation {
            name: name.to_owned(),
        };
        let fold =
            |checker: &Self, state: &mut State, out: &mut ModuleAnalysis, expr: &Expr| {
                let before = out.diagnostics.len();
                let (_, folded) =
                    checker.constant_value(state, globals, out, expr, Some(&item), &want);
                match folded {
                    Ok(value) if out.diagnostics.len() == before => Some(value),
                    _ => None,
                }
            };
        match &constraint.kind {
            ExprKind::Range {
                start,
                end,
                inclusive,
            } => {
                let min = match start {
                    Some(start) => Some(fold(self, state, out, start)?),
                    None => None,
                };
                let max = match end {
                    Some(end) => Some(fold(self, state, out, end)?),
                    None => None,
                };
                Some(FoldedConstraint::Range {
                    min,
                    max,
                    max_inclusive: *inclusive,
                })
            }
            ExprKind::List(items) => {
                let mut values = Vec::with_capacity(items.len());
                for value in items {
                    values.push(fold(self, state, out, value)?);
                }
                Some(FoldedConstraint::OneOf(values))
            }
            _ => None,
        }
    }

    /// Infers the return type of each function written `fn f(…) = expr` with no `->`, by
    /// checking its body now, callees first.
    fn infer_returns(&self, state: &mut State, globals: &mut Globals, out: &mut ModuleAnalysis) {
        let mut pending: Vec<(&String, &FnDecl)> = Vec::new();
        for (name, symbol) in &self.module.symbols {
            if let (
                Decl::Fn(_),
                Some(Item {
                    kind: ItemKind::Fn(decl),
                    ..
                }),
            ) = (symbol.decl, self.item(symbol.decl))
                && decl.ret.is_none()
                && matches!(decl.body, FnBody::Expr(_))
            {
                pending.push((name, decl));
            }
        }
        let open: BTreeSet<String> = pending.iter().map(|(name, _)| (*name).clone()).collect();
        let mut done: BTreeSet<String> = BTreeSet::new();
        while !pending.is_empty() {
            let position = pending
                .iter()
                .position(|(_, decl)| {
                    let mut names = Vec::new();
                    if let FnBody::Expr(body) = &decl.body {
                        names_in(body, &mut names);
                    }
                    names
                        .iter()
                        .all(|name| !open.contains(name) || done.contains(name))
                })
                .unwrap_or(0);
            let (name, decl) = pending.remove(position);
            let ret = super::program::function_body(self, state, globals, out, decl, None);
            done.insert(name.clone());
            state.inferred.insert((self.id().0, name.clone()));
            out.hints.push(super::Hint {
                span: decl.name.span,
                ty: ret.clone(),
                kind: HintKind::Return,
            });
            if let Some(Global::Fn(info)) = globals.values.get(name).cloned() {
                let updated = Arc::new(FnInfo {
                    ret: ret.clone(),
                    ..(*info).clone()
                });
                if let Some(symbol) = state
                    .symbols
                    .get_mut(usize::try_from(info.symbol.0).unwrap_or(usize::MAX))
                {
                    symbol.ty = Some(ret);
                }
                globals
                    .values
                    .insert(name.clone(), Global::Fn(Arc::clone(&updated)));
                if let Some(exports) = state.exports.get_mut(self.index()) {
                    exports.values.insert(name.clone(), Global::Fn(updated));
                }
            }
        }
    }

    /// Runs a checker over a body of this module, returning what it found.
    pub fn run<R>(
        &self,
        state: &mut State,
        globals: &Globals,
        out: &mut ModuleAnalysis,
        body: Body,
        run: impl FnOnce(&mut Checker<'_>) -> R,
    ) -> (R, Calls) {
        let exports = std::mem::take(&mut state.exports);
        let mut symbols = std::mem::take(&mut state.symbols);
        let mut references = std::mem::take(&mut state.references);
        let mut checker = Checker {
            source: &self.module.source,
            module: self.id(),
            std: self.module.std,
            kind: self
                .module
                .file
                .header
                .as_ref()
                .map_or(crate::tariff::syntax::ast::FileKind::Tariff, |h| h.kind),
            globals,
            exports: &exports,
            schema: self.schema,
            context: self.context,
            symbols: &mut symbols,
            references: &mut references,
            out,
            scopes: Scopes::default(),
            body,
            loops: 0,
            statements: Vec::new(),
            calls: Vec::new(),
            file_lets: Vec::new(),
            billed: Billed::default(),
            lambda_depth: 0,
            used: BTreeSet::new(),
            exhaustive: BTreeSet::new(),
            expected_yet: false,
            early: BTreeSet::new(),
            expects_refusal: false,
        };
        let result = run(&mut checker);
        let calls = std::mem::take(&mut checker.calls);
        drop(checker);
        state.exports = exports;
        state.symbols = symbols;
        state.references = references;
        (result, calls)
    }
}

/// The finding for a value that does not fold.
fn fold_diagnostic(error: FoldError) -> Option<Diagnostic> {
    match error {
        FoldError::NotConstant(span) => Some(Diagnostic::error(
            "not-constant",
            span,
            Message::new(message!(
                "This isn't known before rating: a constant or default is computed from literals and other constants"
            )),
        )),
        FoldError::Overflow(span) => Some(Diagnostic::error(
            "overflow",
            span,
            Message::new(message!("This is past 28 significant digits")),
        )),
        FoldError::DivideByZero(span) => Some(Diagnostic::error(
            "divide-by-zero",
            span,
            Message::new(message!("This divides by zero")),
        )),
        FoldError::Upstream => None,
    }
}

/// The camelCase and UPPERCASE names an expression reads, for ordering declarations.
pub fn names_in(expr: &Expr, out: &mut Vec<String>) {
    super::visit::expr(expr, &mut |node| {
        if let ExprKind::Name(name) = &node.kind {
            out.push(name.text.clone());
        }
    });
}

impl Checker<'_> {
    /// Lowers a type written in a body (a `let` annotation).
    pub fn lower_type(&mut self, ty: &TypeExpr) -> Type {
        let mut diagnostics = Vec::new();
        let lowered = Lowerer {
            scope: self.globals,
            std: self.std,
            diagnostics: &mut diagnostics,
        }
        .lower(ty);
        self.out.diagnostics.extend(diagnostics);
        lowered
    }
}
