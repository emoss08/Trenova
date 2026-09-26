//! What the tray says and offers, decided from the state alone.
//!
//! The Windows tray only draws what this builds, so what a person sees in
//! every state is tested here without a desktop.

use std::path::PathBuf;

use capture_protocol::api::{ProfileStatus, SourceProtocol};

use crate::state::{Command, Connection, Snapshot};

/// What choosing a menu item does.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum MenuAction {
    Command(Command),
    /// Open a Trenova page in the browser.
    Open(String),
    /// Open a folder on this computer.
    OpenFolder(PathBuf),
    /// Ask for the server address.
    SetServer,
    Quit,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum MenuEntry {
    Item {
        label: String,
        action: Option<MenuAction>,
    },
    Submenu {
        label: String,
        entries: Vec<MenuEntry>,
    },
    Separator,
}

fn item(label: impl Into<String>, action: MenuAction) -> MenuEntry {
    MenuEntry::Item {
        label: label.into(),
        action: Some(action),
    }
}

/// A line that says something and does nothing.
fn note(label: impl Into<String>) -> MenuEntry {
    MenuEntry::Item {
        label: label.into(),
        action: None,
    }
}

fn pages(count: u32) -> String {
    if count == 1 {
        "1 page".to_owned()
    } else {
        format!("{count} pages")
    }
}

impl Snapshot {
    /// One line on where things stand, for the tooltip and the menu.
    pub fn status_line(&self) -> String {
        let waiting = (self.pages_waiting > 0)
            .then(|| format!("{} waiting to upload", pages(self.pages_waiting)));
        match &self.connection {
            Connection::NeedsServer => "Set the server address to begin".to_owned(),
            Connection::SignedOut => "Not signed in".to_owned(),
            Connection::Pairing { code, .. } => format!("Waiting for approval, code {code}"),
            Connection::Connecting => "Connecting".to_owned(),
            Connection::Online => self
                .scanning
                .clone()
                .or(waiting)
                .unwrap_or_else(|| "Ready".to_owned()),
            Connection::Offline { .. } => match waiting {
                Some(waiting) => format!("Offline, {waiting}"),
                None => "Offline, reconnecting".to_owned(),
            },
            Connection::Blocked { reason } => reason.clone(),
        }
    }

    /// The tray's tooltip; Windows shows at most 127 characters.
    pub fn tooltip(&self) -> String {
        let text = format!("Trenova Capture: {}", self.status_line());
        match text.char_indices().nth(127) {
            Some((end, _)) => text[..end].to_owned(),
            None => text,
        }
    }

    fn scan_menu(&self) -> MenuEntry {
        if self.sources.is_empty() {
            return note("No scanners found");
        }
        let profiles: Vec<_> = self
            .profiles
            .iter()
            .filter(|p| p.status != ProfileStatus::Inactive)
            .collect();
        let entries = self
            .sources
            .iter()
            .map(|source| {
                let name = match source.protocol {
                    SourceProtocol::Wia => format!("{} (WIA)", source.name),
                    _ => source.name.clone(),
                };
                let scan = |profile: Option<&capture_protocol::api::CaptureProfile>| {
                    MenuAction::Command(Command::Scan {
                        source: source.name.clone(),
                        protocol: source.protocol,
                        profile: profile.map(|p| p.id.clone()),
                    })
                };
                if profiles.is_empty() {
                    return item(name, scan(None));
                }
                let mut ordered = profiles.clone();
                ordered.sort_by_key(|p| !p.is_default);
                MenuEntry::Submenu {
                    label: name,
                    entries: ordered
                        .into_iter()
                        .map(|profile| {
                            let label = if profile.is_default {
                                format!("{} (default)", profile.name)
                            } else {
                                profile.name.clone()
                            };
                            item(label, scan(Some(profile)))
                        })
                        .collect(),
                }
            })
            .collect();
        MenuEntry::Submenu {
            label: "Scan to intake".to_owned(),
            entries,
        }
    }

    /// The tray menu, top to bottom.
    pub fn menu(&self) -> Vec<MenuEntry> {
        let mut menu = Vec::new();
        let who = match (&self.person, &self.organization) {
            (Some(person), Some(org)) => format!("{person}, {org}"),
            (Some(person), None) => person.clone(),
            _ if self.signed_in() => "Signed in".to_owned(),
            _ => "Not signed in".to_owned(),
        };
        menu.push(note(who));
        menu.push(note(self.status_line()));
        if let Some(version) = &self.update_required {
            menu.push(note(format!(
                "Update to Trenova Capture {version} or later"
            )));
        }
        menu.push(MenuEntry::Separator);

        let online = self.signed_in() && !matches!(self.connection, Connection::Blocked { .. });
        if online && self.scanning.is_none() {
            menu.push(self.scan_menu());
        }
        for paused in &self.paused {
            menu.push(item(
                format!(
                    "Continue scanning from {} ({} so far)",
                    paused.label,
                    pages(paused.pages)
                ),
                MenuAction::Command(Command::Continue(paused.key.clone())),
            ));
            menu.push(item(
                format!(
                    "Finish and send the {} from {}",
                    pages(paused.pages),
                    paused.label
                ),
                MenuAction::Command(Command::Finish(paused.key.clone())),
            ));
        }

        if !self.recent.is_empty() {
            menu.push(MenuEntry::Separator);
            menu.push(note("Recently sent"));
            for batch in &self.recent {
                let label = format!("{}: {}", batch.label, pages(batch.pages));
                menu.push(if batch.link.is_empty() {
                    note(label)
                } else {
                    item(label, MenuAction::Open(batch.link.clone()))
                });
            }
        }

        menu.push(MenuEntry::Separator);
        if let Some(intake) = self.intake_link(None).filter(|_| self.signed_in()) {
            menu.push(item("Open Intake in Trenova", MenuAction::Open(intake)));
        }
        if self.signed_in() {
            menu.push(item(
                "Look for scanners again",
                MenuAction::Command(Command::RefreshScanners),
            ));
        }
        if self.failed > 0
            && let Some(dir) = &self.failed_dir
        {
            menu.push(item(
                format!(
                    "Open the folder of uploads Trenova refused ({})",
                    self.failed
                ),
                MenuAction::OpenFolder(dir.clone()),
            ));
        }

        menu.push(MenuEntry::Separator);
        match &self.connection {
            Connection::NeedsServer => {}
            Connection::SignedOut => {
                menu.push(item("Sign in", MenuAction::Command(Command::SignIn)));
            }
            Connection::Pairing { url, .. } => {
                menu.push(item(
                    "Open the approval page",
                    MenuAction::Open(url.clone()),
                ));
                menu.push(item(
                    "Cancel sign-in",
                    MenuAction::Command(Command::CancelSignIn),
                ));
            }
            _ => menu.push(item("Sign out", MenuAction::Command(Command::SignOut))),
        }
        menu.push(item("Set server address", MenuAction::SetServer));
        menu.push(item("Quit Trenova Capture", MenuAction::Quit));
        menu
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::agent::plan::default_profile;
    use crate::state::{PausedBatch, RecentBatch};
    use capture_protocol::api::{Id, SourceInfo};
    use capture_protocol::helper::ScanCondition;

    fn actions(menu: &[MenuEntry]) -> Vec<MenuAction> {
        menu.iter()
            .flat_map(|entry| match entry {
                MenuEntry::Item {
                    action: Some(action),
                    ..
                } => vec![action.clone()],
                MenuEntry::Submenu { entries, .. } => actions(entries),
                _ => Vec::new(),
            })
            .collect()
    }

    fn source(name: &str, protocol: SourceProtocol) -> SourceInfo {
        SourceInfo {
            name: name.into(),
            protocol,
            bitness: 64,
            is_default: false,
            duplex: true,
            feeder: true,
            patch_codes: false,
            barcodes: false,
            blank_discard: false,
            resolutions: Vec::new(),
            pixel_types: Vec::new(),
        }
    }

    #[test]
    fn a_new_install_asks_for_the_server_and_offers_nothing_else() {
        let snapshot = Snapshot::default();
        assert_eq!(snapshot.status_line(), "Set the server address to begin");
        assert_eq!(
            actions(&snapshot.menu()),
            vec![MenuAction::SetServer, MenuAction::Quit]
        );
    }

    #[test]
    fn a_signed_in_computer_offers_each_scanner_with_its_profiles_default_first() {
        let mut other = default_profile();
        other.id = Id::from("cprf_color");
        other.name = "Colour".into();
        other.is_default = false;
        let mut default = default_profile();
        default.id = Id::from("cprf_default");
        let snapshot = Snapshot {
            connection: Connection::Online,
            person: Some("Jordan Doe".into()),
            organization: Some("Acme Freight".into()),
            web_base: Some("https://app.acme.com".into()),
            sources: vec![
                source("fi-8170", SourceProtocol::Twain),
                source("DS-530", SourceProtocol::Wia),
            ],
            profiles: vec![other, default],
            ..Snapshot::default()
        };
        let menu = snapshot.menu();
        assert_eq!(menu[0], note("Jordan Doe, Acme Freight"));
        let MenuEntry::Submenu { label, entries } = &menu[3] else {
            panic!("expected the scan menu, got {:?}", menu[3]);
        };
        assert_eq!(label, "Scan to intake");
        let MenuEntry::Submenu {
            label,
            entries: profiles,
        } = &entries[1]
        else {
            panic!("expected a scanner submenu");
        };
        assert_eq!(label, "DS-530 (WIA)");
        assert_eq!(
            profiles[0],
            item(
                "Paperwork (default)",
                MenuAction::Command(Command::Scan {
                    source: "DS-530".into(),
                    protocol: SourceProtocol::Wia,
                    profile: Some(Id::from("cprf_default")),
                })
            )
        );
        assert!(actions(&menu).contains(&MenuAction::Open("https://app.acme.com/intake".into())));
        assert!(actions(&menu).contains(&MenuAction::Command(Command::SignOut)));
    }

    #[test]
    fn a_stopped_scan_offers_to_continue_or_finish_and_hides_new_scans_while_scanning() {
        let snapshot = Snapshot {
            connection: Connection::Online,
            scanning: Some("Scanning from fi-8170: 3 pages".into()),
            paused: vec![PausedBatch {
                key: "cap-1".into(),
                label: "fi-8170".into(),
                pages: 39,
                condition: ScanCondition::PaperJam,
            }],
            ..Snapshot::default()
        };
        assert_eq!(snapshot.status_line(), "Scanning from fi-8170: 3 pages");
        let menu = snapshot.menu();
        assert!(
            !menu.iter().any(
                |e| matches!(e, MenuEntry::Submenu { label, .. } if label == "Scan to intake")
            )
        );
        let all = actions(&menu);
        assert!(all.contains(&MenuAction::Command(Command::Continue("cap-1".into()))));
        assert!(all.contains(&MenuAction::Command(Command::Finish("cap-1".into()))));
    }

    #[test]
    fn waiting_pages_refused_uploads_and_recent_batches_are_shown() {
        let snapshot = Snapshot {
            connection: Connection::Offline {
                reason: "dns".into(),
            },
            pages_waiting: 12,
            failed: 1,
            failed_dir: Some(PathBuf::from("C:\\spool\\failed")),
            recent: [RecentBatch {
                label: "fi-8170".into(),
                pages: 1,
                link: "https://app.acme.com/intake?batch=cbat_1".into(),
                at: std::time::SystemTime::UNIX_EPOCH,
                requested: false,
            }]
            .into(),
            ..Snapshot::default()
        };
        assert_eq!(
            snapshot.status_line(),
            "Offline, 12 pages waiting to upload"
        );
        let all = actions(&snapshot.menu());
        assert!(all.contains(&MenuAction::Open(
            "https://app.acme.com/intake?batch=cbat_1".into()
        )));
        assert!(all.contains(&MenuAction::OpenFolder(PathBuf::from("C:\\spool\\failed"))));
    }

    #[test]
    fn pairing_shows_the_code_and_a_blocked_agent_says_why() {
        let pairing = Snapshot {
            connection: Connection::Pairing {
                code: "BCDF-GHJK".into(),
                url: "https://app.acme.com/capture/pair?code=BCDF-GHJK".into(),
            },
            ..Snapshot::default()
        };
        assert_eq!(
            pairing.status_line(),
            "Waiting for approval, code BCDF-GHJK"
        );
        assert!(actions(&pairing.menu()).contains(&MenuAction::Command(Command::CancelSignIn)));

        let blocked = Snapshot {
            connection: Connection::Blocked {
                reason: "Trenova Capture 2.0.0 or later is required.".into(),
            },
            update_required: Some("2.0.0".into()),
            sources: vec![source("fi-8170", SourceProtocol::Twain)],
            ..Snapshot::default()
        };
        assert_eq!(
            blocked.tooltip(),
            "Trenova Capture: Trenova Capture 2.0.0 or later is required."
        );
        assert!(
            !actions(&blocked.menu())
                .iter()
                .any(|a| matches!(a, MenuAction::Command(Command::Scan { .. })))
        );
    }

    #[test]
    fn a_long_tooltip_is_cut_to_what_windows_shows() {
        let snapshot = Snapshot {
            connection: Connection::Blocked {
                reason: "x".repeat(300),
            },
            ..Snapshot::default()
        };
        assert_eq!(snapshot.tooltip().chars().count(), 127);
    }
}
