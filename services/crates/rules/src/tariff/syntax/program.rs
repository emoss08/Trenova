//! Loading a file with everything it imports, as one [`Program`] the checker, the evaluator and
//! the editor share.
//!
//! The loader parses the root, follows each `use` transitively (standard library modules from
//! [`stdlib::source`], organisation libraries from [`Context::libraries`]) and parses each module
//! once. For every module it builds a symbol table of what the module declares and resolves its
//! imports to the modules and names they refer to. It does no type checking and no name
//! resolution inside bodies.
//!
//! What it reports, each against the module where the problem is written:
//!
//! | Code | Finding |
//! |---|---|
//! | `unknown-module` | `use` names a module that does not exist (with a did-you-mean) |
//! | `module-path` | a module path whose first part is not `std` or `org` |
//! | `import-cycle` | modules that import each other, naming the cycle |
//! | `import-not-library` | importing a `tariff` or `condition`; only libraries can be imported |
//! | `unknown-import` | importing a name the module does not declare (with a did-you-mean) |
//! | `private-import` | importing a name the module declares `private` |
//! | `duplicate-import` | the same name imported twice |
//! | `import-collision` | an imported name that is also declared in the importing file |
//! | `duplicate-declaration` | a name declared twice in one module (related: the first) |
//! | `library-item` | a library holding anything but `use`, `const`, `type`, `enum`, `fn`, `test` |
//! | `file-kind` | the root's header is not the kind the caller expected |
//! | `limit` | more than [`MAX_MODULES`] modules or [`MAX_PROGRAM_BYTES`] of source |

use super::ast;
use super::ast::FileKind;
use super::ast::ItemKind;
use super::parse;
use crate::span::Span;
use crate::span::Utf16Index;
use crate::tariff::context::Context;
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::diagnostic::nearest;
use crate::tariff::stdlib;
use std::collections::BTreeMap;
use trenova_module_kit::message;
use trenova_module_kit::message::Arg;
use trenova_module_kit::message::Message;

/// The most modules one program may load, the root included.
pub const MAX_MODULES: usize = 64;

/// The most source one program may load, all modules together, in bytes.
pub const MAX_PROGRAM_BYTES: usize = 1024 * 1024;

/// A module's index in [`Program::modules`]; the root is `ModuleId(0)`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord, Default)]
pub struct ModuleId(pub u32);

impl ModuleId {
    /// The root module.
    pub const ROOT: Self = Self(0);

    fn index(self) -> usize {
        self.0 as usize
    }
}

/// What a declared name is, with the index of its item in the module's [`ast::File::items`].
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Decl {
    /// `fn`.
    Fn(usize),
    /// `const`.
    Const(usize),
    /// `type`.
    Type(usize),
    /// `enum`.
    Enum(usize),
    /// `input`.
    Input(usize),
    /// `test`, keyed by the test's name.
    Test(usize),
}

impl Decl {
    /// The index of the declaring item in the module's file.
    #[must_use]
    pub const fn item(self) -> usize {
        match self {
            Self::Fn(item)
            | Self::Const(item)
            | Self::Type(item)
            | Self::Enum(item)
            | Self::Input(item)
            | Self::Test(item) => item,
        }
    }
}

/// One declared name.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Symbol {
    /// What it declares.
    pub decl: Decl,
    /// Written `private`: other modules cannot import it.
    pub private: bool,
    /// Where the name is written in the declaration.
    pub span: Span,
}

/// A name brought into a module by `use path.{ name }`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Import {
    /// The module the name is declared in.
    pub module: ModuleId,
    /// The name as that module declares it (the same as the local name: imports are not
    /// renamed).
    pub name: String,
    /// Where the name is written in the `use`.
    pub span: Span,
}

/// A module named in a file by `use std.fuel` (as `fuel`) or `use std.fuel as f` (as `f`).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Alias {
    /// The module it names.
    pub module: ModuleId,
    /// Where the alias is written: the last part of the path, or the name after `as`.
    pub span: Span,
}

/// One loaded file.
#[derive(Debug, Clone)]
pub struct Module {
    /// Its index in the program.
    pub id: ModuleId,
    /// The module path (`std.ltl`, `org.accessorials`); `None` for the root.
    pub path: Option<String>,
    /// The pinned version of an organisation library, from the context.
    pub version: Option<String>,
    /// Whether it is a standard library module: loaded by a `std.` path, or a root whose source
    /// is a standard library module's exactly. Its doc comments are translated
    /// ([`crate::tariff::docs`]).
    pub std: bool,
    /// The source.
    pub source: String,
    /// Byte to UTF-16 conversion for the source.
    pub utf16: Utf16Index,
    /// The parsed file.
    pub file: ast::File,
    /// Every name the module declares except tests (functions, constants, types, enums and
    /// inputs share one namespace), by name.
    pub symbols: BTreeMap<String, Symbol>,
    /// The module's tests, by name.
    pub tests: BTreeMap<String, Symbol>,
    /// The names it imports, by local name.
    pub imports: BTreeMap<String, Import>,
    /// The modules it names, by alias.
    pub aliases: BTreeMap<String, Alias>,
}

/// What a name in a module refers to.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Resolved {
    /// A declaration, in the module that declares it (the module itself, or the one an import
    /// came from).
    Decl {
        /// The declaring module.
        module: ModuleId,
        /// What it is.
        decl: Decl,
    },
    /// A module, named by `use`.
    Module(ModuleId),
}

/// A file and everything it imports.
#[derive(Debug, Clone)]
pub struct Program {
    /// The modules; the root is `ModuleId(0)`.
    pub modules: Vec<Module>,
    /// Every finding so far, with the module it is in: each module's syntax diagnostics followed
    /// by the loader's own. A consumer reports these, not each file's `diagnostics`.
    pub diagnostics: Vec<(ModuleId, Diagnostic)>,
    /// The modules in dependency order: every module after the modules it imports.
    order: Vec<ModuleId>,
}

impl Program {
    /// The root module: always present in a program [`load`] built.
    #[must_use]
    pub fn root(&self) -> Option<&Module> {
        self.modules.first()
    }

    /// The module with this id.
    #[must_use]
    pub fn module(&self, id: ModuleId) -> Option<&Module> {
        self.modules.get(id.index())
    }

    /// The loaded module at `path` (`std.ltl`).
    #[must_use]
    pub fn module_named(&self, path: &str) -> Option<&Module> {
        self.modules
            .iter()
            .find(|module| module.path.as_deref() == Some(path))
    }

    /// What `module` declares as `name` (not what it imports).
    #[must_use]
    pub fn decl(&self, module: ModuleId, name: &str) -> Option<&Symbol> {
        self.module(module)?.symbols.get(name)
    }

    /// What `name` refers to in `module`: a local declaration, an imported name (resolved to
    /// its declaration) or a module alias.
    #[must_use]
    pub fn resolve(&self, module: ModuleId, name: &str) -> Option<Resolved> {
        let current = self.module(module)?;
        if let Some(symbol) = current.symbols.get(name) {
            return Some(Resolved::Decl {
                module,
                decl: symbol.decl,
            });
        }
        if let Some(import) = current.imports.get(name) {
            let symbol = self.decl(import.module, &import.name)?;
            return Some(Resolved::Decl {
                module: import.module,
                decl: symbol.decl,
            });
        }
        current
            .aliases
            .get(name)
            .map(|alias| Resolved::Module(alias.module))
    }

    /// Every module in dependency order: each after the modules it imports, the root last.
    pub fn in_dependency_order(&self) -> impl Iterator<Item = &Module> {
        self.order.iter().filter_map(|id| self.module(*id))
    }
}

/// Loads `source` (whose header should be `expected`, when given) with every module it imports.
#[must_use]
pub fn load(source: &str, expected: Option<FileKind>, context: &Context) -> Program {
    let mut loader = Loader {
        context,
        modules: Vec::new(),
        diagnostics: Vec::new(),
        order: Vec::new(),
        bytes: 0,
    };
    let root = loader.add(None, None, source.to_owned());
    if let Some(expected) = expected {
        loader.check_kind(expected);
    }
    let mut stack = Vec::new();
    loader.visit(root, &mut stack);
    let syntax: Vec<(ModuleId, Diagnostic)> = loader
        .modules
        .iter()
        .flat_map(|module| {
            module
                .file
                .diagnostics
                .iter()
                .map(|diagnostic| (module.id, diagnostic.clone()))
        })
        .collect();
    loader.diagnostics.splice(0..0, syntax);
    loader
        .diagnostics
        .sort_by_key(|(module, diagnostic)| (*module, diagnostic.span.start));
    Program {
        modules: loader.modules,
        diagnostics: loader.diagnostics,
        order: loader.order,
    }
}

struct Loader<'a> {
    context: &'a Context,
    modules: Vec<Module>,
    diagnostics: Vec<(ModuleId, Diagnostic)>,
    order: Vec<ModuleId>,
    bytes: usize,
}

/// A module a `use` refers to, before it is loaded.
struct Found {
    source: String,
    version: Option<String>,
}

impl Loader<'_> {
    fn report(&mut self, module: ModuleId, diagnostic: Diagnostic) {
        self.diagnostics.push((module, diagnostic));
    }

    fn add(&mut self, path: Option<String>, version: Option<String>, source: String) -> ModuleId {
        let id = ModuleId(u32::try_from(self.modules.len()).unwrap_or(u32::MAX));
        self.bytes += source.len();
        let file = parse(&source);
        let utf16 = Utf16Index::new(&source);
        let std = match &path {
            Some(path) => path.starts_with("std."),
            None => file
                .header
                .as_ref()
                .filter(|header| header.kind == FileKind::Library)
                .and_then(|header| stdlib::source(&header.name.text))
                .is_some_and(|shipped| shipped == source),
        };
        let mut module = Module {
            id,
            path,
            version,
            std,
            source,
            utf16,
            file,
            symbols: BTreeMap::new(),
            tests: BTreeMap::new(),
            imports: BTreeMap::new(),
            aliases: BTreeMap::new(),
        };
        let findings = declare(&mut module);
        for finding in findings {
            self.report(id, finding);
        }
        self.modules.push(module);
        id
    }

    fn check_kind(&mut self, expected: FileKind) {
        let Some(header) = self
            .modules
            .first()
            .and_then(|root| root.file.header.clone())
        else {
            return;
        };
        if header.kind != expected {
            self.report(
                ModuleId::ROOT,
                Diagnostic::error(
                    "file-kind",
                    header.span,
                    Message::with(
                        message!("This file must be a `{0}`, not a `{1}`"),
                        [
                            Arg::from(expected.keyword()),
                            Arg::from(header.kind.keyword()),
                        ],
                    ),
                ),
            );
        }
    }

    fn visit(&mut self, id: ModuleId, stack: &mut Vec<ModuleId>) {
        stack.push(id);
        let uses: Vec<ast::Use> = self
            .modules
            .get(id.index())
            .map(|module| {
                module
                    .file
                    .items
                    .iter()
                    .filter_map(|item| match &item.kind {
                        ItemKind::Use(using) => Some(using.clone()),
                        _ => None,
                    })
                    .collect()
            })
            .unwrap_or_default();
        for using in uses {
            if let Some(target) = self.follow(id, &using, stack) {
                self.bind(id, &using, target);
            }
        }
        stack.pop();
        self.order.push(id);
    }

    fn path_span(using: &ast::Use) -> Span {
        using
            .path
            .iter()
            .fold(None, |span: Option<Span>, part| {
                Some(span.map_or(part.span, |span| span.cover(part.span)))
            })
            .unwrap_or_default()
    }

    /// The module a `use` refers to, loading it (and what it imports) the first time.
    fn follow(
        &mut self,
        from: ModuleId,
        using: &ast::Use,
        stack: &mut Vec<ModuleId>,
    ) -> Option<ModuleId> {
        if using.path.iter().any(|part| part.text.is_empty()) {
            return None;
        }
        let path = using.dotted();
        let span = Self::path_span(using);
        let root = using.path.first().map(|part| part.text.as_str());
        if !matches!(root, Some("std" | "org")) || using.path.len() < 2 {
            self.report(
                from,
                Diagnostic::error(
                    "module-path",
                    span,
                    Message::with(
                        message!(
                            "`{0}` is not a module path; a module is `std.<name>` from the standard library or `org.<name>` from your organisation's libraries"
                        ),
                        [Arg::from(path)],
                    ),
                ),
            );
            return None;
        }
        if let Some(existing) = self
            .modules
            .iter()
            .find(|m| m.path.as_deref() == Some(&path))
        {
            let existing = existing.id;
            if let Some(position) = stack.iter().position(|id| *id == existing) {
                self.cycle(
                    from,
                    span,
                    stack.get(position..).unwrap_or_default(),
                    existing,
                );
                return None;
            }
            return Some(existing);
        }
        let Some(found) = self.find(&path) else {
            self.unknown_module(from, span, &path);
            return None;
        };
        if self.modules.len() >= MAX_MODULES || self.bytes + found.source.len() > MAX_PROGRAM_BYTES
        {
            self.report(
                from,
                Diagnostic::error(
                    "limit",
                    span,
                    Message::with(
                        message!(
                            "This tariff imports too much: a program can load at most {0} modules and {1} KiB of source"
                        ),
                        [Arg::from(MAX_MODULES), Arg::from(MAX_PROGRAM_BYTES / 1024)],
                    ),
                ),
            );
            return None;
        }
        let id = self.add(Some(path), found.version, found.source);
        self.visit(id, stack);
        Some(id)
    }

    fn find(&self, path: &str) -> Option<Found> {
        if path.starts_with("std.") {
            return stdlib::source(path).map(|source| Found {
                source: source.to_owned(),
                version: None,
            });
        }
        self.context
            .libraries
            .iter()
            .find(|library| library.path == path)
            .map(|library| Found {
                source: library.source.clone(),
                version: library.version.clone(),
            })
    }

    fn unknown_module(&mut self, from: ModuleId, span: Span, path: &str) {
        let available: Vec<&str> = stdlib::MODULES
            .iter()
            .map(|(module, _)| *module)
            .chain(
                self.context
                    .libraries
                    .iter()
                    .map(|library| library.path.as_str()),
            )
            .collect();
        let message = match nearest(path, available.iter().copied()) {
            Some(suggestion) => Message::with(
                message!("There is no module `{0}`. Did you mean `{1}`?"),
                [Arg::from(path), Arg::from(suggestion)],
            ),
            None => Message::with(message!("There is no module `{0}`"), [Arg::from(path)]),
        };
        let mut diagnostic = Diagnostic::error("unknown-module", span, message);
        if let Some(suggestion) = nearest(path, available.iter().copied()) {
            diagnostic = diagnostic.with_fix(replace_fix(span, suggestion));
        }
        self.report(from, diagnostic);
    }

    fn cycle(&mut self, from: ModuleId, span: Span, stack: &[ModuleId], target: ModuleId) {
        let names: Vec<String> = stack
            .iter()
            .chain(std::iter::once(&target))
            .filter_map(|id| self.modules.get(id.index()))
            .map(|module| {
                module
                    .path
                    .clone()
                    .unwrap_or_else(|| String::from("this file"))
            })
            .collect();
        self.report(
            from,
            Diagnostic::error(
                "import-cycle",
                span,
                Message::with(
                    message!("These modules import each other in a cycle: {0}"),
                    [Arg::from(names.join(" → "))],
                ),
            ),
        );
    }

    /// Records what a `use` brings into `from`: its names and its alias.
    fn bind(&mut self, from: ModuleId, using: &ast::Use, target: ModuleId) {
        let Some(target_module) = self.modules.get(target.index()) else {
            return;
        };
        if target_module.file.header.as_ref().map(|h| h.kind) != Some(FileKind::Library) {
            let kind = target_module
                .file
                .header
                .as_ref()
                .map_or("tariff", |h| h.kind.keyword());
            self.report(
                from,
                Diagnostic::error(
                    "import-not-library",
                    Self::path_span(using),
                    Message::with(
                        message!("`{0}` is a `{1}`; only a library can be imported"),
                        [Arg::from(using.dotted()), Arg::from(kind)],
                    ),
                ),
            );
            return;
        }
        let target_public: Vec<(String, Symbol)> = target_module
            .symbols
            .iter()
            .map(|(name, symbol)| (name.clone(), symbol.clone()))
            .collect();
        if let Some(names) = &using.names {
            for name in names {
                self.import_name(from, using, target, &target_public, name);
            }
        }
        let alias = match (&using.alias, &using.names) {
            (Some(alias), _) => Some(alias.clone()),
            (None, None) => using.path.last().cloned(),
            (None, Some(_)) => None,
        };
        if let Some(alias) = alias
            && !alias.text.is_empty()
            && self.free_name(from, &alias.text, alias.span)
            && let Some(module) = self.modules.get_mut(from.index())
        {
            module.aliases.insert(
                alias.text.clone(),
                Alias {
                    module: target,
                    span: alias.span,
                },
            );
        }
    }

    fn import_name(
        &mut self,
        from: ModuleId,
        using: &ast::Use,
        target: ModuleId,
        target_symbols: &[(String, Symbol)],
        name: &ast::Name,
    ) {
        let declared = target_symbols
            .iter()
            .find(|(declared, _)| *declared == name.text);
        match declared {
            None => {
                let public = target_symbols
                    .iter()
                    .filter(|(_, symbol)| !symbol.private)
                    .map(|(declared, _)| declared.as_str());
                let suggestion = nearest(&name.text, public);
                let message = match suggestion {
                    Some(suggestion) => Message::with(
                        message!("`{0}` does not declare `{1}`. Did you mean `{2}`?"),
                        [
                            Arg::from(using.dotted()),
                            Arg::from(name.text.clone()),
                            Arg::from(suggestion),
                        ],
                    ),
                    None => Message::with(
                        message!("`{0}` does not declare `{1}`"),
                        [Arg::from(using.dotted()), Arg::from(name.text.clone())],
                    ),
                };
                let mut diagnostic = Diagnostic::error("unknown-import", name.span, message);
                if let Some(suggestion) = suggestion {
                    diagnostic = diagnostic.with_fix(replace_fix(name.span, suggestion));
                }
                self.report(from, diagnostic);
            }
            Some((_, symbol)) if symbol.private => {
                self.report(
                    from,
                    Diagnostic::error(
                        "private-import",
                        name.span,
                        Message::with(
                            message!("`{0}` is private to `{1}`, so it can't be imported"),
                            [Arg::from(name.text.clone()), Arg::from(using.dotted())],
                        ),
                    ),
                );
            }
            Some(_) => {
                if self.free_name(from, &name.text, name.span)
                    && let Some(module) = self.modules.get_mut(from.index())
                {
                    module.imports.insert(
                        name.text.clone(),
                        Import {
                            module: target,
                            name: name.text.clone(),
                            span: name.span,
                        },
                    );
                }
            }
        }
    }

    /// Whether `name` can be brought into `module`: not declared there and not imported yet.
    /// Reports the collision when it cannot.
    fn free_name(&mut self, module: ModuleId, name: &str, span: Span) -> bool {
        let Some(current) = self.modules.get(module.index()) else {
            return false;
        };
        if let Some(local) = current.symbols.get(name) {
            let local = local.span;
            self.report(
                module,
                Diagnostic::error(
                    "import-collision",
                    span,
                    Message::with(
                        message!("`{0}` is imported, but this file also declares `{0}`"),
                        [Arg::from(name)],
                    ),
                )
                .with_related(local, Message::new(message!("Declared here"))),
            );
            return false;
        }
        let earlier = current
            .imports
            .get(name)
            .map(|import| import.span)
            .or_else(|| current.aliases.get(name).map(|alias| alias.span));
        if let Some(earlier) = earlier {
            self.report(
                module,
                Diagnostic::error(
                    "duplicate-import",
                    span,
                    Message::with(message!("`{0}` is already imported"), [Arg::from(name)]),
                )
                .with_related(earlier, Message::new(message!("First imported here"))),
            );
            return false;
        }
        true
    }
}

fn replace_fix(span: Span, with: &str) -> crate::tariff::diagnostic::Fix {
    crate::tariff::diagnostic::Fix {
        label: Message::with(message!("Change to `{0}`"), [Arg::from(with)]),
        changes: vec![crate::tariff::diagnostic::Change {
            span,
            insert: with.to_owned(),
        }],
    }
}

/// Builds a module's symbol tables, returning the duplicate declarations and, for a library, the
/// items a library may not hold.
fn declare(module: &mut Module) -> Vec<Diagnostic> {
    let mut findings = Vec::new();
    let library = module.file.header.as_ref().map(|h| h.kind) == Some(FileKind::Library);
    for (index, item) in module.file.items.iter().enumerate() {
        let entry = match &item.kind {
            ItemKind::Fn(function) => Some((&function.name, Decl::Fn(index))),
            ItemKind::Const(constant) => Some((&constant.name, Decl::Const(index))),
            ItemKind::Type(ty) => Some((&ty.name, Decl::Type(index))),
            ItemKind::Enum(enumeration) => Some((&enumeration.name, Decl::Enum(index))),
            ItemKind::Input(input) => Some((&input.name, Decl::Input(index))),
            ItemKind::Test(test) => Some((&test.name, Decl::Test(index))),
            ItemKind::Property(property) => Some((&property.name, Decl::Test(index))),
            ItemKind::Use(_) | ItemKind::Stmt(_) => None,
        };
        if library && matches!(item.kind, ItemKind::Input(_) | ItemKind::Stmt(_)) {
            findings.push(Diagnostic::error(
                "library-item",
                item.span,
                Message::new(message!(
                    "A library holds only `use`, `const`, `type`, `enum`, `fn` and `test`; inputs, bindings and charges belong in a tariff"
                )),
            ));
        }
        let Some((name, decl)) = entry else {
            continue;
        };
        if name.text.is_empty() {
            continue;
        }
        let table = if matches!(decl, Decl::Test(_)) {
            &mut module.tests
        } else {
            &mut module.symbols
        };
        if let Some(first) = table.get(&name.text) {
            let message = if matches!(decl, Decl::Test(_)) {
                Message::with(
                    message!("There is already a test named \"{0}\""),
                    [Arg::from(name.text.clone())],
                )
            } else {
                Message::with(
                    message!("`{0}` is already declared in this file"),
                    [Arg::from(name.text.clone())],
                )
            };
            findings.push(
                Diagnostic::error("duplicate-declaration", name.span, message)
                    .with_related(first.span, Message::new(message!("First declared here"))),
            );
            continue;
        }
        table.insert(
            name.text.clone(),
            Symbol {
                decl,
                private: item.private.is_some(),
                span: name.span,
            },
        );
    }
    findings
}
