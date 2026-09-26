//! A TWAIN session: the DSM opened, one source opened, negotiated, enabled,
//! and its pages taken by buffered memory transfer.
//!
//! The TWAIN state machine is walked forward explicitly and backward by
//! `Drop`, so a scan that ends any way at all (finished, jammed, cancelled,
//! or an error half way through a page) leaves the source and the DSM closed
//! in the order the specification requires:
//!
//! | State | Meaning |
//! |---|---|
//! | 3 | DSM open ([`Manager`]) |
//! | 4 | Source open ([`Source`]) |
//! | 5 | Source enabled, waiting for `MSG_XFERREADY` |
//! | 6 | Transfers pending |
//! | 7 | A page is being transferred |

use core::ffi::c_void;
use core::mem::size_of;
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::Duration;

use capture_imaging::{OwnedRaster, PixelFormat, Resolution};
use capture_protocol::api::{PixelType, Settings, SourceInfo, SourceProtocol, known_patch_code};
use capture_protocol::helper::ScanCondition;

use crate::capability::{self, CapValue, CapValues, ONE_VALUE_SIZE};
#[allow(clippy::wildcard_imports)] // The constants mirror twain.h's own namespace.
use crate::consts::*;
use crate::dsm::{Dsm, DsmMemory};
use crate::pump::{EventPump, Processed, PumpEvent};
use crate::sys::{
    TW_CAPABILITY, TW_ENTRYPOINT, TW_EVENT, TW_EXTIMAGEINFO, TW_IDENTITY, TW_IMAGEINFO,
    TW_IMAGEMEMXFER, TW_INFO, TW_MEMORY, TW_PENDINGXFERS, TW_SETUPMEMXFER, TW_STATUS,
    TW_USERINTERFACE, TW_VERSION,
};
use crate::text::{decode_str, encode_str32};
use crate::{TwainError, TwainResult};

/// Resolutions offered to the web app when a source reports a range, the
/// ones document scanners and paperwork actually use.
const STANDARD_RESOLUTIONS: [i64; 7] = [100, 150, 200, 240, 300, 400, 600];
/// Bounds on the transfer buffer a source may ask for.
const MIN_BUFFER: u32 = 64 << 10;
const MAX_BUFFER: u32 = 8 << 20;
/// Bounds on what extended image info may carry.
const MAX_BARCODES: usize = 32;
const MAX_BARCODE_LEN: usize = 4096;
/// How long a source enabled without its own window may take to start. A
/// feeder that is empty often simply never says it is ready.
const START_TIMEOUT: Duration = Duration::from_secs(90);

/// Who this application says it is to the DSM and the sources.
#[derive(Clone, Debug)]
pub struct AppIdentity {
    pub version: String,
    pub major: u16,
    pub minor: u16,
}

/// A source as the DSM lists it, before it is opened.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SourceEntry {
    pub name: String,
    pub manufacturer: String,
    pub family: String,
    pub version: String,
    pub is_default: bool,
    /// Whether it speaks TWAIN 2, which is what its callbacks need.
    pub twain2: bool,
}

/// What a scan was asked for.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct ScanSettings {
    pub dpi: u32,
    pub pixel_type: PixelType,
    pub duplex: bool,
    pub use_feeder: bool,
    pub discard_blank_pages: bool,
    pub show_ui: bool,
    pub detect_patch_codes: bool,
    pub detect_barcodes: bool,
}

/// The settings a source ended up using.
#[derive(Clone, Debug, PartialEq)]
pub struct Negotiated {
    pub settings: Settings,
    /// Whether blank pages are dropped by the source. When not, the server's
    /// own blank detection still marks them.
    pub source_discards_blanks: bool,
}

/// One page, as transferred.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ScannedPage {
    pub raster: OwnedRaster,
    pub resolution: Resolution,
    pub patch_code: Option<&'static str>,
    pub barcodes: Vec<String>,
}

/// How a scan ended.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ScanEnd {
    Finished {
        pages: u32,
    },
    /// The source stopped early. The pages delivered before stand.
    Stopped {
        condition: ScanCondition,
        pages: u32,
    },
    /// The agent asked to stop.
    Canceled {
        pages: u32,
    },
}

/// The DSM, open (state 3).
pub struct Manager<D: Dsm> {
    dsm: D,
    app: Box<TW_IDENTITY>,
    /// The parent window, boxed so the pointer the DSM keeps stays valid.
    parent: Box<*mut c_void>,
    twain2: bool,
}

impl<D: Dsm> core::fmt::Debug for Manager<D> {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.debug_struct("Manager")
            .field("twain2", &self.twain2)
            .finish_non_exhaustive()
    }
}

impl<D: Dsm> Manager<D> {
    /// Opens the DSM (state 2 → 3). `parent` is the window the DSM and the
    /// sources' dialogs are owned by.
    pub fn open(dsm: D, app: &AppIdentity, parent: *mut c_void) -> TwainResult<Self> {
        let mut identity = Box::new(app_identity(app));
        let mut parent = Box::new(parent);
        // SAFETY: the identity and the parent handle are boxed and outlive
        // the call; `parent` points at a window handle as MSG_OPENDSM takes.
        let rc = unsafe {
            dsm.entry(
                &raw mut *identity,
                core::ptr::null_mut(),
                DG_CONTROL,
                DAT_PARENT,
                MSG_OPENDSM,
                (&raw mut *parent).cast(),
            )
        };
        if rc != TWRC_SUCCESS {
            return Err(TwainError::Dsm {
                operation: "open the TWAIN data source manager",
                rc,
            });
        }

        let twain2 = identity.SupportedGroups & DF_DSM2 != 0;
        let manager = Self {
            dsm,
            app: identity,
            parent,
            twain2,
        };
        if twain2 {
            manager.attach_entry_points();
        }
        Ok(manager)
    }

    fn attach_entry_points(&self) {
        // SAFETY: TW_ENTRYPOINT is plain data; all-zero is a valid "no
        // functions" value, and Size tells the DSM which version it is.
        let mut entry_points: TW_ENTRYPOINT = unsafe { core::mem::zeroed() };
        entry_points.Size = u32::try_from(size_of::<TW_ENTRYPOINT>()).unwrap_or(u32::MAX);
        let rc = self.call(
            None,
            DG_CONTROL,
            DAT_ENTRYPOINT,
            MSG_GET,
            (&raw mut entry_points).cast(),
        );
        if rc == TWRC_SUCCESS {
            self.dsm.use_entry_points(&entry_points);
        }
    }

    fn app_ptr(&self) -> *mut TW_IDENTITY {
        core::ptr::from_ref::<TW_IDENTITY>(&self.app).cast_mut()
    }

    fn call(
        &self,
        dest: Option<*mut TW_IDENTITY>,
        dg: u32,
        dat: u16,
        msg: u16,
        data: *mut c_void,
    ) -> u16 {
        // SAFETY: every caller passes `data` pointing at the structure the
        // triplet names, alive for the call; the identities are boxed.
        unsafe {
            self.dsm.entry(
                self.app_ptr(),
                dest.unwrap_or(core::ptr::null_mut()),
                dg,
                dat,
                msg,
                data,
            )
        }
    }

    /// The condition code behind the last failure, from the DSM or a source.
    fn condition(&self, dest: Option<*mut TW_IDENTITY>) -> u16 {
        let mut status = TW_STATUS {
            ConditionCode: TWCC_SUCCESS,
            __bindgen_anon_1: crate::sys::TW_STATUS__bindgen_ty_1 { Reserved: 0 },
        };
        let rc = self.call(
            dest,
            DG_CONTROL,
            DAT_STATUS,
            MSG_GET,
            (&raw mut status).cast(),
        );
        if rc == TWRC_SUCCESS {
            status.ConditionCode
        } else {
            TWCC_BUMMER
        }
    }

    /// Lists every source the DSM knows, in its order, marking its default.
    pub fn sources(&self) -> TwainResult<Vec<SourceEntry>> {
        let default_name = {
            // SAFETY: plain data; zero is a valid empty identity.
            let mut identity: TW_IDENTITY = unsafe { core::mem::zeroed() };
            let rc = self.call(
                None,
                DG_CONTROL,
                DAT_IDENTITY,
                MSG_GETDEFAULT,
                (&raw mut identity).cast(),
            );
            (rc == TWRC_SUCCESS).then(|| decode_str(&identity.ProductName))
        };

        let mut sources = Vec::new();
        let mut msg = MSG_GETFIRST;
        loop {
            // SAFETY: plain data, filled by the DSM.
            let mut identity: TW_IDENTITY = unsafe { core::mem::zeroed() };
            let rc = self.call(
                None,
                DG_CONTROL,
                DAT_IDENTITY,
                msg,
                (&raw mut identity).cast(),
            );
            match rc {
                TWRC_SUCCESS => {
                    let name = decode_str(&identity.ProductName);
                    if !name.is_empty() && !sources.iter().any(|s: &SourceEntry| s.name == name) {
                        sources.push(SourceEntry {
                            is_default: default_name.as_deref() == Some(name.as_str()),
                            manufacturer: decode_str(&identity.Manufacturer),
                            family: decode_str(&identity.ProductFamily),
                            version: decode_str(&identity.Version.Info),
                            twain2: identity.SupportedGroups & DF_DSM2 != 0,
                            name,
                        });
                    }
                    msg = MSG_GETNEXT;
                }
                TWRC_ENDOFLIST => return Ok(sources),
                _ if sources.is_empty() && self.condition(None) == TWCC_NODS => return Ok(sources),
                _ => {
                    return Err(TwainError::Dsm {
                        operation: "list TWAIN sources",
                        rc,
                    });
                }
            }
        }
    }

    /// Opens a source by the name enumeration reported (state 3 → 4).
    pub fn open_source(&mut self, name: &str) -> TwainResult<Source<'_, D>> {
        let wanted = name.trim();
        // SAFETY: plain data.
        let mut identity: Box<TW_IDENTITY> = Box::new(unsafe { core::mem::zeroed() });
        let mut msg = MSG_GETFIRST;
        loop {
            let rc = self.call(
                None,
                DG_CONTROL,
                DAT_IDENTITY,
                msg,
                (&raw mut *identity).cast(),
            );
            if rc != TWRC_SUCCESS {
                return Err(TwainError::SourceNotFound(wanted.to_owned()));
            }
            if decode_str(&identity.ProductName) == wanted {
                break;
            }
            msg = MSG_GETNEXT;
        }

        let rc = self.call(
            None,
            DG_CONTROL,
            DAT_IDENTITY,
            MSG_OPENDS,
            (&raw mut *identity).cast(),
        );
        if rc != TWRC_SUCCESS {
            let condition = self.condition(None);
            return Err(TwainError::Open {
                name: wanted.to_owned(),
                condition,
            });
        }

        Ok(Source {
            manager: self,
            identity,
            state: 4,
            callback: false,
        })
    }
}

impl<D: Dsm> Drop for Manager<D> {
    fn drop(&mut self) {
        let parent: *mut *mut c_void = &raw mut *self.parent;
        let rc = self.call(None, DG_CONTROL, DAT_PARENT, MSG_CLOSEDSM, parent.cast());
        if rc != TWRC_SUCCESS {
            tracing::warn!(rc, "the TWAIN data source manager did not close cleanly");
        }
    }
}

fn app_identity(app: &AppIdentity) -> TW_IDENTITY {
    TW_IDENTITY {
        Id: 0,
        Version: TW_VERSION {
            MajorNum: app.major,
            MinorNum: app.minor,
            Language: TWLG_ENGLISH_USA,
            Country: TWCY_USA,
            Info: encode_str32(&app.version),
        },
        ProtocolMajor: u16::try_from(TWON_PROTOCOLMAJOR).unwrap_or(2),
        ProtocolMinor: u16::try_from(TWON_PROTOCOLMINOR).unwrap_or(4),
        SupportedGroups: DG_CONTROL | DG_IMAGE | DF_APP2,
        Manufacturer: encode_str32("Trenova"),
        ProductFamily: encode_str32("Trenova Capture"),
        ProductName: encode_str32("Trenova Capture"),
    }
}

/// An open source (state 4 and up).
pub struct Source<'m, D: Dsm> {
    manager: &'m mut Manager<D>,
    identity: Box<TW_IDENTITY>,
    state: u8,
    callback: bool,
}

impl<D: Dsm> core::fmt::Debug for Source<'_, D> {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.debug_struct("Source")
            .field("name", &decode_str(&self.identity.ProductName))
            .field("state", &self.state)
            .finish_non_exhaustive()
    }
}

/// How a `MSG_SET` went.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum SetOutcome {
    Applied,
    /// The source took a nearby value instead (`TWRC_CHECKSTATUS`).
    Adjusted,
    Refused,
}

impl<D: Dsm> Source<'_, D> {
    pub fn name(&self) -> String {
        decode_str(&self.identity.ProductName)
    }

    pub fn driver_version(&self) -> String {
        decode_str(&self.identity.Version.Info)
    }

    fn dest(&self) -> *mut TW_IDENTITY {
        core::ptr::from_ref::<TW_IDENTITY>(&self.identity).cast_mut()
    }

    fn call(&self, dg: u32, dat: u16, msg: u16, data: *mut c_void) -> u16 {
        self.manager.call(Some(self.dest()), dg, dat, msg, data)
    }

    fn condition(&self) -> u16 {
        self.manager.condition(Some(self.dest()))
    }

    /// Reads a capability: its current value with `MSG_GETCURRENT`, or
    /// everything it allows with `MSG_GET`. `None` when the source does not
    /// support it.
    fn read_cap(&self, cap: u16, msg: u16) -> Option<CapValues> {
        let mut request = TW_CAPABILITY {
            Cap: cap,
            ConType: TWON_DONTCARE16,
            hContainer: core::ptr::null_mut(),
        };
        let rc = self.call(DG_CONTROL, DAT_CAPABILITY, msg, (&raw mut request).cast());
        // SAFETY: on any answer the source owns nothing further; a container
        // it filled is now the application's to free.
        let container = unsafe { DsmMemory::adopt(&self.manager.dsm, request.hContainer) };
        if rc != TWRC_SUCCESS {
            return None;
        }
        let container = container?;
        // SAFETY: a source that answered success filled a container of the
        // type it named.
        container
            .with(|ptr| unsafe { capability::decode(request.ConType, ptr) })
            .flatten()
    }

    fn current(&self, cap: u16) -> Option<CapValue> {
        self.read_cap(cap, MSG_GETCURRENT).and_then(|v| v.current())
    }

    /// Sets a capability to one value.
    fn set_cap(&self, cap: u16, item_type: u16, value: CapValue) -> SetOutcome {
        let Some(container) = DsmMemory::alloc(&self.manager.dsm, ONE_VALUE_SIZE) else {
            return SetOutcome::Refused;
        };
        if container
            // SAFETY: the container is a one-value container's size.
            .with(|ptr| unsafe { capability::write_one_value(ptr, item_type, value) })
            .is_none()
        {
            return SetOutcome::Refused;
        }
        let mut request = TW_CAPABILITY {
            Cap: cap,
            ConType: TWON_ONEVALUE,
            hContainer: container.handle(),
        };
        match self.call(
            DG_CONTROL,
            DAT_CAPABILITY,
            MSG_SET,
            (&raw mut request).cast(),
        ) {
            TWRC_SUCCESS => SetOutcome::Applied,
            TWRC_CHECKSTATUS => SetOutcome::Adjusted,
            _ => SetOutcome::Refused,
        }
    }

    /// What this source can do, learned by asking it. Only an open source
    /// can answer, which is why enumeration alone does not report it.
    pub fn describe(&self, is_default: bool, bitness: u8) -> SourceInfo {
        let supported = |cap| self.read_cap(cap, MSG_GET).is_some();

        let duplex = self
            .current(CAP_DUPLEX)
            .is_some_and(|v| v.as_i64() != i64::from(TWDX_NONE));

        let resolutions = match self.read_cap(ICAP_XRESOLUTION, MSG_GET) {
            Some(values @ CapValues::Range { .. }) => STANDARD_RESOLUTIONS
                .iter()
                .copied()
                .filter(|&dpi| {
                    #[allow(clippy::cast_precision_loss)]
                    let dpi = dpi as f64;
                    values.allows(dpi)
                })
                .collect(),
            Some(values) => values.whole_values(),
            None => Vec::new(),
        };
        let mut resolutions: Vec<u32> = resolutions
            .into_iter()
            .filter_map(|dpi| u32::try_from(dpi).ok())
            .filter(|dpi| (50..=1200).contains(dpi))
            .collect();
        resolutions.sort_unstable();
        resolutions.dedup();

        let mut pixel_types: Vec<PixelType> = self
            .read_cap(ICAP_PIXELTYPE, MSG_GET)
            .map(|v| v.whole_values())
            .unwrap_or_default()
            .into_iter()
            .filter_map(|v| pixel_type_of(u16::try_from(v).ok()?))
            .collect();
        pixel_types.dedup();

        SourceInfo {
            name: self.name(),
            protocol: SourceProtocol::Twain,
            bitness,
            is_default,
            duplex,
            feeder: supported(CAP_FEEDERENABLED),
            patch_codes: supported(ICAP_PATCHCODEDETECTIONENABLED),
            barcodes: supported(ICAP_BARCODEDETECTIONENABLED),
            blank_discard: supported(ICAP_AUTODISCARDBLANKPAGES),
            resolutions,
            pixel_types,
        }
    }

    /// Applies a scan's settings (state 4). Buffered memory transfer is the
    /// one thing a source must accept; everything else it declines is
    /// recorded by name and the scan goes on with what it will do.
    pub fn negotiate(&mut self, want: &ScanSettings, bitness: u8) -> TwainResult<Negotiated> {
        let memory = CapValue::Uint(u32::from(TWSX_MEMORY));
        if self.set_cap(ICAP_XFERMECH, TWTY_UINT16, memory) == SetOutcome::Refused {
            return Err(TwainError::Unsupported(
                "the scanner does not support buffered memory transfer".into(),
            ));
        }

        let mut refused: Vec<String> = requested_capabilities(want)
            .into_iter()
            .filter(|&(cap, item_type, value)| {
                self.set_cap(cap, item_type, value) == SetOutcome::Refused
            })
            .map(|(cap, _, _)| capability_name(cap).to_owned())
            .collect();

        let dpi_now = self
            .current(ICAP_XRESOLUTION)
            .map_or(i64::from(want.dpi), CapValue::as_i64);
        let pixel_now = self
            .current(ICAP_PIXELTYPE)
            .and_then(|v| pixel_type_of(u16::try_from(v.as_i64()).ok()?))
            .unwrap_or(want.pixel_type);
        let discards = self
            .current(ICAP_AUTODISCARDBLANKPAGES)
            .is_some_and(|v| v.as_i64() != i64::from(TWBP_DISABLE));

        refused.sort();
        refused.dedup();
        Ok(Negotiated {
            settings: Settings {
                protocol: Some(SourceProtocol::Twain),
                bitness,
                dpi: u32::try_from(dpi_now).unwrap_or(want.dpi),
                pixel_type: Some(pixel_now),
                duplex: self
                    .current(CAP_DUPLEXENABLED)
                    .is_some_and(CapValue::as_bool),
                feeder: self
                    .current(CAP_FEEDERENABLED)
                    .is_some_and(CapValue::as_bool),
                blank_discard: discards,
                show_driver_ui: want.show_ui,
                driver_version: self.driver_version(),
                application: String::new(),
                refused,
            },
            source_discards_blanks: discards,
        })
    }

    /// Runs a scan: enables the source, waits for it, and hands each page to
    /// `on_page` as it is transferred. Returns how the scan ended; the source
    /// is disabled again (state 4) whatever happens.
    pub fn scan<E>(
        &mut self,
        want: &ScanSettings,
        pump: &mut dyn EventPump,
        cancel: &AtomicBool,
        on_page: &mut dyn FnMut(ScannedPage) -> Result<(), E>,
    ) -> TwainResult<ScanEnd>
    where
        E: std::fmt::Display,
    {
        if want.use_feeder
            && !want.show_ui
            && self.current(CAP_FEEDERLOADED).is_some_and(|v| !v.as_bool())
        {
            return Ok(ScanEnd::Stopped {
                condition: ScanCondition::FeederEmpty,
                pages: 0,
            });
        }

        self.callback = self.manager.twain2
            && self.identity.SupportedGroups & DF_DSM2 != 0
            && pump
                .callback()
                .is_some_and(|proc_| self.register_callback(proc_));

        let show_ui = want.show_ui
            || self
                .current(CAP_UICONTROLLABLE)
                .is_some_and(|controllable| !controllable.as_bool());
        let mut ui = TW_USERINTERFACE {
            ShowUI: u16::from(show_ui),
            ModalUI: 0,
            hParent: *self.manager.parent,
        };
        let rc = self.call(
            DG_CONTROL,
            DAT_USERINTERFACE,
            MSG_ENABLEDS,
            (&raw mut ui).cast(),
        );
        match rc {
            TWRC_SUCCESS | TWRC_CHECKSTATUS => self.state = 5,
            TWRC_CANCEL => {
                return Ok(ScanEnd::Stopped {
                    condition: ScanCondition::CanceledByOperator,
                    pages: 0,
                });
            }
            _ => {
                return Err(self.failure("start the scanner"));
            }
        }

        let result = self.run(show_ui, pump, cancel, on_page);
        self.disable();
        result
    }

    fn run<E: std::fmt::Display>(
        &mut self,
        show_ui: bool,
        pump: &mut dyn EventPump,
        cancel: &AtomicBool,
        on_page: &mut dyn FnMut(ScannedPage) -> Result<(), E>,
    ) -> TwainResult<ScanEnd> {
        let timeout = (!show_ui).then_some(START_TIMEOUT);
        loop {
            match self.wait(pump, timeout) {
                PumpEvent::Twain(MSG_XFERREADY) => {
                    self.state = 6;
                    return self.transfer_all(cancel, on_page);
                }
                PumpEvent::Twain(MSG_CLOSEDSREQ | MSG_CLOSEDSOK) => {
                    return Ok(ScanEnd::Stopped {
                        condition: ScanCondition::CanceledByOperator,
                        pages: 0,
                    });
                }
                PumpEvent::Twain(_) => {}
                PumpEvent::Cancel => return Ok(ScanEnd::Canceled { pages: 0 }),
                PumpEvent::TimedOut => {
                    return Ok(ScanEnd::Stopped {
                        condition: ScanCondition::FeederEmpty,
                        pages: 0,
                    });
                }
            }
        }
    }

    fn wait(&self, pump: &mut dyn EventPump, timeout: Option<Duration>) -> PumpEvent {
        let callback = self.callback;
        let mut process = |msg: *mut c_void| -> Processed {
            if callback {
                return Processed::NotDsEvent;
            }
            let mut event = TW_EVENT {
                pEvent: msg,
                TWMessage: MSG_NULL,
            };
            match self.call(
                DG_CONTROL,
                DAT_EVENT,
                MSG_PROCESSEVENT,
                (&raw mut event).cast(),
            ) {
                TWRC_DSEVENT => Processed::DsEvent(event.TWMessage),
                _ => Processed::NotDsEvent,
            }
        };
        pump.wait(timeout, &mut process)
    }

    fn register_callback(&self, proc_: *mut c_void) -> bool {
        let mut callback = crate::sys::TW_CALLBACK {
            CallBackProc: proc_,
            RefCon: 0,
            Message: 0,
        };
        self.call(
            DG_CONTROL,
            DAT_CALLBACK,
            MSG_REGISTER_CALLBACK,
            (&raw mut callback).cast(),
        ) == TWRC_SUCCESS
    }

    /// Transfers pages until the source has none pending (state 6 → 5).
    fn transfer_all<E: std::fmt::Display>(
        &mut self,
        cancel: &AtomicBool,
        on_page: &mut dyn FnMut(ScannedPage) -> Result<(), E>,
    ) -> TwainResult<ScanEnd> {
        let mut pages = 0u32;
        loop {
            if cancel.load(Ordering::Acquire) {
                self.reset_pending();
                return Ok(ScanEnd::Canceled { pages });
            }

            let page = match self.transfer_page() {
                Ok(Transfer::Page(page)) => page,
                Ok(Transfer::Canceled) => {
                    self.end_transfer();
                    self.reset_pending();
                    return Ok(ScanEnd::Stopped {
                        condition: ScanCondition::CanceledByOperator,
                        pages,
                    });
                }
                Ok(Transfer::Condition(condition)) => {
                    self.end_transfer();
                    self.reset_pending();
                    return Ok(ScanEnd::Stopped { condition, pages });
                }
                Err(err) => {
                    self.end_transfer();
                    self.reset_pending();
                    return Err(err);
                }
            };

            let pending = self.end_transfer();
            pages += 1;
            if let Err(err) = on_page(page) {
                self.reset_pending();
                return Err(TwainError::Delivery(err.to_string()));
            }
            if pending == 0 {
                self.state = 5;
                return Ok(ScanEnd::Finished { pages });
            }
        }
    }

    /// Ends the current transfer (state 7 → 6, or 5 when nothing is left),
    /// returning how many transfers are still pending; -1 is "unknown", as a
    /// feeder reports it. A source that refuses is treated as having left the
    /// transfer anyway, so the teardown after it still runs.
    fn end_transfer(&mut self) -> i16 {
        if self.state < 6 {
            return 0;
        }
        let mut pending = TW_PENDINGXFERS {
            Count: 0,
            __bindgen_anon_1: crate::sys::TW_PENDINGXFERS__bindgen_ty_1 { EOJ: 0 },
        };
        let rc = self.call(
            DG_CONTROL,
            DAT_PENDINGXFERS,
            MSG_ENDXFER,
            (&raw mut pending).cast(),
        );
        if rc != TWRC_SUCCESS {
            tracing::warn!(rc, "the TWAIN source refused to end a transfer");
            self.state = 6;
            return -1;
        }
        let count = i16::from_ne_bytes(pending.Count.to_ne_bytes());
        self.state = if count == 0 { 5 } else { 6 };
        count
    }

    /// Drops every pending transfer (state 6 → 5).
    fn reset_pending(&mut self) {
        if self.state != 6 {
            return;
        }
        let mut pending = TW_PENDINGXFERS {
            Count: 0,
            __bindgen_anon_1: crate::sys::TW_PENDINGXFERS__bindgen_ty_1 { EOJ: 0 },
        };
        let rc = self.call(
            DG_CONTROL,
            DAT_PENDINGXFERS,
            MSG_RESET,
            (&raw mut pending).cast(),
        );
        if rc != TWRC_SUCCESS {
            tracing::warn!(rc, "the TWAIN source refused to drop its pending transfers");
        }
        self.state = 5;
    }

    /// Disables the source (back to state 4) from wherever it is. Each step
    /// is attempted even if the one before it was refused: a source left
    /// enabled would keep the scanner busy until the helper exits.
    fn disable(&mut self) {
        if self.state == 7 {
            self.end_transfer();
        }
        if self.state == 6 {
            self.reset_pending();
        }
        if self.state != 5 {
            return;
        }
        let mut ui = TW_USERINTERFACE {
            ShowUI: 0,
            ModalUI: 0,
            hParent: *self.manager.parent,
        };
        let rc = self.call(
            DG_CONTROL,
            DAT_USERINTERFACE,
            MSG_DISABLEDS,
            (&raw mut ui).cast(),
        );
        if rc != TWRC_SUCCESS {
            tracing::warn!(rc, "the TWAIN source did not disable cleanly");
        }
        self.state = 4;
    }

    fn failure(&self, operation: &'static str) -> TwainError {
        TwainError::Source {
            operation,
            condition: self.condition(),
        }
    }

    /// Transfers one page (state 6 → 7).
    fn transfer_page(&mut self) -> TwainResult<Transfer> {
        // SAFETY: plain data, filled by the source.
        let mut info: TW_IMAGEINFO = unsafe { core::mem::zeroed() };
        if self.call(DG_IMAGE, DAT_IMAGEINFO, MSG_GET, (&raw mut info).cast()) != TWRC_SUCCESS {
            return Err(self.failure("read the page's image information"));
        }
        let layout = PageLayout::from_info(&info, self.current(ICAP_PIXELFLAVOR))?;

        let mut setup = TW_SETUPMEMXFER {
            MinBufSize: 0,
            MaxBufSize: 0,
            Preferred: 0,
        };
        if self.call(
            DG_CONTROL,
            DAT_SETUPMEMXFER,
            MSG_GET,
            (&raw mut setup).cast(),
        ) != TWRC_SUCCESS
        {
            return Err(self.failure("set up the memory transfer"));
        }
        let size = buffer_size(&setup);
        let mut buffer = vec![0u8; size as usize];
        let mut page = PageBuffer::new(layout);

        self.state = 7;
        loop {
            let mut strip = TW_IMAGEMEMXFER {
                Compression: TWON_DONTCARE16,
                BytesPerRow: u32::MAX,
                Columns: u32::MAX,
                Rows: u32::MAX,
                XOffset: u32::MAX,
                YOffset: u32::MAX,
                BytesWritten: u32::MAX,
                Memory: TW_MEMORY {
                    Flags: TWMF_APPOWNS | TWMF_POINTER,
                    Length: size,
                    TheMem: buffer.as_mut_ptr().cast(),
                },
            };
            let rc = self.call(DG_IMAGE, DAT_IMAGEMEMXFER, MSG_GET, (&raw mut strip).cast());
            match rc {
                TWRC_SUCCESS | TWRC_XFERDONE => {
                    page.add_strip(&strip, &buffer)?;
                    if rc == TWRC_XFERDONE {
                        break;
                    }
                }
                TWRC_CANCEL => return Ok(Transfer::Canceled),
                _ => {
                    let condition = self.condition();
                    return match scan_condition(condition) {
                        Some(condition) => Ok(Transfer::Condition(condition)),
                        None => Err(TwainError::Source {
                            operation: "transfer a page",
                            condition,
                        }),
                    };
                }
            }
        }

        let (patch_code, barcodes) = self.extended_info();
        Ok(Transfer::Page(ScannedPage {
            resolution: page.layout.resolution,
            raster: page.finish()?,
            patch_code,
            barcodes,
        }))
    }

    /// Reads the patch code and barcodes the source found on the page just
    /// transferred (state 7), when it offers them.
    fn extended_info(&self) -> (Option<&'static str>, Vec<String>) {
        const IDS: [u16; 3] = [TWEI_PATCHCODE, TWEI_BARCODECOUNT, TWEI_BARCODETEXT];
        let header = core::mem::offset_of!(TW_EXTIMAGEINFO, Info);
        let size = header + IDS.len() * size_of::<TW_INFO>();
        let mut block = vec![0u8; size];
        // Only ever read and written unaligned: the structure is packed.
        #[allow(clippy::cast_ptr_alignment)]
        let infos = block[header..].as_mut_ptr().cast::<TW_INFO>();
        // SAFETY: the block holds the header and `IDS.len()` infos; writes
        // are unaligned because the structure is packed.
        unsafe {
            block
                .as_mut_ptr()
                .cast::<u32>()
                .write_unaligned(u32::try_from(IDS.len()).unwrap_or(0));
            for (i, id) in IDS.iter().enumerate() {
                infos.add(i).write_unaligned(TW_INFO {
                    InfoID: *id,
                    ItemType: 0,
                    NumItems: 0,
                    __bindgen_anon_1: crate::sys::TW_INFO__bindgen_ty_1 {
                        ReturnCode: TWRC_INFONOTSUPPORTED,
                    },
                    Item: 0,
                });
            }
        }
        if self.call(
            DG_IMAGE,
            DAT_EXTIMAGEINFO,
            MSG_GET,
            block.as_mut_ptr().cast(),
        ) != TWRC_SUCCESS
        {
            return (None, Vec::new());
        }

        // SAFETY: the source filled the infos in place.
        let read = |i: usize| unsafe { infos.add(i).read_unaligned() };
        let patch = read(0);
        let count = read(1);
        let text = read(2);

        let patch_code = (return_code(&patch) == TWRC_SUCCESS && patch.NumItems == 1)
            .then(|| patch_code_name(item_of(&patch)))
            .flatten();

        let barcode_count = if return_code(&count) == TWRC_SUCCESS && count.NumItems == 1 {
            item_of(&count)
        } else {
            0
        };
        let barcodes = if return_code(&text) == TWRC_SUCCESS && barcode_count > 0 {
            self.barcode_texts(&text, barcode_count.min(MAX_BARCODES))
        } else {
            Vec::new()
        };
        // Handles the source handed over are the application's to free,
        // whether or not they were read.
        self.free_info_handles(&text);

        (patch_code, barcodes)
    }

    /// Reads barcode strings: one handle holding the text when there is one
    /// barcode, a handle to an array of handles when there are more.
    fn barcode_texts(&self, info: &TW_INFO, count: usize) -> Vec<String> {
        if info.ItemType != TWTY_HANDLE || item_of(info) == 0 {
            return Vec::new();
        }
        let dsm = &self.manager.dsm;
        let handle: *mut c_void = core::ptr::with_exposed_provenance_mut(item_of(info));
        if info.NumItems <= 1 {
            // SAFETY: the source owns this handle until `free_info_handles`.
            let text = unsafe { read_text(dsm, handle) };
            return text.into_iter().collect();
        }
        // SAFETY: a handle to an array of `NumItems` handles, per the spec.
        unsafe {
            let array = dsm.lock(handle).cast::<*mut c_void>();
            if array.is_null() {
                return Vec::new();
            }
            let texts = (0..count.min(usize::from(info.NumItems)))
                .filter_map(|i| read_text(dsm, array.add(i).read_unaligned()))
                .collect();
            dsm.unlock(handle);
            texts
        }
    }

    fn free_info_handles(&self, info: &TW_INFO) {
        if return_code(info) != TWRC_SUCCESS || info.ItemType != TWTY_HANDLE || item_of(info) == 0 {
            return;
        }
        let dsm = &self.manager.dsm;
        let handle: *mut c_void = core::ptr::with_exposed_provenance_mut(item_of(info));
        // SAFETY: the handles came from the source and are freed once.
        unsafe {
            if info.NumItems > 1 {
                let array = dsm.lock(handle).cast::<*mut c_void>();
                if !array.is_null() {
                    for i in 0..usize::from(info.NumItems) {
                        let inner = array.add(i).read_unaligned();
                        if !inner.is_null() {
                            dsm.free(inner);
                        }
                    }
                    dsm.unlock(handle);
                }
            }
            dsm.free(handle);
        }
    }
}

impl<D: Dsm> Drop for Source<'_, D> {
    fn drop(&mut self) {
        self.disable();
        if self.state == 4 {
            let rc = self.manager.call(
                None,
                DG_CONTROL,
                DAT_IDENTITY,
                MSG_CLOSEDS,
                self.dest().cast(),
            );
            if rc != TWRC_SUCCESS {
                tracing::warn!(rc, "the TWAIN source did not close cleanly");
            }
            self.state = 3;
        }
    }
}

/// # Safety
///
/// `handle` is null or a live handle holding a NUL-terminated string.
unsafe fn read_text<D: Dsm + ?Sized>(dsm: &D, handle: *mut c_void) -> Option<String> {
    if handle.is_null() {
        return None;
    }
    // SAFETY: per the caller; the read stops at NUL or the length cap.
    unsafe {
        let text = dsm.lock(handle).cast::<u8>();
        if text.is_null() {
            return None;
        }
        let mut bytes = Vec::new();
        while bytes.len() < MAX_BARCODE_LEN {
            let byte = text.add(bytes.len()).read();
            if byte == 0 {
                break;
            }
            bytes.push(byte);
        }
        dsm.unlock(handle);
        let text = String::from_utf8_lossy(&bytes).trim().to_owned();
        (!text.is_empty()).then_some(text)
    }
}

enum Transfer {
    Page(ScannedPage),
    Canceled,
    Condition(ScanCondition),
}

fn buffer_size(setup: &TW_SETUPMEMXFER) -> u32 {
    let preferred = if setup.Preferred == 0 || setup.Preferred == u32::MAX {
        setup.MaxBufSize.min(MAX_BUFFER)
    } else {
        setup.Preferred
    };
    preferred.clamp(setup.MinBufSize.clamp(MIN_BUFFER, MAX_BUFFER), MAX_BUFFER)
}

/// Every capability a scan sets after the transfer mechanism, in the order
/// the specification's negotiation sequence gives: units before resolution,
/// pixel type before bit depth.
fn requested_capabilities(want: &ScanSettings) -> Vec<(u16, u16, CapValue)> {
    let uint = |value: u16| CapValue::Uint(u32::from(value));
    let (twpt, depth) = match want.pixel_type {
        PixelType::Grayscale => (TWPT_GRAY, 8),
        PixelType::Color => (TWPT_RGB, 8),
        PixelType::BlackWhite | PixelType::Unknown => (TWPT_BW, 1),
    };
    let dpi = CapValue::Fix32(f64::from(want.dpi));
    let blank = if want.discard_blank_pages {
        TWBP_AUTO
    } else {
        TWBP_DISABLE
    };

    let mut caps = vec![
        (ICAP_UNITS, TWTY_UINT16, uint(TWUN_INCHES)),
        (ICAP_PIXELTYPE, TWTY_UINT16, uint(twpt)),
        (ICAP_BITDEPTH, TWTY_UINT16, uint(depth)),
        (ICAP_COMPRESSION, TWTY_UINT16, uint(TWCP_NONE)),
    ];
    if want.pixel_type == PixelType::Color {
        caps.push((ICAP_PLANARCHUNKY, TWTY_UINT16, uint(TWPC_CHUNKY)));
    }
    caps.extend([
        (ICAP_XRESOLUTION, TWTY_FIX32, dpi),
        (ICAP_YRESOLUTION, TWTY_FIX32, dpi),
        (
            CAP_FEEDERENABLED,
            TWTY_BOOL,
            CapValue::Bool(want.use_feeder),
        ),
    ]);
    if want.use_feeder {
        caps.push((CAP_AUTOFEED, TWTY_BOOL, CapValue::Bool(true)));
    }
    caps.extend([
        (CAP_DUPLEXENABLED, TWTY_BOOL, CapValue::Bool(want.duplex)),
        (CAP_XFERCOUNT, TWTY_INT16, CapValue::Int(-1)),
        (ICAP_AUTODISCARDBLANKPAGES, TWTY_INT32, CapValue::Int(blank)),
    ]);
    if want.detect_patch_codes {
        caps.push((
            ICAP_PATCHCODEDETECTIONENABLED,
            TWTY_BOOL,
            CapValue::Bool(true),
        ));
    }
    if want.detect_barcodes {
        caps.push((
            ICAP_BARCODEDETECTIONENABLED,
            TWTY_BOOL,
            CapValue::Bool(true),
        ));
    }
    if want.detect_patch_codes || want.detect_barcodes {
        caps.push((ICAP_EXTIMAGEINFO, TWTY_BOOL, CapValue::Bool(true)));
    }
    caps.push((CAP_INDICATORS, TWTY_BOOL, CapValue::Bool(want.show_ui)));
    caps
}

/// What a source condition means for a scan in progress.
fn scan_condition(condition: u16) -> Option<ScanCondition> {
    match condition {
        TWCC_PAPERJAM => Some(ScanCondition::PaperJam),
        TWCC_PAPERDOUBLEFEED => Some(ScanCondition::DoubleFeed),
        TWCC_INTERLOCK => Some(ScanCondition::CoverOpen),
        TWCC_NOMEDIA => Some(ScanCondition::FeederEmpty),
        _ => None,
    }
}

fn pixel_type_of(twpt: u16) -> Option<PixelType> {
    match twpt {
        TWPT_BW => Some(PixelType::BlackWhite),
        TWPT_GRAY => Some(PixelType::Grayscale),
        TWPT_RGB => Some(PixelType::Color),
        _ => None,
    }
}

fn patch_code_name(item: usize) -> Option<&'static str> {
    let code = u32::try_from(item).ok()?;
    let name = match code {
        TWPCH_PATCH1 => "1",
        TWPCH_PATCH2 => "2",
        TWPCH_PATCH3 => "3",
        TWPCH_PATCH4 => "4",
        TWPCH_PATCH6 => "6",
        TWPCH_PATCHT => "T",
        _ => return None,
    };
    known_patch_code(name)
}

/// The shape of the page being transferred.
#[derive(Clone, Copy, Debug)]
struct PageLayout {
    width: u32,
    /// `None` when the source does not know yet, as a feeder with length
    /// detection reports it.
    height: Option<u32>,
    format: PixelFormat,
    bits_per_pixel: u32,
    invert: bool,
    resolution: Resolution,
}

impl PageLayout {
    fn from_info(info: &TW_IMAGEINFO, flavor: Option<CapValue>) -> TwainResult<Self> {
        let width = u32::try_from(info.ImageWidth)
            .ok()
            .filter(|w| (1..=capture_imaging::raster::MAX_SIDE).contains(w))
            .ok_or_else(|| {
                let width = info.ImageWidth;
                TwainError::Unsupported(format!("a page {width} pixels wide"))
            })?;
        let height = u32::try_from(info.ImageLength)
            .ok()
            .filter(|&h| h > 0 && h <= capture_imaging::raster::MAX_SIDE);
        if info.Planar != 0 && info.SamplesPerPixel > 1 {
            return Err(TwainError::Unsupported("planar colour data".into()));
        }
        if info.Compression != TWCP_NONE {
            return Err(TwainError::Unsupported(
                "compressed memory transfers".into(),
            ));
        }

        let chocolate = flavor.is_none_or(|f| f.as_i64() == i64::from(TWPF_CHOCOLATE));
        let pixel_type = u16::from_ne_bytes(info.PixelType.to_ne_bytes());
        let (format, bits_per_pixel, invert) = match (pixel_type, info.BitsPerPixel) {
            (TWPT_BW, 1) => (
                PixelFormat::Bilevel {
                    zero_is_black: chocolate,
                },
                1,
                false,
            ),
            (TWPT_GRAY, 8) => (PixelFormat::Gray8, 8, !chocolate),
            (TWPT_RGB, 24) => (PixelFormat::Rgb8, 24, false),
            (kind, bits) => {
                return Err(TwainError::Unsupported(format!(
                    "{bits}-bit pages of pixel type {kind}"
                )));
            }
        };

        let dpi = |fix| {
            let value = capability::from_fix32(fix).round();
            #[allow(clippy::cast_possible_truncation, clippy::cast_sign_loss)]
            let value = value as u32;
            if (50..=1200).contains(&value) {
                value
            } else {
                300
            }
        };
        Ok(Self {
            width,
            height,
            format,
            bits_per_pixel,
            invert,
            resolution: Resolution {
                x: dpi(info.XResolution),
                y: dpi(info.YResolution),
            },
        })
    }

    fn row_bytes(&self) -> usize {
        self.format.row_bytes(self.width)
    }
}

/// A page assembled from the strips a source sends.
struct PageBuffer {
    layout: PageLayout,
    data: Vec<u8>,
    rows: u32,
}

impl PageBuffer {
    fn new(layout: PageLayout) -> Self {
        let capacity = layout.height.map_or(0, |h| layout.row_bytes() * h as usize);
        Self {
            layout,
            data: Vec::with_capacity(capacity),
            rows: 0,
        }
    }

    /// Copies one strip into place, refusing one that does not fit the page
    /// or the buffer it arrived in.
    fn add_strip(&mut self, strip: &TW_IMAGEMEMXFER, buffer: &[u8]) -> TwainResult<()> {
        let malformed =
            |what: &str| TwainError::Unsupported(format!("a malformed transfer strip: {what}"));
        if strip.Compression != TWCP_NONE {
            return Err(malformed("compressed"));
        }
        let rows = strip.Rows;
        if rows == 0 {
            return Ok(());
        }
        let bytes_per_row = strip.BytesPerRow as usize;
        let columns = strip.Columns;
        let x_offset = strip.XOffset;
        let y_offset = strip.YOffset;
        let bits = self.layout.bits_per_pixel;

        if x_offset
            .checked_add(columns)
            .is_none_or(|end| end > self.layout.width)
        {
            return Err(malformed("wider than the page"));
        }
        let x_bits = u64::from(x_offset) * u64::from(bits);
        if x_bits % 8 != 0 {
            return Err(malformed("not byte aligned"));
        }
        let strip_row_bytes = (u64::from(columns) * u64::from(bits)).div_ceil(8);
        let strip_row_bytes =
            usize::try_from(strip_row_bytes).map_err(|_| malformed("too wide"))?;
        if bytes_per_row < strip_row_bytes {
            return Err(malformed("rows shorter than their pixels"));
        }
        let needed = bytes_per_row
            .checked_mul(rows as usize - 1)
            .and_then(|n| n.checked_add(strip_row_bytes))
            .ok_or_else(|| malformed("too large"))?;
        if needed > buffer.len() || needed > strip.BytesWritten as usize {
            return Err(malformed("more rows than bytes written"));
        }
        let end_row = y_offset
            .checked_add(rows)
            .ok_or_else(|| malformed("too tall"))?;
        let max_rows = self
            .layout
            .height
            .unwrap_or(capture_imaging::raster::MAX_SIDE);
        if end_row > max_rows {
            return Err(malformed("taller than the page"));
        }

        let page_row_bytes = self.layout.row_bytes();
        let needed_len = page_row_bytes * end_row as usize;
        if self.data.len() < needed_len {
            self.data.resize(needed_len, 0);
        }
        let x_byte = usize::try_from(x_bits / 8).map_err(|_| malformed("too wide"))?;
        for r in 0..rows as usize {
            let source = &buffer[r * bytes_per_row..r * bytes_per_row + strip_row_bytes];
            let at = (y_offset as usize + r) * page_row_bytes + x_byte;
            self.data[at..at + strip_row_bytes].copy_from_slice(source);
        }
        self.rows = self.rows.max(end_row);
        Ok(())
    }

    fn finish(mut self) -> TwainResult<OwnedRaster> {
        let height = self.layout.height.unwrap_or(self.rows);
        if height == 0 || self.rows == 0 {
            return Err(TwainError::Unsupported("a page with no rows".into()));
        }
        let row_bytes = self.layout.row_bytes();
        self.data.resize(row_bytes * height as usize, 0);
        if self.layout.invert {
            for byte in &mut self.data {
                *byte = !*byte;
            }
        }
        Ok(OwnedRaster {
            width: self.layout.width,
            height,
            stride: row_bytes,
            format: self.layout.format,
            data: self.data,
        })
    }
}

/// The outcome a source wrote into an extended-info slot. Both members of
/// the union are the same `TW_UINT16`.
/// The item slot of an extended-info entry, which is pointer sized.
fn item_of(info: &TW_INFO) -> usize {
    let item = info.Item;
    usize::try_from(item).unwrap_or(0)
}

fn return_code(info: &TW_INFO) -> u16 {
    // SAFETY: both union members are `TW_UINT16`, so either reading is
    // defined whatever the source wrote.
    unsafe { info.__bindgen_anon_1.ReturnCode }
}
