//! Time zones, without a zone database.
//!
//! The module has no clock and no files, so it carries the zones freight in North America is
//! scheduled in, with today's daylight saving rules (the second Sunday of March to the first
//! Sunday of November at 02:00 local, in force since 2007 in the United States and Canada),
//! and the X12 time codes (element 623) for each. Dates before 2007 are read with the same
//! rules. A zone outside the table is refused by the checker.

/// A zone of the table.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Zone {
    /// Its IANA name.
    pub name: &'static str,
    /// Its standard offset from UTC, in minutes.
    pub standard: i32,
    /// Whether it keeps daylight saving time.
    pub daylight: bool,
    /// Its X12 time code, then the codes for standard and daylight time.
    pub codes: (&'static str, &'static str, &'static str),
}

const fn zone(
    name: &'static str,
    standard: i32,
    daylight: bool,
    codes: (&'static str, &'static str, &'static str),
) -> Zone {
    Zone {
        name,
        standard,
        daylight,
        codes,
    }
}

const EASTERN: (&str, &str, &str) = ("ET", "ES", "ED");
const CENTRAL: (&str, &str, &str) = ("CT", "CS", "CD");
const MOUNTAIN: (&str, &str, &str) = ("MT", "MS", "MD");
const PACIFIC: (&str, &str, &str) = ("PT", "PS", "PD");
const ALASKA: (&str, &str, &str) = ("AT", "AS", "AD");
const HAWAII: (&str, &str, &str) = ("HT", "HS", "HD");
const ATLANTIC: (&str, &str, &str) = ("TT", "TS", "TD");
const NEWFOUNDLAND: (&str, &str, &str) = ("NT", "NS", "ND");
const UNIVERSAL: (&str, &str, &str) = ("UT", "UT", "UT");

/// Every zone Lading knows.
pub const ZONES: &[Zone] = &[
    zone("UTC", 0, false, UNIVERSAL),
    zone("Etc/UTC", 0, false, UNIVERSAL),
    zone("America/New_York", -300, true, EASTERN),
    zone("America/Detroit", -300, true, EASTERN),
    zone("America/Indiana/Indianapolis", -300, true, EASTERN),
    zone("America/Kentucky/Louisville", -300, true, EASTERN),
    zone("America/Toronto", -300, true, EASTERN),
    zone("America/Cancun", -300, false, EASTERN),
    zone("America/Chicago", -360, true, CENTRAL),
    zone("America/Winnipeg", -360, true, CENTRAL),
    zone("America/Regina", -360, false, CENTRAL),
    zone("America/Mexico_City", -360, false, CENTRAL),
    zone("America/Monterrey", -360, false, CENTRAL),
    zone("America/Matamoros", -360, true, CENTRAL),
    zone("America/Denver", -420, true, MOUNTAIN),
    zone("America/Boise", -420, true, MOUNTAIN),
    zone("America/Edmonton", -420, true, MOUNTAIN),
    zone("America/Phoenix", -420, false, MOUNTAIN),
    zone("America/Ciudad_Juarez", -420, true, MOUNTAIN),
    zone("America/Los_Angeles", -480, true, PACIFIC),
    zone("America/Vancouver", -480, true, PACIFIC),
    zone("America/Tijuana", -480, true, PACIFIC),
    zone("America/Anchorage", -540, true, ALASKA),
    zone("Pacific/Honolulu", -600, false, HAWAII),
    zone("America/Halifax", -240, true, ATLANTIC),
    zone("America/Puerto_Rico", -240, false, ATLANTIC),
    zone("America/St_Johns", -210, true, NEWFOUNDLAND),
];

impl Zone {
    /// A zone by IANA name.
    #[must_use]
    pub fn named(name: &str) -> Option<&'static Self> {
        ZONES.iter().find(|zone| zone.name == name)
    }

    /// The zone an X12 time code stands for: the first zone of the table with that code.
    #[must_use]
    pub fn from_code(code: &str) -> Option<&'static Self> {
        ZONES.iter().find(|zone| {
            let (general, standard, daylight) = zone.codes;
            code == general || code == standard || code == daylight
        })
    }

    /// Its offset from UTC, in minutes, at an instant (seconds since the epoch, UTC).
    #[must_use]
    pub fn offset_at(&self, instant: i64) -> i32 {
        if !self.daylight {
            return self.standard;
        }
        let local_standard = instant + i64::from(self.standard) * 60;
        let year = civil_from_days(local_standard.div_euclid(86_400)).0;
        let (starts, ends) = daylight_bounds(year);
        let starts_utc = starts - i64::from(self.standard) * 60;
        let ends_utc = ends - i64::from(self.standard + 60) * 60;
        if instant >= starts_utc && instant < ends_utc {
            self.standard + 60
        } else {
            self.standard
        }
    }

    /// Whether daylight time is in force at an instant.
    #[must_use]
    pub fn is_daylight(&self, instant: i64) -> bool {
        self.offset_at(instant) != self.standard
    }

    /// The instant a local date and time stand for. A local time skipped when clocks go forward
    /// is read in standard time; one repeated when they go back is the first.
    #[must_use]
    pub fn instant(&self, local_seconds: i64) -> i64 {
        let as_standard = local_seconds - i64::from(self.standard) * 60;
        if !self.daylight {
            return as_standard;
        }
        let as_daylight = local_seconds - i64::from(self.standard + 60) * 60;
        if self.offset_at(as_daylight) == self.standard + 60 {
            as_daylight
        } else {
            as_standard
        }
    }

    /// The X12 code for the zone at an instant: general (`CT`) or specific (`CS`, `CD`).
    #[must_use]
    pub fn code(&self, instant: i64, specific: bool) -> &'static str {
        let (general, standard, daylight) = self.codes;
        if !specific {
            general
        } else if self.is_daylight(instant) {
            daylight
        } else {
            standard
        }
    }
}

/// The local-standard-time seconds at which daylight time starts and ends in `year`.
fn daylight_bounds(year: i64) -> (i64, i64) {
    let march_first = days_from_civil(year, 3, 1);
    let march_weekday = weekday(march_first);
    let second_sunday = march_first + (7 - march_weekday) % 7 + 7;
    let november_first = days_from_civil(year, 11, 1);
    let november_weekday = weekday(november_first);
    let first_sunday = november_first + (7 - november_weekday) % 7;
    (
        second_sunday * 86_400 + 2 * 3600,
        first_sunday * 86_400 + 2 * 3600,
    )
}

/// The weekday of a day count, Sunday being 0.
#[must_use]
pub fn weekday(days: i64) -> i64 {
    (days + 4).rem_euclid(7)
}

/// Days since 1970-01-01 of a civil date.
#[must_use]
pub fn days_from_civil(year: i64, month: u32, day: u32) -> i64 {
    let year = if month <= 2 { year - 1 } else { year };
    let era = year.div_euclid(400);
    let year_of_era = year - era * 400;
    let month = i64::from(month);
    let day_of_year =
        (153 * (if month > 2 { month - 3 } else { month + 9 }) + 2) / 5 + i64::from(day) - 1;
    let day_of_era = year_of_era * 365 + year_of_era / 4 - year_of_era / 100 + day_of_year;
    era * 146_097 + day_of_era - 719_468
}

/// The civil date of a day count since 1970-01-01.
#[must_use]
pub fn civil_from_days(days: i64) -> (i64, u32, u32) {
    let days = days + 719_468;
    let era = days.div_euclid(146_097);
    let day_of_era = days - era * 146_097;
    let year_of_era =
        (day_of_era - day_of_era / 1460 + day_of_era / 36_524 - day_of_era / 146_096) / 365;
    let year = year_of_era + era * 400;
    let day_of_year = day_of_era - (365 * year_of_era + year_of_era / 4 - year_of_era / 100);
    let month_index = (5 * day_of_year + 2) / 153;
    let day = u32::try_from(day_of_year - (153 * month_index + 2) / 5 + 1).unwrap_or(1);
    let month = u32::try_from(if month_index < 10 {
        month_index + 3
    } else {
        month_index - 9
    })
    .unwrap_or(1);
    (if month <= 2 { year + 1 } else { year }, month, day)
}

/// Whether a civil date exists.
#[must_use]
pub fn valid_date(year: i64, month: u32, day: u32) -> bool {
    if !(1..=12).contains(&month) || day == 0 {
        return false;
    }
    let leap = (year % 4 == 0 && year % 100 != 0) || year % 400 == 0;
    let days = match month {
        2 if leap => 29,
        2 => 28,
        4 | 6 | 9 | 11 => 30,
        _ => 31,
    };
    day <= days
}

#[cfg(test)]
mod tests {
    use super::{Zone, civil_from_days, days_from_civil};

    #[test]
    fn civil_dates_round_trip() {
        for days in [-1, 0, 1, 19_000, 20_463, 47_000] {
            let (year, month, day) = civil_from_days(days);
            assert_eq!(days_from_civil(year, month, day), days);
        }
        assert_eq!(days_from_civil(2026, 1, 10), 20_463);
    }

    #[test]
    fn chicago_keeps_daylight_time_in_summer_only() {
        let chicago = Zone::named("America/Chicago").expect("Chicago");
        let january = days_from_civil(2026, 1, 10) * 86_400 + 14 * 3600 + 30 * 60;
        assert_eq!(chicago.offset_at(january), -360);
        let july = days_from_civil(2026, 7, 10) * 86_400;
        assert_eq!(chicago.offset_at(july), -300);
        assert_eq!(chicago.code(july, true), "CD");
        assert_eq!(chicago.code(january, false), "CT");
    }

    #[test]
    fn local_times_read_back_to_their_instant() {
        let chicago = Zone::named("America/Chicago").expect("Chicago");
        let local = days_from_civil(2026, 1, 10) * 86_400 + 8 * 3600 + 30 * 60;
        let instant = chicago.instant(local);
        assert_eq!(
            instant,
            days_from_civil(2026, 1, 10) * 86_400 + 14 * 3600 + 30 * 60
        );
        let summer_local = days_from_civil(2026, 7, 10) * 86_400 + 8 * 3600;
        assert_eq!(chicago.instant(summer_local), summer_local + 5 * 3600);
    }

    #[test]
    fn phoenix_has_no_daylight_time() {
        let phoenix = Zone::named("America/Phoenix").expect("Phoenix");
        let july = days_from_civil(2026, 7, 10) * 86_400;
        assert_eq!(phoenix.offset_at(july), -420);
        assert_eq!(
            Zone::from_code("PD").map(|zone| zone.name),
            Some("America/Los_Angeles")
        );
    }
}
