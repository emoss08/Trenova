//! The IPP wire encoding (RFC 8010 §3).
//!
//! A message is a version, an operation or status code, a request ID, then
//! attribute groups, each opened by a delimiter tag, closed by the end tag,
//! and followed by the document data, if any. Every attribute is a value tag,
//! a two-byte name length, the name, a two-byte value length and the value;
//! further values of the same attribute repeat with an empty name.
//! Collections (RFC 8010 §3.1.6) nest members between `begCollection` and
//! `endCollection`.
//!
//! Decoding is bounded: every length is checked against what is left of the
//! message, collections nest at most [`MAX_DEPTH`] deep, and a message holds
//! at most [`MAX_ATTRIBUTES`] attributes, so a hostile client on the loopback
//! interface cannot make the service allocate or recurse without limit.

use crate::value::{Attribute, Group, GroupTag, Value, tag};

/// How deeply collections may nest.
pub const MAX_DEPTH: usize = 8;
/// The most attributes (counting collection members) one message may hold.
pub const MAX_ATTRIBUTES: usize = 4096;

#[derive(Debug, thiserror::Error, PartialEq, Eq)]
pub enum DecodeError {
    #[error("the message ends early")]
    Truncated,
    #[error("an attribute arrived before any group")]
    NoGroup,
    #[error("an additional value has no attribute to belong to")]
    OrphanValue,
    #[error("a collection is malformed")]
    Collection,
    #[error("collections nest too deeply")]
    TooDeep,
    #[error("the message has too many attributes")]
    TooMany,
    #[error("a {0} value has the wrong length")]
    Length(&'static str),
    #[error("a text value is not UTF-8")]
    Utf8,
}

/// A decoded request or response, with the document data after its
/// attributes.
#[derive(Clone, Debug, PartialEq)]
pub struct Message {
    pub version: (u8, u8),
    /// The operation ID of a request, or the status code of a response.
    pub code: u16,
    pub request_id: u32,
    pub groups: Vec<Group>,
    /// Where the document data starts in the buffer decoded.
    pub data_offset: usize,
}

impl Message {
    /// The first group of a kind.
    pub fn group(&self, tag: GroupTag) -> Option<&Group> {
        self.groups.iter().find(|g| g.tag == tag)
    }

    /// An operation attribute by name.
    pub fn operation(&self, name: &str) -> Option<&Attribute> {
        self.group(GroupTag::Operation).and_then(|g| g.get(name))
    }
}

struct Reader<'a> {
    bytes: &'a [u8],
    at: usize,
    attributes: usize,
}

impl<'a> Reader<'a> {
    fn take(&mut self, len: usize) -> Result<&'a [u8], DecodeError> {
        let end = self.at.checked_add(len).ok_or(DecodeError::Truncated)?;
        let slice = self.bytes.get(self.at..end).ok_or(DecodeError::Truncated)?;
        self.at = end;
        Ok(slice)
    }

    fn u8(&mut self) -> Result<u8, DecodeError> {
        Ok(self.take(1)?[0])
    }

    fn u16(&mut self) -> Result<u16, DecodeError> {
        let b = self.take(2)?;
        Ok(u16::from_be_bytes([b[0], b[1]]))
    }

    fn u32(&mut self) -> Result<u32, DecodeError> {
        let b = self.take(4)?;
        Ok(u32::from_be_bytes([b[0], b[1], b[2], b[3]]))
    }

    fn peek(&self) -> Option<u8> {
        self.bytes.get(self.at).copied()
    }

    /// One attribute record: its tag, name and raw value.
    fn record(&mut self) -> Result<(u8, &'a [u8], &'a [u8]), DecodeError> {
        let value_tag = self.u8()?;
        let name_len = usize::from(self.u16()?);
        let name = self.take(name_len)?;
        let value_len = usize::from(self.u16()?);
        let value = self.take(value_len)?;
        Ok((value_tag, name, value))
    }

    fn count(&mut self) -> Result<(), DecodeError> {
        self.attributes += 1;
        if self.attributes > MAX_ATTRIBUTES {
            return Err(DecodeError::TooMany);
        }
        Ok(())
    }

    /// The members of a collection whose `begCollection` was just read.
    fn collection(&mut self, depth: usize) -> Result<Vec<Attribute>, DecodeError> {
        if depth > MAX_DEPTH {
            return Err(DecodeError::TooDeep);
        }
        let mut members: Vec<Attribute> = Vec::new();
        loop {
            let (value_tag, name, raw) = self.record()?;
            if !name.is_empty() {
                return Err(DecodeError::Collection);
            }
            match value_tag {
                tag::END_COLLECTION => return Ok(members),
                tag::MEMBER_ATTR_NAME => {
                    self.count()?;
                    let member = std::str::from_utf8(raw).map_err(|_| DecodeError::Utf8)?;
                    let (value_tag, name, raw) = self.record()?;
                    if !name.is_empty() {
                        return Err(DecodeError::Collection);
                    }
                    let value = self.value(value_tag, raw, depth)?;
                    members.push(Attribute::new(member, vec![value]));
                }
                _ => {
                    let value = self.value(value_tag, raw, depth)?;
                    members
                        .last_mut()
                        .ok_or(DecodeError::Collection)?
                        .values
                        .push(value);
                }
            }
        }
    }

    fn value(&mut self, value_tag: u8, raw: &[u8], depth: usize) -> Result<Value, DecodeError> {
        if value_tag == tag::BEG_COLLECTION {
            return self.collection(depth + 1).map(Value::Collection);
        }
        Value::decode(value_tag, raw)
    }
}

/// Decodes a message. The document data is left in place, from
/// [`Message::data_offset`].
pub fn decode(bytes: &[u8]) -> Result<Message, DecodeError> {
    let mut reader = Reader {
        bytes,
        at: 0,
        attributes: 0,
    };
    let version = (reader.u8()?, reader.u8()?);
    let code = reader.u16()?;
    let request_id = reader.u32()?;
    let mut groups: Vec<Group> = Vec::new();

    loop {
        let next = reader.peek().ok_or(DecodeError::Truncated)?;
        if next == tag::END_OF_ATTRIBUTES {
            reader.u8()?;
            break;
        }
        if next < 0x10 {
            reader.u8()?;
            groups.push(Group::new(GroupTag::from_byte(next)));
            continue;
        }
        let (value_tag, name, raw) = reader.record()?;
        let value = reader.value(value_tag, raw, 0)?;
        let group = groups.last_mut().ok_or(DecodeError::NoGroup)?;
        if name.is_empty() {
            group
                .attributes
                .last_mut()
                .ok_or(DecodeError::OrphanValue)?
                .values
                .push(value);
        } else {
            reader.count()?;
            let name = std::str::from_utf8(name).map_err(|_| DecodeError::Utf8)?;
            group.attributes.push(Attribute::new(name, vec![value]));
        }
    }

    Ok(Message {
        version,
        code,
        request_id,
        groups,
        data_offset: reader.at,
    })
}

fn put_record(out: &mut Vec<u8>, value_tag: u8, name: &str, value: &[u8]) {
    out.push(value_tag);
    let name_len = u16::try_from(name.len()).unwrap_or(u16::MAX);
    out.extend_from_slice(&name_len.to_be_bytes());
    out.extend_from_slice(&name.as_bytes()[..usize::from(name_len)]);
    let value_len = u16::try_from(value.len()).unwrap_or(u16::MAX);
    out.extend_from_slice(&value_len.to_be_bytes());
    out.extend_from_slice(&value[..usize::from(value_len)]);
}

fn put_value(out: &mut Vec<u8>, name: &str, value: &Value) {
    match value {
        Value::Collection(members) => {
            put_record(out, tag::BEG_COLLECTION, name, &[]);
            for member in members {
                put_record(out, tag::MEMBER_ATTR_NAME, "", member.name.as_bytes());
                for member_value in &member.values {
                    put_value(out, "", member_value);
                }
            }
            put_record(out, tag::END_COLLECTION, "", &[]);
        }
        other => {
            let (value_tag, raw) = other.encode();
            put_record(out, value_tag, name, &raw);
        }
    }
}

/// Encodes a message with no document data.
pub fn encode(version: (u8, u8), code: u16, request_id: u32, groups: &[Group]) -> Vec<u8> {
    let mut out = Vec::with_capacity(512);
    out.extend_from_slice(&[version.0, version.1]);
    out.extend_from_slice(&code.to_be_bytes());
    out.extend_from_slice(&request_id.to_be_bytes());
    for group in groups {
        out.push(group.tag.byte());
        for attribute in &group.attributes {
            let mut values = attribute.values.iter();
            if let Some(first) = values.next() {
                put_value(&mut out, &attribute.name, first);
            }
            for value in values {
                put_value(&mut out, "", value);
            }
        }
    }
    out.push(tag::END_OF_ATTRIBUTES);
    out
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::value::Resolution;

    fn media_col() -> Value {
        Value::Collection(vec![
            Attribute::new(
                "media-size",
                vec![Value::Collection(vec![
                    Attribute::new("x-dimension", vec![Value::Integer(21590)]),
                    Attribute::new("y-dimension", vec![Value::Integer(27940)]),
                ])],
            ),
            Attribute::new("media-top-margin", vec![Value::Integer(0)]),
        ])
    }

    fn sample() -> Vec<Group> {
        let mut operation = Group::new(GroupTag::Operation);
        operation.push("attributes-charset", Value::Charset("utf-8".into()));
        operation.push(
            "attributes-natural-language",
            Value::NaturalLanguage("en".into()),
        );
        operation.push(
            "printer-uri",
            Value::Uri("ipp://127.0.0.1:631/ipp/print".into()),
        );
        operation.push(
            "requested-attributes",
            Value::Keyword("printer-name".into()),
        );
        operation
            .get_mut("requested-attributes")
            .expect("attribute")
            .values
            .push(Value::Keyword("media-col-database".into()));
        let mut job = Group::new(GroupTag::Job);
        job.push("media-col", media_col());
        job.push("copies", Value::Integer(1));
        job.push(
            "printer-resolution",
            Value::Resolution(Resolution {
                x: 300,
                y: 300,
                units: 3,
            }),
        );
        job.push("page-ranges", Value::Range(1, 5));
        job.push("job-name", Value::Name("Rate confirmation".into()));
        vec![operation, job]
    }

    #[test]
    fn a_message_round_trips_with_multi_values_and_nested_collections() {
        let bytes = encode((2, 0), 0x0002, 42, &sample());
        let mut with_data = bytes.clone();
        with_data.extend_from_slice(b"%PDF-1.7");
        let message = decode(&with_data).expect("decodes");
        assert_eq!(message.version, (2, 0));
        assert_eq!(message.code, 2);
        assert_eq!(message.request_id, 42);
        assert_eq!(message.groups, sample());
        assert_eq!(&with_data[message.data_offset..], b"%PDF-1.7");
        assert_eq!(
            message
                .operation("requested-attributes")
                .map(|a| a.values.len()),
            Some(2)
        );
    }

    #[test]
    fn every_length_is_checked_against_the_message() {
        let bytes = encode((2, 0), 0x000B, 1, &sample());
        for cut in [0, 7, 9, 20, bytes.len() - 1] {
            assert!(decode(&bytes[..cut]).is_err(), "cut at {cut}");
        }
        let mut orphan = vec![2, 0, 0, 11, 0, 0, 0, 1, 0x01];
        orphan.extend_from_slice(&[0x44, 0, 0, 0, 1, b'x', 0x03]);
        assert_eq!(decode(&orphan), Err(DecodeError::OrphanValue));
        let no_group = vec![2, 0, 0, 11, 0, 0, 0, 1, 0x44, 0, 1, b'a', 0, 0, 0x03];
        assert_eq!(decode(&no_group), Err(DecodeError::NoGroup));
    }

    #[test]
    fn collections_cannot_nest_without_limit() {
        let mut bytes = vec![2, 0, 0, 2, 0, 0, 0, 1, 0x02];
        put_record(&mut bytes, tag::BEG_COLLECTION, "media-col", &[]);
        for _ in 0..=MAX_DEPTH {
            put_record(&mut bytes, tag::MEMBER_ATTR_NAME, "", b"m");
            put_record(&mut bytes, tag::BEG_COLLECTION, "", &[]);
        }
        assert_eq!(decode(&bytes), Err(DecodeError::TooDeep));
    }

    #[test]
    fn a_flood_of_attributes_is_refused() {
        let mut bytes = vec![2, 0, 0, 2, 0, 0, 0, 1, 0x01];
        for i in 0..=MAX_ATTRIBUTES {
            put_record(
                &mut bytes,
                tag::INTEGER,
                &format!("a{i}"),
                &1i32.to_be_bytes(),
            );
        }
        bytes.push(tag::END_OF_ATTRIBUTES);
        assert_eq!(decode(&bytes), Err(DecodeError::TooMany));
    }
}
