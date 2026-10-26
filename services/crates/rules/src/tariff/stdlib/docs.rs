//! The standard library's doc comments as message templates.
//!
//! Every `///` block in a `std.*` module is product text, translated like any `message!`: its
//! template is the block with its whitespace collapsed ([`collapse`]), and each function
//! parameter's doc ([`param_text`] of its function's) is a template too. `task i18n` collects the
//! same templates from the `.tariff` sources (`i18n/tools/extract-tariff.mjs`).
//!
//! [`trenova_module_kit::message::Message`] carries a `&'static str`, while a collapsed doc is computed. The
//! templates are therefore built once per process into a table that lives for the program, and
//! [`template`] hands out `&'static str`s into it: no leak per call, and no template that the
//! standard library sources do not hold.

use crate::tariff::docs::{collapse, param_text};
use crate::tariff::stdlib::MODULES;
use crate::tariff::syntax::ast::ItemKind;
use crate::tariff::syntax::parse;
use crate::tariff::syntax::token::TokenKind;
use std::collections::BTreeMap;
use std::sync::OnceLock;

/// Where a template is written: the module and the 1-based line its doc comment starts on.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Site {
    /// The module path, `std.ltl`.
    pub module: &'static str,
    /// The line of the doc comment's first `///`.
    pub line: usize,
}

impl Site {
    /// The source file, relative to the repository: `services/crates/rules/src/tariff/stdlib/ltl.tariff`.
    #[must_use]
    pub fn file(&self) -> String {
        let name = self.module.strip_prefix("std.").unwrap_or(self.module);
        format!("services/crates/rules/src/tariff/stdlib/{name}.tariff")
    }
}

fn table() -> &'static BTreeMap<String, Site> {
    static TABLE: OnceLock<BTreeMap<String, Site>> = OnceLock::new();
    TABLE.get_or_init(build)
}

fn build() -> BTreeMap<String, Site> {
    let mut out = BTreeMap::new();
    for (module, source) in MODULES {
        add_module(&mut out, module, source);
    }
    out
}

fn add_module(out: &mut BTreeMap<String, Site>, module: &'static str, source: &str) {
    let file = parse(source);
    let line = |offset: u32| {
        let end = usize::try_from(offset).unwrap_or(usize::MAX);
        source
            .get(..end)
            .map_or(0, |before| before.matches('\n').count())
            + 1
    };
    let mut block: Vec<&str> = Vec::new();
    let mut start = 0;
    let mut newlines = 0;
    for token in &file.tokens {
        match token.kind {
            TokenKind::DocComment => {
                if newlines > 1 {
                    flush(out, module, line(start), &mut block);
                }
                if block.is_empty() {
                    start = token.span.start;
                }
                let raw = source
                    .get(token.span.start as usize..token.span.end as usize)
                    .unwrap_or_default();
                block.push(raw.strip_prefix("///").unwrap_or(raw));
                newlines = 0;
            }
            TokenKind::Newline => newlines += 1,
            _ => {
                flush(out, module, line(start), &mut block);
                newlines = 0;
            }
        }
    }
    flush(out, module, line(start), &mut block);
    for item in &file.items {
        let (ItemKind::Fn(decl), Some(docs)) = (&item.kind, &item.docs) else {
            continue;
        };
        let site = Site {
            module,
            line: line(docs.span.start),
        };
        let doc = collapse(&docs.text);
        for param in &decl.params {
            if let Some(text) = param_text(&doc, &param.name.text) {
                out.entry(text).or_insert(site);
            }
        }
    }
}

fn flush(
    out: &mut BTreeMap<String, Site>,
    module: &'static str,
    line: usize,
    block: &mut Vec<&str>,
) {
    if block.is_empty() {
        return;
    }
    let text = collapse(&block.join(" "));
    block.clear();
    if !text.is_empty() {
        out.entry(text).or_insert(Site { module, line });
    }
}

/// The template a standard library doc reads as, given its collapsed text; `None` when no
/// standard library doc reads so.
#[must_use]
pub fn template(collapsed: &str) -> Option<&'static str> {
    table()
        .get_key_value(collapsed)
        .map(|(template, _)| template.as_str())
}

/// Every standard library doc template, with where it is written, in template order.
pub fn templates() -> impl Iterator<Item = (&'static str, Site)> {
    table()
        .iter()
        .map(|(template, site)| (template.as_str(), *site))
}

#[cfg(test)]
mod tests {
    use super::{Site, template, templates};
    use crate::tariff::stdlib::MODULES;
    use serde_json::Value as Json;

    const CATALOG: &str = include_str!("../../../../../../i18n/messages.en.json");

    /// What the hosts or the catalog tooling would misread in a template, if anything.
    fn icu_problem(template: &str) -> Option<&'static str> {
        if template.contains(['{', '}']) {
            return Some("a brace, which ICU reads as a placeholder");
        }
        if template.contains("''") {
            return Some("a doubled apostrophe, which ICU reads as one apostrophe");
        }
        if template.contains('#') {
            return Some("a `#`, which ICU reads as a count inside a plural");
        }
        if tag_like(template) {
            return Some("an angle-bracketed word, which rich messages read as a tag");
        }
        if template != template.split_whitespace().collect::<Vec<_>>().join(" ") {
            return Some("whitespace the catalogs would collapse");
        }
        if !template.chars().any(char::is_alphabetic) || template.chars().count() < 2 {
            return Some("no words, which the catalog filter drops");
        }
        if template.starts_with('/') || template.starts_with("http") {
            return Some("a start the catalog filter reads as a path or URL");
        }
        None
    }

    fn tag_like(template: &str) -> bool {
        template.match_indices('<').any(|(at, _)| {
            let rest = template.get(at + 1..).unwrap_or_default();
            let rest = rest.strip_prefix('/').unwrap_or(rest);
            let name: String = rest
                .chars()
                .take_while(|c| c.is_alphanumeric() || *c == '_' || *c == '-')
                .collect();
            name.chars().next().is_some_and(char::is_alphabetic)
                && rest
                    .get(name.len()..)
                    .is_some_and(|after| after.trim_start().starts_with(['>', '/']))
        })
    }

    fn at(site: Site) -> String {
        format!("{}:{}", site.file(), site.line)
    }

    #[test]
    fn every_module_has_docs_in_the_table() {
        for (module, _) in MODULES {
            assert!(
                templates().any(|(_, site)| site.module == *module),
                "{module} has no doc comments in the table"
            );
        }
        assert_eq!(
            template("One weight break of an LTL rate."),
            Some("One weight break of an LTL rate.")
        );
        assert_eq!(
            template(
                "Less-than-truckload pricing: weight breaks, deficit weight rating, billed weight and the NMFC density scale."
            ),
            Some(
                "Less-than-truckload pricing: weight breaks, deficit weight rating, billed weight and the NMFC density scale."
            )
        );
        assert_eq!(
            template("Billed at `rate`."),
            None,
            "only standard library docs are templates"
        );
    }

    #[test]
    fn parameter_docs_are_templates() {
        assert_eq!(
            template("Extra stops: `fee` for every stop beyond the first `included`."),
            Some("Extra stops: `fee` for every stop beyond the first `included`.")
        );
        assert_eq!(
            template(
                "Detention: the time `dwell` runs past `free`, rounded up to the next `increment`, at `rate`."
            ),
            Some(
                "Detention: the time `dwell` runs past `free`, rounded up to the next `increment`, at `rate`."
            )
        );
    }

    #[test]
    fn every_template_is_a_valid_icu_message() {
        let problems: Vec<String> = templates()
            .filter_map(|(template, site)| {
                icu_problem(template)
                    .map(|problem| format!("{}: {problem}: {template:?}", at(site)))
            })
            .collect();
        assert!(problems.is_empty(), "{}", problems.join("\n"));
    }

    #[test]
    fn the_icu_check_finds_what_hosts_misread() {
        assert!(icu_problem("Rate {0} per mile.").is_some());
        assert!(icu_problem("It''s billed.").is_some());
        assert!(icu_problem("Every # stop.").is_some());
        assert!(icu_problem("A `List<Break>` of breaks.").is_some());
        assert!(icu_problem("Ends with </b> here.").is_some());
        assert!(icu_problem("Weight < 100 lb or > 5 lb.").is_none());
        assert!(icu_problem("The break's minimum.").is_none());
        assert!(icu_problem("$0").is_some());
    }

    #[test]
    fn every_template_is_in_the_catalog_for_both_hosts() {
        let catalog: Json = serde_json::from_str(CATALOG).unwrap_or(Json::Null);
        let has = |entry: &Json, key: &str, value: &str| {
            entry
                .get(key)
                .and_then(Json::as_array)
                .is_some_and(|list| list.iter().any(|item| item.as_str() == Some(value)))
        };
        let missing: Vec<String> = templates()
            .filter(|(template, _)| {
                catalog.get(*template).is_none_or(|entry| {
                    !(has(entry, "scope", "go")
                        && has(entry, "scope", "ts")
                        && has(entry, "areas", "apps/web/rules")
                        && has(entry, "areas", "rules/tariff"))
                })
            })
            .map(|(template, site)| format!("{}: {template:?}", at(site)))
            .collect();
        assert!(
            missing.is_empty(),
            "these standard library docs are not in i18n/messages.en.json for the Go scope and the web app; run `task i18n`:\n{}",
            missing.join("\n")
        );
    }
}
