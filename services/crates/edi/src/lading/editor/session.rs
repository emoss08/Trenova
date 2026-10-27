//! What every editor operation starts from: the project checked, the file being edited, its
//! tokens, and conversions between UTF-16 offsets and bytes.

use serde::Serialize;
use trenova_module_kit::message::Message;
use trenova_module_kit::span::{Span, Utf16Index};

use std::rc::Rc;

use crate::lading::check::Program;
use crate::lading::facts::ScopeFact;
use crate::lading::syntax::lexer::lex;
use crate::lading::syntax::token::{Tok, Token};

/// Help text: what the author wrote (a doc comment, a guide's note), or a message the hosts
/// translate.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
#[serde(untagged)]
pub enum Doc {
    /// The author's words.
    Text(String),
    /// A message.
    Message(Message),
}

impl Doc {
    /// A message without arguments.
    #[must_use]
    pub fn message(text: &'static str) -> Self {
        Self::Message(Message::new(text))
    }
}

/// A checked project and the file being edited.
#[derive(Debug)]
pub struct Session {
    /// The program.
    pub program: Rc<Program>,
    /// The file being edited.
    pub unit: usize,
    /// Its UTF-16 index.
    pub index: Utf16Index,
    /// Its tokens.
    pub tokens: Vec<Token>,
}

impl Session {
    /// Opens one file of a checked project; `None` when the path is not in it.
    #[must_use]
    pub fn open(program: Rc<Program>, path: &str) -> Option<Self> {
        let unit = program.unit_at(path)?;
        let source = &program.units[unit].source;
        let index = Utf16Index::new(source);
        let tokens = lex(source).tokens;
        Some(Self {
            program,
            unit,
            index,
            tokens,
        })
    }

    /// The file's text.
    #[must_use]
    pub fn source(&self) -> &str {
        &self.program.units[self.unit].source
    }

    /// The text of a span.
    #[must_use]
    pub fn text(&self, span: Span) -> &str {
        self.source()
            .get(span.start as usize..span.end as usize)
            .unwrap_or_default()
    }

    /// A UTF-16 offset as a byte offset.
    #[must_use]
    pub fn byte(&self, utf16: u32) -> u32 {
        self.index.to_byte(self.source(), utf16)
    }

    /// The token the cursor is in or just after, when it is a word.
    #[must_use]
    pub fn word_at(&self, offset: u32) -> Option<Token> {
        self.tokens.iter().copied().find(|token| {
            matches!(
                token.kind,
                Tok::Name | Tok::TypeName | Tok::X12 | Tok::Digits | Tok::Number
            ) && token.span.start <= offset
                && offset <= token.span.end
                && token.span.start < token.span.end
        })
    }

    /// The tokens before an offset, nearest last, line ends included.
    #[must_use]
    pub fn before(&self, offset: u32) -> Vec<Token> {
        self.tokens
            .iter()
            .copied()
            .filter(|token| token.span.end <= offset && token.kind != Tok::Eof)
            .collect()
    }

    /// The innermost scope around an offset.
    #[must_use]
    pub fn scope(&self, offset: u32) -> Option<&ScopeFact> {
        self.program.facts.scope_at(self.unit, offset)
    }

    /// The first word of the line an offset is on.
    #[must_use]
    pub fn line_word(&self, offset: u32) -> String {
        let source = self.source();
        let start = source[..(offset as usize).min(source.len())]
            .rfind('\n')
            .map_or(0, |at| at + 1);
        source[start..]
            .split(|c: char| !(c.is_alphanumeric() || c == '_'))
            .find(|word| !word.is_empty())
            .unwrap_or_default()
            .to_owned()
    }

    /// The first word of the statement an offset is in, looking back over continuation lines.
    #[must_use]
    pub fn clause_word(&self, offset: u32) -> String {
        let source = self.source();
        let mut end = (offset as usize).min(source.len());
        loop {
            let start = source[..end].rfind('\n').map_or(0, |at| at + 1);
            let line = &source[start..];
            let indented = line.starts_with(' ') || line.starts_with('\t');
            if !indented || start == 0 {
                return line
                    .split(|c: char| !(c.is_alphanumeric() || c == '_'))
                    .find(|word| !word.is_empty())
                    .unwrap_or_default()
                    .to_owned();
            }
            end = start - 1;
        }
    }
}
