//! The outline of a file: what the editor's outline panel lists and what it can fold.

use serde::Serialize;
use trenova_module_kit::span::{Range, Span, Utf16Index};

use super::ast::{Block, FlowDecl, GuideDecl, Header, Item, Stmt};

/// One entry of the outline.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Symbol {
    /// What it is called.
    pub name: String,
    /// `setting`, `interchange`, `segment`, `loop`, `field`, `rule`, `receive`, `respond`,
    /// `send`, `expect`, `fn`, `crosswalk`, `const`, `test`, `property` or `guide`.
    pub kind: &'static str,
    /// The whole declaration.
    pub range: Range,
    /// Its name.
    pub selection: Range,
    /// What it contains.
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub children: Vec<Symbol>,
}

/// A foldable range.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
pub struct Fold {
    /// From the start of its first line to the end of its last.
    pub range: Range,
}

/// The outline of a file.
#[must_use]
pub fn outline(source: &str, items: &[Item], header: &Header) -> Vec<Symbol> {
    let index = Utf16Index::new(source);
    let builder = Builder {
        source,
        index: &index,
    };
    let _ = header;
    items.iter().filter_map(|item| builder.item(item)).collect()
}

/// Every multi-line block of a file, outermost first.
#[must_use]
pub fn folds(source: &str, items: &[Item]) -> Vec<Fold> {
    let index = Utf16Index::new(source);
    let mut spans = Vec::new();
    for item in items {
        collect_item_folds(item, &mut spans);
    }
    spans
        .into_iter()
        .filter(|span| {
            source
                .get(span.start as usize..span.end as usize)
                .is_some_and(|text| text.contains('\n'))
        })
        .map(|span| Fold {
            range: index.range(span),
        })
        .collect()
}

fn collect_item_folds(item: &Item, spans: &mut Vec<Span>) {
    match item {
        Item::Fn(_)
        | Item::Crosswalk(_)
        | Item::Test(_)
        | Item::Property(_)
        | Item::Interchange(_)
        | Item::Flow(_)
        | Item::Guide(_) => spans.push(item.span()),
        Item::Rule(rule) => {
            spans.push(rule.span);
            collect_block_folds(&rule.body, spans);
        }
        Item::Stmt(stmt) => collect_stmt_folds(stmt, spans),
        _ => {}
    }
}

fn collect_block_folds(block: &Block, spans: &mut Vec<Span>) {
    for stmt in &block.stmts {
        collect_stmt_folds(stmt, spans);
    }
}

fn collect_stmt_folds(stmt: &Stmt, spans: &mut Vec<Span>) {
    match stmt {
        Stmt::When { body, span, .. }
        | Stmt::Loop { body, span, .. }
        | Stmt::Each { body, span, .. }
        | Stmt::For { body, span, .. } => {
            spans.push(*span);
            collect_block_folds(body, spans);
        }
        Stmt::Segment { span, .. } => spans.push(*span),
        Stmt::Assign { value, span, .. } => {
            if let super::ast::ExprKind::Each { body, .. } = &value.kind {
                spans.push(*span);
                collect_block_folds(body, spans);
            }
        }
        _ => {}
    }
}

struct Builder<'a> {
    source: &'a str,
    index: &'a Utf16Index,
}

impl Builder<'_> {
    fn symbol(
        &self,
        name: impl Into<String>,
        kind: &'static str,
        span: Span,
        name_span: Span,
    ) -> Symbol {
        Symbol {
            name: name.into(),
            kind,
            range: self.index.range(span),
            selection: self.index.range(name_span),
            children: Vec::new(),
        }
    }

    fn snippet(&self, span: Span) -> String {
        let text = self
            .source
            .get(span.start as usize..span.end as usize)
            .unwrap_or_default();
        let first = text.lines().next().unwrap_or_default().trim();
        if first.chars().count() > 60 {
            let cut: String = first.chars().take(57).collect();
            format!("{cut}...")
        } else {
            first.to_owned()
        }
    }

    fn item(&self, item: &Item) -> Option<Symbol> {
        Some(match item {
            Item::Fn(decl) => self.symbol(&decl.name.name, "fn", decl.span, decl.name.span),
            Item::Crosswalk(decl) => {
                self.symbol(&decl.name.name, "crosswalk", decl.span, decl.name.span)
            }
            Item::Const(decl) => self.symbol(&decl.name.name, "const", decl.span, decl.name.span),
            Item::Test(decl) => self.symbol(&decl.name.value, "test", decl.span, decl.name.span),
            Item::Property(decl) => {
                self.symbol(&decl.name.value, "property", decl.span, decl.name.span)
            }
            Item::Rule(decl) => self.symbol(&decl.title.value, "rule", decl.span, decl.title.span),
            Item::Setting(decl) => {
                self.symbol(&decl.name.name, "setting", decl.span, decl.name.span)
            }
            Item::Interchange(block) => {
                self.symbol("interchange", "interchange", block.span, block.span)
            }
            Item::Zone(zone) => self.symbol(
                format!("zone {}", zone.name),
                "setting",
                zone.span,
                zone.span,
            ),
            Item::Currency(currency) => self.symbol(
                format!("currency {}", currency.value),
                "setting",
                currency.span,
                currency.span,
            ),
            Item::Guide(decl) => self.symbol(
                self.snippet(decl.span()),
                "guide",
                decl.span(),
                match decl {
                    GuideDecl::Segment { path, .. } | GuideDecl::Loop { path, .. } => {
                        path.first().map_or(decl.span(), |first| {
                            first
                                .span
                                .cover(path.last().map_or(first.span, |last| last.span))
                        })
                    }
                    GuideDecl::Severity { kind, .. } => kind.span,
                },
            ),
            Item::Flow(decl) => self.flow(decl),
            Item::Stmt(stmt) => self.stmt(stmt)?,
            Item::Error(_) => return None,
        })
    }

    fn flow(&self, decl: &FlowDecl) -> Symbol {
        match decl {
            FlowDecl::Receive(receive) => {
                let mut symbol = self.symbol(
                    format!("receive {} as {}", receive.set.name, receive.binder.name),
                    "receive",
                    receive.span,
                    receive.set.span,
                );
                symbol.children = receive
                    .arms
                    .iter()
                    .map(|arm| {
                        let name = arm.purpose.as_ref().map_or_else(
                            || "->".to_owned(),
                            |purpose| format!("on {}", purpose.name),
                        );
                        self.symbol(name, "arm", arm.span, arm.span)
                    })
                    .collect();
                symbol
            }
            FlowDecl::Respond(respond) => self.symbol(
                format!("respond {}", respond.set.name),
                "respond",
                respond.span,
                respond.set.span,
            ),
            FlowDecl::Send(send) => self.symbol(
                format!("send {}", send.set.name),
                "send",
                send.span,
                send.set.span,
            ),
            FlowDecl::Expect(expect) => self.symbol(
                format!("expect {}", expect.set.name),
                "expect",
                expect.span,
                expect.set.span,
            ),
        }
    }

    fn stmt(&self, stmt: &Stmt) -> Option<Symbol> {
        Some(match stmt {
            Stmt::Assign {
                target,
                value,
                span,
                ..
            } => {
                let mut symbol =
                    self.symbol(self.snippet(target.span), "field", *span, target.span);
                if let super::ast::ExprKind::Each { body, .. } = &value.kind {
                    symbol.children = self.block(body);
                }
                symbol
            }
            Stmt::Segment { id, span, .. } => self.symbol(&id.name, "segment", *span, id.span),
            Stmt::Loop { id, body, span, .. } => {
                let mut symbol = self.symbol(&id.name, "loop", *span, id.span);
                symbol.children = self.block(body);
                symbol
            }
            Stmt::Each {
                binder, body, span, ..
            } => {
                let mut symbol =
                    self.symbol(format!("each {}", binder.name), "loop", *span, binder.span);
                symbol.children = self.block(body);
                symbol
            }
            Stmt::When { body, span, .. } => {
                let mut symbol = self.symbol(self.snippet(*span), "field", *span, *span);
                symbol.children = self.block(body);
                symbol
            }
            _ => return None,
        })
    }

    fn block(&self, block: &Block) -> Vec<Symbol> {
        block
            .stmts
            .iter()
            .filter_map(|stmt| self.stmt(stmt))
            .collect()
    }
}

#[cfg(test)]
mod tests {
    use super::{folds, outline};
    use crate::lading::syntax::parser::parse;

    #[test]
    fn a_map_outlines_its_fields_and_folds_its_blocks() {
        let source = include_str!("../../../testdata/lading/acme/maps/tender-in.lading");
        let parsed = parse(source);
        let symbols = outline(source, &parsed.file.items, &parsed.file.header);
        assert!(
            symbols
                .iter()
                .any(|symbol| symbol.name == "stops" && !symbol.children.is_empty())
        );
        assert!(symbols.iter().any(|symbol| symbol.kind == "test"));
        assert!(!folds(source, &parsed.file.items).is_empty());
    }
}
