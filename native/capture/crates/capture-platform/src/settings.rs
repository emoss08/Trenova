//! The server address and other settings, from the registry.
//!
//! An address set by policy (`HKLM\SOFTWARE\Policies\Trenova\Capture`) wins,
//! so IT can point every computer at one server. Otherwise the one a person
//! typed (`HKCU\SOFTWARE\Trenova\Capture`) is used, and failing that the one
//! the installer wrote from its `TRENOVAURL` property
//! (`HKLM\SOFTWARE\Trenova\Capture`).

use windows::Win32::Foundation::ERROR_SUCCESS;
use windows::Win32::System::Registry::{
    HKEY, HKEY_CURRENT_USER, HKEY_LOCAL_MACHINE, REG_SZ, RRF_RT_REG_SZ, RegGetValueW,
    RegSetKeyValueW,
};
use windows_core::PCWSTR;

use crate::wide::{from_buffer, wide};

const POLICY_KEY: &str = "SOFTWARE\\Policies\\Trenova\\Capture";
const KEY: &str = "SOFTWARE\\Trenova\\Capture";
const SERVER_URL: &str = "ServerUrl";

fn read_string(root: HKEY, key: &str, value: &str) -> Option<String> {
    let key = wide(key);
    let value = wide(value);
    let mut size = 0u32;
    // SAFETY: asks for the size only.
    let status = unsafe {
        RegGetValueW(
            root,
            PCWSTR(key.as_ptr()),
            PCWSTR(value.as_ptr()),
            RRF_RT_REG_SZ,
            None,
            None,
            Some(&raw mut size),
        )
    };
    if status != ERROR_SUCCESS || size == 0 || size > 8192 {
        return None;
    }
    let mut buffer = vec![0u16; (size as usize).div_ceil(2)];
    // SAFETY: the buffer holds `size` bytes.
    let status = unsafe {
        RegGetValueW(
            root,
            PCWSTR(key.as_ptr()),
            PCWSTR(value.as_ptr()),
            RRF_RT_REG_SZ,
            None,
            Some(buffer.as_mut_ptr().cast()),
            Some(&raw mut size),
        )
    };
    (status == ERROR_SUCCESS)
        .then(|| from_buffer(&buffer).trim().to_owned())
        .filter(|s| !s.is_empty())
}

/// Where the server address came from.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ServerSource {
    Policy,
    User,
    Installer,
}

/// The configured server address, and where it came from.
pub fn server_url() -> Option<(String, ServerSource)> {
    read_string(HKEY_LOCAL_MACHINE, POLICY_KEY, SERVER_URL)
        .map(|url| (url, ServerSource::Policy))
        .or_else(|| {
            read_string(HKEY_CURRENT_USER, KEY, SERVER_URL).map(|url| (url, ServerSource::User))
        })
        .or_else(|| {
            read_string(HKEY_LOCAL_MACHINE, KEY, SERVER_URL)
                .map(|url| (url, ServerSource::Installer))
        })
}

/// Saves the address a person typed, for this Windows user.
pub fn set_server_url(url: &str) -> std::io::Result<()> {
    let key = wide(KEY);
    let value = wide(SERVER_URL);
    let data = wide(url);
    let bytes = u32::try_from(data.len() * 2).map_err(std::io::Error::other)?;
    // SAFETY: NUL-terminated names and a NUL-terminated REG_SZ of `bytes`.
    let status = unsafe {
        RegSetKeyValueW(
            HKEY_CURRENT_USER,
            PCWSTR(key.as_ptr()),
            PCWSTR(value.as_ptr()),
            REG_SZ.0,
            Some(data.as_ptr().cast()),
            bytes,
        )
    };
    if status == ERROR_SUCCESS {
        Ok(())
    } else {
        Err(std::io::Error::other(windows_core::Error::from(
            status.to_hresult(),
        )))
    }
}
