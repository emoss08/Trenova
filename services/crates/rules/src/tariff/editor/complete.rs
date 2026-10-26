//! `tariff.complete`: what may be written at a position, from what the checker knows there.
//!
//! The context is read from the tokens before the position first, since the file being typed
//! seldom parses where the cursor is, then from the nodes and types around it:
//!
//! | Before the cursor | Offered |
//! |---|---|
//! | inside a comment or the literal part of a text | nothing |
//! | `use`, `use std.`, `use org.`, `use std.x.{` | the module roots, the modules, the names a module exports |
//! | `value.`, `value?.` | the value's fields, properties and methods (an optional's after `.` rewrite the dot to `?.`) |
//! | `tables.`, `tables.Code.` | the table codes, the table methods |
//! | `Module.`, `alias.` | an enum's variants, an imported module's functions and constants |
//! | `as` in a charge, `charge(` in a test | the active accessorial codes, the labels billed |
//! | a number, `$2.35/` | the units of the dimension expected there, else every unit |
//! | `given …` | the record's fields along the path, `input.` and the inputs |
//! | a type annotation | the types in scope, and units after `Money/` |
//! | `tables.Code[` | the axis's keys from the cells the context carries, then values |
//! | a call's `(`, `,` | the parameters not given yet as `name:`, then values for the parameter |
//! | a record literal's `{` | the fields of the record expected there not written yet |
//! | the start of a statement | the keywords and snippets valid there, then values |
//! | anywhere else a value goes | names in scope with their types, functions, enum variants (the expected enum's first) and the value keywords |
//!
//! Items are not filtered by what has been typed (the editor filters as the author types); each
//! carries a `sort` key that puts what fits the expected type first. Nothing is offered that
//! would not check there: a function body is not offered the record or the file's bindings, a
//! test is not offered the inputs, a bare variant two enums share is offered qualified.

use super::cursor::{self, Body, Bracket, Located, Scopes, line_start, previous, word_at};
use super::describe::{function_signature, method_signature, type_text};
use super::names;
use super::session::Session;
use trenova_module_kit::span::Span;
use crate::tariff::check::describe::snippet as source_text;
use crate::tariff::check::lower::BUILTIN_TYPES;
use crate::tariff::check::methods::{self, ParamKind, Receiver};
use crate::tariff::check::{
    CONSTRUCTORS, Callee, REQUESTED, REQUESTED_DOC, REQUESTED_SIGNATURE, SymbolKind, TABLE_METHODS,
};
use crate::tariff::docs::Doc;
use crate::tariff::stdlib;
use crate::tariff::syntax::ast::{Expr, ExprKind, FileKind, ItemKind, StmtKind};
use crate::tariff::syntax::program::{Decl, ModuleId};
use crate::tariff::syntax::token::{Token, TokenKind};
use crate::tariff::types::Type;
use crate::tariff::units::{Dim, SIMPLE_UNITS, Simple};
use std::collections::BTreeSet;
use trenova_module_kit::message::Message;

/// What a completion item is, for its icon.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Kind {
    /// A keyword.
    Keyword,
    /// A local name or a schema root.
    Name,
    /// An `input`.
    Input,
    /// A function, a conversion, a test constructor or `charge(…)`.
    Function,
    /// A built-in method.
    Method,
    /// A built-in property.
    Property,
    /// A record field, or a parameter written `name:`.
    Field,
    /// A type.
    Type,
    /// An enum type.
    Enum,
    /// An enum variant.
    Variant,
    /// A unit.
    Unit,
    /// An accessorial charge code.
    Code,
    /// A rate table, or one of its methods.
    Table,
    /// A module.
    Module,
    /// A snippet of several tokens.
    Snippet,
    /// A literal value: a table key, a charge label.
    Value,
}

impl Kind {
    /// The kind's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Keyword => "keyword",
            Self::Name => "name",
            Self::Input => "input",
            Self::Function => "function",
            Self::Method => "method",
            Self::Property => "property",
            Self::Field => "field",
            Self::Type => "type",
            Self::Enum => "enum",
            Self::Variant => "variant",
            Self::Unit => "unit",
            Self::Code => "code",
            Self::Table => "table",
            Self::Module => "module",
            Self::Snippet => "snippet",
            Self::Value => "value",
        }
    }
}

/// One completion item.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CompletionItem {
    /// What the list shows.
    pub label: String,
    /// What it is.
    pub kind: Kind,
    /// Its type or signature.
    pub detail: Option<String>,
    /// Its documentation.
    pub doc: Option<Doc>,
    /// The text that replaces the range.
    pub insert: String,
    /// Whether `insert` is a snippet with `${1:placeholder}` tab stops.
    pub snippet: bool,
    /// The order: items sort by this key.
    pub sort: String,
    /// The text it replaces, when not the completion's range (`.` rewritten to `?.`).
    pub range: Option<Span>,
}

/// The answer: the text the items replace and the items.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Completion {
    /// The text the items replace: the word being typed, or the position.
    pub range: Span,
    /// The items, in no particular order (see [`CompletionItem::sort`]).
    pub items: Vec<CompletionItem>,
}

/// How strongly an item fits, the first part of its sort key.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub enum Rank {
    /// Fits the type expected there.
    Expected = 0,
    /// A local name.
    Local = 1,
    /// A parameter name to write.
    Parameter = 2,
    /// A file-level name.
    Global = 3,
    /// A keyword or snippet.
    Keyword = 4,
    /// A schema root or module.
    Root = 5,
    /// An enum variant that is not the expected type's.
    Variant = 6,
    /// A type or anything else.
    Other = 7,
}

/// The items being collected, one per label.
#[derive(Debug, Default)]
pub struct Items {
    items: Vec<CompletionItem>,
    labels: BTreeSet<String>,
}

impl Items {
    /// Adds an item unless one with the same label is already there, and returns it to fill in.
    pub fn add(&mut self, rank: Rank, kind: Kind, label: &str) -> Option<&mut CompletionItem> {
        if label.is_empty() || !self.labels.insert(label.to_owned()) {
            return None;
        }
        self.items.push(CompletionItem {
            label: label.to_owned(),
            kind,
            detail: None,
            doc: None,
            insert: label.to_owned(),
            snippet: false,
            sort: format!("{}{}", rank as u8, label.to_lowercase()),
            range: None,
        });
        self.items.last_mut()
    }

    /// How many items there are.
    #[must_use]
    pub fn len(&self) -> usize {
        self.items.len()
    }

    /// Whether there are none.
    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.items.is_empty()
    }
}

/// What every context reads.
#[derive(Debug)]
pub struct Site<'s, 'c> {
    /// The file.
    pub session: &'s Session<'c>,
    /// The byte position.
    pub offset: u32,
    /// Where the word being typed starts (the position when none is).
    pub start: u32,
    /// The token before the word, comments skipped.
    pub before: Option<usize>,
    /// The nodes around the position.
    pub located: Located<'s>,
    /// Every local name of the file.
    pub scopes: Scopes<'s>,
}

impl Site<'_, '_> {
    /// The root file's tokens.
    pub fn tokens(&self) -> &[Token] {
        &self.session.file().tokens
    }

    /// A token's text.
    pub fn text(&self, index: usize) -> &str {
        self.tokens()
            .get(index)
            .map_or("", |token| source_text(self.session.source(), token.span))
    }

    /// The token before the word, when it is punctuation or a keyword spelt `text`.
    pub fn before_is(&self, text: &str) -> bool {
        self.before.is_some_and(|index| self.text(index) == text)
    }

    /// The token at `index`.
    pub fn token(&self, index: usize) -> Option<&Token> {
        self.tokens().get(index)
    }

    /// The first token of the line the word is on.
    pub fn line_first(&self) -> Option<usize> {
        let index = self
            .tokens()
            .partition_point(|token| token.span.end <= self.start);
        let first = line_start(self.tokens(), index);
        let token = self.token(first)?;
        (token.span.start <= self.start && token.kind != TokenKind::Newline).then_some(first)
    }

    /// Whether the line the word is on starts with the keyword `word`.
    pub fn line_starts_with(&self, word: &str) -> bool {
        self.line_first().is_some_and(|first| {
            self.token(first)
                .is_some_and(|token| token.span.start < self.start)
                && self.text(first) == word
        })
    }

    /// The file's kind.
    pub fn kind(&self) -> FileKind {
        self.session.kind()
    }
}

/// Completes at byte `offset` of the session's root file.
#[must_use]
pub fn complete(session: &Session<'_>, offset: u32) -> Completion {
    let file = session.file();
    let tokens = &file.tokens;
    if cursor::in_literal(session.source(), tokens, offset) {
        return Completion {
            range: Span::new(offset, offset),
            items: Vec::new(),
        };
    }
    let word = word_at(tokens, offset).and_then(|index| tokens.get(index));
    let range = word.map_or(Span::new(offset, offset), |token| token.span);
    let start = range.start;
    let located = cursor::locate(session.source(), file, offset);
    session.tick(located.visited);
    let scopes = Scopes::of(session.source(), file);
    session.tick(scopes.locals.len());
    let site = Site {
        session,
        offset,
        start,
        before: cursor::before(tokens, start),
        located,
        scopes,
    };
    let mut items = Items::default();
    collect(&site, &mut items);
    session.tick(items.len());
    Completion {
        range,
        items: items.items,
    }
}

fn collect(site: &Site<'_, '_>, items: &mut Items) {
    if site.line_starts_with("use") {
        use_path(site, items);
        return;
    }
    if site.line_starts_with("given") && given_path(site, items, "=") {
        return;
    }
    if in_property(site) {
        if site.line_starts_with("vary")
            && (after_vary_target(site, items) || given_path(site, items, "in"))
        {
            return;
        }
        if site.line_starts_with("expect") && monotone(site, items) {
            return;
        }
    }
    if site.before_is(".") || site.before_is("?.") {
        member(site, items);
        return;
    }
    if site.before_is("as") && site.line_starts_with("charge") {
        codes(site, items);
        return;
    }
    if input_tail(site, items) {
        return;
    }
    if after_number(site, items) {
        return;
    }
    if type_position(site) {
        types(site, items);
        return;
    }
    if let Some(matched) = pattern_position(site) {
        patterns(items, matched.ty.as_ref());
        return;
    }
    if site.before_is("}") && after_if_block(site) {
        names::keyword(items, Rank::Keyword, "else");
        return;
    }
    let mut fallback = None;
    if let Some(bracket) = cursor::enclosing(site.session.source(), site.tokens(), site.start) {
        if bracket.kind == '{' {
            if record_fields(site, &bracket, items) {
                return;
            }
        } else {
            match inside(site, &bracket, items) {
                Then::Values(expected) => fallback = expected,
                Then::Done => return,
            }
        }
    }
    if statement_start(site) && !names::statement(site, items) {
        return;
    }
    let expected = expected_here(site).or(fallback);
    names::expression(site, items, expected.as_ref());
}

/// The type expected for the expression at the position, as the checker recorded it.
pub fn expected_here(site: &Site<'_, '_>) -> Option<Type> {
    let analysis = site.session.analysis.module(ModuleId::ROOT)?;
    let at = |expr: &Expr| analysis.expected_at(expr.id).cloned();
    node_at(site, site.start, site.offset).and_then(at)
}

/// The innermost name, variant or number written from `start`, or the missing value the
/// parser put where the position is (a missing value sits at the next token, which may be on
/// a later line when the line ends with an operator or `=`).
fn node_at<'s>(site: &Site<'s, '_>, start: u32, end: u32) -> Option<&'s Expr> {
    let source = site.session.source();
    let before = source.get(..start as usize).unwrap_or_default();
    let after = source.get(start as usize..).unwrap_or_default();
    let low = start.saturating_sub(gap(before.chars().rev(), false));
    let high = start.saturating_add(gap(after.chars(), true));
    let typing = start < end;
    site.scopes.find_last(|expr| match &expr.kind {
        ExprKind::Name(_) | ExprKind::Path { .. } | ExprKind::Number { .. } if typing => {
            expr.span.start == start && start < expr.span.end
        }
        ExprKind::Missing => {
            (low <= expr.span.start && expr.span.start <= high)
                || (expr.span.start <= start && end <= expr.span.end)
        }
        _ => !typing && start <= expr.span.start && expr.span.start <= high,
    })
}

/// How many bytes of spaces (and, with `lines`, newlines) a text starts with.
fn gap(text: impl Iterator<Item = char>, lines: bool) -> u32 {
    let count: usize = text
        .take_while(|c| *c == ' ' || *c == '\t' || (lines && (*c == '\n' || *c == '\r')))
        .map(char::len_utf8)
        .sum();
    u32::try_from(count).unwrap_or(0)
}

/// Whether the word starts a statement: the first token of its line (or of a block), with no
/// unclosed `(` or `[` before it on the statement.
fn statement_start(site: &Site<'_, '_>) -> bool {
    let Some(before) = site.before else {
        return true;
    };
    let token = site.token(before);
    let starts = token.is_some_and(|token| token.kind == TokenKind::Newline)
        || matches!(site.text(before), "{" | "}");
    if !starts {
        return false;
    }
    match cursor::enclosing(site.session.source(), site.tokens(), site.start) {
        Some(bracket) => bracket.kind == '{' && !is_record(site, bracket.open),
        None => true,
    }
}

fn is_record(site: &Site<'_, '_>, open: usize) -> bool {
    let Some(token) = site.token(open) else {
        return false;
    };
    let start = token.span.start;
    site.scopes
        .find(|expr| expr.span.start == start && matches!(expr.kind, ExprKind::Record(_)))
        .is_some()
}

/// Whether the `}` before the word closes an `if` block that has no `else` yet.
fn after_if_block(site: &Site<'_, '_>) -> bool {
    let Some(close) = site.before.and_then(|index| site.token(index)) else {
        return false;
    };
    let end = close.span.end;
    let located = cursor::locate(
        site.session.source(),
        site.session.file(),
        end.saturating_sub(1),
    );
    site.session.tick(located.visited);
    located.chain.iter().any(|node| {
        matches!(node, cursor::Node::Stmt(stmt) if matches!(&stmt.kind, StmtKind::If(statement)
            if statement.otherwise.is_none()
                && statement.branches.last().is_some_and(|(_, block)| block.span.end == end)))
    })
}

/// Where a `match` arm's pattern is written: the type matched, when the checker knows it.
fn pattern_position(site: &Site<'_, '_>) -> Option<Matched> {
    let chain = &site.located.chain;
    let at = chain.iter().rposition(|node| {
        matches!(node, cursor::Node::Stmt(stmt) if matches!(stmt.kind, StmtKind::Match(_)))
            || matches!(node, cursor::Node::Expr(expr) if matches!(expr.kind, ExprKind::Match { .. }))
    })?;
    let scrutinee = match chain.get(at)? {
        cursor::Node::Stmt(stmt) => match &stmt.kind {
            StmtKind::Match(statement) => &statement.scrutinee,
            _ => return None,
        },
        cursor::Node::Expr(expr) => match &expr.kind {
            ExprKind::Match { scrutinee, .. } => scrutinee,
            _ => return None,
        },
        _ => return None,
    };
    if site.start <= scrutinee.span.end {
        return None;
    }
    let inner = chain.get(at + 1..).unwrap_or_default();
    let in_pattern = match inner {
        [] => true,
        [cursor::Node::Arm(arm), rest @ ..] => {
            arm.pattern.span.start <= site.start
                && site.start <= arm.pattern.span.end
                && rest
                    .iter()
                    .all(|node| matches!(node, cursor::Node::Pattern(_)))
        }
        _ => false,
    };
    in_pattern.then(|| Matched {
        ty: site
            .session
            .analysis
            .type_of(ModuleId::ROOT, scrutinee.id)
            .cloned(),
    })
}

/// The value a `match` whose pattern is being written matches.
struct Matched {
    /// Its type, when the checker knows it.
    ty: Option<Type>,
}

/// What follows an argument list's own items.
#[derive(Debug)]
pub enum Then {
    /// Nothing: the items are complete.
    Done,
    /// Values, of the type the active parameter takes when known.
    Values(Option<Type>),
}

/// What a pattern can be: the matched enum's variants, `_`, `none` for an optional, and
/// `true`/`false` for a `Bool`.
fn patterns(items: &mut Items, ty: Option<&Type>) {
    if let Some(Type::Enum(enumeration)) = ty.map(Type::unwrapped) {
        for (variant, doc) in &enumeration.variants {
            if let Some(item) = items.add(Rank::Expected, Kind::Variant, variant) {
                item.detail = Some(type_text(&Type::Enum(enumeration.clone())));
                item.doc.clone_from(doc);
            }
        }
    }
    if let Some(Type::Record(record)) = ty.map(Type::unwrapped)
        && let Some(item) = items.add(Rank::Expected, Kind::Snippet, "{ … }")
    {
        let fields: Vec<String> = record
            .fields
            .iter()
            .take(1)
            .map(|field| format!("{}: ${{1:_}}", field.name))
            .collect();
        item.insert = format!("{{ {} }}", fields.join(", "));
        item.snippet = true;
    }
    if matches!(ty.map(Type::unwrapped), Some(Type::Bool)) {
        names::keyword(items, Rank::Expected, "true");
        names::keyword(items, Rank::Expected, "false");
    }
    if let Some(item) = items.add(Rank::Keyword, Kind::Keyword, "_") {
        item.doc = Some(Doc::Message(Message::new(trenova_module_kit::message!(
            "Matches anything not matched above."
        ))));
    }
    if ty.is_none_or(Type::is_optional) {
        names::keyword(items, Rank::Keyword, "none");
    }
}

/// `use std.fuel.{ … }`: the module roots, the modules and the names a module exports.
fn use_path(site: &Site<'_, '_>, items: &mut Items) {
    let Some(first) = site.line_first() else {
        return;
    };
    let end = site
        .tokens()
        .partition_point(|token| token.span.end <= site.start);
    let words: Vec<String> = (first + 1..end)
        .map(|index| site.text(index).to_owned())
        .filter(|word| !word.trim().is_empty())
        .collect();
    let open = words.iter().position(|word| word == "{");
    let path: Vec<&str> = words
        .iter()
        .take(open.unwrap_or(words.len()))
        .map(String::as_str)
        .filter(|word| *word != ".")
        .collect();
    if let Some(open) = open {
        let written: Vec<&str> = words.iter().skip(open + 1).map(String::as_str).collect();
        exports(site, items, &path.join("."), &written);
        return;
    }
    if words.last().is_some_and(|word| word == "as") {
        return;
    }
    if path.is_empty() {
        for (root, doc) in [
            (
                "std",
                Message::new(trenova_module_kit::message!(
                    "The standard library, shipped with the module."
                )),
            ),
            (
                "org",
                Message::new(trenova_module_kit::message!(
                    "Your organisation's libraries."
                )),
            ),
        ] {
            if let Some(item) = items.add(Rank::Root, Kind::Module, root) {
                item.doc = Some(Doc::Message(doc));
            }
        }
        return;
    }
    let prefix = format!("{}.", path.join("."));
    let modules = stdlib::MODULES
        .iter()
        .map(|(module, source)| ((*module).to_owned(), Some(*source)))
        .chain(
            site.session
                .context
                .libraries
                .iter()
                .map(|library| (library.path.clone(), None)),
        );
    for (module, source) in modules {
        let Some(rest) = module.strip_prefix(&prefix) else {
            continue;
        };
        let name = rest.split('.').next().unwrap_or(rest);
        if let Some(item) = items.add(Rank::Root, Kind::Module, name) {
            item.detail = Some(module.clone());
            item.doc = source
                .map(crate::tariff::syntax::parse)
                .and_then(|file| Doc::of(true, file.docs.as_ref()));
        }
    }
}

/// The public names a module declares, minus those already written.
fn exports(site: &Site<'_, '_>, items: &mut Items, path: &str, written: &[&str]) {
    let Some(module) = site.session.program.module_named(path) else {
        return;
    };
    for (name, symbol) in &module.symbols {
        if symbol.private
            || written.contains(&name.as_str())
            || matches!(symbol.decl, Decl::Input(_))
        {
            continue;
        }
        let kind = match symbol.decl {
            Decl::Fn(_) => Kind::Function,
            Decl::Type(_) => Kind::Type,
            Decl::Enum(_) => Kind::Enum,
            _ => Kind::Name,
        };
        let id = site.session.declared_at(module.id, symbol.span);
        if let Some(item) = items.add(Rank::Global, kind, name)
            && let Some(id) = id
        {
            item.detail = Some(super::describe::symbol_signature(site.session, id));
            item.doc = site.session.symbol(id).and_then(|s| s.doc.clone());
        }
    }
}

/// After `.` or `?.`: what the value before it has.
fn member(site: &Site<'_, '_>, items: &mut Items) {
    let Some(dot) = site.before.and_then(|index| site.token(index)) else {
        return;
    };
    let dot_span = dot.span;
    let optional_dot = site.before_is("?.");
    if let Some(head) = qualified_head(site, dot_span) {
        enum_variants(site, items, &head);
        return;
    }
    let Some(target) = target_of(site, dot_span) else {
        return;
    };
    let session = site.session;
    let sees_record = matches!(site.located.body, Body::Top | Body::Test);
    match &target.kind {
        ExprKind::Name(name) if name.text == "tables" && sees_record => {
            for table in &session.context.tables {
                if let Some(item) = items.add(Rank::Expected, Kind::Table, &table.code) {
                    item.detail = Some(table_detail(table));
                    item.doc = (!table.name.is_empty()).then(|| Doc::Text(table.name.clone()));
                }
            }
            return;
        }
        ExprKind::Field {
            target: root,
            name,
            optional: false,
        } if matches!(&root.kind, ExprKind::Name(root) if root.text == "tables") && sees_record => {
            if session
                .context
                .tables
                .iter()
                .any(|table| table.code == name.text)
            {
                for (method, doc) in TABLE_METHODS {
                    if let Some(item) = items.add(Rank::Expected, Kind::Table, method) {
                        item.insert = format!("{method}($0)");
                        item.snippet = true;
                        item.doc = Some(Doc::Message(Message::new(doc)));
                    }
                }
            }
            return;
        }
        ExprKind::Name(name)
            if name.text == "bill"
                && site.located.body == Body::Top
                && site.kind() == FileKind::Tariff
                && alias_module(site, &name.text).is_none() =>
        {
            bill_members(items);
            return;
        }
        ExprKind::Name(name) => {
            if let Some(module) = alias_module(site, &name.text) {
                module_members(site, items, module);
                return;
            }
        }
        ExprKind::Path { head, member: None } => {
            enum_variants(site, items, &head.text);
            return;
        }
        _ => {}
    }
    let Some(ty) = session.analysis.type_of(ModuleId::ROOT, target.id).cloned() else {
        return;
    };
    let rewrite = (!optional_dot && ty.is_optional())
        .then(|| Span::new(dot_span.start, site.offset.max(dot_span.end)));
    members(items, ty.unwrapped(), rewrite);
    if session.schema.root("shipment") == Some(ty.unwrapped())
        && let Some(item) = items.add(Rank::Local, Kind::Method, REQUESTED)
    {
        item.detail = Some(String::from(REQUESTED_SIGNATURE));
        item.doc = Some(Doc::Message(Message::new(REQUESTED_DOC)));
        item.insert = format!("{REQUESTED}($0)");
        item.snippet = true;
        if let Some(span) = rewrite {
            item.insert = format!("?.{}", item.insert);
            item.range = Some(span);
        }
    }
}

/// What `bill.` reads: the sums of the bill so far and `charge(…)`.
fn bill_members(items: &mut Items) {
    members(
        items,
        &Type::Record(crate::tariff::schema::bill_type()),
        None,
    );
    if let Some(item) = items.add(Rank::Expected, Kind::Method, "charge") {
        item.insert = String::from("charge($0)");
        item.snippet = true;
        item.detail = Some(String::from("charge(line: CODE | Text) -> Money?"));
        item.doc = Some(Doc::message(crate::tariff::schema::BILL_CHARGE_DOC));
    }
}

/// The expression a `.` at `dot` reads from.
fn target_of<'s>(site: &Site<'s, '_>, dot: Span) -> Option<&'s Expr> {
    let field = site.scopes.find_last(|expr| {
        matches!(&expr.kind, ExprKind::Field { target, name, .. }
            if target.span.end <= dot.start
                && name.span.start >= dot.end
                && name.span.start <= dot.end.saturating_add(MAX_GAP))
    })?;
    match &field.kind {
        ExprKind::Field { target, .. } => Some(target),
        _ => None,
    }
}

/// The type name of `Enum.Variant` when the `.` at `dot` is that one's.
fn qualified_head(site: &Site<'_, '_>, dot: Span) -> Option<String> {
    let path = site.scopes.find_last(|expr| {
        matches!(&expr.kind, ExprKind::Path { head, member: Some(member) }
            if head.span.end <= dot.start
                && member.span.start >= dot.end
                && member.span.start <= dot.end.saturating_add(MAX_GAP))
    })?;
    match &path.kind {
        ExprKind::Path { head, .. } => Some(head.text.clone()),
        _ => None,
    }
}

/// The most spaces between a `.` and the name after it.
const MAX_GAP: u32 = 8;

/// The module a name is an alias of, in the root file.
fn alias_module(site: &Site<'_, '_>, name: &str) -> Option<ModuleId> {
    if site
        .scopes
        .visible(site.offset, site.located.body)
        .iter()
        .any(|local| local.name == name)
    {
        return None;
    }
    site.session
        .root()
        .aliases
        .get(name)
        .map(|alias| alias.module)
}

/// A module's public functions and constants, for `alias.`.
fn module_members(site: &Site<'_, '_>, items: &mut Items, module: ModuleId) {
    let Some(found) = site.session.program.module(module) else {
        return;
    };
    for (name, symbol) in &found.symbols {
        let kind = match symbol.decl {
            Decl::Fn(_) => Kind::Function,
            Decl::Const(_) => Kind::Name,
            _ => continue,
        };
        if symbol.private {
            continue;
        }
        let Some(id) = site.session.declared_at(module, symbol.span) else {
            continue;
        };
        if let Some(item) = items.add(Rank::Global, kind, name) {
            item.detail = Some(super::describe::symbol_signature(site.session, id));
            item.doc = site.session.symbol(id).and_then(|s| s.doc.clone());
            if kind == Kind::Function {
                fill_call(site, item, module, name);
            }
        }
    }
}

/// Makes a function item insert a call, with a tab stop for each parameter it must be given.
pub fn fill_call(site: &Site<'_, '_>, item: &mut CompletionItem, module: ModuleId, name: &str) {
    let Some(decl) = super::describe::function_decl(site.session, module, name) else {
        item.insert = format!("{name}()");
        return;
    };
    let source = site
        .session
        .program
        .module(module)
        .map_or("", |found| found.source.as_str());
    let required: Vec<(&str, String)> = decl
        .params
        .iter()
        .filter(|param| param.default.is_none())
        .map(|param| {
            (
                param.name.text.as_str(),
                super::describe::flat(source_text(source, param.ty.span)),
            )
        })
        .collect();
    let (args, snippet) = call_args(decl.params.len(), &required);
    item.insert = format!("{name}{args}");
    item.snippet = snippet;
    item.detail = Some(function_signature(site.session, module, decl).label);
}

/// A call's arguments as a snippet, and whether it has tab stops: `()` for a call that takes
/// nothing, the cursor between the parentheses when every parameter has a default, a
/// placeholder showing the type of a lone required parameter (`round(${1:Money})`), or each
/// required parameter by name (`deadhead(empty: ${1:Distance}, rate: ${2:Money/mi})`).
pub fn call_args<T: AsRef<str>>(params: usize, required: &[(&str, T)]) -> (String, bool) {
    if params == 0 {
        return (String::from("()"), false);
    }
    if required.is_empty() {
        return (String::from("($0)"), true);
    }
    let field =
        |index: usize, name: &str, ty: &str| format!("${{{index}:{}}}", placeholder(name, ty));
    let args: Vec<String> = match required {
        [(name, ty)] => vec![field(1, name, ty.as_ref())],
        _ => required
            .iter()
            .enumerate()
            .map(|(index, (name, ty))| format!("{name}: {}", field(index + 1, name, ty.as_ref())))
            .collect(),
    };
    (format!("({})", args.join(", ")), true)
}

/// A placeholder's text, escaped for a snippet: the parameter's type, or its name when the type
/// is a record written in braces, which a placeholder can't hold.
fn placeholder(name: &str, ty: &str) -> String {
    let text = if ty.is_empty() || ty.contains(['{', '}']) {
        name
    } else {
        ty
    };
    text.replace('\\', "\\\\")
        .replace('$', "\\$")
        .replace('}', "\\}")
}

/// The variants of the enum named `name`, for `Enum.`.
fn enum_variants(site: &Site<'_, '_>, items: &mut Items, name: &str) {
    let Some(enumeration) = names::enums(site.session)
        .into_iter()
        .find(|(written, _)| written == name)
        .map(|(_, e)| e)
    else {
        return;
    };
    for (variant, doc) in &enumeration.variants {
        if let Some(item) = items.add(Rank::Expected, Kind::Variant, variant) {
            item.detail = Some(type_text(&Type::Enum(enumeration.clone())));
            item.doc.clone_from(doc);
        }
    }
}

/// The fields, properties and methods of a value of type `ty`. `rewrite` is the text of a `.`
/// read from an optional, which each item replaces with `?.name`.
pub fn members(items: &mut Items, ty: &Type, rewrite: Option<Span>) {
    if let Type::Record(record) = ty {
        for field in &record.fields {
            if let Some(item) = items.add(Rank::Expected, Kind::Field, &field.name) {
                item.detail = Some(type_text(&field.ty));
                item.doc.clone_from(&field.doc);
                if let Some(span) = rewrite {
                    item.insert = format!("?.{}", field.name);
                    item.range = Some(span);
                }
            }
        }
    }
    let Some(receiver) = Receiver::of(ty) else {
        return;
    };
    for method in methods::of(receiver) {
        let kind = if method.property {
            Kind::Property
        } else {
            Kind::Method
        };
        let rank = if method.property {
            Rank::Expected
        } else {
            Rank::Local
        };
        let Some(item) = items.add(rank, kind, method.name) else {
            continue;
        };
        item.detail = Some(method_signature(method, ty).label);
        item.doc = Some(Doc::Message(Message::new(method.doc)));
        if !method.property {
            let (insert, snippet) = method_insert(method, ty);
            item.insert = insert;
            item.snippet = snippet;
        }
        if let Some(span) = rewrite {
            item.insert = format!("?.{}", item.insert);
            item.range = Some(span);
        }
    }
}

/// A method call to insert: `floor()`, `max(${1:Money})`, `where(${1:s} => $0)`.
fn method_insert(method: &methods::Method, ty: &Type) -> (String, bool) {
    let Some(first) = method.params.first() else {
        return (format!("{}()", method.name), false);
    };
    let lambda = matches!(
        first.kind,
        ParamKind::Predicate
            | ParamKind::Map
            | ParamKind::Summable
            | ParamKind::Ordered
            | ParamKind::Key
    );
    if lambda {
        let name = match ty {
            Type::List(item) => item
                .to_string()
                .chars()
                .next()
                .map_or('x', |c| c.to_ascii_lowercase()),
            _ => 'x',
        };
        let name = if name.is_ascii_alphabetic() {
            name
        } else {
            'x'
        };
        return (format!("{}(${{1:{name}}} => $0)", method.name), true);
    }
    let required: Vec<(&str, String)> = method
        .params
        .iter()
        .filter(|param| param.default.is_none())
        .map(|param| {
            let shown = if param.ty == "Self" {
                type_text(ty.unwrapped())
            } else {
                param.ty.to_owned()
            };
            (param.name, shown)
        })
        .collect();
    let (args, snippet) = call_args(method.params.len(), &required);
    (format!("{}{args}", method.name), snippet)
}

/// A table's axes and value type, for its detail.
fn table_detail(table: &crate::tariff::schema::TableShape) -> String {
    let axes: Vec<String> = table
        .axes
        .iter()
        .map(|axis| {
            if axis.label.is_empty() {
                axis.kind.clone()
            } else {
                axis.label.clone()
            }
        })
        .collect();
    format!("[{}] -> {}?", axes.join(", "), table.value_type)
}

/// After `as` in a charge, and in `shipment.requested(…)`: the active accessorial codes.
pub fn codes(site: &Site<'_, '_>, items: &mut Items) {
    let Some(codes) = &site.session.context.codes else {
        return;
    };
    for code in codes.iter().filter(|code| code.active) {
        if let Some(item) = items.add(Rank::Expected, Kind::Code, &code.code) {
            item.detail = (!code.description.is_empty()).then(|| code.description.clone());
            item.doc = (!code.description.is_empty()).then(|| Doc::Text(code.description.clone()));
        }
    }
}

/// After a number, or `$x/`: the units. Returns whether this is that context.
fn after_number(site: &Site<'_, '_>, items: &mut Items) -> bool {
    let Some(before) = site.before else {
        return false;
    };
    let Some(token) = site.token(before) else {
        return false;
    };
    let after_money = site.text(before) == "/"
        && previous(site.tokens(), before)
            .and_then(|index| site.token(index))
            .is_some_and(|money| {
                money.kind == TokenKind::Money && money.span.end == token.span.start
            });
    let after_count = token.kind == TokenKind::Number && site.start > token.span.end;
    if !after_money && !after_count {
        return false;
    }
    let wanted = if after_count {
        number_dim(site, token.span.start)
    } else {
        None
    };
    units(items, wanted, after_money);
    true
}

/// The dimension expected for the number literal starting at `start`.
fn number_dim(site: &Site<'_, '_>, start: u32) -> Option<Dim> {
    let analysis = site.session.analysis.module(ModuleId::ROOT)?;
    let number = site.scopes.find_last(|expr| {
        expr.span.start == start && matches!(expr.kind, ExprKind::Number { .. })
    })?;
    match analysis.expected_at(number.id)?.unwrapped() {
        Type::Quantity(quantity) => Some(quantity.dim),
        Type::Int | Type::Number => Some(Dim::NONE),
        _ => None,
    }
}

/// The units of a dimension (`Dim::NONE` for the count labels), or every unit.
pub fn units(items: &mut Items, wanted: Option<Dim>, per: bool) {
    for unit in SIMPLE_UNITS {
        if unit.symbol == "$" || (per && unit.symbol == "%") {
            continue;
        }
        let dim = unit.base.map_or(Dim::NONE, Dim::of);
        let rank = match wanted {
            Some(wanted) if wanted == dim => Rank::Expected,
            Some(_) => continue,
            None => Rank::Other,
        };
        if wanted == Some(Dim::NONE) && unit.symbol == "%" {
            continue;
        }
        add_unit(items, rank, unit);
    }
}

fn add_unit(items: &mut Items, rank: Rank, unit: &Simple) {
    if let Some(item) = items.add(rank, Kind::Unit, unit.symbol) {
        item.detail = Some(match unit.base {
            Some(base) => base.name().to_owned(),
            None if unit.symbol == "%" => String::from("Percent"),
            None => String::from("Count"),
        });
        item.doc = Some(Doc::Message(super::describe::unit_doc(unit)));
    }
}

/// Whether the position is in a `property` block.
fn in_property(site: &Site<'_, '_>) -> bool {
    site.located
        .item()
        .is_some_and(|item| matches!(item.kind, ItemKind::Property(_)))
}

/// The words of the line before the word being typed, after its first, `.` and blanks left
/// out.
fn line_words<'s>(site: &'s Site<'_, '_>) -> Vec<&'s str> {
    let Some(first) = site.line_first() else {
        return Vec::new();
    };
    let end = site
        .tokens()
        .partition_point(|token| token.span.end <= site.start);
    (first + 1..end)
        .map(|index| site.text(index))
        .filter(|word| *word != "." && !word.trim().is_empty())
        .collect()
}

/// Whether a word is being started after a space, past the end of the token before it.
fn after_space(site: &Site<'_, '_>) -> bool {
    site.before
        .and_then(|index| site.token(index))
        .is_some_and(|token| token.span.end < site.start && token.kind != TokenKind::Newline)
}

/// `vary target |`: the `in` that gives its values. Returns whether this is that context.
fn after_vary_target(site: &Site<'_, '_>, items: &mut Items) -> bool {
    let words = line_words(site);
    let target_done = site
        .before
        .and_then(|index| site.token(index))
        .is_some_and(|token| matches!(token.kind, TokenKind::Name | TokenKind::Keyword))
        && !site.before_is("vary")
        && !site.before_is("input")
        && !site.before_is("in");
    if words.is_empty() || words.contains(&"in") || !target_done || !after_space(site) {
        return false;
    }
    names::keyword(items, Rank::Expected, "in");
    true
}

/// In a property's `expect`: `nondecreasing` and `nonincreasing` after a value, `in` after
/// them, then the targets the property varies. Returns whether this is that context.
fn monotone(site: &Site<'_, '_>, items: &mut Items) -> bool {
    let words = line_words(site);
    let direction = |word: &str| matches!(word, "nondecreasing" | "nonincreasing");
    if words.last().copied().is_some_and(direction) && after_space(site) {
        names::keyword(items, Rank::Expected, "in");
        return true;
    }
    let before_in = words
        .len()
        .checked_sub(2)
        .and_then(|at| words.get(at))
        .copied();
    if site.before_is("in") && before_in.is_some_and(direction) {
        varied_targets(site, items);
        return true;
    }
    let value_ends = site
        .before
        .and_then(|index| site.token(index))
        .is_some_and(|token| {
            matches!(
                token.kind,
                TokenKind::Name
                    | TokenKind::Number
                    | TokenKind::Money
                    | TokenKind::Percent
                    | TokenKind::Text
                    | TokenKind::Date
                    | TokenKind::Unit
                    | TokenKind::Code
            ) || matches!(site.text(site.before.unwrap_or(0)), ")" | "]")
        });
    if words.is_empty()
        || words.iter().any(|word| direction(word))
        || !value_ends
        || !after_space(site)
    {
        return false;
    }
    names::keyword(items, Rank::Expected, "nondecreasing");
    names::keyword(items, Rank::Expected, "nonincreasing");
    true
}

/// The targets the property at the position varies, as a monotone `expect` names them.
fn varied_targets(site: &Site<'_, '_>, items: &mut Items) {
    let Some(ItemKind::Property(property)) = site.located.item().map(|item| &item.kind) else {
        return;
    };
    for stmt in &property.body.stmts {
        let StmtKind::Vary { target, .. } = &stmt.kind else {
            continue;
        };
        let written: Vec<&str> = target.iter().map(|part| part.text.as_str()).collect();
        if written.iter().any(|part| part.is_empty()) {
            continue;
        }
        if let Some(item) = items.add(Rank::Expected, Kind::Value, &written.join(".")) {
            item.detail = Some(String::from("vary"));
        }
    }
}

/// After an input's type or default: `where` and its snippet. Returns whether nothing else
/// fits (after the type, before any default).
fn input_tail(site: &Site<'_, '_>, items: &mut Items) -> bool {
    let Some(ItemKind::Input(input)) = site.located.item().map(|item| &item.kind) else {
        return false;
    };
    if !site.line_starts_with("input")
        || !after_space(site)
        || input
            .constraint
            .as_ref()
            .is_some_and(|constraint| constraint.span.start <= site.start)
        || matches!(
            input.ty.kind,
            crate::tariff::syntax::ast::TypeExprKind::Missing
        )
    {
        return false;
    }
    let before_end = site
        .before
        .and_then(|index| site.token(index))
        .map_or(0, |token| token.span.end);
    let after_type = input.default.is_none() && before_end == input.ty.span.end;
    let after_default = input
        .default
        .as_ref()
        .is_some_and(|default| before_end == default.span.end);
    if !after_type && !after_default {
        return false;
    }
    names::keyword(items, Rank::Expected, "where");
    if let Some(item) = items.add(Rank::Keyword, Kind::Snippet, "where …..…") {
        item.insert = String::from("where ${1:low}..=${2:high}");
        item.snippet = true;
        item.doc = Some(Doc::Message(Message::new(trenova_module_kit::message!(
            "Bounds the values a contract may set: a range, or a list after `where`."
        ))));
    }
    after_type
}

/// `given …` or `vary …`: the fields along the path, or `input.` and the inputs, and for a
/// `given` the rate tables a test may give cells of. Returns whether the position is in the path
/// (before `stop`, the `=` or `in` that ends it, or a cell's `[`, whose keys are values).
fn given_path(site: &Site<'_, '_>, items: &mut Items, stop: &str) -> bool {
    let Some(first) = site.line_first() else {
        return false;
    };
    let end = site
        .tokens()
        .partition_point(|token| token.span.end <= site.start);
    let words: Vec<&str> = (first + 1..end)
        .map(|index| site.text(index))
        .filter(|word| *word != "." && !word.trim().is_empty())
        .collect();
    if words.contains(&stop) || words.contains(&"[") {
        return false;
    }
    let session = site.session;
    let cells = stop == "=";
    let Some((head, rest)) = words.split_first() else {
        for (root, ty, doc) in &session.schema.roots {
            if let Some(item) = items.add(Rank::Root, Kind::Name, root) {
                item.detail = Some(type_text(ty));
                item.doc = Some(doc.clone());
            }
        }
        if matches!(session.kind(), FileKind::Tariff | FileKind::Condition)
            && let Some(item) = items.add(Rank::Root, Kind::Keyword, "input")
        {
            item.doc = Some(Doc::Message(Message::new(trenova_module_kit::message!(
                "Sets one of the file's inputs for this test: `given input.rate = $2.10/mi`."
            ))));
        }
        if cells && let Some(item) = items.add(Rank::Root, Kind::Name, "tables") {
            item.doc = Some(Doc::Message(Message::new(trenova_module_kit::message!(
                "Gives a rate table's cell for this test: `given tables.Zones[\"A\", \"B\"] = $20/cwt`. The table then holds only the cells the test gives."
            ))));
        }
        return true;
    };
    if *head == "tables" {
        if cells && rest.is_empty() {
            for table in &session.context.tables {
                if let Some(item) = items.add(Rank::Expected, Kind::Table, &table.code) {
                    item.detail = Some(table_detail(table));
                    item.doc = (!table.name.is_empty()).then(|| Doc::Text(table.name.clone()));
                }
            }
        }
        return true;
    }
    if *head == "input" {
        if rest.is_empty() {
            names::inputs(site, items, Rank::Expected);
        }
        return true;
    }
    let Some(mut ty) = session.schema.root(head).cloned() else {
        return true;
    };
    for part in rest {
        let Type::Record(record) = ty.unwrapped() else {
            return true;
        };
        let Some(field) = record.field(part) else {
            return true;
        };
        ty = field.ty.clone();
    }
    if let Type::Record(record) = ty.unwrapped() {
        for field in &record.fields {
            if let Some(item) = items.add(Rank::Expected, Kind::Field, &field.name) {
                item.detail = Some(type_text(&field.ty));
                item.doc.clone_from(&field.doc);
            }
        }
    }
    true
}

/// Whether the word is where a type is written.
fn type_position(site: &Site<'_, '_>) -> bool {
    if site.located.ty().is_some() && site.located.body == Body::Declaration {
        return true;
    }
    if site.before_is("->") {
        return true;
    }
    if site.before_is("<") || site.before_is("?") {
        return site.located.ty().is_some();
    }
    if site.before_is("/") {
        return site.located.ty().is_some();
    }
    if !site.before_is(":") {
        return false;
    }
    let Some(colon) = site.before else {
        return false;
    };
    let declaration_line = ["input", "let", "var", "const", "type"]
        .iter()
        .any(|word| site.line_starts_with(word));
    let name_before = previous(site.tokens(), colon).is_some_and(|index| {
        previous(site.tokens(), index)
            .is_some_and(|keyword| matches!(site.text(keyword), "input" | "let" | "var" | "const"))
    });
    if declaration_line && name_before {
        return true;
    }
    if let Some(cursor::Node::Item(item)) = site.located.chain.first() {
        if matches!(item.kind, ItemKind::Type(_)) {
            return true;
        }
        if let ItemKind::Fn(decl) = &item.kind {
            let body_start = match &decl.body {
                crate::tariff::syntax::ast::FnBody::Block(block) => block.span.start,
                crate::tariff::syntax::ast::FnBody::Expr(expr) => expr.span.start,
            };
            let params_end = decl
                .params
                .last()
                .map_or(decl.name.span.end, |param| param.span.end);
            return site.start <= body_start.max(params_end) && site.start > decl.name.span.end;
        }
    }
    false
}

/// The types in scope, or after `Money/` the units.
fn types(site: &Site<'_, '_>, items: &mut Items) {
    if site.before_is("/") {
        units(items, None, true);
        return;
    }
    if let Some(item) = items.add(Rank::Expected, Kind::Type, "List") {
        item.insert = String::from("List<$0>");
        item.snippet = true;
        item.doc = BUILTIN_TYPES
            .iter()
            .find(|(name, _)| *name == "List")
            .map(|(_, doc)| Doc::Message(Message::new(doc)));
    }
    for (name, doc) in BUILTIN_TYPES {
        if let Some(item) = items.add(Rank::Expected, Kind::Type, name) {
            item.doc = Some(Doc::Message(Message::new(doc)));
        }
    }
    for record in &site.session.schema.records {
        if let Some(name) = &record.name
            && let Some(item) = items.add(Rank::Expected, Kind::Type, name)
        {
            item.detail = Some(String::from("Schema"));
        }
    }
    for (name, enumeration) in names::enums(site.session) {
        if let Some(item) = items.add(Rank::Expected, Kind::Enum, &name) {
            item.detail = Some(format!("enum {name}"));
            item.doc.clone_from(&enumeration.doc);
        }
    }
    let root = site.session.root();
    let mut declared: Vec<(String, ModuleId)> = root
        .symbols
        .iter()
        .filter(|(_, symbol)| matches!(symbol.decl, Decl::Type(_)))
        .map(|(name, _)| (name.clone(), ModuleId::ROOT))
        .collect();
    declared.extend(
        root.imports
            .iter()
            .map(|(name, import)| (name.clone(), import.module)),
    );
    for (name, module) in declared {
        let Some(id) = site.session.global(ModuleId::ROOT, &name) else {
            continue;
        };
        let Some(symbol) = site.session.symbol(id) else {
            continue;
        };
        if symbol.kind != SymbolKind::Type {
            continue;
        }
        if let Some(item) = items.add(Rank::Expected, Kind::Type, &name) {
            item.detail = Some(super::describe::symbol_signature(site.session, id));
            item.doc.clone_from(&symbol.doc);
        }
        let _ = module;
    }
}

/// Inside `(…)` or `[…]`: parameters, table keys, units, and what follows them.
fn inside(site: &Site<'_, '_>, bracket: &Bracket, items: &mut Items) -> Then {
    let Some(open) = site.token(bracket.open) else {
        return Then::Done;
    };
    let open_start = open.span.start;
    let at_part_start = site.before == Some(bracket.open) || site.before_is(",");
    if bracket.kind == '[' {
        names::table_keys(site, items, open_start, bracket.part);
        return Then::Values(None);
    }
    let before_open = previous(site.tokens(), bracket.open);
    let after_label = before_open
        .and_then(|index| site.token(index))
        .is_some_and(|token| token.kind == TokenKind::Text);
    if after_label && site.line_starts_with("charge") {
        if bracket.current_name().is_some() {
            return Then::Values(None);
        }
        if at_part_start {
            charge_options(items, bracket);
        }
        return Then::Done;
    }
    let Some(call) = call_with_open(&site.scopes, open_start) else {
        return Then::Values(None);
    };
    if super::choices::is_chosen(site.session, call) {
        names::choice_labels(site, items);
        return Then::Done;
    }
    names::arguments(site, items, call, bracket, at_part_start)
}

/// A charge's `quantity:`, `rate:`, `minimum:` and `maximum:` options not written yet.
fn charge_options(items: &mut Items, bracket: &Bracket) {
    for (option, doc) in [
        (
            "quantity",
            trenova_module_kit::message!(
                "How much is billed: the invoice line's quantity, in the rate's unit."
            ),
        ),
        (
            "rate",
            trenova_module_kit::message!(
                "The price per unit of the quantity: the invoice line's unit price."
            ),
        ),
        (
            "minimum",
            trenova_module_kit::message!(
                "The least the line bills: a smaller amount is raised to it, and the line says the minimum applied."
            ),
        ),
        (
            "maximum",
            trenova_module_kit::message!(
                "The most the line bills: a larger amount is lowered to it, and the line says the maximum applied."
            ),
        ),
    ] {
        if bracket.named.iter().any(|(_, name)| name == option) {
            continue;
        }
        if let Some(item) = items.add(Rank::Parameter, Kind::Field, &format!("{option}:")) {
            item.insert = format!("{option}: ");
            item.doc = Some(Doc::Message(Message::new(doc)));
        }
    }
}

/// The call whose `(` is at `open`.
pub fn call_with_open<'a>(scopes: &Scopes<'a>, open: u32) -> Option<&'a Expr> {
    scopes.find_last(|expr| {
        matches!(&expr.kind, ExprKind::Call { callee, .. }
            if callee.span.end <= open
                && open <= callee.span.end.saturating_add(MAX_GAP)
                && expr.span.end > open)
    })
}

/// Inside a record literal's `{`: the fields of the record expected there not written yet.
fn record_fields(site: &Site<'_, '_>, bracket: &Bracket, items: &mut Items) -> bool {
    if bracket.kind != '{' || bracket.current_name().is_some() {
        return false;
    }
    let at_part_start = site.before == Some(bracket.open) || site.before_is(",");
    if !at_part_start {
        return false;
    }
    let Some(open) = site.token(bracket.open) else {
        return false;
    };
    let start = open.span.start;
    let Some(record) = site
        .scopes
        .find_last(|expr| expr.span.start == start && matches!(expr.kind, ExprKind::Record(_)))
    else {
        return false;
    };
    let analysis = site.session.analysis.module(ModuleId::ROOT);
    let ty = analysis
        .and_then(|analysis| {
            analysis
                .expected_at(record.id)
                .or_else(|| analysis.type_of(record.id))
        })
        .cloned();
    let Some(Type::Record(shape)) = ty.as_ref().map(Type::unwrapped) else {
        return false;
    };
    let ExprKind::Record(written) = &record.kind else {
        return false;
    };
    for field in &shape.fields {
        if written.iter().any(|init| init.name.text == field.name) {
            continue;
        }
        if let Some(item) = items.add(Rank::Parameter, Kind::Field, &format!("{}:", field.name)) {
            item.insert = format!("{}: ", field.name);
            item.detail = Some(type_text(&field.ty));
            item.doc.clone_from(&field.doc);
        }
    }
    true
}

/// A callee's resolution from the checker, or, when the call did not check, from its shape.
pub fn callee_of(session: &Session<'_>, call: &Expr) -> Option<Callee> {
    if let Some(callee) = session
        .analysis
        .module(ModuleId::ROOT)
        .and_then(|analysis| analysis.callees.get(&call.id))
    {
        return Some(callee.clone());
    }
    let ExprKind::Call { callee, .. } = &call.kind else {
        return None;
    };
    match &callee.kind {
        ExprKind::Name(name) => {
            if let Some((constructor, _)) = CONSTRUCTORS
                .iter()
                .find(|(constructor, _)| *constructor == name.text)
            {
                return Some(Callee::Constructor(constructor));
            }
            let id = session.global(ModuleId::ROOT, &name.text)?;
            let symbol = session.symbol(id)?;
            (symbol.kind == SymbolKind::Function).then(|| Callee::Function {
                module: symbol.module.unwrap_or(ModuleId::ROOT),
                name: name.text.clone(),
            })
        }
        ExprKind::Field { target, name, .. } => {
            if let ExprKind::Name(alias) = &target.kind
                && let Some(module) = session.root().aliases.get(&alias.text)
            {
                return Some(Callee::Function {
                    module: module.module,
                    name: name.text.clone(),
                });
            }
            let ty = session.analysis.type_of(ModuleId::ROOT, target.id)?;
            methods::find(ty.unwrapped(), &name.text).map(Callee::Method)
        }
        _ => None,
    }
}
