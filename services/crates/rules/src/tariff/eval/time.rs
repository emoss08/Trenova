//! Dates, date-times and times of day, with the civil calendar computed here: no time zone
//! database, no clock.
//!
//! A [`Date`] is a calendar day. A [`DateTime`] is an instant (whole seconds since the Unix epoch,
//! UTC) plus the UTC offset it is shown in, so `stop.arrival.hour` is the hour on the dock's
//! clock. Two date-times are equal when they are the same instant, whatever their offsets.
//! Date-times are exact to the second: Tariff's literals and the wire form carry no fractions.
//! A [`Time`] is a wall-clock time of day, seconds since midnight, with no date and no offset:
//! `stop.arrival.time` is the time the dock's clock shows.
//!
//! Calendar fields come from Howard Hinnant's `days_from_civil` and `civil_from_days`, exact for
//! the proleptic Gregorian calendar. Values stay within years 0000 to 9999, the range ISO 8601
//! writes with four digits; anything that would leave it is an overflow, never a wrap.

use std::cmp::Ordering;
use std::fmt::Write as _;

const SECONDS_PER_DAY: i64 = 86_400;
/// Days from 1970-01-01 to 0000-01-01.
const MIN_DAYS: i64 = -719_528;
/// Days from 1970-01-01 to 9999-12-31.
const MAX_DAYS: i64 = 2_932_896;
/// Offsets stay within ±18:00, as ISO 8601 and the lexer allow.
const MAX_OFFSET_MINUTES: i32 = 18 * 60;

/// A day of the week, Monday first, as the `Weekday` schema enum names it.
pub const WEEKDAYS: [&str; 7] = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];

/// A calendar date.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct Date {
    days: i64,
}

/// An instant to the second and the UTC offset it is shown in.
#[derive(Debug, Clone, Copy)]
pub struct DateTime {
    seconds: i64,
    offset_minutes: i32,
}

/// A time of day: seconds since midnight, 0 to 86 399.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct Time {
    seconds: i64,
}

/// A date's calendar fields.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Civil {
    /// The year, 0 to 9999.
    pub year: i64,
    /// The month, 1 to 12.
    pub month: i64,
    /// The day of the month, 1 to 31.
    pub day: i64,
}

impl Date {
    /// The date with these calendar fields, or `None` when they name no day in years 0000 to
    /// 9999.
    #[must_use]
    pub fn from_civil(year: i64, month: i64, day: i64) -> Option<Self> {
        if !(0..=9999).contains(&year)
            || !(1..=12).contains(&month)
            || day < 1
            || day > days_in_month(year, month)
        {
            return None;
        }
        Some(Self {
            days: days_from_civil(year, month, day),
        })
    }

    fn from_days(days: i64) -> Option<Self> {
        (MIN_DAYS..=MAX_DAYS)
            .contains(&days)
            .then_some(Self { days })
    }

    /// Days since 1970-01-01.
    #[must_use]
    pub const fn days(self) -> i64 {
        self.days
    }

    /// The calendar fields.
    #[must_use]
    pub fn civil(self) -> Civil {
        let (year, month, day) = civil_from_days(self.days);
        Civil { year, month, day }
    }

    /// The day of the week, 0 for Monday.
    #[must_use]
    pub fn weekday_index(self) -> usize {
        usize::try_from((self.days + 3).rem_euclid(7)).unwrap_or(0)
    }

    /// The weekday's variant name: `Mon` … `Sun`.
    #[must_use]
    pub fn weekday(self) -> &'static str {
        WEEKDAYS.get(self.weekday_index()).copied().unwrap_or("Mon")
    }

    /// Whether the date is a Saturday or a Sunday.
    #[must_use]
    pub fn weekend(self) -> bool {
        self.weekday_index() >= 5
    }

    /// The date `count` days after this one (before it when negative), or `None` outside years
    /// 0000 to 9999.
    #[must_use]
    pub fn add_days(self, count: i64) -> Option<Self> {
        Self::from_days(self.days.checked_add(count)?)
    }

    /// ISO 8601: `2026-10-22`.
    #[must_use]
    pub fn iso(self) -> String {
        let civil = self.civil();
        format!("{:04}-{:02}-{:02}", civil.year, civil.month, civil.day)
    }

    /// Reads `YYYY-MM-DD`.
    #[must_use]
    pub fn parse(text: &str) -> Option<Self> {
        let mut reader = Reader::new(text);
        let date = reader.date()?;
        reader.done().then_some(date)
    }
}

impl DateTime {
    /// The date-time with these local fields read in `offset_minutes`, or `None` when a field
    /// is out of range.
    #[must_use]
    pub fn from_local(
        date: Date,
        hour: i64,
        minute: i64,
        second: i64,
        offset: i32,
    ) -> Option<Self> {
        if !(0..24).contains(&hour)
            || !(0..60).contains(&minute)
            || !(0..60).contains(&second)
            || offset.abs() > MAX_OFFSET_MINUTES
        {
            return None;
        }
        let local = date.days * SECONDS_PER_DAY + hour * 3600 + minute * 60 + second;
        Self::from_instant(local - i64::from(offset) * 60, offset)
    }

    /// The date-time at `seconds` since the epoch shown in `offset_minutes`, or `None` when its
    /// local date leaves years 0000 to 9999.
    #[must_use]
    pub fn from_instant(seconds: i64, offset_minutes: i32) -> Option<Self> {
        if offset_minutes.abs() > MAX_OFFSET_MINUTES {
            return None;
        }
        let candidate = Self {
            seconds,
            offset_minutes,
        };
        let local = seconds.checked_add(i64::from(offset_minutes) * 60)?;
        Date::from_days(local.div_euclid(SECONDS_PER_DAY))?;
        Date::from_days(seconds.div_euclid(SECONDS_PER_DAY))?;
        Some(candidate)
    }

    /// Seconds since the Unix epoch.
    #[must_use]
    pub const fn unix(self) -> i64 {
        self.seconds
    }

    /// The UTC offset in minutes.
    #[must_use]
    pub const fn offset_minutes(self) -> i32 {
        self.offset_minutes
    }

    fn local_seconds(self) -> i64 {
        self.seconds + i64::from(self.offset_minutes) * 60
    }

    /// The local calendar date.
    #[must_use]
    pub fn date(self) -> Date {
        Date {
            days: self.local_seconds().div_euclid(SECONDS_PER_DAY),
        }
    }

    /// Seconds since local midnight.
    #[must_use]
    pub fn second_of_day(self) -> i64 {
        self.local_seconds().rem_euclid(SECONDS_PER_DAY)
    }

    /// The local hour, 0 to 23.
    #[must_use]
    pub fn hour(self) -> i64 {
        self.second_of_day() / 3600
    }

    /// The local minute, 0 to 59.
    #[must_use]
    pub fn minute(self) -> i64 {
        self.second_of_day() % 3600 / 60
    }

    /// The date-time moved by `seconds`, keeping its offset.
    #[must_use]
    pub fn add_seconds(self, seconds: i64) -> Option<Self> {
        Self::from_instant(self.seconds.checked_add(seconds)?, self.offset_minutes)
    }

    /// Orders two date-times by instant.
    #[must_use]
    pub fn cmp_instant(self, other: Self) -> Ordering {
        self.seconds.cmp(&other.seconds)
    }

    /// ISO 8601 with seconds and the offset (`Z` for UTC): `2026-10-22T08:00:00-05:00`.
    #[must_use]
    pub fn iso(self) -> String {
        let within = self.second_of_day();
        let mut out = self.date().iso();
        let _ = write!(
            out,
            "T{:02}:{:02}:{:02}",
            within / 3600,
            within % 3600 / 60,
            within % 60
        );
        if self.offset_minutes == 0 {
            out.push('Z');
        } else {
            let sign = if self.offset_minutes < 0 { '-' } else { '+' };
            let minutes = self.offset_minutes.abs();
            let _ = write!(out, "{sign}{:02}:{:02}", minutes / 60, minutes % 60);
        }
        out
    }

    /// Reads `YYYY-MM-DDTHH:MM[:SS[.000…]](Z|±HH:MM)`. A fraction of a second must be zero.
    #[must_use]
    pub fn parse(text: &str) -> Option<Self> {
        let mut reader = Reader::new(text);
        let date = reader.date()?;
        if !matches!(reader.peek(), Some(b'T' | b't' | b' ')) {
            return None;
        }
        reader.at += 1;
        let hour = reader.digits(2)?;
        reader.byte(b':')?;
        let minute = reader.digits(2)?;
        let second = if reader.peek() == Some(b':') {
            reader.at += 1;
            let second = reader.digits(2)?;
            reader.zero_fraction()?;
            second
        } else {
            0
        };
        let offset = reader.offset()?;
        if !reader.done() {
            return None;
        }
        Self::from_local(date, hour, minute, second, offset)
    }
}

impl Time {
    /// The time `seconds` after midnight, or `None` outside one day.
    #[must_use]
    pub fn from_seconds(seconds: i64) -> Option<Self> {
        (0..SECONDS_PER_DAY)
            .contains(&seconds)
            .then_some(Self { seconds })
    }

    /// Seconds since midnight.
    #[must_use]
    pub const fn seconds(self) -> i64 {
        self.seconds
    }

    /// The hour, 0 to 23.
    #[must_use]
    pub const fn hour(self) -> i64 {
        self.seconds / 3600
    }

    /// The minute, 0 to 59.
    #[must_use]
    pub const fn minute(self) -> i64 {
        self.seconds % 3600 / 60
    }

    /// The second, 0 to 59.
    #[must_use]
    pub const fn second(self) -> i64 {
        self.seconds % 60
    }

    /// The wire form, always with seconds: `17:30:00`.
    #[must_use]
    pub fn iso(self) -> String {
        format!(
            "{:02}:{:02}:{:02}",
            self.hour(),
            self.minute(),
            self.second()
        )
    }

    /// As interpolation writes it: `17:30`, or `07:30:15` when the seconds aren't zero.
    #[must_use]
    pub fn display(self) -> String {
        if self.second() == 0 {
            format!("{:02}:{:02}", self.hour(), self.minute())
        } else {
            self.iso()
        }
    }

    /// Reads `HH:MM` or `HH:MM:SS`.
    #[must_use]
    pub fn parse(text: &str) -> Option<Self> {
        let mut reader = Reader::new(text);
        let hour = reader.digits(2)?;
        reader.byte(b':')?;
        let minute = reader.digits(2)?;
        let second = if reader.peek() == Some(b':') {
            reader.at += 1;
            reader.digits(2)?
        } else {
            0
        };
        if !reader.done() || hour > 23 || minute > 59 || second > 59 {
            return None;
        }
        Self::from_seconds(hour * 3600 + minute * 60 + second)
    }
}

impl DateTime {
    /// The time of day the date-time shows in its own offset.
    #[must_use]
    pub fn time(self) -> Time {
        Time {
            seconds: self.second_of_day(),
        }
    }
}

impl PartialEq for DateTime {
    fn eq(&self, other: &Self) -> bool {
        self.seconds == other.seconds
    }
}

impl Eq for DateTime {}

/// Whether `year` is a leap year.
#[must_use]
pub const fn is_leap(year: i64) -> bool {
    year % 4 == 0 && (year % 100 != 0 || year % 400 == 0)
}

/// The days in a month.
#[must_use]
pub const fn days_in_month(year: i64, month: i64) -> i64 {
    match month {
        2 if is_leap(year) => 29,
        2 => 28,
        4 | 6 | 9 | 11 => 30,
        _ => 31,
    }
}

/// Days since 1970-01-01 of a proleptic Gregorian date.
const fn days_from_civil(year: i64, month: i64, day: i64) -> i64 {
    let year = if month <= 2 { year - 1 } else { year };
    let era = year.div_euclid(400);
    let year_of_era = year - era * 400;
    let month_index = (month + 9) % 12;
    let day_of_year = (153 * month_index + 2) / 5 + day - 1;
    let day_of_era = year_of_era * 365 + year_of_era / 4 - year_of_era / 100 + day_of_year;
    era * 146_097 + day_of_era - 719_468
}

/// The proleptic Gregorian date of a day count since 1970-01-01.
const fn civil_from_days(days: i64) -> (i64, i64, i64) {
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
    let year = year_of_era + era * 400 + if month <= 2 { 1 } else { 0 };
    (year, month, day)
}

/// A cursor over date text.
struct Reader<'a> {
    bytes: &'a [u8],
    at: usize,
}

impl<'a> Reader<'a> {
    const fn new(text: &'a str) -> Self {
        Self {
            bytes: text.as_bytes(),
            at: 0,
        }
    }

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

    const fn done(&self) -> bool {
        self.at == self.bytes.len()
    }

    fn date(&mut self) -> Option<Date> {
        let year = self.digits(4)?;
        self.byte(b'-')?;
        let month = self.digits(2)?;
        self.byte(b'-')?;
        let day = self.digits(2)?;
        Date::from_civil(year, month, day)
    }

    /// `.000` after the seconds: accepted only when every digit is zero.
    fn zero_fraction(&mut self) -> Option<()> {
        if self.peek() != Some(b'.') {
            return Some(());
        }
        self.at += 1;
        let start = self.at;
        while self.peek() == Some(b'0') {
            self.at += 1;
        }
        (self.at > start && !self.peek().is_some_and(|byte| byte.is_ascii_digit())).then_some(())
    }

    /// `Z` or `±hh:mm`.
    fn offset(&mut self) -> Option<i32> {
        match self.peek()? {
            b'Z' | b'z' => {
                self.at += 1;
                Some(0)
            }
            sign @ (b'+' | b'-') => {
                self.at += 1;
                let hours = self.digits(2)?;
                self.byte(b':')?;
                let minutes = self.digits(2)?;
                if minutes > 59 {
                    return None;
                }
                let total = i32::try_from(hours * 60 + minutes).ok()?;
                Some(if sign == b'-' { -total } else { total })
            }
            _ => None,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::{Date, DateTime, Time};

    #[test]
    fn times_read_and_write_the_clock() {
        let time = Time::parse("17:30").expect("a time");
        assert_eq!(time.iso(), "17:30:00");
        assert_eq!(time.display(), "17:30");
        assert_eq!(
            Time::parse("07:30:15").map(Time::display).as_deref(),
            Some("07:30:15")
        );
        assert_eq!(Time::parse("24:00"), None);
        assert_eq!(Time::parse("7:30"), None);
        assert_eq!(Time::parse("17:30:00Z"), None);
        let at = DateTime::parse("2026-10-22T23:45:00-05:00").expect("a date-time");
        assert_eq!(at.time().display(), "23:45");
        assert_eq!(Time::from_seconds(86_400), None);
    }

    #[test]
    fn dates_read_and_write_iso() {
        let date = Date::parse("2026-10-22").expect("a date");
        assert_eq!(date.iso(), "2026-10-22");
        assert_eq!(date.weekday(), "Thu");
        assert!(!date.weekend());
        assert_eq!(Date::parse("2026-02-29"), None);
        assert_eq!(
            Date::parse("2024-02-29").map(Date::iso).as_deref(),
            Some("2024-02-29")
        );
        assert_eq!(
            date.add_days(10).map(Date::iso).as_deref(),
            Some("2026-11-01")
        );
        assert_eq!(Date::parse("9999-12-31").and_then(|d| d.add_days(1)), None);
        assert_eq!(
            Date::parse("0000-01-01").map(Date::iso).as_deref(),
            Some("0000-01-01")
        );
    }

    #[test]
    fn date_times_keep_their_offset_and_compare_by_instant() {
        let local = DateTime::parse("2026-10-22T08:00:00-05:00").expect("a date-time");
        let utc = DateTime::parse("2026-10-22T13:00Z").expect("a date-time");
        assert_eq!(local, utc);
        assert_eq!(local.hour(), 8);
        assert_eq!(utc.hour(), 13);
        assert_eq!(local.iso(), "2026-10-22T08:00:00-05:00");
        assert_eq!(utc.iso(), "2026-10-22T13:00:00Z");
        assert_eq!(
            DateTime::parse("2026-10-22T23:30:00-05:00").map(|d| d.date().iso()),
            Some(String::from("2026-10-22"))
        );
        assert!(DateTime::parse("2026-10-22T08:00:00.000Z").is_some());
        assert!(DateTime::parse("2026-10-22T08:00:00.5Z").is_none());
        assert!(DateTime::parse("2026-10-22T24:00Z").is_none());
        assert!(DateTime::parse("2026-10-22T08:00+19:00").is_none());
        assert!(DateTime::parse("2026-10-22").is_none());
    }
}
