//! What a script prints, with only its end kept.
//!
//! The model reads the end of a script's output (its totals and its last lines), so the
//! buffer keeps at most twice the limit and drops from the front when it fills, and
//! [`Tail::finish`] cuts it to the limit on a character boundary. Printing never fails: a
//! script that prints more than the limit loses its first lines, not its run.

use std::borrow::Cow;

use monty_types::{MontyException, PrintWriterCallback};

/// The end of a script's output.
#[derive(Debug)]
pub struct Tail {
    text: String,
    limit: usize,
    truncated: bool,
}

impl Tail {
    /// A buffer that keeps the last `limit` characters.
    #[must_use]
    pub fn new(limit: usize) -> Self {
        Self {
            text: String::new(),
            limit,
            truncated: false,
        }
    }

    fn push(&mut self, text: &str) {
        self.text.push_str(text);
        if self.text.len() > self.limit.saturating_mul(2).max(64) {
            self.trim();
        }
    }

    fn trim(&mut self) {
        let chars = self.text.chars().count();
        if chars <= self.limit {
            return;
        }
        let cut = self
            .text
            .char_indices()
            .nth(chars - self.limit)
            .map_or(self.text.len(), |(index, _)| index);
        self.text.drain(..cut);
        self.truncated = true;
    }

    /// The kept output and whether anything before it was dropped.
    #[must_use]
    pub fn finish(mut self) -> (String, bool) {
        self.trim();
        (self.text, self.truncated)
    }
}

impl PrintWriterCallback for Tail {
    fn stdout_write(&mut self, output: Cow<'_, str>) -> Result<(), MontyException> {
        self.push(&output);
        Ok(())
    }

    fn stdout_push(&mut self, end: char) -> Result<(), MontyException> {
        let mut buffer = [0u8; 4];
        self.push(end.encode_utf8(&mut buffer));
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::Tail;

    #[test]
    fn keeps_everything_under_the_limit() {
        let mut tail = Tail::new(10);
        tail.push("hello");
        assert_eq!(tail.finish(), ("hello".to_owned(), false));
    }

    #[test]
    fn keeps_the_end_on_a_character_boundary() {
        let mut tail = Tail::new(3);
        for _ in 0..100 {
            tail.push("é");
        }
        tail.push("abc");
        assert_eq!(tail.finish(), ("abc".to_owned(), true));
    }
}
