//! Date, time and instant literals as Lading writes them after `@`.

use super::tz::{Zone, days_from_civil, valid_date};

/// Where an instant's local time is read: a zone of the table, or a fixed offset in minutes.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ZoneValue {
    /// A named zone.
    Named(&'static Zone),
    /// A fixed offset from UTC, in minutes (`Z` is 0).
    Fixed(i32),
}

impl ZoneValue {
    /// UTC.
    pub const UTC: Self = Self::Fixed(0);

    /// Its offset from UTC at an instant, in minutes.
    #[must_use]
    pub fn offset_at(self, instant: i64) -> i32 {
        match self {
            Self::Named(zone) => zone.offset_at(instant),
            Self::Fixed(offset) => offset,
        }
    }

    /// Its name: the IANA name, or the offset (`-06:00`, `UTC`).
    #[must_use]
    pub fn name(self) -> String {
        match self {
            Self::Named(zone) => zone.name.to_owned(),
            Self::Fixed(0) => "UTC".to_owned(),
            Self::Fixed(offset) => {
                let sign = if offset < 0 { '-' } else { '+' };
                let offset = offset.unsigned_abs();
                format!("{sign}{:02}:{:02}", offset / 60, offset % 60)
            }
        }
    }

    /// A zone by IANA name or offset.
    #[must_use]
    pub fn parse(text: &str) -> Option<Self> {
        if text == "UTC" || text == "Z" {
            return Some(Self::UTC);
        }
        if let Some(zone) = Zone::named(text) {
            return Some(Self::Named(zone));
        }
        parse_offset(text).map(Self::Fixed)
    }

    /// The instant a local time stands for in this zone.
    #[must_use]
    pub fn instant(self, local_seconds: i64) -> i64 {
        match self {
            Self::Named(zone) => zone.instant(local_seconds),
            Self::Fixed(offset) => local_seconds - i64::from(offset) * 60,
        }
    }
}

fn digits(text: &str) -> Option<u32> {
    if text.is_empty() || !text.bytes().all(|byte| byte.is_ascii_digit()) {
        return None;
    }
    text.parse().ok()
}

/// `2026-01-10` as days since 1970-01-01.
#[must_use]
pub fn parse_date(text: &str) -> Option<i64> {
    let mut parts = text.split('-');
    let year = parts
        .next()
        .filter(|part| part.len() == 4)
        .and_then(digits)?;
    let month = parts
        .next()
        .filter(|part| part.len() == 2)
        .and_then(digits)?;
    let day = parts
        .next()
        .filter(|part| part.len() == 2)
        .and_then(digits)?;
    if parts.next().is_some() || !valid_date(i64::from(year), month, day) {
        return None;
    }
    Some(days_from_civil(i64::from(year), month, day))
}

/// `14:30` or `14:30:15` as seconds since midnight.
#[must_use]
pub fn parse_time(text: &str) -> Option<u32> {
    let mut parts = text.split(':');
    let hours = parts
        .next()
        .filter(|part| part.len() == 2)
        .and_then(digits)?;
    let minutes = parts
        .next()
        .filter(|part| part.len() == 2)
        .and_then(digits)?;
    let seconds = match parts.next() {
        Some(part) if part.len() == 2 => digits(part)?,
        Some(_) => return None,
        None => 0,
    };
    if parts.next().is_some() || hours > 23 || minutes > 59 || seconds > 59 {
        return None;
    }
    Some(hours * 3600 + minutes * 60 + seconds)
}

/// `-06:00` or `+0530` as minutes.
fn parse_offset(text: &str) -> Option<i32> {
    let sign = match text.as_bytes().first()? {
        b'+' => 1,
        b'-' => -1,
        _ => return None,
    };
    let rest = text[1..].replace(':', "");
    if rest.len() != 4 {
        return None;
    }
    let hours = i32::try_from(digits(&rest[..2])?).ok()?;
    let minutes = i32::try_from(digits(&rest[2..])?).ok()?;
    if hours > 14 || minutes > 59 {
        return None;
    }
    Some(sign * (hours * 60 + minutes))
}

/// `2026-01-10T14:30Z`, `2026-01-10T08:30-06:00` or `2026-01-10T08:30[America/Chicago]` as the
/// instant (seconds since the epoch, UTC) and the zone it is written in.
#[must_use]
pub fn parse_instant(text: &str) -> Option<(i64, ZoneValue)> {
    let (date, rest) = text.split_once('T')?;
    let days = parse_date(date)?;
    let (time, zone) = if let Some(open) = rest.find('[') {
        let name = rest[open + 1..].strip_suffix(']')?;
        (&rest[..open], ZoneValue::Named(Zone::named(name)?))
    } else if let Some(time) = rest.strip_suffix('Z') {
        (time, ZoneValue::UTC)
    } else {
        let split = rest.rfind(['+', '-'])?;
        (
            &rest[..split],
            ZoneValue::Fixed(parse_offset(&rest[split..])?),
        )
    };
    let seconds = parse_time(time)?;
    let local = days * 86_400 + i64::from(seconds);
    Some((zone.instant(local), zone))
}

#[cfg(test)]
mod tests {
    use super::{ZoneValue, parse_date, parse_instant, parse_time};

    #[test]
    fn literals_read() {
        assert_eq!(parse_date("2026-01-10"), Some(20_463));
        assert_eq!(parse_date("2026-02-30"), None);
        assert_eq!(parse_time("14:30"), Some(52_200));
        assert_eq!(parse_time("24:00"), None);
        let utc = parse_instant("2026-01-10T14:30Z").expect("utc");
        let chicago = parse_instant("2026-01-10T08:30[America/Chicago]").expect("chicago");
        let offset = parse_instant("2026-01-10T08:30-06:00").expect("offset");
        assert_eq!(utc.0, chicago.0);
        assert_eq!(utc.0, offset.0);
        assert_eq!(offset.1, ZoneValue::Fixed(-360));
        assert!(parse_instant("2026-01-10T08:30[Mars/Olympus]").is_none());
    }
}
