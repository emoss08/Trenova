import { translate } from "@trenova/shared/i18n/runtime";
import { classificationIsRecordable } from "@trenova/shared/lib/injury";

const SECONDS_IN_DAY = 86_400;

/** How many years back the picker offers. Five is the retention period (29 CFR 1904.33). */
export const YEARS_OFFERED = 5;

/** The years the log can be read for, this year first. */
export function yearsOffered(thisYear: number, count = YEARS_OFFERED): number[] {
  return Array.from({ length: count }, (_, index) => thisYear - index);
}

// Structural shapes rather than the generated types, so the same rules serve
// the page and a fixture written from the schema by hand.
export type OshaTotalsLike = {
  deaths: number;
  daysAwayCases: number;
  jobTransferCases: number;
  otherRecordableCases: number;
  totalRecordableCases: number;
  totalDaysAway: number;
  totalDaysRestricted: number;
  injuryCount: number;
  skinDisorderCount: number;
  respiratoryCount: number;
  poisoningCount: number;
  hearingLossCount: number;
  otherIllnessCount: number;
  openCases: number;
};

export type OshaSummaryLike = {
  status: string;
  averageEmployees: number;
  totalHoursWorked: number;
  executiveName?: string | null;
  certifiedAt?: number | null;
  submittedAt?: number | null;
};

export type OshaCaseLike = {
  id: string;
  caseNumber: number;
  classification: string;
  illnessType: string;
  status: string;
  recordable: boolean;
  logName: string;
  description: string;
  location?: string | null;
  bodyPart?: string | null;
  occurredAt: number;
  daysAway: number;
  daysRestricted: number;
};

/**
 * The column of the paper 300 log a recordable case is checked in. The letters
 * are OSHA's own, so somebody reading this beside the form sees the same ones.
 */
export type LogColumn = "G" | "H" | "I" | "J";

export const LOG_COLUMNS: readonly {
  column: LogColumn;
  key: keyof Pick<
    OshaTotalsLike,
    "deaths" | "daysAwayCases" | "jobTransferCases" | "otherRecordableCases"
  >;
  label: string;
}[] = [
  { column: "G", key: "deaths", label: "Death" },
  { column: "H", key: "daysAwayCases", label: "Days away from work" },
  { column: "I", key: "jobTransferCases", label: "Job transfer or restriction" },
  { column: "J", key: "otherRecordableCases", label: "Other recordable cases" },
];

export function logColumn(classification: string): LogColumn | null {
  switch (classification) {
    case "Death":
      return "G";
    case "DaysAway":
      return "H";
    case "JobTransferOrRestriction":
      return "I";
    case "OtherRecordable":
      return "J";
    default:
      return null;
  }
}

/** The numbered injury and illness types on the 300A, in the form's order. */
export const ILLNESS_TYPES: readonly {
  number: number;
  key: keyof Pick<
    OshaTotalsLike,
    | "injuryCount"
    | "skinDisorderCount"
    | "respiratoryCount"
    | "poisoningCount"
    | "hearingLossCount"
    | "otherIllnessCount"
  >;
  illnessType: string;
  label: string;
}[] = [
  { number: 1, key: "injuryCount", illnessType: "Injury", label: "Injuries" },
  { number: 2, key: "skinDisorderCount", illnessType: "SkinDisorder", label: "Skin disorders" },
  {
    number: 3,
    key: "respiratoryCount",
    illnessType: "RespiratoryCondition",
    label: "Respiratory conditions",
  },
  { number: 4, key: "poisoningCount", illnessType: "Poisoning", label: "Poisonings" },
  { number: 5, key: "hearingLossCount", illnessType: "HearingLoss", label: "Hearing loss" },
  {
    number: 6,
    key: "otherIllnessCount",
    illnessType: "OtherIllness",
    label: "All other illnesses",
  },
];

export function illnessTypeNumber(illnessType: string): number | null {
  return ILLNESS_TYPES.find((type) => type.illnessType === illnessType)?.number ?? null;
}

export type PostingPhase = "before" | "open" | "closed";

export type PostingState = {
  phase: PostingPhase;
  /** Whole days until the window opens, or left inside it. Zero once it has closed. */
  days: number;
};

/**
 * Where today sits against the February 1 – April 30 posting window
 * (29 CFR 1904.32(b)(6)). The day counts round up, so the day the window opens
 * reads as "1 day left" of waiting rather than none.
 */
export function postingState(now: number, postFrom: number, postThrough: number): PostingState {
  if (now < postFrom) {
    return { phase: "before", days: Math.ceil((postFrom - now) / SECONDS_IN_DAY) };
  }
  if (now <= postThrough) {
    return { phase: "open", days: Math.max(1, Math.ceil((postThrough - now) / SECONDS_IN_DAY)) };
  }
  return { phase: "closed", days: 0 };
}

/** Whether the summary carries the establishment figures a certification needs. */
export function hasEstablishmentFigures(summary: OshaSummaryLike | null | undefined): boolean {
  return Boolean(summary && summary.averageEmployees > 0 && summary.totalHoursWorked > 0);
}

export function isCertified(summary: OshaSummaryLike | null | undefined): boolean {
  return summary?.status === "Certified";
}

export type TrackStepId = "close" | "figures" | "certify" | "post" | "submit";

export type TrackStepState = "done" | "active" | "pending";

export type TrackStep = {
  id: TrackStepId;
  label: string;
  detail: string;
  state: TrackStepState;
};

export type CertificationTrackInput = {
  totals: Pick<OshaTotalsLike, "openCases" | "totalRecordableCases">;
  summary: OshaSummaryLike | null | undefined;
  postFrom: number;
  postThrough: number;
  now: number;
  formatDate: (unix: number) => string;
};

function plural(count: number, singular: string, pluralForm = `${singular}s`): string {
  return `${count} ${count === 1 ? singular : pluralForm}`;
}

/**
 * The road from a year's log to a posted 300A, as a sequence: cases must stop
 * moving before the figures are signed for, the signature comes before the
 * posting, and the posting window is fixed by the rule. The first step not yet
 * done is the one to act on; a time-gated step that cannot be acted on yet
 * still counts as active so the reader sees what they are waiting for.
 */
export function certificationTrack(input: CertificationTrackInput): TrackStep[] {
  const { totals, summary, postFrom, postThrough, now, formatDate } = input;
  const closed = totals.openCases === 0;
  const figures = hasEstablishmentFigures(summary);
  const certified = isCertified(summary);
  const posting = postingState(now, postFrom, postThrough);
  const submitted = Boolean(summary?.submittedAt);

  const steps: TrackStep[] = [
    {
      id: "close",
      label: translate("Every case closed"),
      detail: closed
        ? totals.totalRecordableCases === 0
          ? "Nothing recordable this year; the summary still has to be posted"
          : "No case is still accruing days"
        : `${plural(totals.openCases, "case")} still accruing days`,
      state: closed ? "done" : "active",
    },
    {
      id: "figures",
      label: translate("Establishment figures recorded"),
      detail: figures
        ? `${summary!.averageEmployees.toLocaleString("en-US")} employees on average over ${summary!.totalHoursWorked.toLocaleString("en-US")} hours`
        : summary
          ? "Add the average headcount and the hours worked"
          : "Start the summary to record headcount and hours",
      state: figures ? "done" : "pending",
    },
    {
      id: "certify",
      label: translate("Certified by an executive"),
      detail: certified
        ? `${summary?.executiveName?.trim() || "Certified"}${summary?.certifiedAt ? ` on ${formatDate(summary.certifiedAt)}` : ""}`
        : "A company executive signs that the summary is true",
      state: certified ? "done" : "pending",
    },
    {
      id: "post",
      label: translate("Posted where employees can see it"),
      detail:
        posting.phase === "before"
          ? `Window opens ${formatDate(postFrom)}, in ${plural(posting.days, "day")}`
          : posting.phase === "open"
            ? `Window is open until ${formatDate(postThrough)}, ${plural(posting.days, "day")} left`
            : `Window closed ${formatDate(postThrough)}`,
      state: certified && posting.phase === "closed" ? "done" : "pending",
    },
    {
      id: "submit",
      label: translate("Submitted electronically"),
      detail: submitted
        ? `Sent ${formatDate(summary!.submittedAt!)}`
        : "Due March 2 where the establishment is required to submit",
      state: submitted ? "done" : "pending",
    },
  ];

  // Steps are sequential: the first one not done is the one to act on.
  const first = steps.findIndex((step) => step.state !== "done");
  if (first >= 0) {
    for (let index = 0; index < steps.length; index += 1) {
      if (steps[index].state === "done") continue;
      steps[index].state = index === first ? "active" : "pending";
    }
  }
  return steps;
}

export type CertifyBlocker = "no-summary" | "no-figures" | "open-cases";

/**
 * Why certifying would be refused, before the server has to say so. The
 * server holds the same rules; this is only to say them first.
 */
export function certifyBlocker(
  summary: OshaSummaryLike | null | undefined,
  totals: Pick<OshaTotalsLike, "openCases">,
): CertifyBlocker | null {
  if (!summary) return "no-summary";
  if (!hasEstablishmentFigures(summary)) return "no-figures";
  if (totals.openCases > 0) return "open-cases";
  return null;
}

export function certifyBlockerMessage(blocker: CertifyBlocker, openCases: number): string {
  switch (blocker) {
    case "no-summary":
      return "Start the summary and record its establishment figures first";
    case "no-figures":
      return "Record the average headcount and hours worked first";
    case "open-cases":
      return `Close the ${plural(openCases, "case")} still accruing days first`;
  }
}

export type CaseFilter = "all" | "recordable" | "open" | "off";

export const CASE_FILTERS: readonly { value: CaseFilter; label: string }[] = [
  { value: "all", label: "Every case" },
  { value: "recordable", label: "On the log" },
  { value: "open", label: "Still open" },
  { value: "off", label: "Off the log" },
];

export function caseFilterCounts<TCase extends OshaCaseLike>(
  cases: readonly TCase[],
): Record<CaseFilter, number> {
  const counts: Record<CaseFilter, number> = { all: cases.length, recordable: 0, open: 0, off: 0 };
  for (const entry of cases) {
    if (entry.recordable) counts.recordable += 1;
    else counts.off += 1;
    if (entry.status === "Open") counts.open += 1;
  }
  return counts;
}

function matchesFilter(entry: OshaCaseLike, filter: CaseFilter): boolean {
  switch (filter) {
    case "recordable":
      return entry.recordable;
    case "open":
      return entry.status === "Open";
    case "off":
      return !entry.recordable;
    default:
      return true;
  }
}

/**
 * The words a reader would search a log by: the name as posted, what happened,
 * where, and the case number. The confidential name behind a privacy case is
 * deliberately not searched, so typing it cannot reveal which row is theirs.
 */
export function matchesQuery(entry: OshaCaseLike, query: string): boolean {
  const needle = query.trim().toLowerCase();
  if (needle === "") return true;
  const haystack = [
    String(entry.caseNumber),
    entry.logName,
    entry.description,
    entry.location ?? "",
    entry.bodyPart ?? "",
  ]
    .join(" ")
    .toLowerCase();
  return haystack.includes(needle);
}

/** The rows the log shows, in case-number order, which is how the paper log reads. */
export function filterCases<TCase extends OshaCaseLike>(
  cases: readonly TCase[],
  filter: CaseFilter,
  query: string,
): TCase[] {
  return cases
    .filter((entry) => matchesFilter(entry, filter) && matchesQuery(entry, query))
    .sort((a, b) => a.caseNumber - b.caseNumber);
}

export function caseLabel(entry: { caseYear: number; caseNumber: number }): string {
  return `${entry.caseYear}-${entry.caseNumber}`;
}

/** Whether the classification keeps a case off the totals, the distinction the log turns on. */
export function isOffTheLog(classification: string): boolean {
  return !classificationIsRecordable(classification);
}

export type SummaryCaption = "Certified" | "Draft" | "No summary";

export function summaryCaption(
  summary: Pick<OshaSummaryLike, "status"> | undefined,
): SummaryCaption {
  if (!summary) return "No summary";
  return summary.status === "Certified" ? "Certified" : "Draft";
}

export type DaysLost = { away: number; restricted: number; total: number };

export function daysLost(
  totals: Pick<OshaTotalsLike, "totalDaysAway" | "totalDaysRestricted">,
): DaysLost {
  return {
    away: totals.totalDaysAway,
    restricted: totals.totalDaysRestricted,
    total: totals.totalDaysAway + totals.totalDaysRestricted,
  };
}

/** The calendar year a Unix instant falls in, read the way the server counts cases. */
export function yearOf(unixSeconds: number): number {
  return new Date(unixSeconds * 1000).getUTCFullYear();
}
