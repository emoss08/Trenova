//! A scripted DSM with one or more sources, for the session tests.
//!
//! It answers the triplets the session uses the way the TWAIN specification
//! says a DSM and a source do: it hands out containers the application must
//! free, fills strips into the application's buffer at the offsets it names,
//! reports conditions through `DAT_STATUS`, and records every call so a test
//! can check the order the state machine was walked in. Allocations are
//! counted, so a leaked container or barcode handle fails the test.

#![allow(
    clippy::cast_possible_truncation,
    clippy::cast_sign_loss,
    clippy::cast_possible_wrap,
    clippy::cast_ptr_alignment,
    clippy::match_same_arms,
    clippy::wildcard_imports
)]

use core::ffi::c_void;
use std::cell::RefCell;
use std::collections::HashMap;

use crate::consts::*;
use crate::dsm::Dsm;
use crate::pump::{EventPump, Processed, PumpEvent};
use crate::sys::{
    TW_CAPABILITY, TW_ENTRYPOINT, TW_ENUMERATION, TW_EVENT, TW_EXTIMAGEINFO, TW_HANDLE,
    TW_IDENTITY, TW_IMAGEINFO, TW_IMAGEMEMXFER, TW_INFO, TW_ONEVALUE, TW_PENDINGXFERS,
    TW_SETUPMEMXFER, TW_STATUS,
};
use crate::text::{decode_str, encode_str32};

/// A capability as the fake source holds it.
#[derive(Clone, Debug)]
pub struct FakeCap {
    pub item_type: u16,
    pub current: u32,
    /// What `MSG_GET` lists; empty answers with the current value alone.
    pub allowed: Vec<u32>,
    /// Refuse every `MSG_SET`.
    pub read_only: bool,
}

impl FakeCap {
    pub fn new(item_type: u16, current: u32) -> Self {
        Self {
            item_type,
            current,
            allowed: Vec::new(),
            read_only: false,
        }
    }

    pub fn allowing(mut self, allowed: &[u32]) -> Self {
        self.allowed = allowed.to_vec();
        self
    }

    pub fn read_only(mut self) -> Self {
        self.read_only = true;
        self
    }
}

/// One page the fake scanner will produce.
#[derive(Clone, Debug)]
pub struct FakePage {
    pub width: u32,
    pub pixel_type: u16,
    pub bits_per_pixel: i16,
    /// Packed rows, top first.
    pub rows: Vec<Vec<u8>>,
    pub patch: Option<u32>,
    pub barcodes: Vec<String>,
}

#[derive(Clone, Debug, Default)]
pub struct FakeSource {
    pub name: String,
    pub caps: HashMap<u16, FakeCap>,
    pub pages: Vec<FakePage>,
    /// Rows per strip.
    pub strip_rows: usize,
    /// Padding after each row in a strip.
    pub row_padding: usize,
    /// Report `ImageLength` as -1, as a feeder with length detection does.
    pub unknown_length: bool,
    /// Fail the transfer of page `n` (0-based) after its first strip.
    pub fail_page: Option<(usize, u16)>,
}

#[derive(Debug, Default)]
struct State {
    dsm_open: bool,
    open: Option<usize>,
    enabled: bool,
    ready_sent: bool,
    page: usize,
    row: usize,
    pending_reset: bool,
    condition: u16,
    enumerate_at: usize,
    live_allocations: usize,
    calls: Vec<(u16, u16)>,
    set_values: HashMap<(usize, u16), u32>,
}

#[derive(Debug)]
pub struct FakeDsm {
    pub sources: Vec<FakeSource>,
    pub default: usize,
    state: RefCell<State>,
}

impl FakeDsm {
    pub fn new(sources: Vec<FakeSource>) -> Self {
        Self {
            sources,
            default: 0,
            state: RefCell::new(State::default()),
        }
    }

    pub fn calls(&self) -> Vec<(u16, u16)> {
        self.state.borrow().calls.clone()
    }

    pub fn live_allocations(&self) -> usize {
        self.state.borrow().live_allocations
    }

    /// A capability's value on the source last opened, after any sets.
    pub fn cap(&self, index: usize, cap: u16) -> Option<u32> {
        let state = self.state.borrow();
        state
            .set_values
            .get(&(index, cap))
            .copied()
            .or_else(|| self.sources[index].caps.get(&cap).map(|c| c.current))
    }

    fn fail(&self, condition: u16) -> u16 {
        self.state.borrow_mut().condition = condition;
        TWRC_FAILURE
    }

    fn identity_of(&self, index: usize) -> TW_IDENTITY {
        // SAFETY: plain data.
        let mut identity: TW_IDENTITY = unsafe { core::mem::zeroed() };
        identity.Id = u32::try_from(index + 1).expect("id");
        identity.ProductName = encode_str32(&self.sources[index].name);
        identity.Manufacturer = encode_str32("Fake");
        identity.Version.Info = encode_str32("9.1.0");
        identity.SupportedGroups = DG_CONTROL | DG_IMAGE | DF_DSM2;
        identity
    }

    fn page(&self) -> Option<&FakePage> {
        let state = self.state.borrow();
        self.sources[state.open?].pages.get(state.page)
    }

    fn pages_left(&self) -> usize {
        let state = self.state.borrow();
        state.open.map_or(0, |i| {
            self.sources[i].pages.len().saturating_sub(state.page)
        })
    }

    /// Allocates a container of `size` bytes the application will free.
    fn container(&self, size: usize, fill: impl FnOnce(*mut u8)) -> TW_HANDLE {
        let handle = self.alloc(u32::try_from(size).expect("size"));
        // SAFETY: just allocated with `size` bytes.
        unsafe {
            let ptr = self.lock(handle).cast::<u8>();
            fill(ptr);
            self.unlock(handle);
        }
        handle
    }

    fn capability(&self, msg: u16, request: &mut TW_CAPABILITY) -> u16 {
        let Some(index) = self.state.borrow().open else {
            return self.fail(TWCC_SEQERROR);
        };
        let cap_id = request.Cap;
        let Some(mut cap) = self.sources[index].caps.get(&cap_id).cloned() else {
            return self.fail(TWCC_CAPUNSUPPORTED);
        };
        if let Some(value) = self.state.borrow().set_values.get(&(index, cap_id)) {
            cap.current = *value;
        }
        match msg {
            MSG_GET if !cap.allowed.is_empty() => {
                let header = core::mem::offset_of!(TW_ENUMERATION, ItemList);
                let item = match cap.item_type {
                    TWTY_FIX32 | TWTY_INT32 | TWTY_UINT32 => 4,
                    _ => 2,
                };
                let current = cap
                    .allowed
                    .iter()
                    .position(|&v| v == cap.current)
                    .unwrap_or(0);
                request.ConType = TWON_ENUMERATION;
                request.hContainer = self.container(header + item * cap.allowed.len(), |ptr| {
                    // SAFETY: the container holds the header and every item.
                    unsafe {
                        ptr.cast::<TW_ENUMERATION>()
                            .write_unaligned(TW_ENUMERATION {
                                ItemType: cap.item_type,
                                NumItems: u32::try_from(cap.allowed.len()).expect("count"),
                                CurrentIndex: u32::try_from(current).expect("index"),
                                DefaultIndex: 0,
                                ItemList: [0],
                            });
                        for (i, value) in cap.allowed.iter().enumerate() {
                            let at = ptr.add(header + i * item);
                            if item == 4 {
                                at.cast::<u32>().write_unaligned(*value);
                            } else {
                                at.cast::<u16>().write_unaligned(*value as u16);
                            }
                        }
                    }
                });
                TWRC_SUCCESS
            }
            MSG_GET | MSG_GETCURRENT => {
                request.ConType = TWON_ONEVALUE;
                request.hContainer = self.container(core::mem::size_of::<TW_ONEVALUE>(), |ptr| {
                    // SAFETY: the container is one value's size.
                    unsafe {
                        ptr.cast::<TW_ONEVALUE>().write_unaligned(TW_ONEVALUE {
                            ItemType: cap.item_type,
                            Item: cap.current,
                        });
                    }
                });
                TWRC_SUCCESS
            }
            MSG_SET => {
                if cap.read_only {
                    return self.fail(TWCC_BADCAP);
                }
                // SAFETY: the application's one-value container.
                let wanted = unsafe {
                    let ptr = self.lock(request.hContainer).cast::<TW_ONEVALUE>();
                    let one = ptr.read_unaligned();
                    self.unlock(request.hContainer);
                    one.Item
                };
                let (value, rc) = if cap.allowed.is_empty() || cap.allowed.contains(&wanted) {
                    (wanted, TWRC_SUCCESS)
                } else {
                    let nearest = *cap
                        .allowed
                        .iter()
                        .min_by_key(|&&v| {
                            (i64::from(v as u16 as i16) - i64::from(wanted as u16 as i16)).abs()
                        })
                        .expect("allowed");
                    (nearest, TWRC_CHECKSTATUS)
                };
                self.state
                    .borrow_mut()
                    .set_values
                    .insert((index, cap_id), value);
                rc
            }
            _ => self.fail(TWCC_BADPROTOCOL),
        }
    }

    fn transfer_strip(&self, strip: &mut TW_IMAGEMEMXFER) -> u16 {
        let Some(page) = self.page().cloned() else {
            return self.fail(TWCC_SEQERROR);
        };
        let (index, row) = {
            let state = self.state.borrow();
            (state.open.expect("open"), state.row)
        };
        let source = &self.sources[index];
        let page_index = self.state.borrow().page;
        if let Some((fail_page, condition)) = source.fail_page
            && fail_page == page_index
            && row > 0
        {
            return self.fail(condition);
        }

        let row_bytes = page.rows[0].len();
        let bytes_per_row = row_bytes + source.row_padding;
        let rows = source.strip_rows.min(page.rows.len() - row);
        let capacity = strip.Memory.Length as usize;
        assert!(
            rows * bytes_per_row <= capacity,
            "strip larger than the application's buffer"
        );
        let mem = strip.Memory.TheMem.cast::<u8>();
        for r in 0..rows {
            // SAFETY: inside the application's buffer, checked above.
            unsafe {
                let at = mem.add(r * bytes_per_row);
                core::ptr::copy_nonoverlapping(page.rows[row + r].as_ptr(), at, row_bytes);
                core::ptr::write_bytes(at.add(row_bytes), 0xAB, source.row_padding);
            }
        }
        strip.Compression = TWCP_NONE;
        strip.BytesPerRow = u32::try_from(bytes_per_row).expect("bytes");
        strip.Columns = page.width;
        strip.Rows = u32::try_from(rows).expect("rows");
        strip.XOffset = 0;
        strip.YOffset = u32::try_from(row).expect("row");
        strip.BytesWritten = u32::try_from(rows * bytes_per_row).expect("bytes");

        let next = row + rows;
        self.state.borrow_mut().row = next;
        if next == page.rows.len() {
            TWRC_XFERDONE
        } else {
            TWRC_SUCCESS
        }
    }

    fn ext_info(&self, block: *mut c_void) -> u16 {
        let Some(page) = self.page().cloned() else {
            return self.fail(TWCC_SEQERROR);
        };
        let header = core::mem::offset_of!(TW_EXTIMAGEINFO, Info);
        // SAFETY: the application's block: a count and that many infos.
        unsafe {
            let count = block.cast::<u32>().read_unaligned() as usize;
            #[allow(clippy::cast_ptr_alignment)]
            let infos = block.cast::<u8>().add(header).cast::<TW_INFO>();
            for i in 0..count {
                let mut info = infos.add(i).read_unaligned();
                let mut rc = TWRC_INFONOTSUPPORTED;
                match info.InfoID {
                    TWEI_PATCHCODE => {
                        if let Some(patch) = page.patch {
                            info.ItemType = TWTY_UINT32;
                            info.NumItems = 1;
                            info.Item = crate::sys::TW_UINTPTR::from(patch);
                            rc = TWRC_SUCCESS;
                        }
                    }
                    TWEI_BARCODECOUNT => {
                        info.ItemType = TWTY_UINT32;
                        info.NumItems = 1;
                        info.Item =
                            crate::sys::TW_UINTPTR::try_from(page.barcodes.len()).expect("count");
                        rc = TWRC_SUCCESS;
                    }
                    TWEI_BARCODETEXT if !page.barcodes.is_empty() => {
                        let texts: Vec<TW_HANDLE> = page
                            .barcodes
                            .iter()
                            .map(|text| {
                                self.container(text.len() + 1, |ptr| {
                                    core::ptr::copy_nonoverlapping(text.as_ptr(), ptr, text.len());
                                    ptr.add(text.len()).write(0);
                                })
                            })
                            .collect();
                        info.ItemType = TWTY_HANDLE;
                        info.NumItems = u16::try_from(texts.len()).expect("count");
                        info.Item = if texts.len() == 1 {
                            uintptr(texts[0])
                        } else {
                            let size = texts.len() * core::mem::size_of::<TW_HANDLE>();
                            self.container(size, |ptr| {
                                for (j, handle) in texts.iter().enumerate() {
                                    ptr.cast::<TW_HANDLE>().add(j).write_unaligned(*handle);
                                }
                            })
                            .pipe(uintptr)
                        };
                        rc = TWRC_SUCCESS;
                    }
                    _ => {}
                }
                info.__bindgen_anon_1.ReturnCode = rc;
                infos.add(i).write_unaligned(info);
            }
        }
        TWRC_SUCCESS
    }
}

impl Dsm for FakeDsm {
    #[allow(clippy::too_many_lines)]
    unsafe fn entry(
        &self,
        origin: *mut TW_IDENTITY,
        dest: *mut TW_IDENTITY,
        dg: u32,
        dat: u16,
        msg: u16,
        data: *mut c_void,
    ) -> u16 {
        self.state.borrow_mut().calls.push((dat, msg));
        let to_source = !dest.is_null();
        match (dg, dat, msg) {
            (DG_CONTROL, DAT_PARENT, MSG_OPENDSM) => {
                self.state.borrow_mut().dsm_open = true;
                // SAFETY: the application's identity.
                unsafe { (*origin).SupportedGroups |= DF_DSM2 };
                TWRC_SUCCESS
            }
            (DG_CONTROL, DAT_PARENT, MSG_CLOSEDSM) => {
                let mut state = self.state.borrow_mut();
                assert!(state.open.is_none(), "closed the DSM with a source open");
                state.dsm_open = false;
                TWRC_SUCCESS
            }
            (DG_CONTROL, DAT_ENTRYPOINT | DAT_CALLBACK, _) => self.fail(TWCC_BADPROTOCOL),
            (DG_CONTROL, DAT_STATUS, MSG_GET) => {
                let condition = core::mem::take(&mut self.state.borrow_mut().condition);
                // SAFETY: the application's status.
                unsafe { (*data.cast::<TW_STATUS>()).ConditionCode = condition };
                TWRC_SUCCESS
            }
            (DG_CONTROL, DAT_IDENTITY, MSG_GETDEFAULT) => {
                if self.sources.is_empty() {
                    return self.fail(TWCC_NODS);
                }
                // SAFETY: the application's identity.
                unsafe { *data.cast::<TW_IDENTITY>() = self.identity_of(self.default) };
                TWRC_SUCCESS
            }
            (DG_CONTROL, DAT_IDENTITY, MSG_GETFIRST | MSG_GETNEXT) => {
                let mut state = self.state.borrow_mut();
                if msg == MSG_GETFIRST {
                    state.enumerate_at = 0;
                }
                if state.enumerate_at >= self.sources.len() {
                    drop(state);
                    return if self.sources.is_empty() {
                        self.fail(TWCC_NODS)
                    } else {
                        TWRC_ENDOFLIST
                    };
                }
                let at = state.enumerate_at;
                state.enumerate_at += 1;
                drop(state);
                // SAFETY: the application's identity.
                unsafe { *data.cast::<TW_IDENTITY>() = self.identity_of(at) };
                TWRC_SUCCESS
            }
            (DG_CONTROL, DAT_IDENTITY, MSG_OPENDS) => {
                // SAFETY: the application's identity.
                let name = unsafe { decode_str(&(*data.cast::<TW_IDENTITY>()).ProductName) };
                let Some(index) = self.sources.iter().position(|s| s.name == name) else {
                    return self.fail(TWCC_NODS);
                };
                self.state.borrow_mut().open = Some(index);
                TWRC_SUCCESS
            }
            (DG_CONTROL, DAT_IDENTITY, MSG_CLOSEDS) => {
                let mut state = self.state.borrow_mut();
                assert!(!state.enabled, "closed a source that was still enabled");
                state.open = None;
                TWRC_SUCCESS
            }
            (DG_CONTROL, DAT_CAPABILITY, _) if to_source => {
                // SAFETY: the application's capability request.
                self.capability(msg, unsafe { &mut *data.cast::<TW_CAPABILITY>() })
            }
            (DG_CONTROL, DAT_USERINTERFACE, MSG_ENABLEDS) => {
                let mut state = self.state.borrow_mut();
                state.enabled = true;
                state.ready_sent = false;
                TWRC_SUCCESS
            }
            (DG_CONTROL, DAT_USERINTERFACE, MSG_DISABLEDS) => {
                self.state.borrow_mut().enabled = false;
                TWRC_SUCCESS
            }
            (DG_CONTROL, DAT_EVENT, MSG_PROCESSEVENT) => {
                let left = self.pages_left();
                let mut state = self.state.borrow_mut();
                // SAFETY: the application's event.
                let event = unsafe { &mut *data.cast::<TW_EVENT>() };
                if !state.enabled {
                    return TWRC_NOTDSEVENT;
                }
                if !state.ready_sent {
                    state.ready_sent = true;
                    event.TWMessage = if left > 0 {
                        MSG_XFERREADY
                    } else {
                        MSG_CLOSEDSREQ
                    };
                    return TWRC_DSEVENT;
                }
                TWRC_NOTDSEVENT
            }
            (DG_IMAGE, DAT_IMAGEINFO, MSG_GET) => {
                let Some(page) = self.page().cloned() else {
                    return self.fail(TWCC_SEQERROR);
                };
                let unknown = self.sources[self.state.borrow().open.expect("open")].unknown_length;
                // SAFETY: the application's image info.
                let info = unsafe { &mut *data.cast::<TW_IMAGEINFO>() };
                info.XResolution = crate::capability::to_fix32(300.0);
                info.YResolution = crate::capability::to_fix32(300.0);
                info.ImageWidth = i32::try_from(page.width).expect("width");
                info.ImageLength = if unknown {
                    -1
                } else {
                    i32::try_from(page.rows.len()).expect("rows")
                };
                info.SamplesPerPixel = if page.pixel_type == TWPT_RGB { 3 } else { 1 };
                info.BitsPerPixel = page.bits_per_pixel;
                info.Planar = 0;
                info.PixelType = page.pixel_type as i16;
                info.Compression = TWCP_NONE;
                self.state.borrow_mut().row = 0;
                TWRC_SUCCESS
            }
            (DG_CONTROL, DAT_SETUPMEMXFER, MSG_GET) => {
                // SAFETY: the application's setup.
                let setup = unsafe { &mut *data.cast::<TW_SETUPMEMXFER>() };
                *setup = TW_SETUPMEMXFER {
                    MinBufSize: 1024,
                    MaxBufSize: 1 << 20,
                    Preferred: 16 << 10,
                };
                TWRC_SUCCESS
            }
            (DG_IMAGE, DAT_IMAGEMEMXFER, MSG_GET) => {
                // SAFETY: the application's strip request.
                self.transfer_strip(unsafe { &mut *data.cast::<TW_IMAGEMEMXFER>() })
            }
            (DG_IMAGE, DAT_EXTIMAGEINFO, MSG_GET) => self.ext_info(data),
            (DG_CONTROL, DAT_PENDINGXFERS, MSG_ENDXFER) => {
                let total = self.sources[self.state.borrow().open.expect("open")]
                    .pages
                    .len();
                let mut state = self.state.borrow_mut();
                state.page += 1;
                state.row = 0;
                let left = total.saturating_sub(state.page);
                // SAFETY: the application's pending transfers.
                unsafe {
                    (*data.cast::<TW_PENDINGXFERS>()).Count = if left == 0 { 0 } else { u16::MAX }
                };
                TWRC_SUCCESS
            }
            (DG_CONTROL, DAT_PENDINGXFERS, MSG_RESET) => {
                let mut state = self.state.borrow_mut();
                state.pending_reset = true;
                // SAFETY: the application's pending transfers.
                unsafe { (*data.cast::<TW_PENDINGXFERS>()).Count = 0 };
                TWRC_SUCCESS
            }
            _ => self.fail(TWCC_BADPROTOCOL),
        }
    }

    fn use_entry_points(&self, _: &TW_ENTRYPOINT) {}

    fn alloc(&self, size: u32) -> TW_HANDLE {
        self.state.borrow_mut().live_allocations += 1;
        Box::into_raw(Box::new(vec![0u8; size as usize])).cast()
    }

    unsafe fn free(&self, handle: TW_HANDLE) {
        self.state.borrow_mut().live_allocations -= 1;
        // SAFETY: allocated by `alloc` as a boxed vector.
        drop(unsafe { Box::from_raw(handle.cast::<Vec<u8>>()) });
    }

    unsafe fn lock(&self, handle: TW_HANDLE) -> *mut c_void {
        // SAFETY: allocated by `alloc` as a boxed vector.
        unsafe { (*handle.cast::<Vec<u8>>()).as_mut_ptr().cast() }
    }

    unsafe fn unlock(&self, _: TW_HANDLE) {}
}

/// Hands every wait's message to the source, as a message loop would.
#[derive(Debug, Default)]
pub struct FakePump {
    pub cancel: bool,
}

impl EventPump for FakePump {
    fn callback(&self) -> Option<*mut c_void> {
        None
    }

    fn wait(
        &mut self,
        _timeout: Option<std::time::Duration>,
        process: &mut dyn FnMut(*mut c_void) -> Processed,
    ) -> PumpEvent {
        if self.cancel {
            return PumpEvent::Cancel;
        }
        for _ in 0..4 {
            if let Processed::DsEvent(msg) = process(core::ptr::null_mut())
                && msg != MSG_NULL
            {
                return PumpEvent::Twain(msg);
            }
        }
        PumpEvent::TimedOut
    }
}

fn uintptr(handle: TW_HANDLE) -> crate::sys::TW_UINTPTR {
    crate::sys::TW_UINTPTR::try_from(handle.expose_provenance()).expect("pointer fits")
}

trait Pipe: Sized {
    fn pipe<R>(self, f: impl FnOnce(Self) -> R) -> R {
        f(self)
    }
}

impl<T> Pipe for T {}
