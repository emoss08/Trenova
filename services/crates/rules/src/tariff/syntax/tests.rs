//! Tests over the whole front end: the reference's examples, the corpus and the properties.

use super::parse;

const REFERENCE: &str = include_str!("../../../../../../docs/engineering/formula-language.md");

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
    use super::program::{ModuleId, Resolved, load};
    use crate::tariff::context::{Context, Library};
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
        Some(super::ast::FileKind::Tariff),
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
