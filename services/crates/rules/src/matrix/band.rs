//! Rate matrix bands (`RateMatrixCell.ContainsQuantity`).
//!
//! A band is half open, `[min, max)`: a shipment weighing exactly the upper bound belongs to the
//! band above, which is how tariffs are written ("1000 to 2000" means 2000 lb rates at the next
//! tier) and the only reading under which two adjacent bands cannot both claim a boundary. A
//! missing bound is no bound: no `min` reaches down forever, no `max` up forever.

use rust_decimal::Decimal;

/// Whether `value` falls in the band `[min, max)`.
#[must_use]
pub fn contains(min: Option<Decimal>, max: Option<Decimal>, value: Decimal) -> bool {
    if min.is_some_and(|min| value < min) {
        return false;
    }
    if max.is_some_and(|max| value >= max) {
        return false;
    }
    true
}

#[cfg(test)]
mod tests {
    use super::contains;
    use rust_decimal::Decimal;

    #[test]
    fn half_open_with_open_ends() {
        let thousand = Decimal::from(1000);
        let two = Decimal::from(2000);
        assert!(contains(Some(thousand), Some(two), thousand));
        assert!(!contains(Some(thousand), Some(two), two));
        assert!(contains(None, Some(two), Decimal::from(-5)));
        assert!(contains(Some(thousand), None, Decimal::from(48_000)));
        assert!(contains(None, None, Decimal::ZERO));
    }
}
