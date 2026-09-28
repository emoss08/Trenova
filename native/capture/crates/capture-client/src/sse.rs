//! A server-sent events parser, per the WHATWG HTML event-stream grammar.
//!
//! It takes the body in whatever chunks the network delivers and yields each
//! event once its terminating blank line arrives. A line longer than
//! [`MAX_LINE`] ends the stream: the device stream's frames are small, and a
//! peer that never sends a newline must not grow the buffer without bound.

/// The longest line accepted.
pub const MAX_LINE: usize = 1 << 20;

/// One dispatched event.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct SseEvent {
    /// `message` when the stream named none.
    pub event: String,
    pub data: String,
    /// The last event ID seen at dispatch, for `Last-Event-ID`.
    pub id: Option<String>,
}

#[derive(Debug, thiserror::Error, PartialEq, Eq)]
#[error("an event-stream line is longer than {MAX_LINE} bytes")]
pub struct LineTooLong;

#[derive(Debug, Default)]
pub struct SseParser {
    line: Vec<u8>,
    /// A CR ended the last line; a following LF belongs to it.
    after_cr: bool,
    event: String,
    data: String,
    has_data: bool,
    last_id: Option<String>,
}

impl SseParser {
    pub fn new() -> Self {
        Self::default()
    }

    /// Feeds a chunk, returning the events it completed.
    pub fn push(&mut self, chunk: &[u8]) -> Result<Vec<SseEvent>, LineTooLong> {
        let mut events = Vec::new();
        for &byte in chunk {
            match byte {
                b'\n' if self.after_cr => self.after_cr = false,
                b'\n' | b'\r' => {
                    self.after_cr = byte == b'\r';
                    let line = std::mem::take(&mut self.line);
                    if let Some(event) = self.line_done(&line) {
                        events.push(event);
                    }
                }
                _ => {
                    self.after_cr = false;
                    if self.line.len() >= MAX_LINE {
                        return Err(LineTooLong);
                    }
                    self.line.push(byte);
                }
            }
        }
        Ok(events)
    }

    fn line_done(&mut self, line: &[u8]) -> Option<SseEvent> {
        if line.is_empty() {
            return self.dispatch();
        }
        if line[0] == b':' {
            return None;
        }
        let text = String::from_utf8_lossy(line);
        let (field, value) = match text.find(':') {
            Some(colon) => {
                let value = &text[colon + 1..];
                (&text[..colon], value.strip_prefix(' ').unwrap_or(value))
            }
            None => (text.as_ref(), ""),
        };
        match field {
            "event" => value.clone_into(&mut self.event),
            "data" => {
                if self.has_data {
                    self.data.push('\n');
                }
                self.data.push_str(value);
                self.has_data = true;
            }
            "id" if !value.contains('\0') => self.last_id = Some(value.to_owned()),
            _ => {}
        }
        None
    }

    fn dispatch(&mut self) -> Option<SseEvent> {
        let event = std::mem::take(&mut self.event);
        if !self.has_data {
            return None;
        }
        self.has_data = false;
        Some(SseEvent {
            event: if event.is_empty() {
                "message".to_owned()
            } else {
                event
            },
            data: std::mem::take(&mut self.data),
            id: self.last_id.clone(),
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn events_are_dispatched_on_the_blank_line_across_chunks() {
        let mut parser = SseParser::new();
        assert!(
            parser
                .push(b"id: 17-0\nevent: capture.req")
                .expect("chunk")
                .is_empty()
        );
        let events = parser
            .push(b"uest\ndata: {\"deviceId\":\"cdev_1\"}\n\n: keepalive\n\n")
            .expect("chunk");
        assert_eq!(
            events,
            vec![SseEvent {
                event: "capture.request".into(),
                data: r#"{"deviceId":"cdev_1"}"#.into(),
                id: Some("17-0".into()),
            }]
        );
    }

    #[test]
    fn crlf_and_cr_line_endings_and_multi_line_data() {
        let mut parser = SseParser::new();
        let events = parser
            .push(b"data: a\r\ndata:b\r\rdata: c\r")
            .expect("chunk");
        assert_eq!(events.len(), 1);
        assert_eq!(events[0].data, "a\nb");
        assert_eq!(events[0].event, "message");
        let events = parser.push(b"\n\r\n").expect("chunk");
        assert_eq!(events.len(), 1);
        assert_eq!(events[0].data, "c");
    }

    #[test]
    fn an_event_with_no_data_is_not_dispatched_and_the_id_persists() {
        let mut parser = SseParser::new();
        assert!(
            parser
                .push(b"event: heartbeat\nid: 3\n\n")
                .expect("chunk")
                .is_empty()
        );
        let events = parser
            .push(b"event: heartbeat\ndata: {}\n\n")
            .expect("chunk");
        assert_eq!(events[0].id.as_deref(), Some("3"));
        assert_eq!(events[0].event, "heartbeat");
    }

    #[test]
    fn a_line_that_never_ends_is_refused() {
        let mut parser = SseParser::new();
        let chunk = vec![b'x'; MAX_LINE];
        assert!(parser.push(&chunk).is_ok());
        assert_eq!(parser.push(b"y"), Err(LineTooLong));
    }
}
