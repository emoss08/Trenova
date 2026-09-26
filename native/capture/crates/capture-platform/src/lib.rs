//! What the Trenova Capture agent needs from Windows.
//!
//! - [`Dpapi`] encrypts spooled pages to the signed-in user.
//! - [`CredentialManager`] keeps the device credential, which Windows itself
//!   protects with DPAPI to that user.
//! - [`machine`] names the computer, the Windows user and the OS version for
//!   pairing and every token refresh.
//! - [`settings`] reads the server address an administrator or the
//!   installer configured, and the one a person typed.
//! - [`shell`] opens a web page, and [`SingleInstance`] keeps one agent per
//!   Windows session.
//! - [`paths`] names the per-user data directory and the machine-wide one
//!   the print service shares with each user's agent, and [`logging`] writes
//!   both processes' logs.
//!
//! - [`accounts`] turns Windows accounts into the SIDs that name and guard a
//!   person's print inbox, and [`acl`] creates the directories they guard.
//!
//! On anything but Windows this crate is empty.

#[cfg(windows)]
pub mod accounts;
#[cfg(windows)]
pub mod acl;
#[cfg(windows)]
mod credman;
#[cfg(windows)]
mod dpapi;
#[cfg(windows)]
pub mod logging;
#[cfg(windows)]
pub mod machine;
#[cfg(windows)]
pub mod paths;
#[cfg(windows)]
pub mod settings;
#[cfg(windows)]
pub mod shell;
#[cfg(windows)]
mod single_instance;
#[cfg(windows)]
mod wide;

#[cfg(windows)]
pub use credman::CredentialManager;
#[cfg(windows)]
pub use dpapi::Dpapi;
#[cfg(windows)]
pub use single_instance::SingleInstance;
