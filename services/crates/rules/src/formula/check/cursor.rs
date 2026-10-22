//! What is under the editor's cursor: the call it is inside (signature help) and the name it is
//! on (the hover card).

use super::knowledge::Knowledge;
use super::knowledge::Schema;
use crate::formula::catalog;
use crate::formula::syntax;
use crate::formula::syntax::Program;
use crate::formula::syntax::ast::Call;
use crate::formula::syntax::ast::Expr;
use crate::formula::syntax::ast::ExprKind;
use crate::formula::syntax::ast::TokenKind;
use crate::formula::syntax::lexer;
use crate::span::Span;
use std::collections::HashSet;

/// The call the cursor is inside and which argument it is on.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Signature {
    /// The function's name, as the catalog lists it.
    pub name: &'static str,
    /// Which argument the cursor is in, from 0; for a pipe the piped value is argument 0.
    pub active: usize,
    /// The call's parentheses, from `(` to `)`; signature help anchors at the start.
    pub parens: Span,
}

/// The innermost call around `position` (a byte offset) whose function the catalog has.
#[must_use]
pub fn signature_at(source: &str, position: u32, operators: &[&str]) -> Option<Signature> {
    let program = syntax::parse(source, operators);
    let path = node_path(&program, position);
    for node in path.iter().rev() {
        let ExprKind::Call(call) = &node.kind else {
            continue;
        };
        let closed = call.parens.end > call.parens.start
            && source.as_bytes().get(call.parens.end as usize - 1) == Some(&b')');
        let last = call.parens.end - u32::from(closed);
        if position <= call.parens.start || position > last {
            continue;
        }
        let Some(function) = catalog::callable(&call.function.text) else {
            continue;
        };
        return Some(Signature {
            name: function.name,
            active: active_argument(call, position, source),
            parens: call.parens,
        });
    }
    None
}

fn active_argument(call: &Call, position: u32, source: &str) -> usize {
    let mut active = 0;
    for (index, arg) in call.args.iter().enumerate() {
        if position > arg.span.end {
            let until = position.min(call.parens.end);
            let between = source
                .get(arg.span.end as usize..until as usize)
                .unwrap_or_default();
            active = if between.contains(',') {
                index + 1
            } else {
                index
            };
        } else {
            return index;
        }
    }
    active
}

/// The innermost node whose span holds `position`, with its ancestors outermost first.
fn node_path(program: &Program, position: u32) -> Vec<&Expr> {
    let mut path = Vec::new();
    let Some(statement) = program
        .statements
        .iter()
        .find(|statement| statement.span.start <= position && position <= statement.span.end)
    else {
        return path;
    };
    let mut current = Some(&statement.value);
    while let Some(node) = current {
        if position < node.span.start || position > node.span.end {
            break;
        }
        path.push(node);
        current = node
            .children()
            .into_iter()
            .find(|child| child.span.start <= position && position <= child.span.end);
    }
    path
}

/// One name the text reads, dotted paths joined.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct IdentifierToken {
    /// The name, such as `customer.name`.
    pub name: String,
    /// Where it is written.
    pub span: Span,
    /// It is followed by `(`.
    pub is_call: bool,
    /// It follows a `.` that does not join it to a name before it: an item's field (`.weight`)
    /// or a member of something that is not a name (`stops[0].weight`).
    pub after_dot: bool,
}

/// The names a formula reads, dotted paths joined, with whether each is called. Built on the
/// lexer, so strings and comments never count.
#[must_use]
pub fn identifier_tokens(source: &str) -> Vec<IdentifierToken> {
    let tokens = lexer::tokenize(source, &[]).tokens;
    let text = |index: usize| {
        tokens
            .get(index)
            .and_then(|token| source.get(token.span.start as usize..token.span.end as usize))
            .unwrap_or_default()
    };
    let is =
        |index: usize, kind: TokenKind| tokens.get(index).is_some_and(|token| token.kind == kind);
    let mut out = Vec::new();
    let mut index = 0;
    while index < tokens.len() {
        if !is(index, TokenKind::Ident) {
            index += 1;
            continue;
        }
        let start = tokens[index].span.start;
        let mut end = tokens[index].span.end;
        let mut name = text(index).to_owned();
        let mut cursor = index + 1;
        while is(cursor, TokenKind::Punct)
            && text(cursor) == "."
            && tokens
                .get(cursor + 1)
                .is_some_and(|next| next.span.start == tokens[cursor].span.end)
            && (is(cursor + 1, TokenKind::Ident) || is(cursor + 1, TokenKind::Keyword))
        {
            name.push('.');
            name.push_str(text(cursor + 1));
            end = tokens[cursor + 1].span.end;
            cursor += 2;
        }
        let after_dot = index > 0
            && ((is(index - 1, TokenKind::Punct) && text(index - 1) == ".")
                || (is(index - 1, TokenKind::Operator) && text(index - 1) == "?."));
        out.push(IdentifierToken {
            name,
            span: Span::new(start, end),
            is_call: is(cursor, TokenKind::Punct) && text(cursor) == "(",
            after_dot,
        });
        index = cursor;
    }
    out
}

/// What a hovered name is.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum IdentifierKind {
    /// A field of the record.
    Variable,
    /// One of the template's inputs.
    Input,
    /// A function call.
    Function,
    /// A method call on a value, such as `pickupDate.Year()`.
    Method,
    /// A name the formula binds with `let`.
    Let,
    /// `#`, `#acc`, `#index`.
    Pointer,
    /// A field of an item (`.weight`) or of a value that is not a name.
    Field,
    /// A name the schema does not have.
    Unknown,
}

impl IdentifierKind {
    /// The kind's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Variable => "variable",
            Self::Input => "input",
            Self::Function => "function",
            Self::Method => "method",
            Self::Let => "let",
            Self::Pointer => "pointer",
            Self::Field => "field",
            Self::Unknown => "unknown",
        }
    }
}

/// The name under the cursor.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Identifier {
    /// The name, dotted paths joined: `customer.name`.
    pub name: String,
    /// What it is.
    pub kind: IdentifierKind,
    /// Where it is written.
    pub span: Span,
}

/// The name at `position` (a byte offset), if the position is on one. Without a schema a name
/// that is neither a call nor a `let` is a `Variable`.
#[must_use]
pub fn identifier_at(source: &str, position: u32, schema: Option<&Schema>) -> Option<Identifier> {
    let found = identifier_tokens(source)
        .into_iter()
        .find(|token| token.span.start <= position && position < token.span.end);
    let Some(token) = found else {
        return pointer_at(source, position);
    };
    let root = token.name.split('.').next().unwrap_or_default().to_owned();
    let kind = if token.is_call {
        if token.after_dot || token.name.contains('.') {
            IdentifierKind::Method
        } else {
            IdentifierKind::Function
        }
    } else if token.after_dot {
        IdentifierKind::Field
    } else if let_names(source).contains(&root) {
        IdentifierKind::Let
    } else if let Some(schema) = schema {
        let knowledge = Knowledge::new(schema);
        match knowledge
            .top
            .doc(&token.name)
            .or_else(|| knowledge.top.doc(&root))
        {
            Some(doc) if doc.input => IdentifierKind::Input,
            Some(_) => IdentifierKind::Variable,
            None if knowledge.top.is_root(&root) => IdentifierKind::Variable,
            None => IdentifierKind::Unknown,
        }
    } else {
        IdentifierKind::Variable
    };
    Some(Identifier {
        name: token.name,
        kind,
        span: token.span,
    })
}

fn pointer_at(source: &str, position: u32) -> Option<Identifier> {
    let token = lexer::tokenize(source, &[])
        .tokens
        .into_iter()
        .find(|token| {
            token.kind == TokenKind::Pointer
                && token.span.start <= position
                && position < token.span.end
        })?;
    Some(Identifier {
        name: source
            .get(token.span.start as usize..token.span.end as usize)
            .unwrap_or_default()
            .to_owned(),
        kind: IdentifierKind::Pointer,
        span: token.span,
    })
}

/// The names the formula binds with `let`, anywhere in it.
fn let_names(source: &str) -> HashSet<String> {
    let tokens = lexer::tokenize(source, &[]).tokens;
    let text = |span: Span| {
        source
            .get(span.start as usize..span.end as usize)
            .unwrap_or_default()
    };
    tokens
        .windows(2)
        .filter(|pair| {
            pair[0].kind == TokenKind::Keyword
                && text(pair[0].span) == "let"
                && pair[1].kind == TokenKind::Ident
        })
        .map(|pair| text(pair[1].span).to_owned())
        .collect()
}
