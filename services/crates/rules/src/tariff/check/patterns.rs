//! `match` coverage: exhaustiveness, overlapping arms and arms nothing reaches.
//!
//! The checker types each arm's pattern and lowers it to a [`Pat`]; this module then runs the
//! usefulness algorithm (Maranget's "Warnings for pattern matching") over the arms:
//!
//! - **Exhaustive** when the wildcard is not useful after every unguarded arm. The values it is
//!   still useful for are the witnesses the error names (`5_000 lb and above`, `Reefer`,
//!   `none`).
//! - **Reachable** when an arm's pattern is useful after the unguarded arms above it; one that
//!   is not is a warning.
//! - **Overlapping** ranges and repeated variants or literals between two arms are errors that
//!   name both arms, so a weight-break table can never bill a weight twice.
//!
//! Ranges of numbers, percents, money, quantities, dates, date-times and times of day are
//! intervals on a line. The line is cut at every bound any arm writes, so each piece is either
//! inside or outside every arm, and coverage is decided per piece. `Int`, `Date` and `Time` lines
//! are discrete (`..=4` and `5..` leave no gap); the others are continuous (`..500 lb` and
//! `500 lb..` leave none either, since the start of a range is inclusive). A line may have a
//! [`Domain`]: a time of day runs from `@00:00` to `@23:59:59`, so pieces outside it need no arm.
//! Guarded arms never count toward coverage.
//!
//! Text is a [`Space::Text`]: its literals are never all of it, so without ranges a text match
//! needs `_`. Once an arm writes a text range (`"300".."320"`), the text is a continuous line in
//! Unicode scalar order, starting at the empty text, and its literals are points on it. A range
//! of an ordered enum (`Mon..=Fri`) is lowered by the checker to the variants it spans.
//!
//! The work is bounded: past [`WORK_LIMIT`] steps the analysis stops and reports nothing more,
//! so a pathological match costs a bounded amount of time.

use crate::message;
use crate::message::{Arg, Message};
use rust_decimal::Decimal;
use std::cmp::Ordering;

/// How many matrix rows the analysis may visit before it stops.
pub const WORK_LIMIT: usize = 20_000;

/// The most uncovered values an error names; the rest are counted.
const MOST_NAMED: usize = 3;

/// A pattern, lowered for coverage.
#[derive(Debug, Clone, PartialEq)]
pub enum Pat {
    /// Matches anything: `_`, a binding, or a pattern that did not check.
    Any,
    /// `none`.
    None,
    /// `true` or `false`.
    Bool(bool),
    /// An enum variant.
    Variant(String),
    /// A range, or a number, money, quantity, percent or date literal (a closed interval of one
    /// point).
    Interval(Interval),
    /// A text literal: a point on the text line.
    Literal(String),
    /// A record pattern: fields not written match anything.
    Record(Vec<(String, Pat)>),
    /// `a | b`.
    Or(Vec<Pat>),
}

/// A position on a line: a number (a quantity in the match's first unit, a date as days, a
/// date-time or a time as seconds) or a text.
#[derive(Debug, Clone, PartialEq, Eq, PartialOrd, Ord)]
pub enum Key {
    /// A position on a number line.
    Number(Decimal),
    /// A text, ordered by Unicode scalar values.
    Text(String),
}

impl Key {
    /// The next position on a discrete line; a text has none, so it stays.
    fn next(&self) -> Self {
        match self {
            Self::Number(value) => Self::Number(value.checked_add(Decimal::ONE).unwrap_or(*value)),
            Self::Text(_) => self.clone(),
        }
    }

    /// The previous position on a discrete line.
    fn previous(&self) -> Self {
        match self {
            Self::Number(value) => Self::Number(value.checked_sub(Decimal::ONE).unwrap_or(*value)),
            Self::Text(_) => self.clone(),
        }
    }
}

/// The values a line holds, both ends included: a time of day from 0 to 86 399 seconds.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Domain {
    /// The least value.
    pub min: Decimal,
    /// The greatest value.
    pub max: Decimal,
}

/// One end of an interval.
#[derive(Debug, Clone, PartialEq)]
pub struct Bound {
    /// The value, in one unit for every bound of the match.
    pub value: Key,
    /// Whether the end itself is inside the interval.
    pub inclusive: bool,
    /// The bound as written, for messages: `5_000 lb`.
    pub label: String,
}

/// An interval on a line; a missing end is unbounded.
#[derive(Debug, Clone, PartialEq)]
pub struct Interval {
    /// The lower end.
    pub lo: Option<Bound>,
    /// The upper end.
    pub hi: Option<Bound>,
}

/// The values a scrutinee can take, by structure.
#[derive(Debug, Clone, PartialEq)]
pub enum Space {
    /// `T?`: `none` or a value of the inner space.
    Optional(Box<Space>),
    /// `true` and `false`.
    Bool,
    /// The variants of an enum.
    Enum(Vec<String>),
    /// An ordered line; `discrete` for whole numbers, dates and times of day.
    Line {
        /// Whether only whole steps exist.
        discrete: bool,
        /// The values the line holds, when it is bounded.
        domain: Option<Domain>,
    },
    /// Text: literals, and once a range is written, a line in Unicode scalar order.
    Text,
    /// Values no finite set of patterns covers without `_` (lists).
    Open,
    /// A record's fields.
    Record(Vec<(String, Space)>),
    /// A value of unknown type: only `_` is meaningful.
    Unknown,
}

/// One arm, for the analysis.
#[derive(Debug, Clone)]
pub struct ArmPat {
    /// Its pattern.
    pub pat: Pat,
    /// Whether it has a guard, so it never counts toward coverage.
    pub guarded: bool,
    /// Whether its pattern did not check, so it takes no part in the analysis.
    pub broken: bool,
}

/// What the analysis found.
#[derive(Debug, Default)]
pub struct Coverage {
    /// The values no arm covers, named; empty when the match is exhaustive.
    pub missing: Vec<Message>,
    /// How many more values are uncovered than `missing` names.
    pub more: usize,
    /// Arms whose pattern overlaps an earlier arm's: (arm, earlier arm, what both match).
    pub overlaps: Vec<(usize, usize, Message)>,
    /// Arms nothing reaches.
    pub unreachable: Vec<usize>,
}

/// Analyses a match's arms over its scrutinee's space.
#[must_use]
pub fn analyze(space: &Space, arms: &[ArmPat]) -> Coverage {
    let mut coverage = Coverage::default();
    let mut work = Work { left: WORK_LIMIT };
    let mut rows: Vec<Vec<Pat>> = Vec::new();
    for (index, arm) in arms.iter().enumerate() {
        if arm.broken {
            continue;
        }
        if let Some((earlier, what)) = overlap(space, arms, index) {
            coverage.overlaps.push((index, earlier, what));
        } else {
            let useful = useful(
                &rows,
                std::slice::from_ref(&arm.pat),
                std::slice::from_ref(space),
                &mut work,
                1,
            );
            if work.exhausted() {
                return Coverage::default();
            }
            if useful.is_empty() {
                coverage.unreachable.push(index);
            }
        }
        if !arm.guarded {
            rows.push(vec![arm.pat.clone()]);
        }
    }
    let witnesses = useful(
        &rows,
        &[Pat::Any],
        std::slice::from_ref(space),
        &mut work,
        64,
    );
    if work.exhausted() {
        return Coverage::default();
    }
    let merged = merge(
        witnesses
            .into_iter()
            .filter_map(|mut row| row.pop())
            .collect(),
    );
    coverage.more = merged.len().saturating_sub(MOST_NAMED);
    coverage.missing = merged
        .iter()
        .take(MOST_NAMED)
        .map(Witness::describe)
        .collect();
    coverage
}

/// The list of uncovered values as one phrase: `` `Reefer` and `Flatbed` ``.
#[must_use]
pub fn listing(items: &[Message], more: usize) -> Message {
    let arg = |index: usize| -> Arg {
        items
            .get(index)
            .cloned()
            .map_or(Arg::Text(String::new()), Arg::from)
    };
    if more > 0 {
        return Message::with(
            message!("{0}, {1}, {2} and {3, plural, one {# other value} other {# other values}}"),
            [arg(0), arg(1), arg(2), Arg::from(more)],
        );
    }
    match items.len() {
        0 | 1 => items
            .first()
            .cloned()
            .unwrap_or_else(|| Message::new(message!("any value"))),
        2 => Message::with(message!("{0} and {1}"), [arg(0), arg(1)]),
        _ => Message::with(message!("{0}, {1} and {2}"), [arg(0), arg(1), arg(2)]),
    }
}

struct Work {
    left: usize,
}

impl Work {
    fn spend(&mut self, amount: usize) -> bool {
        self.left = self.left.saturating_sub(amount.max(1));
        self.left > 0
    }

    const fn exhausted(&self) -> bool {
        self.left == 0
    }
}

/// A value no arm matches, as a tree of constructors.
#[derive(Debug, Clone, PartialEq)]
enum Witness {
    Any,
    None,
    Some(Box<Witness>),
    Bool(bool),
    Variant(String),
    Segment(Segment),
    Other,
    Record(Vec<(String, Witness)>),
}

/// A piece of a cut line.
#[derive(Debug, Clone, PartialEq)]
struct Segment {
    lo: Option<Bound>,
    hi: Option<Bound>,
    discrete: bool,
}

/// The constructors of a column.
#[derive(Debug, Clone, PartialEq)]
enum Ctor {
    None,
    Some,
    Bool(bool),
    Variant(String),
    Segment(Segment),
    Literal(String),
    Other,
    Record,
}

fn expand_or(pat: &Pat) -> Vec<Pat> {
    match pat {
        Pat::Or(alts) => alts.iter().flat_map(expand_or).collect(),
        other => vec![other.clone()],
    }
}

/// The rows of a matrix with or-patterns in the first column expanded.
fn expanded(rows: &[Vec<Pat>]) -> Vec<Vec<Pat>> {
    let mut out = Vec::with_capacity(rows.len());
    for row in rows {
        let Some((head, rest)) = row.split_first() else {
            out.push(row.clone());
            continue;
        };
        for alt in expand_or(head) {
            let mut next = Vec::with_capacity(row.len());
            next.push(alt);
            next.extend_from_slice(rest);
            out.push(next);
        }
    }
    out
}

/// The witnesses for which `q` is useful against `rows`, up to `limit`.
fn useful(
    rows: &[Vec<Pat>],
    q: &[Pat],
    spaces: &[Space],
    work: &mut Work,
    limit: usize,
) -> Vec<Vec<Witness>> {
    if !work.spend(rows.len()) {
        return Vec::new();
    }
    let (Some(head), Some(space)) = (q.first(), spaces.first()) else {
        return if rows.is_empty() {
            vec![Vec::new()]
        } else {
            Vec::new()
        };
    };
    if let Pat::Or(alts) = head {
        let mut out = Vec::new();
        for alt in alts {
            let mut next = vec![alt.clone()];
            next.extend_from_slice(q.get(1..).unwrap_or_default());
            out.extend(useful(
                rows,
                &next,
                spaces,
                work,
                limit.saturating_sub(out.len()),
            ));
            if out.len() >= limit {
                break;
            }
        }
        return out;
    }
    let rows = expanded(rows);
    let ctors = constructors(space, &rows, head);
    if !matches!(head, Pat::Any) {
        let covered: Vec<&Ctor> = ctors.iter().filter(|ctor| pat_covers(head, ctor)).collect();
        return specialize_all(&rows, q, spaces, &covered, work, limit);
    }
    let complete = ctors.iter().all(|ctor| {
        rows.iter().any(|row| {
            row.first()
                .is_some_and(|pat| !matches!(pat, Pat::Any) && pat_covers(pat, ctor))
        })
    });
    if complete || space_is_finite_record(space) {
        let all: Vec<&Ctor> = ctors.iter().collect();
        return specialize_all(&rows, q, spaces, &all, work, limit);
    }
    let defaults: Vec<Vec<Pat>> = rows
        .iter()
        .filter(|row| matches!(row.first(), Some(Pat::Any)))
        .map(|row| row.get(1..).unwrap_or_default().to_vec())
        .collect();
    let tail = useful(
        &defaults,
        q.get(1..).unwrap_or_default(),
        spaces.get(1..).unwrap_or_default(),
        work,
        limit,
    );
    if tail.is_empty() {
        return Vec::new();
    }
    let missing: Vec<Witness> = ctors
        .iter()
        .filter(|ctor| {
            !rows.iter().any(|row| {
                row.first()
                    .is_some_and(|pat| !matches!(pat, Pat::Any) && pat_covers(pat, ctor))
            })
        })
        .map(|ctor| witness_of(ctor, space))
        .collect();
    let mut out = Vec::new();
    for rest in tail {
        for head in &missing {
            let mut row = rest.clone();
            row.insert(0, head.clone());
            out.push(row);
            if out.len() >= limit {
                return out;
            }
        }
    }
    out
}

/// The witnesses of specialising by each constructor in turn, up to `limit`.
fn specialize_all(
    rows: &[Vec<Pat>],
    q: &[Pat],
    spaces: &[Space],
    ctors: &[&Ctor],
    work: &mut Work,
    limit: usize,
) -> Vec<Vec<Witness>> {
    let mut out = Vec::new();
    for ctor in ctors {
        out.extend(specialized(
            rows,
            q,
            spaces,
            ctor,
            work,
            limit.saturating_sub(out.len()),
        ));
        if out.len() >= limit {
            break;
        }
    }
    out
}

fn space_is_finite_record(space: &Space) -> bool {
    matches!(space, Space::Record(_))
}

/// The witness for a constructor whose arguments are anything.
fn witness_of(ctor: &Ctor, space: &Space) -> Witness {
    match ctor {
        Ctor::None => Witness::None,
        Ctor::Some => Witness::Some(Box::new(Witness::Any)),
        Ctor::Bool(value) => Witness::Bool(*value),
        Ctor::Variant(name) => Witness::Variant(name.clone()),
        Ctor::Segment(segment) => Witness::Segment(segment.clone()),
        Ctor::Literal(_) | Ctor::Other => Witness::Other,
        Ctor::Record => match space {
            Space::Record(fields) => Witness::Record(
                fields
                    .iter()
                    .map(|(name, _)| (name.clone(), Witness::Any))
                    .collect(),
            ),
            _ => Witness::Any,
        },
    }
}

/// `S(c, P)` and `S(c, q)`, then usefulness, with the witnesses rebuilt around `c`.
fn specialized(
    rows: &[Vec<Pat>],
    q: &[Pat],
    spaces: &[Space],
    ctor: &Ctor,
    work: &mut Work,
    limit: usize,
) -> Vec<Vec<Witness>> {
    let Some(space) = spaces.first() else {
        return Vec::new();
    };
    let arity_spaces = sub_spaces(space, ctor);
    let mut next_spaces = arity_spaces.clone();
    next_spaces.extend_from_slice(spaces.get(1..).unwrap_or_default());
    let specialize = |row: &[Pat]| -> Option<Vec<Pat>> {
        let (head, rest) = row.split_first()?;
        let mut out = specialize_pat(head, ctor, space)?;
        out.extend_from_slice(rest);
        Some(out)
    };
    let next_rows: Vec<Vec<Pat>> = rows.iter().filter_map(|row| specialize(row)).collect();
    let Some(next_q) = specialize(q) else {
        return Vec::new();
    };
    let found = useful(&next_rows, &next_q, &next_spaces, work, limit);
    let arity = arity_spaces.len();
    found
        .into_iter()
        .map(|mut row| {
            let args: Vec<Witness> = row.drain(..arity.min(row.len())).collect();
            row.insert(0, rebuild(ctor, space, args));
            row
        })
        .collect()
}

fn rebuild(ctor: &Ctor, space: &Space, mut args: Vec<Witness>) -> Witness {
    match ctor {
        Ctor::Some => Witness::Some(Box::new(args.pop().unwrap_or(Witness::Any))),
        Ctor::Record => match space {
            Space::Record(fields) => Witness::Record(
                fields
                    .iter()
                    .map(|(name, _)| name.clone())
                    .zip(args)
                    .collect(),
            ),
            _ => Witness::Any,
        },
        other => witness_of(other, space),
    }
}

/// The spaces of a constructor's arguments.
fn sub_spaces(space: &Space, ctor: &Ctor) -> Vec<Space> {
    match (space, ctor) {
        (Space::Optional(inner), Ctor::Some) => vec![(**inner).clone()],
        (Space::Record(fields), Ctor::Record) => {
            fields.iter().map(|(_, space)| space.clone()).collect()
        }
        _ => Vec::new(),
    }
}

/// A pattern specialised by a constructor: its arguments' patterns, or `None` when it does not
/// match the constructor.
fn specialize_pat(pat: &Pat, ctor: &Ctor, space: &Space) -> Option<Vec<Pat>> {
    let arity = sub_spaces(space, ctor).len();
    match (space, pat) {
        (_, Pat::Any) => Some(vec![Pat::Any; arity]),
        (Space::Optional(_), _) => match (pat, ctor) {
            (Pat::None, Ctor::None) => Some(Vec::new()),
            (_, Ctor::Some) if !matches!(pat, Pat::None) => Some(vec![pat.clone()]),
            _ => None,
        },
        (Space::Record(fields), Pat::Record(written)) => Some(
            fields
                .iter()
                .map(|(name, _)| {
                    written
                        .iter()
                        .find(|(field, _)| field == name)
                        .map_or(Pat::Any, |(_, pat)| pat.clone())
                })
                .collect(),
        ),
        _ => pat_covers(pat, ctor).then(Vec::new),
    }
}

/// Whether a (non-or) pattern matches every value of a constructor of arity zero, or (for an
/// optional's `Some` and a record) matches the constructor at all.
fn pat_covers(pat: &Pat, ctor: &Ctor) -> bool {
    match (pat, ctor) {
        (Pat::Or(alts), _) => alts.iter().any(|alt| pat_covers(alt, ctor)),
        (Pat::Bool(a), Ctor::Bool(b)) => a == b,
        (Pat::Variant(a), Ctor::Variant(b)) | (Pat::Literal(a), Ctor::Literal(b)) => a == b,
        (Pat::Interval(interval), Ctor::Segment(segment)) => contains(interval, segment),
        (Pat::Literal(text), Ctor::Segment(segment)) => contains(&point(text), segment),
        (Pat::None, Ctor::Some) => false,
        _ => matches!(
            (pat, ctor),
            (Pat::Any, _)
                | (Pat::None, Ctor::None)
                | (_, Ctor::Some)
                | (Pat::Record(_), Ctor::Record)
        ),
    }
}

/// The constructors of the first column's space, cut by the patterns written in it.
fn constructors(space: &Space, rows: &[Vec<Pat>], head: &Pat) -> Vec<Ctor> {
    match space {
        Space::Optional(_) => vec![Ctor::None, Ctor::Some],
        Space::Bool => vec![Ctor::Bool(true), Ctor::Bool(false)],
        Space::Enum(variants) => variants.iter().cloned().map(Ctor::Variant).collect(),
        Space::Record(_) => vec![Ctor::Record],
        Space::Line { discrete, domain } => {
            let mut intervals: Vec<Interval> = Vec::new();
            let heads = rows
                .iter()
                .filter_map(|row| row.first())
                .chain(std::iter::once(head));
            for pat in heads {
                for alt in expand_or_ref(pat) {
                    if let Pat::Interval(interval) = alt {
                        intervals.push(interval.clone());
                    }
                }
            }
            segments(&intervals, *discrete)
                .into_iter()
                .filter(|segment| domain.is_none_or(|domain| inside(segment, &domain)))
                .map(Ctor::Segment)
                .collect()
        }
        Space::Text if text_line(rows, head) => {
            let mut intervals: Vec<Interval> = Vec::new();
            let heads = rows
                .iter()
                .filter_map(|row| row.first())
                .chain(std::iter::once(head));
            for pat in heads {
                for alt in expand_or_ref(pat) {
                    match alt {
                        Pat::Interval(interval) => intervals.push(interval.clone()),
                        Pat::Literal(text) => intervals.push(point(text)),
                        _ => {}
                    }
                }
            }
            segments(&intervals, false)
                .into_iter()
                .filter(|segment| !below_empty_text(segment))
                .map(Ctor::Segment)
                .collect()
        }
        Space::Text | Space::Open => {
            let mut literals: Vec<String> = Vec::new();
            let mut push = |pat: &Pat| {
                for alt in expand_or_ref(pat) {
                    if let Pat::Literal(text) = alt
                        && !literals.contains(text)
                    {
                        literals.push(text.clone());
                    }
                }
            };
            for row in rows {
                if let Some(pat) = row.first() {
                    push(pat);
                }
            }
            push(head);
            let mut out: Vec<Ctor> = literals.into_iter().map(Ctor::Literal).collect();
            out.push(Ctor::Other);
            out
        }
        Space::Unknown => vec![Ctor::Other],
    }
}

fn expand_or_ref(pat: &Pat) -> Vec<&Pat> {
    match pat {
        Pat::Or(alts) => alts.iter().flat_map(expand_or_ref).collect(),
        other => vec![other],
    }
}

/// Whether a text column is cut as a line: some pattern in it is a range.
fn text_line(rows: &[Vec<Pat>], head: &Pat) -> bool {
    rows.iter()
        .filter_map(|row| row.first())
        .chain(std::iter::once(head))
        .any(|pat| {
            expand_or_ref(pat)
                .iter()
                .any(|alt| matches!(alt, Pat::Interval(_)))
        })
}

/// A text literal as a point on the text line.
fn point(text: &str) -> Interval {
    let bound = Bound {
        value: Key::Text(text.to_owned()),
        inclusive: true,
        label: text_label(text),
    };
    Interval {
        lo: Some(bound.clone()),
        hi: Some(bound),
    }
}

/// A text as Tariff writes it: quoted, with its quotes, backslashes and braces escaped.
fn text_label(text: &str) -> String {
    let mut out = String::with_capacity(text.len() + 2);
    out.push('"');
    for c in text.chars() {
        match c {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '{' => out.push_str("{{"),
            '}' => out.push_str("}}"),
            other => out.push(other),
        }
    }
    out.push('"');
    out
}

/// Whether a piece of the text line lies below the empty text, where no text is.
fn below_empty_text(segment: &Segment) -> bool {
    segment.lo.is_none()
        && segment
            .hi
            .as_ref()
            .is_some_and(|hi| !hi.inclusive && hi.value == Key::Text(String::new()))
}

/// Whether a piece of a bounded line holds any value of its domain.
fn inside(segment: &Segment, domain: &Domain) -> bool {
    let min = Key::Number(domain.min);
    let max = Key::Number(domain.max);
    let below = segment
        .hi
        .as_ref()
        .is_some_and(|hi| hi.value < min || (hi.value == min && !hi.inclusive));
    let above = segment
        .lo
        .as_ref()
        .is_some_and(|lo| lo.value > max || (lo.value == max && !lo.inclusive));
    !below && !above
}

/// The pieces a line is cut into by the bounds of the intervals written on it.
fn segments(intervals: &[Interval], discrete: bool) -> Vec<Segment> {
    let mut bounds: Vec<Bound> = Vec::new();
    for interval in intervals {
        let lo = interval.lo.clone();
        let hi = interval.hi.as_ref().map(|bound| {
            if discrete {
                half_open(bound)
            } else {
                bound.clone()
            }
        });
        for bound in [lo, hi].into_iter().flatten() {
            if !bounds.iter().any(|known| known.value == bound.value) {
                bounds.push(bound);
            }
        }
    }
    bounds.sort_by(|a, b| a.value.cmp(&b.value));
    let mut out = Vec::with_capacity(bounds.len() * 2 + 1);
    let mut previous: Option<Bound> = None;
    for bound in bounds {
        out.push(Segment {
            lo: previous.clone().map(|lo| Bound {
                inclusive: discrete,
                ..lo
            }),
            hi: Some(Bound {
                inclusive: false,
                ..bound.clone()
            }),
            discrete,
        });
        if !discrete {
            out.push(Segment {
                lo: Some(Bound {
                    inclusive: true,
                    ..bound.clone()
                }),
                hi: Some(Bound {
                    inclusive: true,
                    ..bound.clone()
                }),
                discrete,
            });
        }
        previous = Some(bound);
    }
    out.push(Segment {
        lo: previous.map(|lo| Bound {
            inclusive: discrete,
            ..lo
        }),
        hi: None,
        discrete,
    });
    out
}

/// On a discrete line, an upper bound as the exclusive bound of a half-open interval: `..=4`
/// is `..5`.
fn half_open(bound: &Bound) -> Bound {
    if bound.inclusive {
        let value = bound.value.next();
        return Bound {
            label: discrete_label(&value, &bound.label),
            value,
            inclusive: false,
        };
    }
    bound.clone()
}

/// Whether an interval contains a whole segment.
fn contains(interval: &Interval, segment: &Segment) -> bool {
    let lo_ok = match (&interval.lo, &segment.lo) {
        (None, _) => true,
        (Some(_), None) => false,
        (Some(lo), Some(start)) => match lo.value.cmp(&start.value) {
            Ordering::Less => true,
            Ordering::Equal => lo.inclusive || !start.inclusive,
            Ordering::Greater => false,
        },
    };
    let hi_ok = match (&interval.hi, &segment.hi) {
        (None, _) => true,
        (Some(_), None) => false,
        (Some(hi), Some(end)) => {
            let hi_value = if segment.discrete && hi.inclusive {
                hi.value.next()
            } else {
                hi.value.clone()
            };
            let hi_inclusive = hi.inclusive && !segment.discrete;
            match hi_value.cmp(&end.value) {
                Ordering::Greater => true,
                Ordering::Equal => hi_inclusive || !end.inclusive,
                Ordering::Less => false,
            }
        }
    };
    lo_ok && hi_ok
}

/// Two intervals' common part, when they have one.
fn intersection(a: &Interval, b: &Interval) -> Option<Interval> {
    let lo = match (&a.lo, &b.lo) {
        (None, other) | (other, None) => other.clone(),
        (Some(x), Some(y)) => Some(match x.value.cmp(&y.value) {
            Ordering::Greater => x.clone(),
            Ordering::Less => y.clone(),
            Ordering::Equal => {
                if x.inclusive {
                    y.clone()
                } else {
                    x.clone()
                }
            }
        }),
    };
    let hi = match (&a.hi, &b.hi) {
        (None, other) | (other, None) => other.clone(),
        (Some(x), Some(y)) => Some(match x.value.cmp(&y.value) {
            Ordering::Less => x.clone(),
            Ordering::Greater => y.clone(),
            Ordering::Equal => {
                if x.inclusive {
                    y.clone()
                } else {
                    x.clone()
                }
            }
        }),
    };
    if let (Some(lo), Some(hi)) = (&lo, &hi) {
        match lo.value.cmp(&hi.value) {
            Ordering::Greater => return None,
            Ordering::Equal if !(lo.inclusive && hi.inclusive) => return None,
            _ => {}
        }
    }
    Some(Interval { lo, hi })
}

/// The first earlier unguarded arm whose pattern overlaps this arm's, with what both match.
/// Only ranges, literals, variants and `none` overlap: a wildcard or binding above is an arm
/// nothing reaches, not an overlap.
fn overlap(space: &Space, arms: &[ArmPat], index: usize) -> Option<(usize, Message)> {
    let arm = arms.get(index)?;
    let mine = leaves(&arm.pat);
    for (earlier, other) in arms.iter().enumerate().take(index) {
        if other.guarded || arm.guarded || other.broken {
            continue;
        }
        for theirs in leaves(&other.pat) {
            for leaf in &mine {
                if let Some(what) = leaf_overlap(leaf, theirs, space) {
                    return Some((earlier, what));
                }
            }
        }
    }
    None
}

fn leaves(pat: &Pat) -> Vec<&Pat> {
    expand_or_ref(pat)
}

fn leaf_overlap(a: &Pat, b: &Pat, space: &Space) -> Option<Message> {
    match (a, b) {
        (Pat::Interval(x), Pat::Interval(y)) => {
            let common = intersection(x, y)?;
            let discrete = line_discrete(space);
            Some(describe_interval(
                common.lo.as_ref(),
                common.hi.as_ref(),
                discrete,
            ))
        }
        (Pat::Variant(x), Pat::Variant(y)) | (Pat::Literal(x), Pat::Literal(y)) if x == y => {
            Some(Message::with(message!("`{0}`"), [Arg::Text(x.clone())]))
        }
        (Pat::Literal(text), Pat::Interval(interval))
        | (Pat::Interval(interval), Pat::Literal(text)) => {
            intersection(&point(text), interval)?;
            Some(Message::with(message!("`{0}`"), [Arg::Text(text.clone())]))
        }
        (Pat::Bool(x), Pat::Bool(y)) if x == y => {
            Some(Message::with(message!("`{0}`"), [Arg::Text(x.to_string())]))
        }
        (Pat::None, Pat::None) => Some(Message::new(message!("`none`"))),
        _ => None,
    }
}

fn line_discrete(space: &Space) -> bool {
    match space {
        Space::Optional(inner) => line_discrete(inner),
        Space::Line { discrete, .. } => *discrete,
        _ => false,
    }
}

/// Joins adjacent uncovered pieces of a line into one interval, so the message reads
/// `5_000 lb and above` rather than `exactly 5_000 lb` and `above 5_000 lb`.
fn merge(witnesses: Vec<Witness>) -> Vec<Witness> {
    let mut out: Vec<Witness> = Vec::with_capacity(witnesses.len());
    for witness in witnesses {
        if let Some(last) = out.last_mut()
            && let Some(joined) = join(last, &witness)
        {
            *last = joined;
            continue;
        }
        out.push(witness);
    }
    out
}

fn join(a: &Witness, b: &Witness) -> Option<Witness> {
    match (a, b) {
        (Witness::Segment(x), Witness::Segment(y)) => {
            let (Some(end), Some(start)) = (&x.hi, &y.lo) else {
                return None;
            };
            let touches =
                end.value == start.value && (end.inclusive != start.inclusive || x.discrete);
            touches.then(|| {
                Witness::Segment(Segment {
                    lo: x.lo.clone(),
                    hi: y.hi.clone(),
                    discrete: x.discrete,
                })
            })
        }
        (Witness::Some(x), Witness::Some(y)) => join(x, y).map(|w| Witness::Some(Box::new(w))),
        _ => None,
    }
}

impl Witness {
    fn describe(&self) -> Message {
        match self {
            Self::Any | Self::Other => Message::new(message!("any other value")),
            Self::None => Message::new(message!("`none`")),
            Self::Some(inner) => match **inner {
                Self::Any => Message::new(message!("any value that isn't `none`")),
                _ => inner.describe(),
            },
            Self::Bool(value) => Message::with(message!("`{0}`"), [Arg::Text(value.to_string())]),
            Self::Variant(name) => Message::with(message!("`{0}`"), [Arg::Text(name.clone())]),
            Self::Segment(segment) => {
                describe_interval(segment.lo.as_ref(), segment.hi.as_ref(), segment.discrete)
            }
            Self::Record(_) => Message::with(message!("`{0}`"), [Arg::Text(self.code())]),
        }
    }

    /// The witness as a pattern an author could write.
    fn code(&self) -> String {
        match self {
            Self::Any | Self::Other => String::from("_"),
            Self::None => String::from("none"),
            Self::Some(inner) => inner.code(),
            Self::Bool(value) => value.to_string(),
            Self::Variant(name) => name.clone(),
            Self::Segment(segment) => {
                let lo = segment
                    .lo
                    .as_ref()
                    .map_or(String::new(), |b| b.label.clone());
                let hi = segment
                    .hi
                    .as_ref()
                    .map_or(String::new(), |b| b.label.clone());
                let dots = if segment.hi.as_ref().is_some_and(|b| b.inclusive) {
                    "..="
                } else {
                    ".."
                };
                format!("{lo}{dots}{hi}")
            }
            Self::Record(fields) => {
                let written: Vec<String> = fields
                    .iter()
                    .filter(|(_, witness)| !matches!(witness, Self::Any))
                    .map(|(name, witness)| format!("{name}: {}", witness.code()))
                    .collect();
                if written.is_empty() {
                    String::from("_")
                } else {
                    format!("{{ {} }}", written.join(", "))
                }
            }
        }
    }
}

fn label(bound: &Bound) -> Arg {
    Arg::Text(format!("`{}`", bound.label))
}

/// An interval in words: `below 500 lb`, `5_000 lb and above`, `from 500 lb up to 2_000 lb`.
/// On a discrete line an excluded end is named by the last value inside (`..5` is `4 and
/// below`), except on a time line, whose seconds an author rarely writes: `..@08:00` stays
/// `below @08:00`.
fn describe_interval(lo: Option<&Bound>, hi: Option<&Bound>, discrete: bool) -> Message {
    let hi = if discrete && !hi.is_some_and(|bound| time_label(&bound.label)) {
        hi.map(inclusive_end)
    } else {
        hi.cloned()
    };
    match (lo, hi.as_ref()) {
        (None, None) => Message::new(message!("any value")),
        (None, Some(hi)) if hi.inclusive => Message::with(message!("{0} and below"), [label(hi)]),
        (None, Some(hi)) => Message::with(message!("below {0}"), [label(hi)]),
        (Some(lo), None) if lo.inclusive => Message::with(message!("{0} and above"), [label(lo)]),
        (Some(lo), None) => Message::with(message!("above {0}"), [label(lo)]),
        (Some(lo), Some(hi)) if lo.value == hi.value => {
            Message::with(message!("exactly {0}"), [label(lo)])
        }
        (Some(lo), Some(hi)) => match (lo.inclusive, hi.inclusive) {
            (true, true) => Message::with(message!("from {0} to {1}"), [label(lo), label(hi)]),
            (true, false) => Message::with(message!("from {0} up to {1}"), [label(lo), label(hi)]),
            (false, false) => {
                Message::with(message!("between {0} and {1}"), [label(lo), label(hi)])
            }
            (false, true) => Message::with(message!("above {0} up to {1}"), [label(lo), label(hi)]),
        },
    }
}

/// On a discrete line, an exclusive end as the last value inside: `..5` ends at 4.
fn inclusive_end(bound: &Bound) -> Bound {
    if bound.inclusive {
        return bound.clone();
    }
    let value = bound.value.previous();
    Bound {
        label: discrete_label(&value, &bound.label),
        value,
        inclusive: true,
    }
}

/// Whether a bound's label is a time of day (`@17:30`): an `@` literal with a `:` and no `-`.
fn time_label(label: &str) -> bool {
    label.starts_with('@') && label.contains(':') && !label.contains('-')
}

/// A label for a computed whole-number bound: the number, the date for a date line (whose
/// labels start with `@` and hold a `-`), or the time of day for a time line (`@17:30`, with the
/// seconds when they aren't zero).
fn discrete_label(value: &Key, like: &str) -> String {
    let Key::Number(value) = value else {
        return like.to_owned();
    };
    let whole = i64::try_from(value.trunc().mantissa()).unwrap_or_default();
    if time_label(like) {
        let (hour, minute, second) = (whole / 3_600, whole % 3_600 / 60, whole % 60);
        return if second == 0 {
            format!("@{hour:02}:{minute:02}")
        } else {
            format!("@{hour:02}:{minute:02}:{second:02}")
        };
    }
    if like.starts_with('@') {
        let (year, month, day) = super::consts::civil_from_days(whole);
        return format!("@{year:04}-{month:02}-{day:02}");
    }
    value.normalize().to_string()
}

#[cfg(test)]
mod tests {
    use super::{ArmPat, Bound, Domain, Interval, Key, Pat, Space, analyze};
    use rust_decimal::Decimal;

    fn bound(value: i64, inclusive: bool) -> Bound {
        Bound {
            value: Key::Number(Decimal::from(value)),
            inclusive,
            label: format!("{value} lb"),
        }
    }

    fn named(coverage: &super::Coverage) -> Vec<String> {
        coverage
            .missing
            .iter()
            .map(crate::message::Message::english)
            .collect()
    }

    fn time(seconds: i64, inclusive: bool, label: &str) -> Bound {
        Bound {
            value: Key::Number(Decimal::from(seconds)),
            inclusive,
            label: label.to_owned(),
        }
    }

    fn text(value: &str, inclusive: bool) -> Bound {
        Bound {
            value: Key::Text(value.to_owned()),
            inclusive,
            label: format!("\"{value}\""),
        }
    }

    #[test]
    fn a_day_of_times_is_covered_from_midnight_to_the_last_second() {
        let space = Space::Line {
            discrete: true,
            domain: Some(Domain {
                min: Decimal::ZERO,
                max: Decimal::from(86_399),
            }),
        };
        let day = arms(vec![
            Pat::Interval(Interval {
                lo: Some(time(0, true, "@00:00")),
                hi: Some(time(28_800, false, "@08:00")),
            }),
            Pat::Interval(Interval {
                lo: Some(time(28_800, true, "@08:00")),
                hi: Some(time(86_399, true, "@23:59:59")),
            }),
        ]);
        assert!(analyze(&space, &day).missing.is_empty());
        let short = arms(vec![Pat::Interval(Interval {
            lo: Some(time(0, true, "@00:00")),
            hi: Some(time(61_200, false, "@17:00")),
        })]);
        assert_eq!(named(&analyze(&space, &short)), ["`@17:00` and above"]);
        let early = arms(vec![Pat::Interval(Interval {
            lo: Some(time(28_800, true, "@08:00")),
            hi: None,
        })]);
        assert_eq!(named(&analyze(&space, &early)), ["below `@08:00`"]);
    }

    #[test]
    fn text_ranges_cut_a_line_and_literals_are_points_on_it() {
        let zips = arms(vec![
            Pat::Interval(Interval {
                lo: None,
                hi: Some(text("300", false)),
            }),
            Pat::Interval(Interval {
                lo: Some(text("300", true)),
                hi: Some(text("320", false)),
            }),
            Pat::Interval(Interval {
                lo: Some(text("320", true)),
                hi: None,
            }),
        ]);
        let coverage = analyze(&Space::Text, &zips);
        assert!(coverage.missing.is_empty(), "{:?}", coverage.missing);
        let gap = arms(vec![
            Pat::Interval(Interval {
                lo: Some(text("300", true)),
                hi: Some(text("320", false)),
            }),
            Pat::Literal(String::from("310")),
        ]);
        let coverage = analyze(&Space::Text, &gap);
        assert_eq!(coverage.overlaps.len(), 1);
        assert_eq!(
            coverage.overlaps.first().map(|(_, _, m)| m.english()),
            Some(String::from("`310`"))
        );
        assert_eq!(named(&coverage), ["below `\"300\"`", "`\"320\"` and above"]);
        let literals = arms(vec![Pat::Literal(String::from("a"))]);
        assert_eq!(
            named(&analyze(&Space::Text, &literals)),
            ["any other value"]
        );
    }

    fn range(lo: Option<i64>, hi: Option<i64>) -> Pat {
        Pat::Interval(Interval {
            lo: lo.map(|value| bound(value, true)),
            hi: hi.map(|value| bound(value, false)),
        })
    }

    fn arms(pats: Vec<Pat>) -> Vec<ArmPat> {
        pats.into_iter()
            .map(|pat| ArmPat {
                pat,
                guarded: false,
                broken: false,
            })
            .collect()
    }

    fn line() -> Space {
        Space::Line {
            discrete: false,
            domain: None,
        }
    }

    #[test]
    fn weight_breaks_cover_the_line() {
        let coverage = analyze(
            &line(),
            &arms(vec![
                range(None, Some(500)),
                range(Some(500), Some(2000)),
                range(Some(2000), None),
            ]),
        );
        assert!(coverage.missing.is_empty(), "{:?}", coverage.missing);
        assert!(coverage.overlaps.is_empty());
    }

    #[test]
    fn a_missing_top_band_is_named() {
        let coverage = analyze(
            &line(),
            &arms(vec![range(None, Some(500)), range(Some(500), Some(5000))]),
        );
        let named: Vec<String> = coverage
            .missing
            .iter()
            .map(crate::message::Message::english)
            .collect();
        assert_eq!(named, ["`5000 lb` and above"]);
    }

    #[test]
    fn optionals_need_none() {
        let coverage = analyze(
            &Space::Optional(Box::new(line())),
            &arms(vec![range(None, None)]),
        );
        let named: Vec<String> = coverage
            .missing
            .iter()
            .map(crate::message::Message::english)
            .collect();
        assert_eq!(named, ["`none`"]);
    }

    #[test]
    fn overlapping_bands_are_found() {
        let coverage = analyze(
            &line(),
            &arms(vec![range(None, Some(600)), range(Some(500), None)]),
        );
        assert_eq!(coverage.overlaps.len(), 1);
        assert_eq!(
            coverage.overlaps.first().map(|(_, _, m)| m.english()),
            Some(String::from("from `500 lb` up to `600 lb`"))
        );
    }

    #[test]
    fn enums_and_unreachable_arms() {
        let space = Space::Enum(vec!["A".into(), "B".into(), "C".into()]);
        let coverage = analyze(
            &space,
            &arms(vec![
                Pat::Variant("A".into()),
                Pat::Any,
                Pat::Variant("B".into()),
            ]),
        );
        assert!(coverage.missing.is_empty());
        assert_eq!(coverage.unreachable, [2]);
        let coverage = analyze(&space, &arms(vec![Pat::Variant("A".into())]));
        let named: Vec<String> = coverage
            .missing
            .iter()
            .map(crate::message::Message::english)
            .collect();
        assert_eq!(named, ["`B`", "`C`"]);
    }

    #[test]
    fn discrete_lines_have_no_gaps_between_whole_numbers() {
        let space = Space::Line {
            discrete: true,
            domain: None,
        };
        let coverage = analyze(
            &space,
            &arms(vec![
                Pat::Interval(Interval {
                    lo: None,
                    hi: Some(bound(4, true)),
                }),
                range(Some(5), None),
            ]),
        );
        assert!(coverage.missing.is_empty(), "{:?}", coverage.missing);
    }

    #[test]
    fn records_are_covered_field_by_field() {
        let space = Space::Record(vec![("a".into(), Space::Bool), ("b".into(), Space::Bool)]);
        let coverage = analyze(
            &space,
            &arms(vec![
                Pat::Record(vec![("a".into(), Pat::Bool(true))]),
                Pat::Record(vec![("b".into(), Pat::Bool(true))]),
            ]),
        );
        let named: Vec<String> = coverage
            .missing
            .iter()
            .map(crate::message::Message::english)
            .collect();
        assert_eq!(named, ["`{ a: false, b: false }`"]);
    }
}
