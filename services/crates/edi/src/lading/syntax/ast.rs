//! The syntax tree of a Lading file.
//!
//! Every node keeps its byte span. The parser always produces a tree, with `Error` nodes where
//! it could not read, so the checker and the editor work on a file being typed.

use trenova_module_kit::span::Span;

/// A name and where it is written.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Ident {
    /// The name.
    pub name: String,
    /// Where it is.
    pub span: Span,
}

impl Ident {
    /// A name at a span.
    #[must_use]
    pub fn new(name: impl Into<String>, span: Span) -> Self {
        Self {
            name: name.into(),
            span,
        }
    }
}

/// One parsed file.
#[derive(Debug, Clone, PartialEq)]
pub struct File {
    /// The header: what kind of file it is and its name.
    pub header: Header,
    /// Its declarations and statements, in order.
    pub items: Vec<Item>,
    /// The whole file.
    pub span: Span,
    /// Every keyword the parser read, for highlighting.
    pub keywords: Vec<Span>,
}

/// What kind of file, by its first line.
#[derive(Debug, Clone, PartialEq)]
pub enum Header {
    /// `partner name "Label"`
    Partner {
        /// Its name.
        name: Ident,
        /// What people call it.
        label: Option<TextLit>,
    },
    /// `guide name from base "Title"`
    Guide {
        /// Its dotted name.
        name: Ident,
        /// The guide it narrows.
        base: Ident,
        /// Its title.
        title: Option<TextLit>,
    },
    /// `map name from source to target use libs`
    Map {
        /// Its name.
        name: Ident,
        /// What it reads.
        from: Endpoint,
        /// What it writes.
        to: Endpoint,
        /// The libraries it uses.
        uses: Vec<Ident>,
    },
    /// `rules name for map`
    Rules {
        /// Its name.
        name: Ident,
        /// The map it checks.
        map: Ident,
        /// The libraries it uses.
        uses: Vec<Ident>,
    },
    /// `flow name with partner`
    Flow {
        /// Its name.
        name: Ident,
        /// The partner it is with.
        partner: Ident,
        /// The libraries it uses.
        uses: Vec<Ident>,
    },
    /// `library name`
    Library {
        /// Its name.
        name: Ident,
        /// The libraries it uses.
        uses: Vec<Ident>,
    },
    /// No header could be read.
    Missing(Span),
}

impl Header {
    /// The file's own name, when it has one.
    #[must_use]
    pub fn name(&self) -> Option<&Ident> {
        match self {
            Self::Partner { name, .. }
            | Self::Guide { name, .. }
            | Self::Map { name, .. }
            | Self::Rules { name, .. }
            | Self::Flow { name, .. }
            | Self::Library { name, .. } => Some(name),
            Self::Missing(_) => None,
        }
    }

    /// The libraries it uses.
    #[must_use]
    pub fn uses(&self) -> &[Ident] {
        match self {
            Self::Map { uses, .. }
            | Self::Rules { uses, .. }
            | Self::Flow { uses, .. }
            | Self::Library { uses, .. } => uses,
            _ => &[],
        }
    }

    /// The kind's keyword.
    #[must_use]
    pub fn kind(&self) -> &'static str {
        match self {
            Self::Partner { .. } => "partner",
            Self::Guide { .. } => "guide",
            Self::Map { .. } => "map",
            Self::Rules { .. } => "rules",
            Self::Flow { .. } => "flow",
            Self::Library { .. } => "library",
            Self::Missing(_) => "",
        }
    }
}

/// What a map reads or writes.
#[derive(Debug, Clone, PartialEq)]
pub enum Endpoint {
    /// A guide (`x12.004010.204`, `acme.204`).
    Guide(Ident),
    /// A document type (`loadTender`).
    Document(Ident),
    /// A CSV file with typed columns.
    Csv {
        /// The columns.
        columns: Vec<CsvColumn>,
        /// Where the `csv { ... }` is.
        span: Span,
    },
    /// Any JSON document.
    Json(Span),
    /// Nothing could be read.
    Missing(Span),
}

impl Endpoint {
    /// Where it is written.
    #[must_use]
    pub fn span(&self) -> Span {
        match self {
            Self::Guide(ident) | Self::Document(ident) => ident.span,
            Self::Csv { span, .. } | Self::Json(span) | Self::Missing(span) => *span,
        }
    }
}

/// One CSV column.
#[derive(Debug, Clone, PartialEq)]
pub struct CsvColumn {
    /// Its header name.
    pub name: Ident,
    /// Its type.
    pub ty: TypeExpr,
    /// Its format (`"MM/dd/yyyy"`), unit or currency.
    pub format: Option<Expr>,
}

/// A text literal, decoded, with its span.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TextLit {
    /// The text.
    pub value: String,
    /// Where it is.
    pub span: Span,
}

/// A written type.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TypeExpr {
    /// What it is.
    pub kind: TypeExprKind,
    /// Where it is.
    pub span: Span,
}

/// The forms of a written type.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum TypeExprKind {
    /// A named type, with its arguments (`Code<S502>`, `Segment<G62>`).
    Named(String, Vec<String>),
    /// `[T]`
    List(Box<TypeExpr>),
    /// `[Text: T]`
    Map(Box<TypeExpr>),
    /// `T?`
    Optional(Box<TypeExpr>),
    /// Could not be read.
    Error,
}

/// A doc comment's text.
pub type Doc = Option<String>;

/// One top-level item.
#[derive(Debug, Clone, PartialEq)]
pub enum Item {
    /// `fn`
    Fn(FnDecl),
    /// `crosswalk`
    Crosswalk(CrosswalkDecl),
    /// `const`
    Const(ConstDecl),
    /// `test`
    Test(TestDecl),
    /// `property`
    Property(PropertyDecl),
    /// `rule`
    Rule(RuleDecl),
    /// A map statement.
    Stmt(Stmt),
    /// `interchange { ... }`
    Interchange(Interchange),
    /// `zone ...`
    Zone(Ident),
    /// `currency "..."`
    Currency(TextLit),
    /// `setting ...`
    Setting(SettingDecl),
    /// A guide line.
    Guide(GuideDecl),
    /// A flow clause.
    Flow(FlowDecl),
    /// Something that could not be read.
    Error(Span),
}

impl Item {
    /// Where it is.
    #[must_use]
    pub fn span(&self) -> Span {
        match self {
            Self::Fn(item) => item.span,
            Self::Crosswalk(item) => item.span,
            Self::Const(item) => item.span,
            Self::Test(item) => item.span,
            Self::Property(item) => item.span,
            Self::Rule(item) => item.span,
            Self::Stmt(item) => item.span(),
            Self::Interchange(item) => item.span,
            Self::Zone(item) => item.span,
            Self::Currency(item) => item.span,
            Self::Setting(item) => item.span,
            Self::Guide(item) => item.span(),
            Self::Flow(item) => item.span(),
            Self::Error(span) => *span,
        }
    }
}

/// A function parameter.
#[derive(Debug, Clone, PartialEq)]
pub struct Param {
    /// Its name.
    pub name: Ident,
    /// Its type.
    pub ty: TypeExpr,
    /// Its default.
    pub default: Option<Expr>,
}

/// `fn name(params) -> T = expr` or `{ ... }`.
#[derive(Debug, Clone, PartialEq)]
pub struct FnDecl {
    /// Its doc comment.
    pub doc: Doc,
    /// Its name.
    pub name: Ident,
    /// Its parameters.
    pub params: Vec<Param>,
    /// What it returns.
    pub ret: TypeExpr,
    /// Its body: `let`s then a value.
    pub body: Body,
    /// The whole declaration.
    pub span: Span,
}

/// A body of `let`s ending in a value.
#[derive(Debug, Clone, PartialEq)]
pub struct Body {
    /// The bindings.
    pub lets: Vec<LetStmt>,
    /// The value.
    pub value: Expr,
}

/// A direction of a crosswalk pair.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Direction {
    /// `<->`
    Both,
    /// `->`, left to right only.
    Forward,
    /// `<-`, right to left only.
    Backward,
}

/// One crosswalk pair.
#[derive(Debug, Clone, PartialEq)]
pub struct Pair {
    /// The left values; the first is written.
    pub left: Vec<Expr>,
    /// Which way it works.
    pub direction: Direction,
    /// The right values; the first is written.
    pub right: Vec<Expr>,
    /// The whole pair.
    pub span: Span,
}

/// `crosswalk name: A <-> B { ... }`.
#[derive(Debug, Clone, PartialEq)]
pub struct CrosswalkDecl {
    /// Its doc comment.
    pub doc: Doc,
    /// Its name.
    pub name: Ident,
    /// The left type.
    pub left: TypeExpr,
    /// The right type.
    pub right: TypeExpr,
    /// The pairs.
    pub pairs: Vec<Pair>,
    /// `else -> v` or `else <- v`.
    pub otherwise: Vec<(Direction, Expr)>,
    /// The whole declaration.
    pub span: Span,
}

/// `const name = expr`.
#[derive(Debug, Clone, PartialEq)]
pub struct ConstDecl {
    /// Its doc comment.
    pub doc: Doc,
    /// Its name.
    pub name: Ident,
    /// Its value.
    pub value: Expr,
    /// The whole declaration.
    pub span: Span,
}

/// `setting name: T = default in lo..hi`.
#[derive(Debug, Clone, PartialEq)]
pub struct SettingDecl {
    /// Its doc comment, its label.
    pub doc: Doc,
    /// Its name.
    pub name: Ident,
    /// Its type.
    pub ty: TypeExpr,
    /// Its default.
    pub default: Option<Expr>,
    /// Its range.
    pub range: Option<(Expr, Expr)>,
    /// The whole declaration.
    pub span: Span,
}

/// One party of an interchange: a qualifier and an ID.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct InterchangeParty {
    /// The ISA qualifier (`ZZ`, `01`).
    pub qualifier: Ident,
    /// The ID.
    pub id: TextLit,
}

/// `interchange { ... }`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Interchange {
    /// The partner.
    pub them: Option<InterchangeParty>,
    /// The organisation.
    pub us: Option<InterchangeParty>,
    /// The GS application codes, theirs then ours.
    pub group: Option<(TextLit, TextLit)>,
    /// The X12 version.
    pub version: Option<Ident>,
    /// The separators: element, segment, component, repetition.
    pub separators: Vec<TextLit>,
    /// `test` or `production`.
    pub usage: Option<Ident>,
    /// The whole block.
    pub span: Span,
}

/// A `let` binding.
#[derive(Debug, Clone, PartialEq)]
pub struct LetStmt {
    /// The name.
    pub name: Ident,
    /// The value.
    pub value: Expr,
    /// The whole line.
    pub span: Span,
}

/// One step of an assignment's target.
#[derive(Debug, Clone, PartialEq)]
pub enum LvStep {
    /// `.name`
    Field(Ident),
    /// `[expr]`
    Index(Expr),
}

/// The target of an assignment or fix: `stops`, `refs["PO"]`, `stop.address.postal`.
#[derive(Debug, Clone, PartialEq)]
pub struct LValue {
    /// The first name.
    pub root: Ident,
    /// The steps after it.
    pub steps: Vec<LvStep>,
    /// The whole target.
    pub span: Span,
}

/// What an `else` gives.
#[derive(Debug, Clone, PartialEq)]
pub enum Else {
    /// A default value.
    Value(Expr),
    /// An outcome.
    Outcome(Outcome),
}

/// What kind of outcome.
#[derive(Debug, Clone, PartialEq)]
pub enum OutcomeKind {
    /// `reject [at path]`
    Reject(Option<Expr>),
    /// `decline [code "X"]`
    Decline(Option<Expr>),
    /// `hold`
    Hold,
    /// `warn`
    Warn,
}

impl OutcomeKind {
    /// Its keyword.
    #[must_use]
    pub fn keyword(&self) -> &'static str {
        match self {
            Self::Reject(_) => "reject",
            Self::Decline(_) => "decline",
            Self::Hold => "hold",
            Self::Warn => "warn",
        }
    }
}

/// An outcome: what happens when a rule fails.
#[derive(Debug, Clone, PartialEq)]
pub struct Outcome {
    /// Which.
    pub kind: OutcomeKind,
    /// Its message.
    pub message: Expr,
    /// The whole outcome.
    pub span: Span,
}

/// One element assignment in a segment block: `01 <- expr`, `03.02 <- expr`.
#[derive(Debug, Clone, PartialEq)]
pub struct ElementAssign {
    /// The element position, and the component position for a composite.
    pub position: (u32, Option<u32>),
    /// Where the position is written.
    pub position_span: Span,
    /// The value.
    pub value: Expr,
    /// The whole assignment.
    pub span: Span,
}

/// A block of statements.
#[derive(Debug, Clone, PartialEq)]
pub struct Block {
    /// The statements.
    pub stmts: Vec<Stmt>,
    /// The braces and all.
    pub span: Span,
}

/// A statement of a map, rule or function body.
#[derive(Debug, Clone, PartialEq)]
pub enum Stmt {
    /// `target <- value [else ...]`
    Assign {
        /// What is assigned.
        target: LValue,
        /// The value.
        value: Expr,
        /// The `else`.
        otherwise: Option<Else>,
        /// The whole line.
        span: Span,
    },
    /// `let name = value`
    Let(LetStmt),
    /// `when cond { ... }`
    When {
        /// The condition.
        cond: Expr,
        /// What runs when it holds.
        body: Block,
        /// The whole statement.
        span: Span,
    },
    /// `SEG [when cond] { 01 <- ... }`
    Segment {
        /// The segment ID.
        id: Ident,
        /// The condition.
        cond: Option<Expr>,
        /// The elements.
        elements: Vec<ElementAssign>,
        /// The whole block.
        span: Span,
    },
    /// `LOOP { ... }` or `LOOP <- each list as x { ... }`
    Loop {
        /// The loop ID.
        id: Ident,
        /// The list and the binder, for one instance per item.
        each: Option<(Expr, Ident)>,
        /// The loop's contents.
        body: Block,
        /// The whole block.
        span: Span,
    },
    /// `each list as x { ... }` at statement level.
    Each {
        /// The list.
        list: Expr,
        /// The binder.
        binder: Ident,
        /// What runs per item.
        body: Block,
        /// The whole statement.
        span: Span,
    },
    /// `require cond else outcome`
    Require {
        /// What must hold.
        cond: Expr,
        /// What happens when it does not.
        outcome: Outcome,
        /// The whole statement.
        span: Span,
    },
    /// `outcome when cond`
    OutcomeWhen {
        /// What happens.
        outcome: Outcome,
        /// When.
        cond: Expr,
        /// The whole statement.
        span: Span,
    },
    /// `for x in list where cond { ... }`
    For {
        /// The binder.
        binder: Ident,
        /// The list.
        list: Expr,
        /// The filter.
        filter: Option<Expr>,
        /// The statements.
        body: Block,
        /// The whole statement.
        span: Span,
    },
    /// `fix target <- value`
    Fix {
        /// What is changed.
        target: LValue,
        /// The new value.
        value: Expr,
        /// The whole statement.
        span: Span,
    },
    /// Something that could not be read.
    Error(Span),
}

impl Stmt {
    /// Where it is.
    #[must_use]
    pub fn span(&self) -> Span {
        match self {
            Self::Assign { span, .. }
            | Self::When { span, .. }
            | Self::Segment { span, .. }
            | Self::Loop { span, .. }
            | Self::Each { span, .. }
            | Self::Require { span, .. }
            | Self::OutcomeWhen { span, .. }
            | Self::For { span, .. }
            | Self::Fix { span, .. }
            | Self::Error(span) => *span,
            Self::Let(stmt) => stmt.span,
        }
    }
}

/// `rule "title" { ... }`.
#[derive(Debug, Clone, PartialEq)]
pub struct RuleDecl {
    /// Its doc comment.
    pub doc: Doc,
    /// Its title.
    pub title: TextLit,
    /// Its statements.
    pub body: Block,
    /// The whole rule.
    pub span: Span,
}

/// A binary operator.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BinOp {
    /// `??`
    Default,
    /// `or`
    Or,
    /// `and`
    And,
    /// `==`
    Eq,
    /// `!=`
    Ne,
    /// `<`
    Lt,
    /// `<=`
    Le,
    /// `>`
    Gt,
    /// `>=`
    Ge,
    /// `in`
    In,
    /// `+`
    Add,
    /// `-`
    Sub,
    /// `*`
    Mul,
    /// `/`
    Div,
}

impl BinOp {
    /// How it is written.
    #[must_use]
    pub fn symbol(self) -> &'static str {
        match self {
            Self::Default => "??",
            Self::Or => "or",
            Self::And => "and",
            Self::Eq => "==",
            Self::Ne => "!=",
            Self::Lt => "<",
            Self::Le => "<=",
            Self::Gt => ">",
            Self::Ge => ">=",
            Self::In => "in",
            Self::Add => "+",
            Self::Sub => "-",
            Self::Mul => "*",
            Self::Div => "/",
        }
    }
}

/// A unary operator.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum UnOp {
    /// `not`
    Not,
    /// `-`
    Neg,
}

/// A piece of an interpolated text.
#[derive(Debug, Clone, PartialEq)]
pub enum TextPart {
    /// Literal text.
    Literal(String),
    /// An interpolated expression.
    Expr(Expr),
}

/// How a member is written after a dot.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum MemberKind {
    /// `.name`
    Name,
    /// `.N104`
    X12,
    /// `.0310`, `.02`
    Digits,
    /// `.Pickup`
    Type,
}

/// A call argument, positional or named.
#[derive(Debug, Clone, PartialEq)]
pub struct Arg {
    /// The name, for a named argument.
    pub name: Option<Ident>,
    /// The value.
    pub value: Expr,
}

/// One field of a record literal.
#[derive(Debug, Clone, PartialEq)]
pub struct FieldInit {
    /// The field.
    pub name: Ident,
    /// Its value.
    pub value: Expr,
    /// `else` for the field.
    pub otherwise: Option<Else>,
}

/// One arm of a `match`.
#[derive(Debug, Clone, PartialEq)]
pub struct MatchArm {
    /// The values it matches.
    pub patterns: Vec<Expr>,
    /// What it gives.
    pub value: Expr,
}

/// An expression.
#[derive(Debug, Clone, PartialEq)]
pub struct Expr {
    /// What it is.
    pub kind: ExprKind,
    /// Where it is.
    pub span: Span,
}

/// The forms of an expression.
#[derive(Debug, Clone, PartialEq)]
pub enum ExprKind {
    /// A name.
    Name(String),
    /// A `PascalCase` name: an enum variant.
    Type(String),
    /// An uppercase X12 reference.
    X12(String),
    /// A loop ID or element position.
    Digits(String),
    /// A number, as written without underscores.
    Number(String),
    /// A number with a unit (`45000`, `lb`).
    Quantity(String, String),
    /// A percent, the number written.
    Percent(String),
    /// Dollars, the number written.
    Money(String),
    /// A text, interpolated.
    Text(Vec<TextPart>),
    /// A raw text.
    Raw(String),
    /// `@2026-01-10`
    Date(String),
    /// `@14:30`
    Time(String),
    /// `@2026-01-10T14:30Z`
    Instant(String),
    /// `true` or `false`
    Bool(bool),
    /// `none`
    None,
    /// `[a, b]`
    List(Vec<Expr>),
    /// `["BM": bol, "PO": po]`, or `[:]` with no entries: text keys to values.
    Map(Vec<(Expr, Expr)>),
    /// `Type { field <- v }` or `{ name: v }`
    Record {
        /// The type, when written.
        ty: Option<Ident>,
        /// The fields.
        fields: Vec<FieldInit>,
    },
    /// `target.member`
    Member {
        /// What it is on.
        target: Box<Expr>,
        /// The member.
        member: Ident,
        /// How it is written.
        kind: MemberKind,
    },
    /// `target[index]`
    Index {
        /// What it is on.
        target: Box<Expr>,
        /// The index or filter.
        index: Box<Expr>,
    },
    /// `callee(args)`: a function, or a method when the callee is a member.
    Call {
        /// What is called.
        callee: Box<Expr>,
        /// The arguments.
        args: Vec<Arg>,
    },
    /// A unary operation.
    Unary(UnOp, Box<Expr>),
    /// A binary operation.
    Binary(BinOp, Box<Expr>, Box<Expr>),
    /// `expr via crosswalk`
    Via {
        /// What is translated.
        value: Box<Expr>,
        /// The crosswalk.
        crosswalk: Ident,
        /// Whether `.back` was written.
        back: bool,
    },
    /// `exists expr`
    Exists(Box<Expr>),
    /// `lookup Kind(code) learn hint`
    Lookup {
        /// The kind.
        kind: Ident,
        /// The partner's code.
        code: Box<Expr>,
        /// What to suggest a match from.
        learn: Option<Box<Expr>>,
    },
    /// `match value { ... }`
    Match {
        /// What is matched.
        value: Box<Expr>,
        /// The arms.
        arms: Vec<MatchArm>,
        /// `else => v`
        otherwise: Option<Box<Expr>>,
    },
    /// `if cond then a else b`
    If {
        /// The condition.
        cond: Box<Expr>,
        /// When true.
        then: Box<Expr>,
        /// When false.
        otherwise: Box<Expr>,
    },
    /// `x => body`
    Lambda {
        /// The parameter.
        param: Ident,
        /// The body.
        body: Box<Expr>,
    },
    /// `each list as x { ... }`: a list of records built per item.
    Each {
        /// The list.
        list: Box<Expr>,
        /// The binder.
        binder: Ident,
        /// The assignments per item.
        body: Block,
    },
    /// Could not be read.
    Error,
}

impl Expr {
    /// An expression at a span.
    #[must_use]
    pub fn new(kind: ExprKind, span: Span) -> Self {
        Self { kind, span }
    }

    /// An error node.
    #[must_use]
    pub fn error(span: Span) -> Self {
        Self::new(ExprKind::Error, span)
    }
}

/// One step of a test.
#[derive(Debug, Clone, PartialEq)]
pub enum TestStep {
    /// `given file "path"`
    GivenFile(TextLit),
    /// `given text "..."`
    GivenText(Expr),
    /// `given loadTender { ... }`
    GivenDocument {
        /// The document type.
        document: Ident,
        /// Its value.
        value: Expr,
    },
    /// `given setting name = value`
    GivenSetting {
        /// The setting.
        name: Ident,
        /// Its value.
        value: Expr,
    },
    /// `given lookup Kind("code") = "Label"` or `= none`
    GivenLookup {
        /// The kind.
        kind: Ident,
        /// The code.
        code: Expr,
        /// The label, or none for an unmapped code.
        label: Option<Expr>,
    },
    /// `given now = @...`
    GivenNow(Expr),
    /// `expect cond`
    Expect(Expr),
    /// `expect x12 contains "..."` or `expect x12 == "..."`
    ExpectX12 {
        /// Whether it is `contains`.
        contains: bool,
        /// The text.
        text: Expr,
    },
    /// `expect [no] held|rejected|declined|warned|sent|created|... ["message"]`
    ExpectOutcome {
        /// `no ...`
        negated: bool,
        /// `held`, `rejects`, `sent`, ...
        what: Ident,
        /// The set, for `sent`.
        set: Option<Ident>,
        /// The message or decision.
        detail: Option<Expr>,
    },
    /// `receive file "..."` or `receive text "..."`
    Receive {
        /// Whether it is a file.
        file: bool,
        /// The path or text.
        source: Expr,
    },
    /// `respond accept` or `respond decline "reason"`
    Respond {
        /// Whether it is accept.
        accept: bool,
        /// The reason.
        reason: Option<Expr>,
    },
    /// `event Kind { ... }`
    Event {
        /// The event.
        kind: Ident,
        /// Its fields.
        value: Option<Expr>,
    },
    /// `ack accepted` or `ack rejected`
    Ack(bool),
    /// `advance 2 h`
    Advance(Expr),
}

/// `test "name" { ... }`.
#[derive(Debug, Clone, PartialEq)]
pub struct TestDecl {
    /// Its doc comment.
    pub doc: Doc,
    /// Its name.
    pub name: TextLit,
    /// Its steps, with their spans.
    pub steps: Vec<(TestStep, Span)>,
    /// The whole test.
    pub span: Span,
}

/// Where a property's values come from.
#[derive(Debug, Clone, PartialEq)]
pub enum VarySource {
    /// `samples "glob"`
    Samples(TextLit),
    /// `[a, b]`
    List(Expr),
    /// `a..b`
    Range(Expr, Expr),
    /// An enum type.
    Enum(Ident),
}

/// `vary name in source`.
#[derive(Debug, Clone, PartialEq)]
pub struct Vary {
    /// The name.
    pub name: Ident,
    /// The values.
    pub source: VarySource,
    /// The whole line.
    pub span: Span,
}

/// `property "name" { vary ...; steps }`.
#[derive(Debug, Clone, PartialEq)]
pub struct PropertyDecl {
    /// Its doc comment.
    pub doc: Doc,
    /// Its name.
    pub name: TextLit,
    /// Its varied values.
    pub varies: Vec<Vary>,
    /// Its steps.
    pub steps: Vec<(TestStep, Span)>,
    /// The whole property.
    pub span: Span,
}

/// One rule for an element in a guide.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ElementRule {
    /// `required`
    Required,
    /// `unused`
    Unused,
    /// `codes "a", "b"`
    Codes(Vec<TextLit>),
    /// `length a..b`
    Length(u32, u32),
    /// `repeat n`
    Repeat(u32),
    /// `fixed "v"`
    Fixed(TextLit),
}

/// One element line of a guide segment block.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GuideElement {
    /// The element position.
    pub position: u32,
    /// The component position.
    pub component: Option<u32>,
    /// Its rules.
    pub rules: Vec<(ElementRule, Span)>,
    /// A note.
    pub note: Option<TextLit>,
    /// The whole line.
    pub span: Span,
}

/// One rule for a segment or loop in a guide.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum StructureRule {
    /// `required`
    Required,
    /// `unused`
    Unused,
    /// `max n`
    Max(u32),
    /// `min n`
    Min(u32),
    /// `syntax P0304 ...`
    Syntax(Vec<Ident>),
}

/// A line of a guide.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum GuideDecl {
    /// `0300.S5 max 2 { ... }`
    Segment {
        /// Its doc comment.
        doc: Doc,
        /// The loops then the segment.
        path: Vec<Ident>,
        /// The structure rules.
        rules: Vec<(StructureRule, Span)>,
        /// The element lines.
        elements: Vec<GuideElement>,
        /// A note.
        note: Option<TextLit>,
        /// The whole line.
        span: Span,
    },
    /// `loop 0300 min 2 max 12`
    Loop {
        /// Its doc comment.
        doc: Doc,
        /// The loop path.
        path: Vec<Ident>,
        /// Its rules.
        rules: Vec<(StructureRule, Span)>,
        /// The whole line.
        span: Span,
    },
    /// `severity kind level`
    Severity {
        /// The finding kind.
        kind: Ident,
        /// The level.
        level: Ident,
        /// The whole line.
        span: Span,
    },
}

impl GuideDecl {
    /// Where it is.
    #[must_use]
    pub fn span(&self) -> Span {
        match self {
            Self::Segment { span, .. } | Self::Loop { span, .. } | Self::Severity { span, .. } => {
                *span
            }
        }
    }
}

/// A flow action.
#[derive(Debug, Clone, PartialEq)]
pub enum Action {
    /// `create`
    Create(Span),
    /// `update fields keep fields hold when cond "message"`
    Update {
        /// The fields applied.
        fields: Vec<Ident>,
        /// The fields kept.
        keep: Vec<Ident>,
        /// Hold instead when this holds.
        hold_when: Option<(Expr, Expr)>,
        /// The whole action.
        span: Span,
    },
    /// `cancel unless state else action`
    Cancel {
        /// The state that stops the cancel, and what to do instead.
        unless: Option<(Ident, Box<Action>)>,
        /// The whole action.
        span: Span,
    },
    /// `apply`
    Apply(Span),
    /// `hold "message"`
    Hold(Expr, Span),
    /// `ignore`
    Ignore(Span),
    /// Could not be read.
    Error(Span),
}

impl Action {
    /// Where it is.
    #[must_use]
    pub fn span(&self) -> Span {
        match self {
            Self::Create(span)
            | Self::Apply(span)
            | Self::Ignore(span)
            | Self::Error(span)
            | Self::Hold(_, span)
            | Self::Update { span, .. }
            | Self::Cancel { span, .. } => *span,
        }
    }
}

/// One `on Purpose -> action`, or a bare `-> action`.
#[derive(Debug, Clone, PartialEq)]
pub struct ReceiveArm {
    /// The purpose.
    pub purpose: Option<Ident>,
    /// The action.
    pub action: Action,
    /// The whole arm.
    pub span: Span,
}

/// `receive ...`.
#[derive(Debug, Clone, PartialEq)]
pub struct Receive {
    /// The transaction set.
    pub set: Ident,
    /// The name its document is bound to.
    pub binder: Ident,
    /// The map.
    pub map: Option<Ident>,
    /// The rules.
    pub rules: Option<Ident>,
    /// The key.
    pub key: Option<Expr>,
    /// `acknowledge with ...`
    pub acknowledge: Option<Ident>,
    /// The arms.
    pub arms: Vec<ReceiveArm>,
    /// The whole clause.
    pub span: Span,
}

/// How an unanswered tender is decided.
#[derive(Debug, Clone, PartialEq)]
pub enum Otherwise {
    /// `otherwise planner decides within d else escalate "m"`
    Planner {
        /// The deadline.
        within: Option<Expr>,
        /// The escalation message.
        escalate: Option<Expr>,
    },
    /// `otherwise accept`
    Accept,
    /// `otherwise decline "m"`
    Decline(Expr),
}

/// `respond ...`.
#[derive(Debug, Clone, PartialEq)]
pub struct Respond {
    /// The transaction set.
    pub set: Ident,
    /// The map.
    pub map: Option<Ident>,
    /// The `accept automatically when` conditions.
    pub accept: Vec<Expr>,
    /// The `decline automatically when` conditions.
    pub decline: Vec<Expr>,
    /// The `otherwise`.
    pub otherwise: Option<(Otherwise, Span)>,
    /// The whole clause.
    pub span: Span,
}

/// `send ...`.
#[derive(Debug, Clone, PartialEq)]
pub struct Send {
    /// The transaction set.
    pub set: Ident,
    /// The map.
    pub map: Option<Ident>,
    /// The events.
    pub events: Vec<Ident>,
    /// `where cond`
    pub filter: Option<Expr>,
    /// `key expr`
    pub key: Option<Expr>,
    /// `batch within d`
    pub batch: Option<Expr>,
    /// `until Event`
    pub until: Option<Ident>,
    /// The whole clause.
    pub span: Span,
}

/// `expect 997 for every send within d else resend n then escalate "m"`.
#[derive(Debug, Clone, PartialEq)]
pub struct ExpectAck {
    /// The acknowledgment set.
    pub set: Ident,
    /// The deadline.
    pub within: Option<Expr>,
    /// How many resends.
    pub resend: Option<(u32, Span)>,
    /// The escalation message.
    pub escalate: Option<Expr>,
    /// The whole clause.
    pub span: Span,
}

/// A flow clause.
#[derive(Debug, Clone, PartialEq)]
pub enum FlowDecl {
    /// `receive`
    Receive(Receive),
    /// `respond`
    Respond(Respond),
    /// `send`
    Send(Send),
    /// `expect`
    Expect(ExpectAck),
}

impl FlowDecl {
    /// Where it is.
    #[must_use]
    pub fn span(&self) -> Span {
        match self {
            Self::Receive(item) => item.span,
            Self::Respond(item) => item.span,
            Self::Send(item) => item.span,
            Self::Expect(item) => item.span,
        }
    }
}
