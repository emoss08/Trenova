//! IPP attribute values and groups (RFC 8010 §3.5).

use crate::codec::DecodeError;

/// The tags this printer reads and writes.
pub mod tag {
    pub const OPERATION_ATTRIBUTES: u8 = 0x01;
    pub const JOB_ATTRIBUTES: u8 = 0x02;
    pub const END_OF_ATTRIBUTES: u8 = 0x03;
    pub const PRINTER_ATTRIBUTES: u8 = 0x04;
    pub const UNSUPPORTED_ATTRIBUTES: u8 = 0x05;

    pub const UNSUPPORTED: u8 = 0x10;
    pub const UNKNOWN: u8 = 0x12;
    pub const NO_VALUE: u8 = 0x13;
    pub const INTEGER: u8 = 0x21;
    pub const BOOLEAN: u8 = 0x22;
    pub const ENUM: u8 = 0x23;
    pub const OCTET_STRING: u8 = 0x30;
    pub const DATE_TIME: u8 = 0x31;
    pub const RESOLUTION: u8 = 0x32;
    pub const RANGE_OF_INTEGER: u8 = 0x33;
    pub const BEG_COLLECTION: u8 = 0x34;
    pub const TEXT_WITH_LANGUAGE: u8 = 0x35;
    pub const NAME_WITH_LANGUAGE: u8 = 0x36;
    pub const END_COLLECTION: u8 = 0x37;
    pub const TEXT_WITHOUT_LANGUAGE: u8 = 0x41;
    pub const NAME_WITHOUT_LANGUAGE: u8 = 0x42;
    pub const KEYWORD: u8 = 0x44;
    pub const URI: u8 = 0x45;
    pub const URI_SCHEME: u8 = 0x46;
    pub const CHARSET: u8 = 0x47;
    pub const NATURAL_LANGUAGE: u8 = 0x48;
    pub const MIME_MEDIA_TYPE: u8 = 0x49;
    pub const MEMBER_ATTR_NAME: u8 = 0x4A;
}

/// A resolution, with its units: 3 is dots per inch, 4 per centimetre.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Resolution {
    pub x: i32,
    pub y: i32,
    pub units: u8,
}

/// One attribute value.
#[derive(Clone, Debug, PartialEq)]
pub enum Value {
    Integer(i32),
    Boolean(bool),
    Enum(i32),
    OctetString(Vec<u8>),
    DateTime(Vec<u8>),
    Resolution(Resolution),
    Range(i32, i32),
    Collection(Vec<Attribute>),
    Text(String),
    Name(String),
    Keyword(String),
    Uri(String),
    UriScheme(String),
    Charset(String),
    NaturalLanguage(String),
    MimeMediaType(String),
    /// An out-of-band value: `unsupported`, `unknown` or `no-value`.
    OutOfBand(u8),
    /// A value of a type this printer does not interpret, kept as sent.
    Other(u8, Vec<u8>),
}

fn int(raw: &[u8], what: &'static str) -> Result<i32, DecodeError> {
    <[u8; 4]>::try_from(raw)
        .map(i32::from_be_bytes)
        .map_err(|_| DecodeError::Length(what))
}

fn text(raw: &[u8]) -> Result<String, DecodeError> {
    String::from_utf8(raw.to_vec()).map_err(|_| DecodeError::Utf8)
}

/// The text of a `textWithLanguage` or `nameWithLanguage` value: a two-byte
/// length and the language, then a two-byte length and the text.
fn with_language(raw: &[u8]) -> Result<String, DecodeError> {
    let language_len = usize::from(u16::from_be_bytes(
        raw.get(..2)
            .and_then(|b| b.try_into().ok())
            .ok_or(DecodeError::Length("text with language"))?,
    ));
    let rest = raw
        .get(2 + language_len..)
        .ok_or(DecodeError::Length("text with language"))?;
    let text_len = usize::from(u16::from_be_bytes(
        rest.get(..2)
            .and_then(|b| b.try_into().ok())
            .ok_or(DecodeError::Length("text with language"))?,
    ));
    text(
        rest.get(2..2 + text_len)
            .ok_or(DecodeError::Length("text with language"))?,
    )
}

impl Value {
    pub(crate) fn decode(value_tag: u8, raw: &[u8]) -> Result<Self, DecodeError> {
        Ok(match value_tag {
            tag::UNSUPPORTED | tag::UNKNOWN | tag::NO_VALUE => Self::OutOfBand(value_tag),
            tag::INTEGER => Self::Integer(int(raw, "integer")?),
            tag::ENUM => Self::Enum(int(raw, "enum")?),
            tag::BOOLEAN => match raw {
                [b] => Self::Boolean(*b != 0),
                _ => return Err(DecodeError::Length("boolean")),
            },
            tag::OCTET_STRING => Self::OctetString(raw.to_vec()),
            tag::DATE_TIME => {
                if raw.len() != 11 {
                    return Err(DecodeError::Length("dateTime"));
                }
                Self::DateTime(raw.to_vec())
            }
            tag::RESOLUTION => {
                if raw.len() != 9 {
                    return Err(DecodeError::Length("resolution"));
                }
                Self::Resolution(Resolution {
                    x: int(&raw[..4], "resolution")?,
                    y: int(&raw[4..8], "resolution")?,
                    units: raw[8],
                })
            }
            tag::RANGE_OF_INTEGER => {
                if raw.len() != 8 {
                    return Err(DecodeError::Length("rangeOfInteger"));
                }
                Self::Range(
                    int(&raw[..4], "rangeOfInteger")?,
                    int(&raw[4..], "rangeOfInteger")?,
                )
            }
            tag::TEXT_WITHOUT_LANGUAGE => Self::Text(text(raw)?),
            tag::NAME_WITHOUT_LANGUAGE => Self::Name(text(raw)?),
            tag::TEXT_WITH_LANGUAGE => Self::Text(with_language(raw)?),
            tag::NAME_WITH_LANGUAGE => Self::Name(with_language(raw)?),
            tag::KEYWORD => Self::Keyword(text(raw)?),
            tag::URI => Self::Uri(text(raw)?),
            tag::URI_SCHEME => Self::UriScheme(text(raw)?),
            tag::CHARSET => Self::Charset(text(raw)?),
            tag::NATURAL_LANGUAGE => Self::NaturalLanguage(text(raw)?),
            tag::MIME_MEDIA_TYPE => Self::MimeMediaType(text(raw)?),
            other => Self::Other(other, raw.to_vec()),
        })
    }

    /// The tag and bytes of a value; collections are written by the codec.
    pub(crate) fn encode(&self) -> (u8, Vec<u8>) {
        match self {
            Self::Integer(v) => (tag::INTEGER, v.to_be_bytes().to_vec()),
            Self::Enum(v) => (tag::ENUM, v.to_be_bytes().to_vec()),
            Self::Boolean(v) => (tag::BOOLEAN, vec![u8::from(*v)]),
            Self::OctetString(v) => (tag::OCTET_STRING, v.clone()),
            Self::DateTime(v) => (tag::DATE_TIME, v.clone()),
            Self::Resolution(r) => {
                let mut raw = Vec::with_capacity(9);
                raw.extend_from_slice(&r.x.to_be_bytes());
                raw.extend_from_slice(&r.y.to_be_bytes());
                raw.push(r.units);
                (tag::RESOLUTION, raw)
            }
            Self::Range(low, high) => {
                let mut raw = low.to_be_bytes().to_vec();
                raw.extend_from_slice(&high.to_be_bytes());
                (tag::RANGE_OF_INTEGER, raw)
            }
            Self::Collection(_) => (tag::BEG_COLLECTION, Vec::new()),
            Self::Text(v) => (tag::TEXT_WITHOUT_LANGUAGE, v.as_bytes().to_vec()),
            Self::Name(v) => (tag::NAME_WITHOUT_LANGUAGE, v.as_bytes().to_vec()),
            Self::Keyword(v) => (tag::KEYWORD, v.as_bytes().to_vec()),
            Self::Uri(v) => (tag::URI, v.as_bytes().to_vec()),
            Self::UriScheme(v) => (tag::URI_SCHEME, v.as_bytes().to_vec()),
            Self::Charset(v) => (tag::CHARSET, v.as_bytes().to_vec()),
            Self::NaturalLanguage(v) => (tag::NATURAL_LANGUAGE, v.as_bytes().to_vec()),
            Self::MimeMediaType(v) => (tag::MIME_MEDIA_TYPE, v.as_bytes().to_vec()),
            Self::OutOfBand(t) => (*t, Vec::new()),
            Self::Other(t, raw) => (*t, raw.clone()),
        }
    }

    /// The text of any string-typed value.
    pub fn as_str(&self) -> Option<&str> {
        match self {
            Self::Text(v)
            | Self::Name(v)
            | Self::Keyword(v)
            | Self::Uri(v)
            | Self::UriScheme(v)
            | Self::Charset(v)
            | Self::NaturalLanguage(v)
            | Self::MimeMediaType(v) => Some(v),
            _ => None,
        }
    }

    pub fn as_int(&self) -> Option<i32> {
        match self {
            Self::Integer(v) | Self::Enum(v) => Some(*v),
            _ => None,
        }
    }

    pub fn as_bool(&self) -> Option<bool> {
        match self {
            Self::Boolean(v) => Some(*v),
            _ => None,
        }
    }
}

/// An attribute: a name and one or more values.
#[derive(Clone, Debug, PartialEq)]
pub struct Attribute {
    pub name: String,
    pub values: Vec<Value>,
}

impl Attribute {
    pub fn new(name: impl Into<String>, values: Vec<Value>) -> Self {
        Self {
            name: name.into(),
            values,
        }
    }

    pub fn first(&self) -> Option<&Value> {
        self.values.first()
    }
}

/// Which group an attribute is in.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GroupTag {
    Operation,
    Job,
    Printer,
    Unsupported,
    Other(u8),
}

impl GroupTag {
    pub fn from_byte(byte: u8) -> Self {
        match byte {
            tag::OPERATION_ATTRIBUTES => Self::Operation,
            tag::JOB_ATTRIBUTES => Self::Job,
            tag::PRINTER_ATTRIBUTES => Self::Printer,
            tag::UNSUPPORTED_ATTRIBUTES => Self::Unsupported,
            other => Self::Other(other),
        }
    }

    pub fn byte(self) -> u8 {
        match self {
            Self::Operation => tag::OPERATION_ATTRIBUTES,
            Self::Job => tag::JOB_ATTRIBUTES,
            Self::Printer => tag::PRINTER_ATTRIBUTES,
            Self::Unsupported => tag::UNSUPPORTED_ATTRIBUTES,
            Self::Other(byte) => byte,
        }
    }
}

/// An attribute group.
#[derive(Clone, Debug, PartialEq)]
pub struct Group {
    pub tag: GroupTag,
    pub attributes: Vec<Attribute>,
}

impl Group {
    pub fn new(tag: GroupTag) -> Self {
        Self {
            tag,
            attributes: Vec::new(),
        }
    }

    pub fn get(&self, name: &str) -> Option<&Attribute> {
        self.attributes.iter().find(|a| a.name == name)
    }

    pub fn get_mut(&mut self, name: &str) -> Option<&mut Attribute> {
        self.attributes.iter_mut().find(|a| a.name == name)
    }

    /// Adds a single-valued attribute.
    pub fn push(&mut self, name: &str, value: Value) {
        self.attributes.push(Attribute::new(name, vec![value]));
    }

    /// Adds a multi-valued attribute.
    pub fn push_all(&mut self, name: &str, values: Vec<Value>) {
        self.attributes.push(Attribute::new(name, values));
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn a_name_with_language_reads_as_its_text() {
        let mut raw = vec![0, 5];
        raw.extend_from_slice(b"en-us");
        raw.extend_from_slice(&[0, 4]);
        raw.extend_from_slice(b"jdoe");
        assert_eq!(
            Value::decode(tag::NAME_WITH_LANGUAGE, &raw),
            Ok(Value::Name("jdoe".into()))
        );
        assert_eq!(
            Value::decode(tag::NAME_WITH_LANGUAGE, &raw[..8]),
            Err(DecodeError::Length("text with language"))
        );
    }

    #[test]
    fn fixed_width_values_must_have_their_width() {
        assert!(Value::decode(tag::INTEGER, &[0, 0, 1]).is_err());
        assert!(Value::decode(tag::BOOLEAN, &[]).is_err());
        assert!(Value::decode(tag::RESOLUTION, &[0; 8]).is_err());
        assert_eq!(
            Value::decode(tag::NO_VALUE, &[]),
            Ok(Value::OutOfBand(tag::NO_VALUE))
        );
        assert_eq!(
            Value::decode(0x7F, b"x"),
            Ok(Value::Other(0x7F, b"x".to_vec()))
        );
    }
}
