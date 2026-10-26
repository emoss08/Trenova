//! X12 syntax notes: the relational conditions between elements of one segment.
//!
//! A note is a letter and two or more two-digit element positions: `P0304` says elements 03 and
//! 04 appear together or not at all. The five letters are the standard's:
//!
//! | Letter | Name | Meaning |
//! |---|---|---|
//! | `P` | paired | if any of them is present, all are |
//! | `R` | required | at least one of them is present |
//! | `E` | exclusion | at most one of them is present |
//! | `C` | conditional | if the first is present, all the others are |
//! | `L` | list conditional | if the first is present, at least one of the others is |

/// Which relation a note states.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Relation {
    /// `P`: all or none.
    Paired,
    /// `R`: at least one.
    Required,
    /// `E`: at most one.
    Exclusion,
    /// `C`: if the first, then all the others.
    Conditional,
    /// `L`: if the first, then at least one other.
    ListConditional,
}

/// One parsed syntax note.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SyntaxNote {
    /// The relation.
    pub relation: Relation,
    /// The element positions, one-based, in the order written.
    pub positions: Vec<u32>,
}

impl SyntaxNote {
    /// Parses `P0304`, or `None` when it is not a syntax note.
    #[must_use]
    pub fn parse(text: &str) -> Option<Self> {
        let mut chars = text.chars();
        let relation = match chars.next()? {
            'P' => Relation::Paired,
            'R' => Relation::Required,
            'E' => Relation::Exclusion,
            'C' => Relation::Conditional,
            'L' => Relation::ListConditional,
            _ => return None,
        };
        let digits = chars.as_str();
        if digits.len() < 4
            || !digits.len().is_multiple_of(2)
            || !digits.bytes().all(|byte| byte.is_ascii_digit())
        {
            return None;
        }
        let positions: Vec<u32> = digits
            .as_bytes()
            .chunks(2)
            .filter_map(|pair| std::str::from_utf8(pair).ok()?.parse().ok())
            .collect();
        if positions.contains(&0) {
            return None;
        }
        Some(Self {
            relation,
            positions,
        })
    }

    /// What the note asks of the elements present: the positions that should be present but
    /// are not, and the positions present that the note excludes.
    #[must_use]
    pub fn violations(&self, present: impl Fn(u32) -> bool) -> Violations {
        let present_positions: Vec<u32> = self
            .positions
            .iter()
            .copied()
            .filter(|position| present(*position))
            .collect();
        let absent: Vec<u32> = self
            .positions
            .iter()
            .copied()
            .filter(|position| !present_positions.contains(position))
            .collect();
        let first_present = self.positions.first().is_some_and(|first| present(*first));
        match self.relation {
            Relation::Paired if !present_positions.is_empty() => Violations {
                missing: absent,
                excluded: Vec::new(),
            },
            Relation::Required if present_positions.is_empty() => Violations {
                missing: self.positions.first().copied().into_iter().collect(),
                excluded: Vec::new(),
            },
            Relation::Exclusion if present_positions.len() > 1 => Violations {
                missing: Vec::new(),
                excluded: present_positions[1..].to_vec(),
            },
            Relation::Conditional if first_present => Violations {
                missing: absent,
                excluded: Vec::new(),
            },
            Relation::ListConditional if first_present && present_positions.len() == 1 => {
                Violations {
                    missing: self.positions.get(1).copied().into_iter().collect(),
                    excluded: Vec::new(),
                }
            }
            _ => Violations::default(),
        }
    }
}

/// What a syntax note found wrong.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Violations {
    /// Positions the note requires that are absent.
    pub missing: Vec<u32>,
    /// Positions present that the note excludes.
    pub excluded: Vec<u32>,
}

#[cfg(test)]
mod tests {
    use super::{Relation, SyntaxNote};

    fn check(note: &str, present: &[u32]) -> (Vec<u32>, Vec<u32>) {
        let note = SyntaxNote::parse(note).expect("a note");
        let found = note.violations(|position| present.contains(&position));
        (found.missing, found.excluded)
    }

    #[test]
    fn parses_notes_and_refuses_anything_else() {
        let note = SyntaxNote::parse("L010203").expect("a note");
        assert_eq!(note.relation, Relation::ListConditional);
        assert_eq!(note.positions, [1, 2, 3]);
        for bad in ["", "P", "P03", "P030", "Q0304", "P03x4", "P0003"] {
            assert!(SyntaxNote::parse(bad).is_none(), "{bad}");
        }
    }

    #[test]
    fn checks_each_relation() {
        assert_eq!(check("P0304", &[3]), (vec![4], vec![]));
        assert_eq!(check("P0304", &[]), (vec![], vec![]));
        assert_eq!(check("R0203", &[]), (vec![2], vec![]));
        assert_eq!(check("R0203", &[3]), (vec![], vec![]));
        assert_eq!(check("E0203", &[2, 3]), (vec![], vec![3]));
        assert_eq!(check("C0605", &[6]), (vec![5], vec![]));
        assert_eq!(check("C0605", &[5]), (vec![], vec![]));
        assert_eq!(check("L010203", &[1]), (vec![2], vec![]));
        assert_eq!(check("L010203", &[1, 3]), (vec![], vec![]));
    }
}
