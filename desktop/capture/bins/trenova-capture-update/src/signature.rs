//! What makes an installer's Authenticode signature acceptable, decided here
//! so it runs, and is tested, anywhere; reading the signature is Windows'
//! part.
//!
//! Releases are signed through Azure Trusted Signing, whose certificates last
//! days and whose intermediates change, so neither a thumbprint nor an issuer
//! can be pinned. What stays is the identity the certificate is issued to: its
//! subject's name, organization and country, and Trusted Signing's durable
//! identity, an extended key usage under `1.3.6.1.4.1.311.97.` unique to the
//! verified identity behind the signing profile. An installer must carry the
//! running updater's subject and, when the updater's own certificate carries
//! a durable identity, the same one.
//!
//! Revocation is checked for the whole chain. A certificate Windows reports
//! revoked is refused. When the revocation servers cannot be reached, the
//! signature is verified again without them and accepted on its own, so a
//! computer behind a firewall still updates; the signed manifest and the
//! installer's checksum have already been verified by then.

use crate::UpdaterError;

/// The prefix of Trusted Signing's extended key usages.
pub const TRUSTED_SIGNING_PREFIX: &str = "1.3.6.1.4.1.311.97.";
/// The usage every Trusted Signing public-trust certificate carries, the same
/// for every subscriber, so it identifies nobody.
pub const TRUSTED_SIGNING_PUBLIC_TRUST: &str = "1.3.6.1.4.1.311.97.1.0";

/// `CERT_E_REVOKED`
pub const CERT_E_REVOKED: u32 = 0x800B_010C;
/// `CRYPT_E_REVOKED`
pub const CRYPT_E_REVOKED: u32 = 0x8009_2010;
/// `CERT_E_REVOCATION_FAILURE`
pub const CERT_E_REVOCATION_FAILURE: u32 = 0x800B_010E;
/// `CRYPT_E_NO_REVOCATION_CHECK`
pub const CRYPT_E_NO_REVOCATION_CHECK: u32 = 0x8009_2012;
/// `CRYPT_E_REVOCATION_OFFLINE`
pub const CRYPT_E_REVOCATION_OFFLINE: u32 = 0x8009_2013;

/// The certificate that signed a file, as far as the publisher comparison
/// reads it.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct Publisher {
    /// The subject's simple display name (its common name).
    pub name: String,
    /// The subject's organization (`O`), empty when it has none.
    pub organization: String,
    /// The subject's country (`C`), empty when it has none.
    pub country: String,
    /// The certificate's extended key usages, as dotted OIDs.
    pub key_usages: Vec<String>,
}

impl Publisher {
    /// Trusted Signing durable identities the certificate carries.
    pub fn durable_identities(&self) -> impl Iterator<Item = &str> {
        self.key_usages.iter().map(String::as_str).filter(|usage| {
            usage.len() > TRUSTED_SIGNING_PREFIX.len()
                && usage.starts_with(TRUSTED_SIGNING_PREFIX)
                && *usage != TRUSTED_SIGNING_PUBLIC_TRUST
        })
    }
}

/// Whether `theirs`, an installer's signer, is the publisher `ours`, the
/// running updater's signer, is.
pub fn same_publisher(ours: &Publisher, theirs: &Publisher) -> Result<(), UpdaterError> {
    if ours.name.is_empty() {
        return Err(UpdaterError::Untrusted(
            "the running updater's signer has no name".into(),
        ));
    }
    if ours.name != theirs.name
        || ours.organization != theirs.organization
        || ours.country != theirs.country
    {
        return Err(UpdaterError::Untrusted(format!(
            "signed by {:?} ({:?}, {:?}), not by {:?} ({:?}, {:?})",
            theirs.name,
            theirs.organization,
            theirs.country,
            ours.name,
            ours.organization,
            ours.country
        )));
    }
    let mut ours_durable = ours.durable_identities().peekable();
    if ours_durable.peek().is_some()
        && !ours_durable.any(|identity| theirs.durable_identities().any(|t| t == identity))
    {
        return Err(UpdaterError::Untrusted(format!(
            "signed for another identity than {:?}",
            ours.name
        )));
    }
    Ok(())
}

/// What Windows said about a signature.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Verdict {
    Trusted,
    /// A certificate in the chain is revoked.
    Revoked,
    /// Revocation could not be checked: the servers were unreachable or
    /// gave no answer.
    RevocationUnknown,
    /// Anything else Windows refuses, with its status.
    Untrusted(u32),
}

/// Reads a `WinVerifyTrust` status.
pub fn verdict(status: u32) -> Verdict {
    match status {
        0 => Verdict::Trusted,
        CERT_E_REVOKED | CRYPT_E_REVOKED => Verdict::Revoked,
        CERT_E_REVOCATION_FAILURE | CRYPT_E_NO_REVOCATION_CHECK | CRYPT_E_REVOCATION_OFFLINE => {
            Verdict::RevocationUnknown
        }
        other => Verdict::Untrusted(other),
    }
}

/// How far revocation was checked for a signature that was accepted.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Revocation {
    /// No certificate in the chain is revoked.
    Checked,
    /// The revocation servers could not be reached; the signature was
    /// accepted without them.
    Unreachable,
}

/// Checks a signature with `verify`, which asks Windows and returns its
/// status: with revocation checked for the whole chain when given `true`,
/// without when given `false`. The second is asked only when the first could
/// not reach the revocation servers.
pub fn check_signature(verify: impl Fn(bool) -> u32) -> Result<Revocation, UpdaterError> {
    let refused = |status: u32| {
        UpdaterError::Untrusted(format!(
            "Windows did not accept its signature (0x{status:08x})"
        ))
    };
    match verdict(verify(true)) {
        Verdict::Trusted => Ok(Revocation::Checked),
        Verdict::Revoked => Err(UpdaterError::Untrusted(
            "its signing certificate is revoked".into(),
        )),
        Verdict::Untrusted(status) => Err(refused(status)),
        Verdict::RevocationUnknown => match verdict(verify(false)) {
            Verdict::Trusted => Ok(Revocation::Unreachable),
            Verdict::Revoked => Err(UpdaterError::Untrusted(
                "its signing certificate is revoked".into(),
            )),
            Verdict::RevocationUnknown => Err(refused(CERT_E_REVOCATION_FAILURE)),
            Verdict::Untrusted(status) => Err(refused(status)),
        },
    }
}

#[cfg(test)]
mod tests {
    use std::cell::RefCell;

    use super::*;

    const IDENTITY: &str = "1.3.6.1.4.1.311.97.990309390.766961637.194916062.941502583";
    const OTHER_IDENTITY: &str = "1.3.6.1.4.1.311.97.123456789.987654321.111111111.222222222";
    const CODE_SIGNING: &str = "1.3.6.1.5.5.7.3.3";

    fn trenova(usages: &[&str]) -> Publisher {
        Publisher {
            name: "Trenova Inc".into(),
            organization: "Trenova Inc".into(),
            country: "US".into(),
            key_usages: usages.iter().map(|u| (*u).to_owned()).collect(),
        }
    }

    fn trusted_signing() -> Publisher {
        trenova(&[CODE_SIGNING, TRUSTED_SIGNING_PUBLIC_TRUST, IDENTITY])
    }

    fn is_untrusted(result: &Result<(), UpdaterError>) -> bool {
        matches!(result, Err(UpdaterError::Untrusted(_)))
    }

    #[test]
    fn a_renewed_trusted_signing_certificate_is_the_same_publisher() {
        let ours = trusted_signing();
        let renewed = Publisher {
            key_usages: vec![
                IDENTITY.into(),
                CODE_SIGNING.into(),
                TRUSTED_SIGNING_PUBLIC_TRUST.into(),
            ],
            ..trusted_signing()
        };
        assert!(same_publisher(&ours, &renewed).is_ok());
        assert!(same_publisher(&ours, &trusted_signing()).is_ok());
    }

    #[test]
    fn a_certificate_without_a_durable_identity_compares_its_subject() {
        let ours = trenova(&[CODE_SIGNING]);
        assert!(same_publisher(&ours, &trenova(&[CODE_SIGNING])).is_ok());
        assert!(
            same_publisher(&ours, &trusted_signing()).is_ok(),
            "moving onto Trusted Signing under the same subject"
        );
    }

    #[test]
    fn the_same_name_for_another_identity_is_refused() {
        let ours = trusted_signing();
        let another = trenova(&[CODE_SIGNING, TRUSTED_SIGNING_PUBLIC_TRUST, OTHER_IDENTITY]);
        assert!(is_untrusted(&same_publisher(&ours, &another)));

        let marker_only = trenova(&[CODE_SIGNING, TRUSTED_SIGNING_PUBLIC_TRUST]);
        assert!(
            is_untrusted(&same_publisher(&ours, &marker_only)),
            "the public-trust marker is everyone's"
        );

        let plain = trenova(&[CODE_SIGNING]);
        assert!(is_untrusted(&same_publisher(&ours, &plain)));
    }

    #[test]
    fn another_subject_is_refused() {
        let ours = trusted_signing();
        for theirs in [
            Publisher {
                name: "Trenova".into(),
                ..trusted_signing()
            },
            Publisher {
                organization: "Trenova Holdings".into(),
                ..trusted_signing()
            },
            Publisher {
                country: "GB".into(),
                ..trusted_signing()
            },
        ] {
            assert!(is_untrusted(&same_publisher(&ours, &theirs)), "{theirs:?}");
        }
        let nameless = Publisher {
            name: String::new(),
            ..trusted_signing()
        };
        assert!(is_untrusted(&same_publisher(&nameless, &nameless)));
    }

    #[test]
    fn only_trusted_signing_identities_are_durable() {
        let publisher = trenova(&[
            CODE_SIGNING,
            TRUSTED_SIGNING_PUBLIC_TRUST,
            TRUSTED_SIGNING_PREFIX,
            IDENTITY,
        ]);
        assert_eq!(
            publisher.durable_identities().collect::<Vec<_>>(),
            [IDENTITY]
        );
    }

    #[test]
    fn a_signature_with_revocation_checked_is_accepted_once() {
        let asked = RefCell::new(Vec::new());
        let result = check_signature(|revocation| {
            asked.borrow_mut().push(revocation);
            0
        });
        assert!(matches!(result, Ok(Revocation::Checked)));
        assert_eq!(*asked.borrow(), [true]);
    }

    #[test]
    fn a_revoked_certificate_is_refused_without_asking_again() {
        for status in [CERT_E_REVOKED, CRYPT_E_REVOKED] {
            let asked = RefCell::new(Vec::new());
            let result = check_signature(|revocation| {
                asked.borrow_mut().push(revocation);
                status
            });
            assert!(matches!(result, Err(UpdaterError::Untrusted(_))));
            assert_eq!(*asked.borrow(), [true]);
        }
    }

    #[test]
    fn unreachable_revocation_servers_fall_back_to_the_signature_alone() {
        for status in [
            CERT_E_REVOCATION_FAILURE,
            CRYPT_E_NO_REVOCATION_CHECK,
            CRYPT_E_REVOCATION_OFFLINE,
        ] {
            let asked = RefCell::new(Vec::new());
            let result = check_signature(|revocation| {
                asked.borrow_mut().push(revocation);
                if revocation { status } else { 0 }
            });
            assert!(matches!(result, Ok(Revocation::Unreachable)), "{status:x}");
            assert_eq!(*asked.borrow(), [true, false]);
        }
    }

    #[test]
    fn a_signature_that_fails_without_revocation_too_is_refused() {
        let bad_digest = 0x8009_6010;
        let result = check_signature(|revocation| {
            if revocation {
                CERT_E_REVOCATION_FAILURE
            } else {
                bad_digest
            }
        });
        assert!(matches!(result, Err(UpdaterError::Untrusted(_))));
    }

    #[test]
    fn an_unsigned_or_untrusted_file_is_refused() {
        let no_signature = 0x800B_0100;
        let untrusted_root = 0x800B_0109;
        for status in [no_signature, untrusted_root] {
            let result = check_signature(|_| status);
            assert!(matches!(result, Err(UpdaterError::Untrusted(_))));
        }
    }
}
