//! Dates and durations, with the civil calendar computed here: no time zone database, no clock.
//!
//! A [`Date`] is an instant (seconds and nanoseconds since the Unix epoch) plus the UTC offset it
//! is shown in. The server sends each date in the offset of the stop it came from, so
//! `pickupDate.Hour()` is the hour on the dock's clock. Two dates are equal when they are the
//! same instant, whatever their offsets, exactly as Go's `time.Time.Equal`.
//!
//! Calendar fields come from Howard Hinnant's `days_from_civil` and `civil_from_days`, which are
//! exact for the proleptic Gregorian calendar. Dates are kept within years 0000 to 9999, the
//! range RFC 3339 can write; anything that would leave it is an overflow, never a wrap.
//!
//! A duration is a signed count of nanoseconds in an `i64`, as Go's `time.Duration`; arithmetic
//! that leaves that range is an overflow (Go saturates `Time.Sub`; this does not).

use std::cmp::Ordering;
use std::fmt::Write as _;

const NANOS_PER_SECOND: i64 = 1_000_000_000;
const SECONDS_PER_DAY: i64 = 86_400;
/// The first second of year 0000, UTC.
const MIN_SECONDS: i64 = -62_167_219_200;
/// The last second of year 9999, UTC.
const MAX_SECONDS: i64 = 253_402_300_799;
/// Offsets stay within what RFC 3339 can write: less than a day either way.
const MAX_OFFSET: i32 = 86_399;

/// One nanosecond-precise instant and the offset it is shown in.
#[derive(Debug, Clone, Copy)]
pub struct Date {
    seconds: i64,
    nanos: u32,
    offset: i32,
}

/// A date's calendar fields in its own offset.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Civil {
    /// The year, 0 to 9999.
    pub year: i64,
    /// The month, 1 to 12.
    pub month: i64,
    /// The day of the month, 1 to 31.
    pub day: i64,
    /// The hour, 0 to 23.
    pub hour: i64,
    /// The minute, 0 to 59.
    pub minute: i64,
    /// The second, 0 to 59.
    pub second: i64,
    /// The nanoseconds within the second.
    pub nanos: u32,
    /// The day of the week, 0 for Sunday.
    pub weekday: i64,
    /// The day of the year, 1 to 366.
    pub year_day: i64,
}

impl Date {
    /// A date from an instant and an offset, or `None` outside years 0000 to 9999.
    #[must_use]
    pub fn from_instant(seconds: i64, nanos: u32, offset: i32) -> Option<Self> {
        if nanos >= 1_000_000_000 || offset.abs() > MAX_OFFSET {
            return None;
        }
        let local = seconds.checked_add(i64::from(offset))?;
        let in_range = |value: i64| (MIN_SECONDS..=MAX_SECONDS).contains(&value);
        (in_range(seconds) && in_range(local)).then_some(Self {
            seconds,
            nanos,
            offset,
        })
    }

    /// A date from calendar fields read in `offset`, or `None` when a field is out of range.
    #[must_use]
    pub fn from_civil(fields: CivilInput, offset: i32) -> Option<Self> {
        let CivilInput {
            year,
            month,
            day,
            hour,
            minute,
            second,
            nanos,
        } = fields;
        if !(0..=9999).contains(&year)
            || !(1..=12).contains(&month)
            || day < 1
            || day > days_in_month(year, month)
            || !(0..24).contains(&hour)
            || !(0..60).contains(&minute)
            || !(0..60).contains(&second)
        {
            return None;
        }
        let days = days_from_civil(year, month, day);
        let local = days * SECONDS_PER_DAY + hour * 3600 + minute * 60 + second;
        Self::from_instant(local - i64::from(offset), nanos, offset)
    }

    /// Seconds since the Unix epoch.
    #[must_use]
    pub const fn unix(&self) -> i64 {
        self.seconds
    }

    /// The offset from UTC in seconds.
    #[must_use]
    pub const fn offset(&self) -> i32 {
        self.offset
    }

    /// Nanoseconds since the Unix epoch.
    #[must_use]
    pub fn instant_nanos(&self) -> i128 {
        i128::from(self.seconds) * i128::from(NANOS_PER_SECOND) + i128::from(self.nanos)
    }

    /// The calendar fields in the date's own offset.
    #[must_use]
    pub fn civil(&self) -> Civil {
        let local = self.seconds + i64::from(self.offset);
        let days = local.div_euclid(SECONDS_PER_DAY);
        let within = local.rem_euclid(SECONDS_PER_DAY);
        let (year, month, day) = civil_from_days(days);
        Civil {
            year,
            month,
            day,
            hour: within / 3600,
            minute: within % 3600 / 60,
            second: within % 60,
            nanos: self.nanos,
            weekday: (days + 4).rem_euclid(7),
            year_day: days - days_from_civil(year, 1, 1) + 1,
        }
    }

    /// The date moved by `nanos`, keeping its offset, or `None` past year 9999.
    #[must_use]
    pub fn add_nanos(&self, nanos: i64) -> Option<Self> {
        let total = self.instant_nanos().checked_add(i128::from(nanos))?;
        let seconds = i64::try_from(total.div_euclid(i128::from(NANOS_PER_SECOND))).ok()?;
        let nanos = u32::try_from(total.rem_euclid(i128::from(NANOS_PER_SECOND))).ok()?;
        Self::from_instant(seconds, nanos, self.offset)
    }

    /// `self - other` in nanoseconds, or `None` when it does not fit a duration.
    #[must_use]
    pub fn since(&self, other: &Self) -> Option<i64> {
        i64::try_from(self.instant_nanos() - other.instant_nanos()).ok()
    }

    /// Orders two dates by instant.
    #[must_use]
    pub fn cmp_instant(&self, other: &Self) -> Ordering {
        self.instant_nanos().cmp(&other.instant_nanos())
    }

    /// RFC 3339 in the date's own offset, with the fraction only when it is not zero:
    /// `2026-01-02T03:04:05-05:00`. This is the `$date` wire form and reads back exactly.
    #[must_use]
    pub fn rfc3339(&self) -> String {
        let civil = self.civil();
        let mut out = String::with_capacity(35);
        let _ = write!(
            out,
            "{:04}-{:02}-{:02}T{:02}:{:02}:{:02}",
            civil.year, civil.month, civil.day, civil.hour, civil.minute, civil.second
        );
        if civil.nanos != 0 {
            let fraction = format!("{:09}", civil.nanos);
            out.push('.');
            out.push_str(fraction.trim_end_matches('0'));
        }
        if self.offset == 0 {
            out.push('Z');
        } else {
            let sign = if self.offset < 0 { '-' } else { '+' };
            let minutes = self.offset.abs() / 60;
            let _ = write!(out, "{sign}{:02}:{:02}", minutes / 60, minutes % 60);
        }
        out
    }

    /// RFC 3339 in UTC to the second, as Go's `t.UTC().Format(time.RFC3339)`: the form statement
    /// traces show.
    #[must_use]
    pub fn rfc3339_utc_seconds(&self) -> String {
        Self {
            seconds: self.seconds,
            nanos: 0,
            offset: 0,
        }
        .rfc3339()
    }
}

/// Calendar fields to build a [`Date`] from.
#[derive(Debug, Clone, Copy, Default)]
pub struct CivilInput {
    /// The year.
    pub year: i64,
    /// The month, 1 to 12.
    pub month: i64,
    /// The day of the month.
    pub day: i64,
    /// The hour.
    pub hour: i64,
    /// The minute.
    pub minute: i64,
    /// The second.
    pub second: i64,
    /// The nanoseconds.
    pub nanos: u32,
}

fn is_leap(year: i64) -> bool {
    year % 4 == 0 && (year % 100 != 0 || year % 400 == 0)
}

fn days_in_month(year: i64, month: i64) -> i64 {
    match month {
        2 if is_leap(year) => 29,
        2 => 28,
        4 | 6 | 9 | 11 => 30,
        _ => 31,
    }
}

/// Days since 1970-01-01 of a proleptic Gregorian date.
fn days_from_civil(year: i64, month: i64, day: i64) -> i64 {
    let year = if month <= 2 { year - 1 } else { year };
    let era = year.div_euclid(400);
    let year_of_era = year - era * 400;
    let month_index = (month + 9) % 12;
    let day_of_year = (153 * month_index + 2) / 5 + day - 1;
    let day_of_era = year_of_era * 365 + year_of_era / 4 - year_of_era / 100 + day_of_year;
    era * 146_097 + day_of_era - 719_468
}

/// The proleptic Gregorian date of a day count since 1970-01-01.
fn civil_from_days(days: i64) -> (i64, i64, i64) {
    let shifted = days + 719_468;
    let era = shifted.div_euclid(146_097);
    let day_of_era = shifted - era * 146_097;
    let year_of_era =
        (day_of_era - day_of_era / 1460 + day_of_era / 36_524 - day_of_era / 146_096) / 365;
    let day_of_year = day_of_era - (365 * year_of_era + year_of_era / 4 - year_of_era / 100);
    let month_index = (5 * day_of_year + 2) / 153;
    let day = day_of_year - (153 * month_index + 2) / 5 + 1;
    let month = if month_index < 10 {
        month_index + 3
    } else {
        month_index - 9
    };
    let year = year_of_era + era * 400 + i64::from(month <= 2);
    (year, month, day)
}

/// A cursor over date text.
struct Reader<'a> {
    bytes: &'a [u8],
    at: usize,
}

impl Reader<'_> {
    fn digits(&mut self, count: usize) -> Option<i64> {
        let slice = self.bytes.get(self.at..self.at + count)?;
        let mut value = 0i64;
        for byte in slice {
            if !byte.is_ascii_digit() {
                return None;
            }
            value = value * 10 + i64::from(byte - b'0');
        }
        self.at += count;
        Some(value)
    }

    fn byte(&mut self, expected: u8) -> Option<()> {
        (self.bytes.get(self.at) == Some(&expected)).then(|| self.at += 1)
    }

    fn peek(&self) -> Option<u8> {
        self.bytes.get(self.at).copied()
    }

    fn done(&self) -> bool {
        self.at == self.bytes.len()
    }

    /// `.123456789` after the seconds, 1 to 9 digits.
    fn fraction(&mut self) -> Option<u32> {
        if self.peek() != Some(b'.') {
            return Some(0);
        }
        self.at += 1;
        let start = self.at;
        while self.peek().is_some_and(|byte| byte.is_ascii_digit()) {
            self.at += 1;
        }
        let digits = self.at - start;
        if digits == 0 || digits > 9 {
            return None;
        }
        let mut nanos = 0u32;
        for byte in &self.bytes[start..self.at] {
            nanos = nanos * 10 + u32::from(byte - b'0');
        }
        for _ in digits..9 {
            nanos *= 10;
        }
        Some(nanos)
    }

    /// `Z` or `±hh:mm`.
    fn offset(&mut self) -> Option<i32> {
        match self.peek()? {
            b'Z' => {
                self.at += 1;
                Some(0)
            }
            sign @ (b'+' | b'-') => {
                self.at += 1;
                let hours = self.digits(2)?;
                self.byte(b':')?;
                let minutes = self.digits(2)?;
                if hours > 23 || minutes > 59 {
                    return None;
                }
                let seconds = i32::try_from(hours * 3600 + minutes * 60).ok()?;
                Some(if sign == b'-' { -seconds } else { seconds })
            }
            _ => None,
        }
    }
}

fn read_day(reader: &mut Reader<'_>) -> Option<CivilInput> {
    let year = reader.digits(4)?;
    reader.byte(b'-')?;
    let month = reader.digits(2)?;
    reader.byte(b'-')?;
    let day = reader.digits(2)?;
    Some(CivilInput {
        year,
        month,
        day,
        ..CivilInput::default()
    })
}

/// Reads strict RFC 3339 with an offset: the `$date` wire form.
#[must_use]
pub fn parse_rfc3339(text: &str) -> Option<Date> {
    let mut reader = Reader {
        bytes: text.as_bytes(),
        at: 0,
    };
    let mut fields = read_day(&mut reader)?;
    reader.byte(b'T')?;
    fields.hour = reader.digits(2)?;
    reader.byte(b':')?;
    fields.minute = reader.digits(2)?;
    reader.byte(b':')?;
    fields.second = reader.digits(2)?;
    fields.nanos = reader.fraction()?;
    let offset = reader.offset()?;
    reader.done().then_some(())?;
    Date::from_civil(fields, offset)
}

/// Reads the forms `date(text)` accepts: RFC 3339, `2006-01-02T15:04:05`, `2006-01-02T15:04`,
/// `2006-01-02 15:04` and `2006-01-02`. A form without an offset is UTC. Seconds may carry a
/// fraction, as Go's parser allows.
#[must_use]
pub fn parse_date(text: &str) -> Option<Date> {
    if let Some(date) = parse_rfc3339(text) {
        return Some(date);
    }
    let mut reader = Reader {
        bytes: text.as_bytes(),
        at: 0,
    };
    let mut fields = read_day(&mut reader)?;
    if reader.done() {
        return Date::from_civil(fields, 0);
    }
    let separator = reader.peek()?;
    if separator != b'T' && separator != b' ' {
        return None;
    }
    reader.at += 1;
    fields.hour = reader.digits(2)?;
    reader.byte(b':')?;
    fields.minute = reader.digits(2)?;
    if reader.done() {
        return Date::from_civil(fields, 0);
    }
    if separator != b'T' {
        return None;
    }
    reader.byte(b':')?;
    fields.second = reader.digits(2)?;
    fields.nanos = reader.fraction()?;
    reader.done().then_some(())?;
    Date::from_civil(fields, 0)
}

/// Why duration text did not read, with Go's wording.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DurationError {
    /// Not duration syntax at all.
    Invalid,
    /// A number without a unit.
    MissingUnit,
    /// A unit that is not `ns`, `us`, `µs`, `ms`, `s`, `m` or `h`.
    UnknownUnit,
    /// Past what a duration holds.
    Overflow,
}

const UNITS: &[(&str, i128)] = &[
    ("ns", 1),
    ("us", 1_000),
    ("\u{b5}s", 1_000),
    ("\u{3bc}s", 1_000),
    ("ms", 1_000_000),
    ("s", 1_000_000_000),
    ("m", 60_000_000_000),
    ("h", 3_600_000_000_000),
];

/// Reads Go duration syntax (`1h30m`, `-1.5s`, `300ms`) exactly: the fraction is applied in
/// integer arithmetic and truncated to the nanosecond, where Go goes through `float64`.
pub fn parse_duration(text: &str) -> Result<i64, DurationError> {
    let mut rest = text;
    let negative = if let Some(stripped) = rest.strip_prefix('-') {
        rest = stripped;
        true
    } else {
        if let Some(stripped) = rest.strip_prefix('+') {
            rest = stripped;
        }
        false
    };
    if rest == "0" {
        return Ok(0);
    }
    if rest.is_empty() {
        return Err(DurationError::Invalid);
    }
    let mut total: i128 = 0;
    while !rest.is_empty() {
        let digits_end = rest
            .find(|character: char| !character.is_ascii_digit())
            .unwrap_or(rest.len());
        let whole_text = &rest[..digits_end];
        rest = &rest[digits_end..];
        let mut fraction_text = "";
        if let Some(stripped) = rest.strip_prefix('.') {
            let end = stripped
                .find(|character: char| !character.is_ascii_digit())
                .unwrap_or(stripped.len());
            fraction_text = &stripped[..end];
            rest = &stripped[end..];
        }
        if whole_text.is_empty() && fraction_text.is_empty() {
            return Err(DurationError::Invalid);
        }
        let unit_end = rest
            .find(|character: char| character == '.' || character.is_ascii_digit())
            .unwrap_or(rest.len());
        let unit = &rest[..unit_end];
        rest = &rest[unit_end..];
        if unit.is_empty() {
            return Err(DurationError::MissingUnit);
        }
        let scale = UNITS
            .iter()
            .find(|(name, _)| *name == unit)
            .map(|(_, scale)| *scale)
            .ok_or(DurationError::UnknownUnit)?;
        let mut whole: i128 = 0;
        for byte in whole_text.bytes() {
            whole = whole
                .checked_mul(10)
                .and_then(|value| value.checked_add(i128::from(byte - b'0')))
                .filter(|value| *value <= i128::from(i64::MAX) + 1)
                .ok_or(DurationError::Overflow)?;
        }
        let mut part = whole.checked_mul(scale).ok_or(DurationError::Overflow)?;
        if !fraction_text.is_empty() {
            let mut numerator: i128 = 0;
            let mut denominator: i128 = 1;
            for byte in fraction_text.bytes().take(18) {
                numerator = numerator * 10 + i128::from(byte - b'0');
                denominator *= 10;
            }
            part += numerator * scale / denominator;
        }
        total = total.checked_add(part).ok_or(DurationError::Overflow)?;
        if total > i128::from(i64::MAX) + 1 {
            return Err(DurationError::Overflow);
        }
    }
    let signed = if negative { -total } else { total };
    i64::try_from(signed).map_err(|_| DurationError::Overflow)
}

/// Formats a duration as Go's `Duration.String`: `1h30m0s`, `1.5s`, `250ms`, `0s`.
#[must_use]
pub fn format_duration(nanos: i64) -> String {
    let negative = nanos < 0;
    let magnitude = nanos.unsigned_abs();
    let mut out = String::new();
    if negative {
        out.push('-');
    }
    if magnitude < 1_000_000_000 {
        let (scale, unit) = match magnitude {
            0 => return String::from("0s"),
            1..=999 => (1u64, "ns"),
            1_000..=999_999 => (1_000, "\u{b5}s"),
            _ => (1_000_000, "ms"),
        };
        push_fraction(&mut out, magnitude, scale);
        out.push_str(unit);
        return out;
    }
    let hours = magnitude / 3_600_000_000_000;
    let minutes = magnitude / 60_000_000_000 % 60;
    let seconds = magnitude % 60_000_000_000;
    if hours > 0 {
        let _ = write!(out, "{hours}h");
    }
    if hours > 0 || minutes > 0 {
        let _ = write!(out, "{minutes}m");
    }
    push_fraction(&mut out, seconds, 1_000_000_000);
    out.push('s');
    out
}

fn push_fraction(out: &mut String, value: u64, scale: u64) {
    let _ = write!(out, "{}", value / scale);
    let remainder = value % scale;
    if remainder != 0 {
        let width = scale.ilog10() as usize;
        let digits = format!("{remainder:0width$}");
        out.push('.');
        out.push_str(digits.trim_end_matches('0'));
    }
}

#[cfg(test)]
mod tests {
    use super::{
        CivilInput, Date, DurationError, format_duration, parse_date, parse_duration, parse_rfc3339,
    };

    #[test]
    fn civil_fields_round_trip_through_day_counts() {
        for days in [-719_468i64, -1, 0, 1, 11_016, 20_000, 2_932_896] {
            let date = Date::from_instant(days * 86_400, 0, 0).expect("in range");
            let civil = date.civil();
            let back = Date::from_civil(
                CivilInput {
                    year: civil.year,
                    month: civil.month,
                    day: civil.day,
                    ..CivilInput::default()
                },
                0,
            )
            .expect("valid");
            assert_eq!(back.unix(), days * 86_400);
        }
    }

    #[test]
    fn reads_and_writes_rfc3339_with_its_offset() {
        let date = parse_rfc3339("2026-01-02T03:04:05-05:00").expect("parses");
        assert_eq!(date.rfc3339(), "2026-01-02T03:04:05-05:00");
        assert_eq!(date.civil().hour, 3);
        assert_eq!(date.rfc3339_utc_seconds(), "2026-01-02T08:04:05Z");
        let fraction = parse_rfc3339("2026-01-02T03:04:05.120Z").expect("parses");
        assert_eq!(fraction.rfc3339(), "2026-01-02T03:04:05.12Z");
    }

    #[test]
    fn reads_the_forms_date_accepts() {
        for text in [
            "2026-06-05T20:30:00Z",
            "2026-06-05T20:30:00",
            "2026-06-05T20:30",
            "2026-06-05 20:30",
            "2026-06-05",
        ] {
            let date = parse_date(text).expect(text);
            assert_eq!(date.civil().year, 2026, "{text}");
            assert_eq!(date.civil().month, 6, "{text}");
            assert_eq!(date.offset(), 0, "{text}");
        }
        for text in [
            "2026-02-30",
            "2026-13-01",
            "2026-06-05 20:30:00",
            "05/06/2026",
            "2026-06-05T24:00",
            "",
        ] {
            assert!(parse_date(text).is_none(), "{text}");
        }
    }

    #[test]
    fn weekday_and_year_day() {
        let civil = parse_date("2025-03-01").expect("parses").civil();
        assert_eq!(civil.weekday, 6);
        assert_eq!(civil.year_day, 60);
        let leap = parse_date("2024-12-31").expect("parses").civil();
        assert_eq!(leap.year_day, 366);
        assert_eq!(parse_date("1970-01-01").expect("parses").civil().weekday, 4);
    }

    #[test]
    fn durations_read_go_syntax() {
        assert_eq!(parse_duration("1h30m"), Ok(5_400_000_000_000));
        assert_eq!(parse_duration("-1.5s"), Ok(-1_500_000_000));
        assert_eq!(parse_duration("300ms"), Ok(300_000_000));
        assert_eq!(parse_duration("1\u{b5}s"), Ok(1_000));
        assert_eq!(parse_duration("0"), Ok(0));
        assert_eq!(parse_duration(".5m"), Ok(30_000_000_000));
        assert_eq!(parse_duration("1"), Err(DurationError::MissingUnit));
        assert_eq!(parse_duration("1d"), Err(DurationError::UnknownUnit));
        assert_eq!(parse_duration(""), Err(DurationError::Invalid));
        assert_eq!(parse_duration("h"), Err(DurationError::Invalid));
        assert_eq!(parse_duration("3000000h"), Err(DurationError::Overflow));
    }

    #[test]
    fn durations_print_as_go_does() {
        assert_eq!(format_duration(5_400_000_000_000), "1h30m0s");
        assert_eq!(format_duration(1_500_000_000), "1.5s");
        assert_eq!(format_duration(250_000_000), "250ms");
        assert_eq!(format_duration(0), "0s");
        assert_eq!(format_duration(-1_000), "-1\u{b5}s");
        assert_eq!(format_duration(61_000_000_000), "1m1s");
        assert_eq!(format_duration(i64::MIN), "-2562047h47m16.854775808s");
    }

    #[test]
    fn arithmetic_stays_in_range() {
        let date = parse_date("9999-12-31").expect("parses");
        assert!(date.add_nanos(86_400_000_000_000).is_none());
        let start = parse_date("2026-01-01").expect("parses");
        let end = start.add_nanos(3_600_000_000_000).expect("fits");
        assert_eq!(end.since(&start), Some(3_600_000_000_000));
    }
}
