//! Lane keys and specificity: the `testdata/lane.json` corpus and the key scheme's invariant.

mod rules_support;

use proptest::prelude::*;
use trenova_rules::lane::rule::GEOGRAPHY_SCALE;
use trenova_rules::lane::rule::MAX_CONDITION_SCORE;
use trenova_rules::lane::scope::Place;
use trenova_rules::lane::scope::Scope;
use trenova_rules::lane::scope::ScopeType;

#[test]
fn corpus() {
    let ran = rules_support::run_corpus(include_str!("../testdata/lane.json"));
    assert!(ran > 20, "only {ran} cases ran");
}

#[test]
fn conditions_can_never_outweigh_one_step_of_geography() {
    assert_eq!(MAX_CONDITION_SCORE, 1023);
    assert_eq!(GEOGRAPHY_SCALE, 1024);
}

#[test]
fn scope_weights_are_strictly_ordered() {
    let ordered = [
        ScopeType::Any,
        ScopeType::Country,
        ScopeType::State,
        ScopeType::Zone,
        ScopeType::Radius,
        ScopeType::CityState,
        ScopeType::Zip3,
        ScopeType::Zip5,
        ScopeType::Location,
    ];
    for pair in ordered.windows(2) {
        assert!(pair[1].weight() > pair[0].weight(), "{pair:?}");
        assert_eq!(ScopeType::parse(Some(pair[1].name())), Some(pair[1]));
    }
}

fn reachable(place: &Place, state: &str, location: &str, zone: &str) {
    let keys = place.match_keys();
    let scopes = [
        Scope {
            scope_type: Some("Any"),
            value: "",
            city: "",
        },
        Scope {
            scope_type: Some("Country"),
            value: &place.country_iso3,
            city: "",
        },
        Scope {
            scope_type: Some("State"),
            value: state,
            city: "",
        },
        Scope {
            scope_type: Some("Zone"),
            value: zone,
            city: "",
        },
        Scope {
            scope_type: Some("CityState"),
            value: state,
            city: &place.city,
        },
        Scope {
            scope_type: Some("Zip3"),
            value: &place.postal_code,
            city: "",
        },
        Scope {
            scope_type: Some("Zip5"),
            value: &place.postal_code,
            city: "",
        },
        Scope {
            scope_type: Some("Location"),
            value: location,
            city: "",
        },
    ];
    for scope in scopes {
        let key = scope.key().expect("the scope is keyable");
        assert!(keys.contains(&key), "{key} is not among {keys:?}");
    }
    assert_eq!(keys.last().map(String::as_str), Some("ANY"));
}

#[test]
fn every_keyed_scope_is_reachable_from_its_place() {
    let place = Place {
        location_id: String::from("loc_1"),
        postal_code: String::from("60601"),
        city: String::from("Chicago"),
        state_id: String::from("ust_1"),
        country_iso3: String::from("USA"),
        zone_ids: vec![String::from("rzn_1")],
    };
    reachable(&place, "ust_1", "loc_1", "rzn_1");
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(256))]

    #[test]
    fn a_place_reaches_every_rule_written_for_it(
        postal in "[ ]{0,2}[0-9A-Za-z]{3,7}(-[0-9]{4})?[ ]{0,2}",
        city in "[ ]{0,2}[A-Za-zé]{1,8}( {1,3}[A-Za-z]{1,8})?[ ]{0,2}",
        country in "[A-Za-z]{2,3}",
    ) {
        let place = Place {
            location_id: String::from("loc_9"),
            postal_code: postal,
            city,
            state_id: String::from("ust_9"),
            country_iso3: country,
            zone_ids: vec![String::from("rzn_9"), String::from("rzn_10")],
        };
        reachable(&place, "ust_9", "loc_9", "rzn_10");
    }

    #[test]
    fn no_lane_request_panics(input in "\\PC{0,300}") {
        for op in ["lane.specificity", "lane.keys", "lane.parse", "lane.coverage"] {
            let request = format!("{{\"op\":\"{op}\",\"input\":{input}}}");
            prop_assert!(!trenova_rules::ops::dispatch(request.as_bytes()).is_empty());
        }
    }

    #[test]
    fn any_text_keys_without_panicking(value in "\\PC{0,12}", city in "\\PC{0,12}") {
        for scope_type in ["Zip3", "Zip5", "CityState", "Country", "Location"] {
            let _ = Scope { scope_type: Some(scope_type), value: &value, city: &city }.key();
        }
        let place = Place { postal_code: value.clone(), city: city.clone(), state_id: value, ..Place::default() };
        prop_assert!(!place.match_keys().is_empty());
    }
}
