//! Who and where this is, for pairing and every refresh.

use windows::Wdk::System::SystemServices::RtlGetVersion;
use windows::Win32::System::SystemInformation::{
    ComputerNameDnsHostname, GetComputerNameExW, OSVERSIONINFOW,
};
use windows::Win32::System::WindowsProgramming::GetUserNameW;
use windows_core::PWSTR;

use crate::wide::from_buffer;

/// The computer's host name.
pub fn machine_name() -> String {
    let mut size = 256u32;
    let mut buffer = vec![0u16; size as usize];
    // SAFETY: the buffer holds `size` UTF-16 units.
    let named = unsafe {
        GetComputerNameExW(
            ComputerNameDnsHostname,
            Some(PWSTR(buffer.as_mut_ptr())),
            &raw mut size,
        )
    };
    if named.is_err() {
        return std::env::var("COMPUTERNAME").unwrap_or_default();
    }
    from_buffer(&buffer)
}

/// The signed-in Windows user.
pub fn windows_user() -> String {
    let mut size = 257u32;
    let mut buffer = vec![0u16; size as usize];
    // SAFETY: the buffer holds `size` UTF-16 units.
    if unsafe { GetUserNameW(Some(PWSTR(buffer.as_mut_ptr())), &raw mut size) }.is_err() {
        return std::env::var("USERNAME").unwrap_or_default();
    }
    from_buffer(&buffer)
}

/// The real Windows version, e.g. `Windows 10.0.22631`. `RtlGetVersion` is
/// used because `GetVersionEx` answers what the manifest claims, not what
/// is running.
pub fn os_version() -> String {
    let mut info = OSVERSIONINFOW {
        dwOSVersionInfoSize: u32::try_from(std::mem::size_of::<OSVERSIONINFOW>()).unwrap_or(0),
        ..OSVERSIONINFOW::default()
    };
    // SAFETY: a correctly sized version structure.
    let status = unsafe { RtlGetVersion(&raw mut info) };
    if status.is_err() {
        return "Windows".to_owned();
    }
    format!(
        "Windows {}.{}.{}",
        info.dwMajorVersion, info.dwMinorVersion, info.dwBuildNumber
    )
}
