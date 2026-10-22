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

/// Calls `visit` on every expression of `file`, in source order, each before its children.
pub fn exprs_mut(file: &mut File, visit: &mut impl FnMut(&mut Expr)) {
    for item in &mut file.items {
        item_mut(item, visit);
    }
}

fn item_mut(item: &mut Item, visit: &mut impl FnMut(&mut Expr)) {
    match &mut item.kind {
        ItemKind::Use(_) | ItemKind::Type(_) | ItemKind::Enum(_) => {}
        ItemKind::Input(input) => {
            if let Some(default) = &mut input.default {
                expr_mut(default, visit);
            }
        }
        ItemKind::Const(constant) => expr_mut(&mut constant.value, visit),
        ItemKind::Fn(function) => {
            for param in &mut function.params {
                if let Some(default) = &mut param.default {
                    expr_mut(default, visit);
                }
            }
            match &mut function.body {
                FnBody::Block(block) => block_mut(block, visit),
                FnBody::Expr(expr) => expr_mut(expr, visit),
            }
        }
        ItemKind::Test(test) => block_mut(&mut test.body, visit),
        ItemKind::Stmt(stmt) => stmt_mut(stmt, visit),
    }
}

fn block_mut(block: &mut Block, visit: &mut impl FnMut(&mut Expr)) {
    for stmt in &mut block.stmts {
        stmt_mut(stmt, visit);
    }
}

fn condition_mut(condition: &mut Condition, visit: &mut impl FnMut(&mut Expr)) {
    match condition {
        Condition::Expr(expr) | Condition::Let { value: expr, .. } => expr_mut(expr, visit),
    }
}

fn text_mut(text: &mut Text, visit: &mut impl FnMut(&mut Expr)) {
    for part in &mut text.parts {
        if let TextPart::Expr(expr) = part {
            expr_mut(expr, visit);
        }
    }
}

fn stmt_mut(stmt: &mut Stmt, visit: &mut impl FnMut(&mut Expr)) {
    match &mut stmt.kind {
        StmtKind::Let(binding) | StmtKind::Var(binding) => expr_mut(&mut binding.value, visit),
        StmtKind::Assign { value, .. }
        | StmtKind::Return(value)
        | StmtKind::Given { value, .. }
        | StmtKind::Expr(value)
        | StmtKind::Expect(Expect::Holds(value))
        | StmtKind::Total(TotalClause::Minimum(value) | TotalClause::Maximum(value)) => {
            expr_mut(value, visit);
        }
        StmtKind::Total(TotalClause::Round { to, .. }) => {
            if let RoundTo::Amount(amount) = to {
                expr_mut(amount, visit);
            }
        }
        StmtKind::Charge(charge) => {
            text_mut(&mut charge.label, visit);
            for option in &mut charge.options {
                expr_mut(&mut option.value, visit);
            }
            if let Some(amount) = &mut charge.amount {
                expr_mut(amount, visit);
            }
        }
        StmtKind::Note(text)
        | StmtKind::Fail(text)
        | StmtKind::Expect(Expect::Fails(Some(text))) => {
            text_mut(text, visit);
        }
        StmtKind::Expect(Expect::Fails(None)) | StmtKind::Break | StmtKind::Continue => {}
        StmtKind::If(statement) => {
            for (condition, block) in &mut statement.branches {
                condition_mut(condition, visit);
                block_mut(block, visit);
            }
            if let Some(block) = &mut statement.otherwise {
                block_mut(block, visit);
            }
        }
        StmtKind::Match(statement) => {
            expr_mut(&mut statement.scrutinee, visit);
            arms_mut(&mut statement.arms, visit);
        }
        StmtKind::For(statement) => {
            expr_mut(&mut statement.iterable, visit);
            if let Some(filter) = &mut statement.filter {
                expr_mut(filter, visit);
            }
            block_mut(&mut statement.body, visit);
        }
        StmtKind::Require { condition, message } => {
            condition_mut(condition, visit);
            text_mut(message, visit);
        }
        StmtKind::Item(item) => item_mut(item, visit),
    }
}

fn arms_mut(arms: &mut [Arm], visit: &mut impl FnMut(&mut Expr)) {
    for arm in arms {
        pattern_mut(&mut arm.pattern, visit);
        if let Some(guard) = &mut arm.guard {
            expr_mut(guard, visit);
        }
        match &mut arm.body {
            ArmBody::Block(block) => block_mut(block, visit),
            ArmBody::Expr(expr) => expr_mut(expr, visit),
        }
    }
}

fn pattern_mut(pattern: &mut Pattern, visit: &mut impl FnMut(&mut Expr)) {
    match &mut pattern.kind {
        PatternKind::Literal(expr) => expr_mut(expr, visit),
        PatternKind::Range { start, end, .. } => {
            for side in [start, end].into_iter().flatten() {
                expr_mut(side, visit);
            }
        }
        PatternKind::Record(fields) => {
            for (_, field) in fields {
                pattern_mut(field, visit);
            }
        }
        PatternKind::Or(alternatives) => {
            for alternative in alternatives {
                pattern_mut(alternative, visit);
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
pub fn expr_mut(expr: &mut Expr, visit: &mut impl FnMut(&mut Expr)) {
    visit(expr);
    match &mut expr.kind {
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
        ExprKind::Text(text) | ExprKind::Fail(text) => text_mut(text, visit),
        ExprKind::Field { target, .. } => expr_mut(target, visit),
        ExprKind::Call { callee, args } => {
            expr_mut(callee, visit);
            for arg in args {
                expr_mut(&mut arg.value, visit);
            }
        }
        ExprKind::Index { target, keys } => {
            expr_mut(target, visit);
            for key in keys {
                expr_mut(key, visit);
            }
        }
        ExprKind::Unary { operand, .. } => expr_mut(operand, visit),
        ExprKind::Binary { left, right, .. } => {
            expr_mut(left, visit);
            expr_mut(right, visit);
        }
        ExprKind::Range { start, end, .. } => {
            for side in [start, end].into_iter().flatten() {
                expr_mut(side, visit);
            }
        }
        ExprKind::List(items) => {
            for item in items {
                expr_mut(item, visit);
            }
        }
        ExprKind::Record(fields) => {
            for field in fields {
                expr_mut(&mut field.value, visit);
            }
        }
        ExprKind::If {
            branches,
            otherwise,
        } => {
            for (condition, value) in branches {
                condition_mut(condition, visit);
                expr_mut(value, visit);
            }
            expr_mut(otherwise, visit);
        }
        ExprKind::Match { scrutinee, arms } => {
            expr_mut(scrutinee, visit);
            arms_mut(arms, visit);
        }
        ExprKind::Lambda { body, .. } => expr_mut(body, visit),
    }
}
