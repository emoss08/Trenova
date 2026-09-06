/**
 * Labels and tones for the drug and alcohol testing programme. The server owns
 * the enums; this is the one place the client turns them into words, so a
 * status reads the same on the roster, the panel and the home screen.
 */

export type DrugAlcoholStatusValue = "Unknown" | "Clear" | "Pending" | "Prohibited";

export type ReturnToDutyStatusValue =
  | "NotRequired"
  | "SAPEvaluation"
  | "RTDTestRequired"
  | "FollowUpTesting"
  | "Complete";

export type BadgeTone = "active" | "inactive" | "info" | "warning" | "secondary";

type StatusMeta = {
  label: string;
  /** One line saying what the status means for dispatch today. */
  detail: string;
  tone: BadgeTone;
};

export const DRUG_ALCOHOL_STATUS_META: Record<DrugAlcoholStatusValue, StatusMeta> = {
  Clear: {
    label: "Clear",
    detail: "Nothing on the testing record stands in the way of dispatch.",
    tone: "active",
  },
  Pending: {
    label: "Awaiting result",
    detail: "A collection has been taken and the result has not come back.",
    tone: "warning",
  },
  Prohibited: {
    label: "Prohibited",
    detail:
      "The driver must not perform safety-sensitive functions until the " +
      "return-to-duty process is finished (49 CFR 382.501).",
    tone: "inactive",
  },
  Unknown: {
    label: "Not on file",
    detail:
      "No test is on file. Nothing has been found against the driver, and nothing clears them either.",
    tone: "secondary",
  },
};

export function drugAlcoholStatusMeta(status: string): StatusMeta {
  return (
    DRUG_ALCOHOL_STATUS_META[status as DrugAlcoholStatusValue] ?? DRUG_ALCOHOL_STATUS_META.Unknown
  );
}

export const RETURN_TO_DUTY_LABELS: Record<ReturnToDutyStatusValue, string> = {
  NotRequired: "Not required",
  SAPEvaluation: "SAP evaluation",
  RTDTestRequired: "Return-to-duty test required",
  FollowUpTesting: "Follow-up testing",
  Complete: "Complete",
};

export function returnToDutyLabel(status: string): string {
  return RETURN_TO_DUTY_LABELS[status as ReturnToDutyStatusValue] ?? status;
}

export const DOT_TEST_TYPE_LABELS: Record<string, string> = {
  PreEmployment: "Pre-employment",
  Random: "Random",
  PostAccident: "Post-accident",
  ReasonableSuspicion: "Reasonable suspicion",
  ReturnToDuty: "Return to duty",
  FollowUp: "Follow-up",
  Other: "Other",
};

export function dotTestTypeLabel(value: string): string {
  return DOT_TEST_TYPE_LABELS[value] ?? value;
}

export const DOT_TEST_STATUS_LABELS: Record<string, string> = {
  Scheduled: "Scheduled",
  Collected: "Collected",
  AwaitingResult: "At the lab",
  Completed: "Completed",
  Cancelled: "Cancelled",
};

export function dotTestStatusLabel(value: string): string {
  return DOT_TEST_STATUS_LABELS[value] ?? value;
}

export const DOT_TEST_RESULT_LABELS: Record<string, string> = {
  Pending: "Pending",
  Negative: "Negative",
  NegativeDilute: "Negative (dilute)",
  Positive: "Positive",
  Refusal: "Refusal",
  Adulterated: "Adulterated",
  Substituted: "Substituted",
  Invalid: "Invalid",
  Cancelled: "Cancelled",
};

export function dotTestResultLabel(value: string): string {
  return DOT_TEST_RESULT_LABELS[value] ?? value;
}

/**
 * Results 49 CFR 382.501 treats as a prohibition. An adulterated or substituted
 * specimen is a refusal, so it weighs the same as a positive.
 */
const VIOLATING_RESULTS = new Set(["Positive", "Refusal", "Adulterated", "Substituted"]);

export function isViolatingResult(result: string): boolean {
  return VIOLATING_RESULTS.has(result);
}

export function dotResultTone(result: string): BadgeTone {
  if (isViolatingResult(result)) return "inactive";
  if (result === "Negative" || result === "NegativeDilute") return "active";
  if (result === "Pending") return "warning";
  return "secondary";
}

export const DOT_VIOLATION_TYPE_LABELS: Record<string, string> = {
  PositiveTest: "Positive test",
  TestRefusal: "Refusal to test",
  AlcoholUse: "Alcohol use",
  DrugUse: "Drug use",
  ActualKnowledge: "Actual knowledge",
  Other: "Other",
};

export function dotViolationTypeLabel(value: string): string {
  return DOT_VIOLATION_TYPE_LABELS[value] ?? value;
}

export const DOT_VIOLATION_STATUS_LABELS: Record<string, string> = {
  Open: "Awaiting SAP referral",
  SAPEvaluation: "SAP evaluation",
  RTDPending: "Return-to-duty test required",
  FollowUp: "Follow-up testing",
  Resolved: "Resolved",
};

export function dotViolationStatusLabel(value: string): string {
  return DOT_VIOLATION_STATUS_LABELS[value] ?? value;
}

/**
 * Whether a violation at this stage keeps the driver off safety-sensitive
 * duty. Follow-up testing happens after they are back at work, so it does not.
 */
export function violationProhibits(status: string): boolean {
  return status === "Open" || status === "SAPEvaluation" || status === "RTDPending";
}

export const CLEARINGHOUSE_QUERY_TYPE_LABELS: Record<string, string> = {
  PreEmploymentFull: "Pre-employment full query",
  AnnualLimited: "Annual limited query",
  Full: "Full query",
  Limited: "Limited query",
};

export function clearinghouseQueryTypeLabel(value: string): string {
  return CLEARINGHOUSE_QUERY_TYPE_LABELS[value] ?? value;
}

export const CLEARINGHOUSE_RESULT_LABELS: Record<string, string> = {
  Pending: "Awaiting answer",
  NoViolations: "No violations",
  ViolationsFound: "Violations found",
  ConsentDenied: "Consent denied",
};

export function clearinghouseResultLabel(value: string): string {
  return CLEARINGHOUSE_RESULT_LABELS[value] ?? value;
}

export function clearinghouseResultTone(result: string): BadgeTone {
  if (result === "NoViolations") return "active";
  if (result === "ViolationsFound" || result === "ConsentDenied") return "inactive";
  return "warning";
}

export const RANDOM_PERIOD_LABELS: Record<string, string> = {
  Monthly: "Monthly",
  Quarterly: "Quarterly",
  SemiAnnual: "Twice a year",
  Annual: "Yearly",
};

export function randomPeriodLabel(value: string): string {
  return RANDOM_PERIOD_LABELS[value] ?? value;
}

export const RANDOM_ENTRY_STATUS_LABELS: Record<string, string> = {
  Selected: "Selected",
  Notified: "Notified",
  Completed: "Collected",
  Excused: "Excused",
  Missed: "Missed",
};

export function randomEntryStatusLabel(value: string): string {
  return RANDOM_ENTRY_STATUS_LABELS[value] ?? value;
}

/**
 * The FMCSA minimum annual random testing rates (49 CFR 382.305). A pool set
 * below either is usable but is not evidence of DOT compliance.
 */
export const DOT_MINIMUM_DRUG_RATE = 50;
export const DOT_MINIMUM_ALCOHOL_RATE = 10;

/**
 * The number of collections a round owes, from the annual rate and how many
 * rounds the year holds. It mirrors the server's arithmetic so the form can
 * show the figure before the draw is run — the server's answer is the one that
 * counts.
 */
export function roundsPerYear(period: string): number {
  switch (period) {
    case "Monthly":
      return 12;
    case "Quarterly":
      return 4;
    case "SemiAnnual":
      return 2;
    case "Annual":
      return 1;
    default:
      return 0;
  }
}

export function projectedRoundTarget(
  poolSize: number,
  ratePercent: number,
  period: string,
): number {
  const perYear = roundsPerYear(period);
  if (poolSize <= 0 || perYear === 0 || ratePercent <= 0) return 0;
  return Math.min(poolSize, Math.ceil((poolSize * ratePercent) / (100 * perYear)));
}
