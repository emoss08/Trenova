//! The browser's unit table, generated from the crate's (`SIMPLE_UNITS`).
//!
//! The web app formats and edits wire values before the module has loaded (a bill, an input's
//! form field), so it needs the unit symbols and their dimensions synchronously. Rather than a
//! hand copy that drifts, `client/packages/shared/src/rules/tariff-units.generated.ts` is written
//! from the crate: `build-wasm.sh install` runs this test with `TRENOVA_WRITE_GENERATED=1`, which
//! rewrites the file, and every other run (`cargo test`, `build-wasm.sh check`) fails when the
//! committed file differs from what the crate would write.

use std::fmt::Write as _;
use std::path::Path;
use trenova_rules::tariff::units::SIMPLE_UNITS;

const GENERATED: &str = "../../../client/packages/shared/src/rules/tariff-units.generated.ts";

/// Every dimension a simple unit measures, in the order the type lists them.
const DIMENSIONS: [&str; 9] = [
    "Money",
    "Weight",
    "Distance",
    "Length",
    "Volume",
    "Duration",
    "Temperature",
    "Count",
    "Percent",
];

fn render() -> String {
    let mut out = String::from(
        "// Generated from the rules crate's unit table (services/crates/rules/src/tariff/units.rs) by\n\
         // services/crates/rules/tests/client_units.rs, which `task rust:wasm-rules` runs. Do not edit:\n\
         // change the crate and rebuild the module.\n\n\
         /** The base dimensions a unit measures; `Count` and `Percent` are dimensionless. */\n\
         export type TariffDimension =\n",
    );
    for (index, dimension) in DIMENSIONS.iter().enumerate() {
        let end = if index + 1 == DIMENSIONS.len() {
            ";"
        } else {
            ""
        };
        let _ = writeln!(out, "  | \"{dimension}\"{end}");
    }
    out.push_str(
        "\n/** A unit that is not built from others, and what it measures. */\n\
         export type TariffSimpleUnit = {\n  symbol: string;\n  dimension: TariffDimension;\n};\n\n\
         /** Every simple unit the language knows, in the crate's order. */\n\
         export const TARIFF_UNITS: readonly TariffSimpleUnit[] = [\n",
    );
    for unit in SIMPLE_UNITS {
        let _ = writeln!(
            out,
            "  {{ symbol: \"{}\", dimension: \"{}\" }},",
            unit.symbol,
            unit.dimension()
        );
    }
    out.push_str("];\n");
    out
}

#[test]
fn the_browser_unit_table_is_the_crates() {
    for unit in SIMPLE_UNITS {
        assert!(
            DIMENSIONS.contains(&unit.dimension()),
            "{} measures {}, which the generated type does not list",
            unit.symbol,
            unit.dimension()
        );
    }
    let path = Path::new(env!("CARGO_MANIFEST_DIR")).join(GENERATED);
    let rendered = render();
    if std::env::var_os("TRENOVA_WRITE_GENERATED").is_some() {
        std::fs::write(&path, &rendered).expect("write the browser's unit table");
        return;
    }
    let committed = std::fs::read_to_string(&path).unwrap_or_default();
    assert!(
        committed == rendered,
        "{} is not what the crate's unit table gives; run `task rust:wasm-rules` and commit it",
        path.display()
    );
}
