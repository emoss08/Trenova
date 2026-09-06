/**
 * Labels and tones for injury and illness recordkeeping. The OSHA words are
 * used verbatim wherever the form uses them, because somebody reading the log
 * beside a paper 300 should see the same phrases.
 */

export type InjuryTone = "active" | "inactive" | "warning" | "secondary" | "info";

export const CASE_CLASSIFICATION_LABELS: Record<string, string> = {
  NotRecordable: "Not recordable",
  FirstAidOnly: "First aid only",
  OtherRecordable: "Other recordable case",
  JobTransferOrRestriction: "Job transfer or restriction",
  DaysAway: "Days away from work",
  Death: "Death",
};

export function caseClassificationLabel(value: string): string {
  return CASE_CLASSIFICATION_LABELS[value] ?? value;
}

/**
 * Whether a case belongs on the 300 log. First aid alone is explicitly not
 * recordable (29 CFR 1904.7(b)(5)(ii)), and that distinction is what the whole
 * log turns on.
 */
export function classificationIsRecordable(value: string): boolean {
  return (
    value === "OtherRecordable" ||
    value === "JobTransferOrRestriction" ||
    value === "DaysAway" ||
    value === "Death"
  );
}

export function classificationTone(value: string): InjuryTone {
  switch (value) {
    case "Death":
    case "DaysAway":
      return "inactive";
    case "JobTransferOrRestriction":
      return "warning";
    case "OtherRecordable":
      return "info";
    default:
      return "secondary";
  }
}

export const ILLNESS_TYPE_LABELS: Record<string, string> = {
  Injury: "Injury",
  SkinDisorder: "Skin disorder",
  RespiratoryCondition: "Respiratory condition",
  Poisoning: "Poisoning",
  HearingLoss: "Hearing loss",
  OtherIllness: "Other illness",
};

export function illnessTypeLabel(value: string): string {
  return ILLNESS_TYPE_LABELS[value] ?? value;
}

export const INJURY_TREATMENT_LABELS: Record<string, string> = {
  None: "None",
  FirstAid: "First aid",
  MedicalTreatment: "Medical treatment",
  EmergencyRoom: "Emergency room",
  Hospitalized: "Hospitalised",
};

export function injuryTreatmentLabel(value: string): string {
  return INJURY_TREATMENT_LABELS[value] ?? value;
}

export const CLAIM_STATUS_LABELS: Record<string, string> = {
  NotFiled: "Not filed",
  Filed: "Filed",
  Accepted: "Accepted",
  Denied: "Denied",
  Closed: "Closed",
};

export function claimStatusLabel(value: string): string {
  return CLAIM_STATUS_LABELS[value] ?? value;
}

export function claimStatusTone(value: string): InjuryTone {
  switch (value) {
    case "Accepted":
      return "active";
    case "Denied":
      return "inactive";
    case "Filed":
      return "warning";
    default:
      return "secondary";
  }
}

/**
 * A case stops counting at 180 days away or restricted
 * (29 CFR 1904.7(b)(3)(viii)).
 */
export const MAX_COUNTED_DAYS = 180;

/**
 * The classification the recorded facts imply. It mirrors the server's
 * suggestion so the form can show it live; the server's answer is the one that
 * is stored.
 */
export function suggestClassification(
  treatment: string,
  daysAway: number,
  daysRestricted: number,
): string {
  if (daysAway > 0) return "DaysAway";
  if (daysRestricted > 0) return "JobTransferOrRestriction";
  if (
    treatment === "MedicalTreatment" ||
    treatment === "EmergencyRoom" ||
    treatment === "Hospitalized"
  ) {
    return "OtherRecordable";
  }
  if (treatment === "FirstAid") return "FirstAidOnly";
  return "NotRecordable";
}

/**
 * Recordable cases per 100 full-time workers a year — 200,000 hours is what
 * "100 full-time workers" means. Null over zero hours, because a rate with no
 * denominator is not a small number; it is not a number.
 */
export function incidentRate(cases: number, totalHours: number): number | null {
  if (totalHours <= 0) return null;
  return (cases * 200_000) / totalHours;
}

export function formatRate(rate: number | null | undefined): string {
  if (rate === null || rate === undefined) return "—";
  return rate.toFixed(2);
}
