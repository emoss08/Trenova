//! What the editor says about guide nodes, schema fields, functions and declarations: one
//! place for the words completion and hover share.

use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

use super::session::Doc;
use crate::lading::check::x12 as nav;
use crate::lading::check::{Decl, FnSig, Program};
use crate::lading::schema;
use crate::lading::syntax::ast::{GuideDecl, Header, Item};
use crate::lading::types::{NodeRef, Type};
use crate::x12::guide::compiled::NodeKind;
use crate::x12::guide::model::{ElementDef, ElementType, Usage};

/// A usage in words.
#[must_use]
pub fn usage(usage: Usage) -> Message {
    Message::new(match usage {
        Usage::Mandatory => message!("Mandatory"),
        Usage::Optional => message!("Optional"),
        Usage::Conditional => message!("Conditional"),
        Usage::NotUsed => message!("Not used"),
    })
}

/// An element's type as X12 writes it: `AN 2/80`, `N2 1/12`.
#[must_use]
pub fn element_kind(element: &ElementDef) -> String {
    let kind = match element.kind {
        ElementType::Alphanumeric => "AN".to_owned(),
        ElementType::Identifier => "ID".to_owned(),
        ElementType::Date => "DT".to_owned(),
        ElementType::Time => "TM".to_owned(),
        ElementType::Numeric(places) => format!("N{places}"),
        ElementType::Decimal => "R".to_owned(),
        ElementType::Binary => "B".to_owned(),
        ElementType::Composite => return "composite".to_owned(),
    };
    format!("{kind} {}/{}", element.min, element.max)
}

/// The doc comment and note a partner guide gives a segment path (`0300.0310.N1`).
fn guide_note(program: &Program, guide: usize, path: &str) -> Option<String> {
    let unit = program.guides[guide].unit?;
    program.units[unit]
        .items()
        .iter()
        .find_map(|item| match item {
            Item::Guide(GuideDecl::Segment {
                doc,
                path: segment_path,
                note,
                ..
            }) => {
                let written: Vec<&str> = segment_path
                    .iter()
                    .map(|ident| ident.name.as_str())
                    .collect();
                (written.join(".") == path).then(|| {
                    [doc.clone(), note.as_ref().map(|note| note.value.clone())]
                        .into_iter()
                        .flatten()
                        .collect::<Vec<_>>()
                        .join("\n\n")
                })
            }
            _ => None,
        })
}

/// The loop path of a node: `0300.0310.N1`.
#[must_use]
pub fn node_path(program: &Program, reference: NodeRef) -> String {
    let compiled = &program.guides[reference.guide].compiled;
    let mut parents = vec![None; compiled.node_count()];
    for index in 0..compiled.node_count() {
        for child in &compiled.node(index).children {
            parents[*child] = Some(index);
        }
    }
    let mut parts = vec![match &compiled.node(reference.node).kind {
        NodeKind::Segment(id) | NodeKind::Loop { id, .. } => id.clone(),
    }];
    let mut cursor = parents[reference.node];
    while let Some(node) = cursor {
        if let NodeKind::Loop { id, .. } = &compiled.node(node).kind {
            parts.push(id.clone());
        }
        cursor = parents[node];
    }
    parts.reverse();
    parts.join(".")
}

/// A segment or loop: its detail line and doc.
#[must_use]
pub fn node(program: &Program, reference: NodeRef) -> (String, Doc) {
    let compiled = &program.guides[reference.guide].compiled;
    let node = compiled.node(reference.node);
    let path = node_path(program, reference);
    let (detail, name) = match &node.kind {
        NodeKind::Segment(id) => {
            let name = nav::definition(compiled, reference.node)
                .map(|definition| definition.name.clone())
                .unwrap_or_default();
            (format!("{id} · {name}"), name)
        }
        NodeKind::Loop { id, trigger } => {
            let name = loop_name(program, reference).unwrap_or_else(|| trigger.clone());
            (format!("loop {id} · {name}"), name)
        }
    };
    let mut message = Message::with(
        message!("{0}: {1}, at most {2}"),
        [
            Arg::from(name),
            Arg::from(usage(node.usage)),
            Arg::from(node.max as usize),
        ],
    );
    if let Some(note) = guide_note(program, reference.guide, &path) {
        message = Message::with(
            message!("{0}\n\n{1}"),
            [Arg::from(message), Arg::from(note)],
        );
    }
    (detail, Doc::Message(message))
}

fn loop_named(nodes: &[crate::x12::guide::model::Node], id: &str) -> Option<String> {
    for node in nodes {
        if let crate::x12::guide::model::Node::Loop(found) = node {
            if found.id == id {
                return Some(found.name.clone());
            }
            if let Some(name) = loop_named(&found.children, id) {
                return Some(name);
            }
        }
    }
    None
}

fn loop_name(program: &Program, reference: NodeRef) -> Option<String> {
    let compiled = &program.guides[reference.guide].compiled;
    let id = nav::loop_id(compiled, reference.node)?;
    loop_named(&compiled.guide().structure, id).filter(|name| !name.is_empty())
}

/// An element of a segment use: its label (`N104`), detail and doc.
#[must_use]
pub fn element(
    program: &Program,
    reference: NodeRef,
    position: u32,
) -> Option<(String, String, Doc)> {
    let compiled = &program.guides[reference.guide].compiled;
    let segment = nav::segment_id(compiled, reference.node).to_owned();
    let definition = nav::definition(compiled, reference.node)?;
    let element = definition.elements.get(position.checked_sub(1)? as usize)?;
    Some(element_parts(&segment, position, element))
}

/// An element by its definition.
#[must_use]
pub fn element_parts(segment: &str, position: u32, element: &ElementDef) -> (String, String, Doc) {
    let label = format!("{segment}{position:02}");
    let detail = format!("{} · {}", element.name, element_kind(element));
    let codes = element.codes.as_ref().map(|codes| {
        codes
            .iter()
            .take(30)
            .map(|code| {
                if code.description.is_empty() {
                    code.code.clone()
                } else {
                    format!("{} {}", code.code, code.description)
                }
            })
            .collect::<Vec<_>>()
            .join(", ")
    });
    let mut message = Message::with(
        message!("{0} ({1}), element {2}"),
        [
            Arg::from(element.name.as_str()),
            Arg::from(usage(element.usage)),
            Arg::from(element.reference.as_str()),
        ],
    );
    if let Some(codes) = codes.filter(|codes| !codes.is_empty()) {
        message = Message::with(
            message!("{0}. Codes: {1}"),
            [Arg::from(message), Arg::from(codes)],
        );
    }
    if !element.notes.is_empty() {
        message = Message::with(
            message!("{0}\n\n{1}"),
            [Arg::from(message), Arg::from(element.notes.as_str())],
        );
    }
    (label, detail, Doc::Message(message))
}

/// A schema field's detail and doc.
#[must_use]
pub fn field(program: &Program, field: &schema::Field) -> (String, Doc) {
    let ty = program.read_type(field);
    (format!("{}: {ty}", field.name), Doc::message(field.doc))
}

/// A function's signature as the editor shows it.
#[must_use]
pub fn signature(name: &str, signature: &FnSig) -> String {
    let params: Vec<String> = signature
        .params
        .iter()
        .map(|(param, ty, default)| {
            if *default {
                format!("{param}: {ty} = …")
            } else {
                format!("{param}: {ty}")
            }
        })
        .collect();
    format!("fn {name}({}) -> {}", params.join(", "), signature.ret)
}

/// A declaration's detail and doc comment.
#[must_use]
pub fn decl(program: &Program, unit: usize, name: &str) -> Option<(String, Option<Doc>)> {
    let decl = program.units[unit].decls.get(name)?;
    let item_index = match decl {
        Decl::Fn(index) | Decl::Crosswalk(index) | Decl::Const(index) => *index,
    };
    let item = program.units[unit].items().get(item_index)?;
    let doc = match item {
        Item::Fn(item) => item.doc.clone(),
        Item::Crosswalk(item) => item.doc.clone(),
        Item::Const(item) => item.doc.clone(),
        _ => None,
    }
    .map(Doc::Text);
    let detail = match decl {
        Decl::Fn(_) => program
            .fns
            .get(&(unit, name.to_owned()))
            .map(|sig| signature(name, sig))?,
        Decl::Crosswalk(_) => {
            let (left, right) = program.crosswalks.get(&(unit, name.to_owned()))?;
            format!("crosswalk {name}: {left} <-> {right}")
        }
        Decl::Const(_) => {
            let ty = program
                .facts
                .types
                .iter()
                .find(|typed| {
                    typed.unit == unit
                        && Some(typed.span) == const_value_span(program, unit, item_index)
                })
                .map_or(Type::Any, |typed| typed.ty.clone());
            format!("const {name}: {ty}")
        }
    };
    Some((detail, doc))
}

fn const_value_span(
    program: &Program,
    unit: usize,
    index: usize,
) -> Option<trenova_module_kit::span::Span> {
    match program.units[unit].items().get(index)? {
        Item::Const(item) => Some(item.value.span),
        _ => None,
    }
}

/// A partner setting's detail and doc comment.
#[must_use]
pub fn setting(program: &Program, name: &str) -> Option<(String, Option<Doc>)> {
    let partner = program.partner.as_ref()?;
    let setting = partner
        .settings
        .iter()
        .find(|setting| setting.name == name)?;
    let doc = match program.units[setting.unit].items().get(setting.item) {
        Some(Item::Setting(decl)) => decl.doc.clone().map(Doc::Text),
        _ => None,
    };
    Some((format!("setting {name}: {}", setting.ty), doc))
}

/// The header a file's name stands for: its kind and a one-line description.
#[must_use]
pub fn file(program: &Program, name: &str) -> Option<String> {
    let unit = program.unit_named(name)?;
    Some(match program.units[unit].header() {
        Header::Map { from, to, .. } => format!(
            "map {name}: {} → {}",
            endpoint(program, unit, from),
            endpoint(program, unit, to)
        ),
        Header::Rules { map, .. } => format!("rules {name} for {}", map.name),
        Header::Flow { partner, .. } => format!("flow {name} with {}", partner.name),
        Header::Library { .. } => format!("library {name}"),
        Header::Guide { base, .. } => format!("guide {name} from {}", base.name),
        Header::Partner { label, .. } => format!(
            "partner {name} {}",
            label.as_ref().map_or("", |label| label.value.as_str())
        ),
        Header::Missing(_) => return None,
    })
}

fn endpoint(
    program: &Program,
    unit: usize,
    endpoint: &crate::lading::syntax::ast::Endpoint,
) -> String {
    use crate::lading::syntax::ast::Endpoint;
    match endpoint {
        Endpoint::Guide(name) | Endpoint::Document(name) => name.name.clone(),
        Endpoint::Csv { .. } => "csv".to_owned(),
        Endpoint::Json(_) => "json".to_owned(),
        Endpoint::Missing(_) => {
            let _ = (program, unit);
            String::new()
        }
    }
}
