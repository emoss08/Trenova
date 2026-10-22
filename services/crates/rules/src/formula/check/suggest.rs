//! "Did you mean" suggestions and the type a new input starts as.

use super::InputType;

/// The edit distance between two names, or `usize::MAX` when their lengths differ by more than
/// three (no suggestion would be close enough).
fn levenshtein(a: &[char], b: &[char]) -> usize {
    if a.len().abs_diff(b.len()) > 3 {
        return usize::MAX;
    }
    let mut previous: Vec<usize> = (0..=b.len()).collect();
    let mut current = vec![0; b.len() + 1];
    for (i, left) in a.iter().enumerate() {
        current[0] = i + 1;
        for (j, right) in b.iter().enumerate() {
            let cost = usize::from(left != right);
            current[j + 1] = (previous[j + 1] + 1)
                .min(current[j] + 1)
                .min(previous[j] + cost);
        }
        std::mem::swap(&mut previous, &mut current);
    }
    previous[b.len()]
}

/// Names longer than this get no suggestion, so a suggestion never costs more than a bounded
/// comparison per candidate, however long the text.
const LONGEST_SUGGESTED: usize = 64;

/// The closest known name, when one is within two edits or differs only in case. Ties go to the
/// first candidate.
#[must_use]
pub fn nearest_known<'a>(
    name: &str,
    candidates: impl IntoIterator<Item = &'a str>,
) -> Option<&'a str> {
    if name.len() > LONGEST_SUGGESTED * 4 {
        return None;
    }
    let lower: Vec<char> = name.to_lowercase().chars().collect();
    if lower.len() > LONGEST_SUGGESTED {
        return None;
    }
    let mut best = None;
    let mut best_distance = 3;
    for candidate in candidates {
        if candidate == name || candidate.len() > (LONGEST_SUGGESTED + 3) * 4 {
            continue;
        }
        let other: Vec<char> = candidate.to_lowercase().chars().collect();
        let distance = if other == lower {
            0
        } else {
            levenshtein(&lower, &other)
        };
        if distance < best_distance {
            best = Some(candidate);
            best_distance = distance;
        }
    }
    best
}

/// The type an input declared from an unknown name starts as: a flag for a name such as
/// `isWeekend` or `hasLiftgate`, a number otherwise.
#[must_use]
pub fn infer_input_type(name: &str) -> InputType {
    const FLAGS: &[&str] = &["is", "has", "requires", "uses", "needs", "allows", "can"];
    let flag = FLAGS.iter().any(|prefix| {
        name.strip_prefix(prefix)
            .and_then(|rest| rest.chars().next())
            .is_some_and(|next| next.is_ascii_uppercase() || next == '_')
    });
    if flag {
        InputType::Boolean
    } else {
        InputType::Number
    }
}
