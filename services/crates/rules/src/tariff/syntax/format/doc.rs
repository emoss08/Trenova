//! A small layout language and its renderer: text, line breaks that are spaces when their group
//! fits on the line, indentation, and comments that force the lines around them to break.

/// A piece of layout.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(super) enum Doc {
    /// Text with no newline.
    Text(String),
    /// A space when the enclosing group fits on the line, else a newline.
    Line,
    /// Nothing when the enclosing group fits on the line, else a newline.
    SoftLine,
    /// Always a newline; the enclosing groups break.
    HardLine,
    /// A `//` comment, written as is; the enclosing groups break, and a newline follows it.
    Comment(String),
    /// Text written only when the enclosing group breaks: a trailing comma.
    IfBreak(&'static str),
    /// The parts, indented one level after each newline.
    Nest(Vec<Doc>),
    /// The parts, laid out flat when they fit on the line, else broken.
    Group(Vec<Doc>),
    /// The parts, in order.
    Concat(Vec<Doc>),
}

impl Doc {
    pub(super) fn text(text: impl Into<String>) -> Self {
        Self::Text(text.into())
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Mode {
    Flat,
    Break,
}

const INDENT: usize = 2;

/// Lays `doc` out within `width` columns where it can.
pub(super) fn render(doc: &Doc, width: usize) -> String {
    let mut out = String::new();
    let mut column = 0usize;
    let mut pending_indent: Option<usize> = Some(0);
    let mut stack: Vec<(usize, Mode, &Doc)> = vec![(0, Mode::Break, doc)];
    while let Some((indent, mode, doc)) = stack.pop() {
        match doc {
            Doc::Text(text) | Doc::Comment(text) => {
                if text.is_empty() {
                    continue;
                }
                if let Some(spaces) = pending_indent.take() {
                    out.extend(std::iter::repeat_n(' ', spaces));
                    column = spaces;
                }
                out.push_str(text);
                column += text.chars().count();
            }
            Doc::IfBreak(text) => {
                if mode == Mode::Break {
                    out.push_str(text);
                    column += text.len();
                }
            }
            Doc::Line | Doc::SoftLine if mode == Mode::Flat => {
                if matches!(doc, Doc::Line) {
                    out.push(' ');
                    column += 1;
                }
            }
            Doc::Line | Doc::SoftLine | Doc::HardLine => {
                while out.ends_with(' ') {
                    out.pop();
                }
                out.push('\n');
                pending_indent = Some(indent);
                column = indent;
            }
            Doc::Nest(parts) => {
                for part in parts.iter().rev() {
                    stack.push((indent + INDENT, mode, part));
                }
            }
            Doc::Concat(parts) => {
                for part in parts.iter().rev() {
                    stack.push((indent, mode, part));
                }
            }
            Doc::Group(parts) => {
                let chosen =
                    if mode == Mode::Flat || fits(parts, &stack, width.saturating_sub(column)) {
                        Mode::Flat
                    } else {
                        Mode::Break
                    };
                for part in parts.iter().rev() {
                    stack.push((indent, chosen, part));
                }
            }
        }
    }
    while out.ends_with(' ') || out.ends_with('\n') {
        out.pop();
    }
    out.push('\n');
    out
}

/// Whether `parts`, laid out flat, and what follows them up to the next line break, fit in
/// `room` columns.
fn fits(parts: &[Doc], rest: &[(usize, Mode, &Doc)], room: usize) -> bool {
    let mut room = i64::try_from(room).unwrap_or(i64::MAX);
    let mut stack: Vec<(Mode, &Doc)> = parts.iter().rev().map(|part| (Mode::Flat, part)).collect();
    let mut rest = rest.iter().rev();
    loop {
        if room < 0 {
            return false;
        }
        let Some((mode, doc)) = stack
            .pop()
            .or_else(|| rest.next().map(|(_, mode, doc)| (*mode, *doc)))
        else {
            return true;
        };
        match doc {
            Doc::Text(text) => {
                room -= i64::try_from(text.chars().count()).unwrap_or(i64::MAX);
            }
            Doc::Comment(_) | Doc::HardLine => return mode == Mode::Break,
            Doc::Line => {
                if mode == Mode::Break {
                    return true;
                }
                room -= 1;
            }
            Doc::SoftLine => {
                if mode == Mode::Break {
                    return true;
                }
            }
            Doc::IfBreak(text) => {
                if mode == Mode::Break {
                    room -= i64::try_from(text.len()).unwrap_or(0);
                }
            }
            Doc::Nest(children) | Doc::Concat(children) | Doc::Group(children) => {
                for child in children.iter().rev() {
                    stack.push((mode, child));
                }
            }
        }
    }
}
