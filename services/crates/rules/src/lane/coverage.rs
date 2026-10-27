//! Lanes that will never fire, found before anybody wonders why a rate did not apply.
//!
//! The engine orders matching rules by priority (higher first), then agreement priority, then
//! specificity, then the most recent effective date, then the ID. Two active lanes that reach a
//! shipment by the same key with the same specificity, during overlapping effective windows,
//! therefore never both apply:
//!
//! - when their priorities differ, the lower-priority lane is **shadowed**: it can never win;
//! - when they are equal, the later one in the list is a **duplicate**: the winner is settled by
//!   a tie-break, not by the author.
//!
//! A bidirectional lane is reachable by its reverse key too, so it meets the lanes written the
//! other way round. Lanes matched by radius are not judged: whether two circles overlap cannot
//! be told from the fields alone. Each lane gets at most one issue, and `other` names the lane
//! it collides with (the first, by position).
//!
//! The editor runs this as lanes are typed, and the server can run it over an import. Lanes
//! are bucketed by key and specificity, so only lanes that share a bucket are compared.

use crate::lane::rule::RuleInput;
use std::collections::HashMap;
use trenova_module_kit::message;
use trenova_module_kit::message::Message;

/// Why a lane will never apply.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum IssueKind {
    /// An identical lane with the same priority leaves the winner to a tie-break.
    Duplicate,
    /// An identical lane with a higher priority always wins.
    Shadowed,
}

impl IssueKind {
    /// The wire name.
    #[must_use]
    pub const fn name(self) -> &'static str {
        match self {
            Self::Duplicate => "duplicate",
            Self::Shadowed => "shadowed",
        }
    }
}

/// One lane that will never apply.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Issue {
    /// The lane's position.
    pub index: usize,
    /// Why.
    pub kind: IssueKind,
    /// The lane it collides with.
    pub other: usize,
    /// What to tell the author.
    pub message: Message,
}

struct Lane {
    index: usize,
    priority: i64,
    from: i64,
    to: Option<i64>,
}

fn overlaps(left: &Lane, right: &Lane) -> bool {
    let starts_before_right_ends = right.to.is_none_or(|to| left.from < to);
    let starts_before_left_ends = left.to.is_none_or(|to| right.from < to);
    starts_before_right_ends && starts_before_left_ends
}

/// Finds the lanes that will never apply, ordered by position.
#[must_use]
pub fn find_issues(rules: &[RuleInput]) -> Vec<Issue> {
    let mut buckets: HashMap<(String, i64), Vec<usize>> = HashMap::new();
    let lanes: Vec<Lane> = rules
        .iter()
        .enumerate()
        .map(|(index, rule)| Lane {
            index,
            priority: rule.priority.unwrap_or_default(),
            from: rule.effective_from.unwrap_or_default(),
            to: rule.effective_to,
        })
        .collect();
    for (index, rule) in rules.iter().enumerate() {
        if !rule.is_active() {
            continue;
        }
        let keys = rule.lane_keys();
        let Some(key) = keys.key else {
            continue;
        };
        let specificity = rule.specificity();
        let mut reachable = vec![key];
        if let Some(reverse) = keys.reverse
            && !reachable.contains(&reverse)
        {
            reachable.push(reverse);
        }
        for key in reachable {
            buckets.entry((key, specificity)).or_default().push(index);
        }
    }

    let mut found: HashMap<usize, (IssueKind, usize)> = HashMap::new();
    for members in buckets.values() {
        for &index in members {
            let lane = &lanes[index];
            let rivals = members
                .iter()
                .map(|&other| &lanes[other])
                .filter(|other| other.index != index && overlaps(lane, other));
            let mut shadowed_by = None;
            let mut duplicate_of = None;
            for rival in rivals {
                if rival.priority > lane.priority {
                    shadowed_by =
                        Some(shadowed_by.map_or(rival.index, |seen: usize| seen.min(rival.index)));
                } else if rival.priority == lane.priority && rival.index < index {
                    duplicate_of =
                        Some(duplicate_of.map_or(rival.index, |seen: usize| seen.min(rival.index)));
                }
            }
            let issue = shadowed_by
                .map(|other| (IssueKind::Shadowed, other))
                .or_else(|| duplicate_of.map(|other| (IssueKind::Duplicate, other)));
            if let Some(issue) = issue {
                found
                    .entry(index)
                    .and_modify(|current| {
                        if rank(issue) < rank(*current) {
                            *current = issue;
                        }
                    })
                    .or_insert(issue);
            }
        }
    }

    let mut issues: Vec<Issue> = found
        .into_iter()
        .map(|(index, (kind, other))| Issue {
            index,
            kind,
            other,
            message: Message::new(match kind {
                IssueKind::Duplicate => message!(
                    "This lane is written exactly as narrowly as another one, so which of the two applies is decided by a tie-break rather than by you."
                ),
                IssueKind::Shadowed => message!(
                    "Another lane covers the same geography with a higher priority, so this one can never apply."
                ),
            }),
        })
        .collect();
    issues.sort_by_key(|issue| issue.index);
    issues
}

/// Shadowed outranks duplicate; then the earlier rival.
fn rank((kind, other): (IssueKind, usize)) -> (u8, usize) {
    (
        match kind {
            IssueKind::Shadowed => 0,
            IssueKind::Duplicate => 1,
        },
        other,
    )
}
