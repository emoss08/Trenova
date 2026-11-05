//! Whether a file is a Trenova release: Windows verifies its Authenticode
//! signature (a valid chain to a trusted root, nothing in it revoked, a
//! signature over the file as it is), and its signer must be the same
//! publisher as the running updater, which the release build signed with
//! Trenova's certificate. What counts as either is decided in
//! [`trenova_capture_update::signature`]; this reads what it decides on.

use std::path::Path;

use trenova_capture_update::UpdaterError;
use trenova_capture_update::signature::{Publisher, Revocation, check_signature, same_publisher};
use windows::Win32::Foundation::{HANDLE, HWND, INVALID_HANDLE_VALUE};
use windows::Win32::Security::Cryptography::{
    CERT_CONTEXT, CERT_FIND_EXT_ONLY_ENHKEY_USAGE_FLAG, CERT_FIND_SUBJECT_CERT, CERT_INFO,
    CERT_NAME_ATTR_TYPE, CERT_NAME_SIMPLE_DISPLAY_TYPE, CERT_QUERY_CONTENT_FLAG_PKCS7_SIGNED_EMBED,
    CERT_QUERY_FORMAT_FLAG_BINARY, CERT_QUERY_OBJECT_FILE, CMSG_SIGNER_INFO,
    CMSG_SIGNER_INFO_PARAM, CTL_USAGE, CertCloseStore, CertFindCertificateInStore,
    CertFreeCertificateContext, CertGetEnhancedKeyUsage, CertGetNameStringW, CryptMsgClose,
    CryptMsgGetParam, CryptQueryObject, HCERTSTORE, PKCS_7_ASN_ENCODING, X509_ASN_ENCODING,
    szOID_COUNTRY_NAME, szOID_ORGANIZATION_NAME,
};
use windows::Win32::Security::WinTrust::{
    WINTRUST_ACTION_GENERIC_VERIFY_V2, WINTRUST_DATA, WINTRUST_DATA_0, WINTRUST_FILE_INFO,
    WTD_CHOICE_FILE, WTD_REVOCATION_CHECK_CHAIN_EXCLUDE_ROOT, WTD_REVOCATION_CHECK_NONE,
    WTD_REVOKE_NONE, WTD_REVOKE_WHOLECHAIN, WTD_STATEACTION_CLOSE, WTD_STATEACTION_VERIFY,
    WTD_UI_NONE, WTD_UICONTEXT_INSTALL, WinVerifyTrust,
};
use windows_core::{HSTRING, PCSTR, PCWSTR};

fn size_of<T>() -> u32 {
    u32::try_from(std::mem::size_of::<T>()).unwrap_or(0)
}

/// Asks Windows about the file's Authenticode signature and returns its
/// status, with revocation checked for every certificate below the root when
/// `revocation` is set.
fn trust_status(path: &Path, revocation: bool) -> u32 {
    let wide = HSTRING::from(path.as_os_str());
    let file = WINTRUST_FILE_INFO {
        cbStruct: size_of::<WINTRUST_FILE_INFO>(),
        pcwszFilePath: PCWSTR(wide.as_ptr()),
        hFile: HANDLE::default(),
        pgKnownSubject: std::ptr::null_mut(),
    };
    let (checks, flags) = if revocation {
        (
            WTD_REVOKE_WHOLECHAIN,
            WTD_REVOCATION_CHECK_CHAIN_EXCLUDE_ROOT,
        )
    } else {
        (WTD_REVOKE_NONE, WTD_REVOCATION_CHECK_NONE)
    };
    let mut data = WINTRUST_DATA {
        cbStruct: size_of::<WINTRUST_DATA>(),
        dwUIChoice: WTD_UI_NONE,
        fdwRevocationChecks: checks,
        dwUnionChoice: WTD_CHOICE_FILE,
        Anonymous: WINTRUST_DATA_0 {
            pFile: (&raw const file).cast_mut(),
        },
        dwStateAction: WTD_STATEACTION_VERIFY,
        dwProvFlags: flags,
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
    status.cast_unsigned()
}

/// Asks Windows whether the file carries a valid Authenticode signature
/// whose certificates are not revoked.
fn verify_authenticode(path: &Path) -> Result<Revocation, UpdaterError> {
    check_signature(|revocation| trust_status(path, revocation))
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

/// A name from the certificate: its simple display name, or the subject
/// attribute `attribute` names. Empty when it has none.
fn name_of(certificate: &Certificate, kind: u32, attribute: Option<PCSTR>) -> String {
    let mut name = vec![0u16; 512];
    // SAFETY: a valid certificate context, an attribute OID that is a static
    // NUL-terminated string when given, and a buffer with its length.
    let written = unsafe {
        CertGetNameStringW(
            certificate.0,
            kind,
            0,
            attribute.map(|oid| oid.0.cast()),
            Some(&mut name),
        )
    };
    if written <= 1 {
        return String::new();
    }
    String::from_utf16_lossy(&name[..written as usize - 1])
}

/// The extended key usages the certificate's own extension lists.
fn key_usages(certificate: &Certificate) -> Result<Vec<String>, UpdaterError> {
    let unreadable = || UpdaterError::Untrusted("the signer's key usages could not be read".into());
    let flags = CERT_FIND_EXT_ONLY_ENHKEY_USAGE_FLAG.0;
    let mut size = 0u32;
    // SAFETY: asks for the size only.
    unsafe { CertGetEnhancedKeyUsage(certificate.0, flags, None, &raw mut size) }
        .map_err(|_| unreadable())?;
    let mut buffer = vec![0u64; (size as usize).div_ceil(std::mem::size_of::<u64>()).max(1)];
    // SAFETY: an aligned buffer of at least `size` bytes.
    unsafe {
        CertGetEnhancedKeyUsage(
            certificate.0,
            flags,
            Some(buffer.as_mut_ptr().cast()),
            &raw mut size,
        )
    }
    .map_err(|_| unreadable())?;
    // SAFETY: the buffer now holds a CTL_USAGE, and is aligned for it.
    let usage = unsafe { &*buffer.as_ptr().cast::<CTL_USAGE>() };
    if usage.cUsageIdentifier == 0 || usage.rgpszUsageIdentifier.is_null() {
        return Ok(Vec::new());
    }
    // SAFETY: `cUsageIdentifier` pointers into the same buffer, alive here.
    let identifiers = unsafe {
        std::slice::from_raw_parts(usage.rgpszUsageIdentifier, usage.cUsageIdentifier as usize)
    };
    identifiers
        .iter()
        .map(|identifier| {
            if identifier.is_null() {
                return Err(unreadable());
            }
            // SAFETY: a NUL-terminated OID string inside the buffer.
            unsafe { identifier.to_string() }.map_err(|_| unreadable())
        })
        .collect()
}

/// The certificate that signed the file, as the publisher comparison reads
/// it: the publisher as the certificate authority verified it.
fn signer(path: &Path) -> Result<Publisher, UpdaterError> {
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
    let info = unsafe { &*buffer.as_ptr().cast::<CMSG_SIGNER_INFO>() };

    let wanted = CERT_INFO {
        Issuer: info.Issuer,
        SerialNumber: info.SerialNumber,
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

    let name = name_of(&certificate, CERT_NAME_SIMPLE_DISPLAY_TYPE, None);
    if name.is_empty() {
        return Err(untrusted("the signer's certificate has no name"));
    }
    Ok(Publisher {
        name,
        organization: name_of(
            &certificate,
            CERT_NAME_ATTR_TYPE,
            Some(szOID_ORGANIZATION_NAME),
        ),
        country: name_of(&certificate, CERT_NAME_ATTR_TYPE, Some(szOID_COUNTRY_NAME)),
        key_usages: key_usages(&certificate)?,
    })
}

/// Checks that `installer` is validly signed, by a certificate nobody
/// revoked, for the same publisher as `reference`, the running updater.
pub fn verify_same_publisher(
    installer: &Path,
    reference: &Path,
) -> Result<Revocation, UpdaterError> {
    let revocation = verify_authenticode(installer)?;
    same_publisher(&signer(reference)?, &signer(installer)?)?;
    Ok(revocation)
}
