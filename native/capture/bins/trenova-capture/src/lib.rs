//! Trenova Capture, the per-user agent.
//!
//! [`agent`] is the part that talks to the server and the scanners; it knows
//! nothing of Windows, so it runs in tests against a mock server and a fake
//! scanner. [`state`] and [`menu`] are what the tray shows, decided without a
//! desktop. The Windows tray and start-up live in the binary.

pub mod agent;
pub mod icon;
pub mod menu;
pub mod scanners;
pub mod state;
