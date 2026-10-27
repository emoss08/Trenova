//! The base guides compiled into the module.
//!
//! One per transaction set Trenova reads and writes: 004010 204, 210, 214, 990 and 997, and
//! 005010 999 (X231A1). Each is assembled from its layout and the dictionaries for its
//! version, parsed once on first use and kept compiled for the life of the instance. A
//! partner's guide is derived from one of these and arrives with the request instead.

use std::sync::OnceLock;

use super::compiled::CompiledGuide;
use super::model::Guide;
use super::text::{Dictionary, parse_dictionary, parse_layout};

const ENVELOPE: &str = include_str!("data/envelope.dict");
const DICTIONARY_004010: &str = include_str!("data/004010.dict");
const DICTIONARY_005010: &str = include_str!("data/005010.dict");

const LAYOUTS: &[(&str, &str)] = &[
    ("004010", include_str!("data/004010-204.guide")),
    ("004010", include_str!("data/004010-210.guide")),
    ("004010", include_str!("data/004010-214.guide")),
    ("004010", include_str!("data/004010-990.guide")),
    ("004010", include_str!("data/004010-997.guide")),
    ("005010", include_str!("data/005010-999.guide")),
];

/// The envelope dictionary: ISA, IEA, TA1, GS and GE.
pub fn envelope() -> &'static Dictionary {
    static ENVELOPE_DICTIONARY: OnceLock<Dictionary> = OnceLock::new();
    ENVELOPE_DICTIONARY.get_or_init(|| parse_dictionary(ENVELOPE).unwrap_or_default())
}

fn dictionary(version: &str) -> Result<Dictionary, String> {
    let text = match version {
        "004010" => DICTIONARY_004010,
        "005010" => DICTIONARY_005010,
        other => return Err(format!("no dictionary for version {other}")),
    };
    let mut dictionary = parse_dictionary(text)?;
    dictionary.extend(envelope());
    Ok(dictionary)
}

/// Assembles every base guide, or the first error. Only tests and the guide listing need the
/// error; [`all`] keeps the guides that assembled.
pub fn assemble() -> Result<Vec<Guide>, String> {
    let mut dictionaries: Vec<(&str, Dictionary)> = Vec::new();
    let mut guides = Vec::with_capacity(LAYOUTS.len());
    for (version, layout) in LAYOUTS {
        if !dictionaries.iter().any(|(known, _)| known == version) {
            dictionaries.push((version, dictionary(version)?));
        }
        let dictionary = dictionaries
            .iter()
            .find(|(known, _)| known == version)
            .map(|(_, dictionary)| dictionary)
            .ok_or_else(|| format!("no dictionary for version {version}"))?;
        guides.push(parse_layout(layout)?.into_guide(dictionary)?);
    }
    Ok(guides)
}

/// Every base guide, compiled.
pub fn all() -> &'static [CompiledGuide] {
    static GUIDES: OnceLock<Vec<CompiledGuide>> = OnceLock::new();
    GUIDES.get_or_init(|| {
        assemble()
            .unwrap_or_default()
            .into_iter()
            .filter_map(|guide| CompiledGuide::compile(guide).ok())
            .collect()
    })
}

/// Whether any base dictionary or the envelope defines a segment with this ID, which tells a
/// segment that is not part of a set from one X12 does not have.
#[must_use]
pub fn is_known_segment(id: &str) -> bool {
    static KNOWN: OnceLock<std::collections::BTreeSet<String>> = OnceLock::new();
    KNOWN
        .get_or_init(|| {
            let mut known: std::collections::BTreeSet<String> =
                envelope().segments.keys().cloned().collect();
            for text in [DICTIONARY_004010, DICTIONARY_005010] {
                if let Ok(dictionary) = parse_dictionary(text) {
                    known.extend(dictionary.segments.into_keys());
                }
            }
            known
        })
        .contains(id)
}

/// The base guide with this identifier.
#[must_use]
pub fn by_id(id: &str) -> Option<&'static CompiledGuide> {
    all().iter().find(|guide| guide.guide().id == id)
}

/// The base guide for a transaction set, preferring the one for `version` (GS08) and falling
/// back to the set's only base guide. The flag says whether it fell back.
#[must_use]
pub fn for_set(transaction_set: &str, version: &str) -> Option<(&'static CompiledGuide, bool)> {
    let mut candidates = all()
        .iter()
        .filter(|guide| guide.guide().transaction_set == transaction_set);
    let first = candidates.clone().next()?;
    let exact = candidates.find(|guide| {
        let guide = guide.guide();
        guide.version == version || guide.base_version() == version.get(..6).unwrap_or(version)
    });
    Some(match exact {
        Some(guide) => (guide, false),
        None => (first, true),
    })
}

#[cfg(test)]
mod tests {
    use super::{all, assemble, by_id, for_set};
    use crate::x12::guide::check::check_guide;

    #[test]
    fn every_base_guide_assembles_compiles_and_passes_its_own_check() {
        let guides = assemble().expect("the base guides assemble");
        assert_eq!(guides.len(), 6);
        assert_eq!(all().len(), 6, "every base guide compiles");
        for guide in &guides {
            let problems = check_guide(guide, None);
            assert!(problems.is_empty(), "{}: {problems:?}", guide.id);
        }
    }

    #[test]
    fn finds_guides_by_id_and_by_set() {
        assert!(by_id("x12.004010.204").is_some());
        let (guide, fell_back) = for_set("214", "004010").expect("a 214 guide");
        assert_eq!(guide.guide().id, "x12.004010.214");
        assert!(!fell_back);
        let (guide, fell_back) = for_set("214", "004030").expect("a 214 guide");
        assert_eq!(guide.guide().id, "x12.004010.214");
        assert!(fell_back);
        let (guide, fell_back) = for_set("999", "005010X231A1").expect("a 999 guide");
        assert_eq!(guide.guide().id, "x12.005010.999");
        assert!(!fell_back);
        assert!(for_set("856", "004010").is_none());
    }
}
