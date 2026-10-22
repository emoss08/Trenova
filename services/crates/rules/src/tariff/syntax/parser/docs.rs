//! Doc comments: consecutive `///` lines grouped into blocks, each attached to what is written on
//! the line right after it (an item, an enum variant or a record field), or, right after the
//! header, to the file. A block attached to nothing is a `doc` warning.

use super::Parser;
use crate::message;
use crate::message::Message;
use crate::span::Span;
use crate::tariff::diagnostic::Change;
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::diagnostic::Fix;
use crate::tariff::syntax::ast::Doc;
use crate::tariff::syntax::token::Token;
use crate::tariff::syntax::token::TokenKind;

#[derive(Debug, Clone)]
struct Block {
    /// The `///` tokens' spans.
    lines: Vec<Span>,
    text: String,
    /// The index (into all tokens) of the first significant token after the block, when it is
    /// on the very next line.
    attaches_to: Option<usize>,
    /// The index (into all tokens) of the first token after the block.
    next: usize,
    used: bool,
}

/// The doc blocks of a file.
#[derive(Debug, Clone, Default)]
pub(super) struct DocTable {
    blocks: Vec<Block>,
}

/// The text of one `///` line: the slashes and one following space removed, trailing spaces
/// trimmed.
fn line_text(raw: &str) -> &str {
    let body = raw.strip_prefix("///").unwrap_or(raw);
    body.strip_prefix(' ').unwrap_or(body).trim_end()
}

impl DocTable {
    pub(super) fn new(source: &str, tokens: &[Token]) -> Self {
        let mut blocks: Vec<Block> = Vec::new();
        let mut index = 0;
        while index < tokens.len() {
            if tokens
                .get(index)
                .is_none_or(|t| t.kind != TokenKind::DocComment)
            {
                index += 1;
                continue;
            }
            let mut lines = Vec::new();
            let mut cursor = index;
            while let Some(token) = tokens.get(cursor)
                && token.kind == TokenKind::DocComment
            {
                lines.push(token.span);
                let newline = tokens
                    .get(cursor + 1)
                    .is_some_and(|t| t.kind == TokenKind::Newline);
                let doc_next = tokens
                    .get(cursor + 2)
                    .is_some_and(|t| t.kind == TokenKind::DocComment);
                if newline && doc_next {
                    cursor += 2;
                } else {
                    cursor += 1;
                    break;
                }
            }
            let next = cursor;
            let attaches_to = match (tokens.get(next), tokens.get(next + 1)) {
                (Some(newline), Some(token))
                    if newline.kind == TokenKind::Newline
                        && !matches!(
                            token.kind,
                            TokenKind::Newline
                                | TokenKind::Comment
                                | TokenKind::DocComment
                                | TokenKind::Eof
                        ) =>
                {
                    Some(next + 1)
                }
                _ => None,
            };
            let text = lines
                .iter()
                .map(|span| {
                    line_text(
                        source
                            .get(span.start as usize..span.end as usize)
                            .unwrap_or_default(),
                    )
                })
                .collect::<Vec<_>>()
                .join("\n");
            blocks.push(Block {
                lines,
                text,
                attaches_to,
                next,
                used: false,
            });
            index = next;
        }
        Self { blocks }
    }

    fn doc(block: &Block) -> Doc {
        let start = block.lines.first().map_or(0, |span| span.start);
        let end = block.lines.last().map_or(0, |span| span.end);
        Doc {
            text: block.text.clone(),
            span: Span::new(start, end),
        }
    }
}

impl Parser<'_> {
    /// The doc comment attached to the current token, taken so it is attached once.
    pub(super) fn take_doc(&mut self) -> Option<Doc> {
        let token = self.sig.get(self.pos).copied()?;
        let block = self
            .docs
            .blocks
            .iter_mut()
            .find(|block| !block.used && block.attaches_to == Some(token))?;
        block.used = true;
        Some(DocTable::doc(block))
    }

    /// The file's doc: the first block after the header when nothing else took it.
    pub(super) fn take_file_doc(&mut self, header_end: u32) -> Option<Doc> {
        let first_after = self
            .tokens
            .iter()
            .position(|t| t.span.start >= header_end)?;
        let only_newlines = |from: usize, to: usize, tokens: &[Token]| {
            tokens
                .get(from..to)
                .is_some_and(|between| between.iter().all(|t| t.kind == TokenKind::Newline))
        };
        let tokens = &self.tokens;
        let block = self.docs.blocks.iter_mut().find(|block| {
            let start = tokens
                .iter()
                .position(|t| block.lines.first() == Some(&t.span))
                .unwrap_or(0);
            !block.used && only_newlines(first_after, start, tokens)
        })?;
        block.used = true;
        Some(DocTable::doc(block))
    }

    /// Reports every doc block nothing took.
    pub(super) fn report_stray_docs(&mut self) {
        let stray: Vec<Block> = self
            .docs
            .blocks
            .iter()
            .filter(|block| !block.used)
            .cloned()
            .collect();
        for block in stray {
            let doc = DocTable::doc(&block);
            let changes = block
                .lines
                .iter()
                .map(|span| Change {
                    span: Span::new(span.start, span.start + 3),
                    insert: String::from("//"),
                })
                .collect();
            let message = if block.next < self.tokens.len() && block.attaches_to.is_none() {
                message!(
                    "This doc comment is not attached to anything; write it on the line right above what it describes, or make it a plain `//` comment"
                )
            } else {
                message!("This doc comment is not attached to anything")
            };
            self.diagnostics.push(
                Diagnostic::warning("doc", doc.span, Message::new(message)).with_fix(Fix {
                    label: Message::new(message!("Make it a plain comment")),
                    changes,
                }),
            );
        }
    }
}
