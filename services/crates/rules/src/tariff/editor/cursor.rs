//! Where a position is in a file that may not parse: the token at it, the syntax nodes around
//! it, the kind of body it is in, and the local names visible there.
//!
//! The file being typed is usually broken, so nothing here assumes a complete tree:
//!
//! - **Tokens** come first. [`word_at`] is the identifier being typed (the token that ends at or
//!   contains the position), [`before`] the significant token before a position, newlines
//!   included, so a feature can tell `shipment.|` from `let x = |` from a statement start.
//! - **Nodes**: [`locate`] descends from the file's items to the innermost node whose text holds
//!   the position. A statement is taken to run to the end of its line, so a half-written `let x
//!   = |` still holds a cursor after the `=`, and a block the author has not closed runs to the
//!   end of the file, as the parser read it.
//! - **Scopes**: [`Scopes::of`] walks the file once and records every local name (a `let`, a
//!   `var`, a parameter, a loop variable, a lambda parameter, an `if let`, `require let` or
//!   pattern binding) with the extent of text where it is visible. Which names a body sees
//!   besides its locals (the record, inputs, the file's bindings) follows the body's kind
//!   ([`Body`]), the same rules the checker applies.

use crate::span::Span;
use crate::tariff::syntax::ast::{
    Arg, Arm, ArmBody, Block, Condition, Expect, Expr, ExprKind, File, FnBody, Item, ItemKind,
    Name, Param, Pattern, PatternKind, RoundTo, Stmt, StmtKind, Text, TextPart, TotalClause,
    TypeExpr, TypeExprKind,
};
use crate::tariff::syntax::token::{Token, TokenKind};

/// What kind of body a position is in, which decides what it can name.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Body {
    /// A file's top level and the blocks inside it: everything is visible.
    Top,
    /// A function body: its parameters, constants, functions, types and imports, nothing of the
    /// record.
    Function,
    /// A `test` block: the record, the bill, constants and functions, not the file's inputs or
    /// bindings.
    Test,
    /// A constant's value, an input's default or a parameter's default: compile-time values.
    Constant,
    /// A `use`, `type` or `enum` declaration, or a type annotation: no values at all.
    Declaration,
}

/// One syntax node on the way from the file to a position.
#[derive(Debug, Clone, Copy)]
pub enum Node<'a> {
    /// A file-level item.
    Item(&'a Item),
    /// A statement.
    Stmt(&'a Stmt),
    /// A block.
    Block(&'a Block),
    /// An expression.
    Expr(&'a Expr),
    /// A call argument or a charge option.
    Arg(&'a Arg),
    /// A `match` arm.
    Arm(&'a Arm),
    /// A pattern.
    Pattern(&'a Pattern),
    /// A written type.
    Type(&'a TypeExpr),
    /// A function parameter.
    Param(&'a Param),
}

/// Where a position is: the nodes holding it, outermost first, and the body they are in.
#[derive(Debug, Clone)]
pub struct Located<'a> {
    /// The nodes holding the position, outermost first.
    pub chain: Vec<Node<'a>>,
    /// The body the position is in.
    pub body: Body,
    /// How many nodes the search visited, for the work meter.
    pub visited: usize,
}

impl<'a> Located<'a> {
    /// The innermost expression holding the position.
    #[must_use]
    pub fn expr(&self) -> Option<&'a Expr> {
        self.chain.iter().rev().find_map(|node| match node {
            Node::Expr(expr) => Some(*expr),
            _ => None,
        })
    }

    /// The expressions holding the position, innermost first.
    pub fn exprs(&self) -> impl Iterator<Item = &'a Expr> + '_ {
        self.chain.iter().rev().filter_map(|node| match node {
            Node::Expr(expr) => Some(*expr),
            _ => None,
        })
    }

    /// The innermost statement holding the position.
    #[must_use]
    pub fn stmt(&self) -> Option<&'a Stmt> {
        self.chain.iter().rev().find_map(|node| match node {
            Node::Stmt(stmt) => Some(*stmt),
            _ => None,
        })
    }

    /// The file-level item holding the position.
    #[must_use]
    pub fn item(&self) -> Option<&'a Item> {
        self.chain.iter().find_map(|node| match node {
            Node::Item(item) => Some(*item),
            _ => None,
        })
    }

    /// The innermost written type holding the position.
    #[must_use]
    pub fn ty(&self) -> Option<&'a TypeExpr> {
        self.chain.iter().rev().find_map(|node| match node {
            Node::Type(ty) => Some(*ty),
            _ => None,
        })
    }

    /// The innermost pattern holding the position.
    #[must_use]
    pub fn pattern(&self) -> Option<&'a Pattern> {
        self.chain.iter().rev().find_map(|node| match node {
            Node::Pattern(pattern) => Some(*pattern),
            _ => None,
        })
    }

    /// How many `for` loops hold the position, inside the innermost function or test body.
    #[must_use]
    pub fn loops(&self) -> usize {
        self.chain
            .iter()
            .filter(|node| {
                matches!(
                    node,
                    Node::Stmt(Stmt {
                        kind: StmtKind::For(_),
                        ..
                    })
                )
            })
            .count()
    }

    /// Whether the position is directly in a file-level statement list (not inside a block).
    #[must_use]
    pub fn at_file_level(&self) -> bool {
        !self.chain.iter().any(|node| matches!(node, Node::Block(_)))
    }
}

/// Finds the nodes holding byte `offset` of `file`.
#[must_use]
pub fn locate<'a>(source: &str, file: &'a File, offset: u32) -> Located<'a> {
    let mut locator = Locator {
        source,
        offset,
        chain: Vec::new(),
        body: Body::Top,
        visited: 0,
    };
    for item in &file.items {
        locator.visited += 1;
        let span = Span::new(item.span.start, line_end(source, item.span.end));
        if holds(span, offset) && locator.item(item) {
            break;
        }
    }
    Located {
        chain: locator.chain,
        body: locator.body,
        visited: locator.visited,
    }
}

/// Whether `span` holds `offset`, either end included.
#[must_use]
pub const fn holds(span: Span, offset: u32) -> bool {
    span.start <= offset && offset <= span.end
}

/// Where a line ends after `end`: past spaces and a trailing comment to the newline, or `end`
/// when something else follows on the line.
#[must_use]
pub fn line_end(source: &str, end: u32) -> u32 {
    let start = usize::try_from(end).unwrap_or(usize::MAX);
    let Some(rest) = source.get(start..) else {
        return end;
    };
    let trimmed = rest.trim_start_matches([' ', '\t']);
    let skipped = rest.len() - trimmed.len();
    if trimmed.is_empty() || trimmed.starts_with('\n') || trimmed.starts_with("\r\n") {
        return end.saturating_add(u32::try_from(skipped).unwrap_or(0));
    }
    if trimmed.starts_with("//") {
        let line = trimmed.find('\n').unwrap_or(trimmed.len());
        return end.saturating_add(u32::try_from(skipped + line).unwrap_or(0));
    }
    end
}

/// `end` moved past the spaces and tabs after it.
#[must_use]
pub fn past_spaces(source: &str, end: u32) -> u32 {
    let rest = source.get(end as usize..).unwrap_or_default();
    let spaces = rest.len() - rest.trim_start_matches([' ', '\t']).len();
    end.saturating_add(u32::try_from(spaces).unwrap_or(0))
}

/// A block's extent: to its `}`, or to the end of the file when it was never closed.
#[must_use]
pub fn block_end(source: &str, block: &Block) -> u32 {
    if block_closed(source, block) {
        block.span.end
    } else {
        u32::try_from(source.len()).unwrap_or(u32::MAX)
    }
}

/// Whether a block ends with its `}`.
#[must_use]
pub fn block_closed(source: &str, block: &Block) -> bool {
    block
        .span
        .end
        .checked_sub(1)
        .and_then(|last| source.get(last as usize..block.span.end as usize))
        == Some("}")
}

struct Locator<'a, 's> {
    source: &'s str,
    offset: u32,
    chain: Vec<Node<'a>>,
    body: Body,
    visited: usize,
}

impl<'a> Locator<'a, '_> {
    fn holds(&self, span: Span) -> bool {
        holds(span, self.offset)
    }

    fn inside_block(&self, block: &Block) -> bool {
        if block.span.start >= self.offset {
            return false;
        }
        if block_closed(self.source, block) {
            self.offset < block.span.end
        } else {
            self.offset <= block_end(self.source, block)
        }
    }

    /// Descends into an item; true when it holds the position.
    fn item(&mut self, item: &'a Item) -> bool {
        self.chain.push(Node::Item(item));
        match &item.kind {
            ItemKind::Use(_) | ItemKind::Type(_) | ItemKind::Enum(_) => {
                self.body = Body::Declaration;
                if let ItemKind::Type(decl) = &item.kind {
                    for field in &decl.fields {
                        self.ty(&field.ty);
                    }
                }
            }
            ItemKind::Input(input) => {
                self.body = Body::Constant;
                self.ty(&input.ty);
                if let Some(default) = &input.default {
                    self.expr(default);
                }
            }
            ItemKind::Const(constant) => {
                self.body = Body::Constant;
                if let Some(ty) = &constant.ty {
                    self.ty(ty);
                }
                self.expr(&constant.value);
            }
            ItemKind::Fn(function) => self.function(function),
            ItemKind::Test(test) => {
                self.body = Body::Test;
                self.block(&test.body);
            }
            ItemKind::Property(property) => {
                self.body = Body::Test;
                self.block(&property.body);
            }
            ItemKind::Stmt(stmt) => {
                self.body = Body::Top;
                self.stmt(stmt);
            }
        }
        true
    }

    fn function(&mut self, function: &'a crate::tariff::syntax::ast::FnDecl) {
        self.body = Body::Function;
        for param in &function.params {
            if self.holds(param.span) {
                self.chain.push(Node::Param(param));
                self.body = Body::Declaration;
                self.ty(&param.ty);
                if let Some(default) = &param.default
                    && self.holds(default.span)
                {
                    self.body = Body::Constant;
                    self.expr(default);
                }
                return;
            }
        }
        if let Some(ret) = &function.ret
            && self.holds(ret.span)
        {
            self.body = Body::Declaration;
            self.ty(ret);
            return;
        }
        match &function.body {
            FnBody::Block(block) => {
                self.block(block);
            }
            FnBody::Expr(expr) => {
                self.expr(expr);
            }
        }
    }

    fn block(&mut self, block: &'a Block) -> bool {
        self.visited += 1;
        if !self.inside_block(block) {
            return false;
        }
        self.chain.push(Node::Block(block));
        for stmt in &block.stmts {
            self.visited += 1;
            let span = Span::new(stmt.span.start, line_end(self.source, stmt.span.end));
            if self.holds(span) {
                self.stmt(stmt);
                break;
            }
        }
        true
    }

    fn stmt(&mut self, stmt: &'a Stmt) {
        self.chain.push(Node::Stmt(stmt));
        match &stmt.kind {
            StmtKind::Let(binding) | StmtKind::Var(binding) => {
                if let Some(ty) = &binding.ty {
                    self.ty(ty);
                }
                self.expr(&binding.value);
            }
            StmtKind::Assign { value, .. }
            | StmtKind::Return(value)
            | StmtKind::Expr(value)
            | StmtKind::Assume(value)
            | StmtKind::Given { value, .. } => {
                self.expr(value);
            }
            StmtKind::Charge(charge) => {
                self.text(&charge.label);
                for option in &charge.options {
                    self.arg(option);
                }
                if let Some(amount) = &charge.amount {
                    self.expr(amount);
                }
            }
            StmtKind::Note(text) | StmtKind::Fail(text) => self.text(text),
            StmtKind::If(statement) => {
                for (condition, block) in &statement.branches {
                    self.condition(condition);
                    self.block(block);
                }
                if let Some(otherwise) = &statement.otherwise {
                    self.block(otherwise);
                }
            }
            StmtKind::Match(statement) => {
                self.expr(&statement.scrutinee);
                self.arms(&statement.arms);
            }
            StmtKind::For(statement) => {
                self.expr(&statement.iterable);
                if let Some(filter) = &statement.filter {
                    self.expr(filter);
                }
                self.block(&statement.body);
            }
            StmtKind::Choose(choose) => {
                for option in &choose.options {
                    self.block(&option.body);
                }
            }
            StmtKind::Vary { values, .. } => {
                if let Some(values) = values {
                    self.expr(values);
                }
            }
            StmtKind::Require { condition, message } => {
                self.condition(condition);
                self.text(message);
            }
            StmtKind::Total(clause) => match clause {
                TotalClause::Minimum(value) | TotalClause::Maximum(value) => {
                    self.expr(value);
                }
                TotalClause::Round { to, .. } => {
                    if let RoundTo::Amount(value) = to {
                        self.expr(value);
                    }
                }
            },
            StmtKind::Expect(expect) => match expect {
                Expect::Holds(value) | Expect::Monotone { value, .. } => {
                    self.expr(value);
                }
                Expect::Fails(Some(text)) => self.text(text),
                Expect::Fails(None) => {}
            },
            StmtKind::Item(item) => {
                self.item(item);
            }
            StmtKind::Break | StmtKind::Continue => {}
        }
    }

    fn condition(&mut self, condition: &'a Condition) {
        match condition {
            Condition::Expr(value) | Condition::Let { value, .. } => {
                self.expr(value);
            }
        }
    }

    fn text(&mut self, text: &'a Text) {
        if !self.holds(text.span) {
            return;
        }
        for part in &text.parts {
            if let TextPart::Expr(expr) = part {
                self.expr(expr);
            }
        }
    }

    fn arg(&mut self, arg: &'a Arg) -> bool {
        self.visited += 1;
        if !self.holds(arg.span) {
            return false;
        }
        self.chain.push(Node::Arg(arg));
        self.expr(&arg.value);
        true
    }

    fn arms(&mut self, arms: &'a [Arm]) {
        for arm in arms {
            self.visited += 1;
            let span = Span::new(arm.span.start, line_end(self.source, arm.span.end));
            if !self.holds(span) {
                continue;
            }
            self.chain.push(Node::Arm(arm));
            self.pattern(&arm.pattern);
            if let Some(guard) = &arm.guard {
                self.expr(guard);
            }
            match &arm.body {
                ArmBody::Block(block) => {
                    self.block(block);
                }
                ArmBody::Expr(expr) => {
                    self.expr(expr);
                }
            }
            return;
        }
    }

    fn pattern(&mut self, pattern: &'a Pattern) {
        self.visited += 1;
        if !self.holds(pattern.span) {
            return;
        }
        self.chain.push(Node::Pattern(pattern));
        match &pattern.kind {
            PatternKind::Literal(expr) => {
                self.expr(expr);
            }
            PatternKind::Range { start, end, .. } => {
                for part in [start, end].into_iter().flatten() {
                    self.expr(part);
                }
            }
            PatternKind::Record(fields) => {
                for (_, inner) in fields {
                    self.pattern(inner);
                }
            }
            PatternKind::Or(alternatives) => {
                for inner in alternatives {
                    self.pattern(inner);
                }
            }
            PatternKind::Wildcard
            | PatternKind::None
            | PatternKind::Variant { .. }
            | PatternKind::Binding(_)
            | PatternKind::Missing => {}
        }
    }

    fn ty(&mut self, ty: &'a TypeExpr) {
        self.visited += 1;
        if !self.holds(ty.span) {
            return;
        }
        self.chain.push(Node::Type(ty));
        match &ty.kind {
            TypeExprKind::Named { args, .. } => {
                for arg in args {
                    self.ty(arg);
                }
            }
            TypeExprKind::Record(fields) => {
                for field in fields {
                    self.ty(&field.ty);
                }
            }
            TypeExprKind::Optional(inner) => self.ty(inner),
            TypeExprKind::Quotient { .. } | TypeExprKind::Missing => {}
        }
    }

    fn expr(&mut self, expr: &'a Expr) -> bool {
        self.visited += 1;
        if !self.holds(expr.span) {
            return false;
        }
        self.chain.push(Node::Expr(expr));
        match &expr.kind {
            ExprKind::Text(text) | ExprKind::Fail(text) => self.text(text),
            ExprKind::Field { target, .. } => {
                self.expr(target);
            }
            ExprKind::Call { callee, args } => {
                if !self.expr(callee) {
                    for arg in args {
                        if self.arg(arg) {
                            break;
                        }
                    }
                }
            }
            ExprKind::Index { target, keys } => {
                if !self.expr(target) {
                    for key in keys {
                        if self.expr(key) {
                            break;
                        }
                    }
                }
            }
            ExprKind::Unary { operand, .. } => {
                self.expr(operand);
            }
            ExprKind::Binary { left, right, .. } => {
                if !self.expr(left) {
                    self.expr(right);
                }
            }
            ExprKind::Range { start, end, .. } => {
                for part in [start, end].into_iter().flatten() {
                    if self.expr(part) {
                        break;
                    }
                }
            }
            ExprKind::List(items) => {
                for item in items {
                    if self.expr(item) {
                        break;
                    }
                }
            }
            ExprKind::Record(fields) => {
                for field in fields {
                    if self.expr(&field.value) {
                        break;
                    }
                }
            }
            ExprKind::If {
                branches,
                otherwise,
            } => {
                for (condition, value) in branches {
                    self.condition(condition);
                    self.expr(value);
                }
                self.expr(otherwise);
            }
            ExprKind::Match { scrutinee, arms } => {
                self.expr(scrutinee);
                self.arms(arms);
            }
            ExprKind::Lambda { body, .. } => {
                self.expr(body);
            }
            ExprKind::Number { .. }
            | ExprKind::Money { .. }
            | ExprKind::Percent(_)
            | ExprKind::Date(_)
            | ExprKind::DateTime(_)
            | ExprKind::Bool(_)
            | ExprKind::None
            | ExprKind::Name(_)
            | ExprKind::Path { .. }
            | ExprKind::Missing => {}
        }
        true
    }
}

/// Where a local name is visible.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Extent {
    /// Inside this text.
    Span(Span),
    /// In the file's top-level statements from this offset on: a file-level `let`, `var` or
    /// `require let`, which functions and tests cannot see.
    TopLevel {
        /// Where it starts being visible.
        from: u32,
    },
}

/// A local name and where it is visible.
#[derive(Debug, Clone)]
pub struct Scoped {
    /// The name as declared.
    pub name: String,
    /// Where it is declared.
    pub span: Span,
    /// Where it is visible.
    pub extent: Extent,
}

impl Scoped {
    /// Whether the name is visible at `offset` in a body of kind `body`.
    #[must_use]
    pub fn visible(&self, offset: u32, body: Body) -> bool {
        match self.extent {
            Extent::Span(span) => holds(span, offset) && offset >= self.span.end,
            Extent::TopLevel { from } => body == Body::Top && offset >= from,
        }
    }

    /// Whether the extent holds `span`.
    #[must_use]
    pub fn covers(&self, span: Span) -> bool {
        match self.extent {
            Extent::Span(extent) => extent.start <= span.start && span.end <= extent.end,
            Extent::TopLevel { from } => span.start >= from,
        }
    }
}

/// Every local name of a file with where it is visible, and every expression of the file.
#[derive(Debug, Clone, Default)]
pub struct Scopes<'a> {
    /// The names, in the order the walk met them.
    pub locals: Vec<Scoped>,
    /// Every expression, each before its children, in source order.
    pub exprs: Vec<&'a Expr>,
    /// Every name written in a type annotation (`Break` in `List<Break>`), in source order.
    pub types: Vec<&'a Name>,
}

impl<'a> Scopes<'a> {
    /// Walks a file once and records every local name and every expression.
    #[must_use]
    pub fn of(source: &str, file: &'a File) -> Self {
        let mut walker = ScopeWalker {
            source,
            locals: Vec::new(),
            exprs: Vec::new(),
            types: Vec::new(),
        };
        for item in &file.items {
            walker.item(item, true);
        }
        Self {
            locals: walker.locals,
            exprs: walker.exprs,
            types: walker.types,
        }
    }

    /// The first expression, in source order, that `test` holds for.
    pub fn find(&self, test: impl Fn(&Expr) -> bool) -> Option<&'a Expr> {
        self.exprs.iter().copied().find(|expr| test(expr))
    }

    /// The last expression, in source order (so the innermost of nested ones), that `test`
    /// holds for.
    pub fn find_last(&self, test: impl Fn(&Expr) -> bool) -> Option<&'a Expr> {
        self.exprs.iter().rev().copied().find(|expr| test(expr))
    }

    /// The names visible at `offset` in a body of kind `body`, the innermost (latest declared)
    /// one for each name.
    #[must_use]
    pub fn visible(&self, offset: u32, body: Body) -> Vec<&Scoped> {
        let mut found: Vec<&Scoped> = Vec::new();
        for local in &self.locals {
            if !local.visible(offset, body) {
                continue;
            }
            match found.iter_mut().find(|known| known.name == local.name) {
                Some(known) if known.span.start < local.span.start => *known = local,
                Some(_) => {}
                None => found.push(local),
            }
        }
        found
    }

    /// The local declared with its name at `span`.
    #[must_use]
    pub fn declared_at(&self, span: Span) -> Option<&Scoped> {
        self.locals.iter().find(|local| local.span == span)
    }
}

struct ScopeWalker<'a, 's> {
    source: &'s str,
    locals: Vec<Scoped>,
    exprs: Vec<&'a Expr>,
    types: Vec<&'a Name>,
}

impl<'a> ScopeWalker<'a, '_> {
    fn add(&mut self, name: &Name, extent: Extent) {
        if !name.text.is_empty() {
            self.locals.push(Scoped {
                name: name.text.clone(),
                span: name.span,
                extent,
            });
        }
    }

    fn ty(&mut self, ty: &'a TypeExpr) {
        match &ty.kind {
            TypeExprKind::Named { name, args } => {
                self.types.push(name);
                for arg in args {
                    self.ty(arg);
                }
            }
            TypeExprKind::Record(fields) => {
                for field in fields {
                    self.ty(&field.ty);
                }
            }
            TypeExprKind::Optional(inner) => self.ty(inner),
            TypeExprKind::Quotient { .. } | TypeExprKind::Missing => {}
        }
    }

    fn item(&mut self, item: &'a Item, top: bool) {
        match &item.kind {
            ItemKind::Use(_) | ItemKind::Enum(_) => {}
            ItemKind::Type(decl) => {
                for field in &decl.fields {
                    self.ty(&field.ty);
                }
            }
            ItemKind::Input(input) => {
                self.ty(&input.ty);
                if let Some(default) = &input.default {
                    self.expr(default);
                }
            }
            ItemKind::Const(constant) => {
                if let Some(ty) = &constant.ty {
                    self.ty(ty);
                }
                self.expr(&constant.value);
            }
            ItemKind::Fn(function) => {
                let after_params = function
                    .ret
                    .as_ref()
                    .map(|ret| ret.span.end)
                    .or_else(|| function.params.last().map(|param| param.span.end))
                    .unwrap_or(function.name.span.end);
                let body = match &function.body {
                    FnBody::Block(block) => {
                        Span::new(block.span.start, block_end(self.source, block))
                    }
                    FnBody::Expr(expr) => Span::new(
                        after_params,
                        line_end(self.source, past_spaces(self.source, expr.span.end)),
                    ),
                };
                for param in &function.params {
                    self.add(&param.name, Extent::Span(body));
                    self.ty(&param.ty);
                    if let Some(default) = &param.default {
                        self.expr(default);
                    }
                }
                if let Some(ret) = &function.ret {
                    self.ty(ret);
                }
                match &function.body {
                    FnBody::Block(block) => self.block(block),
                    FnBody::Expr(expr) => self.expr(expr),
                }
            }
            ItemKind::Test(test) => self.block(&test.body),
            ItemKind::Property(property) => self.block(&property.body),
            ItemKind::Stmt(stmt) => {
                if top {
                    let from = Extent::TopLevel {
                        from: self.after_line(stmt.span.end),
                    };
                    self.binder(stmt, from);
                }
                self.stmt(stmt);
            }
        }
    }

    /// The name a statement declares for the statements after it.
    fn binder(&mut self, stmt: &'a Stmt, extent: Extent) {
        match &stmt.kind {
            StmtKind::Let(binding) | StmtKind::Var(binding) => self.add(&binding.name, extent),
            StmtKind::Require {
                condition: Condition::Let { name, .. },
                ..
            } => self.add(name, extent),
            _ => {}
        }
    }

    fn block(&mut self, block: &'a Block) {
        let end = block_end(self.source, block);
        for stmt in &block.stmts {
            let from = self.after_line(stmt.span.end).min(end);
            self.binder(stmt, Extent::Span(Span::new(from, end)));
            self.stmt(stmt);
        }
    }

    /// Where the statement ending at `end` stops hiding its own name: past the end of its
    /// line, so `let miles = |` does not see `miles`.
    fn after_line(&self, end: u32) -> u32 {
        let line = line_end(self.source, end);
        if line == end {
            end
        } else {
            line.saturating_add(1)
        }
    }

    fn block_span(&self, block: &'a Block) -> Span {
        Span::new(block.span.start, block_end(self.source, block))
    }

    fn stmt(&mut self, stmt: &'a Stmt) {
        match &stmt.kind {
            StmtKind::Let(binding) | StmtKind::Var(binding) => {
                if let Some(ty) = &binding.ty {
                    self.ty(ty);
                }
                self.expr(&binding.value);
            }
            StmtKind::Assign { value, .. }
            | StmtKind::Return(value)
            | StmtKind::Expr(value)
            | StmtKind::Given { value, .. }
            | StmtKind::Total(
                TotalClause::Minimum(value)
                | TotalClause::Maximum(value)
                | TotalClause::Round {
                    to: RoundTo::Amount(value),
                    ..
                },
            )
            | StmtKind::Expect(Expect::Holds(value) | Expect::Monotone { value, .. })
            | StmtKind::Assume(value) => {
                self.expr(value);
            }
            StmtKind::Charge(charge) => {
                self.text(&charge.label);
                for option in &charge.options {
                    self.expr(&option.value);
                }
                if let Some(amount) = &charge.amount {
                    self.expr(amount);
                }
            }
            StmtKind::Note(text) | StmtKind::Fail(text) => self.text(text),
            StmtKind::If(statement) => {
                for (condition, block) in &statement.branches {
                    let extent = Extent::Span(self.block_span(block));
                    self.condition(condition, extent);
                    self.block(block);
                }
                if let Some(otherwise) = &statement.otherwise {
                    self.block(otherwise);
                }
            }
            StmtKind::Match(statement) => {
                self.expr(&statement.scrutinee);
                self.arms(&statement.arms);
            }
            StmtKind::Choose(choose) => {
                for option in &choose.options {
                    self.block(&option.body);
                }
            }
            StmtKind::Vary { values, .. } => {
                if let Some(values) = values {
                    self.expr(values);
                }
            }
            StmtKind::For(statement) => {
                self.expr(&statement.iterable);
                let start = statement
                    .filter
                    .as_ref()
                    .map_or(statement.body.span.start, |filter| filter.span.start);
                let extent = Span::new(start, block_end(self.source, &statement.body));
                self.add(&statement.binding, Extent::Span(extent));
                if let Some(filter) = &statement.filter {
                    self.expr(filter);
                }
                self.block(&statement.body);
            }
            StmtKind::Require { condition, message } => {
                match condition {
                    Condition::Expr(value) | Condition::Let { value, .. } => self.expr(value),
                }
                self.text(message);
            }
            StmtKind::Expect(Expect::Fails(text)) => {
                if let Some(text) = text {
                    self.text(text);
                }
            }
            StmtKind::Item(item) => self.item(item, false),
            StmtKind::Total(TotalClause::Round {
                to: RoundTo::Cents, ..
            })
            | StmtKind::Break
            | StmtKind::Continue => {}
        }
    }

    fn condition(&mut self, condition: &'a Condition, extent: Extent) {
        match condition {
            Condition::Expr(value) => self.expr(value),
            Condition::Let { name, value } => {
                self.expr(value);
                self.add(name, extent);
            }
        }
    }

    fn text(&mut self, text: &'a Text) {
        for part in &text.parts {
            if let TextPart::Expr(expr) = part {
                self.expr(expr);
            }
        }
    }

    fn arms(&mut self, arms: &'a [Arm]) {
        for arm in arms {
            let end = match &arm.body {
                ArmBody::Block(block) => block_end(self.source, block),
                ArmBody::Expr(expr) => expr.span.end,
            };
            let extent = Extent::Span(Span::new(
                arm.pattern.span.end,
                line_end(self.source, past_spaces(self.source, end.max(arm.span.end))),
            ));
            self.pattern(&arm.pattern, extent);
            if let Some(guard) = &arm.guard {
                self.expr(guard);
            }
            match &arm.body {
                ArmBody::Block(block) => self.block(block),
                ArmBody::Expr(expr) => self.expr(expr),
            }
        }
    }

    fn pattern(&mut self, pattern: &'a Pattern, extent: Extent) {
        match &pattern.kind {
            PatternKind::Binding(name) => self.add(name, extent),
            PatternKind::Literal(expr) => self.expr(expr),
            PatternKind::Range { start, end, .. } => {
                for part in [start, end].into_iter().flatten() {
                    self.expr(part);
                }
            }
            PatternKind::Record(fields) => {
                for (_, inner) in fields {
                    self.pattern(inner, extent);
                }
            }
            PatternKind::Or(alternatives) => {
                for inner in alternatives {
                    self.pattern(inner, extent);
                }
            }
            PatternKind::Wildcard
            | PatternKind::None
            | PatternKind::Variant { .. }
            | PatternKind::Missing => {}
        }
    }

    fn expr(&mut self, expr: &'a Expr) {
        self.exprs.push(expr);
        match &expr.kind {
            ExprKind::Text(text) | ExprKind::Fail(text) => self.text(text),
            ExprKind::Field { target, .. } => self.expr(target),
            ExprKind::Call { callee, args } => {
                self.expr(callee);
                for arg in args {
                    self.expr(&arg.value);
                }
            }
            ExprKind::Index { target, keys } => {
                self.expr(target);
                for key in keys {
                    self.expr(key);
                }
            }
            ExprKind::Unary { operand, .. } => self.expr(operand),
            ExprKind::Binary { left, right, .. } => {
                self.expr(left);
                self.expr(right);
            }
            ExprKind::Range { start, end, .. } => {
                for part in [start, end].into_iter().flatten() {
                    self.expr(part);
                }
            }
            ExprKind::List(items) => {
                for item in items {
                    self.expr(item);
                }
            }
            ExprKind::Record(fields) => {
                for field in fields {
                    self.expr(&field.value);
                }
            }
            ExprKind::If {
                branches,
                otherwise,
            } => {
                for (condition, value) in branches {
                    self.condition(condition, Extent::Span(value.span));
                    self.expr(value);
                }
                self.expr(otherwise);
            }
            ExprKind::Match { scrutinee, arms } => {
                self.expr(scrutinee);
                self.arms(arms);
            }
            ExprKind::Lambda { params, body } => {
                let start = params
                    .last()
                    .map_or(body.span.start, |param| param.span.end);
                let end = past_spaces(self.source, expr.span.end.max(body.span.end));
                for param in params {
                    self.add(param, Extent::Span(Span::new(start, end)));
                }
                self.expr(body);
            }
            ExprKind::Number { .. }
            | ExprKind::Money { .. }
            | ExprKind::Percent(_)
            | ExprKind::Date(_)
            | ExprKind::DateTime(_)
            | ExprKind::Bool(_)
            | ExprKind::None
            | ExprKind::Name(_)
            | ExprKind::Path { .. }
            | ExprKind::Missing => {}
        }
    }
}

/// Whether a token is a word: a name, type name, code, keyword or unit.
#[must_use]
pub const fn is_word(kind: TokenKind) -> bool {
    matches!(
        kind,
        TokenKind::Name
            | TokenKind::TypeName
            | TokenKind::Code
            | TokenKind::Keyword
            | TokenKind::Unit
    )
}

/// The index of the word being typed at `offset`: a word token that ends at it or holds it.
#[must_use]
pub fn word_at(tokens: &[Token], offset: u32) -> Option<usize> {
    let index = tokens.partition_point(|token| token.span.end < offset);
    let token = tokens.get(index)?;
    (is_word(token.kind) && token.span.start < offset && offset <= token.span.end).then_some(index)
}

/// The index of the last token that ends at or before `offset`, skipping comments; newlines
/// count.
#[must_use]
pub fn before(tokens: &[Token], offset: u32) -> Option<usize> {
    let end = tokens.partition_point(|token| token.span.end <= offset);
    (0..end).rev().find(|index| {
        tokens.get(*index).is_some_and(|token| {
            !matches!(token.kind, TokenKind::Comment | TokenKind::DocComment)
                && token.span.start < token.span.end
        })
    })
}

/// The significant token before token `index`, skipping comments; newlines count.
#[must_use]
pub fn previous(tokens: &[Token], index: usize) -> Option<usize> {
    (0..index).rev().find(|candidate| {
        tokens.get(*candidate).is_some_and(|token| {
            !matches!(token.kind, TokenKind::Comment | TokenKind::DocComment)
                && token.span.start < token.span.end
        })
    })
}

/// Whether `offset` is inside a comment or the literal part of a text, where nothing completes.
#[must_use]
pub fn in_literal(source: &str, tokens: &[Token], offset: u32) -> bool {
    let index = tokens.partition_point(|token| token.span.end < offset);
    let Some(token) = tokens.get(index) else {
        return false;
    };
    match token.kind {
        TokenKind::Comment | TokenKind::DocComment => token.span.start < offset,
        TokenKind::Text | TokenKind::Escape => {
            let closed = source
                .get(token.span.start as usize..token.span.end as usize)
                .is_some_and(|text| text.len() > 1 && text.ends_with('"'));
            token.span.start < offset && (offset < token.span.end || !closed)
        }
        _ => false,
    }
}

/// The token at `offset`: the one holding it, else the word or punctuation ending at it.
#[must_use]
pub fn token_at(tokens: &[Token], offset: u32) -> Option<usize> {
    let index = tokens.partition_point(|token| token.span.end <= offset);
    if tokens
        .get(index)
        .is_some_and(|token| token.span.start <= offset && token.span.start < token.span.end)
    {
        return Some(index);
    }
    let previous = index.checked_sub(1)?;
    tokens
        .get(previous)
        .filter(|token| token.span.end == offset && token.kind != TokenKind::Newline)
        .map(|_| previous)
}

/// The tokens of the line holding token `index`, from its first token to it.
#[must_use]
pub fn line_start(tokens: &[Token], index: usize) -> usize {
    (0..index)
        .rev()
        .find(|candidate| {
            tokens
                .get(*candidate)
                .is_some_and(|token| token.kind == TokenKind::Newline)
        })
        .map_or(0, |newline| newline + 1)
}

/// The bracket a position is inside: the `(`, `[` or `{` not closed before it, and which of its
/// comma-separated parts the position is in.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Bracket {
    /// The opening token's index.
    pub open: usize,
    /// `(`, `[` or `{`.
    pub kind: char,
    /// How many commas at the bracket's own depth come before the position.
    pub part: usize,
    /// The names written `name:` at the bracket's own depth before the position, with the part
    /// each is in.
    pub named: Vec<(usize, String)>,
}

impl Bracket {
    /// The name written for the part the position is in, when it is named.
    #[must_use]
    pub fn current_name(&self) -> Option<&str> {
        self.named
            .iter()
            .find(|(part, _)| *part == self.part)
            .map(|(_, name)| name.as_str())
    }
}

/// The innermost bracket open at `offset`, scanning back from it over balanced brackets.
#[must_use]
pub fn enclosing(source: &str, tokens: &[Token], offset: u32) -> Option<Bracket> {
    let text = |token: &Token| {
        source
            .get(token.span.start as usize..token.span.end as usize)
            .unwrap_or_default()
    };
    let end = tokens.partition_point(|token| token.span.end <= offset);
    let mut depth = 0usize;
    let mut commas = 0usize;
    let mut colons: Vec<usize> = Vec::new();
    let mut index = end;
    while index > 0 {
        index -= 1;
        let token = tokens.get(index)?;
        if token.kind != TokenKind::Punct {
            continue;
        }
        match text(token) {
            ")" | "]" | "}" => depth += 1,
            open @ ("(" | "[" | "{") => {
                if depth == 0 {
                    let kind = open.chars().next().unwrap_or('(');
                    let named = colons
                        .iter()
                        .filter_map(|at| {
                            let word = tokens.get(at.checked_sub(1)?)?;
                            is_word(word.kind).then(|| text(word).to_owned())
                        })
                        .zip(colons.iter())
                        .map(|(name, at)| (part_of(source, tokens, index, *at), name))
                        .collect();
                    return Some(Bracket {
                        open: index,
                        kind,
                        part: commas,
                        named,
                    });
                }
                depth -= 1;
            }
            "," if depth == 0 => commas += 1,
            ":" if depth == 0 => colons.push(index),
            _ => {}
        }
    }
    None
}

/// Which comma-separated part of the bracket opened at `open` token `at` is in.
fn part_of(source: &str, tokens: &[Token], open: usize, at: usize) -> usize {
    let mut depth = 0usize;
    let mut part = 0;
    for token in tokens.get(open + 1..at).unwrap_or_default() {
        if token.kind != TokenKind::Punct {
            continue;
        }
        match source
            .get(token.span.start as usize..token.span.end as usize)
            .unwrap_or_default()
        {
            "(" | "[" | "{" => depth += 1,
            ")" | "]" | "}" => depth = depth.saturating_sub(1),
            "," if depth == 0 => part += 1,
            _ => {}
        }
    }
    part
}
