//! Documentation the module shows a person: the doc of a schema field, a declaration, a
//! parameter or a module, as completion, hover, signature help, the catalog and the inputs form
//! send it.
//!
//! A doc is one of two things, and [`Doc`] says which:
//!
//! - the author's own words (a doc comment in a tariff or an organisation library, a rate
//!   table's name, a charge code's description), sent as written: `"doc": "Paid per mile."`;
//! - text the module ships (the schema's docs, the built-ins', the standard library's doc
//!   comments), sent as a [`Message`] the hosts translate: `"doc": {"message": "…"}`.
//!
//! A standard library doc comment becomes a message whose template is the comment with its
//! whitespace collapsed ([`collapse`]), the only form the catalogs, which collapse whitespace in
//! their keys, can find. [`crate::tariff::stdlib::docs`] holds those templates.

use crate::tariff::stdlib;
use crate::tariff::syntax::ast;
use serde::{Serialize, Serializer};
use trenova_module_kit::message::Message;

/// A doc as the hosts receive it: the author's text, or a message they translate.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Doc {
    /// The author's words (markdown), shown as written.
    Text(String),
    /// Text the module ships, which the hosts translate.
    Message(Message),
}

impl Serialize for Doc {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        match self {
            Self::Text(text) => serializer.serialize_str(text),
            Self::Message(message) => message.serialize(serializer),
        }
    }
}

impl Doc {
    /// A doc the module ships, from its `message!` template.
    #[must_use]
    pub fn message(template: &'static str) -> Self {
        Self::Message(Message::new(template))
    }

    /// The doc comment `doc` as written in a module: a message when the module is the standard
    /// library's (`std` is [`crate::tariff::syntax::program::Module::std`]), the author's text
    /// otherwise.
    #[must_use]
    pub fn written(std: bool, doc: &ast::Doc) -> Self {
        if std && let Some(template) = stdlib::docs::template(&collapse(&doc.text)) {
            return Self::message(template);
        }
        Self::Text(doc.text.clone())
    }

    /// [`Doc::written`] for an optional doc comment.
    #[must_use]
    pub fn of(std: bool, doc: Option<&ast::Doc>) -> Option<Self> {
        doc.map(|doc| Self::written(std, doc))
    }

    /// The doc of a function's parameter `name`, from this doc of the function: the sentences
    /// naming it ([`param_text`]), translated when the function's doc is.
    #[must_use]
    pub fn param(&self, name: &str) -> Option<Self> {
        match self {
            Self::Text(text) => param_text(text, name).map(Self::Text),
            Self::Message(message) => {
                let text = param_text(message.message, name)?;
                Some(stdlib::docs::template(&text).map_or(Self::Text(text), Self::message))
            }
        }
    }

    /// The text: the author's, or the message's English template.
    #[must_use]
    pub fn text(&self) -> &str {
        match self {
            Self::Text(text) => text,
            Self::Message(message) => message.message,
        }
    }
}

/// A doc comment's text as one line: its lines joined with single spaces, runs of whitespace
/// made one space, trimmed. This is a standard library doc's message template.
#[must_use]
pub fn collapse(text: &str) -> String {
    text.split_whitespace().collect::<Vec<_>>().join(" ")
}

/// The sentences of a function's doc that name the parameter in backticks (`` `rate` ``),
/// joined with a space, as that parameter's doc; `None` when no sentence names it. A sentence
/// ends at `. `.
#[must_use]
pub fn param_text(doc: &str, param: &str) -> Option<String> {
    let needle = format!("`{param}`");
    let flat = collapse(doc);
    let sentences: Vec<&str> = flat
        .split_inclusive(". ")
        .map(str::trim)
        .filter(|sentence| sentence.contains(&needle))
        .collect();
    (!sentences.is_empty()).then(|| sentences.join(" "))
}

#[cfg(test)]
mod tests {
    use super::{Doc, collapse, param_text};
    use crate::span::Span;
    use crate::tariff::syntax::ast;

    #[test]
    fn collapses_lines_and_runs_of_whitespace() {
        assert_eq!(
            collapse("  One weight break\nof an  LTL rate.\n"),
            "One weight break of an LTL rate."
        );
    }

    #[test]
    fn a_parameter_doc_is_the_sentences_naming_it() {
        let doc = "Time past `free`, rounded up to `increment`. Billed at `rate`.\nOr `free`.";
        assert_eq!(
            param_text(doc, "rate").as_deref(),
            Some("Billed at `rate`.")
        );
        assert_eq!(
            param_text(doc, "free").as_deref(),
            Some("Time past `free`, rounded up to `increment`. Or `free`.")
        );
        assert_eq!(param_text(doc, "dwell"), None);
    }

    #[test]
    fn serialises_text_as_a_string_and_a_message_as_an_object() {
        let text = serde_json::to_string(&Doc::Text(String::from("Paid per mile.")));
        assert_eq!(text.ok().as_deref(), Some(r#""Paid per mile.""#));
        let message = serde_json::to_string(&Doc::message("The city."));
        assert_eq!(message.ok().as_deref(), Some(r#"{"message":"The city."}"#));
    }

    #[test]
    fn a_doc_outside_the_standard_library_stays_the_authors_words() {
        let written = ast::Doc {
            text: String::from("One weight break of an LTL rate."),
            span: Span::new(0, 0),
        };
        assert_eq!(
            Doc::written(false, &written),
            Doc::Text(String::from("One weight break of an LTL rate."))
        );
        assert_eq!(
            Doc::written(true, &written),
            Doc::message("One weight break of an LTL rate.")
        );
        let unknown = ast::Doc {
            text: String::from("Not a standard library doc."),
            span: Span::new(0, 0),
        };
        assert_eq!(
            Doc::written(true, &unknown),
            Doc::Text(String::from("Not a standard library doc."))
        );
    }
}
