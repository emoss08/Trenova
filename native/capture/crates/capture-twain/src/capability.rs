//! Capability containers: what `DAT_CAPABILITY` reads and writes.
//!
//! A capability's value travels in a container the application or the source
//! allocates through the DSM: one value, an enumeration, an array or a range,
//! each tagged with the item type. This reads the four shapes into
//! [`CapValues`] and builds the one-value container every `MSG_SET` here
//! uses. Items are read unaligned, because the structures are packed to two
//! bytes.

use core::ffi::c_void;
use core::mem::offset_of;

use crate::consts::{
    TWON_ARRAY, TWON_ENUMERATION, TWON_ONEVALUE, TWON_RANGE, TWTY_BOOL, TWTY_FIX32, TWTY_INT8,
    TWTY_INT16, TWTY_INT32, TWTY_UINT8, TWTY_UINT16, TWTY_UINT32,
};
use crate::sys::{TW_ARRAY, TW_ENUMERATION, TW_FIX32, TW_ONEVALUE, TW_RANGE};

/// More items than any real capability has; a container claiming more is
/// corrupt and is not read.
const MAX_ITEMS: u32 = 4096;

/// One capability item.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum CapValue {
    Int(i32),
    Uint(u32),
    Bool(bool),
    Fix32(f64),
}

impl CapValue {
    /// The value as a whole number, rounding a fixed-point one.
    #[allow(clippy::cast_possible_truncation)]
    pub fn as_i64(self) -> i64 {
        match self {
            Self::Int(v) => i64::from(v),
            Self::Uint(v) => i64::from(v),
            Self::Bool(v) => i64::from(v),
            Self::Fix32(v) => v.round() as i64,
        }
    }

    pub fn as_f64(self) -> f64 {
        match self {
            Self::Int(v) => f64::from(v),
            Self::Uint(v) => f64::from(v),
            Self::Bool(v) => f64::from(u8::from(v)),
            Self::Fix32(v) => v,
        }
    }

    pub fn as_bool(self) -> bool {
        self.as_i64() != 0
    }
}

/// What a source answered for a capability.
#[derive(Clone, Debug, PartialEq)]
pub enum CapValues {
    One(CapValue),
    List {
        items: Vec<CapValue>,
        current: Option<usize>,
    },
    Range {
        min: f64,
        max: f64,
        step: f64,
        current: f64,
    },
}

impl CapValues {
    /// The value in force.
    pub fn current(&self) -> Option<CapValue> {
        match self {
            Self::One(value) => Some(*value),
            Self::List { items, current } => current.and_then(|i| items.get(i).copied()),
            Self::Range { current, .. } => Some(CapValue::Fix32(*current)),
        }
    }

    /// Whether a value is allowed.
    pub fn allows(&self, value: f64) -> bool {
        match self {
            Self::One(one) => (one.as_f64() - value).abs() < 0.5,
            Self::List { items, .. } => items.iter().any(|v| (v.as_f64() - value).abs() < 0.5),
            Self::Range { min, max, step, .. } => {
                if value < *min - 0.5 || value > *max + 0.5 {
                    return false;
                }
                if *step <= 0.0 {
                    return true;
                }
                let steps = ((value - min) / step).round();
                (min + steps * step - value).abs() < 0.5
            }
        }
    }

    /// Every whole value offered, for a list or a single value.
    pub fn whole_values(&self) -> Vec<i64> {
        match self {
            Self::One(v) => vec![v.as_i64()],
            Self::List { items, .. } => items.iter().map(|v| v.as_i64()).collect(),
            Self::Range { .. } => Vec::new(),
        }
    }
}

/// `TW_FIX32` from a real number, rounded to the nearest 1/65536 the way the
/// TWAIN specification's own sample does.
#[allow(clippy::cast_possible_truncation, clippy::cast_sign_loss)]
pub fn to_fix32(value: f64) -> TW_FIX32 {
    let scaled = (value * 65536.0 + if value < 0.0 { -0.5 } else { 0.5 }) as i32;
    TW_FIX32 {
        Whole: (scaled >> 16) as i16,
        Frac: (scaled & 0xFFFF) as u16,
    }
}

pub fn from_fix32(fix: TW_FIX32) -> f64 {
    f64::from(fix.Whole) + f64::from(fix.Frac) / 65536.0
}

/// The four bytes of a one-value or range slot holding a `TW_FIX32`.
fn fix32_bits(fix: TW_FIX32) -> u32 {
    let whole = u32::from(u16::from_ne_bytes(fix.Whole.to_ne_bytes()));
    whole | (u32::from(fix.Frac) << 16)
}

fn fix32_from_bits(bits: u32) -> TW_FIX32 {
    let [a, b, c, d] = bits.to_le_bytes();
    TW_FIX32 {
        Whole: i16::from_le_bytes([a, b]),
        Frac: u16::from_le_bytes([c, d]),
    }
}

/// Bytes one item of a type takes in a list.
fn item_size(item_type: u16) -> Option<usize> {
    match item_type {
        TWTY_INT8 | TWTY_UINT8 => Some(1),
        TWTY_INT16 | TWTY_UINT16 | TWTY_BOOL => Some(2),
        TWTY_INT32 | TWTY_UINT32 | TWTY_FIX32 => Some(4),
        _ => None,
    }
}

/// Reads one item of a numeric type.
///
/// # Safety
///
/// `ptr` points at `item_size(item_type)` readable bytes.
unsafe fn read_item(item_type: u16, ptr: *const u8) -> Option<CapValue> {
    // SAFETY: the caller guarantees the bytes; reads are unaligned because
    // the containers are packed.
    unsafe {
        Some(match item_type {
            TWTY_INT8 => CapValue::Int(i32::from(ptr.cast::<i8>().read_unaligned())),
            TWTY_UINT8 => CapValue::Uint(u32::from(ptr.read_unaligned())),
            TWTY_INT16 => CapValue::Int(i32::from(ptr.cast::<i16>().read_unaligned())),
            TWTY_UINT16 => CapValue::Uint(u32::from(ptr.cast::<u16>().read_unaligned())),
            TWTY_BOOL => CapValue::Bool(ptr.cast::<u16>().read_unaligned() != 0),
            TWTY_INT32 => CapValue::Int(ptr.cast::<i32>().read_unaligned()),
            TWTY_UINT32 => CapValue::Uint(ptr.cast::<u32>().read_unaligned()),
            TWTY_FIX32 => CapValue::Fix32(from_fix32(ptr.cast::<TW_FIX32>().read_unaligned())),
            _ => return None,
        })
    }
}

/// Reads a one-value or range slot (always 32 bits) as its item type.
#[allow(clippy::cast_possible_truncation, clippy::cast_possible_wrap)]
fn slot_value(item_type: u16, bits: u32) -> Option<CapValue> {
    Some(match item_type {
        TWTY_INT8 => CapValue::Int(i32::from(bits as u8 as i8)),
        TWTY_UINT8 => CapValue::Uint(bits & 0xFF),
        TWTY_INT16 => CapValue::Int(i32::from(bits as u16 as i16)),
        TWTY_UINT16 => CapValue::Uint(bits & 0xFFFF),
        TWTY_BOOL => CapValue::Bool(bits & 0xFFFF != 0),
        TWTY_INT32 => CapValue::Int(bits as i32),
        TWTY_UINT32 => CapValue::Uint(bits),
        TWTY_FIX32 => CapValue::Fix32(from_fix32(fix32_from_bits(bits))),
        _ => return None,
    })
}

/// The 32-bit slot a value is written into a one-value container as.
#[allow(clippy::cast_sign_loss, clippy::cast_possible_truncation)]
fn slot_bits(item_type: u16, value: CapValue) -> u32 {
    match (item_type, value) {
        (TWTY_FIX32, v) => fix32_bits(to_fix32(v.as_f64())),
        (TWTY_INT16, v) => u32::from(v.as_i64() as i16 as u16),
        (TWTY_INT8, v) => u32::from(v.as_i64() as i8 as u8),
        (TWTY_INT32, v) => v.as_i64() as i32 as u32,
        (TWTY_BOOL, v) => u32::from(v.as_bool()),
        (_, v) => v.as_i64() as u32,
    }
}

/// The size of a one-value container.
pub(crate) const ONE_VALUE_SIZE: usize = core::mem::size_of::<TW_ONEVALUE>();

/// Fills a one-value container.
///
/// # Safety
///
/// `ptr` points at [`ONE_VALUE_SIZE`] writable bytes.
pub(crate) unsafe fn write_one_value(ptr: *mut c_void, item_type: u16, value: CapValue) {
    let container = TW_ONEVALUE {
        ItemType: item_type,
        Item: slot_bits(item_type, value),
    };
    // SAFETY: the caller guarantees the space.
    unsafe { ptr.cast::<TW_ONEVALUE>().write_unaligned(container) };
}

/// Reads a container a source filled.
///
/// # Safety
///
/// `ptr` points at a locked container of type `con_type` whose item list is
/// as long as its count says, as TWAIN requires of a source.
pub(crate) unsafe fn decode(con_type: u16, ptr: *const c_void) -> Option<CapValues> {
    let base = ptr.cast::<u8>();
    // SAFETY: every read below is inside the container the caller vouches
    // for, and the item counts are bounded before a list is walked.
    unsafe {
        match con_type {
            TWON_ONEVALUE => {
                let one = ptr.cast::<TW_ONEVALUE>().read_unaligned();
                slot_value(one.ItemType, one.Item).map(CapValues::One)
            }
            TWON_RANGE => {
                let range = ptr.cast::<TW_RANGE>().read_unaligned();
                let value = |bits| slot_value(range.ItemType, bits).map(CapValue::as_f64);
                Some(CapValues::Range {
                    min: value(range.MinValue)?,
                    max: value(range.MaxValue)?,
                    step: value(range.StepSize)?,
                    current: value(range.CurrentValue)?,
                })
            }
            TWON_ENUMERATION => {
                let head = ptr.cast::<TW_ENUMERATION>().read_unaligned();
                let items = read_list(
                    head.ItemType,
                    head.NumItems,
                    base.add(offset_of!(TW_ENUMERATION, ItemList)),
                )?;
                let current = usize::try_from(head.CurrentIndex)
                    .ok()
                    .filter(|&i| i < items.len());
                Some(CapValues::List { items, current })
            }
            TWON_ARRAY => {
                let head = ptr.cast::<TW_ARRAY>().read_unaligned();
                let items = read_list(
                    head.ItemType,
                    head.NumItems,
                    base.add(offset_of!(TW_ARRAY, ItemList)),
                )?;
                Some(CapValues::List {
                    items,
                    current: None,
                })
            }
            _ => None,
        }
    }
}

/// # Safety
///
/// `list` holds `count` items of `item_type`.
unsafe fn read_list(item_type: u16, count: u32, list: *const u8) -> Option<Vec<CapValue>> {
    if count > MAX_ITEMS {
        return None;
    }
    let size = item_size(item_type)?;
    (0..count as usize)
        // SAFETY: bounded by `count`, which the caller vouches for.
        .map(|i| unsafe { read_item(item_type, list.add(i * size)) })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fixed_point_round_trips_resolutions_and_fractions() {
        for value in [300.0, 72.0, 0.5, -2.25, 1200.0] {
            assert!((from_fix32(to_fix32(value)) - value).abs() < 1.0 / 65536.0);
        }
        let fix = to_fix32(300.0);
        assert_eq!((fix.Whole, fix.Frac), (300, 0));
        let fix = to_fix32(-2.25);
        assert!((from_fix32(fix) + 2.25).abs() < f64::EPSILON);
    }

    #[test]
    fn a_one_value_slot_keeps_its_type() {
        let mut container = [0u8; ONE_VALUE_SIZE];
        for (item_type, value, expected) in [
            (TWTY_INT16, CapValue::Int(-1), CapValue::Int(-1)),
            (TWTY_INT32, CapValue::Int(-2), CapValue::Int(-2)),
            (TWTY_BOOL, CapValue::Bool(true), CapValue::Bool(true)),
            (TWTY_UINT16, CapValue::Uint(2), CapValue::Uint(2)),
            (TWTY_FIX32, CapValue::Fix32(300.0), CapValue::Fix32(300.0)),
        ] {
            // SAFETY: the buffer is a one-value container's size.
            unsafe { write_one_value(container.as_mut_ptr().cast(), item_type, value) };
            // SAFETY: just written as a one-value container.
            let read = unsafe { decode(TWON_ONEVALUE, container.as_ptr().cast()) };
            assert_eq!(read, Some(CapValues::One(expected)));
        }
    }

    #[test]
    fn an_enumeration_reads_its_current_item() {
        let mut container = vec![0u8; offset_of!(TW_ENUMERATION, ItemList) + 3 * 4];
        let head = TW_ENUMERATION {
            ItemType: TWTY_FIX32,
            NumItems: 3,
            CurrentIndex: 1,
            DefaultIndex: 1,
            ItemList: [0],
        };
        // SAFETY: the buffer is larger than the header.
        unsafe {
            container
                .as_mut_ptr()
                .cast::<TW_ENUMERATION>()
                .write_unaligned(head);
        };
        for (i, dpi) in [200.0, 300.0, 600.0].into_iter().enumerate() {
            let at = offset_of!(TW_ENUMERATION, ItemList) + i * 4;
            // SAFETY: inside the buffer.
            unsafe {
                container
                    .as_mut_ptr()
                    .add(at)
                    .cast::<TW_FIX32>()
                    .write_unaligned(to_fix32(dpi));
            }
        }
        // SAFETY: a well-formed enumeration of three items.
        let values =
            unsafe { decode(TWON_ENUMERATION, container.as_ptr().cast()) }.expect("values");
        assert_eq!(values.current(), Some(CapValue::Fix32(300.0)));
        assert_eq!(values.whole_values(), vec![200, 300, 600]);
        assert!(values.allows(600.0));
        assert!(!values.allows(400.0));
    }

    #[test]
    fn a_range_allows_its_steps_only() {
        let range = CapValues::Range {
            min: 100.0,
            max: 600.0,
            step: 50.0,
            current: 300.0,
        };
        assert!(range.allows(150.0));
        assert!(!range.allows(175.0));
        assert!(!range.allows(1200.0));
    }

    #[test]
    fn a_corrupt_count_is_not_walked() {
        let head = TW_ARRAY {
            ItemType: TWTY_UINT16,
            NumItems: MAX_ITEMS + 1,
            ItemList: [0],
        };
        // SAFETY: the count is refused before any item is read.
        let read = unsafe { decode(TWON_ARRAY, core::ptr::from_ref(&head).cast()) };
        assert_eq!(read, None);
    }
}
