//! Read-only walks over the syntax tree, for the checker's pre-passes (the names a
//! declaration reads, the charges a tariff bills).

use crate::tariff::syntax::ast::{
    ArmBody, Block, Condition, Expect, Expr, ExprKind, Pattern, PatternKind, RoundTo, Stmt,
    StmtKind, TextPart, TotalClause,
};

/// Visits an expression and every expression inside it, outermost first.
pub fn expr(expr_node: &Expr, visit: &mut impl FnMut(&Expr)) {
    visit(expr_node);
    match &expr_node.kind {
        ExprKind::Text(text) | ExprKind::Fail(text) => {
            for part in &text.parts {
                if let TextPart::Expr(inner) = part {
                    expr(inner, visit);
                }
            }
        }
        ExprKind::Field { target, .. } => expr(target, visit),
        ExprKind::Call { callee, args } => {
            expr(callee, visit);
            for arg in args {
                expr(&arg.value, visit);
            }
        }
        ExprKind::Index { target, keys } => {
            expr(target, visit);
            for key in keys {
                expr(key, visit);
            }
        }
        ExprKind::Unary { operand, .. } => expr(operand, visit),
        ExprKind::Binary { left, right, .. } => {
            expr(left, visit);
            expr(right, visit);
        }
        ExprKind::Range { start, end, .. } => {
            for part in [start, end].into_iter().flatten() {
                expr(part, visit);
            }
        }
        ExprKind::List(items) => {
            for item in items {
                expr(item, visit);
            }
        }
        ExprKind::Record(fields) => {
            for field in fields {
                expr(&field.value, visit);
            }
        }
        ExprKind::If {
            branches,
            otherwise,
        } => {
            for (condition, value) in branches {
                self::condition(condition, visit);
                expr(value, visit);
            }
            expr(otherwise, visit);
        }
        ExprKind::Match { scrutinee, arms } => {
            expr(scrutinee, visit);
            for arm in arms {
                pattern(&arm.pattern, visit);
                if let Some(guard) = &arm.guard {
                    expr(guard, visit);
                }
                match &arm.body {
                    ArmBody::Expr(body) => expr(body, visit),
                    ArmBody::Block(body) => block(body, &mut |_| {}, visit),
                }
            }
        }
        ExprKind::Lambda { body, .. } => expr(body, visit),
        _ => {}
    }
}

fn condition(condition: &Condition, visit: &mut impl FnMut(&Expr)) {
    match condition {
        Condition::Expr(value) | Condition::Let { value, .. } => expr(value, visit),
    }
}

fn pattern(pattern_node: &Pattern, visit: &mut impl FnMut(&Expr)) {
    match &pattern_node.kind {
        PatternKind::Literal(value) => expr(value, visit),
        PatternKind::Range { start, end, .. } => {
            for part in [start, end].into_iter().flatten() {
                expr(part, visit);
            }
        }
        PatternKind::Record(fields) => {
            for (_, sub) in fields {
                pattern(sub, visit);
            }
        }
        PatternKind::Or(alternatives) => {
            for alternative in alternatives {
                pattern(alternative, visit);
            }
        }
        _ => {}
    }
}

/// Visits every statement of a block, nested ones included, and every expression in them.
pub fn block(block_node: &Block, on_stmt: &mut impl FnMut(&Stmt), visit: &mut impl FnMut(&Expr)) {
    for stmt in &block_node.stmts {
        self::stmt(stmt, on_stmt, visit);
    }
}

/// Visits a statement, the statements nested in it, and every expression in them.
pub fn stmt(stmt_node: &Stmt, on_stmt: &mut impl FnMut(&Stmt), visit: &mut impl FnMut(&Expr)) {
    on_stmt(stmt_node);
    match &stmt_node.kind {
        StmtKind::Let(binding) | StmtKind::Var(binding) => expr(&binding.value, visit),
        StmtKind::Assign { value, .. }
        | StmtKind::Return(value)
        | StmtKind::Given { value, .. }
        | StmtKind::Expr(value) => {
            expr(value, visit);
        }
        StmtKind::Charge(charge) => {
            for part in &charge.label.parts {
                if let TextPart::Expr(inner) = part {
                    expr(inner, visit);
                }
            }
            for option in &charge.options {
                expr(&option.value, visit);
            }
            if let Some(amount) = &charge.amount {
                expr(amount, visit);
            }
        }
        StmtKind::If(statement) => {
            for (condition, body) in &statement.branches {
                self::condition(condition, visit);
                block(body, on_stmt, visit);
            }
            if let Some(otherwise) = &statement.otherwise {
                block(otherwise, on_stmt, visit);
            }
        }
        StmtKind::Match(statement) => {
            expr(&statement.scrutinee, visit);
            for arm in &statement.arms {
                pattern(&arm.pattern, visit);
                if let Some(guard) = &arm.guard {
                    expr(guard, visit);
                }
                match &arm.body {
                    ArmBody::Expr(body) => expr(body, visit),
                    ArmBody::Block(body) => block(body, on_stmt, visit),
                }
            }
        }
        StmtKind::For(looped) => {
            expr(&looped.iterable, visit);
            if let Some(filter) = &looped.filter {
                expr(filter, visit);
            }
            block(&looped.body, on_stmt, visit);
        }
        StmtKind::Require { condition, .. } => self::condition(condition, visit),
        StmtKind::Total(
            TotalClause::Minimum(value)
            | TotalClause::Maximum(value)
            | TotalClause::Round {
                to: RoundTo::Amount(value),
                ..
            },
        )
        | StmtKind::Expect(Expect::Holds(value)) => expr(value, visit),
        _ => {}
    }
}
