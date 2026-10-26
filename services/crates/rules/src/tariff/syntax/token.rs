//! Tokens: what the lexer produces, what the parser reads and what `tariff.tokens` sends the
//! editor for highlighting.

use trenova_module_kit::span::Span;

/// One token.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Token {
    /// What it is.
    pub kind: TokenKind,
    /// Where it is.
    pub span: Span,
}

/// The kinds of token.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum TokenKind {
    /// A keyword (`let`, `charge`, `and`, …).
    Keyword,
    /// A `camelCase` name.
    Name,
    /// A `PascalCase` name: a type or enum variant.
    TypeName,
    /// An `UPPERCASE` name: a charge code or a constant.
    Code,
    /// A number, without its unit.
    Number,
    /// A unit after a number, after `$x/` or in a type.
    Unit,
    /// `$450`, `$2.35`: the `$` and the amount.
    Money,
    /// `18%`.
    Percent,
    /// `@2026-10-22`, `@2026-10-22T08:00-05:00`, `@17:30`.
    Date,
    /// The opening quote and literal characters of a text, up to an interpolation or the end.
    Text,
    /// An escape inside a text: `\n`, `\"`, `{{`.
    Escape,
    /// `{` or `}` around an interpolation.
    Interpolation,
    /// An operator or punctuation: `+`, `??`, `=>`, `(`, `,`.
    Punct,
    /// `// …`.
    Comment,
    /// `/// …`.
    DocComment,
    /// A newline that ends a statement.
    Newline,
    /// Text the lexer could not read.
    Error,
    /// The end of the file.
    Eof,
}

impl TokenKind {
    /// The class the editor highlights the token as, on the wire.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Keyword => "keyword",
            Self::Name => "name",
            Self::TypeName => "type",
            Self::Code => "code",
            Self::Number => "number",
            Self::Unit => "unit",
            Self::Money => "money",
            Self::Percent => "percent",
            Self::Date => "date",
            Self::Text => "text",
            Self::Escape => "escape",
            Self::Interpolation => "interpolation",
            Self::Punct => "punct",
            Self::Comment => "comment",
            Self::DocComment => "doc",
            Self::Newline => "newline",
            Self::Error => "error",
            Self::Eof => "eof",
        }
    }
}
