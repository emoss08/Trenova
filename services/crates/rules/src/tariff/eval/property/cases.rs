//! Which combinations of the axes' values a property runs, at most [`MAX_CASES`].
//!
//! - When every combination fits, all of them, the first axis outermost.
//! - Otherwise each axis's marked values (its ends, landmarks and listed items) against the
//!   other axes' ends, then combinations drawn from a generator seeded by the property's name
//!   ([`seed`], FNV-1a), until there are [`MAX_CASES`]. The same file always gives the same
//!   cases.

use std::collections::BTreeSet;

/// The most cases one property runs.
pub const MAX_CASES: usize = 256;

/// How many draws the generator gets to find new combinations.
const DRAWS: usize = MAX_CASES * 16;

/// A property's seed: the FNV-1a hash of its name.
#[must_use]
pub fn seed(name: &str) -> u64 {
    let mut hash: u64 = 0xcbf2_9ce4_8422_2325;
    for byte in name.bytes() {
        hash ^= u64::from(byte);
        hash = hash.wrapping_mul(0x0100_0000_01b3);
    }
    hash
}

/// `SplitMix64`: a small, fast, deterministic generator.
#[derive(Debug, Clone)]
struct Generator(u64);

impl Generator {
    fn next(&mut self) -> u64 {
        self.0 = self.0.wrapping_add(0x9e37_79b9_7f4a_7c15);
        let mut mixed = self.0;
        mixed = (mixed ^ (mixed >> 30)).wrapping_mul(0xbf58_476d_1ce4_e5b9);
        mixed = (mixed ^ (mixed >> 27)).wrapping_mul(0x94d0_49bb_1331_11eb);
        mixed ^ (mixed >> 31)
    }

    fn below(&mut self, bound: usize) -> usize {
        let bound = u64::try_from(bound.max(1)).unwrap_or(u64::MAX);
        usize::try_from(self.next() % bound).unwrap_or(0)
    }
}

/// The cases, as one value index per axis. `marked[axis][index]` says whether a value is one
/// of the axis's ends, landmarks or listed items.
#[must_use]
pub fn combinations(marked: &[Vec<bool>], seed: u64) -> Vec<Vec<usize>> {
    let sizes: Vec<usize> = marked.iter().map(Vec::len).collect();
    if sizes.is_empty() || sizes.contains(&0) {
        return Vec::new();
    }
    let total = sizes
        .iter()
        .try_fold(1usize, |product, size| product.checked_mul(*size));
    if total.is_some_and(|total| total <= MAX_CASES) {
        return every(&sizes);
    }
    let mut known: BTreeSet<Vec<usize>> = BTreeSet::new();
    let mut out: Vec<Vec<usize>> = Vec::new();
    let mut add = |case: Vec<usize>, out: &mut Vec<Vec<usize>>| {
        if out.len() < MAX_CASES && known.insert(case.clone()) {
            out.push(case);
        }
    };
    for (axis, flags) in marked.iter().enumerate() {
        for (index, _) in flags.iter().enumerate().filter(|(_, marked)| **marked) {
            for ends in corners(&sizes, axis) {
                let mut case = ends;
                if let Some(slot) = case.get_mut(axis) {
                    *slot = index;
                }
                add(case, &mut out);
            }
        }
    }
    let mut generator = Generator(seed);
    for _ in 0..DRAWS {
        if out.len() >= MAX_CASES {
            break;
        }
        let case: Vec<usize> = sizes.iter().map(|size| generator.below(*size)).collect();
        add(case, &mut out);
    }
    out
}

/// Every combination, the first axis outermost.
fn every(sizes: &[usize]) -> Vec<Vec<usize>> {
    let mut out = vec![Vec::new()];
    for size in sizes {
        let mut next = Vec::with_capacity(out.len() * size);
        for prefix in &out {
            for index in 0..*size {
                let mut case = prefix.clone();
                case.push(index);
                next.push(case);
            }
        }
        out = next;
    }
    out
}

/// Every combination of the other axes' ends (first and last values), `axis` left at 0.
fn corners(sizes: &[usize], axis: usize) -> Vec<Vec<usize>> {
    let mut out = vec![Vec::new()];
    for (position, size) in sizes.iter().enumerate() {
        let ends: Vec<usize> = if position == axis || *size <= 1 {
            vec![0]
        } else {
            vec![0, size - 1]
        };
        let mut next = Vec::with_capacity(out.len() * ends.len());
        for prefix in &out {
            for end in &ends {
                let mut case = prefix.clone();
                case.push(*end);
                next.push(case);
            }
        }
        out = next;
        if out.len() > MAX_CASES {
            out.truncate(MAX_CASES);
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::{MAX_CASES, combinations, seed};

    #[test]
    fn small_products_take_every_combination() {
        let cases = combinations(&[vec![true; 2], vec![true; 3]], 1);
        assert_eq!(
            cases,
            [[0, 0], [0, 1], [0, 2], [1, 0], [1, 1], [1, 2]].map(|case| case.to_vec())
        );
    }

    #[test]
    fn large_products_are_capped_and_repeat() {
        let marked = vec![vec![true, false, false, true]; 5];
        let first = combinations(&marked, seed("p"));
        assert_eq!(first.len(), MAX_CASES);
        assert_eq!(first, combinations(&marked, seed("p")));
        assert_ne!(first, combinations(&marked, seed("q")));
    }

    #[test]
    fn the_seed_is_fnv1a() {
        assert_eq!(seed(""), 0xcbf2_9ce4_8422_2325);
        assert_eq!(seed("a"), 0xaf63_dc4c_8601_ec8c);
    }
}
