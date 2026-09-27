//! Reading and writing WIA properties.
//!
//! A WIA property is a `PROPVARIANT` in an `IWiaPropertyStorage`, addressed by
//! its property ID. [`Prop`] owns one that was read and clears it when
//! dropped; writes build theirs on the stack and never clear them, because
//! nothing they point at was allocated for them.

use windows::Win32::Devices::ImageAcquisition::{
    IWiaPropertyStorage, WIA_FLAG_VALUES, WIA_IPA_FIRST, WIA_LIST_COUNT, WIA_LIST_VALUES,
    WIA_PROP_FLAG, WIA_PROP_LIST, WIA_PROP_RANGE, WIA_RANGE_MAX, WIA_RANGE_MIN, WIA_RANGE_STEP,
};
use windows::Win32::System::Com::StructuredStorage::{
    PROPSPEC, PROPSPEC_0, PROPVARIANT, PRSPEC_PROPID, PropVariantClear,
};
use windows::Win32::System::Variant::{
    VARENUM, VT_BSTR, VT_CLSID, VT_I2, VT_I4, VT_INT, VT_LPWSTR, VT_UI2, VT_UI4, VT_UINT, VT_VECTOR,
};
use windows_core::GUID;

pub(crate) use crate::allowed::Allowed;

/// More values than any property lists; a longer vector is not read.
const MAX_VALUES: u32 = 1024;

fn spec(id: u32) -> PROPSPEC {
    PROPSPEC {
        ulKind: PRSPEC_PROPID,
        Anonymous: PROPSPEC_0 { propid: id },
    }
}

/// A property value that was read, cleared when dropped.
pub(crate) struct Prop(PROPVARIANT);

impl Drop for Prop {
    fn drop(&mut self) {
        // SAFETY: filled by WIA, cleared once.
        unsafe {
            let _ = PropVariantClear(&raw mut self.0);
        }
    }
}

impl Prop {
    fn vt(&self) -> VARENUM {
        // SAFETY: `vt` is valid in every PROPVARIANT.
        unsafe { self.0.Anonymous.Anonymous.vt }
    }

    pub(crate) fn as_i32(&self) -> Option<i32> {
        // SAFETY: each member is read only when `vt` says it is the one set.
        unsafe {
            let value = &self.0.Anonymous.Anonymous.Anonymous;
            match self.vt() {
                VT_I4 | VT_INT => Some(value.lVal),
                VT_UI4 | VT_UINT => i32::try_from(value.ulVal).ok(),
                VT_I2 => Some(i32::from(value.iVal)),
                VT_UI2 => Some(i32::from(value.uiVal)),
                _ => None,
            }
        }
    }

    pub(crate) fn as_string(&self) -> Option<String> {
        // SAFETY: as above; WIA's strings are NUL-terminated.
        unsafe {
            let value = &self.0.Anonymous.Anonymous.Anonymous;
            match self.vt() {
                VT_BSTR => Some(value.bstrVal.to_string()),
                VT_LPWSTR if !value.pwszVal.is_null() => value.pwszVal.to_string().ok(),
                _ => None,
            }
        }
    }

    /// A vector of 32-bit integers, which is how WIA lists valid values.
    pub(crate) fn as_i32_vector(&self) -> Option<Vec<i32>> {
        let vt = self.vt();
        if vt.0 & VT_VECTOR.0 == 0 {
            return None;
        }
        let element = VARENUM(vt.0 & !VT_VECTOR.0);
        // SAFETY: as above; the vector holds `cElems` elements.
        unsafe {
            let value = &self.0.Anonymous.Anonymous.Anonymous;
            match element {
                VT_I4 | VT_UI4 => {
                    let list = value.cal;
                    if list.pElems.is_null() || list.cElems > MAX_VALUES {
                        return None;
                    }
                    Some(core::slice::from_raw_parts(list.pElems, list.cElems as usize).to_vec())
                }
                _ => None,
            }
        }
    }
}

/// Reads one property.
pub(crate) fn read(storage: &IWiaPropertyStorage, id: u32) -> Option<Prop> {
    let spec = spec(id);
    let mut value = PROPVARIANT::default();
    // SAFETY: one spec and one value, as the count says.
    unsafe { storage.ReadMultiple(1, &raw const spec, &raw mut value) }.ok()?;
    let prop = Prop(value);
    // SAFETY: `vt` is valid in every PROPVARIANT.
    let empty = unsafe { prop.0.Anonymous.Anonymous.vt.0 } == 0;
    (!empty).then_some(prop)
}

pub(crate) fn read_i32(storage: &IWiaPropertyStorage, id: u32) -> Option<i32> {
    read(storage, id).and_then(|p| p.as_i32())
}

pub(crate) fn read_string(storage: &IWiaPropertyStorage, id: u32) -> Option<String> {
    read(storage, id).and_then(|p| p.as_string())
}

fn write(storage: &IWiaPropertyStorage, id: u32, value: &PROPVARIANT) -> bool {
    let spec = spec(id);
    // SAFETY: one spec and one value, as the count says; the value is only
    // read for the duration of the call.
    unsafe { storage.WriteMultiple(1, &raw const spec, value, WIA_IPA_FIRST) }.is_ok()
}

/// Sets a 32-bit integer property, reporting whether the driver took it.
pub(crate) fn write_i32(storage: &IWiaPropertyStorage, id: u32, value: i32) -> bool {
    let mut variant = PROPVARIANT::default();
    // SAFETY: setting the tag and the matching member of a zeroed value.
    unsafe {
        let inner = &mut *variant.Anonymous.Anonymous;
        inner.vt = VT_I4;
        inner.Anonymous.lVal = value;
    }
    write(storage, id, &variant)
}

/// Sets a GUID property, as the transfer format is.
pub(crate) fn write_guid(storage: &IWiaPropertyStorage, id: u32, value: GUID) -> bool {
    let mut guid = value;
    let mut variant = PROPVARIANT::default();
    // SAFETY: the value points at `guid`, which outlives the write and is
    // never freed through it.
    unsafe {
        let inner = &mut *variant.Anonymous.Anonymous;
        inner.vt = VT_CLSID;
        inner.Anonymous.puuid = &raw mut guid;
    }
    write(storage, id, &variant)
}

/// Reads what a property allows.
pub(crate) fn allowed(storage: &IWiaPropertyStorage, id: u32) -> Allowed {
    let spec = spec(id);
    let mut flags = 0u32;
    let mut value = PROPVARIANT::default();
    // SAFETY: one spec, one flag and one value, as the count says.
    if unsafe { storage.GetPropertyAttributes(1, &raw const spec, &raw mut flags, &raw mut value) }
        .is_err()
    {
        return Allowed::Unknown;
    }
    let attributes = Prop(value);
    let Some(vector) = attributes.as_i32_vector() else {
        return Allowed::Unknown;
    };
    let at = |i: u32| vector.get(i as usize).copied();

    if flags & WIA_PROP_LIST != 0 {
        let count = at(WIA_LIST_COUNT)
            .and_then(|c| usize::try_from(c).ok())
            .unwrap_or(0);
        let start = WIA_LIST_VALUES as usize;
        let values = vector.get(start..start + count.min(vector.len().saturating_sub(start)));
        return values.map_or(Allowed::Unknown, |v| Allowed::List(v.to_vec()));
    }
    if flags & WIA_PROP_RANGE != 0 {
        return match (at(WIA_RANGE_MIN), at(WIA_RANGE_MAX), at(WIA_RANGE_STEP)) {
            (Some(min), Some(max), Some(step)) => Allowed::Range { min, max, step },
            _ => Allowed::Unknown,
        };
    }
    if flags & WIA_PROP_FLAG != 0 {
        return at(WIA_FLAG_VALUES).map_or(Allowed::Unknown, Allowed::Flags);
    }
    Allowed::Unknown
}
