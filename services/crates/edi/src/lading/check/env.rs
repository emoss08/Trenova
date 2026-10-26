//! What a piece of code can name while it is checked.

use trenova_module_kit::span::Span;

use crate::lading::types::{NodeRef, Type};

/// A local name.
#[derive(Debug, Clone)]
pub struct Local {
    /// The name.
    pub name: String,
    /// Its type.
    pub ty: Type,
    /// Where it is declared.
    pub span: Span,
}

/// The names and context of the code being checked.
#[derive(Debug, Clone, Default)]
pub struct Env {
    /// The file.
    pub unit: usize,
    /// The locals, innermost scope last.
    pub scopes: Vec<Vec<Local>>,
    /// The guide bare X12 references read from: the transaction of an inbound map.
    pub x12_root: Option<usize>,
    /// The document whose fields are names here.
    pub document: Option<&'static str>,
    /// Inside a segment filter: the segment bare element positions belong to.
    pub filter: Option<Type>,
    /// The function being checked, for the call graph.
    pub function: Option<(usize, String)>,
    /// The X12 node segment blocks belong to, in an outbound map.
    pub node: Option<NodeRef>,
    /// The guide an outbound map writes with.
    pub write_guide: Option<usize>,
    /// Whether a rule's outcomes may decline: rules for an inbound map.
    pub inbound: bool,
    /// Whether the code is a test.
    pub test: bool,
}

impl Env {
    /// An environment for a file.
    #[must_use]
    pub fn new(unit: usize) -> Self {
        Self {
            unit,
            scopes: vec![Vec::new()],
            ..Self::default()
        }
    }

    /// Opens a scope.
    pub fn push(&mut self) {
        self.scopes.push(Vec::new());
    }

    /// Closes the innermost scope.
    pub fn pop(&mut self) {
        self.scopes.pop();
        if self.scopes.is_empty() {
            self.scopes.push(Vec::new());
        }
    }

    /// Binds a name in the innermost scope.
    pub fn bind(&mut self, name: &str, ty: Type, span: Span) {
        if let Some(scope) = self.scopes.last_mut() {
            scope.push(Local {
                name: name.to_owned(),
                ty,
                span,
            });
        }
    }

    /// The innermost local with a name.
    #[must_use]
    pub fn lookup(&self, name: &str) -> Option<&Local> {
        self.scopes
            .iter()
            .rev()
            .flat_map(|scope| scope.iter().rev())
            .find(|local| local.name == name)
    }

    /// Every local in scope, innermost last, shadowed ones left out.
    #[must_use]
    pub fn visible(&self) -> Vec<(String, Type, Span)> {
        let mut seen = std::collections::BTreeSet::new();
        let mut out = Vec::new();
        for local in self
            .scopes
            .iter()
            .rev()
            .flat_map(|scope| scope.iter().rev())
        {
            if seen.insert(local.name.clone()) {
                out.push((local.name.clone(), local.ty.clone(), local.span));
            }
        }
        out.reverse();
        out
    }
}
