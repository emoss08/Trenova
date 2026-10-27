//! Completion: what may be written at the cursor, read from the tokens before it and what the
//! checker learned about the scope there.

use serde::Serialize;
use trenova_module_kit::span::Span;

use super::describe;
use super::session::{Doc, Session};
use crate::lading::check::x12 as nav;
use crate::lading::check::{Decl, Program};
use crate::lading::schema;
use crate::lading::stdlib::{self, Receiver};
use crate::lading::syntax::ast::{Header, Item, Stmt};
use crate::lading::syntax::token::Tok;
use crate::lading::types::{NodeRef, Place, Type};
use crate::x12::guide::compiled::NodeKind;

/// One completion.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Item2 {
    /// What the list shows.
    pub label: String,
    /// What it is: `field`, `variable`, `function`, `method`, `segment`, `element`, `loop`,
    /// `keyword`, `type`, `variant`, `crosswalk`, `constant`, `setting`, `event`, `map`,
    /// `rules`, `lookup`, `file`, `snippet`.
    pub kind: &'static str,
    /// One line beside it.
    pub detail: String,
    /// What it is for.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub doc: Option<Doc>,
    /// What is inserted.
    pub insert: String,
    /// Whether `insert` has tab stops.
    pub snippet: bool,
    /// Its order.
    pub sort: String,
}

/// Completions at a byte offset, and the span they replace.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Completion {
    /// What a chosen item replaces.
    pub range: Span,
    /// The items.
    pub items: Vec<Item2>,
}

fn item(
    label: impl Into<String>,
    kind: &'static str,
    detail: impl Into<String>,
    doc: Option<Doc>,
    sort: &str,
) -> Item2 {
    let label = label.into();
    Item2 {
        insert: label.clone(),
        label,
        kind,
        detail: detail.into(),
        doc,
        snippet: false,
        sort: sort.to_owned(),
    }
}

fn snippet(label: &str, kind: &'static str, detail: &str, insert: &str, sort: &str) -> Item2 {
    Item2 {
        label: label.to_owned(),
        kind,
        detail: detail.to_owned(),
        doc: None,
        insert: insert.to_owned(),
        snippet: true,
        sort: sort.to_owned(),
    }
}

/// What can be written at `offset`.
#[must_use]
#[allow(clippy::too_many_lines)]
pub fn complete(session: &Session, offset: u32) -> Completion {
    if let Some(found) = quoted_codes(session, offset) {
        return found;
    }
    let word = session.word_at(offset);
    let range = word.map_or(Span::new(offset, offset), |token| {
        Span::new(token.span.start, offset.max(token.span.start))
    });
    let start = range.start;
    let before = session.before(start);
    let previous = before
        .iter()
        .rev()
        .find(|token| token.kind != Tok::Newline)
        .copied();
    let at_line_start = before
        .last()
        .is_none_or(|token| matches!(token.kind, Tok::Newline | Tok::LBrace | Tok::Semi));
    let previous_word = previous.map_or("", |token| session.text(token.span));
    let program = &session.program;
    let header = program.units[session.unit].header().clone();
    let mut items = Vec::new();
    if previous.is_some_and(|token| token.kind == Tok::Dot) {
        let dot = previous.map_or(start, |token| token.span.start);
        members(session, dot, &mut items);
    } else if previous_word == "via" && matches!(header, Header::Flow { .. }) {
        let inbound = session.clause_word(start) == "receive";
        for (name, info) in &program.maps {
            if info.inbound() == inbound {
                items.push(item(
                    name,
                    "map",
                    describe::file(program, name).unwrap_or_default(),
                    None,
                    "0",
                ));
            }
        }
    } else if previous_word == "via" {
        crosswalks(session, &mut items);
    } else if previous_word == "rules" && matches!(header, Header::Flow { .. }) {
        for name in program.rules.keys() {
            items.push(item(
                name,
                "rules",
                describe::file(program, name).unwrap_or_default(),
                None,
                "0",
            ));
        }
    } else if previous_word == "lookup" {
        for lookup in schema::LOOKUPS {
            items.push(item(
                lookup.name,
                "lookup",
                lookup.entity,
                Some(Doc::message(lookup.doc)),
                "0",
            ));
        }
    } else if previous_word == "unless" {
        for (state, doc) in schema::STATES {
            items.push(item(*state, "keyword", "", Some(Doc::message(doc)), "0"));
        }
    } else if previous_word == "on" && session.clause_word(start) == "receive" {
        if let Some(purposes) = schema::enumeration("TenderPurpose") {
            for (variant, doc) in purposes.variants {
                items.push(item(
                    *variant,
                    "variant",
                    "TenderPurpose",
                    Some(Doc::message(doc)),
                    "0",
                ));
            }
        }
    } else if matches!(previous_word, "on" | "until" | "event")
        || (previous_word == "," && session.clause_word(start) == "send")
    {
        for event in schema::EVENTS {
            items.push(item(
                event.name,
                "event",
                event.document,
                Some(Doc::message(event.doc)),
                "0",
            ));
        }
    } else if matches!(previous_word, "use")
        || (previous_word == "," && session.line_word(start) == "use")
    {
        for name in program.libraries.keys() {
            items.push(item(name, "file", "library", None, "0"));
        }
    } else if matches!(previous_word, "from" | "to")
        && matches!(header, Header::Map { .. } | Header::Guide { .. })
    {
        for name in program.guide_names.keys() {
            items.push(item(name, "file", "guide", None, "0"));
        }
        for guide in crate::x12::guide::builtin::all() {
            items.push(item(
                &guide.guide().id,
                "file",
                guide.guide().name.clone(),
                None,
                "1",
            ));
        }
        for document in schema::DOCUMENTS {
            items.push(item(
                document.name,
                "type",
                document.set.unwrap_or_default(),
                Some(Doc::message(document.doc)),
                "0",
            ));
        }
    } else if previous.is_some_and(|token| token.kind == Tok::Colon)
        && matches!(
            session.line_word(start).as_str(),
            "fn" | "setting" | "crosswalk"
        )
        || previous_word == "->" && session.line_word(start) == "fn"
    {
        types(&mut items);
    } else if previous_word == "given" {
        for word in ["file", "text", "setting", "lookup", "now"] {
            items.push(item(word, "keyword", "", None, "0"));
        }
        for document in schema::DOCUMENTS {
            items.push(item(
                document.name,
                "type",
                "",
                Some(Doc::message(document.doc)),
                "1",
            ));
        }
    } else if previous_word == "expect" {
        for word in [
            "x12",
            "no",
            "held",
            "rejected",
            "declined",
            "warned",
            "sent",
            "escalated",
            "created",
            "updated",
            "cancelled",
        ] {
            items.push(item(word, "keyword", "", None, "1"));
        }
        expression(session, start, &mut items);
    } else if let Some(segment) = segment_block(session, start) {
        elements(session, segment, &mut items);
    } else if at_line_start {
        statement(session, start, &header, &mut items);
    } else {
        expression(session, start, &mut items);
    }
    let typed = session.text(range).to_owned();
    if !typed.is_empty() {
        let lower = typed.to_lowercase();
        items.retain(|candidate| candidate.label.to_lowercase().contains(&lower));
    }
    items.sort_by(|a, b| a.sort.cmp(&b.sort).then_with(|| a.label.cmp(&b.label)));
    items.dedup_by(|a, b| a.label == b.label && a.kind == b.kind);
    Completion { range, items }
}

/// The codes of the element a text is compared with, when the cursor is inside it:
/// `L11[02 == "|"]`, `qualifier != "|"`, `d.01 in ["37", "|"]`.
fn quoted_codes(session: &Session, offset: u32) -> Option<Completion> {
    let text = session.tokens.iter().copied().find(|token| {
        token.kind == Tok::Text && token.span.start < offset && offset < token.span.end
    })?;
    let before = session.before(text.span.start);
    let mut tokens = before
        .iter()
        .rev()
        .copied()
        .filter(|token| token.kind != Tok::Newline);
    let mut previous = tokens.next()?;
    if matches!(previous.kind, Tok::Comma | Tok::LBracket) {
        while previous.kind != Tok::LBracket {
            previous = tokens
                .next()
                .filter(|token| matches!(token.kind, Tok::Text | Tok::Comma | Tok::LBracket))?;
        }
        let keyword = tokens.next()?;
        if session.text(keyword.span) != "in" {
            return None;
        }
    } else if !matches!(previous.kind, Tok::EqEq | Tok::NotEq) {
        return None;
    }
    let subject = tokens.next()?;
    let typed = session
        .program
        .facts
        .type_ending_at(session.unit, subject.span.end)?;
    let Type::Code(element, place) = typed.ty.unwrapped() else {
        return None;
    };
    let codes = element_codes(&session.program, element, *place)?;
    let range = Span::new(text.span.start + 1, offset);
    let typed = session.text(range).to_lowercase();
    let items = codes
        .into_iter()
        .enumerate()
        .filter(|(_, (code, description))| {
            typed.is_empty()
                || code.to_lowercase().starts_with(&typed)
                || description.to_lowercase().contains(&typed)
        })
        .map(|(order, (code, description))| {
            item(code, "constant", description, None, &format!("{order:05}"))
        })
        .collect();
    Some(Completion {
        range: Span::new(range.start, text.span.end - 1),
        items,
    })
}

/// The codes an element takes, from the node its value was read at when the checker knew it;
/// otherwise from the first use of its segment that lists codes for it, the project's own guides
/// before the built-in ones.
fn element_codes(program: &Program, element: &str, place: Place) -> Option<Vec<(String, String)>> {
    let Some(nav::Reference::Element(segment, position)) = nav::reference(element) else {
        return None;
    };
    if let Place(Some(reference)) = place {
        return nav::codes(
            &program.guides[reference.guide].compiled,
            Some(reference.node),
            segment,
            position,
        );
    }
    let (own, builtin): (Vec<_>, Vec<_>) = program
        .guides
        .iter()
        .partition(|entry| entry.unit.is_some());
    own.into_iter().chain(builtin).find_map(|entry| {
        let compiled = &entry.compiled;
        (0..compiled.node_count())
            .filter(
                |node| matches!(&compiled.node(*node).kind, NodeKind::Segment(id) if id == segment),
            )
            .find_map(|node| nav::codes(compiled, Some(node), segment, position))
            .filter(|codes| !codes.is_empty())
    })
}

fn types(items: &mut Vec<Item2>) {
    for name in [
        "Text",
        "Int",
        "Decimal",
        "Bool",
        "Date",
        "Time",
        "DateTime",
        "Zone",
        "Duration",
        "Percent",
        "Money",
        "Weight",
        "Length",
        "Volume",
        "Temperature",
        "Json",
    ] {
        items.push(item(name, "type", "", None, "0"));
    }
    items.push(snippet("Code<...>", "type", "", "Code<${1:S502}>", "1"));
    items.push(snippet(
        "Segment<...>",
        "type",
        "",
        "Segment<${1:G62}>",
        "1",
    ));
    items.push(snippet("Loop<...>", "type", "", "Loop<${1:0310}>", "1"));
    for record in schema::RECORDS {
        items.push(item(
            record.name,
            "type",
            "",
            Some(Doc::message(record.doc)),
            "2",
        ));
    }
    for enumeration in schema::ENUMS {
        items.push(item(
            enumeration.name,
            "type",
            "",
            Some(Doc::message(enumeration.doc)),
            "2",
        ));
    }
    for lookup in schema::LOOKUPS {
        items.push(item(
            lookup.name,
            "type",
            "",
            Some(Doc::message(lookup.doc)),
            "2",
        ));
    }
}

fn crosswalks(session: &Session, items: &mut Vec<Item2>) {
    let program = &session.program;
    let mut units = vec![session.unit];
    for library in program.units[session.unit].header().uses() {
        if let Some(unit) = program.libraries.get(&library.name) {
            units.push(*unit);
        }
    }
    for unit in units {
        for (name, decl) in &program.units[unit].decls {
            if matches!(decl, Decl::Crosswalk(_)) {
                let (detail, doc) = describe::decl(program, unit, name).unwrap_or_default();
                items.push(item(name, "crosswalk", detail, doc, "0"));
            }
        }
    }
}

/// The segment block the cursor is in, between its braces.
fn segment_block(session: &Session, offset: u32) -> Option<NodeRef> {
    fn find(stmts: &[Stmt], offset: u32) -> Option<&Stmt> {
        for stmt in stmts {
            match stmt {
                Stmt::Segment { span, .. } if span.start < offset && offset < span.end => {
                    return Some(stmt);
                }
                Stmt::Loop { body, .. } | Stmt::When { body, .. } | Stmt::Each { body, .. } => {
                    if let Some(found) = find(&body.stmts, offset) {
                        return Some(found);
                    }
                }
                _ => {}
            }
        }
        None
    }
    let program = &session.program;
    let stmts: Vec<Stmt> = program.units[session.unit]
        .items()
        .iter()
        .filter_map(|item| match item {
            Item::Stmt(stmt) => Some(stmt.clone()),
            _ => None,
        })
        .collect();
    let Stmt::Segment { id, .. } = find(&stmts, offset)? else {
        return None;
    };
    let brace = session.source()[..offset as usize].rfind('{')?;
    if brace < id.span.end as usize {
        return None;
    }
    program
        .facts
        .occurrences
        .iter()
        .find_map(|occurrence| match occurrence.target {
            crate::lading::facts::Target::Segment(reference)
                if occurrence.unit == session.unit && occurrence.span == id.span =>
            {
                Some(reference)
            }
            _ => None,
        })
}

fn elements(session: &Session, segment: NodeRef, items: &mut Vec<Item2>) {
    let program = &session.program;
    let compiled = &program.guides[segment.guide].compiled;
    let Some(definition) = nav::definition(compiled, segment.node) else {
        return;
    };
    let id = nav::segment_id(compiled, segment.node).to_owned();
    for (offset, element) in definition.elements.iter().enumerate() {
        let position = u32::try_from(offset + 1).unwrap_or(0);
        let (_, detail, doc) = describe::element_parts(&id, position, element);
        let mut entry = item(
            format!("{position:02}"),
            "element",
            detail,
            Some(doc),
            &format!("{position:03}"),
        );
        entry.insert = format!("{position:02} <- ");
        items.push(entry);
    }
}

#[allow(clippy::too_many_lines)]
fn statement(session: &Session, offset: u32, header: &Header, items: &mut Vec<Item2>) {
    let program = &session.program;
    let scope = session.scope(offset);
    match header {
        Header::Map { name, .. } => {
            let inbound = program
                .maps
                .get(&name.name)
                .is_none_or(super::super::check::MapInfo::inbound);
            if inbound {
                let record = scope
                    .and_then(|scope| match &scope.builds {
                        Some(Type::Record(record)) => schema::record(record),
                        _ => None,
                    })
                    .or_else(|| {
                        program
                            .maps
                            .get(&name.name)
                            .and_then(|info| info.document)
                            .and_then(schema::record)
                    });
                if let Some(record) = record {
                    for field in record.fields {
                        let (detail, doc) = describe::field(program, field);
                        let sort = if field.required { "0" } else { "1" };
                        let mut entry = item(field.name, "field", detail, Some(doc), sort);
                        entry.insert = match field.spec {
                            schema::Spec::Records(_) => format!(
                                "{} <- each ${{1:list}} as ${{2:item}} {{\n  $0\n}}",
                                field.name
                            ),
                            _ => format!("{} <- $0", field.name),
                        };
                        entry.snippet = true;
                        items.push(entry);
                    }
                }
                items.push(snippet("let", "keyword", "", "let ${1:name} = $0", "2"));
                items.push(snippet(
                    "when",
                    "keyword",
                    "",
                    "when ${1:condition} {\n  $0\n}",
                    "2",
                ));
            } else {
                let guide = scope.and_then(|scope| scope.guide).or_else(|| {
                    program
                        .maps
                        .get(&name.name)
                        .and_then(|info| info.target_guide)
                });
                if let Some(guide) = guide {
                    let parent = scope.and_then(|scope| scope.x12).map(|node| node.node);
                    children(program, guide, parent, true, items);
                }
                items.push(snippet(
                    "each",
                    "keyword",
                    "",
                    "each ${1:list} as ${2:item} {\n  $0\n}",
                    "2",
                ));
                items.push(snippet(
                    "when",
                    "keyword",
                    "",
                    "when ${1:condition} {\n  $0\n}",
                    "2",
                ));
                items.push(snippet("let", "keyword", "", "let ${1:name} = $0", "2"));
            }
        }
        Header::Rules { .. } => {
            items.push(snippet(
                "rule",
                "keyword",
                "",
                "rule \"${1:title}\" {\n  $0\n}",
                "0",
            ));
            items.push(snippet(
                "require",
                "keyword",
                "",
                "require ${1:condition}\n  else ${2|reject,decline,hold,warn|} \"${3:message}\"",
                "0",
            ));
            items.push(snippet(
                "for",
                "keyword",
                "",
                "for ${1:item} in ${2:list} {\n  $0\n}",
                "0",
            ));
            items.push(snippet("fix", "keyword", "", "fix ${1:field} <- $0", "0"));
            items.push(snippet(
                "warn",
                "keyword",
                "",
                "warn \"${1:message}\" when $0",
                "1",
            ));
            items.push(snippet(
                "hold",
                "keyword",
                "",
                "hold \"${1:message}\" when $0",
                "1",
            ));
            items.push(snippet("let", "keyword", "", "let ${1:name} = $0", "2"));
        }
        Header::Flow { .. } => {
            items.push(snippet("receive", "keyword", "", "receive ${1:204} as ${2:tender}\n  via ${3:map}\n  key $2.key\n  acknowledge with 997\n  on Original -> create", "0"));
            items.push(snippet(
                "respond",
                "keyword",
                "",
                "respond 990 via ${1:map}\n  otherwise planner decides within ${2:2 h}",
                "0",
            ));
            items.push(snippet(
                "send",
                "keyword",
                "",
                "send ${1:214} via ${2:map} on ${3:StopArrived}",
                "0",
            ));
            items.push(snippet(
                "expect",
                "keyword",
                "",
                "expect 997 for every send within ${1:30 min} else escalate \"${2:message}\"",
                "0",
            ));
        }
        Header::Partner { .. } => {
            items.push(snippet(
                "interchange",
                "keyword",
                "",
                "interchange {\n  them ZZ \"${1:THEIRID}\"\n  us ZZ \"${2:OURID}\"\n  version 004010\n  usage test\n}",
                "0",
            ));
            items.push(snippet(
                "zone",
                "keyword",
                "",
                "zone ${1:America/Chicago}",
                "0",
            ));
            items.push(snippet(
                "currency",
                "keyword",
                "",
                "currency \"${1:USD}\"",
                "0",
            ));
            items.push(snippet(
                "setting",
                "keyword",
                "",
                "/// ${3:What it is for}\nsetting ${1:name}: ${2:Bool} = $0",
                "0",
            ));
        }
        Header::Guide { name, .. } => {
            if let Some(guide) = program.guide_names.get(&name.name) {
                children(program, *guide, None, false, items);
            }
            items.push(snippet(
                "loop",
                "keyword",
                "",
                "loop ${1:0300} min ${2:1} max ${3:10}",
                "1",
            ));
            items.push(snippet(
                "severity",
                "keyword",
                "",
                "severity ${1:invalid-code} ${2|warning,error,info|}",
                "1",
            ));
        }
        Header::Library { .. } | Header::Missing(_) => {}
    }
    items.push(snippet(
        "fn",
        "keyword",
        "",
        "fn ${1:name}(${2:value}: ${3:Text}) -> ${4:Text} = $0",
        "3",
    ));
    items.push(snippet(
        "crosswalk",
        "keyword",
        "",
        "crosswalk ${1:name}: ${2:Code<S502>} <-> ${3:StopType} {\n  $0\n}",
        "3",
    ));
    items.push(snippet("const", "keyword", "", "const ${1:name} = $0", "3"));
    items.push(snippet(
        "test",
        "keyword",
        "",
        "test \"${1:name}\" {\n  $0\n}",
        "4",
    ));
    items.push(snippet(
        "property",
        "keyword",
        "",
        "property \"${1:name}\" {\n  vary ${2:value} in $3\n  $0\n}",
        "4",
    ));
}

/// The segments and loops under a node, as segment blocks (outbound) or guide lines.
fn children(
    program: &Program,
    guide: usize,
    parent: Option<usize>,
    blocks: bool,
    items: &mut Vec<Item2>,
) {
    let compiled = &program.guides[guide].compiled;
    for (order, child) in nav::children(compiled, parent).iter().enumerate() {
        let reference = NodeRef {
            guide,
            node: *child,
        };
        let (detail, doc) = describe::node(program, reference);
        let sort = format!("0{order:03}");
        match &compiled.node(*child).kind {
            NodeKind::Segment(id) if matches!(id.as_str(), "ST" | "SE") => {}
            NodeKind::Segment(id) => {
                let mut entry = item(id, "segment", detail, Some(doc), &sort);
                if blocks {
                    entry.insert = format!("{id} {{ 01 <- $0 }}");
                    entry.snippet = true;
                }
                items.push(entry);
            }
            NodeKind::Loop { id, .. } => {
                let mut entry = item(id, "loop", detail, Some(doc), &sort);
                if blocks {
                    entry.insert = format!("{id} <- each ${{1:list}} as ${{2:item}} {{\n  $0\n}}");
                    entry.snippet = true;
                }
                items.push(entry);
            }
        }
    }
}

#[allow(clippy::too_many_lines)]
fn expression(session: &Session, offset: u32, items: &mut Vec<Item2>) {
    let program = &session.program;
    if let Some(scope) = session.scope(offset) {
        for (name, ty, span) in &scope.locals {
            if span.start <= offset && !name.is_empty() {
                items.push(item(name, "variable", ty.to_string(), None, "0"));
            }
        }
        if let Some(document) = scope.document.and_then(schema::document) {
            for field in document.fields {
                let (detail, doc) = describe::field(program, field);
                items.push(item(field.name, "field", detail, Some(doc), "1"));
            }
        }
        if let Some(guide) = scope.guide
            && scope.document.is_none()
        {
            children(program, guide, None, false, items);
            let compiled = &program.guides[guide].compiled;
            for child in nav::children(compiled, None) {
                if let NodeKind::Segment(id) = &compiled.node(*child).kind
                    && compiled.node(*child).max <= 1
                    && !matches!(id.as_str(), "ST" | "SE")
                    && let Some(definition) = nav::definition(compiled, *child)
                {
                    for (index, element) in definition.elements.iter().enumerate() {
                        let position = u32::try_from(index + 1).unwrap_or(0);
                        let (label, detail, doc) = describe::element_parts(id, position, element);
                        items.push(item(label, "element", detail, Some(doc), "2"));
                    }
                }
            }
        }
    }
    items.push(item(
        "partner",
        "variable",
        "the trading partner and its settings",
        None,
        "1",
    ));
    items.push(item("org", "variable", "the organisation", None, "1"));
    let mut units = vec![session.unit];
    for library in program.units[session.unit].header().uses() {
        if let Some(unit) = program.libraries.get(&library.name) {
            units.push(*unit);
        }
    }
    for unit in units {
        for (name, decl) in &program.units[unit].decls {
            let (detail, doc) = describe::decl(program, unit, name).unwrap_or_default();
            match decl {
                Decl::Fn(_) => {
                    let mut entry = item(name, "function", detail, doc, "3");
                    entry.insert = format!("{name}($0)");
                    entry.snippet = true;
                    items.push(entry);
                }
                Decl::Const(_) => items.push(item(name, "constant", detail, doc, "2")),
                Decl::Crosswalk(_) => {}
            }
        }
    }
    for function in stdlib::FUNCTIONS
        .iter()
        .filter(|function| function.receiver == Receiver::None)
    {
        let mut entry = item(
            function.name,
            "function",
            function.signature,
            Some(Doc::message(function.doc)),
            "4",
        );
        entry.insert = format!("{}($0)", function.name);
        entry.snippet = true;
        items.push(entry);
    }
    for keyword in [
        "none", "true", "false", "exists", "lookup", "match", "if", "each",
    ] {
        items.push(item(keyword, "keyword", "", None, "5"));
    }
    for enumeration in schema::ENUMS {
        for (variant, doc) in enumeration.variants {
            items.push(item(
                *variant,
                "variant",
                enumeration.name,
                Some(Doc::message(doc)),
                "6",
            ));
        }
    }
}

/// The type of the field path left of `dot` on a line that assigns to it (`custom.` in
/// `custom.po_number <- ...`), read from the record the scope builds: an assignment's target is
/// not an expression, so the checker records no type for it.
fn assigned_type(session: &Session, dot: u32) -> Option<Type> {
    let before = session.before(dot);
    let mut names = Vec::new();
    let mut expect_name = true;
    for token in before.iter().rev() {
        match token.kind {
            Tok::Name if expect_name => {
                names.push(session.text(token.span));
                expect_name = false;
            }
            Tok::Dot if !expect_name => expect_name = true,
            Tok::Newline | Tok::LBrace | Tok::Semi if !expect_name => break,
            _ => return None,
        }
    }
    if expect_name {
        return None;
    }
    let mut ty = session.scope(dot)?.builds.clone()?;
    for name in names.into_iter().rev() {
        ty = match ty.unwrapped() {
            Type::Record(record) => session
                .program
                .field_type(schema::record(record)?.field(name)?),
            Type::Struct(fields) => fields.iter().find(|(field, _)| field == name)?.1.clone(),
            _ => return None,
        };
    }
    Some(ty)
}

/// What may follow `value.`, by the value's type.
#[allow(clippy::too_many_lines)]
fn members(session: &Session, dot: u32, items: &mut Vec<Item2>) {
    let program = &session.program;
    let Some(ty) = program
        .facts
        .type_ending_at(session.unit, dot)
        .map(|typed| typed.ty.unwrapped().clone())
        .or_else(|| assigned_type(session, dot).map(|ty| ty.unwrapped().clone()))
    else {
        return;
    };
    match &ty {
        Type::Record(name) => {
            if let Some(record) = schema::record(name) {
                for field in record.fields {
                    let (detail, doc) = describe::field(program, field);
                    items.push(item(field.name, "field", detail, Some(doc), "0"));
                }
            }
        }
        Type::Struct(fields) => {
            let partner = program.partner_type();
            let custom = program
                .context
                .custom_fields
                .iter()
                .find(|(record, _)| ty == program.custom_type(record))
                .map(|(_, fields)| fields);
            for (name, field_ty) in fields {
                let doc = if ty == partner {
                    describe::setting(program, name).and_then(|(_, doc)| doc)
                } else {
                    custom
                        .and_then(|fields| fields.iter().find(|field| field.name == *name))
                        .filter(|field| !field.label.is_empty())
                        .map(|field| Doc::Text(field.label.clone()))
                };
                items.push(item(name, "field", field_ty.to_string(), doc, "0"));
            }
        }
        Type::Ref(kind) => {
            for (name, field_ty, doc) in schema::ref_fields(kind) {
                items.push(item(
                    name,
                    "field",
                    field_ty.to_string(),
                    Some(Doc::message(doc)),
                    "0",
                ));
            }
        }
        Type::Transaction(guide) => node_members(program, *guide, None, items),
        Type::Loop(reference) => {
            node_members(program, reference.guide, Some(reference.node), items);
        }
        Type::Segment(reference) => {
            let compiled = &program.guides[reference.guide].compiled;
            if let Some(definition) = nav::definition(compiled, reference.node) {
                let id = nav::segment_id(compiled, reference.node).to_owned();
                for (index, element) in definition.elements.iter().enumerate() {
                    let position = u32::try_from(index + 1).unwrap_or(0);
                    let (_, detail, doc) = describe::element_parts(&id, position, element);
                    items.push(item(
                        format!("{position:02}"),
                        "element",
                        detail,
                        Some(doc),
                        &format!("0{position:03}"),
                    ));
                }
            }
        }
        _ => {}
    }
    let receiver = match &ty {
        Type::List(item_ty)
            if matches!(item_ty.as_ref(), Type::Segment(_) | Type::SegmentId(_)) =>
        {
            vec![Receiver::List, Receiver::X12]
        }
        Type::List(_) | Type::Json => vec![Receiver::List],
        Type::Text | Type::Code(..) => vec![Receiver::Text],
        Type::Int | Type::Decimal | Type::Percent => vec![Receiver::Number],
        Type::Money => vec![Receiver::Money, Receiver::Number],
        Type::Measure(_) => vec![Receiver::Measure, Receiver::Number],
        Type::Date => vec![Receiver::Date],
        Type::DateTime => vec![Receiver::DateTime],
        Type::Loop(_) | Type::LoopId(_) | Type::Segment(_) | Type::SegmentId(_) => {
            vec![Receiver::X12]
        }
        _ => Vec::new(),
    };
    for function in stdlib::FUNCTIONS
        .iter()
        .filter(|function| receiver.contains(&function.receiver))
    {
        let mut entry = item(
            function.name,
            "method",
            function.signature,
            Some(Doc::message(function.doc)),
            "5",
        );
        entry.insert = format!("{}($0)", function.name);
        entry.snippet = true;
        items.push(entry);
    }
}

fn node_members(program: &Program, guide: usize, parent: Option<usize>, items: &mut Vec<Item2>) {
    let compiled = &program.guides[guide].compiled;
    children(program, guide, parent, false, items);
    for child in nav::children(compiled, parent) {
        if let NodeKind::Segment(id) = &compiled.node(*child).kind
            && compiled.node(*child).max <= 1
            && let Some(definition) = nav::definition(compiled, *child)
        {
            for (index, element) in definition.elements.iter().enumerate() {
                let position = u32::try_from(index + 1).unwrap_or(0);
                let (label, detail, doc) = describe::element_parts(id, position, element);
                items.push(item(label, "element", detail, Some(doc), "1"));
            }
        }
    }
}
