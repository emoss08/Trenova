//! User-facing text, as English message templates the caller translates.
//!
//! The crate never translates. Every sentence a person may read is written as
//! `message!("...")`: an ICU message in English with positional arguments (`{0}`, and
//! `{0, plural, one {# stop} other {# stops}}` for counts), exactly the form `t()`/`translate()`
//! take in the web app and `shared/i18n` takes in Go. A [`Message`] carries the template and its
//! arguments over the wire; the browser and the server look the template up in their catalogs,
//! fill the arguments and fall back to English.
//!
//! `task i18n` extracts every `message!` literal in this crate into the catalogs (the Go scope
//! and the web app's `web` bundle), so the literal must be a single string literal: no
//! `concat!`, no variables. A template never contains a bare `{` or `}` that is not a
//! placeholder, since ICU would read it as one; write `'{'` instead.

use serde::Serialize;

/// Marks a string literal as a user-facing message template for the i18n extractor.
#[macro_export]
macro_rules! message {
    ($text:literal) => {
        $text
    };
}

/// One argument of a message.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
#[serde(untagged)]
pub enum Arg {
    /// Text substituted as written: a name, a value shown as text.
    Text(String),
    /// A count, which a plural placeholder selects on.
    Count(i64),
}

impl From<&str> for Arg {
    fn from(value: &str) -> Self {
        Self::Text(value.to_owned())
    }
}

impl From<String> for Arg {
    fn from(value: String) -> Self {
        Self::Text(value)
    }
}

impl From<i64> for Arg {
    fn from(value: i64) -> Self {
        Self::Count(value)
    }
}

impl From<usize> for Arg {
    fn from(value: usize) -> Self {
        Self::Count(i64::try_from(value).unwrap_or(i64::MAX))
    }
}

/// A message template and its arguments, as it crosses the wire:
/// `{"message": "unknown name {0}", "args": ["foo"]}`.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub struct Message {
    /// The English ICU template, also the catalog key.
    pub message: &'static str,
    /// Positional arguments, `{0}` first.
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub args: Vec<Arg>,
}

impl Message {
    /// A message with no arguments.
    #[must_use]
    pub fn new(message: &'static str) -> Self {
        Self {
            message,
            args: Vec::new(),
        }
    }

    /// A message with positional arguments.
    #[must_use]
    pub fn with(message: &'static str, args: impl IntoIterator<Item = Arg>) -> Self {
        Self {
            message,
            args: args.into_iter().collect(),
        }
    }

    /// The English text with the arguments filled in, for tests and developer logs only.
    ///
    /// Plural placeholders take the `other` branch unless the count is exactly one.
    #[must_use]
    pub fn english(&self) -> String {
        render(self.message, &self.args)
    }
}

fn render(template: &str, args: &[Arg]) -> String {
    let mut out = String::with_capacity(template.len());
    let mut rest = template;
    while let Some(open) = rest.find('{') {
        out.push_str(&rest[..open]);
        let Some((inner, after)) = balanced(&rest[open..]) else {
            out.push_str(&rest[open..]);
            return out;
        };
        out.push_str(&placeholder(inner, args));
        rest = after;
    }
    out.push_str(rest);
    out
}

/// Splits `{...}rest` at the brace matching the first one, returning the inside and the rest.
fn balanced(text: &str) -> Option<(&str, &str)> {
    let mut depth = 0usize;
    for (index, character) in text.char_indices() {
        match character {
            '{' => depth += 1,
            '}' => {
                depth -= 1;
                if depth == 0 {
                    return Some((&text[1..index], &text[index + 1..]));
                }
            }
            _ => {}
        }
    }
    None
}

fn placeholder(inner: &str, args: &[Arg]) -> String {
    let mut parts = inner.splitn(3, ',');
    let index = parts.next().unwrap_or_default().trim();
    let Ok(position) = index.parse::<usize>() else {
        return format!("{{{inner}}}");
    };
    let arg = args.get(position);
    match (parts.next().map(str::trim), parts.next()) {
        (Some("plural"), Some(branches)) => {
            let count = match arg {
                Some(Arg::Count(count)) => *count,
                _ => 0,
            };
            let branch = plural_branch(branches, if count == 1 { "one" } else { "other" })
                .or_else(|| plural_branch(branches, "other"))
                .unwrap_or_default();
            render(branch, args).replace('#', &count.to_string())
        }
        _ => match arg {
            Some(Arg::Text(text)) => text.clone(),
            Some(Arg::Count(count)) => count.to_string(),
            None => format!("{{{inner}}}"),
        },
    }
}

fn plural_branch<'a>(branches: &'a str, selector: &str) -> Option<&'a str> {
    let mut rest = branches;
    while let Some(open) = rest.find('{') {
        let key = rest[..open].trim();
        let (inner, after) = balanced(&rest[open..])?;
        if key == selector {
            return Some(inner);
        }
        rest = after;
    }
    None
}

#[cfg(test)]
mod tests {
    use super::{Arg, Message};

    #[test]
    fn fills_positional_arguments() {
        let message = Message::with(
            "unknown name {0} near {1}",
            [Arg::from("foo"), Arg::from("bar")],
        );
        assert_eq!(message.english(), "unknown name foo near bar");
    }

    #[test]
    fn selects_plural_branches() {
        let template = "{0, plural, one {# stop} other {# stops}} past free time";
        assert_eq!(
            Message::with(template, [Arg::from(1i64)]).english(),
            "1 stop past free time"
        );
        assert_eq!(
            Message::with(template, [Arg::from(3i64)]).english(),
            "3 stops past free time"
        );
    }

    #[test]
    fn serialises_without_empty_args() {
        let json = serde_json::to_string(&Message::new("Add ';'")).unwrap_or_default();
        assert_eq!(json, r#"{"message":"Add ';'"}"#);
    }
}
