//! One file loaded with its imports and checked, with the indexes every editor feature shares.
//!
//! A [`Session`] is built once per operation. Besides the [`Program`] and its [`Analysis`] it
//! holds the symbol declared at each position and the root file's references sorted by
//! position, so finding what a name means is a binary search rather than a scan, and a work
//! meter ([`Session::tick`]) every feature charges for the nodes and candidates it visits.

use trenova_module_kit::span::{Span, Utf16Index};
use crate::tariff::check::{Analysis, Reference, Symbol, SymbolId, SymbolKind, check};
use crate::tariff::context::Context;
use crate::tariff::schema::Schema;
use crate::tariff::syntax::ast::{File, FileKind};
use crate::tariff::syntax::program::{Decl, Module, ModuleId, Program, Resolved, load};
use std::cell::Cell;
use std::collections::BTreeMap;

/// A loaded and checked file.
#[derive(Debug)]
pub struct Session<'c> {
    /// The request's context.
    pub context: &'c Context,
    /// The file and every library it imports.
    pub program: Program,
    /// What the checker found.
    pub analysis: Analysis,
    /// The schema the file was checked against.
    pub schema: Schema,
    /// The symbol declared at each `(module, name start)`.
    declared: BTreeMap<(u32, u32), SymbolId>,
    /// The root file's references, by where they start.
    references: Vec<Reference>,
    /// Work done so far: nodes, tokens and candidates visited.
    steps: Cell<u64>,
    /// An empty module, answered when the program has no root.
    fallback: Module,
}

impl<'c> Session<'c> {
    /// Loads and checks `source`, whose header should be `expected` when given.
    #[must_use]
    pub fn new(source: &str, expected: Option<FileKind>, context: &'c Context) -> Self {
        let program = load(source, expected, context);
        let analysis = check(&program, context);
        let schema = Schema::new(&context.settings);
        let mut declared = BTreeMap::new();
        for (index, symbol) in analysis.symbols.iter().enumerate() {
            if let (Some(module), Some(span)) = (symbol.module, symbol.span) {
                let id = SymbolId(u32::try_from(index).unwrap_or(u32::MAX));
                declared.entry((module.0, span.start)).or_insert(id);
            }
        }
        let mut references: Vec<Reference> = analysis
            .references
            .iter()
            .filter(|reference| reference.module == ModuleId::ROOT)
            .copied()
            .collect();
        references.sort_by_key(|reference| (reference.span.start, reference.span.end));
        let work = program
            .modules
            .iter()
            .map(|module| module.file.tokens.len() as u64 + u64::from(module.file.node_count))
            .sum();
        Self {
            context,
            program,
            analysis,
            schema,
            declared,
            references,
            steps: Cell::new(work),
            fallback: empty_module(),
        }
    }

    /// Charges `count` units of work.
    pub fn tick(&self, count: usize) {
        self.steps
            .set(self.steps.get().saturating_add(count as u64));
    }

    /// The work done so far, the load and check included (each counted as one step per token
    /// and per node of every module).
    #[must_use]
    pub fn steps(&self) -> u64 {
        self.steps.get()
    }

    /// The root module.
    #[must_use]
    pub fn root(&self) -> &Module {
        self.program.root().unwrap_or(&self.fallback)
    }

    /// The root file's syntax tree.
    #[must_use]
    pub fn file(&self) -> &File {
        &self.root().file
    }

    /// The root file's source.
    #[must_use]
    pub fn source(&self) -> &str {
        &self.root().source
    }

    /// The root file's kind; a file without a header is read as a tariff.
    #[must_use]
    pub fn kind(&self) -> FileKind {
        self.file()
            .header
            .as_ref()
            .map_or(FileKind::Tariff, |header| header.kind)
    }

    /// The root's byte offset of a UTF-16 position.
    #[must_use]
    pub fn byte(&self, utf16: u32) -> u32 {
        let root = self.root();
        root.utf16.to_byte(&root.source, utf16)
    }

    /// The UTF-16 index of a module's source.
    #[must_use]
    pub fn utf16(&self, module: ModuleId) -> &Utf16Index {
        self.program
            .module(module)
            .map_or(&self.fallback.utf16, |found| &found.utf16)
    }

    /// The symbol with an id.
    #[must_use]
    pub fn symbol(&self, id: SymbolId) -> Option<&Symbol> {
        self.analysis.symbol(id)
    }

    /// The symbol whose name is declared at `span` in `module`.
    #[must_use]
    pub fn declared_at(&self, module: ModuleId, span: Span) -> Option<SymbolId> {
        self.declared.get(&(module.0, span.start)).copied()
    }

    /// The symbol a module's top-level name refers to: a declaration of the module, an imported
    /// name (the declaring module's symbol) or a module alias.
    #[must_use]
    pub fn global(&self, module: ModuleId, name: &str) -> Option<SymbolId> {
        match self.program.resolve(module, name)? {
            Resolved::Decl { module, decl } => {
                let declaring = self.program.module(module)?;
                let symbol = match decl {
                    Decl::Test(_) => declaring.tests.get(name),
                    _ => declaring.symbols.get(name),
                }?;
                self.declared_at(module, symbol.span)
            }
            Resolved::Module(_) => {
                let alias = self.program.module(module)?.aliases.get(name)?;
                self.declared_at(module, alias.span)
            }
        }
    }

    /// The symbol of a schema root (or `tables` or `bill`).
    #[must_use]
    pub fn root_symbol(&self, name: &str) -> Option<SymbolId> {
        self.analysis
            .symbols
            .iter()
            .take(self.schema.roots.len() + 2)
            .position(|symbol| symbol.kind == SymbolKind::Root && symbol.name == name)
            .and_then(|index| u32::try_from(index).ok())
            .map(SymbolId)
    }

    /// The name at a byte offset of the root file and the symbol it refers to or declares: a
    /// reference covering the offset, else a declaration whose name covers it.
    #[must_use]
    pub fn symbol_at(&self, offset: u32) -> Option<(Span, SymbolId)> {
        let after = self
            .references
            .partition_point(|reference| reference.span.start <= offset);
        self.tick(after.min(64));
        let reference = self
            .references
            .get(..after)
            .unwrap_or_default()
            .iter()
            .rev()
            .take(8)
            .find(|reference| {
                reference.span.end >= offset && reference.span.start < reference.span.end
            });
        if let Some(reference) = reference {
            return Some((reference.span, reference.symbol));
        }
        let (_, id) = self
            .declared
            .range((ModuleId::ROOT.0, 0)..=(ModuleId::ROOT.0, offset))
            .next_back()?;
        let span = self.symbol(*id)?.span?;
        (span.end >= offset && span.start < span.end).then_some((span, *id))
    }

    /// The symbol the checker recorded a reference to at exactly `span` of the root file.
    #[must_use]
    pub fn reference_at(&self, span: Span) -> Option<SymbolId> {
        let from = self.references.partition_point(|reference| {
            (reference.span.start, reference.span.end) < (span.start, span.end)
        });
        self.tick(1);
        self.references
            .get(from)
            .filter(|reference| reference.span == span)
            .map(|reference| reference.symbol)
    }

    /// Every reference to a symbol in the root file, in source order.
    pub fn references_to(&self, symbol: SymbolId) -> impl Iterator<Item = &Reference> {
        self.tick(self.references.len());
        self.references
            .iter()
            .filter(move |reference| reference.symbol == symbol)
    }

    /// The module path of a module: `std.ltl`, `org.fees`; `None` for the root.
    #[must_use]
    pub fn module_path(&self, module: ModuleId) -> Option<String> {
        self.program
            .module(module)
            .and_then(|found| found.path.clone())
    }
}

fn empty_module() -> Module {
    Module {
        id: ModuleId::ROOT,
        path: None,
        version: None,
        std: false,
        source: String::new(),
        utf16: Utf16Index::new(""),
        file: crate::tariff::syntax::parse(""),
        symbols: BTreeMap::new(),
        tests: BTreeMap::new(),
        imports: BTreeMap::new(),
        aliases: BTreeMap::new(),
    }
}
