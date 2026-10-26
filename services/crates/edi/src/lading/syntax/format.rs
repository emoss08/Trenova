//! The one canonical layout of a Lading file.
//!
//! The formatter reprints the tokens it was given, so nothing is ever lost: comments stay where
//! they were written, line breaks the author made stay, and blank lines collapse to one. It
//! fixes indentation (two spaces per brace, two more for a line that continues the one before
//! or a clause of the line above), spacing around every token, and aligns the arrows of
//! consecutive assignments and the keywords of a map's header. A file with syntax errors is
//! left as it is.

use super::parser::{Parsed, parse};
use super::token::{Comment, Tok, Token};

/// Why a file was not formatted.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FormatError {
    /// It has syntax errors.
    Syntax,
}

/// Formats `source`.
pub fn format(source: &str) -> Result<String, FormatError> {
    let parsed = parse(source);
    if !parsed.diagnostics.is_empty() {
        return Err(FormatError::Syntax);
    }
    Ok(Printer::new(source, &parsed).print())
}

/// One output line before alignment.
#[derive(Debug, Clone)]
struct Line {
    indent: usize,
    text: String,
    /// The byte column of the first `<-` when the line is a simple assignment.
    arrow: Option<usize>,
    /// The byte column of the `->` of a receive arm (`on Change -> ...`).
    arm: Option<usize>,
    /// The width of a map header keyword (`from`, `to`, `use`) when the line starts with one.
    header_word: Option<usize>,
}

/// Words that start a continuation clause of the line above in a flow or a map header.
const CLAUSE_WORDS: &[&str] = &[
    "from",
    "to",
    "use",
    "via",
    "key",
    "acknowledge",
    "on",
    "accept",
    "decline",
    "otherwise",
    "within",
    "where",
    "batch",
    "until",
    "keep",
    "else",
];

/// Where the printer is.
#[derive(Debug, Default)]
struct State {
    depth: usize,
    paren_depth: usize,
    comment: usize,
    blank_pending: bool,
    previous_continues: bool,
    clause_parent: Option<usize>,
}

struct Printer<'a> {
    source: &'a str,
    tokens: &'a [Token],
    comments: &'a [Comment],
    lines: Vec<Line>,
    state: State,
}

impl<'a> Printer<'a> {
    fn new(source: &'a str, parsed: &'a Parsed) -> Self {
        Self {
            source,
            tokens: &parsed.tokens,
            comments: &parsed.comments,
            lines: Vec::new(),
            state: State::default(),
        }
    }

    fn text(&self, token: &Token) -> &'a str {
        self.source
            .get(token.span.start as usize..token.span.end as usize)
            .unwrap_or_default()
    }

    fn print(mut self) -> String {
        let mut index = 0usize;
        while index < self.tokens.len() {
            let token = self.tokens[index];
            if token.kind == Tok::Eof {
                break;
            }
            if token.kind == Tok::Newline {
                if self.blank_line_after(index) {
                    self.state.blank_pending = true;
                }
                index += 1;
                continue;
            }
            self.leading_comments(token);
            if self.state.blank_pending && !self.lines.is_empty() {
                self.push_blank();
            }
            self.state.blank_pending = false;
            let mut line_end = index;
            while line_end < self.tokens.len()
                && !matches!(self.tokens[line_end].kind, Tok::Newline | Tok::Eof)
            {
                line_end += 1;
            }
            self.line(index, line_end);
            index = line_end;
        }
        while let Some(comment) = self.comments.get(self.state.comment) {
            let text = self.comment_text(comment);
            self.push(self.state.depth * 2, text);
            self.state.comment += 1;
        }
        self.align();
        let mut out = String::with_capacity(self.source.len());
        for line in &self.lines {
            if !line.text.is_empty() {
                out.push_str(&" ".repeat(line.indent));
                out.push_str(&line.text);
            }
            out.push('\n');
        }
        while out.ends_with("\n\n") {
            out.pop();
        }
        out
    }

    fn push(&mut self, indent: usize, text: String) {
        self.lines.push(Line {
            indent,
            text,
            arrow: None,
            arm: None,
            header_word: None,
        });
    }

    /// The comments on their own lines before `token`.
    fn leading_comments(&mut self, token: Token) {
        while let Some(comment) = self.comments.get(self.state.comment) {
            if comment.span.start >= token.span.start || self.is_trailing(comment) {
                break;
            }
            if self.state.blank_pending && !self.lines.is_empty() {
                self.push_blank();
            }
            self.state.blank_pending = false;
            let depth = if token.kind == Tok::RBrace {
                self.state.depth.saturating_sub(1)
            } else {
                self.state.depth
            };
            let text = self.comment_text(comment);
            self.push(depth * 2, text);
            self.state.comment += 1;
        }
    }

    /// The indentation of a source line starting at `start`.
    fn indent_of(&mut self, start: usize) -> (usize, bool) {
        let token = self.tokens[start];
        let first = self.text(&token);
        let base = if token.kind == Tok::RBrace {
            self.state.depth.saturating_sub(1)
        } else {
            self.state.depth
        };
        let second = self.tokens.get(start + 1).map(|next| next.kind);
        let assignment_like = matches!(
            second,
            Some(Tok::LArrow | Tok::Colon | Tok::Dot | Tok::LBracket | Tok::Assign | Tok::LParen)
        );
        let crosswalk_else = first == "else" && matches!(second, Some(Tok::RArrow | Tok::LArrow));
        let starts_clause = token.kind == Tok::RArrow
            || (token.kind == Tok::Name
                && CLAUSE_WORDS.contains(&first)
                && !assignment_like
                && !crosswalk_else);
        let clause_tail = token.kind == Tok::Name
            && matches!(first, "keep" | "hold")
            && self.state.clause_parent.is_some()
            && !assignment_like;
        let continuation = self.state.previous_continues || self.state.paren_depth > 0;
        let mut indent = base * 2;
        if clause_tail {
            indent = self.state.clause_parent.unwrap_or(base) * 2 + 4;
        } else if continuation || starts_clause {
            indent += 2;
        }
        if starts_clause {
            self.state.clause_parent = Some(base);
        } else if !continuation && !clause_tail {
            self.state.clause_parent = None;
        }
        (indent, starts_clause)
    }

    /// Prints the source line `[start, end)`.
    fn line(&mut self, start: usize, end: usize) {
        let token = self.tokens[start];
        let first = self.text(&token);
        let (indent, starts_clause) = self.indent_of(start);
        let mut text = self.render_line(start, end);
        let second = self.tokens.get(start + 1).map(|next| next.kind);
        let arrow = if base_is_assignment(&self.tokens[start..end]) {
            text.find(" <- ")
        } else {
            None
        };
        let arm = if first == "on" && starts_clause && second == Some(Tok::TypeName) {
            text.find(" -> ")
        } else {
            None
        };
        let header_word = (token.kind == Tok::Name && matches!(first, "from" | "to" | "use"))
            .then_some(first.len());
        self.trailing_comments(start, end, &mut text);
        self.lines.push(Line {
            indent,
            text,
            arrow,
            arm,
            header_word,
        });
        for token in &self.tokens[start..end] {
            match token.kind {
                Tok::LBrace => self.state.depth += 1,
                Tok::RBrace => self.state.depth = self.state.depth.saturating_sub(1),
                Tok::LParen | Tok::LBracket => self.state.paren_depth += 1,
                Tok::RParen | Tok::RBracket => {
                    self.state.paren_depth = self.state.paren_depth.saturating_sub(1);
                }
                _ => {}
            }
        }
        let last = self.tokens[end.saturating_sub(1).max(start)];
        self.state.previous_continues = last.kind.continues_line() && last.kind != Tok::LBrace;
    }

    /// Appends the comment after the code of the line `[start, end)`.
    fn trailing_comments(&mut self, start: usize, end: usize, text: &mut String) {
        let line_last_end = self.tokens[end.saturating_sub(1).max(start)].span.end;
        let next_newline = self.tokens.get(end).map_or(u32::MAX, |next| next.span.start);
        while let Some(comment) = self.comments.get(self.state.comment) {
            if !self.is_trailing(comment)
                || comment.span.start < line_last_end
                || comment.span.start > next_newline
            {
                break;
            }
            text.push(' ');
            text.push_str(&self.comment_text(comment));
            self.state.comment += 1;
        }
    }

    fn push_blank(&mut self) {
        if self.lines.last().is_some_and(|line| !line.text.is_empty()) {
            self.push(0, String::new());
        }
    }

    fn comment_text(&self, comment: &Comment) -> String {
        self.source
            .get(comment.span.start as usize..comment.span.end as usize)
            .unwrap_or_default()
            .trim_end()
            .to_owned()
    }

    /// Whether a comment follows code on its line.
    fn is_trailing(&self, comment: &Comment) -> bool {
        let before = &self.source[..comment.span.start as usize];
        let line = before.rsplit('\n').next().unwrap_or_default();
        !line.trim().is_empty()
    }

    /// Whether the newline at `index` is followed by another (a blank line in the source).
    fn blank_line_after(&self, index: usize) -> bool {
        let Some(token) = self.tokens.get(index) else {
            return false;
        };
        let next = self.tokens.get(index + 1);
        let gap_end = next.map_or(self.source.len(), |next| next.span.start as usize);
        let gap = self
            .source
            .get(token.span.end as usize..gap_end)
            .unwrap_or_default();
        let has_comment = self
            .comments
            .iter()
            .any(|comment| comment.span.start >= token.span.end && (comment.span.start as usize) < gap_end);
        if has_comment {
            let first_comment = self
                .comments
                .iter()
                .find(|comment| comment.span.start >= token.span.end)
                .map_or(gap_end, |comment| comment.span.start as usize);
            let before_comment = self
                .source
                .get(token.span.end as usize..first_comment)
                .unwrap_or_default();
            return before_comment.contains('\n');
        }
        next.is_some_and(|next| next.kind == Tok::Newline) || gap.contains('\n')
    }

    fn render_line(&self, start: usize, end: usize) -> String {
        let mut out = String::new();
        let mut previous: Option<Token> = None;
        let line = &self.tokens[start..end];
        let tight: Vec<bool> = (0..line.len())
            .map(|position| {
                let token = line[position];
                matches!(token.kind, Tok::Minus | Tok::Lt | Tok::Gt | Tok::Slash)
                    && position > 0
                    && position + 1 < line.len()
                    && line[position - 1].span.end == token.span.start
                    && token.span.end == line[position + 1].span.start
            })
            .collect();
        let mut angles = 0usize;
        for (position, token) in line.iter().enumerate() {
            let text = self.text(token);
            let closes_angle = token.kind == Tok::Gt
                && angles > 0
                && position > 0
                && line[position - 1].span.end == token.span.start;
            if token.kind == Tok::Lt && tight[position] {
                angles += 1;
            }
            if closes_angle {
                angles -= 1;
            }
            let glued = (position > 0 && tight[position - 1]) || tight[position] || closes_angle;
            if glued {
                // An operator written touching both neighbours (`Code<S502>`, `code-not-in-list`).
            } else if let Some(prev) = previous
                && space_between(prev, *token, text)
            {
                out.push(' ');
            }
            out.push_str(text);
            previous = Some(*token);
        }
        out
    }

    /// Aligns the arrows of consecutive assignments and the words of a map header.
    fn align(&mut self) {
        let mut index = 0;
        while index < self.lines.len() {
            if self.lines[index].arrow.is_some() {
                let mut end = index;
                while end < self.lines.len()
                    && self.lines[end].arrow.is_some()
                    && self.lines[end].indent == self.lines[index].indent
                {
                    end += 1;
                }
                let width = self.lines[index..end]
                    .iter()
                    .filter_map(|line| line.arrow)
                    .max()
                    .unwrap_or(0);
                for line in &mut self.lines[index..end] {
                    if let Some(arrow) = line.arrow {
                        let pad = width - arrow;
                        line.text.insert_str(arrow, &" ".repeat(pad));
                    }
                }
                index = end;
                continue;
            }
            if self.lines[index].arm.is_some() {
                let mut end = index;
                let mut widest = 0;
                let mut group = Vec::new();
                while end < self.lines.len()
                    && (self.lines[end].arm.is_some()
                        || (end > index && self.lines[end].indent > self.lines[index].indent))
                {
                    if let Some(arm) = self.lines[end].arm {
                        widest = widest.max(arm);
                        group.push(end);
                    }
                    end += 1;
                }
                for line_index in group {
                    if let Some(arm) = self.lines[line_index].arm {
                        let pad = widest - arm;
                        self.lines[line_index].text.insert_str(arm, &" ".repeat(pad));
                    }
                }
                index = end;
                continue;
            }
            if self.lines[index].header_word.is_some() {
                let mut end = index;
                while end < self.lines.len() && self.lines[end].header_word.is_some() {
                    end += 1;
                }
                let width = self.lines[index..end]
                    .iter()
                    .filter_map(|line| line.header_word)
                    .max()
                    .unwrap_or(0);
                for line in &mut self.lines[index..end] {
                    if let Some(word) = line.header_word {
                        line.text.insert_str(word, &" ".repeat(width - word));
                    }
                }
                index = end;
                continue;
            }
            index += 1;
        }
    }
}

/// Whether a line is `target <- value` with a plain target (a name and fields).
fn base_is_assignment(tokens: &[Token]) -> bool {
    let mut index = 0;
    if tokens.first().map(|token| token.kind) != Some(Tok::Name) {
        return false;
    }
    index += 1;
    while index + 1 < tokens.len() && tokens[index].kind == Tok::Dot {
        index += 2;
    }
    tokens.get(index).map(|token| token.kind) == Some(Tok::LArrow)
}

fn space_between(prev: Token, next: Token, next_text: &str) -> bool {
    let touching = prev.span.end == next.span.start;
    let tight_after = matches!(prev.kind, Tok::Dot | Tok::DotDot | Tok::LParen | Tok::LBracket);
    let tight_before = matches!(
        next.kind,
        Tok::Comma | Tok::Semi | Tok::RParen | Tok::RBracket | Tok::Dot | Tok::DotDot | Tok::Colon
    );
    let percent = prev.kind == Tok::Number && next.kind == Tok::Percent;
    let postfix_question = next.kind == Tok::Question && touching;
    let call = touching
        && matches!(
            prev.kind,
            Tok::Name | Tok::TypeName | Tok::X12 | Tok::Digits | Tok::RParen | Tok::RBracket
        )
        && matches!(next.kind, Tok::LParen | Tok::LBracket);
    let unary_minus = prev.kind == Tok::Minus && touching && next_text != "-";
    let empty_braces = prev.kind == Tok::LBrace && next.kind == Tok::RBrace;
    !(tight_after || tight_before || percent || postfix_question || call || unary_minus || empty_braces)
}

#[cfg(test)]
mod tests {
    use super::format;

    const CORPUS: &[&str] = &[
        include_str!("../../../testdata/lading/acme/partner.lading"),
        include_str!("../../../testdata/lading/acme/guides/acme-204.lading"),
        include_str!("../../../testdata/lading/acme/lib/common.lading"),
        include_str!("../../../testdata/lading/acme/maps/tender-in.lading"),
        include_str!("../../../testdata/lading/acme/maps/status-out.lading"),
        include_str!("../../../testdata/lading/acme/rules/tender-checks.lading"),
        include_str!("../../../testdata/lading/acme/flows/tender.lading"),
    ];

    #[test]
    fn formatting_is_idempotent_over_the_corpus() {
        for source in CORPUS {
            let once = format(source).expect("formats");
            let twice = format(&once).expect("formats again");
            assert_eq!(once, twice);
        }
    }

    #[test]
    fn the_corpus_is_already_formatted() {
        for source in CORPUS {
            let formatted = format(source).expect("formats");
            assert_eq!(&formatted, source, "\n{formatted}");
        }
    }

    #[test]
    fn spacing_and_indentation_are_normalised() {
        let source = "map m\nfrom loadTender\nto x12.004010.214\n\n\n\nB10{01<-proNumber,02<-key}\n0205<-each events as e{\nLX{01<-index+1}\n}\n";
        let formatted = format(source).expect("formats");
        assert_eq!(
            formatted,
            "map m\n  from loadTender\n  to   x12.004010.214\n\nB10 { 01 <- proNumber, 02 <- key }\n0205 <- each events as e {\n  LX { 01 <- index + 1 }\n}\n"
        );
    }

    #[test]
    fn a_file_with_errors_is_left_alone() {
        assert!(format("map m\n from x\n to y\nkey <- (\n").is_err());
    }
}
