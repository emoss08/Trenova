//! Whether a file is a Trenova release: Windows verifies its Authenticode
//! signature (a valid chain to a trusted root, a signature over the file as
//! it is), and its signer must be the same publisher as the running updater,
//! which the release build signed with Trenova's certificate. Any file that
//! passes both is one Trenova built.

use std::path::Path;

use trenova_capture_update::UpdaterError;
use windows::Win32::Foundation::{HANDLE, HWND, INVALID_HANDLE_VALUE};
use windows::Win32::Security::Cryptography::{
    CERT_CONTEXT, CERT_FIND_SUBJECT_CERT, CERT_INFO, CERT_NAME_SIMPLE_DISPLAY_TYPE,
    CERT_QUERY_CONTENT_FLAG_PKCS7_SIGNED_EMBED, CERT_QUERY_FORMAT_FLAG_BINARY,
    CERT_QUERY_OBJECT_FILE, CMSG_SIGNER_INFO, CMSG_SIGNER_INFO_PARAM, CertCloseStore,
    CertFindCertificateInStore, CertFreeCertificateContext, CertGetNameStringW, CryptMsgClose,
    CryptMsgGetParam, CryptQueryObject, HCERTSTORE, PKCS_7_ASN_ENCODING, X509_ASN_ENCODING,
};
use windows::Win32::Security::WinTrust::{
    WINTRUST_ACTION_GENERIC_VERIFY_V2, WINTRUST_DATA, WINTRUST_DATA_0, WINTRUST_FILE_INFO,
    WTD_CHOICE_FILE, WTD_REVOCATION_CHECK_CHAIN_EXCLUDE_ROOT, WTD_REVOKE_NONE,
    WTD_STATEACTION_CLOSE, WTD_STATEACTION_VERIFY, WTD_UI_NONE, WTD_UICONTEXT_INSTALL,
    WinVerifyTrust,
};
use windows_core::{HSTRING, PCWSTR};

fn size_of<T>() -> u32 {
    u32::try_from(std::mem::size_of::<T>()).unwrap_or(0)
}

/// Asks Windows whether the file carries a valid Authenticode signature.
pub fn verify_authenticode(path: &Path) -> Result<(), UpdaterError> {
    let wide = HSTRING::from(path.as_os_str());
    let file = WINTRUST_FILE_INFO {
        cbStruct: size_of::<WINTRUST_FILE_INFO>(),
        pcwszFilePath: PCWSTR(wide.as_ptr()),
        hFile: HANDLE::default(),
        pgKnownSubject: std::ptr::null_mut(),
    };
    let mut data = WINTRUST_DATA {
        cbStruct: size_of::<WINTRUST_DATA>(),
        dwUIChoice: WTD_UI_NONE,
        fdwRevocationChecks: WTD_REVOKE_NONE,
        dwUnionChoice: WTD_CHOICE_FILE,
        Anonymous: WINTRUST_DATA_0 {
            pFile: (&raw const file).cast_mut(),
        },
        dwStateAction: WTD_STATEACTION_VERIFY,
        dwProvFlags: WTD_REVOCATION_CHECK_CHAIN_EXCLUDE_ROOT,
        dwUIContext: WTD_UICONTEXT_INSTALL,
        ..WINTRUST_DATA::default()
    };
    let mut action = WINTRUST_ACTION_GENERIC_VERIFY_V2;
    // SAFETY: `file` outlives both calls, `data` points at it, and the
    // invalid window handle means no UI. The second call releases the state
    // the first opened.
    let status = unsafe {
        let status = WinVerifyTrust(
            HWND(INVALID_HANDLE_VALUE.0),
            &raw mut action,
            (&raw mut data).cast(),
        );
        data.dwStateAction = WTD_STATEACTION_CLOSE;
        let _ = WinVerifyTrust(
            HWND(INVALID_HANDLE_VALUE.0),
            &raw mut action,
            (&raw mut data).cast(),
        );
        status
    };
    if status == 0 {
        Ok(())
    } else {
        Err(UpdaterError::Untrusted(format!(
            "Windows did not accept its signature (0x{status:08x})"
        )))
    }
}

struct Signature {
    store: HCERTSTORE,
    message: *mut core::ffi::c_void,
}

impl Drop for Signature {
    fn drop(&mut self) {
        // SAFETY: handles CryptQueryObject returned, closed once.
        unsafe {
            let _ = CryptMsgClose(Some(self.message));
            let _ = CertCloseStore(Some(self.store), 0);
        }
    }
}

struct Certificate(*mut CERT_CONTEXT);

impl Drop for Certificate {
    fn drop(&mut self) {
        // SAFETY: a context CertFindCertificateInStore returned, freed once.
        unsafe {
            let _ = CertFreeCertificateContext(Some(self.0));
        }
    }
}

/// The simple display name of the certificate that signed the file: the
/// publisher's name as the certificate authority verified it.
pub fn signer_name(path: &Path) -> Result<String, UpdaterError> {
    let untrusted = |what: &str| UpdaterError::Untrusted(format!("{what} ({})", path.display()));
    let wide = HSTRING::from(path.as_os_str());
    let mut store = HCERTSTORE::default();
    let mut message: *mut core::ffi::c_void = std::ptr::null_mut();
    // SAFETY: a NUL-terminated path and out handles.
    unsafe {
        CryptQueryObject(
            CERT_QUERY_OBJECT_FILE,
            wide.as_ptr().cast(),
            CERT_QUERY_CONTENT_FLAG_PKCS7_SIGNED_EMBED,
            CERT_QUERY_FORMAT_FLAG_BINARY,
            0,
            None,
            None,
            None,
            Some(&raw mut store),
            Some(&raw mut message),
            None,
        )
    }
    .map_err(|_| untrusted("the file carries no signature"))?;
    let signature = Signature { store, message };

    let mut size = 0u32;
    // SAFETY: asks for the size of the signer information only.
    unsafe {
        CryptMsgGetParam(
            signature.message,
            CMSG_SIGNER_INFO_PARAM,
            0,
            None,
            &raw mut size,
        )
    }
    .map_err(|_| untrusted("the signature names no signer"))?;
    let mut buffer = vec![0u64; (size as usize).div_ceil(std::mem::size_of::<u64>()).max(1)];
    // SAFETY: an aligned buffer of at least `size` bytes.
    unsafe {
        CryptMsgGetParam(
            signature.message,
            CMSG_SIGNER_INFO_PARAM,
            0,
            Some(buffer.as_mut_ptr().cast()),
            &raw mut size,
        )
    }
    .map_err(|_| untrusted("the signer information is unreadable"))?;
    // SAFETY: the buffer holds a CMSG_SIGNER_INFO and is aligned for it.
    let signer = unsafe { &*buffer.as_ptr().cast::<CMSG_SIGNER_INFO>() };

    let wanted = CERT_INFO {
        Issuer: signer.Issuer,
        SerialNumber: signer.SerialNumber,
        ..CERT_INFO::default()
    };
    // SAFETY: the store the signature opened and a CERT_INFO naming the
    // signer's issuer and serial number, which is what the find type reads.
    let found = unsafe {
        CertFindCertificateInStore(
            signature.store,
            X509_ASN_ENCODING | PKCS_7_ASN_ENCODING,
            0,
            CERT_FIND_SUBJECT_CERT,
            Some((&raw const wanted).cast()),
            None,
        )
    };
    if found.is_null() {
        return Err(untrusted(
            "the signer's certificate is not in the signature",
        ));
    }
    let certificate = Certificate(found);

    let mut name = vec![0u16; 512];
    // SAFETY: a valid certificate context and a buffer with its length.
    let written = unsafe {
        CertGetNameStringW(
            certificate.0,
            CERT_NAME_SIMPLE_DISPLAY_TYPE,
            0,
            None,
            Some(&mut name),
        )
    };
    if written <= 1 {
        return Err(untrusted("the signer's certificate has no name"));
    }
    Ok(String::from_utf16_lossy(&name[..written as usize - 1]))
}

/// Checks that `installer` is validly signed by the same publisher as
/// `reference`, the running updater.
pub fn verify_same_publisher(installer: &Path, reference: &Path) -> Result<(), UpdaterError> {
    verify_authenticode(installer)?;
    let ours = signer_name(reference)?;
    let theirs = signer_name(installer)?;
    if ours != theirs {
        return Err(UpdaterError::Untrusted(format!(
            "signed by {theirs:?}, not by {ours:?}"
        )));
    }
    Ok(())
}
