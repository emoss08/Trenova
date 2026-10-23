//! Coverage: which parts of the root file the tests and property cases of one `tariff.test` run
//! reached.
//!
//! The root file's **regions** are numbered once, in source order (an enclosing region before
//! the regions inside it), by [`Regions::of`]:
//!
//! | Kind | Region | Entered when |
//! |---|---|---|
//! | `if` | each branch block of an `if` statement (`else if` too), each branch value of an `if` expression | that branch runs |
//! | `else` | the `else` block or value | it runs |
//! | `arm` | each `match` arm, statement or expression | the arm is chosen |
//! | `loop` | a `for` body | each iteration runs it |
//! | `option` | each `option` of a `best` or `worst` | the option runs |
//! | `charge` | a `charge` statement | it runs |
//! | `refusal` | a `fail` statement, a `require` statement, a `?? fail` fallback | it refuses |
//! | `function` | a function body | the function is called |
//!
//! Tests and properties are not regions: they are what covers the rest. A library run on its
//! own reports its own regions; a library a tariff imports is not counted.
//!
//! Recording is cheap and opt-in: an interpreter carries [`Hits`] only under `tariff.test`
//! (`tariff.evaluate` never sets it, so a rating pays one `None` check per region entry). Each
//! entry looks its region up by span ([`Interp::cover`]) and bumps a dense counter indexed by
//! region id ([`Interp::enter_region`]).

use super::interp::{Env, Interp};
use crate::span::{Span, Utf16Index};
use crate::tariff::eval::trace::range;
use crate::tariff::syntax::ast::{
    ArmBody, Block, Condition, Expr, ExprKind, File, FnBody, ItemKind, RoundTo, Stmt, StmtKind,
    Text, TextPart, TotalClause,
};
use crate::tariff::syntax::walk;
use serde_json::{Map, Value as Json};
use std::collections::BTreeMap;
use std::rc::Rc;

/// What a region is.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RegionKind {
    /// A branch of an `if`.
    If,
    /// An `else`.
    Else,
    /// A `match` arm.
    Arm,
    /// A `for` body.
    Loop,
    /// An `option` of a choice.
    Option,
    /// A `charge`.
    Charge,
    /// A refusal: `fail`, `require`, `?? fail`.
    Refusal,
    /// A function body.
    Function,
}

impl RegionKind {
    /// The kind's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::If => "if",
            Self::Else => "else",
            Self::Arm => "arm",
            Self::Loop => "loop",
            Self::Option => "option",
            Self::Charge => "charge",
            Self::Refusal => "refusal",
            Self::Function => "function",
        }
    }
}

/// One region: where it is and what it is.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Region {
    /// The region's block or statement.
    pub span: Span,
    /// What it is.
    pub kind: RegionKind,
}

/// A file's regions, numbered in source order, with an index by span.
#[derive(Debug, Default)]
pub struct Regions {
    regions: Vec<Region>,
    by_span: BTreeMap<(u32, u32), usize>,
}

impl Regions {
    /// Numbers a file's regions.
    #[must_use]
    pub fn of(file: &File) -> Self {
        let mut found = Collector::default();
        for item in &file.items {
            match &item.kind {
                ItemKind::Fn(decl) => match &decl.body {
                    FnBody::Block(block) => {
                        found.push(block.span, RegionKind::Function);
                        found.block(block);
                    }
                    FnBody::Expr(expr) => {
                        found.push(expr.span, RegionKind::Function);
                        found.expr(expr);
                    }
                },
                ItemKind::Stmt(stmt) => found.stmt(stmt),
                ItemKind::Use(_)
                | ItemKind::Input(_)
                | ItemKind::Const(_)
                | ItemKind::Type(_)
                | ItemKind::Enum(_)
                | ItemKind::Test(_)
                | ItemKind::Property(_) => {}
            }
        }
        let mut regions = found.regions;
        regions.sort_by_key(|region| (region.span.start, std::cmp::Reverse(region.span.end)));
        let mut by_span = BTreeMap::new();
        let mut kept = Vec::with_capacity(regions.len());
        for region in regions {
            let key = (region.span.start, region.span.end);
            if by_span.contains_key(&key) {
                continue;
            }
            by_span.insert(key, kept.len());
            kept.push(region);
        }
        Self {
            regions: kept,
            by_span,
        }
    }

    /// The id of the region written at `span`.
    #[must_use]
    pub fn id(&self, span: Span) -> Option<usize> {
        self.by_span.get(&(span.start, span.end)).copied()
    }

    /// How many regions there are.
    #[must_use]
    pub fn len(&self) -> usize {
        self.regions.len()
    }

    /// Whether the file has no region.
    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.regions.is_empty()
    }

    /// The regions, in source order.
    #[must_use]
    pub fn regions(&self) -> &[Region] {
        &self.regions
    }
}

/// How many times each region was entered.
#[derive(Debug, Clone)]
pub struct Hits {
    regions: Rc<Regions>,
    counts: Vec<u32>,
}

impl Hits {
    /// No region entered yet.
    #[must_use]
    pub fn new(regions: Rc<Regions>) -> Self {
        let counts = vec![0; regions.len()];
        Self { regions, counts }
    }

    /// Counts one entry into region `id`.
    pub fn enter(&mut self, id: usize) {
        if let Some(count) = self.counts.get_mut(id) {
            *count = count.saturating_add(1);
        }
    }

    /// Adds another run's counts to these.
    pub fn merge(&mut self, other: &Self) {
        for (count, more) in self.counts.iter_mut().zip(&other.counts) {
            *count = count.saturating_add(*more);
        }
    }

    /// How many regions were entered at least once.
    #[must_use]
    pub fn reached(&self) -> usize {
        self.counts.iter().filter(|count| **count > 0).count()
    }

    /// `{"regions": [{"range", "kind", "hits"}], "reached", "total"}`.
    #[must_use]
    pub fn to_json(&self, index: &Utf16Index) -> Json {
        let regions: Vec<Json> = self
            .regions
            .regions()
            .iter()
            .zip(&self.counts)
            .map(|(region, hits)| {
                let mut out = Map::new();
                out.insert(String::from("range"), range(region.span, index));
                out.insert(
                    String::from("kind"),
                    Json::String(region.kind.as_str().to_owned()),
                );
                out.insert(String::from("hits"), Json::from(*hits));
                Json::Object(out)
            })
            .collect();
        let mut out = Map::new();
        out.insert(String::from("regions"), Json::Array(regions));
        out.insert(String::from("reached"), Json::from(self.reached()));
        out.insert(String::from("total"), Json::from(self.counts.len()));
        Json::Object(out)
    }
}

impl Interp<'_, '_> {
    /// Counts an entry into region `id`, when this run records coverage.
    pub fn enter_region(&mut self, id: usize) {
        if let Some(hits) = &mut self.coverage {
            hits.enter(id);
        }
    }

    /// Counts an entry into the region written at `span` in the root file, when this run
    /// records coverage and a region is written there. Code that may run in a library calls
    /// [`Interp::cover_in`] instead.
    pub fn cover(&mut self, span: Span) {
        let Some(hits) = &mut self.coverage else {
            return;
        };
        if let Some(id) = hits.regions.id(span) {
            hits.enter(id);
        }
    }

    /// [`Interp::cover`] for code running in `env`: only the root file's code counts.
    pub fn cover_in(&mut self, env: &Env, span: Span) {
        if env.module == 0 {
            self.cover(span);
        }
    }
}

/// Gathers regions from the tree.
#[derive(Default)]
struct Collector {
    regions: Vec<Region>,
}

impl Collector {
    fn push(&mut self, span: Span, kind: RegionKind) {
        self.regions.push(Region { span, kind });
    }

    fn block(&mut self, block: &Block) {
        for stmt in &block.stmts {
            self.stmt(stmt);
        }
    }

    fn stmt(&mut self, stmt: &Stmt) {
        match &stmt.kind {
            StmtKind::Let(binding) | StmtKind::Var(binding) => self.expr(&binding.value),
            StmtKind::Assign { value, .. }
            | StmtKind::Return(value)
            | StmtKind::Expr(value)
            | StmtKind::Total(
                TotalClause::Minimum(value)
                | TotalClause::Maximum(value)
                | TotalClause::Round {
                    to: RoundTo::Amount(value),
                    ..
                },
            ) => self.expr(value),
            StmtKind::Charge(charge) => {
                self.push(stmt.span, RegionKind::Charge);
                self.text(&charge.label);
                for option in &charge.options {
                    self.expr(&option.value);
                }
                if let Some(amount) = &charge.amount {
                    self.expr(amount);
                }
            }
            StmtKind::Note(text) => self.text(text),
            StmtKind::If(statement) => {
                for (condition, block) in &statement.branches {
                    self.condition(condition);
                    self.push(block.span, RegionKind::If);
                    self.block(block);
                }
                if let Some(otherwise) = &statement.otherwise {
                    self.push(otherwise.span, RegionKind::Else);
                    self.block(otherwise);
                }
            }
            StmtKind::Choose(choose) => {
                for option in &choose.options {
                    self.push(option.span, RegionKind::Option);
                    self.block(&option.body);
                }
            }
            StmtKind::Match(statement) => {
                self.expr(&statement.scrutinee);
                for arm in &statement.arms {
                    if let Some(guard) = &arm.guard {
                        self.expr(guard);
                    }
                    self.push(arm.span, RegionKind::Arm);
                    match &arm.body {
                        ArmBody::Block(block) => self.block(block),
                        ArmBody::Expr(expr) => self.expr(expr),
                    }
                }
            }
            StmtKind::For(looped) => {
                self.expr(&looped.iterable);
                if let Some(filter) = &looped.filter {
                    self.expr(filter);
                }
                self.push(looped.body.span, RegionKind::Loop);
                self.block(&looped.body);
            }
            StmtKind::Require { condition, message } => {
                self.push(stmt.span, RegionKind::Refusal);
                self.condition(condition);
                self.text(message);
            }
            StmtKind::Fail(text) => {
                self.push(stmt.span, RegionKind::Refusal);
                self.text(text);
            }
            StmtKind::Total(TotalClause::Round {
                to: RoundTo::Cents, ..
            })
            | StmtKind::Break
            | StmtKind::Continue
            | StmtKind::Given { .. }
            | StmtKind::Vary { .. }
            | StmtKind::Assume(_)
            | StmtKind::Expect(_)
            | StmtKind::Item(_) => {}
        }
    }

    fn condition(&mut self, condition: &Condition) {
        match condition {
            Condition::Expr(expr) | Condition::Let { value: expr, .. } => self.expr(expr),
        }
    }

    fn text(&mut self, text: &Text) {
        for part in &text.parts {
            if let TextPart::Expr(expr) = part {
                self.expr(expr);
            }
        }
    }

    fn expr(&mut self, expr: &Expr) {
        walk::expr(expr, &mut |node| match &node.kind {
            ExprKind::If {
                branches,
                otherwise,
            } => {
                for (_, value) in branches {
                    self.regions.push(Region {
                        span: value.span,
                        kind: RegionKind::If,
                    });
                }
                self.regions.push(Region {
                    span: otherwise.span,
                    kind: RegionKind::Else,
                });
            }
            ExprKind::Match { arms, .. } => {
                for arm in arms {
                    self.regions.push(Region {
                        span: arm.span,
                        kind: RegionKind::Arm,
                    });
                }
            }
            ExprKind::Fail(_) => self.regions.push(Region {
                span: node.span,
                kind: RegionKind::Refusal,
            }),
            _ => {}
        });
    }
}
