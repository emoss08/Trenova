//! The Tariff syntax tree.
//!
//! The tree is the contract between the parser, the checker, the evaluator and the editor
//! features: each of them reads it, none of them rewrites it. Every node carries the byte
//! [`Span`] of the text it was parsed from, and every expression a [`NodeId`] unique within its
//! file, so the checker can record types and resolutions in side tables the evaluator and the
//! editor look up.
//!
//! The parser is error-tolerant: a part the author has not written yet is a `Missing` node
//! ([`ExprKind::Missing`], [`TypeExprKind::Missing`], an empty [`Name`]), and a file whose tree
//! holds one also carries a syntax diagnostic explaining it. A tree is never deeper than the
//! parser's depth limit, so every walk over it may recurse.

use super::token::Token;
use crate::span::Span;
use crate::tariff::diagnostic::Diagnostic;
use rust_decimal::Decimal;

/// An expression's identity within its file, assigned by the parser in source order from 0.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord, Default)]
pub struct NodeId(pub u32);

/// What a file is, from its header.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum FileKind {
    /// `tariff "Name"`: inputs, bindings, charges, total clauses and tests; produces a bill.
    Tariff,
    /// `library "name"`: functions, constants, types, enums and tests; imported by others.
    Library,
    /// `condition "Name"`: inputs, bindings and functions, ending in one `Bool` expression.
    Condition,
}

impl FileKind {
    /// The header keyword.
    #[must_use]
    pub const fn keyword(self) -> &'static str {
        match self {
            Self::Tariff => "tariff",
            Self::Library => "library",
            Self::Condition => "condition",
        }
    }
}

/// A parsed file.
#[derive(Debug, Clone, PartialEq)]
pub struct File {
    /// The header, when the first line is one.
    pub header: Option<Header>,
    /// The doc comment that follows the header, describing the whole file.
    pub docs: Option<Doc>,
    /// Everything after the header, in source order.
    pub items: Vec<Item>,
    /// Lexical and grammatical problems, sorted by where they start.
    pub diagnostics: Vec<Diagnostic>,
    /// Every token, comments and newlines included, ending with one end-of-file token.
    pub tokens: Vec<Token>,
    /// How many [`NodeId`]s the parser assigned: every id is below this.
    pub node_count: u32,
}

/// `tariff "Dry van TL"`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Header {
    /// The file's kind.
    pub kind: FileKind,
    /// The name, as written between the quotes (escapes decoded, no interpolation).
    pub name: Name,
    /// From the keyword to the closing quote.
    pub span: Span,
}

/// A doc comment: one or more consecutive `///` lines, joined with newlines, the `///` and one
/// following space removed from each.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Doc {
    /// The text.
    pub text: String,
    /// From the first `///` to the end of the last line.
    pub span: Span,
}

/// A name as written, with where it is. Names, type names, variants and charge codes all use
/// it; an empty text with an empty span is a name the author has not written yet.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct Name {
    /// The identifier.
    pub text: String,
    /// Where it is written.
    pub span: Span,
}

/// One file-level declaration or statement.
#[derive(Debug, Clone, PartialEq)]
pub struct Item {
    /// The doc comment written directly above it.
    pub docs: Option<Doc>,
    /// The `private` keyword, when written.
    pub private: Option<Span>,
    /// From `private` (or the first keyword) to the end of the item; the doc is not included.
    pub span: Span,
    /// What the item is.
    pub kind: ItemKind,
}

/// The kinds of file-level item.
#[derive(Debug, Clone, PartialEq)]
pub enum ItemKind {
    /// `use std.ltl.{ deficit, Break }`.
    Use(Use),
    /// `input rate: Money/mi = $2.35/mi`.
    Input(Input),
    /// `const DIESEL_BASE: Money = $3.50`.
    Const(Const),
    /// `type Break = { from: Weight, rate: Money/cwt }`.
    Type(TypeDecl),
    /// `enum Zone { A, B, C }`.
    Enum(EnumDecl),
    /// `fn deadhead(empty: Distance, rate: Money/mi) -> Money { … }`.
    Fn(FnDecl),
    /// `test "name" { … }`.
    Test(Test),
    /// `property "name" { … }`: a test over many cases.
    Property(Property),
    /// Any statement written at file level: `let`, `charge`, `if`, a total clause, and a
    /// condition's final expression.
    Stmt(Stmt),
}

/// `use std.fuel` or `use std.fuel.{ stepPercent } as f`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Use {
    /// The module path, `std` or `org` first: `["std", "fuel"]`.
    pub path: Vec<Name>,
    /// The names imported into scope by `.{ … }`; `None` imports the module itself, read as
    /// `fuel.stepPercent(…)` (or by its alias).
    pub names: Option<Vec<Name>>,
    /// `as name`: what the module is called in this file.
    pub alias: Option<Name>,
}

impl Use {
    /// The path as written: `std.fuel`.
    #[must_use]
    pub fn dotted(&self) -> String {
        self.path
            .iter()
            .map(|part| part.text.as_str())
            .collect::<Vec<_>>()
            .join(".")
    }
}

/// `input name: Type = default`.
#[derive(Debug, Clone, PartialEq)]
pub struct Input {
    /// The input's name.
    pub name: Name,
    /// Its declared type.
    pub ty: TypeExpr,
    /// Its default value; without one, a value must be supplied.
    pub default: Option<Expr>,
    /// `where constraint`: a range or a list of the values a contract may set.
    pub constraint: Option<Expr>,
}

/// `const NAME: Type = value`.
#[derive(Debug, Clone, PartialEq)]
pub struct Const {
    /// The constant's name.
    pub name: Name,
    /// The declared type, required in libraries.
    pub ty: Option<TypeExpr>,
    /// The value, which must be computable without a record.
    pub value: Expr,
}

/// `type Name = { field: Type, … }`.
#[derive(Debug, Clone, PartialEq)]
pub struct TypeDecl {
    /// The type's name.
    pub name: Name,
    /// Its fields, in declaration order.
    pub fields: Vec<FieldType>,
}

/// `enum Name { A, B }`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct EnumDecl {
    /// The enum's name.
    pub name: Name,
    /// The variants, in declaration order, each with the doc comment written above it.
    pub variants: Vec<Variant>,
}

/// One enum variant.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Variant {
    /// Its doc comment.
    pub docs: Option<Doc>,
    /// Its name.
    pub name: Name,
}

/// `fn name(params) -> Type { … }` or `fn name(params) = expr`.
#[derive(Debug, Clone, PartialEq)]
pub struct FnDecl {
    /// The function's name.
    pub name: Name,
    /// Its parameters, in order.
    pub params: Vec<Param>,
    /// The declared return type; may be left out only with an `=` body, where it is inferred.
    pub ret: Option<TypeExpr>,
    /// The body.
    pub body: FnBody,
}

/// `name: Type = default`.
#[derive(Debug, Clone, PartialEq)]
pub struct Param {
    /// The parameter's name, also its name at a call site.
    pub name: Name,
    /// Its type.
    pub ty: TypeExpr,
    /// Its default value.
    pub default: Option<Expr>,
    /// From the name to the end of the default.
    pub span: Span,
}

/// A function body.
#[derive(Debug, Clone, PartialEq)]
pub enum FnBody {
    /// `{ … }`, which ends in `return`.
    Block(Block),
    /// `= expr`.
    Expr(Expr),
}

/// `test "name" { … }`.
#[derive(Debug, Clone, PartialEq)]
pub struct Test {
    /// The test's name.
    pub name: Name,
    /// Its `given`s, `expect`s and any other statements.
    pub body: Block,
}

/// `property "name" { … }`.
#[derive(Debug, Clone, PartialEq)]
pub struct Property {
    /// The property's name.
    pub name: Name,
    /// Its `vary`s, `assume`s, `given`s, `expect`s and any other statements.
    pub body: Block,
}

/// `{ … }`: a sequence of statements and a scope.
#[derive(Debug, Clone, PartialEq)]
pub struct Block {
    /// The statements, in order.
    pub stmts: Vec<Stmt>,
    /// From `{` to `}`.
    pub span: Span,
}

/// A statement.
#[derive(Debug, Clone, PartialEq)]
pub struct Stmt {
    /// From the first token to the last token of the statement; the ending newline is not part
    /// of it.
    pub span: Span,
    /// What the statement is.
    pub kind: StmtKind,
}

/// The kinds of statement.
#[derive(Debug, Clone, PartialEq)]
pub enum StmtKind {
    /// `let name: Type = value`.
    Let(Binding),
    /// `var name: Type = value`.
    Var(Binding),
    /// `name = value`, `name += value`, `name -= value`.
    Assign {
        /// The `var` assigned.
        target: Name,
        /// Which assignment.
        op: AssignOp,
        /// The value.
        value: Expr,
    },
    /// `charge "Label" as CODE (quantity: q, rate: r) = amount`.
    Charge(Charge),
    /// `note "Text"`.
    Note(Text),
    /// `review "Text"`: the rating bills, and asks to be approved before it is used.
    Review(Text),
    /// `if … { … } else if … { … } else { … }`.
    If(IfStmt),
    /// `best "Label" { option "A" { … } … }` or `worst "Label" { … }`.
    Choose(Choose),
    /// `match value { pattern => { … } }`.
    Match(MatchStmt),
    /// `for x in list where cond { … }`.
    For(ForStmt),
    /// `require cond else "message"` or `require let x = optional else "message"`.
    Require {
        /// What must hold.
        condition: Condition,
        /// Why the rating is refused when it does not.
        message: Text,
    },
    /// `fail "message"`.
    Fail(Text),
    /// `return value`.
    Return(Expr),
    /// `break`.
    Break,
    /// `continue`.
    Continue,
    /// `minimum`, `maximum` or `round`.
    Total(TotalClause),
    /// `given shipment.weight = 1_240 lb`, or a rate table's cell `given tables.Zones["A",
    /// "B"] = $20/cwt`, in tests and properties.
    Given {
        /// The path overridden: a schema root's field (`shipment.weight`), `input.name`, or
        /// `tables.Code` for a cell.
        target: Vec<Name>,
        /// A cell's keys, one per axis, when the target is a table: `["A", "B"]`.
        keys: Option<Vec<Expr>>,
        /// The value it takes.
        value: Expr,
    },
    /// `vary shipment.weight in 1 lb..40_000 lb`, in properties.
    Vary {
        /// The path varied (`shipment.weight`, `input.rate`) or a new name (`w`).
        target: Vec<Name>,
        /// `in values`: a range or a list; `None` takes the values from the target's type or
        /// constraint.
        values: Option<Expr>,
    },
    /// `assume cond`, in properties: cases where it is false are skipped.
    Assume(Expr),
    /// `expect cond`, `expect fails "message"` or `expect x nondecreasing in target`, in tests
    /// and properties.
    Expect(Expect),
    /// An expression on its own: a condition's final value, or a call whose result is unused.
    Expr(Expr),
    /// A file-level declaration (`fn`, `const`, `use`, …) written inside a block, where it is
    /// misplaced. The parser reports it; the tree keeps it so later phases and the editor can
    /// still see it.
    Item(Box<Item>),
}

/// The name, annotation and value of a `let` or `var`.
#[derive(Debug, Clone, PartialEq)]
pub struct Binding {
    /// The bound name.
    pub name: Name,
    /// The annotated type, when written.
    pub ty: Option<TypeExpr>,
    /// The value.
    pub value: Expr,
}

/// Which assignment.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum AssignOp {
    /// `=`.
    Set,
    /// `+=`.
    Add,
    /// `-=`.
    Sub,
}

/// `charge "Label" as CODE (options) = amount`.
#[derive(Debug, Clone, PartialEq)]
pub struct Charge {
    /// The line's label, which may interpolate.
    pub label: Text,
    /// `as CODE`: an accessorial charge code. Without one, the line is freight.
    pub code: Option<Name>,
    /// The `(quantity: q, rate: r)` options, as named arguments; empty when not written.
    pub options: Vec<Arg>,
    /// `= amount`; may be left out only with both `quantity` and `rate`.
    pub amount: Option<Expr>,
}

/// A choice between methods: `best` or `worst`.
#[derive(Debug, Clone, PartialEq)]
pub struct Choose {
    /// Which option wins.
    pub kind: ChooseKind,
    /// The choice's label, as written between the quotes.
    pub label: Name,
    /// The options, in order.
    pub options: Vec<ChoiceOption>,
    /// From `best` or `worst` to the closing brace.
    pub span: Span,
}

/// Which option a choice keeps.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum ChooseKind {
    /// `best`: the lowest subtotal.
    Best,
    /// `worst`: the highest subtotal.
    Worst,
}

impl ChooseKind {
    /// The keyword.
    #[must_use]
    pub const fn keyword(self) -> &'static str {
        match self {
            Self::Best => "best",
            Self::Worst => "worst",
        }
    }
}

/// `option "Label" { … }` inside a choice.
#[derive(Debug, Clone, PartialEq)]
pub struct ChoiceOption {
    /// The option's label, as written between the quotes.
    pub label: Name,
    /// What it bills.
    pub body: Block,
    /// From `option` to the closing brace.
    pub span: Span,
}

/// An `if` statement.
#[derive(Debug, Clone, PartialEq)]
pub struct IfStmt {
    /// The `if` and each `else if`, with its block.
    pub branches: Vec<(Condition, Block)>,
    /// The final `else` block.
    pub otherwise: Option<Block>,
}

/// What an `if` or `require` tests.
#[derive(Debug, Clone, PartialEq)]
pub enum Condition {
    /// A `Bool` expression.
    Expr(Expr),
    /// `let name = optional`: holds when the optional is present, binding it.
    Let {
        /// The name bound to the present value.
        name: Name,
        /// The optional.
        value: Expr,
    },
}

/// A `match` statement.
#[derive(Debug, Clone, PartialEq)]
pub struct MatchStmt {
    /// The value matched.
    pub scrutinee: Expr,
    /// The arms, in order.
    pub arms: Vec<Arm>,
    /// From `match` to `}`.
    pub span: Span,
}

/// `pattern if guard => body`.
#[derive(Debug, Clone, PartialEq)]
pub struct Arm {
    /// What the arm matches.
    pub pattern: Pattern,
    /// `if guard`.
    pub guard: Option<Expr>,
    /// What it runs or gives.
    pub body: ArmBody,
    /// From the pattern to the end of the body.
    pub span: Span,
}

/// The body of a `match` arm.
#[derive(Debug, Clone, PartialEq)]
pub enum ArmBody {
    /// `{ … }`, in a `match` statement.
    Block(Block),
    /// An expression: the arm's value in a `match` expression, or a lone expression statement.
    Expr(Expr),
}

/// `for binding in iterable where filter { … }`.
#[derive(Debug, Clone, PartialEq)]
pub struct ForStmt {
    /// The loop variable.
    pub binding: Name,
    /// The list or `Int` range walked.
    pub iterable: Expr,
    /// `where cond`.
    pub filter: Option<Expr>,
    /// The body.
    pub body: Block,
}

/// A total clause.
#[derive(Debug, Clone, PartialEq)]
pub enum TotalClause {
    /// `minimum amount` or `minimum amount on freight`.
    Minimum(TotalBound),
    /// `maximum amount` or `maximum amount on freight`.
    Maximum(TotalBound),
    /// `round [half even | up | down] to (cents | $X)`.
    Round {
        /// How.
        mode: RoundMode,
        /// To what.
        to: RoundTo,
    },
}

/// The amount of a `minimum` or `maximum` clause and what it bounds.
#[derive(Debug, Clone, PartialEq)]
pub struct TotalBound {
    /// The amount.
    pub amount: Expr,
    /// `on freight`: the name after `on`, when written; without it the clause bounds the whole
    /// bill. The checker accepts only `freight`.
    pub on: Option<Name>,
}

impl TotalBound {
    /// Whether the clause bounds the freight lines only.
    #[must_use]
    pub fn on_freight(&self) -> bool {
        self.on.is_some()
    }
}

/// How a `round` clause rounds.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Default)]
pub enum RoundMode {
    /// Half away from zero: the default, written with no mode.
    #[default]
    HalfUp,
    /// `half even`: banker's rounding.
    HalfEven,
    /// `up`: away from zero.
    Up,
    /// `down`: toward zero.
    Down,
}

/// What a `round` clause rounds to.
#[derive(Debug, Clone, PartialEq)]
pub enum RoundTo {
    /// `cents`.
    Cents,
    /// A money amount: `$5`.
    Amount(Expr),
}

/// `expect …` in a test.
#[derive(Debug, Clone, PartialEq)]
pub enum Expect {
    /// `expect cond`: a `Bool` that must hold.
    Holds(Expr),
    /// `expect fails` or `expect fails "message"`: the rating must be refused, with that
    /// message when one is given.
    Fails(Option<Text>),
    /// `expect value nondecreasing in target`, in properties: `value` must not fall (or, for
    /// `nonincreasing`, rise) as the varied `target` grows.
    Monotone {
        /// The ordered value compared across cases.
        value: Expr,
        /// Which way it must go.
        direction: Direction,
        /// Where the direction word is written.
        direction_span: Span,
        /// The varied path or name it is compared along.
        target: Vec<Name>,
    },
}

/// The direction of a monotone expectation.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Direction {
    /// `nondecreasing`.
    Nondecreasing,
    /// `nonincreasing`.
    Nonincreasing,
}

impl Direction {
    /// The word as written.
    #[must_use]
    pub const fn word(self) -> &'static str {
        match self {
            Self::Nondecreasing => "nondecreasing",
            Self::Nonincreasing => "nonincreasing",
        }
    }
}

/// A text literal: `"Detention, stop {stop.sequence}"`.
#[derive(Debug, Clone, PartialEq)]
pub struct Text {
    /// The literal and interpolated parts, in order; adjacent literal parts are merged.
    pub parts: Vec<TextPart>,
    /// From the opening quote to the closing quote.
    pub span: Span,
}

impl Text {
    /// The text when it has no interpolation.
    #[must_use]
    pub fn literal(&self) -> Option<String> {
        let mut out = String::new();
        for part in &self.parts {
            match part {
                TextPart::Literal(text, _) => out.push_str(text),
                TextPart::Expr(_) => return None,
            }
        }
        Some(out)
    }
}

/// One part of a text literal.
#[derive(Debug, Clone, PartialEq)]
pub enum TextPart {
    /// Characters, escapes decoded, with where they are written: the span covers the raw text
    /// (escapes as written, quotes excluded).
    Literal(String, Span),
    /// `{expr}`.
    Expr(Expr),
}

/// A date literal: `@2026-10-22`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct DateLit {
    /// The year.
    pub year: i32,
    /// The month, 1 to 12.
    pub month: u8,
    /// The day of the month, valid for the month.
    pub day: u8,
}

/// A date-time literal: `@2026-10-22T08:00-05:00`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct DateTimeLit {
    /// The local date.
    pub date: DateLit,
    /// The local hour, 0 to 23.
    pub hour: u8,
    /// The minute.
    pub minute: u8,
    /// The second; 0 when not written.
    pub second: u8,
    /// The UTC offset in minutes; 0 for `Z`.
    pub offset_minutes: i16,
}

/// A unit as written after a number, after `$x/` or in a type: `lb`, `ft3`, `lb/ft3`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct UnitExpr {
    /// The unit names, the first in the numerator and the rest in the denominator.
    pub parts: Vec<Name>,
    /// From the first name to the last.
    pub span: Span,
}

impl UnitExpr {
    /// The unit as written, without spaces: `lb/ft3`.
    #[must_use]
    pub fn text(&self) -> String {
        self.parts
            .iter()
            .map(|part| part.text.as_str())
            .collect::<Vec<_>>()
            .join("/")
    }
}

/// An expression node.
#[derive(Debug, Clone, PartialEq)]
pub struct Expr {
    /// The node's identity within its file.
    pub id: NodeId,
    /// The text the node covers, including the parentheses the author put around it.
    pub span: Span,
    /// How many pairs of parentheses the author wrote around this node.
    pub parens: u8,
    /// What the node is.
    pub kind: ExprKind,
}

/// The kinds of expression.
#[derive(Debug, Clone, PartialEq)]
pub enum ExprKind {
    /// `1200`, `2.35`, `500 lb`, `30 min`. `integer` is true when no `.` was written; a count
    /// unit (`3 pallet`) leaves it an integer.
    Number {
        /// The value, `_` removed.
        value: Decimal,
        /// No decimal point was written.
        integer: bool,
        /// The unit written after it.
        unit: Option<UnitExpr>,
    },
    /// `$450`, `$2.35/mi`.
    Money {
        /// The amount.
        value: Decimal,
        /// The unit after `/`, for a rate.
        per: Option<UnitExpr>,
    },
    /// `18%`, holding 18.
    Percent(Decimal),
    /// A text literal.
    Text(Text),
    /// `@2026-10-22`.
    Date(DateLit),
    /// `@2026-10-22T08:00-05:00`.
    DateTime(DateTimeLit),
    /// `true` or `false`.
    Bool(bool),
    /// `none`.
    None,
    /// A `camelCase` name: a binding, input, parameter, function, constant, module or root.
    Name(Name),
    /// A `PascalCase` name, optionally followed by `.member`: an enum variant (`Delivery`,
    /// `StopKind.Delivery`), or a type used for a conversion (`Number(x)`, `Int.parse(text)`).
    Path {
        /// The `PascalCase` name.
        head: Name,
        /// The `.member` after it.
        member: Option<Name>,
    },
    /// `target.name` or `target?.name`.
    Field {
        /// What is read.
        target: Box<Expr>,
        /// The field, property or method.
        name: Name,
        /// Written `?.`.
        optional: bool,
    },
    /// `callee(args)`: a function (`deficit(…)`), a method (`target.round(…)`, whose callee is
    /// a [`ExprKind::Field`]), a module function (`fuel.stepPercent(…)`) or a conversion.
    Call {
        /// What is called.
        callee: Box<Expr>,
        /// The arguments.
        args: Vec<Arg>,
    },
    /// `target[key, …]`: a rate table cell or a list item.
    Index {
        /// What is indexed.
        target: Box<Expr>,
        /// The keys.
        keys: Vec<Expr>,
    },
    /// `-x` or `not x`.
    Unary {
        /// Which operator.
        op: UnaryOp,
        /// Where the operator is written.
        op_span: Span,
        /// The operand.
        operand: Box<Expr>,
    },
    /// `left op right`.
    Binary {
        /// Which operator.
        op: BinaryOp,
        /// Where the operator is written.
        op_span: Span,
        /// The left operand.
        left: Box<Expr>,
        /// The right operand.
        right: Box<Expr>,
    },
    /// `start..end`, `start..=end`, `..end`, `start..`.
    Range {
        /// The start, when written.
        start: Option<Box<Expr>>,
        /// The end, when written.
        end: Option<Box<Expr>>,
        /// Written `..=`.
        inclusive: bool,
    },
    /// `fail "message"`, only as the right side of `??`.
    Fail(Text),
    /// `[a, b]`.
    List(Vec<Expr>),
    /// `{ name: value, … }`.
    Record(Vec<FieldInit>),
    /// `if cond { a } else if cond { b } else { c }`.
    If {
        /// The `if` and each `else if`, with its value.
        branches: Vec<(Condition, Expr)>,
        /// The `else` value.
        otherwise: Box<Expr>,
    },
    /// `match value { pattern => expr }`.
    Match {
        /// The value matched.
        scrutinee: Box<Expr>,
        /// The arms, whose bodies are [`ArmBody::Expr`].
        arms: Vec<Arm>,
    },
    /// `best "Label" { … }` or `worst "Label" { … }` as a value: the kept option's subtotal.
    /// The checker allows it only as the whole value of a `let`, `var` or assignment.
    Choose(Box<Choose>),
    /// `x => body` or `(a, b) => body`, only as an argument to a list method.
    Lambda {
        /// The parameters.
        params: Vec<Name>,
        /// The body.
        body: Box<Expr>,
    },
    /// A part the author has not written yet.
    Missing,
}

/// A call argument: `x`, or `name: x`.
#[derive(Debug, Clone, PartialEq)]
pub struct Arg {
    /// The parameter named at the call site.
    pub name: Option<Name>,
    /// The value.
    pub value: Expr,
    /// From the name (or value) to the end of the value.
    pub span: Span,
}

/// `name: value` in a record literal.
#[derive(Debug, Clone, PartialEq)]
pub struct FieldInit {
    /// The field.
    pub name: Name,
    /// Its value.
    pub value: Expr,
}

/// A unary operator.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum UnaryOp {
    /// `-`.
    Neg,
    /// `not`.
    Not,
}

/// A binary operator.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum BinaryOp {
    /// `??`.
    Coalesce,
    /// `or`.
    Or,
    /// `and`.
    And,
    /// `==`.
    Eq,
    /// `!=`.
    Ne,
    /// `<`.
    Lt,
    /// `<=`.
    Le,
    /// `>`.
    Gt,
    /// `>=`.
    Ge,
    /// `in`.
    In,
    /// `not in`.
    NotIn,
    /// `+`.
    Add,
    /// `-`.
    Sub,
    /// `*`.
    Mul,
    /// `/`.
    Div,
    /// `%`.
    Rem,
}

impl BinaryOp {
    /// The operator as written.
    #[must_use]
    pub const fn symbol(self) -> &'static str {
        match self {
            Self::Coalesce => "??",
            Self::Or => "or",
            Self::And => "and",
            Self::Eq => "==",
            Self::Ne => "!=",
            Self::Lt => "<",
            Self::Le => "<=",
            Self::Gt => ">",
            Self::Ge => ">=",
            Self::In => "in",
            Self::NotIn => "not in",
            Self::Add => "+",
            Self::Sub => "-",
            Self::Mul => "*",
            Self::Div => "/",
            Self::Rem => "%",
        }
    }
}

/// A `match` pattern.
#[derive(Debug, Clone, PartialEq)]
pub struct Pattern {
    /// Where it is written.
    pub span: Span,
    /// What it matches.
    pub kind: PatternKind,
}

/// The kinds of pattern.
#[derive(Debug, Clone, PartialEq)]
pub enum PatternKind {
    /// `_`.
    Wildcard,
    /// `none`.
    None,
    /// A literal: a number with its unit, money, a percent, text, a date, `true`, `false`. A
    /// negative number is a [`ExprKind::Unary`] negation of a literal.
    Literal(Expr),
    /// `lo..hi`, `lo..=hi`, `..hi`, `lo..`, with literal ends.
    Range {
        /// The start, when written.
        start: Option<Expr>,
        /// The end, when written.
        end: Option<Expr>,
        /// Written `..=`.
        inclusive: bool,
    },
    /// An enum variant: `Delivery`, or `StopKind.Delivery`.
    Variant {
        /// The enum, when written.
        enumeration: Option<Name>,
        /// The variant.
        variant: Name,
    },
    /// A `camelCase` name: matches anything and binds it.
    Binding(Name),
    /// `{ kind: Delivery, appointment: true }`: the record's named fields match their patterns.
    Record(Vec<(Name, Pattern)>),
    /// `A | B`.
    Or(Vec<Pattern>),
    /// A pattern the author has not written yet.
    Missing,
}

/// A type as written.
#[derive(Debug, Clone, PartialEq)]
pub struct TypeExpr {
    /// Where it is written.
    pub span: Span,
    /// What it is.
    pub kind: TypeExprKind,
}

/// The kinds of type expression.
#[derive(Debug, Clone, PartialEq)]
pub enum TypeExprKind {
    /// `Bool`, `Weight`, `Break`, `List<Stop>`. Imported types are named bare, never by path.
    Named {
        /// The type's name.
        name: Name,
        /// `<T>` arguments.
        args: Vec<TypeExpr>,
    },
    /// `Money/cwt`, `Money/Weight`, `Weight/Volume`, `lb/ft3`: a quotient whose sides are
    /// each a dimension name or a unit.
    Quotient {
        /// The numerator: a dimension (`Money`, `Weight`) or a unit (`lb`).
        numerator: Name,
        /// The denominator: a dimension or a unit.
        denominator: Name,
    },
    /// `{ field: Type, … }`.
    Record(Vec<FieldType>),
    /// `T?`.
    Optional(Box<TypeExpr>),
    /// A type the author has not written yet.
    Missing,
}

/// `name: Type` in a record type, with its doc comment.
#[derive(Debug, Clone, PartialEq)]
pub struct FieldType {
    /// Its doc comment.
    pub docs: Option<Doc>,
    /// The field's name.
    pub name: Name,
    /// Its type.
    pub ty: TypeExpr,
}
