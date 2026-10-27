// Generated from the rules crate's unit table (services/crates/rules/src/tariff/units.rs) by
// services/crates/rules/tests/client_units.rs, which `task rust:wasm-rules` runs. Do not edit:
// change the crate and rebuild the module.

/** The base dimensions a unit measures; `Count` and `Percent` are dimensionless. */
export type TariffDimension =
  | "Money"
  | "Weight"
  | "Distance"
  | "Length"
  | "Volume"
  | "Duration"
  | "Temperature"
  | "Count"
  | "Percent";

/** A unit that is not built from others, and what it measures. */
export type TariffSimpleUnit = {
  symbol: string;
  dimension: TariffDimension;
};

/** Every simple unit the language knows, in the crate's order. */
export const TARIFF_UNITS: readonly TariffSimpleUnit[] = [
  { symbol: "$", dimension: "Money" },
  { symbol: "lb", dimension: "Weight" },
  { symbol: "cwt", dimension: "Weight" },
  { symbol: "kg", dimension: "Weight" },
  { symbol: "t", dimension: "Weight" },
  { symbol: "ton", dimension: "Weight" },
  { symbol: "mi", dimension: "Distance" },
  { symbol: "km", dimension: "Distance" },
  { symbol: "in", dimension: "Length" },
  { symbol: "ft", dimension: "Length" },
  { symbol: "m", dimension: "Length" },
  { symbol: "ft3", dimension: "Volume" },
  { symbol: "m3", dimension: "Volume" },
  { symbol: "gal", dimension: "Volume" },
  { symbol: "l", dimension: "Volume" },
  { symbol: "min", dimension: "Duration" },
  { symbol: "h", dimension: "Duration" },
  { symbol: "d", dimension: "Duration" },
  { symbol: "°F", dimension: "Temperature" },
  { symbol: "°C", dimension: "Temperature" },
  { symbol: "%", dimension: "Percent" },
  { symbol: "pallet", dimension: "Count" },
  { symbol: "piece", dimension: "Count" },
  { symbol: "stop", dimension: "Count" },
];
