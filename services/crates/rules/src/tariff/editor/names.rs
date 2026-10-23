//! The candidates completion offers where a value or a statement goes: the names visible there
//! with their types, the functions, the enum variants, the keywords and snippets valid there,
//! and the parameters of the call or the keys of the table being written.
//!
//! What a body can name follows the checker's rules ([`Body`]): the record and `tables` only at
//! a file's top level and in tests, inputs only at the top level, file-level bindings nowhere
//! but the top level, and only constants in a constant's value or a default.

use super::choices::{CHOSEN, CHOSEN_DOC};
use super::complete::{CompletionItem, Items, Kind, Rank, Site, Then, callee_of, fill_call, units};
use super::cursor::{Body, Bracket};
use super::describe::{symbol_signature, type_text};
use super::session::Session;
use crate::message;
use crate::message::Message;
use crate::span::Span;
use crate::tariff::check::methods::ParamKind;
use crate::tariff::check::{CONSTRUCTORS, Callee, SymbolKind};
use crate::tariff::docs::Doc;
use crate::tariff::schema::TABLES_DOC;
use crate::tariff::syntax::ast::{Expr, ExprKind, FileKind, ItemKind, StmtKind};
use crate::tariff::syntax::program::{Decl, ModuleId};
use crate::tariff::types::{EnumType, Type};
use crate::tariff::units::Dim;
use std::sync::Arc;

/// Adds a keyword.
pub fn keyword(items: &mut Items, rank: Rank, word: &str) {
    if let Some(item) = items.add(rank, Kind::Keyword, word) {
        item.doc = super::describe::keyword_doc(word).map(|doc| Doc::Message(Message::new(doc)));
    }
}

/// Adds a snippet.
fn snippet(items: &mut Items, label: &str, insert: &str, doc: &'static str) {
    if let Some(item) = items.add(Rank::Keyword, Kind::Snippet, label) {
        insert.clone_into(&mut item.insert);
        item.snippet = true;
        item.doc = Some(Doc::Message(Message::new(doc)));
    }
}

/// The enums the root file can name, by the name they are written with: the schema's, its
/// own and the imported ones.
pub fn enums(session: &Session<'_>) -> Vec<(String, Arc<EnumType>)> {
    let mut out: Vec<(String, Arc<EnumType>)> = session
        .schema
        .enums
        .iter()
        .map(|enumeration| (enumeration.name.clone(), Arc::clone(enumeration)))
        .collect();
    let root = session.root();
    let names = root
        .symbols
        .iter()
        .filter(|(_, symbol)| matches!(symbol.decl, Decl::Enum(_)))
        .map(|(name, _)| name.clone())
        .chain(root.imports.keys().cloned());
    for name in names {
        let Some(symbol) = session
            .global(ModuleId::ROOT, &name)
            .and_then(|id| session.symbol(id))
        else {
            continue;
        };
        if let (SymbolKind::Enum, Some(Type::Enum(enumeration))) = (symbol.kind, &symbol.ty) {
            out.push((name, Arc::clone(enumeration)));
        }
    }
    out
}

/// The file's inputs.
pub fn inputs(site: &Site<'_, '_>, items: &mut Items, rank: Rank) {
    let session = site.session;
    for (name, symbol) in &session.root().symbols {
        if !matches!(symbol.decl, Decl::Input(_)) {
            continue;
        }
        let Some(id) = session.declared_at(ModuleId::ROOT, symbol.span) else {
            continue;
        };
        if let Some(item) = items.add(rank, Kind::Input, name) {
            describe(site, item, id);
        }
    }
}

fn describe(site: &Site<'_, '_>, item: &mut CompletionItem, id: crate::tariff::check::SymbolId) {
    let Some(symbol) = site.session.symbol(id) else {
        return;
    };
    item.detail = symbol
        .ty
        .as_ref()
        .filter(|ty| !matches!(ty, Type::Error))
        .map(type_text);
    item.doc.clone_from(&symbol.doc);
}

/// Whether a value of type `ty` fits what is expected.
fn fits(expected: Option<&Type>, ty: Option<&Type>) -> bool {
    match (expected, ty) {
        (Some(expected), Some(ty)) => {
            !matches!(ty, Type::Error | Type::Never)
                && !matches!(expected, Type::Error)
                && expected.accepts(ty)
        }
        _ => false,
    }
}

/// Everything a value can be written with at the position.
pub fn expression(site: &Site<'_, '_>, items: &mut Items, expected: Option<&Type>) {
    let body = site.located.body;
    if body == Body::Declaration {
        return;
    }
    variants(site, items, expected);
    locals(site, items, expected);
    globals(site, items, expected);
    if matches!(body, Body::Top | Body::Test) {
        roots(site, items, expected);
    }
    if body == Body::Test {
        test_names(site, items);
    }
    if body != Body::Constant {
        for (name, insert, doc) in [
            (
                "Number",
                "Number($0)",
                message!("`Number(i)`: an `Int` as a `Number`."),
            ),
            (
                "Int.parse",
                "Int.parse($0)",
                message!("`Int.parse(text)`: the whole number a text spells, or `none`."),
            ),
        ] {
            if let Some(item) = items.add(Rank::Other, Kind::Function, name) {
                insert.clone_into(&mut item.insert);
                item.snippet = true;
                item.doc = Some(Doc::Message(Message::new(doc)));
            }
        }
    }
    value_keywords(site, items, expected);
}

fn value_keywords(site: &Site<'_, '_>, items: &mut Items, expected: Option<&Type>) {
    let boolean = expected.is_some_and(|ty| matches!(ty.unwrapped(), Type::Bool));
    let rank = if boolean {
        Rank::Expected
    } else {
        Rank::Keyword
    };
    for word in ["true", "false", "not"] {
        keyword(items, rank, word);
    }
    for word in ["if", "match"] {
        keyword(items, Rank::Keyword, word);
    }
    if expected.is_none_or(Type::is_optional) {
        let rank = if expected.is_some() {
            Rank::Expected
        } else {
            Rank::Keyword
        };
        keyword(items, rank, "none");
    }
    if site.before_is("??")
        && let Some(item) = items.add(Rank::Expected, Kind::Keyword, "fail")
    {
        item.insert = String::from("fail \"$0\"");
        item.snippet = true;
        item.doc = super::describe::keyword_doc("fail").map(|doc| Doc::Message(Message::new(doc)));
    }
}

/// The local names visible at the position.
fn locals(site: &Site<'_, '_>, items: &mut Items, expected: Option<&Type>) {
    let session = site.session;
    for local in site.scopes.visible(site.offset, site.located.body) {
        let id = session.declared_at(ModuleId::ROOT, local.span);
        let ty = id
            .and_then(|id| session.symbol(id))
            .and_then(|s| s.ty.clone());
        let rank = if fits(expected, ty.as_ref()) {
            Rank::Expected
        } else {
            Rank::Local
        };
        if let Some(item) = items.add(rank, Kind::Name, &local.name) {
            item.detail = ty
                .as_ref()
                .filter(|ty| !matches!(ty, Type::Error))
                .map(type_text);
        }
    }
}

/// The file's inputs, constants, functions, imported names and module aliases the body sees.
fn globals(site: &Site<'_, '_>, items: &mut Items, expected: Option<&Type>) {
    let session = site.session;
    let body = site.located.body;
    let root = session.root();
    let declared = root
        .symbols
        .iter()
        .map(|(name, symbol)| (name, Some(symbol.decl)));
    let imported = root.imports.keys().map(|name| (name, None));
    for (name, decl) in declared.chain(imported) {
        let Some(id) = session.global(ModuleId::ROOT, name) else {
            continue;
        };
        let Some(symbol) = session.symbol(id) else {
            continue;
        };
        let kind = match symbol.kind {
            SymbolKind::Input if body == Body::Top => Kind::Input,
            SymbolKind::Const => Kind::Name,
            SymbolKind::Function if body != Body::Constant => Kind::Function,
            _ => continue,
        };
        let _ = decl;
        let rank = if kind == Kind::Function {
            if fits(expected, symbol.ty.as_ref()) {
                Rank::Expected
            } else {
                Rank::Global
            }
        } else if fits(expected, symbol.ty.as_ref()) {
            Rank::Expected
        } else {
            Rank::Global
        };
        let module = symbol.module.unwrap_or(ModuleId::ROOT);
        if let Some(item) = items.add(rank, kind, name) {
            describe(site, item, id);
            if kind == Kind::Function {
                fill_call(site, item, module, name);
            } else if kind == Kind::Name {
                item.detail = Some(symbol_signature(session, id));
            }
        }
    }
    if body == Body::Constant {
        return;
    }
    for (alias, target) in &root.aliases {
        if let Some(item) = items.add(Rank::Root, Kind::Module, alias) {
            item.detail = session.module_path(target.module);
        }
    }
}

/// The schema roots and `tables`.
fn roots(site: &Site<'_, '_>, items: &mut Items, expected: Option<&Type>) {
    for (root, ty, doc) in &site.session.schema.roots {
        let rank = if fits(expected, Some(ty)) {
            Rank::Expected
        } else {
            Rank::Root
        };
        if let Some(item) = items.add(rank, Kind::Name, root) {
            item.detail = Some(type_text(ty));
            item.doc = Some(doc.clone());
        }
    }
    if let Some(item) = items.add(Rank::Root, Kind::Name, "tables") {
        item.doc = Some(Doc::message(TABLES_DOC));
    }
}

/// What a test reads besides the record: the bill and the record constructors.
fn test_names(site: &Site<'_, '_>, items: &mut Items) {
    if site.kind() == FileKind::Tariff {
        if let Some(item) = items.add(Rank::Global, Kind::Function, "charge") {
            item.insert = String::from("charge($0)");
            item.snippet = true;
            item.detail = Some(String::from("charge(line: CODE | Text) -> Money?"));
            item.doc = Some(Doc::Message(Message::new(message!(
                "The amount the bill's lines with this code or label add up to, or `none` when there are none."
            ))));
        }
        if let Some(item) = items.add(Rank::Global, Kind::Function, CHOSEN) {
            item.insert = format!("{CHOSEN}($0)");
            item.snippet = true;
            item.detail = Some(format!("{CHOSEN}(choice: Text) -> Text?"));
            item.doc = Some(Doc::Message(Message::new(CHOSEN_DOC)));
        }
        for (word, detail) in [("total", "Money"), ("lines", "List<Line>")] {
            if let Some(item) = items.add(Rank::Global, Kind::Name, word) {
                item.detail = Some(detail.to_owned());
                item.doc =
                    super::describe::keyword_doc(word).map(|doc| Doc::Message(Message::new(doc)));
            }
        }
    }
    for (constructor, record) in CONSTRUCTORS {
        if let Some(item) = items.add(Rank::Global, Kind::Function, constructor) {
            item.insert = format!("{constructor}($0)");
            item.snippet = true;
            item.detail = Some(format!("{constructor}(…) -> {record}"));
            item.doc = Some(Doc::Message(Message::with(
                message!(
                    "Builds a `{0}` for a test: every field is named and optional, and the sample record's values fill the rest."
                ),
                [crate::message::Arg::from(*record)],
            )));
        }
    }
}

/// The enum variants: the expected enum's first and bare, the others bare when only one enum
/// has them and qualified (`StopStatus.New`) when several do.
fn variants(site: &Site<'_, '_>, items: &mut Items, expected: Option<&Type>) {
    let visible = enums(site.session);
    let wanted = match expected.map(Type::unwrapped) {
        Some(Type::Enum(enumeration)) => Some(Arc::clone(enumeration)),
        _ => None,
    };
    if let Some(wanted) = &wanted {
        for (variant, doc) in &wanted.variants {
            if let Some(item) = items.add(Rank::Expected, Kind::Variant, variant) {
                item.detail = Some(type_text(&Type::Enum(Arc::clone(wanted))));
                item.doc.clone_from(doc);
            }
        }
        return;
    }
    if expected.is_some_and(|ty| !matches!(ty.unwrapped(), Type::Bool | Type::Error | Type::Never))
    {
        return;
    }
    for (written, enumeration) in &visible {
        if wanted
            .as_ref()
            .is_some_and(|wanted| wanted.name == enumeration.name)
        {
            continue;
        }
        for (variant, doc) in &enumeration.variants {
            let shared = visible
                .iter()
                .filter(|(_, other)| other.has(variant))
                .count()
                > 1;
            let label = if shared {
                format!("{written}.{variant}")
            } else {
                variant.clone()
            };
            if let Some(item) = items.add(Rank::Variant, Kind::Variant, &label) {
                item.detail = Some(written.clone());
                item.doc.clone_from(doc);
            }
        }
    }
}

/// The keywords and snippets that may start a statement at the position. Returns whether an
/// expression may start one there too (not at a library's top level, nor in an empty file).
pub fn statement(site: &Site<'_, '_>, items: &mut Items) -> bool {
    let located = &site.located;
    let kind = site.kind();
    if site.session.file().header.is_none() && site.before.is_none() {
        for (header, insert) in [
            ("tariff", "tariff \"${1:Name}\"\n\n$0"),
            ("library", "library \"${1:name}\"\n\n$0"),
            ("condition", "condition \"${1:Name}\"\n\n$0"),
        ] {
            if let Some(item) = items.add(Rank::Keyword, Kind::Keyword, header) {
                insert.clone_into(&mut item.insert);
                item.snippet = true;
                item.doc =
                    super::describe::keyword_doc(header).map(|doc| Doc::Message(Message::new(doc)));
            }
        }
        return false;
    }
    if in_choice(located) {
        keyword(items, Rank::Keyword, "option");
        snippet(
            items,
            "option \"…\" { … }",
            "option \"${1:Label}\" {\n\t$0\n}",
            message!("One method of the choice, with the charges it bills."),
        );
        return false;
    }
    let file_level = located.at_file_level()
        && located
            .item()
            .is_none_or(|item| matches!(item.kind, ItemKind::Stmt(_)));
    let property = located
        .item()
        .is_some_and(|item| matches!(item.kind, ItemKind::Property(_)));
    let words: &[&str] = match (file_level, located.body, kind) {
        (true, _, FileKind::Tariff) => &[
            "use", "input", "let", "var", "const", "fn", "type", "enum", "charge", "note", "if",
            "match", "for", "require", "fail", "minimum", "maximum", "round", "test", "property",
            "best", "worst",
        ],
        (true, _, FileKind::Library) => &[
            "use", "const", "fn", "type", "enum", "private", "test", "property",
        ],
        (true, _, FileKind::Condition) => &[
            "use", "input", "let", "var", "const", "fn", "type", "enum", "if", "match", "for",
            "require", "fail",
        ],
        (false, Body::Function, _) => &[
            "let", "var", "if", "match", "for", "require", "fail", "return",
        ],
        (false, Body::Test, _) if property => &[
            "vary", "assume", "given", "expect", "let", "var", "if", "match", "for",
        ],
        (false, Body::Test, _) => &[
            "given", "expect", "let", "var", "if", "match", "for", "require", "fail",
        ],
        (false, Body::Top, FileKind::Tariff) => &[
            "let", "var", "charge", "note", "if", "match", "for", "require", "fail", "best",
            "worst",
        ],
        (false, Body::Top, _) => &["let", "var", "if", "match", "for", "require", "fail"],
        _ => &[],
    };
    for word in words {
        keyword(items, Rank::Keyword, word);
    }
    if located.loops() > 0 && !file_level {
        keyword(items, Rank::Keyword, "break");
        keyword(items, Rank::Keyword, "continue");
    }
    snippets(items, file_level, words);
    !(file_level && kind == FileKind::Library)
}

/// Whether the position is directly inside a `best` or `worst`'s braces, between its options.
fn in_choice(located: &super::cursor::Located<'_>) -> bool {
    let Some(at) = located.chain.iter().rposition(|node| {
        matches!(node, super::cursor::Node::Stmt(stmt) if matches!(stmt.kind, StmtKind::Choose(_)))
    }) else {
        return false;
    };
    located
        .chain
        .get(at + 1..)
        .unwrap_or_default()
        .iter()
        .all(|node| !matches!(node, super::cursor::Node::Block(_)))
}

/// The snippets of choices and properties.
fn construct_snippets(items: &mut Items, file_level: bool, words: &[&str]) {
    let has = |word: &str| words.contains(&word);
    if has("best") {
        snippet(
            items,
            "best with two options",
            "best \"${1:Label}\" {\n\toption \"${2:Per mile}\" {\n\t\t$0\n\t}\n\toption \"${3:Flat}\" {\n\t\t\n\t}\n}",
            message!("Bills the cheaper of two methods; add an `option` for each further one."),
        );
    }
    if has("property") && file_level {
        snippet(
            items,
            "property \"…\" { … }",
            "property \"${1:name}\" {\n\tvary ${2:shipment.weight} in ${3:1 lb..40_000 lb}\n\texpect $0\n}",
            message!(
                "A property: vary fields or inputs over ranges with `vary`, and say what holds in every case with `expect`."
            ),
        );
    }
    if has("vary") {
        snippet(
            items,
            "vary … in …",
            "vary ${1:shipment.weight} in ${2:1 lb..40_000 lb}",
            message!("Makes a field, an input or a new name take every value of a range or list."),
        );
    }
}

fn snippets(items: &mut Items, file_level: bool, words: &[&str]) {
    let has = |word: &str| words.contains(&word);
    construct_snippets(items, file_level, words);
    if has("charge") {
        snippet(
            items,
            "charge (quantity, rate)",
            "charge \"${1:Label}\" (quantity: ${2:quantity}, rate: ${3:rate})",
            message!(
                "A charge billed as a rate times a quantity, both printed on the invoice line."
            ),
        );
        snippet(
            items,
            "charge as CODE",
            "charge \"${1:Label}\" as ${2:CODE} = ${3:amount}",
            message!("An accessorial charge billed under a catalog code."),
        );
    }
    if has("for") {
        snippet(
            items,
            "for … in …",
            "for ${1:item} in ${2:list} {\n\t$0\n}",
            message!("Walks a list in order."),
        );
    }
    if has("match") {
        snippet(
            items,
            "match",
            "match ${1:value} {\n\t${2:_} => $0\n}",
            message!("Picks the first arm whose pattern matches."),
        );
    }
    if has("if") {
        snippet(
            items,
            "if let",
            "if let ${1:name} = ${2:optional} {\n\t$0\n}",
            message!("Runs a block with an optional's value, when it has one."),
        );
    }
    if has("require") {
        snippet(
            items,
            "require let … else",
            "require let ${1:name} = ${2:optional} else \"${3:message}\"",
            message!("Refuses to rate without the optional's value, and binds it from there on."),
        );
    }
    if has("test") && file_level {
        snippet(
            items,
            "test",
            "test \"${1:name}\" {\n\tgiven ${2:shipment.weight} = ${3:1_000 lb}\n\texpect $0\n}",
            message!(
                "A test: set fields of the sample record with `given`, check the bill with `expect`."
            ),
        );
    }
    if has("fn") && file_level {
        snippet(
            items,
            "fn",
            "fn ${1:name}(${2:value}: ${3:Money}) -> ${4:Money} {\n\t$0\n}",
            message!("A function: annotated parameters and return type."),
        );
    }
    if has("input") && file_level {
        snippet(
            items,
            "input",
            "/// ${1:What it is}\ninput ${2:name}: ${3:Money} = ${0:\\$0}",
            message!("An input the contract sets, with its label."),
        );
    }
}

/// Inside `tables.Code[`: the axis's keys from the cells the context carries.
pub fn table_keys(site: &Site<'_, '_>, items: &mut Items, open: u32, part: usize) -> bool {
    let Some(index) = site.scopes.find_last(|expr| {
        matches!(&expr.kind, ExprKind::Index { target, .. }
            if target.span.end <= open && open <= target.span.end.saturating_add(4) && expr.span.end > open)
    }) else {
        return false;
    };
    let ExprKind::Index { target, .. } = &index.kind else {
        return false;
    };
    let ExprKind::Field {
        target: root, name, ..
    } = &target.kind
    else {
        return false;
    };
    if !matches!(&root.kind, ExprKind::Name(root) if root.text == "tables") {
        return false;
    }
    let code = name.text.clone();
    let Some(table) = site
        .session
        .context
        .tables
        .iter()
        .find(|table| table.code == code)
    else {
        return false;
    };
    let Some(axis) = table.axes.get(part) else {
        return true;
    };
    let label = if axis.label.is_empty() {
        axis.kind.clone()
    } else {
        axis.label.clone()
    };
    if axis.match_mode != "Exact" {
        return false;
    }
    let mut keys: Vec<&str> = table
        .cells
        .iter()
        .flatten()
        .filter_map(|cell| cell.at.get(part)?.key.as_deref())
        .collect();
    keys.sort_unstable();
    keys.dedup();
    site.session.tick(keys.len());
    let enumerated = axis.kind == "FreightClass";
    for key in keys {
        let text = if enumerated {
            key.to_owned()
        } else {
            format!("\"{}\"", key.replace('\\', "\\\\").replace('"', "\\\""))
        };
        if let Some(item) = items.add(Rank::Expected, Kind::Value, &text) {
            item.detail = Some(format!("{code}: {label}"));
        }
    }
    false
}

/// What a call's argument list offers: the parameters not given yet as `name:`, units for
/// `.in(`, codes and labels for `charge(`, and what follows them.
pub fn arguments(
    site: &Site<'_, '_>,
    items: &mut Items,
    call: &Expr,
    bracket: &Bracket,
    at_part_start: bool,
) -> Then {
    let Some(callee) = callee_of(site.session, call) else {
        return Then::Values(None);
    };
    let params = parameters(site.session, call, &callee);
    match &callee {
        Callee::ChargeLookup => {
            lookups(site, items);
            return Then::Done;
        }
        Callee::Method(method) if method.params.iter().any(|p| p.kind == ParamKind::Unit) => {
            let dim = receiver_dim(site, call);
            units(items, dim, false);
            return Then::Done;
        }
        _ => {}
    }
    let ExprKind::Call { args, .. } = &call.kind else {
        return Then::Values(None);
    };
    let positional = bracket.part.min(args.len())
        - args
            .iter()
            .take(bracket.part)
            .filter(|arg| arg.name.is_some())
            .count();
    let given: Vec<&str> = args
        .iter()
        .filter_map(|arg| arg.name.as_ref().map(|name| name.text.as_str()))
        .collect();
    if at_part_start && bracket.current_name().is_none() {
        for (index, param) in params.iter().enumerate() {
            if index < positional || given.contains(&param.name.as_str()) {
                continue;
            }
            if let Some(item) = items.add(Rank::Parameter, Kind::Field, &format!("{}:", param.name))
            {
                item.insert = format!("{}: ", param.name);
                item.detail = param.ty.as_ref().map(type_text);
                item.doc.clone_from(&param.doc);
            }
        }
    }
    let active = match bracket.current_name() {
        Some(name) => params.iter().find(|param| param.name == name),
        None => params
            .iter()
            .enumerate()
            .filter(|(index, param)| *index >= positional || !given.contains(&param.name.as_str()))
            .map(|(_, param)| param)
            .find(|param| !given.contains(&param.name.as_str())),
    };
    Then::Values(active.and_then(|param| param.ty.clone()))
}

/// One parameter of a callee, as completion and signature help read it.
#[derive(Debug, Clone)]
pub struct Parameter {
    /// Its name.
    pub name: String,
    /// Its type, when known.
    pub ty: Option<Type>,
    /// Its doc.
    pub doc: Option<Doc>,
}

/// The parameters of what a call calls.
pub fn parameters(session: &Session<'_>, call: &Expr, callee: &Callee) -> Vec<Parameter> {
    match callee {
        Callee::Function { module, name } => {
            let Some(decl) = super::describe::function_decl(session, *module, name) else {
                return Vec::new();
            };
            let doc = super::describe::function_doc(session, *module, name);
            decl.params
                .iter()
                .map(|param| Parameter {
                    name: param.name.text.clone(),
                    ty: session
                        .declared_at(*module, param.name.span)
                        .and_then(|id| session.symbol(id))
                        .and_then(|symbol| symbol.ty.clone()),
                    doc: doc.as_ref().and_then(|doc| doc.param(&param.name.text)),
                })
                .collect()
        }
        Callee::Method(method) => {
            let receiver = receiver_type(session, call);
            method
                .params
                .iter()
                .filter(|param| {
                    !matches!(
                        param.kind,
                        ParamKind::Predicate
                            | ParamKind::Map
                            | ParamKind::Summable
                            | ParamKind::Ordered
                            | ParamKind::Key
                            | ParamKind::Unit
                    )
                })
                .map(|param| Parameter {
                    name: param.name.to_owned(),
                    ty: match param.kind {
                        ParamKind::Int => Some(Type::Int),
                        ParamKind::Text => Some(Type::Text),
                        _ => receiver.clone(),
                    },
                    doc: None,
                })
                .collect()
        }
        Callee::Constructor(constructor) => {
            let record = CONSTRUCTORS
                .iter()
                .find(|(name, _)| name == constructor)
                .and_then(|(_, record)| session.schema.record(record));
            record
                .map(|record| {
                    record
                        .fields
                        .iter()
                        .map(|field| Parameter {
                            name: field.name.clone(),
                            ty: Some(field.ty.clone()),
                            doc: field.doc.clone(),
                        })
                        .collect()
                })
                .unwrap_or_default()
        }
        Callee::Conversion(_) | Callee::ChargeLookup | Callee::Table { .. } => Vec::new(),
    }
}

/// The type of a method call's receiver.
pub fn receiver_type(session: &Session<'_>, call: &Expr) -> Option<Type> {
    let ExprKind::Call { callee, .. } = &call.kind else {
        return None;
    };
    let ExprKind::Field { target, .. } = &callee.kind else {
        return None;
    };
    session
        .analysis
        .type_of(ModuleId::ROOT, target.id)
        .map(|ty| ty.unwrapped().clone())
}

fn receiver_dim(site: &Site<'_, '_>, call: &Expr) -> Option<Dim> {
    match receiver_type(site.session, call)? {
        Type::Quantity(quantity) => Some(quantity.dim),
        _ => None,
    }
}

/// In a test's `chosen(`: the labels of the file's choices.
pub fn choice_labels(site: &Site<'_, '_>, items: &mut Items) {
    for choose in super::choices::choices(site.session.file()) {
        let text = format!(
            "\"{}\"",
            choose.label.text.replace('\\', "\\\\").replace('"', "\\\"")
        );
        if let Some(item) = items.add(Rank::Expected, Kind::Value, &text) {
            item.detail = Some(format!(
                "{} \"{}\"",
                choose.kind.keyword(),
                choose.label.text
            ));
        }
    }
}

/// In a test's `charge(`: the codes and literal labels the tariff bills.
fn lookups(site: &Site<'_, '_>, items: &mut Items) {
    let file = site.session.file();
    let mut billed: Vec<(Option<String>, Option<String>, Span)> = Vec::new();
    for item in &file.items {
        if let ItemKind::Stmt(_) = &item.kind {
            super::cursor::item_statements(item, &mut |stmt| {
                if let StmtKind::Charge(charge) = &stmt.kind {
                    billed.push((
                        charge.code.as_ref().map(|code| code.text.clone()),
                        charge.label.literal(),
                        stmt.span,
                    ));
                }
            });
        }
    }
    let descriptions = site.session.context.codes.as_deref().unwrap_or_default();
    for (code, label, _) in billed {
        if let Some(code) = code
            && let Some(item) = items.add(Rank::Expected, Kind::Code, &code)
        {
            item.detail = descriptions
                .iter()
                .find(|known| known.code == code)
                .map(|known| known.description.clone())
                .filter(|description| !description.is_empty());
        }
        if let Some(label) = label {
            let text = format!("\"{}\"", label.replace('\\', "\\\\").replace('"', "\\\""));
            if let Some(item) = items.add(Rank::Expected, Kind::Value, &text) {
                item.detail = Some(String::from("Text"));
            }
        }
    }
}
