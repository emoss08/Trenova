//! Checks one segment's elements against its definition.
//!
//! This is where AK403/IK403 codes come from: a mandatory element that is empty, a value too
//! short or too long, a character its type does not allow, a code outside its list, a date or
//! time that does not exist, too many repetitions or components, an element the partner does
//! not use, a value other than the one the partner fixes, and the syntax notes between elements.
//!
//! The check returns [`ElementIssue`]s rather than findings: inside a transaction set an issue
//! becomes an element finding, while in an ISA or GS it becomes the envelope's own finding with
//! the TA1 or group code for that position.

use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

use super::finding::FindingKind;
use super::guide::model::{ElementDef, ElementType, SegmentDef, Usage};
use super::guide::syntax::SyntaxNote;
use super::lexer::{Segment, components, repetitions};
use super::separators::Separators;

/// One thing wrong with an element.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ElementIssue {
    /// What is wrong.
    pub kind: FindingKind,
    /// The element's position, one-based.
    pub position: u32,
    /// The component's position, for a composite.
    pub component: Option<u32>,
    /// The repetition's position, for a repeating element.
    pub repetition: Option<u32>,
    /// The element's reference number.
    pub reference: String,
    /// The bad value, when there is one.
    pub value: String,
    /// The element's byte span.
    pub span: (usize, usize),
    /// What is wrong, for people.
    pub message: Message,
}

/// How to check.
#[derive(Debug, Clone, Copy, Default)]
pub struct CheckOptions {
    /// Skip length checks, for an ISA whose padding was lost.
    pub ignore_lengths: bool,
}

/// Every issue with `segment`'s elements under `definition`.
#[must_use]
pub fn check_elements(
    segment: &Segment<'_>,
    definition: &SegmentDef,
    separators: &Separators,
    options: CheckOptions,
) -> Vec<ElementIssue> {
    let mut issues = Vec::new();
    let values: Vec<&str> = segment.elements(separators).collect();
    let defined = definition.elements.len();

    for (index, element) in definition.elements.iter().enumerate() {
        let position = u32::try_from(index + 1).unwrap_or(u32::MAX);
        let value = values.get(index).copied().unwrap_or("");
        let span = segment.element_span(separators, index + 1);
        let place = Place {
            segment: segment.id,
            position,
            span,
        };
        check_element(&place, element, value, *separators, options, &mut issues);
    }

    if values.len() > defined
        && let Some(extra) = values[defined..].iter().position(|value| !value.is_empty())
    {
        let position = defined + extra + 1;
        issues.push(ElementIssue {
            kind: FindingKind::TooManyElements,
            position: u32::try_from(position).unwrap_or(u32::MAX),
            component: None,
            repetition: None,
            reference: String::new(),
            value: values[position - 1].to_owned(),
            span: segment.element_span(separators, position),
            message: Message::with(
                message!("{0} has {1} elements, but only {2} are defined"),
                [
                    Arg::from(segment.id),
                    Arg::from(values.len()),
                    Arg::from(defined),
                ],
            ),
        });
    }

    let present = |position: u32| -> bool {
        usize::try_from(position)
            .ok()
            .and_then(|position| values.get(position.wrapping_sub(1)))
            .is_some_and(|value| has_data(value, *separators))
    };
    for rule in &definition.rules {
        let Some(note) = SyntaxNote::parse(rule) else {
            continue;
        };
        let violations = note.violations(present);
        for position in violations.missing {
            let index = usize::try_from(position).unwrap_or(usize::MAX);
            let Some(element) = definition.elements.get(index.wrapping_sub(1)) else {
                continue;
            };
            issues.push(ElementIssue {
                kind: FindingKind::ConditionalElementMissing,
                position,
                component: None,
                repetition: None,
                reference: element.reference.clone(),
                value: String::new(),
                span: segment.element_span(separators, index),
                message: Message::with(
                    message!("{0} ({1}) is required by the syntax note {2}"),
                    [
                        Arg::from(label(segment.id, position, None)),
                        Arg::from(element.name.as_str()),
                        Arg::from(rule.as_str()),
                    ],
                ),
            });
        }
        for position in violations.excluded {
            let index = usize::try_from(position).unwrap_or(usize::MAX);
            let Some(element) = definition.elements.get(index.wrapping_sub(1)) else {
                continue;
            };
            issues.push(ElementIssue {
                kind: FindingKind::ExclusionViolated,
                position,
                component: None,
                repetition: None,
                reference: element.reference.clone(),
                value: values.get(index - 1).copied().unwrap_or("").to_owned(),
                span: segment.element_span(separators, index),
                message: Message::with(
                    message!("{0} ({1}) cannot be used with the others the syntax note {2} names"),
                    [
                        Arg::from(label(segment.id, position, None)),
                        Arg::from(element.name.as_str()),
                        Arg::from(rule.as_str()),
                    ],
                ),
            });
        }
    }
    issues
}

/// Where an element is, for messages and spans.
struct Place<'a> {
    segment: &'a str,
    position: u32,
    span: (usize, usize),
}

fn has_data(value: &str, separators: Separators) -> bool {
    value
        .bytes()
        .any(|byte| byte != separators.component && Some(byte) != separators.repetition)
}

/// `N101`, or `C04001-02` for a component.
#[must_use]
pub fn label(segment: &str, position: u32, component: Option<u32>) -> String {
    match component {
        Some(component) => format!("{segment}{position:02}-{component:02}"),
        None => format!("{segment}{position:02}"),
    }
}

fn check_element(
    place: &Place<'_>,
    element: &ElementDef,
    value: &str,
    separators: Separators,
    options: CheckOptions,
    issues: &mut Vec<ElementIssue>,
) {
    let issue = |kind, value: &str, component, repetition, message| ElementIssue {
        kind,
        position: place.position,
        component,
        repetition,
        reference: element.reference.clone(),
        value: value.to_owned(),
        span: place.span,
        message,
    };

    if value.is_empty() {
        if element.usage == Usage::Mandatory {
            issues.push(issue(
                FindingKind::MandatoryElementMissing,
                "",
                None,
                None,
                Message::with(
                    message!("{0} ({1}) is mandatory but is empty"),
                    [
                        Arg::from(label(place.segment, place.position, None)),
                        Arg::from(element.name.as_str()),
                    ],
                ),
            ));
        }
        return;
    }
    if element.usage == Usage::NotUsed {
        issues.push(issue(
            FindingKind::NotUsedElementPresent,
            value,
            None,
            None,
            Message::with(
                message!("{0} ({1}) is not used by this partner, but has a value"),
                [
                    Arg::from(label(place.segment, place.position, None)),
                    Arg::from(element.name.as_str()),
                ],
            ),
        ));
        return;
    }

    let parts = repetitions(value, &separators);
    let limit = usize::try_from(element.repeat.max(1)).unwrap_or(usize::MAX);
    if parts.len() > limit {
        issues.push(issue(
            FindingKind::TooManyRepetitions,
            value,
            None,
            None,
            Message::with(
                message!("{0} ({1}) repeats {2} times, but may repeat at most {3}"),
                [
                    Arg::from(label(place.segment, place.position, None)),
                    Arg::from(element.name.as_str()),
                    Arg::from(parts.len()),
                    Arg::from(limit),
                ],
            ),
        ));
        return;
    }
    let repeating = parts.len() > 1;
    for (index, part) in parts.iter().enumerate() {
        let repetition = repeating.then(|| u32::try_from(index + 1).unwrap_or(u32::MAX));
        if element.kind == ElementType::Composite {
            check_composite(
                place, element, part, repetition, separators, options, issues,
            );
        } else if part.as_bytes().contains(&separators.component) {
            issues.push(issue(
                FindingKind::TooManyComponents,
                part,
                None,
                repetition,
                Message::with(
                    message!("{0} ({1}) is a simple element but has components"),
                    [
                        Arg::from(label(place.segment, place.position, None)),
                        Arg::from(element.name.as_str()),
                    ],
                ),
            ));
        } else {
            check_value(place, element, part, None, repetition, options, issues);
        }
    }
}

fn check_composite(
    place: &Place<'_>,
    composite: &ElementDef,
    value: &str,
    repetition: Option<u32>,
    separators: Separators,
    options: CheckOptions,
    issues: &mut Vec<ElementIssue>,
) {
    let defined = composite.components.as_deref().unwrap_or_default();
    let parts = components(value, &separators);
    if parts.len() > defined.len() && parts[defined.len()..].iter().any(|part| !part.is_empty()) {
        issues.push(ElementIssue {
            kind: FindingKind::TooManyComponents,
            position: place.position,
            component: None,
            repetition,
            reference: composite.reference.clone(),
            value: value.to_owned(),
            span: place.span,
            message: Message::with(
                message!("{0} ({1}) has {2} components, but only {3} are defined"),
                [
                    Arg::from(label(place.segment, place.position, None)),
                    Arg::from(composite.name.as_str()),
                    Arg::from(parts.len()),
                    Arg::from(defined.len()),
                ],
            ),
        });
    }
    for (index, component) in defined.iter().enumerate() {
        let position = u32::try_from(index + 1).unwrap_or(u32::MAX);
        let part = parts.get(index).copied().unwrap_or("");
        if part.is_empty() {
            if component.usage == Usage::Mandatory {
                issues.push(ElementIssue {
                    kind: FindingKind::MandatoryElementMissing,
                    position: place.position,
                    component: Some(position),
                    repetition,
                    reference: component.reference.clone(),
                    value: String::new(),
                    span: place.span,
                    message: Message::with(
                        message!("{0} ({1}) is mandatory but is empty"),
                        [
                            Arg::from(label(place.segment, place.position, Some(position))),
                            Arg::from(component.name.as_str()),
                        ],
                    ),
                });
            }
            continue;
        }
        check_value(
            place,
            component,
            part,
            Some(position),
            repetition,
            options,
            issues,
        );
    }
}

fn check_value(
    place: &Place<'_>,
    element: &ElementDef,
    value: &str,
    component: Option<u32>,
    repetition: Option<u32>,
    options: CheckOptions,
    issues: &mut Vec<ElementIssue>,
) {
    let mut check = ValueCheck {
        name: label(place.segment, place.position, component),
        place,
        element,
        value,
        component,
        repetition,
        issues,
    };
    if element.usage == Usage::NotUsed {
        check.push(
            FindingKind::NotUsedElementPresent,
            message!("{0} ({1}) is not used by this partner, but has a value"),
        );
        return;
    }
    if let Some(bad) = value.chars().find(|character| character.is_control()) {
        let code = format!("{:04X}", u32::from(bad));
        check.push_with(
            FindingKind::InvalidCharacter,
            message!("{0} ({1}) holds the control character U+{2}"),
            [Arg::from(code)],
        );
        return;
    }
    let Some(length) = check.length() else {
        return;
    };
    if !options.ignore_lengths {
        check.lengths(length);
    }
    check.date_and_time();
    check.codes();
}

/// The checks of one value, with what every issue about it shares.
struct ValueCheck<'c, 'p> {
    place: &'c Place<'p>,
    element: &'c ElementDef,
    value: &'c str,
    component: Option<u32>,
    repetition: Option<u32>,
    name: String,
    issues: &'c mut Vec<ElementIssue>,
}

impl ValueCheck<'_, '_> {
    /// Adds an issue whose message takes the element's label and name.
    fn push(&mut self, kind: FindingKind, template: &'static str) {
        self.push_with(kind, template, []);
    }

    /// Adds an issue whose message takes the label, the name and then `more`.
    fn push_with(
        &mut self,
        kind: FindingKind,
        template: &'static str,
        more: impl IntoIterator<Item = Arg>,
    ) {
        let mut args = vec![
            Arg::from(self.name.as_str()),
            Arg::from(self.element.name.as_str()),
        ];
        args.extend(more);
        self.issues.push(ElementIssue {
            kind,
            position: self.place.position,
            component: self.component,
            repetition: self.repetition,
            reference: self.element.reference.clone(),
            value: self.value.to_owned(),
            span: self.place.span,
            message: Message::with(template, args),
        });
    }

    /// The value's length as X12 counts it, or `None` when it is not a number its type needs.
    fn length(&mut self) -> Option<u32> {
        let shape = match self.element.kind {
            ElementType::Numeric(_) => numeric_shape(self.value, false),
            ElementType::Decimal => numeric_shape(self.value, true),
            _ => Shape::Fine(u32::try_from(self.value.chars().count()).unwrap_or(u32::MAX)),
        };
        match shape {
            Shape::Invalid => {
                self.push_with(
                    FindingKind::InvalidCharacter,
                    message!("{0} ({1}) must be a number, but is {2}"),
                    [Arg::from(self.value)],
                );
                None
            }
            Shape::ExplicitDecimal(length) => {
                self.push_with(
                    FindingKind::ExplicitDecimalInImpliedNumber,
                    message!("{0} ({1}) has implied decimal places, so {2} should be written without a decimal point"),
                    [Arg::from(self.value)],
                );
                Some(length)
            }
            Shape::Fine(length) => Some(length),
        }
    }

    fn lengths(&mut self, length: u32) {
        let (min, max) = (self.element.min, self.element.max);
        if length < min {
            self.push_with(
                FindingKind::ElementTooShort,
                message!("{0} ({1}) is {2} characters, shorter than its minimum of {3}"),
                [Arg::from(i64::from(length)), Arg::from(i64::from(min))],
            );
        } else if length > max {
            self.push_with(
                FindingKind::ElementTooLong,
                message!("{0} ({1}) is {2} characters, longer than its maximum of {3}"),
                [Arg::from(i64::from(length)), Arg::from(i64::from(max))],
            );
        }
    }

    fn date_and_time(&mut self) {
        match self.element.kind {
            ElementType::Date if !is_date(self.value) => self.push_with(
                FindingKind::InvalidDate,
                message!("{0} ({1}) is not a date: {2}"),
                [Arg::from(self.value)],
            ),
            ElementType::Time if !is_time(self.value) => self.push_with(
                FindingKind::InvalidTime,
                message!("{0} ({1}) is not a time: {2}"),
                [Arg::from(self.value)],
            ),
            _ => {}
        }
    }

    fn codes(&mut self) {
        if let Some(fixed) = &self.element.fixed {
            if self.value != fixed {
                self.push_with(
                    FindingKind::FixedValueMismatch,
                    message!("{0} ({1}) must be {2}, but is {3}"),
                    [Arg::from(fixed.as_str()), Arg::from(self.value)],
                );
            }
            return;
        }
        if let Some(codes) = &self.element.codes
            && !codes.iter().any(|code| code.code == self.value)
        {
            self.push_with(
                FindingKind::InvalidCode,
                message!("{0} ({1}) holds {2}, which is not one of its codes"),
                [Arg::from(self.value)],
            );
        }
    }
}

/// What a numeric value looks like: its length as X12 counts it (digits only, no sign or
/// point), or why it is not a number.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Shape {
    Fine(u32),
    ExplicitDecimal(u32),
    Invalid,
}

fn numeric_shape(value: &str, decimal_allowed: bool) -> Shape {
    let unsigned = value.strip_prefix('-').unwrap_or(value);
    let (mantissa, exponent) = if decimal_allowed {
        match unsigned.split_once(['E', 'e']) {
            Some((mantissa, exponent)) => (mantissa, Some(exponent)),
            None => (unsigned, None),
        }
    } else {
        (unsigned, None)
    };
    let mut digits = 0u32;
    let mut points = 0u32;
    for byte in mantissa.bytes() {
        match byte {
            b'0'..=b'9' => digits += 1,
            b'.' => points += 1,
            _ => return Shape::Invalid,
        }
    }
    if digits == 0 || points > 1 {
        return Shape::Invalid;
    }
    if let Some(exponent) = exponent {
        let exponent = exponent.strip_prefix('-').unwrap_or(exponent);
        if exponent.is_empty() || !exponent.bytes().all(|byte| byte.is_ascii_digit()) {
            return Shape::Invalid;
        }
    }
    if points == 1 && !decimal_allowed {
        return Shape::ExplicitDecimal(digits);
    }
    Shape::Fine(digits)
}

/// Whether `value` is a real date written `CCYYMMDD` or `YYMMDD`.
#[must_use]
pub fn is_date(value: &str) -> bool {
    if !value.bytes().all(|byte| byte.is_ascii_digit()) {
        return false;
    }
    let number = |range: std::ops::Range<usize>| -> u32 {
        value
            .get(range)
            .and_then(|digits| digits.parse().ok())
            .unwrap_or(0)
    };
    let (year, month, day) = match value.len() {
        8 => (number(0..4), number(4..6), number(6..8)),
        6 => (2000 + number(0..2), number(2..4), number(4..6)),
        _ => return false,
    };
    let leap = (year % 4 == 0 && year % 100 != 0) || year % 400 == 0;
    let days = match month {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        2 if leap => 29,
        2 => 28,
        _ => return false,
    };
    (1..=days).contains(&day)
}

/// Whether `value` is a real time written `HHMM`, `HHMMSS` or `HHMMSSd..dd`.
#[must_use]
pub fn is_time(value: &str) -> bool {
    if !value.bytes().all(|byte| byte.is_ascii_digit())
        || !(4..=8).contains(&value.len())
        || value.len() == 5
    {
        return false;
    }
    let number = |range: std::ops::Range<usize>| -> u32 {
        value
            .get(range)
            .and_then(|digits| digits.parse().ok())
            .unwrap_or(99)
    };
    number(0..2) < 24 && number(2..4) < 60 && (value.len() < 6 || number(4..6) < 60)
}

#[cfg(test)]
mod tests {
    use super::{CheckOptions, check_elements, is_date, is_time};
    use crate::x12::finding::FindingKind;
    use crate::x12::guide::builtin;
    use crate::x12::lexer::Lexer;
    use crate::x12::separators::Separators;

    fn issues(set: &str, text: &str) -> Vec<(FindingKind, u32, Option<u32>)> {
        let guide = builtin::by_id(set).expect("a guide");
        let separators = Separators::default();
        let segment = Lexer::new(text, 0, separators).next().expect("a segment");
        let definition = guide.segment(segment.id).expect("a definition");
        check_elements(&segment, definition, &separators, CheckOptions::default())
            .into_iter()
            .map(|issue| (issue.kind, issue.position, issue.component))
            .collect()
    }

    #[test]
    fn accepts_a_well_formed_segment() {
        assert!(issues("x12.004010.204", "N1*SH*ACME DC*93*A100~").is_empty());
        assert!(issues("x12.004010.204", "G62*10*20260107*Y*1504~").is_empty());
        assert!(issues("x12.004010.204", "L3*1000*G***150000~").is_empty());
    }

    #[test]
    fn reports_missing_short_long_and_bad_values() {
        assert_eq!(
            issues("x12.004010.204", "N1**ACME~"),
            [(FindingKind::MandatoryElementMissing, 1, None)]
        );
        assert_eq!(
            issues("x12.004010.204", "N1*S~"),
            [
                (FindingKind::ElementTooShort, 1, None),
                (FindingKind::ConditionalElementMissing, 2, None)
            ]
        );
        assert_eq!(
            issues("x12.004010.204", "G62*10*20260231~"),
            [(FindingKind::InvalidDate, 2, None)]
        );
        assert_eq!(
            issues("x12.004010.204", "G62*10*20260107*Y*2575~"),
            [(FindingKind::InvalidTime, 4, None)]
        );
        assert_eq!(
            issues("x12.004010.204", "S5*X1*CL~"),
            [(FindingKind::InvalidCharacter, 1, None)]
        );
    }

    #[test]
    fn warns_about_a_decimal_point_in_an_implied_decimal_number() {
        assert_eq!(
            issues("x12.004010.204", "L3*1000*G***1500.00~"),
            [(FindingKind::ExplicitDecimalInImpliedNumber, 5, None)]
        );
    }

    #[test]
    fn checks_syntax_notes_and_codes() {
        assert_eq!(
            issues("x12.004010.204", "N1*SH**93~"),
            [(FindingKind::ConditionalElementMissing, 4, None)]
        );
        assert_eq!(
            issues("x12.004010.204", "N1*SH~"),
            [(FindingKind::ConditionalElementMissing, 2, None)]
        );
        assert_eq!(
            issues("x12.004010.997", "AK5*Q~"),
            [(FindingKind::InvalidCode, 1, None)]
        );
    }

    #[test]
    fn checks_composites_components_and_extra_elements() {
        assert!(issues("x12.004010.997", "AK4*2>1*98*1~").is_empty());
        assert_eq!(
            issues("x12.004010.997", "AK4*>1*98*1~"),
            [(FindingKind::MandatoryElementMissing, 1, Some(1))]
        );
        assert_eq!(
            issues("x12.004010.997", "AK4*2>1>3*98*1~"),
            [(FindingKind::TooManyComponents, 1, None)]
        );
        assert_eq!(
            issues("x12.004010.204", "N3*1 MAIN*SUITE 2*EXTRA~"),
            [(FindingKind::TooManyElements, 3, None)]
        );
        assert!(
            issues("x12.004010.204", "N3*1 MAIN**~").is_empty(),
            "trailing empty elements are tolerated"
        );
        assert_eq!(
            issues("x12.004010.204", "N3*1>MAIN~"),
            [(FindingKind::TooManyComponents, 1, None)]
        );
    }

    #[test]
    fn dates_and_times_are_real_ones() {
        assert!(is_date("20240229"));
        assert!(!is_date("20230229"));
        assert!(is_date("260107"));
        assert!(!is_date("2026010"));
        assert!(is_time("2359"));
        assert!(is_time("235959"));
        assert!(is_time("23595999"));
        assert!(!is_time("2400"));
        assert!(!is_time("23599"));
    }
}
