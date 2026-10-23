//! Walks over a syntax tree: every expression and statement, in source order, each before its
//! children.
//!
//! The parser numbers expressions with it; the checker, evaluator and editor may use it to visit
//! every expression of a file without writing the traversal again. A tree is never deeper than
//! the parser's depth limit, so the walk recurses.

use super::ast::Arm;
use super::ast::ArmBody;
use super::ast::Block;
use super::ast::Condition;
use super::ast::Expect;
use super::ast::Expr;
use super::ast::ExprKind;
use super::ast::File;
use super::ast::FnBody;
use super::ast::Item;
use super::ast::ItemKind;
use super::ast::Pattern;
use super::ast::PatternKind;
use super::ast::RoundTo;
use super::ast::Stmt;
use super::ast::StmtKind;
use super::ast::Text;
use super::ast::TextPart;
use super::ast::TotalClause;

macro_rules! walker {
    ($($m:ident)?; $visitor:ident; $exprs:ident, $item:ident, $block:ident, $condition:ident, $text:ident, $stmt:ident, $arms:ident, $pattern:ident, $expr:ident) => {
    /// Calls `visit` on every expression and statement of `file`, in source order, each before
    /// its children.
    pub fn $exprs(file: & $($m)? File, visit: &mut impl $visitor) {
        for item in & $($m)? file.items {
            $item(item, visit);
        }
    }

    fn $item(item: & $($m)? Item, visit: &mut impl $visitor) {
        match & $($m)? item.kind {
            ItemKind::Use(_) | ItemKind::Type(_) | ItemKind::Enum(_) => {}
            ItemKind::Input(input) => {
                if let Some(default) = & $($m)? input.default {
                    $expr(default, visit);
                }
                if let Some(constraint) = & $($m)? input.constraint {
                    $expr(constraint, visit);
                }
            }
            ItemKind::Const(constant) => $expr(& $($m)? constant.value, visit),
            ItemKind::Fn(function) => {
                for param in & $($m)? function.params {
                    if let Some(default) = & $($m)? param.default {
                        $expr(default, visit);
                    }
                }
                match & $($m)? function.body {
                    FnBody::Block(block) => $block(block, visit),
                    FnBody::Expr(expr) => $expr(expr, visit),
                }
            }
            ItemKind::Test(test) => $block(& $($m)? test.body, visit),
            ItemKind::Property(property) => $block(& $($m)? property.body, visit),
            ItemKind::Stmt(stmt) => $stmt(stmt, visit),
        }
    }

    fn $block(block: & $($m)? Block, visit: &mut impl $visitor) {
        for stmt in & $($m)? block.stmts {
            $stmt(stmt, visit);
        }
    }

    fn $condition(condition: & $($m)? Condition, visit: &mut impl $visitor) {
        match condition {
            Condition::Expr(expr) | Condition::Let { value: expr, .. } => $expr(expr, visit),
        }
    }

    fn $text(text: & $($m)? Text, visit: &mut impl $visitor) {
        for part in & $($m)? text.parts {
            if let TextPart::Expr(expr) = part {
                $expr(expr, visit);
            }
        }
    }

    fn $stmt(stmt: & $($m)? Stmt, visit: &mut impl $visitor) {
        visit.stmt(stmt);
        match & $($m)? stmt.kind {
            StmtKind::Let(binding) | StmtKind::Var(binding) => $expr(& $($m)? binding.value, visit),
            StmtKind::Assign { value, .. }
            | StmtKind::Return(value)
            | StmtKind::Given { value, .. }
            | StmtKind::Expr(value)
            | StmtKind::Expect(Expect::Holds(value) | Expect::Monotone { value, .. })
            | StmtKind::Assume(value) => {
                $expr(value, visit);
            }
            StmtKind::Total(TotalClause::Minimum(bound) | TotalClause::Maximum(bound)) => {
                $expr(& $($m)? bound.amount, visit);
            }
            StmtKind::Total(TotalClause::Round { to, .. }) => {
                if let RoundTo::Amount(amount) = to {
                    $expr(amount, visit);
                }
            }
            StmtKind::Charge(charge) => {
                $text(& $($m)? charge.label, visit);
                for option in & $($m)? charge.options {
                    $expr(& $($m)? option.value, visit);
                }
                if let Some(amount) = & $($m)? charge.amount {
                    $expr(amount, visit);
                }
            }
            StmtKind::Note(text)
            | StmtKind::Review(text)
            | StmtKind::Fail(text)
            | StmtKind::Expect(Expect::Fails(Some(text))) => {
                $text(text, visit);
            }
            StmtKind::Expect(Expect::Fails(None)) | StmtKind::Break | StmtKind::Continue => {}
            StmtKind::Vary { values, .. } => {
                if let Some(values) = values {
                    $expr(values, visit);
                }
            }
            StmtKind::Choose(choose) => {
                for option in & $($m)? choose.options {
                    $block(& $($m)? option.body, visit);
                }
            }
            StmtKind::If(statement) => {
                for (condition, block) in & $($m)? statement.branches {
                    $condition(condition, visit);
                    $block(block, visit);
                }
                if let Some(block) = & $($m)? statement.otherwise {
                    $block(block, visit);
                }
            }
            StmtKind::Match(statement) => {
                $expr(& $($m)? statement.scrutinee, visit);
                $arms(& $($m)? statement.arms, visit);
            }
            StmtKind::For(statement) => {
                $expr(& $($m)? statement.iterable, visit);
                if let Some(filter) = & $($m)? statement.filter {
                    $expr(filter, visit);
                }
                $block(& $($m)? statement.body, visit);
            }
            StmtKind::Require { condition, message } => {
                $condition(condition, visit);
                $text(message, visit);
            }
            StmtKind::Item(item) => $item(item, visit),
        }
    }

    fn $arms(arms: & $($m)? [Arm], visit: &mut impl $visitor) {
        for arm in arms {
            $pattern(& $($m)? arm.pattern, visit);
            if let Some(guard) = & $($m)? arm.guard {
                $expr(guard, visit);
            }
            match & $($m)? arm.body {
                ArmBody::Block(block) => $block(block, visit),
                ArmBody::Expr(expr) => $expr(expr, visit),
            }
        }
    }

    fn $pattern(pattern: & $($m)? Pattern, visit: &mut impl $visitor) {
        match & $($m)? pattern.kind {
            PatternKind::Literal(expr) => $expr(expr, visit),
            PatternKind::Range { start, end, .. } => {
                for side in [start, end].into_iter().flatten() {
                    $expr(side, visit);
                }
            }
            PatternKind::Record(fields) => {
                for (_, field) in fields {
                    $pattern(field, visit);
                }
            }
            PatternKind::Or(alternatives) => {
                for alternative in alternatives {
                    $pattern(alternative, visit);
                }
            }
            PatternKind::Wildcard
            | PatternKind::None
            | PatternKind::Variant { .. }
            | PatternKind::Binding(_)
            | PatternKind::Missing => {}
        }
    }

    /// Calls `visit` on `expr` and then on each expression inside it, in source order.
    pub fn $expr(expr: & $($m)? Expr, visit: &mut impl $visitor) {
        visit.expr(expr);
        match & $($m)? expr.kind {
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
            ExprKind::Text(text) | ExprKind::Fail(text) => $text(text, visit),
            ExprKind::Field { target, .. } => $expr(target, visit),
            ExprKind::Call { callee, args } => {
                $expr(callee, visit);
                for arg in args {
                    $expr(& $($m)? arg.value, visit);
                }
            }
            ExprKind::Index { target, keys } => {
                $expr(target, visit);
                for key in keys {
                    $expr(key, visit);
                }
            }
            ExprKind::Unary { operand, .. } => $expr(operand, visit),
            ExprKind::Binary { left, right, .. } => {
                $expr(left, visit);
                $expr(right, visit);
            }
            ExprKind::Range { start, end, .. } => {
                for side in [start, end].into_iter().flatten() {
                    $expr(side, visit);
                }
            }
            ExprKind::List(items) => {
                for item in items {
                    $expr(item, visit);
                }
            }
            ExprKind::Record(fields) => {
                for field in fields {
                    $expr(& $($m)? field.value, visit);
                }
            }
            ExprKind::If {
                branches,
                otherwise,
            } => {
                for (condition, value) in branches {
                    $condition(condition, visit);
                    $expr(value, visit);
                }
                $expr(otherwise, visit);
            }
            ExprKind::Match { scrutinee, arms } => {
                $expr(scrutinee, visit);
                $arms(arms, visit);
            }
            ExprKind::Lambda { body, .. } => $expr(body, visit),
            ExprKind::Choose(choose) => {
                for option in & $($m)? choose.options {
                    $block(& $($m)? option.body, visit);
                }
            }
        }
    }

    };
}

/// What a walk calls at each node; a closure over expressions is one.
pub trait Visit {
    /// An expression, before its children.
    fn expr(&mut self, expr: &Expr);
    /// A statement, before its children.
    fn stmt(&mut self, _stmt: &Stmt) {}
}

/// What a mutating walk calls at each node; a closure over expressions is one.
pub trait VisitMut {
    /// An expression, before its children.
    fn expr(&mut self, expr: &mut Expr);
    /// A statement, before its children.
    fn stmt(&mut self, _stmt: &mut Stmt) {}
}

/// A closure over expressions, as a visitor.
struct Exprs<'v, F>(&'v mut F);

impl<F: FnMut(&Expr)> Visit for Exprs<'_, F> {
    fn expr(&mut self, expr: &Expr) {
        (self.0)(expr);
    }
}

impl<F: FnMut(&mut Expr)> VisitMut for Exprs<'_, F> {
    fn expr(&mut self, expr: &mut Expr) {
        (self.0)(expr);
    }
}

walker!(mut; VisitMut; nodes_mut, item_mut, block_mut, condition_mut, text_mut, stmt_mut, arms_mut, pattern_mut, node_expr_mut);
walker!(; Visit; nodes, item_ref, block_ref, condition_ref, text_ref, stmt_ref, arms_ref, pattern_ref, node_expr);

/// Calls `visit` on every expression of `file`, in source order, each before its children.
pub fn exprs(file: &File, visit: &mut impl FnMut(&Expr)) {
    nodes(file, &mut Exprs(visit));
}

/// Calls `visit` on every expression of `file`, mutably, in source order.
pub fn exprs_mut(file: &mut File, visit: &mut impl FnMut(&mut Expr)) {
    nodes_mut(file, &mut Exprs(visit));
}

/// Calls `visit` on `expr` and then on each expression inside it, in source order.
pub fn expr(expr: &Expr, visit: &mut impl FnMut(&Expr)) {
    node_expr(expr, &mut Exprs(visit));
}

/// Calls `visit` on `expr` and then on each expression inside it, mutably, in source order.
pub fn expr_mut(expr: &mut Expr, visit: &mut impl FnMut(&mut Expr)) {
    node_expr_mut(expr, &mut Exprs(visit));
}
