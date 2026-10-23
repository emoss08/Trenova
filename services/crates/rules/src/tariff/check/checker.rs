//! The state of one module's check: what the module can name, the scopes of the body being
//! checked, and the side tables the checker fills in.

use super::describe::{quote, snippet};
use super::scope::{Local, LocalKind, PathKey, Scopes};
use super::{
    Callee, Hint, HintKind, ModuleAnalysis, ModuleId, Reference, Symbol, SymbolId, SymbolKind,
};
use crate::message;
use crate::message::Message;
use crate::span::Span;
use crate::tariff::context::Context;
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::schema::Schema;
use crate::tariff::syntax::ast::{Expr, FileKind, Name, NodeId};
use crate::tariff::types::{EnumType, Type};
use std::collections::BTreeMap;
use std::sync::Arc;

/// A function's signature with what the checker needs at call sites.
#[derive(Debug, Clone)]
pub struct FnInfo {
    /// The module that declares it.
    pub module: ModuleId,
    /// Its name.
    pub name: String,
    /// Its parameters, in order.
    pub params: Vec<ParamInfo>,
    /// What it returns.
    pub ret: Type,
    /// Its symbol.
    pub symbol: SymbolId,
    /// Whether it is `private` to its library.
    pub private: bool,
}

/// One parameter of a function.
#[derive(Debug, Clone)]
pub struct ParamInfo {
    /// Its name.
    pub name: String,
    /// Its type.
    pub ty: Type,
    /// Whether it has a default.
    pub default: bool,
}

/// A name a module can see besides its locals.
#[derive(Debug, Clone)]
pub enum Global {
    /// An `input` of the file.
    Input {
        /// Its type.
        ty: Type,
        /// Its symbol.
        symbol: SymbolId,
    },
    /// A `const`, declared here or imported.
    Const {
        /// The declaring module.
        module: ModuleId,
        /// Its name there.
        name: String,
        /// Its type.
        ty: Type,
        /// Its symbol.
        symbol: SymbolId,
        /// Whether it is `private` to its library.
        private: bool,
    },
    /// A function, declared here or imported.
    Fn(Arc<FnInfo>),
    /// A module imported whole (`use std.fuel`), by the name it is read by.
    Module {
        /// The module.
        module: ModuleId,
        /// The alias's symbol.
        symbol: SymbolId,
    },
}

/// What one module can name: its own declarations, its imports and the schema's types.
#[derive(Debug, Default)]
pub struct Globals {
    /// Values by name: inputs, constants, functions and module aliases.
    pub values: BTreeMap<String, Global>,
    /// Record and enum types by the name they are written with.
    pub types: BTreeMap<String, (Type, Option<SymbolId>)>,
    /// Every enum the module can name, for resolving bare variants.
    pub enums: Vec<Arc<EnumType>>,
}

/// The declarations a module exports, for `alias.member`.
#[derive(Debug, Default)]
pub struct Exports {
    /// Constants and functions by name, `private` ones included (and refused at the use).
    pub values: BTreeMap<String, Global>,
}

/// What kind of body is being checked, which decides what is visible.
#[derive(Debug, Clone)]
pub enum Body {
    /// A file's top level, and the blocks inside it.
    Top,
    /// A function body.
    Function {
        /// The function's name.
        name: String,
        /// Its return type.
        ret: Type,
    },
    /// A `test` block.
    Test,
    /// A `const` value, an `input` default or a parameter default: compile-time values only.
    Constant,
}

/// Why a value is checked against a type, for the mismatch message.
#[derive(Debug, Clone)]
pub enum Want {
    /// No particular reason.
    Value,
    /// An `if`, `where`, `require`, guard or `expect` condition.
    Condition,
    /// A charge's amount.
    Amount,
    /// An argument for a parameter of a function or method.
    Argument {
        /// The parameter.
        param: String,
        /// The function or method.
        callee: String,
    },
    /// A function's result.
    Return {
        /// The function.
        name: String,
    },
    /// A binding or input with an annotation.
    Annotation {
        /// The name annotated.
        name: String,
    },
    /// A record field.
    Field {
        /// The field.
        field: String,
    },
    /// A list item.
    Item,
    /// The value a `given` sets.
    Given {
        /// The path set, as written.
        path: String,
    },
    /// A total clause's amount.
    Total,
    /// A charge's `minimum:` or `maximum:` option.
    Limit {
        /// The option.
        option: String,
    },
}

/// One module's check in progress.
pub struct Checker<'a> {
    /// The module's source.
    pub source: &'a str,
    /// The module.
    pub module: ModuleId,
    /// Whether the module is the standard library's, whose doc comments are translated.
    pub std: bool,
    /// The file's kind.
    pub kind: FileKind,
    /// What the module can name.
    pub globals: &'a Globals,
    /// What every module exports, by module index.
    pub exports: &'a [Exports],
    /// The schema.
    pub schema: &'a Schema,
    /// The request's context.
    pub context: &'a Context,
    /// Every symbol of the program.
    pub symbols: &'a mut Vec<Symbol>,
    /// Every reference of the program.
    pub references: &'a mut Vec<Reference>,
    /// This module's findings and side tables.
    pub out: &'a mut ModuleAnalysis,
    /// The scopes of the body being checked.
    pub scopes: Scopes,
    /// The body being checked.
    pub body: Body,
    /// How many `for` loops enclose the statement being checked.
    pub loops: u32,
    /// The spans of the statements enclosing the expression being checked, innermost last.
    pub statements: Vec<Span>,
    /// The functions the body being checked calls, for the call graph.
    pub calls: Vec<(ModuleId, String, Span)>,
    /// Names the file's top level declares with `let` or `var`, which functions and tests
    /// cannot read.
    pub file_lets: Vec<String>,
    /// The labels and codes the tariff bills, for `charge(…)` in its tests.
    pub billed: Billed,
    /// How many lambdas enclose the expression being checked.
    pub lambda_depth: u32,
    /// The globals (inputs, imported names, module aliases) read, for unused warnings.
    pub used: std::collections::BTreeSet<String>,
    /// The span starts of `match`es found exhaustive, for return-path analysis.
    pub exhaustive: std::collections::BTreeSet<u32>,
    /// Whether the test being checked has reached an `expect`.
    pub expected_yet: bool,
    /// Names read before the `let` that defines them (already reported), so that `let` is not
    /// also reported unused.
    pub early: std::collections::BTreeSet<String>,
    /// Whether the body is a test that expects a refusal, whose `let`s run code for the refusal
    /// it may raise and so are not reported unused.
    pub expects_refusal: bool,
    /// What the property being checked has varied and given, when the body is a property's.
    pub property: Option<Box<super::property::PropertyScope>>,
    /// The choices around the statement being checked, and the labels used so far.
    pub choice: super::choose::ChoiceState,
}

/// The charges and choices a tariff writes, for checking `charge(…)` and `chosen(…)` in tests
/// and `bill.charge(…)` in its own code.
#[derive(Debug, Default, Clone)]
pub struct Billed {
    /// Literal labels.
    pub labels: Vec<String>,
    /// Codes.
    pub codes: Vec<String>,
    /// Whether some label interpolates, so any label may match.
    pub any_label: bool,
    /// The labels of the file's `best` and `worst` choices, for `chosen(…)` in tests.
    pub choices: Vec<String>,
    /// Every `charge` of the file with where it is written, for `bill.charge(…)`'s order.
    pub sites: Vec<super::bill::ChargeSite>,
}

impl Checker<'_> {
    /// Reports a finding.
    pub fn report(&mut self, diagnostic: Diagnostic) {
        self.out.diagnostics.push(diagnostic);
    }

    /// The source text of a span.
    #[must_use]
    pub fn text(&self, span: Span) -> &str {
        snippet(self.source, span)
    }

    /// Records an expression's type and returns it.
    pub fn record(&mut self, expr: &Expr, ty: Type) -> Type {
        let index = usize::try_from(expr.id.0).unwrap_or(usize::MAX);
        if self.out.types.len() <= index && index < 1 << 20 {
            self.out.types.resize(index + 1, None);
        }
        if let Some(slot) = self.out.types.get_mut(index) {
            *slot = Some(ty.clone());
        }
        ty
    }

    /// The type recorded for a node.
    #[must_use]
    pub fn recorded(&self, id: NodeId) -> Option<Type> {
        self.out
            .types
            .get(usize::try_from(id.0).unwrap_or(usize::MAX))
            .cloned()
            .flatten()
    }

    /// Records what a call resolved to.
    pub fn callee(&mut self, call: &Expr, callee: Callee) {
        self.out.callees.insert(call.id, callee);
    }

    /// Adds a symbol and returns its id.
    pub fn symbol(&mut self, symbol: Symbol) -> SymbolId {
        let id = SymbolId(u32::try_from(self.symbols.len()).unwrap_or(u32::MAX));
        self.symbols.push(symbol);
        id
    }

    /// Records a reference to a symbol.
    pub fn reference(&mut self, span: Span, symbol: SymbolId) {
        self.references.push(Reference {
            module: self.module,
            span,
            symbol,
        });
    }

    /// Records an inferred type for an inlay hint.
    pub fn hint(&mut self, span: Span, ty: &Type, kind: HintKind) {
        self.out.hints.push(Hint {
            span,
            ty: ty.clone(),
            kind,
        });
    }

    /// Whether the body being checked sees the record (schema roots and tables) and inputs.
    #[must_use]
    pub fn sees_record(&self) -> bool {
        matches!(self.body, Body::Top | Body::Test)
    }

    /// Declares a local, refusing a name that shadows a visible root, input, function or
    /// other name.
    pub fn declare(&mut self, name: &Name, kind: LocalKind, ty: Type) -> Option<usize> {
        if name.text.is_empty() {
            return None;
        }
        self.check_shadow(name);
        let symbol_kind = match kind {
            LocalKind::Let => SymbolKind::Let,
            LocalKind::Var => SymbolKind::Var,
            LocalKind::Param => SymbolKind::Param,
            LocalKind::Loop => SymbolKind::LoopVariable,
            LocalKind::Lambda => SymbolKind::LambdaParam,
            LocalKind::Bound => SymbolKind::Binding,
            LocalKind::Vary => SymbolKind::Vary,
        };
        let symbol = self.symbol(Symbol {
            name: name.text.clone(),
            kind: symbol_kind,
            module: Some(self.module),
            span: Some(name.span),
            ty: Some(ty.clone()),
            doc: None,
        });
        let reads = u32::from(self.early.remove(&name.text));
        Some(self.scopes.declare(Local {
            name: name.text.clone(),
            kind,
            ty,
            span: name.span,
            symbol,
            reads,
        }))
    }

    fn check_shadow(&mut self, name: &Name) {
        let what: Option<(Message, Option<Span>)> = if let Some(id) = self.scopes.find(&name.text) {
            let span = self.scopes.get(id).map(|local| local.span);
            Some((Message::new(message!("declared above")), span))
        } else if self.sees_record()
            && (self.schema.root(&name.text).is_some() || name.text == "tables")
        {
            Some((Message::new(message!("a schema root")), None))
        } else if self.kind == FileKind::Tariff && name.text == "bill" {
            Some((Message::new(message!("the bill so far")), None))
        } else {
            match self.globals.values.get(&name.text) {
                Some(Global::Input { .. }) if matches!(self.body, Body::Top) => Some((
                    Message::new(message!("an input")),
                    self.symbol_span(&name.text),
                )),
                Some(Global::Fn(_)) => Some((
                    Message::new(message!("a function")),
                    self.symbol_span(&name.text),
                )),
                Some(Global::Const { .. }) => Some((
                    Message::new(message!("a constant")),
                    self.symbol_span(&name.text),
                )),
                Some(Global::Module { .. }) => Some((
                    Message::new(message!("an imported module")),
                    self.symbol_span(&name.text),
                )),
                _ => None,
            }
        };
        if let Some((what, span)) = what {
            let mut diagnostic = Diagnostic::error(
                "shadow",
                name.span,
                Message::with(
                    message!("{0} is already {1}; pick another name"),
                    [quote(&name.text), what.into()],
                ),
            );
            if let Some(span) = span
                && (span.start < name.span.start || self.symbol_module(&name.text))
            {
                diagnostic =
                    diagnostic.with_related(span, Message::new(message!("It is declared here")));
            }
            self.report(diagnostic);
        }
    }

    fn symbol_span(&self, name: &str) -> Option<Span> {
        let symbol = match self.globals.values.get(name)? {
            Global::Input { symbol, .. }
            | Global::Const { symbol, .. }
            | Global::Module { symbol, .. } => *symbol,
            Global::Fn(info) => info.symbol,
        };
        let found = self.symbols.get(usize::try_from(symbol.0).ok()?)?;
        if found.module == Some(self.module) {
            found.span
        } else {
            None
        }
    }

    fn symbol_module(&self, name: &str) -> bool {
        self.symbol_span(name).is_some()
    }

    /// The path key of an expression that names a stable, immutable value: a `let`, parameter,
    /// loop or pattern binding, input or root, and the fields read from it with `.`.
    #[must_use]
    pub fn path_key(&self, expr: &Expr) -> Option<PathKey> {
        match &expr.kind {
            super::super::syntax::ast::ExprKind::Name(name) => {
                if let Some(id) = self.scopes.find(&name.text) {
                    let local = self.scopes.get(id)?;
                    if local.kind == LocalKind::Var {
                        return None;
                    }
                    return Some(vec![format!("#local:{id}")]);
                }
                if self.sees_record() {
                    if self.schema.root(&name.text).is_some() {
                        return Some(vec![format!("#root:{}", name.text)]);
                    }
                    if let (Some(Global::Input { .. }), Body::Top) =
                        (self.globals.values.get(&name.text), &self.body)
                    {
                        return Some(vec![format!("#input:{}", name.text)]);
                    }
                }
                None
            }
            super::super::syntax::ast::ExprKind::Field {
                target,
                name,
                optional: false,
            } => {
                let mut key = self.path_key(target)?;
                key.push(name.text.clone());
                Some(key)
            }
            _ => None,
        }
    }
}
