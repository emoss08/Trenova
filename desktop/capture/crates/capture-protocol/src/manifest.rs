//! The digest a batch is sealed with.
//!
//! `captureservice.ManifestDigest`: SHA-256 over each page's own SHA-256 hex
//! digest followed by a newline, in sequence order. The server recomputes it
//! from the pages it stored, so a seal proves it holds exactly the pages this
//! device captured, in the order it captured them.

use sha2::{Digest, Sha256};

/// The lowercase hex SHA-256 of one page, as the server stores it in
/// `checksum_sha256`.
pub fn page_checksum(bytes: &[u8]) -> String {
    hex::encode(Sha256::digest(bytes))
}

/// Accumulates page checksums in sequence order.
#[derive(Debug, Clone, Default)]
pub struct ManifestBuilder {
    hasher: Sha256,
    pages: u32,
}

impl ManifestBuilder {
    pub fn new() -> Self {
        Self::default()
    }

    /// Adds the next page's checksum.
    pub fn push(&mut self, checksum_hex: &str) {
        self.hasher.update(checksum_hex.as_bytes());
        self.hasher.update(b"\n");
        self.pages += 1;
    }

    pub fn page_count(&self) -> u32 {
        self.pages
    }

    /// The digest as the seal sends it.
    pub fn finish(self) -> String {
        hex::encode(self.hasher.finalize())
    }
}

/// The digest of a whole ordered list of checksums.
pub fn manifest_digest<'a>(checksums: impl IntoIterator<Item = &'a str>) -> String {
    let mut builder = ManifestBuilder::new();
    for checksum in checksums {
        builder.push(checksum);
    }
    builder.finish()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn page_checksums_are_lowercase_hex_sha256() {
        assert_eq!(
            page_checksum(b"page one"),
            "08e548c038b1608847f6285d147959da2c6632aca2cda9fd1166ec8f32b460e7"
        );
    }

    #[test]
    fn the_digest_matches_the_server_algorithm() {
        let first = page_checksum(b"page one");
        let second = page_checksum(b"page two");
        assert_eq!(
            manifest_digest([first.as_str(), second.as_str()]),
            "5d124939fa1f7c8dc5c1a2d98f56d702967061d5f8c3cba3d57b4cb80dbd16e4"
        );
    }

    #[test]
    fn order_matters() {
        let first = page_checksum(b"page one");
        let second = page_checksum(b"page two");
        assert_ne!(
            manifest_digest([first.as_str(), second.as_str()]),
            manifest_digest([second.as_str(), first.as_str()])
        );
    }

    #[test]
    fn an_empty_manifest_is_the_digest_of_nothing() {
        let builder = ManifestBuilder::new();
        assert_eq!(builder.page_count(), 0);
        assert_eq!(
            builder.finish(),
            "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
        );
    }
}
