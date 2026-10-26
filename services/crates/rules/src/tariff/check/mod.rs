//! The static checker: everything the reference's "Type system" promises happens here, before
//! anything runs, identically in the browser and on the server.
//!
//! [`check`] takes a loaded [`Program`] (the root file and every library it imports) and the
//! request's [`Context`](crate::tariff::context::Context), and returns an [`Analysis`]:
//!
//! - **diagnostics** per module, each with a stable code, an English [`Message`](trenova_module_kit::message::Message)
//!   and fixes (the loader's and parser's stay in [`Program::diagnostics`]);
//! - the **type** of every expression, by [`NodeId`], per module ([`Analysis::type_of`], which
//!   the evaluator's static type oracle reads);
//! - every **symbol** (name, kind, module, declaration span, type, doc) and every
//!   **reference** to one, so go to definition, rename and hover need no second pass; the first
//!   symbols are the schema roots, in schema order, then `tables` and `bill`;
//! - the inferred type of every `let`, `var`, loop variable, lambda parameter, pattern binding
//!   and `=`-bodied function without `->`, for inlay hints ([`ModuleAnalysis::hints`]);
//! - the resolved **callee** of every call and the **enum** of every variant
//!   ([`ModuleAnalysis::callees`], [`ModuleAnalysis::variants`],
//!   [`ModuleAnalysis::pattern_variants`]);
//! - the root file's **inputs** (with a form descriptor and the default as a wire value), its
//!   **imports** with their pinned versions, and the rate **tables** the program reads
//!   ([`Analysis::tables`]), which a tariff version pins at the versions the context sent;
//! - `ok`: no error in any module, the loader's and parser's findings included.
//!
//! Modules are checked in dependency order: each module's declarations (`decls`: enums, record
//! types, function signatures, constants folded, inputs, imported names), then its bodies
//! (`program`: functions, file-level statements, tests, the rules of its file kind), then the
//! call graph across all of them.
//!
//! The work is split by responsibility: [`lower`] turns written types into [`Type`]s,
//! [`operators`] and [`methods`] are the tables of what each operator, property and method
//! gives, `expr`, `calls`, `tables` and `stmt` type expressions and statements, `arms` and
//! [`patterns`] check `match`, [`consts`] folds compile-time values, `inputs` describes the
//! inputs and validates contract values, `problems` words the recurring findings, `scope`
//! holds the names in scope and what flow narrowing has proven, and [`catalog`] and [`ops`]
//! publish it all.
//!
//! The codes, all errors unless marked:
//!
//! | Code | Finding |
//! |---|---|
//! | `unknown-name`, `unknown-type`, `unknown-field`, `unknown-member`, `unknown-function`, `unknown-variant`, `unknown-unit`, `unknown-table`, `unknown-argument` | a name nothing declares, with a "did you mean" fix |
//! | `unknown-code`, `inactive-code` | a charge code not in the accessorial catalog (once per code, in `charge … as CODE`, `charge(CODE)` and `shipment.requested(CODE)`, suggesting a code by spelling or by description), or inactive |
//! | `type-mismatch` | a value of the wrong type, with a fix writing a literal's unit |
//! | `unit-mismatch` | operands an operator does not combine, explained in pricing words |
//! | `optional` | a value that may be empty used where a value is needed; fixes: `?? zero`, `if let`, `require let` (and `?.` for a member) |
//! | `ambiguous-variant` | a bare variant of several enums with no type expected; a fix per enum |
//! | `before-definition`, `shadow`, `assign`, `closed-function`, `test-scope`, `not-constant` | scope rules; `.holiday` in a function (`closed-function`) or a constant or default (`not-constant`) |
//! | `arity`, `argument-order`, `duplicate-argument`, `not-callable`, `not-a-value`, `method-call`, `property-call`, `lambda`, `private`, `test-only` | calls |
//! | `table`, `table-keys`, `table-shape`, `table-value` | rate tables, and a test's given cells (`given-target` for one without keys) |
//! | `live-table` | a warning: a test or property reads a rate table without giving any of its cells, so it depends on the matrix's live data |
//! | `requested-code` | `shipment.requested(…)` given anything but a bare charge code |
//! | `charge-amount`, `charge-option`, `misplaced`, `total-order`, `round-step` | charges (a line's `minimum:` above its `maximum:` is `charge-option`), notes, reviews, total clauses (a freight or whole-bill minimum above a maximum is `total-order`), statements in the wrong place, a choice used as a value anywhere but a binding's whole value, and `bill` read outside a tariff's own code |
//! | `match-exhaustive`, `match-overlap`, `empty-range`, `pattern-binding` | `match` |
//! | `missing-return`, `return-type`, `recursion`, `recursive-type`, `const-type`, `condition-result`, `annotation-needed`, `branch-mismatch`, `missing-field`, `duplicate`, `given-order`, `given-target`, `not-indexable`, `not-iterable`, `not-summable`, `unordered`, `never-none`, `interpolation`, `pattern`, `quotient`, `unit-type`, `type-arguments`, `range`, `overflow`, `divide-by-zero` | the other rules |
//! | `vary-target`, `vary-values`, `vary-range`, `vary-read`, `vary-missing`, `assume-bill`, `monotone-target` | properties (`check/property.rs`); a property also uses `given-order`, `duplicate`, `range`, `empty-range`, `unordered`, `not-constant` and `shadow` |
//! | `unused`, `unused-value`, `unreachable`, `unreachable-arm`, `no-charge`, `not-billed`, `bill-order`, `needless-fallback`, `needless-optional-chain`, `needless-let`, `bounds`, `text-range` | warnings; `bounds` is a comparison or `match` arm an input's `where` bounds decide, `bill-order` a `bill.charge(…)` of a line billed only after it (`check/bill.rs`), `text-range` a text range whose literal ends have different lengths |
//! | `constraint` | an input's `where` constraint: not a range or a list, on a record input, a range of an unordered type or with no end, an empty range or list, a default or a test's `given` outside it |
//! | `choice`, `duplicate`, `unknown-choice` | `best` and `worst`: an empty label, fewer than two options, a label used twice; `chosen(…)` naming no choice |
//! | `unknown-option` | `chosen(…)` compared with a label that is no option of that choice |
//! | `input-value` | a contract input value that does not fit (`tariff.check` with `values`; a `path`, no range) |

mod arms;
mod bill;
mod bounds;
mod calls;
pub mod catalog;
mod checker;
mod choose;
mod codes;
mod constraints;
pub mod consts;
mod decls;
pub mod describe;
mod expr;
mod fixes;
mod inputs;
pub mod lower;
pub mod methods;
pub mod operators;
pub mod ops;
mod options;
pub mod patterns;
mod problems;
mod program;
mod property;
mod scope;
mod stmt;
mod tables;
mod visit;

pub use calls::{CONSTRUCTORS, REQUESTED, REQUESTED_DOC, REQUESTED_SIGNATURE, location_record};
pub use program::check;
pub use tables::{TABLE_METHODS, cell_type};

/// What a condition's tests read as its result: `expect applies`.
pub const APPLIES: &str = "applies";

use trenova_module_kit::span::Span;
use trenova_module_kit::diagnostic::{Diagnostic, Severity};
use crate::tariff::docs::Doc;
use crate::tariff::syntax::ast::NodeId;
use crate::tariff::types::{EnumType, Type};
use std::collections::{BTreeMap, BTreeSet};
use std::sync::Arc;

pub use crate::tariff::syntax::program::{ModuleId, Program};

/// What the checker found about a program.
#[derive(Debug, Default)]
pub struct Analysis {
    /// Per module, indexed by [`ModuleId`]: findings and side tables.
    pub modules: Vec<ModuleAnalysis>,
    /// Every symbol, indexed by [`SymbolId`].
    pub symbols: Vec<Symbol>,
    /// Every reference to a symbol (declarations excluded), in the order checked.
    pub references: Vec<Reference>,
    /// The root file's inputs, in declaration order.
    pub inputs: Vec<InputInfo>,
    /// Every library the program imports, directly or not, dependencies first.
    pub imports: Vec<ImportInfo>,
    /// No module has an error.
    pub ok: bool,
}

impl Analysis {
    /// The type of an expression node in a module.
    #[must_use]
    pub fn type_of(&self, module: ModuleId, node: NodeId) -> Option<&Type> {
        self.module(module)?.type_of(node)
    }

    /// One module's analysis.
    #[must_use]
    pub fn module(&self, module: ModuleId) -> Option<&ModuleAnalysis> {
        self.modules.get(usize::try_from(module.0).ok()?)
    }

    /// The symbol with an id.
    #[must_use]
    pub fn symbol(&self, id: SymbolId) -> Option<&Symbol> {
        self.symbols.get(usize::try_from(id.0).ok()?)
    }

    /// The codes of the rate tables the program reads, in the root or any library it imports,
    /// each once, in code order: what a tariff version pins.
    #[must_use]
    pub fn tables(&self) -> BTreeSet<&str> {
        self.modules
            .iter()
            .flat_map(|module| module.tables.iter().map(String::as_str))
            .collect()
    }

    /// Every finding with the module it is in, the root's first.
    pub fn diagnostics(&self) -> impl Iterator<Item = (ModuleId, &Diagnostic)> {
        self.modules.iter().enumerate().flat_map(|(index, module)| {
            let id = ModuleId(u32::try_from(index).unwrap_or(u32::MAX));
            module
                .diagnostics
                .iter()
                .map(move |diagnostic| (id, diagnostic))
        })
    }

    /// Whether any module has an error.
    #[must_use]
    pub fn has_errors(&self) -> bool {
        self.diagnostics()
            .any(|(_, diagnostic)| diagnostic.severity == Severity::Error)
    }
}

/// One module's findings and side tables.
#[derive(Debug, Default)]
pub struct ModuleAnalysis {
    /// The checker's findings, sorted by where they start (the loader's and parser's are in the
    /// [`Program`]).
    pub diagnostics: Vec<Diagnostic>,
    /// The type of each expression node, indexed by [`NodeId`]; `None` for a node the checker
    /// did not reach (inside a part that does not parse).
    pub types: Vec<Option<Type>>,
    /// What each call resolved to, by the call node.
    pub callees: BTreeMap<NodeId, Callee>,
    /// The enum of each variant written bare or qualified, by the variant node (and by pattern
    /// span start for patterns, under [`ModuleAnalysis::pattern_variants`]).
    pub variants: BTreeMap<NodeId, Arc<EnumType>>,
    /// The enum of each variant pattern, by the pattern's span start.
    pub pattern_variants: BTreeMap<u32, Arc<EnumType>>,
    /// The inferred types of bindings, for inlay hints.
    pub hints: Vec<Hint>,
    /// The type expected where each expression is written, by node, when the context expects
    /// one (an annotation, a parameter, the other side of a comparison): what the editor
    /// completes and offers enum variants for.
    pub expected: BTreeMap<NodeId, Type>,
    /// The codes of the rate tables the module reads, each once.
    pub tables: BTreeSet<String>,
}

impl ModuleAnalysis {
    /// The type of an expression node.
    #[must_use]
    pub fn type_of(&self, node: NodeId) -> Option<&Type> {
        self.types.get(usize::try_from(node.0).ok()?)?.as_ref()
    }

    /// The type expected where an expression node is written, when one is.
    #[must_use]
    pub fn expected_at(&self, node: NodeId) -> Option<&Type> {
        self.expected.get(&node)
    }
}

/// A symbol's identity, an index into [`Analysis::symbols`].
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub struct SymbolId(pub u32);

/// What a symbol is.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum SymbolKind {
    /// A schema root: `shipment`, `lane`, `customer`, `fuel`, `rating`, `tables`.
    Root,
    /// An `input`.
    Input,
    /// A `const`.
    Const,
    /// A function.
    Function,
    /// A function parameter.
    Param,
    /// A record type.
    Type,
    /// An enum.
    Enum,
    /// A `let`.
    Let,
    /// A `var`.
    Var,
    /// A `for` loop variable.
    LoopVariable,
    /// A lambda parameter.
    LambdaParam,
    /// A name bound by `if let`, `require let` or a `match` pattern.
    Binding,
    /// A property's `vary` name.
    Vary,
    /// A module imported whole, by its alias.
    Module,
    /// A `test`.
    Test,
}

impl SymbolKind {
    /// The kind's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Root => "root",
            Self::Input => "input",
            Self::Const => "const",
            Self::Function => "function",
            Self::Param => "param",
            Self::Type => "type",
            Self::Enum => "enum",
            Self::Let => "let",
            Self::Var => "var",
            Self::LoopVariable => "loop",
            Self::LambdaParam => "lambda",
            Self::Binding => "binding",
            Self::Vary => "vary",
            Self::Module => "module",
            Self::Test => "test",
        }
    }
}

/// A declared name.
#[derive(Debug, Clone)]
pub struct Symbol {
    /// The name.
    pub name: String,
    /// What it is.
    pub kind: SymbolKind,
    /// The module that declares it; `None` for the schema's roots.
    pub module: Option<ModuleId>,
    /// Where its name is declared; `None` for the schema's roots.
    pub span: Option<Span>,
    /// Its type, for values; a record or enum type for types.
    pub ty: Option<Type>,
    /// Its doc: its doc comment; a schema root's doc; for a module alias, the module's doc.
    pub doc: Option<Doc>,
}

/// A use of a symbol.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Reference {
    /// The module the use is in.
    pub module: ModuleId,
    /// The name as written at the use.
    pub span: Span,
    /// What it names.
    pub symbol: SymbolId,
}

/// What a call resolved to.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Callee {
    /// A user or library function.
    Function {
        /// The declaring module.
        module: ModuleId,
        /// The function's name there.
        name: String,
    },
    /// A built-in method of a value.
    Method(&'static methods::Method),
    /// `Number(i)` or `Int.parse(text)`.
    Conversion(&'static str),
    /// A test record constructor: `pickup`, `delivery`, `splitPickup`, `splitDelivery`,
    /// `commodity`.
    Constructor(&'static str),
    /// `charge(CODE)` or `charge("Label")` in a test.
    ChargeLookup,
    /// `shipment.requested(CODE)`.
    Requested,
    /// `bill.charge(CODE)` or `bill.charge("Label")` in a tariff's own code.
    BillCharge,
    /// A rate table method: `cell`, `band` or `interpolate` (indexing is not a call).
    Table {
        /// The table's code.
        code: String,
        /// The method.
        method: &'static str,
    },
}

/// What an inlay hint is for.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum HintKind {
    /// A `let` or `var` without an annotation.
    Binding,
    /// A loop variable.
    Loop,
    /// A lambda parameter.
    Lambda,
    /// A name bound by `if let`, `require let` or a pattern.
    Pattern,
    /// A function's return type left out with an `=` body.
    Return,
}

/// An inferred type to show after a name.
#[derive(Debug, Clone)]
pub struct Hint {
    /// The name (or, for a return type, the parameter list) the type belongs after.
    pub span: Span,
    /// The inferred type.
    pub ty: Type,
    /// What the name is.
    pub kind: HintKind,
}

/// One input of the root file.
#[derive(Debug, Clone)]
pub struct InputInfo {
    /// The input's name.
    pub name: String,
    /// Its doc comment, the form field's label.
    pub doc: Option<String>,
    /// Its type.
    pub ty: Type,
    /// A structured description of the type, for building the form.
    pub descriptor: serde_json::Value,
    /// Whether a value must be supplied (it has no default).
    pub required: bool,
    /// The default, as a wire value, when it has one that folds.
    pub default: Option<serde_json::Value>,
    /// The `where` constraint, folded; `None` when the input has none or it does not fold.
    pub constraint: Option<crate::tariff::constraint::Constraint>,
    /// Where the input's name is declared.
    pub span: Span,
}

/// One imported library.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ImportInfo {
    /// Its path: `std.fuel`, `org.accessorials`.
    pub path: String,
    /// The pinned version: the context's for an organisation library, the crate's for the
    /// standard library.
    pub version: Option<String>,
}

#[cfg(test)]
mod tests;
