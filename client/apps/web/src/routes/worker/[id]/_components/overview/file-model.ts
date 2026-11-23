import {
  choiceLabel,
  drugAlcoholStatusChoices,
  ptoTypeChoices,
  safetyRatingChoices,
} from "@/lib/choices";
import type { WorkerOverview } from "@/lib/graphql/worker-overview";
import { withoutRecordIds } from "@/lib/record-ids";
import type { WorkerAreas } from "@/routes/worker/worker-areas";
import type { WorkerArea, WorkerTab, WorkerView } from "@/routes/worker/use-worker-state";
import { quotedPayRate, reviewDueInDays } from "../overview-model";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatNumber } from "@trenova/shared/i18n/format";
import { daysUntil, formatUnixDateMedium, formatUnixMonthDay } from "@trenova/shared/lib/date";
import { formatMinor } from "@trenova/shared/lib/money";

export type FileCellKey =
  | "credentials"
  | "dq-file"
  | "drug-and-alcohol"
  | "training"
  | "safety"
  | "reviews"
  | "checklists"
  | "time-off"
  | "leave"
  | "pay-profile"
  | "benefits"
  | "driver-portal";

export type FileNoteTone = "muted" | "warning" | "danger";

export type FileCell = {
  key: FileCellKey;
  area: WorkerArea;
  label: string;
  /** Null when the section is withheld: the reader may not see it. */
  state: string | null;
  note: { text: string; tone: FileNoteTone } | null;
};

type CellArea = { key: FileCellKey; tab: WorkerTab; view: WorkerView | null };

/** The parts of the record the file lays out, in the order it reads them. */
export const FILE_CELL_AREAS: readonly CellArea[] = [
  { key: "credentials", tab: "qualification", view: "credentials" },
  { key: "dq-file", tab: "qualification", view: "dq-file" },
  { key: "drug-and-alcohol", tab: "qualification", view: "drug-and-alcohol" },
  { key: "training", tab: "development", view: "training" },
  { key: "safety", tab: "safety", view: null },
  { key: "reviews", tab: "development", view: "reviews" },
  { key: "checklists", tab: "employment", view: "checklists" },
  { key: "time-off", tab: "time", view: "time-off" },
  { key: "leave", tab: "time", view: "leave" },
  { key: "pay-profile", tab: "pay", view: "pay-profile" },
  { key: "benefits", tab: "pay", view: "benefits" },
  { key: "driver-portal", tab: "files", view: "driver-portal" },
];

type Words = Pick<FileCell, "state" | "note">;

const WITHHELD: Words = { state: null, note: null };

function decimal(value: string): string {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? formatNumber(parsed, { maximumFractionDigits: 2 }) : value;
}

function note(text: string, tone: FileNoteTone = "muted"): Words["note"] {
  return { text, tone };
}

function credentialsCell(overview: WorkerOverview, t: TranslateFn): Words {
  const summary = overview.credentials;
  if (!summary) return WITHHELD;
  const state =
    summary.requiredCount === 0
      ? t("No required credentials")
      : t(
          "{0} of {1} required on file",
          summary.requiredCount - summary.missingCount,
          summary.requiredCount,
        );
  if (summary.expiredCount > 0)
    return { state, note: note(t("{0} expired", summary.expiredCount), "danger") };
  if (summary.missingCount > 0)
    return { state, note: note(t("{0} missing", summary.missingCount), "danger") };
  if (summary.expiringCount > 0) {
    return { state, note: note(t("{0} expiring", summary.expiringCount), "warning") };
  }
  return { state, note: note(t("All in date")) };
}

function dqFileCell(overview: WorkerOverview, t: TranslateFn): Words {
  const file = overview.qualificationFile;
  if (!file) return WITHHELD;
  const state = file.complete ? t("Complete") : t("Incomplete");
  if (file.expired > 0) return { state, note: note(t("{0} expired", file.expired), "danger") };
  if (file.missingRequired > 0) {
    return { state, note: note(t("{0} missing", file.missingRequired), "danger") };
  }
  if (file.safetyHistoryLate)
    return { state, note: note(t("Safety history request late"), "danger") };
  if (file.outstanding > 0) {
    return { state, note: note(t("{0} outstanding", file.outstanding), "warning") };
  }
  if (file.expiringSoon > 0) {
    return { state, note: note(t("{0} expiring", file.expiringSoon), "warning") };
  }
  return { state, note: note(t("Nothing outstanding")) };
}

function drugAlcoholCell(overview: WorkerOverview, t: TranslateFn): Words {
  const section = overview.drugAlcohol;
  if (!section) return WITHHELD;
  const { standing, nextQueryDaysUntil } = section;
  const state = choiceLabel(drugAlcoholStatusChoices, standing.status);
  if (standing.status === "Prohibited")
    return { state, note: note(t("Return to duty required"), "danger") };
  if (standing.openTestCount > 0) {
    return {
      state,
      note: note(
        t(
          "{0, plural, one {# test awaiting a result} other {# tests awaiting a result}}",
          standing.openTestCount,
        ),
        "warning",
      ),
    };
  }
  const due = standing.nextClearinghouseQueryDue;
  if (due && nextQueryDaysUntil !== null) {
    if (nextQueryDaysUntil < 0) {
      return { state, note: note(t("Query overdue since {0}", formatUnixMonthDay(due)), "danger") };
    }
    return {
      state,
      note: note(
        t("Query due {0}", formatUnixMonthDay(due)),
        nextQueryDaysUntil <= 30 ? "warning" : "muted",
      ),
    };
  }
  return { state, note: note(t("No query due")) };
}

function trainingCell(overview: WorkerOverview, t: TranslateFn): Words {
  const summary = overview.training;
  if (!summary) return WITHHELD;
  const state =
    summary.requiredCount === 0
      ? t("No required courses")
      : t("{0} of {1} current", summary.currentCount, summary.requiredCount);
  const behind = summary.overdueCount + summary.expiredCount + summary.missingCount;
  if (behind > 0) return { state, note: note(t("{0} behind", behind), "danger") };
  const soon = summary.dueCount + summary.expiringCount;
  if (soon > 0) return { state, note: note(t("{0} due soon", soon), "warning") };
  return { state, note: note(t("All current")) };
}

function safetyCell(overview: WorkerOverview, t: TranslateFn): Words {
  const card = overview.safety;
  if (!card) return WITHHELD;
  const state = t("Score {0} · {1}", card.score, choiceLabel(safetyRatingChoices, card.rating));
  const tone: FileNoteTone =
    card.rating === "AtRisk" ? "danger" : card.rating === "Watch" ? "warning" : "muted";
  return {
    state,
    note:
      card.activePoints > 0
        ? note(
            t("{0, plural, one {# active point} other {# active points}}", card.activePoints),
            tone,
          )
        : note(t("No active points"), tone),
  };
}

function reviewsCell(overview: WorkerOverview, t: TranslateFn): Words {
  const { openReview, lastReview, nextReviewAt } = overview;
  const state = openReview
    ? t("In progress: {0}", withoutRecordIds(openReview.title))
    : lastReview?.overallScore
      ? t("Last: {0}", decimal(lastReview.overallScore))
      : lastReview
        ? t("Last: {0}", withoutRecordIds(lastReview.title))
        : t("No review yet");
  if (nextReviewAt) {
    const overdue = reviewDueInDays(nextReviewAt, overview.asOf) < 0;
    return {
      state,
      note: overdue
        ? note(t("Overdue since {0}", formatUnixDateMedium(nextReviewAt)), "danger")
        : note(t("Next due {0}", formatUnixDateMedium(nextReviewAt))),
    };
  }
  return { state, note: null };
}

function checklistsCell(overview: WorkerOverview, t: TranslateFn): Words {
  const checklist = overview.checklist;
  const open = overview.employment?.openChecklistCount ?? null;
  if (!checklist) {
    return { state: t("No open checklist"), note: null };
  }
  const state = withoutRecordIds(checklist.name);
  const progress = checklist.progress;
  if (!progress) {
    return { state, note: open && open > 1 ? note(t("{0} open", open)) : null };
  }
  if (progress.overdue > 0) {
    return {
      state,
      note: note(
        t(
          "{0} of {1} settled, {2} overdue",
          progress.requiredDone,
          progress.requiredTotal,
          progress.overdue,
        ),
        "danger",
      ),
    };
  }
  return {
    state,
    note: note(t("{0} of {1} settled", progress.requiredDone, progress.requiredTotal)),
  };
}

function timeOffCell(overview: WorkerOverview, t: TranslateFn): Words {
  const balances = overview.pto;
  if (!balances) return WITHHELD;
  const tracked = balances.filter((balance) => balance.tracked);
  const lead = tracked.find((balance) => balance.ptoType === "Vacation") ?? tracked[0];
  if (!lead) return { state: t("No tracked balances"), note: null };
  const available = Number(lead.availableDays);
  const state = Number.isFinite(available)
    ? t(
        "{0}: {1, plural, one {# day} other {# days}}",
        choiceLabel(ptoTypeChoices, lead.ptoType),
        available,
      )
    : t("{0}: {1} days", choiceLabel(ptoTypeChoices, lead.ptoType), lead.availableDays);
  const pending = tracked.reduce((sum, balance) => sum + (Number(balance.pendingDays) || 0), 0);
  return {
    state,
    note:
      pending > 0
        ? note(t("{0, plural, one {# day requested} other {# days requested}}", pending))
        : note(t("Nothing requested")),
  };
}

function leaveCell(overview: WorkerOverview, t: TranslateFn): Words {
  const leave = overview.leave;
  if (!leave) return WITHHELD;
  const state =
    leave.openCaseCount > 0
      ? t("{0, plural, one {# open case} other {# open cases}}", leave.openCaseCount)
      : t("No open case");
  if (leave.exhausted) return { state, note: note(t("Entitlement used up"), "danger") };
  return {
    state,
    note: note(
      t("{0} of {1} hours left", decimal(leave.remainingHours), decimal(leave.totalHours)),
    ),
  };
}

function payCell(overview: WorkerOverview, t: TranslateFn): Words {
  const pay = overview.pay;
  if (!pay)
    return {
      state: t("No pay profile"),
      note: note(t("Assign one to pay this driver"), "warning"),
    };
  const rate = quotedPayRate(pay.headline, t);
  return {
    state: withoutRecordIds(pay.profileName),
    note: rate ? note(rate) : null,
  };
}

function benefitsCell(overview: WorkerOverview, t: TranslateFn): Words {
  const benefits = overview.benefits;
  if (!benefits) return WITHHELD;
  const state =
    benefits.coveredCount > 0
      ? t("{0, plural, one {# plan covered} other {# plans covered}}", benefits.coveredCount)
      : t("No plans covered");
  if (benefits.notStartedCount > 0) {
    return {
      state,
      note: note(
        t(
          "{0, plural, one {# plan not started} other {# plans not started}}",
          benefits.notStartedCount,
        ),
        "warning",
      ),
    };
  }
  return benefits.coveredCount > 0
    ? { state, note: note(t("{0} a period", formatMinor(benefits.employeeCostPerPeriodMinor))) }
    : { state, note: null };
}

function portalCell(overview: WorkerOverview, t: TranslateFn, now: number): Words {
  const portal = overview.portal;
  if (!portal) return WITHHELD;
  const state =
    portal.access === "Linked"
      ? t("Linked")
      : portal.access === "Invited"
        ? t("Invited")
        : t("Not set up");
  const waiting = portal.pendingProfileChangeCount ?? 0;
  if (waiting > 0) {
    return {
      state,
      note: note(
        t("{0, plural, one {# change waiting} other {# changes waiting}}", waiting),
        "warning",
      ),
    };
  }
  if (portal.lastSignInAt) {
    const days = Math.max(0, -daysUntil(portal.lastSignInAt, { now }));
    return {
      state,
      note: note(
        days === 0
          ? t("Signed in today")
          : t("Signed in {0, plural, one {# day} other {# days}} ago", days),
      ),
    };
  }
  return { state, note: portal.access === "Linked" ? note(t("Never signed in")) : null };
}

const CELL_WORDS: Record<
  FileCellKey,
  (overview: WorkerOverview, t: TranslateFn, now: number) => Words
> = {
  credentials: credentialsCell,
  "dq-file": dqFileCell,
  "drug-and-alcohol": drugAlcoholCell,
  training: trainingCell,
  safety: safetyCell,
  reviews: reviewsCell,
  checklists: checklistsCell,
  "time-off": timeOffCell,
  leave: leaveCell,
  "pay-profile": payCell,
  benefits: benefitsCell,
  "driver-portal": portalCell,
};

function cellLabel(key: FileCellKey, t: TranslateFn): string {
  switch (key) {
    case "credentials":
      return t("Credentials");
    case "dq-file":
      return t("DQ file");
    case "drug-and-alcohol":
      return t("Drug and alcohol");
    case "training":
      return t("Training");
    case "safety":
      return t("Safety");
    case "reviews":
      return t("Reviews");
    case "checklists":
      return t("Checklists");
    case "time-off":
      return t("Time off");
    case "leave":
      return t("Leave");
    case "pay-profile":
      return t("Pay");
    case "benefits":
      return t("Benefits");
    case "driver-portal":
      return t("Driver portal");
  }
}

/**
 * One cell per part of the record the reader can open, on the tenant's plan and within their
 * permissions, each with where it stands. A section the read left out is drawn withheld.
 */
export function fileCells(
  overview: WorkerOverview,
  areas: Pick<WorkerAreas, "isDrawable">,
  t: TranslateFn,
  /** Unix seconds, from the reader's `useNow`, for the counts of days since. */
  now: number,
): FileCell[] {
  return FILE_CELL_AREAS.filter(({ tab, view }) => areas.isDrawable({ tab, view })).map(
    ({ key, tab, view }) => ({
      key,
      area: { tab, view },
      label: cellLabel(key, t),
      ...CELL_WORDS[key](overview, t, now),
    }),
  );
}

/** How many columns the file takes at a width: three, two, then one. */
export function fileColumns(width: number): 1 | 2 | 3 {
  if (width <= 0 || width >= 720) return 3;
  if (width >= 440) return 2;
  return 1;
}

/** Which hairlines a cell draws: a rule on its right inside a row, one below it above the last row. */
export function fileCellRules(
  index: number,
  count: number,
  columns: number,
): { right: boolean; bottom: boolean } {
  const lastRowStart = count - (count % columns === 0 ? columns : count % columns);
  return {
    right: index % columns !== columns - 1 && index !== count - 1,
    bottom: index < lastRowStart,
  };
}
