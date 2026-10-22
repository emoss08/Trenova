//! Walks over a syntax tree: every expression, in source order, each before its children.
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
    ($($m:ident)?; $exprs:ident, $item:ident, $block:ident, $condition:ident, $text:ident, $stmt:ident, $arms:ident, $pattern:ident, $expr:ident) => {
    /// Calls `visit` on every expression of `file`, in source order, each before its children.
    pub fn $exprs(file: & $($m)? File, visit: &mut impl FnMut(& $($m)? Expr)) {
        for item in & $($m)? file.items {
            $item(item, visit);
        }
    }

    fn $item(item: & $($m)? Item, visit: &mut impl FnMut(& $($m)? Expr)) {
        match & $($m)? item.kind {
            ItemKind::Use(_) | ItemKind::Type(_) | ItemKind::Enum(_) => {}
            ItemKind::Input(input) => {
                if let Some(default) = & $($m)? input.default {
                    $expr(default, visit);
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
            ItemKind::Stmt(stmt) => $stmt(stmt, visit),
        }
    }

    fn $block(block: & $($m)? Block, visit: &mut impl FnMut(& $($m)? Expr)) {
        for stmt in & $($m)? block.stmts {
            $stmt(stmt, visit);
        }
    }

    fn $condition(condition: & $($m)? Condition, visit: &mut impl FnMut(& $($m)? Expr)) {
        match condition {
            Condition::Expr(expr) | Condition::Let { value: expr, .. } => $expr(expr, visit),
        }
    }

    fn $text(text: & $($m)? Text, visit: &mut impl FnMut(& $($m)? Expr)) {
        for part in & $($m)? text.parts {
            if let TextPart::Expr(expr) = part {
                $expr(expr, visit);
            }
        }
    }

    fn $stmt(stmt: & $($m)? Stmt, visit: &mut impl FnMut(& $($m)? Expr)) {
        match & $($m)? stmt.kind {
            StmtKind::Let(binding) | StmtKind::Var(binding) => $expr(& $($m)? binding.value, visit),
            StmtKind::Assign { value, .. }
            | StmtKind::Return(value)
            | StmtKind::Given { value, .. }
            | StmtKind::Expr(value)
            | StmtKind::Expect(Expect::Holds(value))
            | StmtKind::Total(TotalClause::Minimum(value) | TotalClause::Maximum(value)) => {
                $expr(value, visit);
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
            | StmtKind::Fail(text)
            | StmtKind::Expect(Expect::Fails(Some(text))) => {
                $text(text, visit);
            }
            StmtKind::Expect(Expect::Fails(None)) | StmtKind::Break | StmtKind::Continue => {}
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

    fn $arms(arms: & $($m)? [Arm], visit: &mut impl FnMut(& $($m)? Expr)) {
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

    fn $pattern(pattern: & $($m)? Pattern, visit: &mut impl FnMut(& $($m)? Expr)) {
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
    pub fn $expr(expr: & $($m)? Expr, visit: &mut impl FnMut(& $($m)? Expr)) {
        visit(expr);
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
        }
    }

    };
}

walker!(mut; exprs_mut, item_mut, block_mut, condition_mut, text_mut, stmt_mut, arms_mut, pattern_mut, expr_mut);
walker!(; exprs, item_ref, block_ref, condition_ref, text_ref, stmt_ref, arms_ref, pattern_ref, expr);
