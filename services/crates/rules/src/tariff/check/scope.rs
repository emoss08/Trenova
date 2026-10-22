//! Scopes: the names a statement can see, the bindings it declares, and the optionals flow
//! narrowing has proven present.
//!
//! Every block is a frame. A function or test body starts at a barrier, below which nothing of
//! the enclosing file is visible (functions are closed). Each frame also knows the `let`s and
//! `var`s its block declares further down, so reading one early is "used before it is defined"
//! rather than "unknown".

use super::SymbolId;
use crate::span::Span;
use crate::tariff::types::Type;

/// What declared a local name.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum LocalKind {
    /// `let`.
    Let,
    /// `var`.
    Var,
    /// A function parameter.
    Param,
    /// A `for` loop variable.
    Loop,
    /// A lambda parameter.
    Lambda,
    /// `if let`, `require let` or a `match` pattern binding.
    Bound,
}

/// A name declared in a body.
#[derive(Debug, Clone)]
pub struct Local {
    /// The name.
    pub name: String,
    /// What declared it.
    pub kind: LocalKind,
    /// Its type.
    pub ty: Type,
    /// Where the name is written.
    pub span: Span,
    /// Its symbol.
    pub symbol: SymbolId,
    /// How many times it is read.
    pub reads: u32,
}

/// A path whose optional value flow has proven present: a root (`#local:3`, `#root:shipment`,
/// `#input:rate`) and the fields read from it.
pub type PathKey = Vec<String>;

/// One block's names.
#[derive(Debug, Default)]
struct Frame {
    names: Vec<usize>,
    narrowed: Vec<PathKey>,
    later: Vec<(String, Span)>,
    barrier: bool,
}

/// The stack of frames and every local declared so far.
#[derive(Debug, Default)]
pub struct Scopes {
    frames: Vec<Frame>,
    /// Every local, by id; ids stay valid after their frame is popped.
    pub locals: Vec<Local>,
}

impl Scopes {
    /// Opens a block; `later` are the names it declares further down.
    pub fn push(&mut self, later: Vec<(String, Span)>) {
        self.frames.push(Frame {
            later,
            ..Frame::default()
        });
    }

    /// Opens a function or test body: nothing declared outside it is visible inside.
    pub fn push_barrier(&mut self) {
        self.frames.push(Frame {
            barrier: true,
            ..Frame::default()
        });
    }

    /// Closes the innermost block, returning the locals it declared.
    pub fn pop(&mut self) -> Vec<usize> {
        self.frames
            .pop()
            .map(|frame| frame.names)
            .unwrap_or_default()
    }

    /// Declares a local in the innermost block and returns its id.
    pub fn declare(&mut self, local: Local) -> usize {
        let id = self.locals.len();
        if let Some(frame) = self.frames.last_mut() {
            if let Some(index) = frame.later.iter().position(|(name, _)| *name == local.name) {
                frame.later.remove(index);
            }
            frame.names.push(id);
        }
        self.locals.push(local);
        id
    }

    /// The visible local named `name`, innermost first.
    #[must_use]
    pub fn find(&self, name: &str) -> Option<usize> {
        for frame in self.frames.iter().rev() {
            if let Some(id) = frame.names.iter().rev().find(|id| {
                self.locals
                    .get(**id)
                    .is_some_and(|local| local.name == name)
            }) {
                return Some(*id);
            }
            if frame.barrier {
                break;
            }
        }
        None
    }

    /// Where a visible block declares `name` further down, when one does.
    #[must_use]
    pub fn declared_later(&self, name: &str) -> Option<Span> {
        for frame in self.frames.iter().rev() {
            if let Some((_, span)) = frame.later.iter().find(|(later, _)| later == name) {
                return Some(*span);
            }
            if frame.barrier {
                break;
            }
        }
        None
    }

    /// Every visible local's name, for "did you mean".
    #[must_use]
    pub fn visible(&self) -> Vec<String> {
        let mut out = Vec::new();
        for frame in self.frames.iter().rev() {
            for id in &frame.names {
                if let Some(local) = self.locals.get(*id) {
                    out.push(local.name.clone());
                }
            }
            if frame.barrier {
                break;
            }
        }
        out
    }

    /// Records that a path is present for the rest of the innermost block.
    pub fn narrow(&mut self, path: PathKey) {
        if let Some(frame) = self.frames.last_mut() {
            frame.narrowed.push(path);
        }
    }

    /// Whether flow has proven a path present.
    #[must_use]
    pub fn is_narrowed(&self, path: &[String]) -> bool {
        for frame in self.frames.iter().rev() {
            if frame.narrowed.iter().any(|known| known == path) {
                return true;
            }
            if frame.barrier {
                break;
            }
        }
        false
    }

    /// Counts a read of a local.
    pub fn read(&mut self, id: usize) {
        if let Some(local) = self.locals.get_mut(id) {
            local.reads = local.reads.saturating_add(1);
        }
    }

    /// The local with an id.
    #[must_use]
    pub fn get(&self, id: usize) -> Option<&Local> {
        self.locals.get(id)
    }
}
