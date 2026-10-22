//! The formatter: one statement per line, `let` lines before the value, a single space around
//! every operator, and a long line broken where it reads best: a sum one term per line with the
//! operator leading, a call one argument per line, a ternary one case per line, a `let` after its
//! `=` when the value then fits.
//!
//! It prints the tree the parser built, so the author's parentheses, literals and operator
//! spellings (`and` stays `and`) survive; only whitespace changes. A formula that does not parse,
//! or that holds a comment the tree would drop, is left exactly as it is. Line widths are counted
//! in UTF-16 code units, as the editor counts columns.

use super::ast::Binary;
use super::ast::BinaryOp;
use super::ast::Call;
use super::ast::Chain;
use super::ast::Expr;
use super::ast::ExprKind;
use super::ast::If;
use super::ast::Method;
use super::ast::Pointer;
use super::ast::Program;
use super::ast::Statement;
use super::ast::StatementKind;
use super::ast::Ternary;
use super::ast::UnaryOp;
use super::lexer::adjoins;
use super::parse;

/// The column a line is broken before when the caller names none.
pub const DEFAULT_WIDTH: usize = 80;

const INDENT: usize = 2;

/// Why a formula was left as it is.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FormatRefusal {
    /// It holds nothing but spaces and comments.
    Empty,
    /// It does not parse; formatting a broken tree would drop what the parser could not place.
    Syntax,
    /// It holds a comment, which the tree does not keep.
    Comments,
}

impl FormatRefusal {
    /// The refusal as the wire names it.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Empty => "empty",
            Self::Syntax => "syntax",
            Self::Comments => "comments",
        }
    }
}

/// What formatting a formula came to.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FormatOutcome {
    /// The formatted text, and whether it differs from the source.
    Formatted {
        /// The formatted formula.
        text: String,
        /// It differs from what was given.
        changed: bool,
    },
    /// The formula was left alone.
    Refused(FormatRefusal),
}

/// Formats a formula, breaking lines before `width` columns. `operators` are the schema's word
/// operators beyond the built-in four.
#[must_use]
pub fn format(source: &str, width: usize, operators: &[&str]) -> FormatOutcome {
    let program = parse(source, operators);
    if program.empty {
        return FormatOutcome::Refused(FormatRefusal::Empty);
    }
    if !program.comments.is_empty() {
        return FormatOutcome::Refused(FormatRefusal::Comments);
    }
    if !program.diagnostics.is_empty() || has_missing(&program) {
        return FormatOutcome::Refused(FormatRefusal::Syntax);
    }
    let text = Printer { source, width }.program(&program);
    let changed = text != source;
    FormatOutcome::Formatted { text, changed }
}

fn has_missing(program: &Program) -> bool {
    let mut missing = false;
    program.walk(&mut |expr| missing |= expr.is_missing());
    missing
}

fn units(text: &str) -> usize {
    text.chars().map(char::len_utf16).sum()
}

fn last_line_length(text: &str) -> usize {
    text.rfind('\n').map_or_else(
        || units(text),
        |at| units(text.get(at + 1..).unwrap_or_default()),
    )
}

fn pad(columns: usize) -> String {
    " ".repeat(columns)
}

/// Whether the text between two statements holds a blank line.
fn has_blank_line(gap: &str) -> bool {
    gap.match_indices('\n').any(|(at, _)| {
        let rest = gap.get(at + 1..).unwrap_or_default();
        let rest = rest.trim_start_matches([' ', '\t']);
        let rest = rest.strip_prefix('\r').unwrap_or(rest);
        rest.starts_with('\n')
    })
}

/// `object` then `dot` and `name`, spaced apart when the dot would otherwise join a bare
/// number: `27 .g4` stays a member, where `27.g4` reads as `27.` then `g4`. Only a bare
/// number can take in a following dot; every other unparenthesised object ends in a name,
/// a bracket or a quote.
fn access(object: &Expr, printed: &str, dot: &str, name: &str) -> String {
    let bare_number = object.parens == 0 && matches!(object.kind, ExprKind::Number(_));
    let access = format!("{dot}{name}");
    if bare_number && !adjoins(printed, &access) {
        return format!("{printed} {access}");
    }
    format!("{printed}{access}")
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum Group {
    Add,
    Multiply,
    And,
    Or,
    Coalesce,
}

fn group(op: &BinaryOp) -> Option<Group> {
    match op {
        BinaryOp::Add | BinaryOp::Subtract => Some(Group::Add),
        BinaryOp::Multiply | BinaryOp::Divide | BinaryOp::Modulo => Some(Group::Multiply),
        BinaryOp::And { .. } => Some(Group::And),
        BinaryOp::Or { .. } => Some(Group::Or),
        BinaryOp::Coalesce => Some(Group::Coalesce),
        _ => None,
    }
}

struct Printer<'a> {
    source: &'a str,
    width: usize,
}

impl Printer<'_> {
    fn program(&self, program: &Program) -> String {
        let mut lines: Vec<String> = Vec::with_capacity(program.statements.len());
        let count = program.statements.len();
        for (index, statement) in program.statements.iter().enumerate() {
            if index > 0 {
                let previous = &program.statements[index - 1];
                let gap = self
                    .source
                    .get(previous.span.end as usize..statement.span.start as usize)
                    .unwrap_or_default();
                if has_blank_line(gap) {
                    lines.push(String::new());
                }
            }
            lines.push(self.statement(statement, index + 1 == count));
        }
        lines.join("\n")
    }

    fn statement(&self, statement: &Statement, last: bool) -> String {
        let StatementKind::Let(name) = &statement.kind else {
            let value = self.node(&statement.value, 0, 0);
            return if last { value } else { format!("{value};") };
        };
        let head = format!("let {} = ", name.text);
        let flat = self.flat(&statement.value);
        if units(&head) + units(&flat) < self.width {
            return format!("{head}{flat};");
        }
        if INDENT + units(&flat) < self.width {
            return format!("let {} =\n{}{flat};", name.text, pad(INDENT));
        }
        format!("{head}{};", self.node(&statement.value, units(&head), 0))
    }

    /// The node at a column, broken over lines indented past `indent` when it does not fit.
    fn node(&self, expr: &Expr, column: usize, indent: usize) -> String {
        self.node_in(expr, expr.parens, column, indent)
    }

    fn node_in(&self, expr: &Expr, parens: u8, column: usize, indent: usize) -> String {
        let flat = self.flat_in(expr, parens);
        if column + units(&flat) <= self.width {
            return flat;
        }
        self.broken(expr, parens, column, indent)
    }

    fn broken(&self, expr: &Expr, parens: u8, column: usize, indent: usize) -> String {
        if parens > 0 {
            return format!("({})", self.broken(expr, parens - 1, column + 1, indent));
        }
        match &expr.kind {
            ExprKind::Binary(binary) => self.broken_binary(binary, column, indent),
            ExprKind::Chain(chain) => {
                self.broken_chain(chain, chain.operands.len(), column, indent)
            }
            ExprKind::Ternary(_) => self.broken_ternary(expr, indent),
            ExprKind::Call(call) => self.broken_call(call, column, indent),
            ExprKind::Method(method) => self.broken_method(method, column, indent),
            ExprKind::Unary(unary) => {
                let op = match unary.op {
                    UnaryOp::Not { word: true } => String::from("not "),
                    op => String::from(op.text()),
                };
                let operand = self.node(&unary.operand, column + units(&op), indent);
                format!("{op}{operand}")
            }
            ExprKind::If(branch) => self.broken_if(branch, indent),
            ExprKind::Array(items) => {
                if items.is_empty() {
                    return String::from("[]");
                }
                let inner = indent + INDENT;
                let items: Vec<String> = items
                    .iter()
                    .map(|item| format!("{}{}", pad(inner), self.node(item, inner, inner)))
                    .collect();
                format!("[\n{}\n{}]", items.join(",\n"), pad(indent))
            }
            ExprKind::Map(entries) => {
                if entries.is_empty() {
                    return String::from("{}");
                }
                let inner = indent + INDENT;
                let entries: Vec<String> = entries
                    .iter()
                    .map(|entry| {
                        let head = format!("{}{}: ", pad(inner), self.slice(entry.key.span));
                        let value = self.node(&entry.value, units(&head), inner);
                        format!("{head}{value}")
                    })
                    .collect();
                format!("{{\n{}\n{}}}", entries.join(",\n"), pad(indent))
            }
            ExprKind::Predicate(predicate) => {
                if predicate.braces {
                    format!("{{{}}}", self.node(&predicate.body, column + 1, indent))
                } else {
                    self.node(&predicate.body, column, indent)
                }
            }
            ExprKind::Member(member) => {
                let object = self.node(&member.object, column, indent);
                let dot = if member.optional { "?." } else { "." };
                access(&member.object, &object, dot, &member.property.text)
            }
            ExprKind::Index(index) => {
                let object = self.node(&index.object, column, indent);
                let open = if index.optional { "?.[" } else { "[" };
                let inner = self.node(
                    &index.index,
                    last_line_length(&object) + units(open),
                    indent,
                );
                format!("{object}{open}{inner}]")
            }
            _ => self.flat_in(expr, 0),
        }
    }

    /// A chain of one precedence group, one operand per line with the operator leading the
    /// continuation, the way a column of charges reads.
    fn broken_binary(&self, binary: &Binary, column: usize, indent: usize) -> String {
        let next = indent + INDENT;
        let Some(kind) = group(&binary.op).filter(|_| binary.not.is_none()) else {
            let left = self.node(&binary.left, column, indent);
            let op = if binary.not.is_some() {
                format!("not {}", binary.op.text())
            } else {
                binary.op.text().to_owned()
            };
            let right = self.node(&binary.right, next + units(&op) + 1, next);
            return format!("{left}\n{}{op} {right}", pad(next));
        };
        let mut operands: Vec<&Expr> = vec![&binary.right];
        let mut operators: Vec<&str> = vec![binary.op.text()];
        let mut current: &Expr = &binary.left;
        while let ExprKind::Binary(inner) = &current.kind {
            if current.parens > 0 || inner.not.is_some() || group(&inner.op) != Some(kind) {
                break;
            }
            operands.push(&inner.right);
            operators.push(inner.op.text());
            current = &inner.left;
        }
        operands.push(current);
        operands.reverse();
        operators.reverse();
        let mut lines = vec![self.node(operands[0], column, next)];
        for (op, operand) in operators.iter().zip(operands.iter().skip(1)) {
            let printed = self.node(operand, next + units(op) + 1, next);
            lines.push(format!("{}{op} {printed}", pad(next)));
        }
        lines.join("\n")
    }

    /// The first `count` operands of a comparison chain, printed as the left-nested comparisons
    /// they mean, breaking before the last operator when the whole does not fit.
    fn chain_node(&self, chain: &Chain, count: usize, column: usize, indent: usize) -> String {
        if count <= 1 {
            return chain
                .operands
                .first()
                .map(|operand| self.node(operand, column, indent))
                .unwrap_or_default();
        }
        let flat = self.chain_flat(chain, count);
        if column + units(&flat) <= self.width {
            return flat;
        }
        self.broken_chain(chain, count, column, indent)
    }

    fn broken_chain(&self, chain: &Chain, count: usize, column: usize, indent: usize) -> String {
        let next = indent + INDENT;
        let left = self.chain_node(chain, count - 1, column, indent);
        let op = chain
            .operators
            .get(count - 2)
            .map_or("", |(op, _)| op.text());
        let right = chain
            .operands
            .get(count - 1)
            .map(|operand| self.node(operand, next + units(op) + 1, next))
            .unwrap_or_default();
        format!("{left}\n{}{op} {right}", pad(next))
    }

    fn chain_flat(&self, chain: &Chain, count: usize) -> String {
        let mut out = String::new();
        for (index, operand) in chain.operands.iter().take(count).enumerate() {
            if index > 0 {
                let op = chain
                    .operators
                    .get(index - 1)
                    .map_or("", |(op, _)| op.text());
                out.push(' ');
                out.push_str(op);
                out.push(' ');
            }
            out.push_str(&self.flat(operand));
        }
        out
    }

    /// `test ? a` over `: test ? b` over `: c`, each case on its own line.
    fn broken_ternary(&self, expr: &Expr, indent: usize) -> String {
        let next = indent + INDENT;
        let mut cases = Vec::new();
        let mut current = expr;
        let mut first = true;
        let mut lead = ": ";
        while let ExprKind::Ternary(Ternary {
            test,
            then,
            otherwise,
        }) = &current.kind
        {
            if !first && current.parens > 0 {
                break;
            }
            let column = if first { indent } else { next + lead.len() };
            let test = self.node(test, column, next);
            let branch = match then {
                None => format!("{test} ?:"),
                Some(then) => {
                    let value = self.node(then, last_line_length(&test) + 3, next);
                    format!("{test} ? {value}")
                }
            };
            cases.push(if first {
                branch
            } else {
                format!("{}{lead}{branch}", pad(next))
            });
            lead = if then.is_none() { "" } else { ": " };
            current = otherwise;
            first = false;
        }
        cases.push(format!(
            "{}{lead}{}",
            pad(next),
            self.node(current, next + lead.len(), next)
        ));
        cases.join(
            "
",
        )
    }

    fn broken_call(&self, call: &Call, column: usize, indent: usize) -> String {
        if call.pipe {
            let next = indent + INDENT;
            let left = call
                .args
                .first()
                .map(|arg| self.node(arg, column, indent))
                .unwrap_or_default();
            return format!("{left}\n{}| {}", pad(next), self.pipe_target(call));
        }
        self.broken_arguments(&call.function.text, &call.args, indent)
    }

    fn broken_method(&self, method: &Method, column: usize, indent: usize) -> String {
        let receiver = self.node(&method.receiver, column, indent);
        let dot = if method.optional { "?." } else { "." };
        let callee = access(&method.receiver, &receiver, dot, &method.method.text);
        if method.args.is_empty() {
            return format!("{callee}()");
        }
        self.broken_arguments(&callee, &method.args, indent)
    }

    fn broken_arguments(&self, callee: &str, args: &[Expr], indent: usize) -> String {
        if args.is_empty() {
            return format!("{callee}()");
        }
        let inner = indent + INDENT;
        let args: Vec<String> = args
            .iter()
            .map(|arg| format!("{}{}", pad(inner), self.node(arg, inner, inner)))
            .collect();
        format!("{callee}(\n{}\n{})", args.join(",\n"), pad(indent))
    }

    fn broken_if(&self, branch: &If, indent: usize) -> String {
        let inner = indent + INDENT;
        let test = self.node(&branch.test, indent + 3, indent);
        let then = format!("{}{}", pad(inner), self.node(&branch.then, inner, inner));
        let otherwise = match &branch.otherwise.kind {
            ExprKind::If(next) if branch.otherwise.parens == 0 => self.broken_if(next, indent),
            _ => format!(
                "{{\n{}{}\n{}}}",
                pad(inner),
                self.node(&branch.otherwise, inner, inner),
                pad(indent)
            ),
        };
        format!("if {test} {{\n{then}\n{}}} else {otherwise}", pad(indent))
    }

    fn slice(&self, span: crate::span::Span) -> &str {
        self.source
            .get(span.start as usize..span.end as usize)
            .unwrap_or_default()
    }

    /// A literal as the author wrote it, without the parentheses around it.
    fn literal(&self, expr: &Expr) -> String {
        let mut text = self.slice(expr.span).trim();
        for _ in 0..expr.parens {
            if let Some(inner) = text.strip_prefix('(').and_then(|t| t.strip_suffix(')')) {
                text = inner.trim();
            }
        }
        text.to_owned()
    }

    fn flat(&self, expr: &Expr) -> String {
        self.flat_in(expr, expr.parens)
    }

    fn flat_in(&self, expr: &Expr, parens: u8) -> String {
        let inner = self.flat_inner(expr);
        if parens == 0 {
            return inner;
        }
        let layers = usize::from(parens);
        format!("{}{inner}{}", "(".repeat(layers), ")".repeat(layers))
    }

    fn pipe_target(&self, call: &Call) -> String {
        if call.parens == call.function.span {
            return call.function.text.clone();
        }
        let rest: Vec<String> = call.args.iter().skip(1).map(|arg| self.flat(arg)).collect();
        format!("{}({})", call.function.text, rest.join(", "))
    }

    fn flat_inner(&self, expr: &Expr) -> String {
        match &expr.kind {
            ExprKind::Call(_)
            | ExprKind::Method(_)
            | ExprKind::Unary(_)
            | ExprKind::Binary(_)
            | ExprKind::Chain(_)
            | ExprKind::Ternary(_)
            | ExprKind::If(_) => self.flat_operation(expr),
            ExprKind::Number(_) | ExprKind::String(_) => self.literal(expr),
            ExprKind::Bool(value) => String::from(if *value { "true" } else { "false" }),
            ExprKind::Nil => String::from("nil"),
            ExprKind::Pointer(pointer) => pointer.text(),
            ExprKind::Ident(name) => name.clone(),
            ExprKind::Member(member) => {
                if matches!(member.object.kind, ExprKind::Pointer(Pointer::Implicit)) {
                    return format!(".{}", member.property.text);
                }
                let dot = if member.optional { "?." } else { "." };
                let object = self.flat(&member.object);
                access(&member.object, &object, dot, &member.property.text)
            }
            ExprKind::Index(index) => {
                let open = if index.optional { "?.[" } else { "[" };
                format!(
                    "{}{open}{}]",
                    self.flat(&index.object),
                    self.flat(&index.index)
                )
            }
            ExprKind::Slice(slice) => {
                let open = if slice.optional { "?.[" } else { "[" };
                let start = slice
                    .start
                    .as_ref()
                    .map(|s| self.flat(s))
                    .unwrap_or_default();
                let end = slice.end.as_ref().map(|e| self.flat(e)).unwrap_or_default();
                format!("{}{open}{start}:{end}]", self.flat(&slice.object))
            }
            ExprKind::Array(items) => {
                let items: Vec<String> = items.iter().map(|item| self.flat(item)).collect();
                format!("[{}]", items.join(", "))
            }
            ExprKind::Map(entries) => {
                if entries.is_empty() {
                    return String::from("{}");
                }
                let entries: Vec<String> = entries
                    .iter()
                    .map(|entry| {
                        format!(
                            "{}: {}",
                            self.slice(entry.key.span),
                            self.flat(&entry.value)
                        )
                    })
                    .collect();
                format!("{{{}}}", entries.join(", "))
            }
            ExprKind::Predicate(predicate) => {
                if predicate.braces {
                    format!("{{{}}}", self.flat(&predicate.body))
                } else {
                    self.flat(&predicate.body)
                }
            }
            ExprKind::Sequence(sequence) => {
                let mut out = String::new();
                for entry in &sequence.lets {
                    out.push_str("let ");
                    out.push_str(&entry.name.text);
                    out.push_str(" = ");
                    out.push_str(&self.flat(&entry.value));
                    out.push_str("; ");
                }
                out.push_str(&self.flat(&sequence.body));
                out
            }
            ExprKind::Missing => String::new(),
        }
    }

    /// A call, an operator or a conditional on one line.
    fn flat_operation(&self, expr: &Expr) -> String {
        match &expr.kind {
            ExprKind::Call(call) => {
                if call.pipe {
                    let left = call
                        .args
                        .first()
                        .map(|arg| self.flat(arg))
                        .unwrap_or_default();
                    return format!("{left} | {}", self.pipe_target(call));
                }
                let args: Vec<String> = call.args.iter().map(|arg| self.flat(arg)).collect();
                format!("{}({})", call.function.text, args.join(", "))
            }
            ExprKind::Method(method) => {
                let dot = if method.optional { "?." } else { "." };
                let args: Vec<String> = method.args.iter().map(|arg| self.flat(arg)).collect();
                let receiver = self.flat(&method.receiver);
                let callee = access(&method.receiver, &receiver, dot, &method.method.text);
                format!("{callee}({})", args.join(", "))
            }
            ExprKind::Unary(unary) => {
                let operand = self.flat(&unary.operand);
                if let UnaryOp::Not { word: true } = unary.op {
                    return format!("not {operand}");
                }
                let spaced = matches!(
                    &unary.operand.kind,
                    ExprKind::Unary(inner) if unary.operand.parens == 0 && inner.op == unary.op
                );
                format!(
                    "{}{}{operand}",
                    unary.op.text(),
                    if spaced { " " } else { "" }
                )
            }
            ExprKind::Binary(binary) => {
                let left = self.flat(&binary.left);
                let right = self.flat(&binary.right);
                if binary.not.is_some() {
                    return format!("{left} not {} {right}", binary.op.text());
                }
                if binary.op == BinaryOp::Range {
                    if left.ends_with('.') || right.starts_with('.') {
                        return format!("{left} .. {right}");
                    }
                    return format!("{left}..{right}");
                }
                format!("{left} {} {right}", binary.op.text())
            }
            ExprKind::Chain(chain) => self.chain_flat(chain, chain.operands.len()),
            ExprKind::Ternary(ternary) => {
                let test = self.flat(&ternary.test);
                let otherwise = self.flat(&ternary.otherwise);
                match &ternary.then {
                    None => format!("{test} ?: {otherwise}"),
                    Some(then) => format!("{test} ? {} : {otherwise}", self.flat(then)),
                }
            }
            ExprKind::If(branch) => {
                let otherwise = match &branch.otherwise.kind {
                    ExprKind::If(_) if branch.otherwise.parens == 0 => {
                        self.flat_inner(&branch.otherwise)
                    }
                    _ => format!("{{ {} }}", self.flat(&branch.otherwise)),
                };
                format!(
                    "if {} {{ {} }} else {otherwise}",
                    self.flat(&branch.test),
                    self.flat(&branch.then)
                )
            }
            _ => String::new(),
        }
    }
}
