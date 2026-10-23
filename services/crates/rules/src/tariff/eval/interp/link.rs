//! The program as the interpreter reads it: names resolved through the loader's symbol tables
//! to the declarations in the syntax tree.
//!
//! The loader has already checked that every import names a module and a declaration that
//! exist; this only answers "what does this name mean in this module" with the tree node.
//! Module 0 is the root file.

use crate::tariff::syntax::ast::{Const, EnumDecl, FnDecl, Input, Item, ItemKind, Test, TypeDecl};
use crate::tariff::syntax::program::{Decl as Declared, Module, ModuleId, Program, Resolved};

/// A declaration a name resolves to.
#[derive(Debug, Clone, Copy)]
pub enum Decl<'a> {
    /// A function.
    Fn(&'a FnDecl),
    /// A constant.
    Const(&'a Const),
    /// A record type.
    Type(&'a TypeDecl),
    /// An enum.
    Enum(&'a EnumDecl),
}

/// A loaded program, read by the interpreter.
#[derive(Debug, Clone, Copy)]
pub struct Linked<'a> {
    program: &'a Program,
}

/// The module index the interpreter uses for a [`ModuleId`].
#[must_use]
pub fn index(id: ModuleId) -> usize {
    usize::try_from(id.0).unwrap_or(usize::MAX)
}

/// The [`ModuleId`] of a module index.
#[must_use]
pub fn id(index: usize) -> ModuleId {
    ModuleId(u32::try_from(index).unwrap_or(u32::MAX))
}

impl<'a> Linked<'a> {
    /// Reads a loaded program.
    #[must_use]
    pub const fn new(program: &'a Program) -> Self {
        Self { program }
    }

    /// The program.
    #[must_use]
    pub const fn program(&self) -> &'a Program {
        self.program
    }

    /// The module at `index`.
    #[must_use]
    pub fn module(&self, index: usize) -> Option<&'a Module> {
        self.program.modules.get(index)
    }

    fn item(self, module: ModuleId, item: usize) -> Option<&'a Item> {
        self.program.module(module)?.file.items.get(item)
    }

    /// What `name` means in `module`: a local or imported declaration, with the module that
    /// declares it.
    #[must_use]
    pub fn resolve(&self, module: usize, name: &str) -> Option<(usize, Decl<'a>)> {
        let Resolved::Decl { module, decl } = self.program.resolve(id(module), name)? else {
            return None;
        };
        let item = self.item(module, decl.item())?;
        let found = match (&item.kind, decl) {
            (ItemKind::Fn(decl), Declared::Fn(_)) => Decl::Fn(decl),
            (ItemKind::Const(decl), Declared::Const(_)) => Decl::Const(decl),
            (ItemKind::Type(decl), Declared::Type(_)) => Decl::Type(decl),
            (ItemKind::Enum(decl), Declared::Enum(_)) => Decl::Enum(decl),
            _ => return None,
        };
        Some((index(module), found))
    }

    /// The module a name aliases in `module` (`use std.fuel` makes `fuel` one).
    #[must_use]
    pub fn alias(&self, module: usize, name: &str) -> Option<usize> {
        self.module(module)?
            .aliases
            .get(name)
            .map(|alias| index(alias.module))
    }

    /// The function `name` in `module`.
    #[must_use]
    pub fn function(&self, module: usize, name: &str) -> Option<(usize, &'a FnDecl)> {
        match self.resolve(module, name)? {
            (at, Decl::Fn(decl)) => Some((at, decl)),
            _ => None,
        }
    }

    /// The constant `name` in `module`.
    #[must_use]
    pub fn constant(&self, module: usize, name: &str) -> Option<(usize, &'a Const)> {
        match self.resolve(module, name)? {
            (at, Decl::Const(decl)) => Some((at, decl)),
            _ => None,
        }
    }

    /// A module's inputs, in declaration order.
    #[must_use]
    pub fn inputs(&self, module: usize) -> Vec<&'a Input> {
        self.module(module).map_or_else(Vec::new, |module| {
            module
                .file
                .items
                .iter()
                .filter_map(|item| match &item.kind {
                    ItemKind::Input(input) => Some(input),
                    _ => None,
                })
                .collect()
        })
    }

    /// A module's tests, in declaration order.
    #[must_use]
    pub fn tests(&self, module: usize) -> Vec<&'a Test> {
        self.module(module).map_or_else(Vec::new, |module| {
            module
                .file
                .items
                .iter()
                .filter_map(|item| match &item.kind {
                    ItemKind::Test(test) => Some(test),
                    _ => None,
                })
                .collect()
        })
    }
}
