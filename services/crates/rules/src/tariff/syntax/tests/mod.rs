//! Tests over the whole front end: the reference's examples, the corpus and the properties.

mod corpus;
mod program;
mod properties;

use super::parse;

const REFERENCE: &str = include_str!("../../../../../../../docs/engineering/formula-language.md");

/// Every ```` ```tariff ```` block of the language reference, as whole files: a block that has
/// no header is wrapped in `tariff "t"`.
pub(crate) fn reference_blocks() -> Vec<String> {
    let mut blocks = Vec::new();
    let mut rest = REFERENCE;
    while let Some(start) = rest.find("```tariff\n") {
        let body = &rest[start + "```tariff\n".len()..];
        let Some(end) = body.find("```") else { break };
        let block = &body[..end];
        let headed = ["tariff ", "library ", "condition "]
            .iter()
            .any(|header| block.starts_with(header));
        blocks.push(if headed {
            block.to_owned()
        } else {
            format!("tariff \"t\"\n\n{block}")
        });
        rest = &body[end + 3..];
    }
    blocks
}

#[test]
fn every_reference_example_parses_without_a_diagnostic() {
    let blocks = reference_blocks();
    assert!(blocks.len() >= 7, "found {} blocks", blocks.len());
    for block in blocks {
        let file = parse(&block);
        assert!(
            file.diagnostics.is_empty(),
            "{block}\n{:#?}",
            file.diagnostics
                .iter()
                .map(|d| (d.span, d.message.english()))
                .collect::<Vec<_>>()
        );
        assert!(file.header.is_some());
        assert!(!file.items.is_empty());
    }
}

#[test]
fn loads_a_tariff_with_its_standard_and_organisation_imports() {
    use crate::tariff::context::{Context, Library};
    use crate::tariff::syntax::program::{ModuleId, Resolved, load};
    let context = Context {
        libraries: vec![Library {
            path: String::from("org.accessorials"),
            version: Some(String::from("v1")),
            source: String::from(
                "library \"accessorials\"\n\nfn hazmat(miles: Distance) -> Money = $1/mi * miles\n",
            ),
        }],
        ..Context::default()
    };
    let program = load(
        "tariff \"t\"\n\nuse std.fuel\nuse org.accessorials.{ hazmat }\n",
        Some(crate::tariff::syntax::ast::FileKind::Tariff),
        &context,
    );
    assert!(program.diagnostics.is_empty(), "{:?}", program.diagnostics);
    assert_eq!(program.modules.len(), 3);
    assert!(matches!(
        program.resolve(ModuleId::ROOT, "fuel"),
        Some(Resolved::Module(_))
    ));
    assert!(matches!(
        program.resolve(ModuleId::ROOT, "hazmat"),
        Some(Resolved::Decl { .. })
    ));
    let order: Vec<ModuleId> = program.in_dependency_order().map(|m| m.id).collect();
    assert_eq!(order.last(), Some(&ModuleId::ROOT));
}

#[test]
fn a_doc_right_above_the_first_item_is_the_items_and_one_after_a_blank_line_the_files() {
    let blocks = reference_blocks();
    let condition = blocks
        .iter()
        .find(|block| block.starts_with("condition"))
        .map(|block| parse(block))
        .expect("the reference has a condition");
    assert!(condition.docs.is_none());
    assert!(
        condition
            .items
            .first()
            .is_some_and(|item| item.docs.is_some())
    );
    let first_look = parse(&blocks[0]);
    assert!(first_look.docs.is_some());
}

#[test]
fn a_large_file_with_many_comments_parses_and_formats_in_linear_time() {
    let mut body = String::new();
    for index in 0..4_000 {
        let _ = std::fmt::Write::write_fmt(
            &mut body,
            format_args!("/// Value {index}.\nlet v{index} = {index} + 1 // note {index}\n"),
        );
    }
    let source = format!("tariff \"big\"\n\n{body}");
    let started = std::time::Instant::now();
    let file = parse(&source);
    assert!(file.diagnostics.is_empty());
    let formatted = super::format::format(&source).expect("the file formats");
    assert_eq!(formatted, source);
    assert!(
        started.elapsed().as_secs() < 5,
        "took {:?}",
        started.elapsed()
    );
}
