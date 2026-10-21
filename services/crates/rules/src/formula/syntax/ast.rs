//! The formula syntax tree.
//!
//! Every node carries the byte [`Span`] of the text it was parsed from; a node the author wrapped
//! in parentheses keeps them in its span and counts them in [`Expr::parens`], so the formatter can
//! print them back and a diagnostic underlines what the author wrote. Operators keep the
//! author's spelling (`and` or `&&`, `not` or `!`, `**` or `^`) for the same reason.
//!
//! The tree is built by an error-tolerant parser: a part the author has not written yet is an
//! [`ExprKind::Missing`] node, and every program whose tree holds one also carries a syntax
//! diagnostic explaining it. A tree is never deeper than [`MAX_DEPTH`](super::MAX_DEPTH) nodes,
//! so every walk over it may recurse.

use crate::message::Message;
use crate::span::Span;
use rust_decimal::Decimal;

/// A parsed formula: its statements, what is wrong with its text, and the tokens and comments
/// it was read from.
#[derive(Debug, Clone, PartialEq)]
pub struct Program {
    /// The `let` lines and result expressions, in source order.
    pub statements: Vec<Statement>,
    /// Lexical and grammatical problems, sorted by where they start.
    pub diagnostics: Vec<SyntaxDiagnostic>,
    /// Every token, comments and spaces left out, ending with one [`TokenKind::Eof`].
    pub tokens: Vec<Token>,
    /// Where the comments are; the formatter refuses to drop one.
    pub comments: Vec<Span>,
    /// The text holds nothing but spaces and comments.
    pub empty: bool,
}

/// One top-level statement: `let name = value;` or an expression.
#[derive(Debug, Clone, PartialEq)]
pub struct Statement {
    /// From the first token to the last token of the value; a `let` starts at `let`. The `;`
    /// that ends the statement is not part of it.
    pub span: Span,
    /// Whether this binds a name or is a value.
    pub kind: StatementKind,
    /// The bound value, or the expression itself.
    pub value: Expr,
    /// The `;` that ends the statement, when the author wrote one.
    pub semicolon: Option<Span>,
}

/// What a top-level statement is.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum StatementKind {
    /// `let name = value;`. The name is empty (with an empty span after `let`) when the author
    /// has not written it yet.
    Let(Name),
    /// An expression; the last one is the formula's value.
    Result,
}

/// A name as written, with where it is.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct Name {
    /// The text: an identifier, or a map key's decoded value.
    pub text: String,
    /// Where it is written.
    pub span: Span,
}

/// An expression node.
#[derive(Debug, Clone, PartialEq)]
pub struct Expr {
    /// The text the node covers, including the parentheses the author put around it.
    pub span: Span,
    /// How many pairs of parentheses the author wrote around this node.
    pub parens: u8,
    /// What the node is.
    pub kind: ExprKind,
    /// The height of the subtree rooted here: 1 for a leaf.
    pub(crate) depth: u16,
}

/// The kinds of expression.
#[derive(Debug, Clone, PartialEq)]
pub enum ExprKind {
    /// A number literal.
    Number(Number),
    /// A text literal, its escapes read.
    String(String),
    /// `true` or `false`.
    Bool(bool),
    /// `nil`.
    Nil,
    /// `#`, `#acc`, `#index`, or the item `.field` reads.
    Pointer(Pointer),
    /// A name: a field, an input or a `let`.
    Ident(String),
    /// `object.property` or `object?.property`.
    Member(Member),
    /// `object[index]` or `object?.[index]`.
    Index(Index),
    /// `object[start:end]`, either bound optional.
    Slice(Slice),
    /// A function call `name(args)`, or the pipe `value | name(args)`.
    Call(Call),
    /// A method call on a value, such as `pickupDate.Year()`.
    Method(Method),
    /// `not x`, `!x`, `-x`, `+x`.
    Unary(Unary),
    /// Two operands and an operator, including a negated word operator (`a not in b`).
    Binary(Binary),
    /// A chained comparison of three or more operands: `0 < x <= 5000`.
    Chain(Chain),
    /// `test ? then : otherwise`, or `test ?: otherwise`.
    Ternary(Ternary),
    /// `if test { then } else { otherwise }`; an `else if` is an `If` in `otherwise`.
    If(If),
    /// `[a, b]`.
    Array(Vec<Expr>),
    /// `{key: value, "other key": value}`.
    Map(Vec<MapEntry>),
    /// A predicate: `{ body }`, or the bare expression a predicate function takes per item.
    Predicate(Predicate),
    /// `let` lines and the value they are scoped to, inside parentheses, a block or a
    /// predicate.
    Sequence(Sequence),
    /// A part not written yet. A program with one always has a syntax diagnostic.
    Missing,
}

/// A number literal.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Number {
    /// Its exact value; `None` when it does not fit in a decimal (a diagnostic says so).
    pub value: Option<Decimal>,
    /// It is an integer literal: hexadecimal, octal, binary, or decimal with neither a fraction
    /// nor an exponent.
    pub whole: bool,
}

/// What a pointer reads inside a predicate.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum Pointer {
    /// `#`: the current item.
    Element,
    /// `#acc`: the accumulator inside `reduce`.
    Accumulator,
    /// `#index`: the current item's position.
    Index,
    /// `#name` for a name that is no pointer; the checker reports it.
    Unknown(String),
    /// The item a leading `.field` reads; it has an empty span where the `.` is.
    Implicit,
}

impl Pointer {
    /// The pointer as written (`#`, `#acc`), or the empty text for the item `.field` reads.
    #[must_use]
    pub fn text(&self) -> String {
        match self {
            Self::Element => String::from("#"),
            Self::Accumulator => String::from("#acc"),
            Self::Index => String::from("#index"),
            Self::Unknown(name) => format!("#{name}"),
            Self::Implicit => String::new(),
        }
    }
}

/// `object.property`.
#[derive(Debug, Clone, PartialEq)]
pub struct Member {
    /// What the property is read from.
    pub object: Box<Expr>,
    /// The property name.
    pub property: Name,
    /// Written `?.`: reads nothing from an empty object.
    pub optional: bool,
}

/// `object[index]`.
#[derive(Debug, Clone, PartialEq)]
pub struct Index {
    /// What is indexed.
    pub object: Box<Expr>,
    /// The index or key.
    pub index: Box<Expr>,
    /// Written `?.[`: reads nothing from an empty object.
    pub optional: bool,
}

/// `object[start:end]`.
#[derive(Debug, Clone, PartialEq)]
pub struct Slice {
    /// What is sliced.
    pub object: Box<Expr>,
    /// The first position, or from the start.
    pub start: Option<Box<Expr>>,
    /// One past the last position, or to the end.
    pub end: Option<Box<Expr>>,
    /// Written `?.[`.
    pub optional: bool,
}

/// A function call.
#[derive(Debug, Clone, PartialEq)]
pub struct Call {
    /// The function's name.
    pub function: Name,
    /// The arguments; for a pipe the piped value is the first.
    pub args: Vec<Expr>,
    /// From `(` to `)`. A pipe written without parentheses (a syntax error) has the name's span.
    pub parens: Span,
    /// Written as `value | function(rest)`.
    pub pipe: bool,
}

/// A method call on a value.
#[derive(Debug, Clone, PartialEq)]
pub struct Method {
    /// The value the method is called on.
    pub receiver: Box<Expr>,
    /// The method's name.
    pub method: Name,
    /// Written `?.`: calls nothing on an empty value.
    pub optional: bool,
    /// The arguments.
    pub args: Vec<Expr>,
    /// From `(` to `)`.
    pub parens: Span,
}

/// A prefix operator.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum UnaryOp {
    /// `not` (`word`) or `!`.
    Not {
        /// Spelled `not`.
        word: bool,
    },
    /// `-`.
    Negate,
    /// `+`.
    Plus,
}

impl UnaryOp {
    /// The operator as the author spelled it.
    #[must_use]
    pub const fn text(self) -> &'static str {
        match self {
            Self::Not { word: true } => "not",
            Self::Not { word: false } => "!",
            Self::Negate => "-",
            Self::Plus => "+",
        }
    }
}

/// `op operand`.
#[derive(Debug, Clone, PartialEq)]
pub struct Unary {
    /// The operator.
    pub op: UnaryOp,
    /// Where the operator is written.
    pub op_span: Span,
    /// What it applies to.
    pub operand: Box<Expr>,
}

/// An ordering comparison, the operators that chain.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum CompareOp {
    /// `<`.
    Less,
    /// `>`.
    Greater,
    /// `<=`.
    LessEqual,
    /// `>=`.
    GreaterEqual,
}

impl CompareOp {
    /// The operator as written.
    #[must_use]
    pub const fn text(self) -> &'static str {
        match self {
            Self::Less => "<",
            Self::Greater => ">",
            Self::LessEqual => "<=",
            Self::GreaterEqual => ">=",
        }
    }
}

/// An infix operator.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum BinaryOp {
    /// `or` (`word`) or `||`.
    Or {
        /// Spelled `or`.
        word: bool,
    },
    /// `and` (`word`) or `&&`.
    And {
        /// Spelled `and`.
        word: bool,
    },
    /// `==`.
    Equal,
    /// `!=`.
    NotEqual,
    /// One ordering comparison; three or more operands are a [`Chain`].
    Compare(CompareOp),
    /// `in`.
    In,
    /// `matches`.
    Matches,
    /// `contains`.
    Contains,
    /// `startsWith`.
    StartsWith,
    /// `endsWith`.
    EndsWith,
    /// A word operator the schema publishes beyond the built-in four, at comparison precedence.
    Word(Box<str>),
    /// `..`.
    Range,
    /// `+`.
    Add,
    /// `-`.
    Subtract,
    /// `*`.
    Multiply,
    /// `/`.
    Divide,
    /// `%`.
    Modulo,
    /// `**` or `^` (`caret`).
    Power {
        /// Spelled `^`.
        caret: bool,
    },
    /// `??`.
    Coalesce,
}

impl BinaryOp {
    /// The operator as the author spelled it.
    #[must_use]
    pub fn text(&self) -> &str {
        match self {
            Self::Or { word: true } => "or",
            Self::Or { word: false } => "||",
            Self::And { word: true } => "and",
            Self::And { word: false } => "&&",
            Self::Equal => "==",
            Self::NotEqual => "!=",
            Self::Compare(op) => op.text(),
            Self::In => "in",
            Self::Matches => "matches",
            Self::Contains => "contains",
            Self::StartsWith => "startsWith",
            Self::EndsWith => "endsWith",
            Self::Word(word) => word,
            Self::Range => "..",
            Self::Add => "+",
            Self::Subtract => "-",
            Self::Multiply => "*",
            Self::Divide => "/",
            Self::Modulo => "%",
            Self::Power { caret: false } => "**",
            Self::Power { caret: true } => "^",
            Self::Coalesce => "??",
        }
    }

    /// Whether `not` may be written before it: `a not in b`.
    #[must_use]
    pub fn negatable(&self) -> bool {
        matches!(
            self,
            Self::In
                | Self::Matches
                | Self::Contains
                | Self::StartsWith
                | Self::EndsWith
                | Self::Word(_)
        )
    }
}

/// `left op right`, or `left not op right`.
#[derive(Debug, Clone, PartialEq)]
pub struct Binary {
    /// The operator.
    pub op: BinaryOp,
    /// Where the operator is written.
    pub op_span: Span,
    /// Where the `not` of `a not in b` is written; the node is then the negation.
    pub not: Option<Span>,
    /// The left operand.
    pub left: Box<Expr>,
    /// The right operand.
    pub right: Box<Expr>,
}

/// `a < b <= c`: every adjacent pair compared, all of which must hold.
#[derive(Debug, Clone, PartialEq)]
pub struct Chain {
    /// The operands, at least three; each middle one is compared on both sides.
    pub operands: Vec<Expr>,
    /// The operators between them, one fewer than the operands, with where each is written.
    pub operators: Vec<(CompareOp, Span)>,
}

/// `test ? then : otherwise`.
#[derive(Debug, Clone, PartialEq)]
pub struct Ternary {
    /// The test.
    pub test: Box<Expr>,
    /// The value when it holds; `None` for `test ?: otherwise`, where it is the test itself.
    pub then: Option<Box<Expr>>,
    /// The value otherwise.
    pub otherwise: Box<Expr>,
}

/// `if test { then } else { otherwise }`.
#[derive(Debug, Clone, PartialEq)]
pub struct If {
    /// The test.
    pub test: Box<Expr>,
    /// The block's value when it holds.
    pub then: Box<Expr>,
    /// The `else` block's value, or the `else if` that follows.
    pub otherwise: Box<Expr>,
}

/// One `key: value` of a map literal.
#[derive(Debug, Clone, PartialEq)]
pub struct MapEntry {
    /// The key: its text (a string key's value) and where the key token is.
    pub key: Name,
    /// The value.
    pub value: Expr,
}

/// A predicate.
#[derive(Debug, Clone, PartialEq)]
pub struct Predicate {
    /// What is worked out per item.
    pub body: Box<Expr>,
    /// Written in braces; otherwise it is the bare argument of a predicate function and shares
    /// its body's span.
    pub braces: bool,
}

/// `let` lines and the value they are scoped to.
#[derive(Debug, Clone, PartialEq)]
pub struct Sequence {
    /// The bindings, in order; each is in scope for the ones after it and the body.
    pub lets: Vec<Let>,
    /// The value.
    pub body: Box<Expr>,
}

/// A `let` inside parentheses, a block or a predicate.
#[derive(Debug, Clone, PartialEq)]
pub struct Let {
    /// From `let` to the end of the value; the `;` is not part of it.
    pub span: Span,
    /// The bound name; empty when not written yet.
    pub name: Name,
    /// The bound value.
    pub value: Expr,
}

/// A lexical or grammatical problem.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SyntaxDiagnostic {
    /// What it is about.
    pub span: Span,
    /// Whether the text is wrong, or past one of the parser's limits.
    pub code: SyntaxCode,
    /// What is wrong.
    pub message: Message,
    /// A one-line replacement of the span that fixes it, when there is one.
    pub fix: Option<SyntaxFix>,
}

/// The kinds of syntax diagnostic.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum SyntaxCode {
    /// The text does not follow the grammar.
    Syntax,
    /// The text is longer, deeper or larger than a formula may be.
    Limit,
}

/// A replacement for a syntax diagnostic's span.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SyntaxFix {
    /// What the fix does, for the button.
    pub label: Message,
    /// The text that replaces the span.
    pub insert: String,
}

/// The kinds of token.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum TokenKind {
    /// A number literal.
    Number,
    /// A text literal, closed or not.
    String,
    /// A name.
    Ident,
    /// A keyword or a word operator: `true`, `let`, `in`, `startsWith`, ...
    Keyword,
    /// An operator: `+`, `??`, `?.`, `==`, ...
    Operator,
    /// Punctuation: `( ) , ; [ ] { } .`.
    Punct,
    /// `#`, `#acc`, `#index` or `#name`.
    Pointer,
    /// A character the language has no use for.
    Error,
    /// The end of the text.
    Eof,
}

/// One token.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct Token {
    /// What it is.
    pub kind: TokenKind,
    /// Where it is.
    pub span: Span,
}

impl Expr {
    /// A node with no children.
    #[must_use]
    pub(crate) fn leaf(kind: ExprKind, span: Span) -> Self {
        Self {
            span,
            parens: 0,
            kind,
            depth: 1,
        }
    }

    /// Whether this is a part not written yet.
    #[must_use]
    pub const fn is_missing(&self) -> bool {
        matches!(self.kind, ExprKind::Missing)
    }

    /// The dotted path a name or member chain reads, such as `customer.name`, when it is one.
    #[must_use]
    pub fn path(&self) -> Option<String> {
        match &self.kind {
            ExprKind::Ident(name) => Some(name.clone()),
            ExprKind::Member(member) => member
                .object
                .path()
                .map(|object| format!("{object}.{}", member.property.text)),
            _ => None,
        }
    }

    /// The node's children, in source order.
    #[must_use]
    pub fn children(&self) -> Vec<&Expr> {
        let mut children = Vec::new();
        self.for_each_child(|child| children.push(child));
        children
    }

    /// Calls `visit` on each child, in source order.
    pub fn for_each_child<'a>(&'a self, mut visit: impl FnMut(&'a Expr)) {
        match &self.kind {
            ExprKind::Member(member) => visit(&member.object),
            ExprKind::Index(index) => {
                visit(&index.object);
                visit(&index.index);
            }
            ExprKind::Slice(slice) => {
                visit(&slice.object);
                if let Some(start) = &slice.start {
                    visit(start);
                }
                if let Some(end) = &slice.end {
                    visit(end);
                }
            }
            ExprKind::Call(call) => call.args.iter().for_each(visit),
            ExprKind::Method(method) => {
                visit(&method.receiver);
                method.args.iter().for_each(visit);
            }
            ExprKind::Unary(unary) => visit(&unary.operand),
            ExprKind::Binary(binary) => {
                visit(&binary.left);
                visit(&binary.right);
            }
            ExprKind::Chain(chain) => chain.operands.iter().for_each(visit),
            ExprKind::Ternary(ternary) => {
                visit(&ternary.test);
                if let Some(then) = &ternary.then {
                    visit(then);
                }
                visit(&ternary.otherwise);
            }
            ExprKind::If(branch) => {
                visit(&branch.test);
                visit(&branch.then);
                visit(&branch.otherwise);
            }
            ExprKind::Array(items) => items.iter().for_each(visit),
            ExprKind::Map(entries) => entries.iter().for_each(|entry| visit(&entry.value)),
            ExprKind::Predicate(predicate) => visit(&predicate.body),
            ExprKind::Sequence(sequence) => {
                sequence.lets.iter().for_each(|entry| visit(&entry.value));
                visit(&sequence.body);
            }
            ExprKind::Number(_)
            | ExprKind::String(_)
            | ExprKind::Bool(_)
            | ExprKind::Nil
            | ExprKind::Pointer(_)
            | ExprKind::Ident(_)
            | ExprKind::Missing => {}
        }
    }

    /// The height of the subtree rooted here: 1 for a leaf, never more than
    /// [`MAX_DEPTH`](super::MAX_DEPTH).
    #[must_use]
    pub const fn depth(&self) -> u16 {
        self.depth
    }
}

/// Visits `expr` and every node under it, parents before children.
pub fn walk<'a>(expr: &'a Expr, visit: &mut impl FnMut(&'a Expr)) {
    visit(expr);
    expr.for_each_child(|child| walk(child, visit));
}

impl Program {
    /// Visits every node of every statement, parents before children.
    pub fn walk<'a>(&'a self, visit: &mut impl FnMut(&'a Expr)) {
        for statement in &self.statements {
            walk(&statement.value, visit);
        }
    }

    /// The text of a token.
    #[must_use]
    pub fn token_text<'a>(&self, source: &'a str, token: Token) -> &'a str {
        source
            .get(token.span.start as usize..token.span.end as usize)
            .unwrap_or_default()
    }
}
