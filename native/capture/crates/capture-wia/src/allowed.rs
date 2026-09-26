//! What a WIA property allows, and choosing within it.

/// What a property allows.
#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) enum Allowed {
    List(Vec<i32>),
    Range {
        min: i32,
        max: i32,
        step: i32,
    },
    /// A bit mask of the flags that may be set.
    Flags(i32),
    Unknown,
}

impl Allowed {
    /// The allowed value nearest `wanted`.
    pub(crate) fn nearest(&self, wanted: i32) -> i32 {
        match self {
            Self::List(values) => values
                .iter()
                .copied()
                .min_by_key(|v| (i64::from(*v) - i64::from(wanted)).abs())
                .unwrap_or(wanted),
            Self::Range { min, max, step } => {
                let clamped = wanted.clamp(*min, *max);
                if *step <= 0 {
                    return clamped;
                }
                let steps = (clamped - min + step / 2) / step;
                (min + steps * step).min(*max)
            }
            Self::Flags(_) | Self::Unknown => wanted,
        }
    }

    /// The whole values offered, for a list or the standard resolutions a
    /// range contains.
    pub(crate) fn values(&self, standard: &[i32]) -> Vec<i32> {
        match self {
            Self::List(values) => values.clone(),
            Self::Range { min, max, step } => standard
                .iter()
                .copied()
                .filter(|v| v >= min && v <= max && (*step <= 1 || (v - min) % step == 0))
                .collect(),
            Self::Flags(_) | Self::Unknown => Vec::new(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn the_nearest_listed_value_is_chosen() {
        let list = Allowed::List(vec![150, 300, 600]);
        assert_eq!(list.nearest(400), 300);
        assert_eq!(list.nearest(500), 600);
        assert_eq!(list.nearest(300), 300);
    }

    #[test]
    fn a_range_is_clamped_and_snapped_to_its_step() {
        let range = Allowed::Range {
            min: 100,
            max: 600,
            step: 50,
        };
        assert_eq!(range.nearest(1200), 600);
        assert_eq!(range.nearest(20), 100);
        assert_eq!(range.nearest(320), 300);
        assert_eq!(range.nearest(330), 350);
        let odd = Allowed::Range {
            min: 75,
            max: 1200,
            step: 1,
        };
        assert_eq!(odd.nearest(300), 300);
    }

    #[test]
    fn a_range_offers_only_standard_resolutions_it_contains() {
        let range = Allowed::Range {
            min: 100,
            max: 400,
            step: 100,
        };
        assert_eq!(
            range.values(&[100, 150, 200, 300, 600]),
            vec![100, 200, 300]
        );
        assert_eq!(Allowed::Unknown.values(&[300]), Vec::<i32>::new());
    }
}
