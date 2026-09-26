//! Listing WIA scanners and scanning from one.

// What `#[implement]` expands to trips these two pedantic lints.
#![allow(clippy::ref_as_ptr, clippy::inline_always)]

use std::cell::RefCell;
use std::rc::Rc;
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};

use capture_imaging::scan::{ScanEnd, ScanSettings, ScannedPage};
use capture_imaging::{PixelFormat, Resolution, binarize, decode_bmp};
use capture_protocol::api::{PixelType, Settings, SourceInfo, SourceProtocol};
use capture_protocol::helper::ScanCondition;
use windows::Win32::Devices::ImageAcquisition::{
    ALL_PAGES, DUPLEX, FEEDER, IEnumWiaItem2, IWiaDevMgr2, IWiaItem2, IWiaPropertyStorage,
    IWiaTransfer, IWiaTransferCallback, IWiaTransferCallback_Impl,
    WIA_BLANK_PAGE_DETECTION_DISABLED, WIA_BLANK_PAGE_DISCARD, WIA_CATEGORY_FEEDER,
    WIA_CATEGORY_FEEDER_FRONT, WIA_CATEGORY_FLATBED, WIA_DATA_COLOR, WIA_DATA_GRAYSCALE,
    WIA_DATA_THRESHOLD, WIA_DEVINFO_ENUM_LOCAL, WIA_DIP_DEV_ID, WIA_DIP_DEV_NAME, WIA_DIP_DEV_TYPE,
    WIA_DIP_DRIVER_VERSION, WIA_DPS_DOCUMENT_HANDLING_CAPABILITIES, WIA_ERROR_BUSY,
    WIA_ERROR_COVER_OPEN, WIA_ERROR_OFFLINE, WIA_ERROR_PAPER_EMPTY, WIA_ERROR_PAPER_JAM,
    WIA_ERROR_PAPER_PROBLEM, WIA_IPA_DATATYPE, WIA_IPA_DEPTH, WIA_IPA_FORMAT, WIA_IPA_TYMED,
    WIA_IPS_BLANK_PAGES, WIA_IPS_DOCUMENT_HANDLING_SELECT, WIA_IPS_PAGES, WIA_IPS_XRES,
    WIA_IPS_YRES, WIA_TRANSFER_MSG_DEVICE_STATUS, WIA_TRANSFER_MSG_END_OF_STREAM, WiaDevMgr2,
    WiaImgFmt_BMP, WiaTransferParams,
};
use windows::Win32::Foundation::{E_POINTER, HGLOBAL, RPC_E_CHANGED_MODE, S_FALSE};
use windows::Win32::System::Com::StructuredStorage::CreateStreamOnHGlobal;
use windows::Win32::System::Com::{
    CLSCTX_LOCAL_SERVER, COINIT_APARTMENTTHREADED, CoCreateInstance, CoInitializeEx,
    CoUninitialize, IStream, STATFLAG_NONAME, STATSTG, STREAM_SEEK_SET, TYMED_FILE,
};
use windows_core::{BSTR, GUID, HRESULT, Interface, implement};

use crate::props::{self, Allowed};
use crate::{WiaError, WiaResult};

/// `STI_DEVICE_MJ_TYPE` for a scanner, in the high word of `WIA_DIP_DEV_TYPE`.
const STI_DEVICE_TYPE_SCANNER: i32 = 1;
/// Resolutions offered when a driver reports a range.
const STANDARD_RESOLUTIONS: [i32; 7] = [100, 150, 200, 240, 300, 400, 600];
/// The biggest page stream accepted: a legal page at 600 DPI in 24-bit
/// colour is about 60 MB as a bitmap.
const MAX_STREAM_BYTES: u64 = 256 << 20;

/// What a helper hands each page to.
pub type PageSink = Box<dyn FnMut(ScannedPage) -> Result<(), String> + Send>;

fn com(operation: &'static str) -> impl FnOnce(windows_core::Error) -> WiaError {
    move |err| match err.code() {
        WIA_ERROR_OFFLINE => WiaError::Offline,
        WIA_ERROR_BUSY => WiaError::Busy,
        _ => WiaError::Com {
            operation,
            message: err.message(),
        },
    }
}

/// Keeps COM initialised on this thread for as long as it lives, unless the
/// thread had already chosen a different model, which it then leaves alone.
struct ComApartment {
    owned: bool,
}

impl ComApartment {
    fn enter() -> WiaResult<Self> {
        // SAFETY: initialising COM on the calling thread.
        let hr = unsafe { CoInitializeEx(None, COINIT_APARTMENTTHREADED) };
        if hr == RPC_E_CHANGED_MODE {
            return Ok(Self { owned: false });
        }
        hr.ok().map_err(com("start COM"))?;
        Ok(Self { owned: true })
    }
}

impl Drop for ComApartment {
    fn drop(&mut self) {
        if self.owned {
            // SAFETY: balances the successful initialisation in `enter`.
            unsafe { CoUninitialize() };
        }
    }
}

fn manager() -> WiaResult<IWiaDevMgr2> {
    // SAFETY: creating the WIA 2.0 device manager, an out-of-process server.
    unsafe { CoCreateInstance(&WiaDevMgr2, None, CLSCTX_LOCAL_SERVER) }
        .map_err(com("open the WIA device manager"))
}

struct Device {
    id: String,
    name: String,
    driver_version: String,
}

/// Every installed scanner, by the properties the device manager keeps for
/// it without opening it.
fn devices(manager: &IWiaDevMgr2) -> WiaResult<Vec<Device>> {
    // SAFETY: enumerating local devices.
    let list =
        unsafe { manager.EnumDeviceInfo(i32::try_from(WIA_DEVINFO_ENUM_LOCAL).unwrap_or(16)) }
            .map_err(com("list scanners"))?;
    let mut devices = Vec::new();
    loop {
        let mut item: [Option<IWiaPropertyStorage>; 1] = [None];
        let mut fetched = 0u32;
        // SAFETY: room for one item, as the count says.
        unsafe { list.Next(1, item.as_mut_ptr(), &raw mut fetched) }
            .map_err(com("list scanners"))?;
        let Some(storage) = item[0].take().filter(|_| fetched == 1) else {
            break;
        };
        let is_scanner = props::read_i32(&storage, WIA_DIP_DEV_TYPE)
            .is_some_and(|kind| kind >> 16 == STI_DEVICE_TYPE_SCANNER);
        let id = props::read_string(&storage, WIA_DIP_DEV_ID);
        let name = props::read_string(&storage, WIA_DIP_DEV_NAME);
        if let (true, Some(id), Some(name)) = (is_scanner, id, name) {
            devices.push(Device {
                id,
                name: name.trim().to_owned(),
                driver_version: props::read_string(&storage, WIA_DIP_DRIVER_VERSION)
                    .unwrap_or_default(),
            });
        }
    }
    Ok(devices)
}

fn open(manager: &IWiaDevMgr2, device: &Device) -> WiaResult<IWiaItem2> {
    // SAFETY: opening a device the manager just listed.
    unsafe { manager.CreateDevice(0, &BSTR::from(device.id.as_str())) }
        .map_err(com("open the scanner"))
}

fn storage(item: &IWiaItem2) -> WiaResult<IWiaPropertyStorage> {
    item.cast().map_err(com("read the scanner's settings"))
}

/// The device's scan surfaces: its feeder and its flatbed, if it has them.
struct Surfaces {
    feeder: Option<IWiaItem2>,
    flatbed: Option<IWiaItem2>,
}

fn surfaces(root: &IWiaItem2) -> WiaResult<Surfaces> {
    // SAFETY: listing the root item's children.
    let children: IEnumWiaItem2 =
        unsafe { root.EnumChildItems(None) }.map_err(com("list the scanner's parts"))?;
    let mut found = Surfaces {
        feeder: None,
        flatbed: None,
    };
    loop {
        let mut item: [Option<IWiaItem2>; 1] = [None];
        let mut fetched = 0u32;
        // SAFETY: room for one item, as the count says.
        unsafe { children.Next(1, item.as_mut_ptr(), &raw mut fetched) }
            .map_err(com("list the scanner's parts"))?;
        let Some(child) = item[0].take().filter(|_| fetched == 1) else {
            break;
        };
        // SAFETY: reading the category of a child just listed.
        let category: GUID = unsafe { child.GetItemCategory() }.unwrap_or_default();
        if (category == WIA_CATEGORY_FEEDER || category == WIA_CATEGORY_FEEDER_FRONT)
            && found.feeder.is_none()
        {
            found.feeder = Some(child);
        } else if category == WIA_CATEGORY_FLATBED && found.flatbed.is_none() {
            found.flatbed = Some(child);
        }
    }
    Ok(found)
}

fn pixel_types(allowed: &Allowed) -> Vec<PixelType> {
    let values = match allowed {
        Allowed::List(values) => values.clone(),
        Allowed::Flags(mask) => vec![*mask],
        Allowed::Range { .. } | Allowed::Unknown => Vec::new(),
    };
    let mut types = Vec::new();
    for (data_type, pixel) in [
        (WIA_DATA_THRESHOLD, PixelType::BlackWhite),
        (WIA_DATA_GRAYSCALE, PixelType::Grayscale),
        (WIA_DATA_COLOR, PixelType::Color),
    ] {
        if values
            .iter()
            .any(|v| u32::try_from(*v).is_ok_and(|v| v == data_type))
        {
            types.push(pixel);
        }
    }
    types
}

fn describe(manager: &IWiaDevMgr2, device: &Device) -> SourceInfo {
    let mut info = SourceInfo {
        name: device.name.clone(),
        protocol: SourceProtocol::Wia,
        bitness: 64,
        is_default: false,
        duplex: false,
        feeder: false,
        patch_codes: false,
        barcodes: false,
        blank_discard: false,
        resolutions: Vec::new(),
        pixel_types: Vec::new(),
    };
    let Ok(root) = open(manager, device) else {
        return info;
    };
    let Ok(parts) = surfaces(&root) else {
        return info;
    };
    if let Ok(root_storage) = storage(&root) {
        let handling =
            props::read_i32(&root_storage, WIA_DPS_DOCUMENT_HANDLING_CAPABILITIES).unwrap_or(0);
        let handling = u32::try_from(handling).unwrap_or(0);
        info.duplex = handling & DUPLEX != 0;
        info.feeder = parts.feeder.is_some() || handling & FEEDER != 0;
    }
    if let Some(item) = parts.feeder.as_ref().or(parts.flatbed.as_ref())
        && let Ok(item_storage) = storage(item)
    {
        let mut resolutions: Vec<u32> = props::allowed(&item_storage, WIA_IPS_XRES)
            .values(&STANDARD_RESOLUTIONS)
            .into_iter()
            .filter_map(|v| u32::try_from(v).ok())
            .filter(|v| (50..=1200).contains(v))
            .collect();
        resolutions.sort_unstable();
        resolutions.dedup();
        info.resolutions = resolutions;
        info.pixel_types = pixel_types(&props::allowed(&item_storage, WIA_IPA_DATATYPE));
        info.blank_discard = matches!(
            props::allowed(&item_storage, WIA_IPS_BLANK_PAGES),
            Allowed::List(ref values) if values.iter().any(|v| u32::try_from(*v).is_ok_and(|v| v == WIA_BLANK_PAGE_DISCARD))
        );
    }
    info
}

/// Every WIA scanner, with what it can do. Opening a WIA device shows no
/// window, so unlike TWAIN each is asked directly.
pub fn sources() -> WiaResult<Vec<SourceInfo>> {
    let _apartment = ComApartment::enter()?;
    let manager = manager()?;
    Ok(devices(&manager)?
        .iter()
        .map(|device| describe(&manager, device))
        .collect())
}

/// The transfer as it goes, shared with the callback WIA calls into.
struct Progress {
    stream: Option<IStream>,
    pages: u32,
    sink: PageSink,
    delivery: Option<String>,
    device_error: Option<HRESULT>,
    bilevel: bool,
    dpi: u32,
    cancel: Arc<AtomicBool>,
}

/// WIA calls this on the scanning thread, which is a single-threaded
/// apartment, so the progress it shares needs no lock.
#[implement(IWiaTransferCallback)]
struct Callback {
    progress: Rc<RefCell<Progress>>,
}

fn read_stream(stream: &IStream) -> Result<Vec<u8>, String> {
    let mut stat = STATSTG::default();
    // SAFETY: the stream this callback created; `stat` is ours.
    unsafe {
        stream
            .Stat(&raw mut stat, STATFLAG_NONAME)
            .map_err(|e| e.message())?;
        stream
            .Seek(0, STREAM_SEEK_SET, None)
            .map_err(|e| e.message())?;
    }
    if stat.cbSize > MAX_STREAM_BYTES {
        return Err(format!("a page of {} bytes is too large", stat.cbSize));
    }
    let len = usize::try_from(stat.cbSize).map_err(|_| "a page too large to hold".to_owned())?;
    let mut bytes = vec![0u8; len];
    let mut filled = 0usize;
    while filled < len {
        let mut read = 0u32;
        let chunk = u32::try_from(len - filled).unwrap_or(u32::MAX);
        // SAFETY: reading into the unfilled rest of the buffer.
        let hr = unsafe {
            stream.Read(
                bytes[filled..].as_mut_ptr().cast(),
                chunk,
                Some(&raw mut read),
            )
        };
        if hr.is_err() {
            return Err(windows_core::Error::from(hr).message());
        }
        if read == 0 {
            break;
        }
        filled += read as usize;
    }
    bytes.truncate(filled);
    Ok(bytes)
}

impl Progress {
    fn finish_page(&mut self) -> Result<(), String> {
        let stream = self.stream.take().ok_or("a page ended before it began")?;
        let bytes = read_stream(&stream)?;
        let bitmap = decode_bmp(&bytes).map_err(|e| e.to_string())?;
        let raster = if self.bilevel && !matches!(bitmap.raster.format, PixelFormat::Bilevel { .. })
        {
            binarize(&bitmap.raster.as_raster().map_err(|e| e.to_string())?)
        } else {
            bitmap.raster
        };
        let resolution = bitmap
            .dpi
            .map_or(Resolution::square(self.dpi), |(x, y)| Resolution { x, y });
        self.pages += 1;
        (self.sink)(ScannedPage {
            raster,
            resolution,
            patch_code: None,
            barcodes: Vec::new(),
        })
    }
}

impl IWiaTransferCallback_Impl for Callback_Impl {
    fn TransferCallback(
        &self,
        _flags: i32,
        params: *const WiaTransferParams,
    ) -> windows_core::Result<()> {
        // SAFETY: WIA passes a valid parameter block for the call's duration.
        let params =
            unsafe { params.as_ref() }.ok_or_else(|| windows_core::Error::from(E_POINTER))?;
        let mut progress = self.progress.borrow_mut();
        if progress.cancel.load(Ordering::Acquire) || progress.delivery.is_some() {
            return Err(S_FALSE.into());
        }
        match u32::try_from(params.lMessage).unwrap_or(0) {
            WIA_TRANSFER_MSG_END_OF_STREAM => {
                if let Err(err) = progress.finish_page() {
                    progress.delivery = Some(err);
                    return Err(S_FALSE.into());
                }
            }
            WIA_TRANSFER_MSG_DEVICE_STATUS if params.hrErrorStatus.is_err() => {
                progress.device_error = Some(params.hrErrorStatus);
            }
            _ => {}
        }
        Ok(())
    }

    fn GetNextStream(
        &self,
        _flags: i32,
        _name: &BSTR,
        _full_name: &BSTR,
    ) -> windows_core::Result<IStream> {
        // SAFETY: a new in-memory stream that frees its memory on release.
        let stream = unsafe { CreateStreamOnHGlobal(HGLOBAL::default(), true) }?;
        let mut progress = self.progress.borrow_mut();
        progress.stream = Some(stream.clone());
        Ok(stream)
    }
}

/// Sets one property, noting it by name if the driver refused it.
fn set_i32(
    refused: &mut Vec<String>,
    storage: &IWiaPropertyStorage,
    id: u32,
    name: &str,
    value: i32,
) {
    if !props::write_i32(storage, id, value) {
        refused.push(name.to_owned());
    }
}

/// Applies the profile to a scan surface, recording what the driver refused.
fn negotiate(
    root: &IWiaPropertyStorage,
    item: &IWiaPropertyStorage,
    feeder: bool,
    want: &ScanSettings,
) -> Vec<String> {
    let mut refused = Vec::new();

    let (data_type, depth) = match want.pixel_type {
        PixelType::Grayscale => (WIA_DATA_GRAYSCALE, 8),
        PixelType::Color => (WIA_DATA_COLOR, 24),
        PixelType::BlackWhite | PixelType::Unknown => (WIA_DATA_THRESHOLD, 1),
    };
    set_i32(
        &mut refused,
        item,
        WIA_IPA_DATATYPE,
        "WIA_IPA_DATATYPE",
        i32::try_from(data_type).unwrap_or(0),
    );
    set_i32(&mut refused, item, WIA_IPA_DEPTH, "WIA_IPA_DEPTH", depth);

    let dpi = i32::try_from(want.dpi).unwrap_or(300);
    let x = props::allowed(item, WIA_IPS_XRES).nearest(dpi);
    let y = props::allowed(item, WIA_IPS_YRES).nearest(dpi);
    set_i32(&mut refused, item, WIA_IPS_XRES, "WIA_IPS_XRES", x);
    set_i32(&mut refused, item, WIA_IPS_YRES, "WIA_IPS_YRES", y);

    if !props::write_guid(item, WIA_IPA_FORMAT, WiaImgFmt_BMP) {
        refused.push("WIA_IPA_FORMAT".to_owned());
    }
    set_i32(
        &mut refused,
        item,
        WIA_IPA_TYMED,
        "WIA_IPA_TYMED",
        TYMED_FILE.0,
    );

    if feeder {
        let mut handling = FEEDER;
        if want.duplex {
            handling |= DUPLEX;
        }
        let handling = i32::try_from(handling).unwrap_or(1);
        if !props::write_i32(item, WIA_IPS_DOCUMENT_HANDLING_SELECT, handling)
            && !props::write_i32(root, WIA_IPS_DOCUMENT_HANDLING_SELECT, handling)
        {
            refused.push("WIA_IPS_DOCUMENT_HANDLING_SELECT".to_owned());
        }
        set_i32(
            &mut refused,
            item,
            WIA_IPS_PAGES,
            "WIA_IPS_PAGES",
            i32::try_from(ALL_PAGES).unwrap_or(0),
        );
    }

    let blank = if want.discard_blank_pages {
        WIA_BLANK_PAGE_DISCARD
    } else {
        WIA_BLANK_PAGE_DETECTION_DISABLED
    };
    if want.discard_blank_pages
        && !props::write_i32(item, WIA_IPS_BLANK_PAGES, i32::try_from(blank).unwrap_or(0))
    {
        refused.push("WIA_IPS_BLANK_PAGES".to_owned());
    }
    if want.detect_patch_codes {
        refused.push("ICAP_PATCHCODEDETECTIONENABLED".to_owned());
    }
    if want.detect_barcodes {
        refused.push("ICAP_BARCODEDETECTIONENABLED".to_owned());
    }
    if want.show_ui {
        refused.push("WIA_DEVICE_DIALOG".to_owned());
    }
    refused.sort();
    refused.dedup();
    refused
}

/// Scans from a WIA scanner: `on_started` gets the settings in force before
/// the first page, and `sink` each page as it arrives. Returns how the scan
/// ended.
pub fn scan(
    name: &str,
    want: &ScanSettings,
    cancel: &Arc<AtomicBool>,
    on_started: &mut dyn FnMut(&Settings) -> Result<(), String>,
    sink: PageSink,
) -> WiaResult<ScanEnd> {
    let _apartment = ComApartment::enter()?;
    let manager = manager()?;
    let wanted = name.trim();
    let device = devices(&manager)?
        .into_iter()
        .find(|d| d.name == wanted)
        .ok_or_else(|| WiaError::SourceNotFound(wanted.to_owned()))?;
    let root = open(&manager, &device)?;
    let parts = surfaces(&root)?;
    let (item, feeder) = match (want.use_feeder, parts.feeder, parts.flatbed) {
        (true, Some(feeder), _) | (false, Some(feeder), None) => (feeder, true),
        (_, _, Some(flatbed)) => (flatbed, false),
        (_, None, None) => return Err(WiaError::NoScanSurface),
    };
    let root_storage = storage(&root)?;
    let item_storage = storage(&item)?;
    let refused = negotiate(&root_storage, &item_storage, feeder, want);

    let dpi = props::read_i32(&item_storage, WIA_IPS_XRES)
        .and_then(|v| u32::try_from(v).ok())
        .unwrap_or(want.dpi);
    let data_type = props::read_i32(&item_storage, WIA_IPA_DATATYPE)
        .and_then(|v| u32::try_from(v).ok())
        .unwrap_or(WIA_DATA_THRESHOLD);
    let pixel_type = match data_type {
        WIA_DATA_GRAYSCALE => PixelType::Grayscale,
        WIA_DATA_COLOR => PixelType::Color,
        _ => PixelType::BlackWhite,
    };
    let handling = props::read_i32(&item_storage, WIA_IPS_DOCUMENT_HANDLING_SELECT)
        .or_else(|| props::read_i32(&root_storage, WIA_IPS_DOCUMENT_HANDLING_SELECT))
        .and_then(|v| u32::try_from(v).ok())
        .unwrap_or(0);
    let blank_discard = props::read_i32(&item_storage, WIA_IPS_BLANK_PAGES)
        .is_some_and(|v| u32::try_from(v).is_ok_and(|v| v == WIA_BLANK_PAGE_DISCARD));

    let settings = Settings {
        protocol: Some(SourceProtocol::Wia),
        bitness: 64,
        dpi,
        pixel_type: Some(pixel_type),
        duplex: feeder && handling & DUPLEX != 0,
        feeder,
        blank_discard,
        show_driver_ui: false,
        driver_version: device.driver_version.clone(),
        application: String::new(),
        refused,
    };
    on_started(&settings).map_err(WiaError::Delivery)?;

    let progress = Rc::new(RefCell::new(Progress {
        stream: None,
        pages: 0,
        sink,
        delivery: None,
        device_error: None,
        bilevel: want.pixel_type == PixelType::BlackWhite,
        dpi,
        cancel: Arc::clone(cancel),
    }));
    let callback: IWiaTransferCallback = Callback {
        progress: Rc::clone(&progress),
    }
    .into();
    let transfer: IWiaTransfer = item.cast().map_err(com("start the transfer"))?;
    // SAFETY: a synchronous download into the callback above.
    let result = unsafe { transfer.Download(0, &callback) };
    drop(callback);

    let progress = progress.borrow();
    if let Some(delivery) = &progress.delivery {
        return Err(WiaError::Delivery(delivery.clone()));
    }
    let pages = progress.pages;
    let error = result.err().map(|e| e.code()).or(progress.device_error);
    let end = match error {
        _ if cancel.load(Ordering::Acquire) => ScanEnd::Canceled { pages },
        None => ScanEnd::Finished { pages },
        Some(code) if code == S_FALSE => ScanEnd::Canceled { pages },
        Some(WIA_ERROR_PAPER_EMPTY) if pages > 0 => ScanEnd::Finished { pages },
        Some(WIA_ERROR_PAPER_EMPTY) => ScanEnd::Stopped {
            condition: ScanCondition::FeederEmpty,
            pages,
        },
        Some(WIA_ERROR_PAPER_JAM) => ScanEnd::Stopped {
            condition: ScanCondition::PaperJam,
            pages,
        },
        Some(WIA_ERROR_PAPER_PROBLEM) => ScanEnd::Stopped {
            condition: ScanCondition::DoubleFeed,
            pages,
        },
        Some(WIA_ERROR_COVER_OPEN) => ScanEnd::Stopped {
            condition: ScanCondition::CoverOpen,
            pages,
        },
        Some(code) => return Err(com("scan")(windows_core::Error::from(code))),
    };
    Ok(end)
}
