//! What the checker learned about every name and expression, for the editor: where each name
//! is declared and used, the type of what is under the cursor, and what is in scope where.

use trenova_module_kit::span::Span;

use super::types::{NodeRef, Type};

/// What a name stands for.
#[derive(Debug, Clone, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub enum Target {
    /// A local: a `let`, an `each` or `for` binder, a parameter, a lambda's parameter or a
    /// flow's binder; declared at a span of a file.
    Local {
        /// The file.
        unit: usize,
        /// Where it is declared.
        span: Span,
    },
    /// A file-level declaration (`fn`, `crosswalk`, `const`).
    Decl {
        /// The file it is in.
        unit: usize,
        /// Its name.
        name: String,
    },
    /// A file by its header's name (a map, rules, flow, library, guide or the partner).
    File(String),
    /// A partner setting.
    Setting(String),
    /// A field of a schema record.
    Field {
        /// The record.
        record: &'static str,
        /// The field.
        field: &'static str,
    },
    /// A segment use in a guide.
    Segment(NodeRef),
    /// An element of a segment use.
    Element {
        /// The segment use.
        node: NodeRef,
        /// Its position, from 1.
        position: u32,
    },
    /// A loop in a guide.
    Loop(NodeRef),
    /// A standard function.
    Stdlib(&'static str),
    /// An enum variant.
    Variant {
        /// The enum.
        owner: &'static str,
        /// The variant.
        name: &'static str,
    },
    /// A type of the schema.
    Type(String),
    /// An event.
    Event(&'static str),
    /// `partner`, `org` or one of their fields.
    Builtin(String),
}

/// One use or declaration of a name.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Occurrence {
    /// The file.
    pub unit: usize,
    /// Where it is written.
    pub span: Span,
    /// What it stands for.
    pub target: Target,
    /// Whether this is the declaration.
    pub declaration: bool,
    /// Whether it is written to (an assignment's target).
    pub write: bool,
}

/// The type of an expression.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Typed {
    /// The file.
    pub unit: usize,
    /// The expression.
    pub span: Span,
    /// Its type.
    pub ty: Type,
}

/// What a block's code can name: its locals, and what it builds or reads.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ScopeFact {
    /// The file.
    pub unit: usize,
    /// The block.
    pub span: Span,
    /// The locals in scope, with their types and declarations.
    pub locals: Vec<(String, Type, Span)>,
    /// The record an inbound map's assignments here fill.
    pub builds: Option<Type>,
    /// The X12 node segment blocks here belong to (outbound), or bare references read from
    /// (inbound).
    pub x12: Option<NodeRef>,
    /// The document whose fields are names here (outbound maps, rules, tests).
    pub document: Option<&'static str>,
    /// The guide bare X12 references read from (inbound) or segments are written with
    /// (outbound).
    pub guide: Option<usize>,
}

/// Everything the checker learned.
#[derive(Debug, Clone, Default)]
pub struct Facts {
    /// Every name's uses and declarations.
    pub occurrences: Vec<Occurrence>,
    /// The types of names, members and calls.
    pub types: Vec<Typed>,
    /// What each block can name.
    pub scopes: Vec<ScopeFact>,
}

impl Facts {
    /// The innermost occurrence at a byte offset of a file.
    #[must_use]
    pub fn occurrence_at(&self, unit: usize, offset: u32) -> Option<&Occurrence> {
        self.occurrences
            .iter()
            .filter(|item| {
                item.unit == unit && item.span.start <= offset && offset <= item.span.end
            })
            .min_by_key(|item| item.span.end - item.span.start)
    }

    /// The innermost typed expression at a byte offset of a file.
    #[must_use]
    pub fn type_at(&self, unit: usize, offset: u32) -> Option<&Typed> {
        self.types
            .iter()
            .filter(|item| {
                item.unit == unit && item.span.start <= offset && offset <= item.span.end
            })
            .min_by_key(|item| item.span.end - item.span.start)
    }

    /// The typed expression ending exactly at an offset (what is before a dot).
    #[must_use]
    pub fn type_ending_at(&self, unit: usize, end: u32) -> Option<&Typed> {
        self.types
            .iter()
            .filter(|item| item.unit == unit && item.span.end == end)
            .max_by_key(|item| item.span.end - item.span.start)
    }

    /// The innermost scope around an offset.
    #[must_use]
    pub fn scope_at(&self, unit: usize, offset: u32) -> Option<&ScopeFact> {
        self.scopes
            .iter()
            .filter(|item| {
                item.unit == unit && item.span.start <= offset && offset <= item.span.end
            })
            .min_by_key(|item| item.span.end - item.span.start)
    }

    /// Every occurrence of a target.
    pub fn occurrences_of<'a>(
        &'a self,
        target: &'a Target,
    ) -> impl Iterator<Item = &'a Occurrence> {
        self.occurrences
            .iter()
            .filter(move |item| &item.target == target)
    }
}
