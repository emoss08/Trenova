//! The tokens a Lading source is cut into.

use trenova_module_kit::span::Span;

/// What a token is.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Tok {
    /// A `camelCase` name, keyword or unit (`stopTypes`, `each`, `lb`, `°F`).
    Name,
    /// A `PascalCase` type or enum variant (`StopType`, `Delivery`).
    TypeName,
    /// Uppercase letters and digits: a segment, an element reference or a qualifier (`N104`).
    X12,
    /// A loop identifier or element position (`0300`, `2000A`, `02` after a dot).
    Digits,
    /// A number (`1_200`, `12.50`).
    Number,
    /// A double-quoted text, interpolation and all.
    Text,
    /// A raw text `r"..."`.
    RawText,
    /// `@2026-01-10`.
    Date,
    /// `@14:30`.
    Time,
    /// `@2026-01-10T14:30Z`, with an offset or a zone.
    Instant,
    /// `$1_850.00`.
    Money,
    /// The end of a line.
    Newline,
    /// `{`
    LBrace,
    /// `}`
    RBrace,
    /// `(`
    LParen,
    /// `)`
    RParen,
    /// `[`
    LBracket,
    /// `]`
    RBracket,
    /// `,`
    Comma,
    /// `;`
    Semi,
    /// `.`
    Dot,
    /// `..`
    DotDot,
    /// `:`
    Colon,
    /// `<-`
    LArrow,
    /// `->`
    RArrow,
    /// `<->`
    BiArrow,
    /// `=>`
    FatArrow,
    /// `=`
    Assign,
    /// `==`
    EqEq,
    /// `!=`
    NotEq,
    /// `<`
    Lt,
    /// `<=`
    Le,
    /// `>`
    Gt,
    /// `>=`
    Ge,
    /// `+`
    Plus,
    /// `-`
    Minus,
    /// `*`
    Star,
    /// `/`
    Slash,
    /// `%`
    Percent,
    /// `??`
    QuestionQuestion,
    /// `?`
    Question,
    /// `|`
    Pipe,
    /// Something that is not a token.
    Error,
    /// The end of the source.
    Eof,
}

impl Tok {
    /// Whether a line ending in this token continues onto the next.
    #[must_use]
    pub fn continues_line(self) -> bool {
        matches!(
            self,
            Self::Comma
                | Self::LBrace
                | Self::LParen
                | Self::LBracket
                | Self::Dot
                | Self::LArrow
                | Self::RArrow
                | Self::BiArrow
                | Self::FatArrow
                | Self::Assign
                | Self::EqEq
                | Self::NotEq
                | Self::Lt
                | Self::Le
                | Self::Gt
                | Self::Ge
                | Self::Plus
                | Self::Minus
                | Self::Star
                | Self::Slash
                | Self::QuestionQuestion
                | Self::Pipe
                | Self::Colon
        )
    }
}

/// One token.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Token {
    /// What it is.
    pub kind: Tok,
    /// Where it is.
    pub span: Span,
}

/// A comment, kept beside the tokens for the formatter and the doc of the next declaration.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Comment {
    /// The comment, `//` included.
    pub span: Span,
    /// Whether it is a `///` doc comment.
    pub doc: bool,
}

/// Words that are never names.
pub const RESERVED: &[&str] = &[
    "fn",
    "let",
    "const",
    "crosswalk",
    "rule",
    "require",
    "else",
    "reject",
    "decline",
    "hold",
    "warn",
    "fix",
    "when",
    "for",
    "in",
    "where",
    "each",
    "as",
    "via",
    "lookup",
    "learn",
    "match",
    "if",
    "then",
    "test",
    "property",
    "given",
    "expect",
    "vary",
    "and",
    "or",
    "not",
    "true",
    "false",
    "none",
    "exists",
];

/// Whether `word` is reserved.
#[must_use]
pub fn is_reserved(word: &str) -> bool {
    RESERVED.contains(&word)
}
