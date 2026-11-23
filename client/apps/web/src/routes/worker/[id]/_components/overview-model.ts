import type { SectionStripItem, SectionStripTone } from "@/components/section-strip";
import { dutyStatusLabel } from "@/lib/hos";
import type { WorkerOverview } from "@/lib/graphql/worker-overview";
import { workerDestinationTarget } from "@/lib/worker-concerns";
import { WORKER_TABS, type WorkerTab, type WorkerView } from "@/routes/worker/use-worker-state";
import { formatPayRate } from "./pay/pay-model";
import type { PayCalcMethod } from "@trenova/graphql/generated/graphql";
import {
  Beaker02Icon,
  BrainIcon,
  CalendarClockIcon,
  CalendarRangeIcon,
  ClipboardCheckIcon,
  ClockRewindIcon,
  File06Icon,
  FolderCheckIcon,
  GraduationHat01Icon,
  HeartHandIcon,
  IdCardIcon,
  Phone01Icon,
  Speedometer03Icon,
  Star01Icon,
  SunsetIcon,
  Wallet02Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { formatDuration, formatNumber } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { daysUntil, formatUnixDateMedium, toUserWallClock } from "@trenova/shared/lib/date";
import { HOUR_MS } from "@/lib/time-units";

/** How one view of the record stands, in one line and a tone, or nothing when it cannot be said. */
export type StripState = { state: string | null; tone: SectionStripTone };

const NOTHING: StripState = { state: null, tone: "none" };

export type WorkerTabMark =
  | { kind: "dot"; tone: "warning" | "danger"; label: string }
  | { kind: "count"; value: number };

/**
 * What each tab carries after its name: a red dot for things that stop or need action, an amber
 * one for things worth a look, from the concerns the server ranked; Files carries its document
 * count. The Profile tab's unsaved dot is the page's, not the read's.
 */
export function workerTabMarks(
  overview: WorkerOverview | undefined,
  t: TranslateFn,
): Partial<Record<WorkerTab, WorkerTabMark>> {
  if (!overview) return {};
  const critical = new Map<WorkerTab, number>();
  const warning = new Map<WorkerTab, number>();
  for (const concern of overview.concerns) {
    const { tab } = workerDestinationTarget(concern.destination);
    if (concern.severity === "Critical") critical.set(tab, (critical.get(tab) ?? 0) + 1);
    else if (concern.severity === "Warning") warning.set(tab, (warning.get(tab) ?? 0) + 1);
  }
  const marks: Partial<Record<WorkerTab, WorkerTabMark>> = {};
  for (const tab of WORKER_TABS) {
    if (tab === "overview") continue;
    const needAction = critical.get(tab) ?? 0;
    const worthALook = warning.get(tab) ?? 0;
    if (needAction > 0) {
      marks[tab] = {
        kind: "dot",
        tone: "danger",
        label: t(
          "{0, plural, one {# thing needs action} other {# things need action}}",
          needAction,
        ),
      };
    } else if (worthALook > 0) {
      marks[tab] = {
        kind: "dot",
        tone: "warning",
        label: t(
          "{0, plural, one {# thing is worth a look} other {# things are worth a look}}",
          worthALook,
        ),
      };
    }
  }
  if (!marks.files && overview.documentCount != null && overview.documentCount > 0) {
    marks.files = { kind: "count", value: overview.documentCount };
  }
  return marks;
}

function joined(...parts: (string | null | undefined | false)[]): string {
  return parts.filter((part): part is string => Boolean(part)).join(" · ");
}

/**
 * Calendar days from the overview's read to the next review's due date, in the reader's zone:
 * negative once the day has passed. The Overview strip, the file grid and the Reviews view all
 * count a review overdue this one way, so a review due today is due, not overdue.
 */
export function reviewDueInDays(nextReviewAt: number, asOf: number): number {
  return daysUntil(nextReviewAt, { now: asOf });
}

function historyState(o: WorkerOverview, t: TranslateFn): StripState {
  const employment = o.employment;
  if (!employment) return NOTHING;
  const events =
    employment.eventCount === 0
      ? t("No events yet")
      : t("{0, plural, one {# event} other {# events}}", employment.eventCount);
  const hired = employment.hiredAt
    ? t("hired {0}", formatUnixDateMedium(employment.hiredAt))
    : null;
  return { state: joined(events, hired), tone: "none" };
}

function checklistsState(o: WorkerOverview, t: TranslateFn): StripState {
  const open = o.employment?.openChecklistCount;
  if (open == null) return NOTHING;
  if (open === 0) return { state: t("None open"), tone: "none" };
  const progress = o.checklist?.progress;
  const settled = progress
    ? t("{0} of {1} settled", progress.requiredDone, progress.requiredTotal)
    : null;
  const overdue = (progress?.overdue ?? 0) > 0;
  return {
    state: joined(t("{0} open", open), settled),
    tone: overdue ? "warning" : "none",
  };
}

function credentialsState(o: WorkerOverview, t: TranslateFn): StripState {
  const c = o.credentials;
  if (!c) return NOTHING;
  if (c.requiredCount === 0) return { state: t("None required"), tone: "none" };
  const onFile = t(
    "{0} of {1} required on file",
    c.requiredCount - c.missingCount,
    c.requiredCount,
  );
  if (c.expiredCount > 0) {
    return {
      state: joined(onFile, t("{0} expired", c.expiredCount)),
      tone: "danger",
    };
  }
  if (c.missingCount > 0) return { state: onFile, tone: "danger" };
  if (c.expiringCount > 0) {
    return {
      state: joined(onFile, t("{0} expiring", c.expiringCount)),
      tone: "warning",
    };
  }
  return { state: onFile, tone: "none" };
}

function dqFileState(o: WorkerOverview, t: TranslateFn): StripState {
  const file = o.qualificationFile;
  if (!file) return NOTHING;
  if (!file.complete) {
    return {
      state:
        file.outstanding > 0
          ? joined(t("Incomplete"), t("{0} outstanding", file.outstanding))
          : t("Incomplete"),
      tone: "danger",
    };
  }
  if (file.expired > 0) {
    return { state: joined(t("Complete"), t("{0} expired", file.expired)), tone: "danger" };
  }
  if (file.expiringSoon > 0) {
    return {
      state: joined(t("Complete"), t("{0} expiring", file.expiringSoon)),
      tone: "warning",
    };
  }
  return { state: t("Complete"), tone: "success" };
}

function queryDue(days: number | null, t: TranslateFn): { text: string | null; urgent: boolean } {
  if (days == null) return { text: null, urgent: false };
  if (days < 0) {
    return {
      text: t("{0, plural, one {query overdue by # day} other {query overdue by # days}}", -days),
      urgent: true,
    };
  }
  if (days === 0) return { text: t("query due today"), urgent: true };
  return {
    text: t("{0, plural, one {query due in # day} other {query due in # days}}", days),
    urgent: days <= 30,
  };
}

function drugAlcoholState(o: WorkerOverview, t: TranslateFn): StripState {
  const da = o.drugAlcohol;
  if (!da) return NOTHING;
  const query = queryDue(da.nextQueryDaysUntil, t);
  switch (da.standing.status) {
    case "Prohibited":
      return { state: joined(t("Prohibited"), query.text), tone: "danger" };
    case "Pending":
      return { state: joined(t("Result pending"), query.text), tone: "warning" };
    case "Unknown":
      return {
        state: joined(t("No standing yet"), query.text),
        tone: query.urgent ? "warning" : "none",
      };
    case "Clear":
      return {
        state: joined(t("Clear"), query.text),
        tone:
          da.nextQueryDaysUntil != null && da.nextQueryDaysUntil < 0
            ? "danger"
            : query.urgent
              ? "warning"
              : "none",
      };
  }
}

function trainingState(o: WorkerOverview, t: TranslateFn): StripState {
  const tr = o.training;
  if (!tr) return NOTHING;
  if (tr.requiredCount === 0) return { state: t("None required"), tone: "none" };
  const current = t("{0} of {1} current", tr.currentCount, tr.requiredCount);
  const lapsed = tr.overdueCount + tr.expiredCount + tr.missingCount;
  if (lapsed > 0) return { state: joined(current, t("{0} behind", lapsed)), tone: "danger" };
  const soon = tr.dueCount + tr.expiringCount;
  if (soon > 0) return { state: joined(current, t("{0} due soon", soon)), tone: "warning" };
  return { state: current, tone: "none" };
}

function reviewsState(o: WorkerOverview, t: TranslateFn): StripState {
  if (o.openReview) {
    return {
      state: t("Under way · period ends {0}", formatUnixDateMedium(o.openReview.periodEnd)),
      tone: "none",
    };
  }
  if (o.nextReviewAt != null) {
    const days = reviewDueInDays(o.nextReviewAt, o.asOf);
    if (days < 0) {
      return {
        state: t("Overdue since {0}", formatUnixDateMedium(o.nextReviewAt)),
        tone: "danger",
      };
    }
    return {
      state: t("Next due {0}", formatUnixDateMedium(o.nextReviewAt)),
      tone: days <= 30 ? "warning" : "none",
    };
  }
  if (o.lastReview) {
    return {
      state: t("Last closed {0}", formatUnixDateMedium(o.lastReview.periodEnd)),
      tone: "none",
    };
  }
  return { state: t("No reviews yet"), tone: "none" };
}

function hoursState(o: WorkerOverview, t: TranslateFn): StripState {
  const hos = o.hos;
  if (!hos) return NOTHING;
  switch (hos.link) {
    case "NotConnected":
      return { state: t("Telematics not connected"), tone: "none" };
    case "NotLinked":
      return { state: t("Not linked to a telematics driver"), tone: "none" };
    case "NoData":
      return { state: t("No logs yet"), tone: "none" };
    case "Reporting": {
      const status = hos.dutyStatus ? dutyStatusLabel(hos.dutyStatus) : null;
      const remaining = hos.driveRemainingMs;
      if (remaining == null) return { state: status, tone: "none" };
      if (remaining <= 0) return { state: joined(status, t("no drive time left")), tone: "danger" };
      return {
        state: joined(status, t("{0} drive left", formatDuration(remaining / 1000))),
        tone: remaining < HOUR_MS ? "warning" : "none",
      };
    }
  }
}

function scheduleState(o: WorkerOverview, t: TranslateFn): StripState {
  const schedule = o.schedule;
  if (!schedule) return NOTHING;
  return {
    state: schedule.currentShiftName ? t("On {0}", schedule.currentShiftName) : t("Not on a shift"),
    tone: "none",
  };
}

function timeOffState(o: WorkerOverview, t: TranslateFn): StripState {
  const balances = o.pto;
  if (!balances) return NOTHING;
  const tracked = balances.filter((balance) => balance.tracked);
  if (tracked.length === 0) return { state: t("No balances tracked"), tone: "none" };
  const vacation = tracked.find((balance) => balance.ptoType === "Vacation");
  if (vacation) {
    return {
      state: t(
        "{0, plural, one {# vacation day available} other {# vacation days available}}",
        Number(vacation.availableDays),
      ),
      tone: "none",
    };
  }
  const total = tracked.reduce((sum, balance) => sum + Number(balance.availableDays), 0);
  return {
    state: t("{0, plural, one {# day available} other {# days available}}", total),
    tone: "none",
  };
}

function leaveState(o: WorkerOverview, t: TranslateFn): StripState {
  const leave = o.leave;
  if (!leave) return NOTHING;
  if (leave.exhausted) {
    return {
      state: joined(
        leave.openCaseCount > 0
          ? t("{0, plural, one {# open case} other {# open cases}}", leave.openCaseCount)
          : null,
        t("entitlement used up"),
      ),
      tone: "danger",
    };
  }
  if (leave.openCaseCount === 0) return { state: t("No open case"), tone: "none" };
  return {
    state: joined(
      t("{0, plural, one {# open case} other {# open cases}}", leave.openCaseCount),
      t(
        "{0} of {1} hours left",
        formatNumber(Number(leave.remainingHours)),
        formatNumber(Number(leave.totalHours)),
      ),
    ),
    tone: "warning",
  };
}

/** A pay rate as the trade says it: "$0.62 a loaded mile", "22% of revenue". */
export function payRateText(method: PayCalcMethod, rate: string, t: TranslateFn): string {
  const money = formatPayRate(method, rate);
  switch (method) {
    case "PerLoadedMile":
      return t("{0} a loaded mile", money);
    case "PerTotalMile":
      return t("{0} a mile", money);
    case "PerEmptyMile":
      return t("{0} an empty mile", money);
    case "PerHour":
      return t("{0} an hour", money);
    case "PerDay":
      return t("{0} a day", money);
    case "PerStop":
      return t("{0} a stop", money);
    case "PerEvent":
      return t("{0} an event", money);
    case "FlatPerShipment":
      return t("{0} a load", money);
    case "PercentOfRevenue":
      return t("{0} of revenue", money);
  }
}

/**
 * The rate a strip or cell quotes beside the profile's name, or null when there is none to
 * quote: no component, or one whose own rate is zero because its bands carry the rates.
 */
export function quotedPayRate(
  headline: { method: PayCalcMethod; rate: string } | null | undefined,
  t: TranslateFn,
): string | null {
  if (!headline || !(Number(headline.rate) > 0)) return null;
  return payRateText(headline.method, headline.rate, t);
}

function payState(o: WorkerOverview, t: TranslateFn, drawable: boolean): StripState {
  const pay = o.pay;
  // The read sends no pay both when it is withheld and when none is assigned; a view the plan and
  // the reader's permissions draw is never withheld, so there it means none.
  if (!pay) return drawable ? { state: t("No pay profile"), tone: "none" } : NOTHING;
  return {
    state: joined(pay.profileName, quotedPayRate(pay.headline, t)),
    tone: "none",
  };
}

function benefitsState(o: WorkerOverview, t: TranslateFn): StripState {
  const benefits = o.benefits;
  if (!benefits) return NOTHING;
  if (benefits.coveredCount > 0) {
    return {
      state: joined(
        t("{0, plural, one {# plan covered} other {# plans covered}}", benefits.coveredCount),
        benefits.notStartedCount > 0 ? t("{0} not started", benefits.notStartedCount) : null,
      ),
      tone: "none",
    };
  }
  if (benefits.notStartedCount > 0) {
    return {
      state: t(
        "{0, plural, one {# plan not started} other {# plans not started}}",
        benefits.notStartedCount,
      ),
      tone: "warning",
    };
  }
  return { state: t("No plans covered"), tone: "none" };
}

function documentsState(o: WorkerOverview, t: TranslateFn): StripState {
  if (o.documentCount == null) return NOTHING;
  return {
    state: o.documentCount === 0 ? t("None on file") : t("{0} on file", o.documentCount),
    tone: "none",
  };
}

function portalState(o: WorkerOverview, t: TranslateFn): StripState {
  const portal = o.portal;
  if (!portal) return NOTHING;
  const access =
    portal.access === "Linked"
      ? t("Linked")
      : portal.access === "Invited"
        ? t("Invited")
        : t("Not set up");
  const waiting = portal.pendingProfileChangeCount ?? 0;
  if (waiting > 0) {
    return {
      state: joined(
        access,
        t("{0, plural, one {# change waiting} other {# changes waiting}}", waiting),
      ),
      tone: "warning",
    };
  }
  return { state: access, tone: "none" };
}

function memoryState(o: WorkerOverview, t: TranslateFn): StripState {
  if (o.agentMemoryCount == null) return NOTHING;
  return {
    state:
      o.agentMemoryCount === 0
        ? t("No notes")
        : t("{0, plural, one {# note} other {# notes}}", o.agentMemoryCount),
    tone: "none",
  };
}

const STATE_OF: Record<
  WorkerView,
  (o: WorkerOverview, t: TranslateFn, drawable: boolean) => StripState
> = {
  history: historyState,
  checklists: checklistsState,
  credentials: credentialsState,
  "dq-file": dqFileState,
  "drug-and-alcohol": drugAlcoholState,
  training: trainingState,
  reviews: reviewsState,
  "hours-of-service": hoursState,
  schedule: scheduleState,
  "time-off": timeOffState,
  leave: leaveState,
  "pay-profile": payState,
  benefits: benefitsState,
  documents: documentsState,
  "driver-portal": portalState,
  "agent-memory": memoryState,
};

/**
 * One view's state line from the overview read; nothing before the read answers. `drawable` says
 * the plan and the reader's permissions draw the view (`useWorkerAreas`), which lets a null
 * section that cannot be withheld there read as "none".
 */
export function workerStripState(
  view: WorkerView,
  overview: WorkerOverview | undefined,
  t: TranslateFn,
  { drawable = false }: { drawable?: boolean } = {},
): StripState {
  return overview ? STATE_OF[view](overview, t, drawable) : NOTHING;
}

const VIEW_ICONS: Record<WorkerView, IconComponent> = {
  history: ClockRewindIcon,
  checklists: ClipboardCheckIcon,
  credentials: IdCardIcon,
  "dq-file": FolderCheckIcon,
  "drug-and-alcohol": Beaker02Icon,
  training: GraduationHat01Icon,
  reviews: Star01Icon,
  "hours-of-service": Speedometer03Icon,
  schedule: CalendarClockIcon,
  "time-off": SunsetIcon,
  leave: CalendarRangeIcon,
  "pay-profile": Wallet02Icon,
  benefits: HeartHandIcon,
  documents: File06Icon,
  "driver-portal": Phone01Icon,
  "agent-memory": BrainIcon,
};

export function workerViewLabel(view: WorkerView, t: TranslateFn): string {
  switch (view) {
    case "history":
      return t("History");
    case "checklists":
      return t("Checklists");
    case "credentials":
      return t("Credentials");
    case "dq-file":
      return t("DQ file");
    case "drug-and-alcohol":
      return t("Drug and alcohol");
    case "training":
      return t("Training");
    case "reviews":
      return t("Reviews");
    case "hours-of-service":
      return t("Hours of service");
    case "schedule":
      return t("Schedule");
    case "time-off":
      return t("Time off");
    case "leave":
      return t("Leave");
    case "pay-profile":
      return t("Pay profile");
    case "benefits":
      return t("Benefits");
    case "documents":
      return t("Documents");
    case "driver-portal":
      return t("Driver portal");
    case "agent-memory":
      return t("Agent memory");
  }
}

export function workerTabLabel(tab: WorkerTab, t: TranslateFn): string {
  switch (tab) {
    case "overview":
      return t("Overview");
    case "profile":
      return t("Profile");
    case "employment":
      return t("Employment");
    case "qualification":
      return t("Qualification");
    case "development":
      return t("Development");
    case "safety":
      return t("Safety");
    case "time":
      return t("Time");
    case "pay":
      return t("Pay");
    case "files":
      return t("Files");
    case "activity":
      return t("Activity");
  }
}

/**
 * The cells of a tab's section strip: each view with its icon and state line. `views` are the
 * tab's drawable views (`useWorkerAreas().viewsOf`).
 */
export function workerStripItems(
  views: readonly WorkerView[],
  overview: WorkerOverview | undefined,
  t: TranslateFn,
): SectionStripItem<WorkerView>[] {
  return views.map((view) => {
    const { state, tone } = workerStripState(view, overview, t, { drawable: true });
    return { value: view, label: workerViewLabel(view, t), icon: VIEW_ICONS[view], state, tone };
  });
}

/**
 * How long the person has worked here, in the words the header uses: "4 years 2 months in",
 * "Starts Dec 1, 2026" for a hire still ahead, "Left Mar 3, 2026" once terminated.
 */
export function workerTenure(
  {
    hireDate,
    terminationDate,
  }: { hireDate: number | null | undefined; terminationDate: number | null | undefined },
  nowSeconds: number,
  t: TranslateFn,
): string | null {
  if (terminationDate) return t("Left {0}", formatUnixDateMedium(terminationDate));
  if (!hireDate) return null;
  if (hireDate > nowSeconds) return t("Starts {0}", formatUnixDateMedium(hireDate));
  // Counted on the reader's calendar, the zone a date-only field is entered and shown in.
  const start = toUserWallClock(hireDate) ?? new Date(hireDate * 1000);
  const now = toUserWallClock(nowSeconds) ?? new Date(nowSeconds * 1000);
  let months = (now.getFullYear() - start.getFullYear()) * 12 + now.getMonth() - start.getMonth();
  if (now.getDate() < start.getDate()) months -= 1;
  const years = Math.floor(months / 12);
  const rest = months % 12;
  if (years === 0 && rest === 0) return t("Started {0}", formatUnixDateMedium(hireDate));
  if (years === 0) return t("{0, plural, one {# month in} other {# months in}}", rest);
  if (rest === 0) return t("{0, plural, one {# year in} other {# years in}}", years);
  return t(
    "{0, plural, one {# year} other {# years}} {1, plural, one {# month in} other {# months in}}",
    years,
    rest,
  );
}
