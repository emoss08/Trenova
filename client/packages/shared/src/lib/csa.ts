/**
 * The FMCSA's seven Behavior Analysis and Safety Improvement Categories, in
 * the agency's own words and its own order. A safety director puts this page
 * beside a Safety Measurement System report, so the two have to read the same.
 */

export type CSATone = "critical" | "warning" | "muted";

/** The display order the FMCSA itself uses. */
export const CSA_BASIC_ORDER = [
  "UnsafeDriving",
  "HOSCompliance",
  "DriverFitness",
  "ControlledSubstances",
  "VehicleMaintenance",
  "HazmatCompliance",
  "CrashIndicator",
] as const;

export type CSABasic = (typeof CSA_BASIC_ORDER)[number];

export const CSA_BASIC_LABELS: Record<string, string> = {
  UnsafeDriving: "Unsafe Driving",
  HOSCompliance: "Hours of Service",
  DriverFitness: "Driver Fitness",
  ControlledSubstances: "Controlled Substances",
  VehicleMaintenance: "Vehicle Maintenance",
  HazmatCompliance: "Hazmat Compliance",
  CrashIndicator: "Crash Indicator",
};

export function csaBasicLabel(value: string): string {
  return CSA_BASIC_LABELS[value] ?? value;
}

export const CSA_BASIC_HINTS: Record<string, string> = {
  UnsafeDriving: "Speeding, reckless driving, improper lane change, inattention.",
  HOSCompliance: "Driving beyond the limits, and the records of duty status behind them.",
  DriverFitness: "Licensing, medical qualification, and the driver qualification file.",
  ControlledSubstances: "Use or possession while on duty, and the testing programme behind it.",
  VehicleMaintenance: "Brakes, lights, load securement — everything a roadside inspection checks.",
  HazmatCompliance: "Placarding, packaging and paperwork for regulated loads.",
  CrashIndicator: "Crash history and the pattern in it. Not published publicly by the FMCSA.",
};

export function csaBasicHint(value: string): string {
  return CSA_BASIC_HINTS[value] ?? "";
}

/**
 * How a weighted BASIC score reads against the fleet's own worst. Absolute
 * thresholds would be meaningless — the FMCSA percentile ranks a carrier
 * against its peers, and this data cannot — so the bar is relative and
 * deliberately not called a percentile.
 */
export function csaBasicTone(weightedScore: number, worstScore: number): CSATone {
  if (weightedScore <= 0 || worstScore <= 0) return "muted";
  const share = weightedScore / worstScore;
  if (share >= 0.66) return "critical";
  if (share >= 0.33) return "warning";
  return "muted";
}

/**
 * The share of the widest bar, for a relative-width chart. Clamped to 100 and
 * floored at a visible sliver so a BASIC carrying one violation is not drawn
 * as nothing at all.
 */
export function csaBarWidth(weightedScore: number, worstScore: number): number {
  if (weightedScore <= 0 || worstScore <= 0) return 0;
  return Math.max(3, Math.min(100, Math.round((weightedScore / worstScore) * 100)));
}

export const SAFETY_EVENT_KIND_LABELS: Record<string, string> = {
  Accident: "Accidents",
  Incident: "Incidents",
  NearMiss: "Near misses",
  Citation: "Citations",
  Inspection: "Inspections",
};

export function safetyEventKindLabel(value: string): string {
  return SAFETY_EVENT_KIND_LABELS[value] ?? value;
}

export const SAFETY_RATING_LABELS: Record<string, string> = {
  Excellent: "Excellent",
  Good: "Good",
  Watch: "Watch",
  AtRisk: "At risk",
};

export function safetyRatingLabel(value: string): string {
  return SAFETY_RATING_LABELS[value] ?? value;
}

export function safetyRatingTone(value: string): "active" | "inactive" | "warning" | "secondary" {
  switch (value) {
    case "Excellent":
      return "active";
    case "AtRisk":
      return "inactive";
    case "Watch":
      return "warning";
    default:
      return "secondary";
  }
}

/**
 * The direction a trend is moving, comparing the second half of the window to
 * the first. Two points are not a trend, so anything shorter reports "flat"
 * rather than inventing a direction.
 */
export function trendDirection(values: number[]): "up" | "down" | "flat" {
  if (values.length < 4) return "flat";
  const middle = Math.floor(values.length / 2);
  const first = values.slice(0, middle);
  const second = values.slice(middle);
  const mean = (rows: number[]) => rows.reduce((sum, row) => sum + row, 0) / rows.length;
  const before = mean(first);
  const after = mean(second);
  // A tenth of an event a month is noise, not a direction.
  if (Math.abs(after - before) < 0.1) return "flat";
  return after > before ? "up" : "down";
}

/**
 * The FMCSA severity weights run 1 to 10, and the published tables are what a
 * safety clerk keys from. Offering the range rather than a free number keeps a
 * weight nobody could look up out of the record.
 */
export const CSA_SEVERITY_WEIGHTS = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10] as const;

export const DIGEST_CADENCE_LABELS: Record<string, string> = {
  Immediate: "One notice per obligation",
  Daily: "Daily round-up",
  Weekly: "Weekly round-up",
};

export function digestCadenceLabel(value: string): string {
  return DIGEST_CADENCE_LABELS[value] ?? value;
}

export const WEEKDAY_LABELS = [
  "Sunday",
  "Monday",
  "Tuesday",
  "Wednesday",
  "Thursday",
  "Friday",
  "Saturday",
] as const;

export function weekdayLabel(value: number): string {
  return WEEKDAY_LABELS[value] ?? String(value);
}
