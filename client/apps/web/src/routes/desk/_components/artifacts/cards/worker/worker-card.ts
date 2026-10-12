import type {
  DeskWorker,
  DeskWorkerCredential,
  DeskWorkerHos,
  DeskWorkerStanding,
} from "@/lib/graphql/worker-card";
import { choiceLabel, endorsementTypeChoices } from "@/lib/choices";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { worstFirst, type DueCheck } from "../../record-card/card-due";
import type { CardCheckState, CardFact, CardTone } from "../../record-card/record-card";
import { cardDate, dueReading, personName } from "../../record-card/record-card-values";

const HOUR_MS = 3_600_000;
/** A CDL renewing inside this many days is worth a look. */
export const LICENSE_NOTICE_DAYS = 60;
/** A medical card, hazmat or TWIC expiring, or a physical or MVR falling due, inside this many days. */
export const QUALIFICATION_NOTICE_DAYS = 30;

/** The worker by name; "Unnamed worker" when none is on file. */
export function workerName(worker: Pick<DeskWorker, "firstName" | "lastName">, t: TranslateFn) {
  return personName(worker) || t("Unnamed worker");
}

/** Whether the worker can be put on a load now, as the card's pill says it. */
export type DispatchStanding = {
  label: string;
  tone: CardTone;
  mark: "dot" | "check" | "pause";
  /** Why they cannot, when the record says. */
  reason: string;
};

export function dispatchStanding(worker: DeskWorker, t: TranslateFn): DispatchStanding {
  if (worker.status !== "Active") {
    return { label: t("Inactive"), tone: "muted", mark: "dot", reason: "" };
  }
  const blocked = worker.assignmentBlocked.trim();
  if (blocked) {
    return { label: t("Blocked from dispatch"), tone: "warn", mark: "pause", reason: blocked };
  }
  if (worker.profile && !worker.profile.isQualified) {
    return {
      label: t("Not qualified"),
      tone: "warn",
      mark: "pause",
      reason: worker.profile.disqualificationReason.trim(),
    };
  }
  if (!worker.canBeAssigned || !worker.availableForDispatch) {
    return { label: t("Unavailable"), tone: "warn", mark: "pause", reason: "" };
  }
  return { label: t("Available for dispatch"), tone: "success", mark: "dot", reason: "" };
}

/** The worker's license as a dispatcher reads it: "CDL A · TX · Hazmat (H) · Restrictions L". */
export function licenseLine(profile: NonNullable<DeskWorker["profile"]>, t: TranslateFn): string {
  return [
    t("CDL {0}", profile.cdlClass),
    profile.licenseState?.abbreviation ?? "",
    profile.endorsement === "O"
      ? t("No endorsements")
      : choiceLabel(endorsementTypeChoices, profile.endorsement),
    profile.cdlRestrictions.trim() ? t("Restrictions {0}", profile.cdlRestrictions.trim()) : "",
  ]
    .filter(Boolean)
    .join(" · ");
}

/** Whether the worker's endorsement lets them haul hazardous materials, so the hazmat date counts. */
export function haulsHazmat(endorsement: string): boolean {
  return endorsement === "H" || endorsement === "X";
}

function datedCheck({
  key,
  at,
  now,
  notice,
  required,
  t,
  names,
}: {
  key: string;
  at: number | null | undefined;
  now: number;
  notice: number;
  required: boolean;
  t: TranslateFn;
  names: { label: string; expired: string; soon: (days: number) => string; missing: string };
}): DueCheck | null {
  if (at === null || at === undefined || at === 0) {
    return required ? { key, state: "warn", title: names.label, detail: names.missing } : null;
  }
  const due = dueReading(at, now, notice);
  return {
    key,
    state: due.state,
    title:
      due.state === "bad"
        ? names.expired
        : due.state === "warn"
          ? names.soon(due.days)
          : names.label,
    detail: due.state === "bad" ? t("Since {0}", cardDate(at)) : t("Through {0}", cardDate(at)),
  };
}

/**
 * The dates a driver's qualification hangs on, worst first: their CDL, medical
 * card, hazmat endorsement (only for a driver endorsed for it), TWIC (only
 * when one is on file), annual physical and MVR. A driver the compliance
 * sweep has disqualified says so first.
 */
export function qualificationChecks(
  profile: NonNullable<DeskWorker["profile"]>,
  now: number,
  t: TranslateFn,
): DueCheck[] {
  const checks: (DueCheck | null)[] = [
    profile.isQualified
      ? null
      : {
          key: "qualified",
          state: "bad",
          title: t("Not qualified to drive"),
          detail: profile.disqualificationReason.trim() || t("The compliance check failed"),
        },
    datedCheck({
      key: "license",
      at: profile.licenseExpiry,
      now,
      notice: LICENSE_NOTICE_DAYS,
      required: true,
      t,
      names: {
        label: t("CDL"),
        expired: t("CDL expired"),
        soon: (days) => t("CDL expires in {0, plural, one {# day} other {# days}}", days),
        missing: t("No expiry on file"),
      },
    }),
    datedCheck({
      key: "medical",
      at: profile.medicalCardExpiry,
      now,
      notice: QUALIFICATION_NOTICE_DAYS,
      required: true,
      t,
      names: {
        label: t("Medical card"),
        expired: t("Medical card expired"),
        soon: (days) => t("Medical card expires in {0, plural, one {# day} other {# days}}", days),
        missing: t("No medical card on file"),
      },
    }),
    haulsHazmat(profile.endorsement)
      ? datedCheck({
          key: "hazmat",
          at: profile.hazmatExpiry,
          now,
          notice: QUALIFICATION_NOTICE_DAYS,
          required: true,
          t,
          names: {
            label: t("Hazmat endorsement"),
            expired: t("Hazmat endorsement expired"),
            soon: (days) =>
              t("Hazmat endorsement expires in {0, plural, one {# day} other {# days}}", days),
            missing: t("No hazmat expiry on file"),
          },
        })
      : null,
    datedCheck({
      key: "twic",
      at: profile.twicExpiry,
      now,
      notice: QUALIFICATION_NOTICE_DAYS,
      required: false,
      t,
      names: {
        label: t("TWIC"),
        expired: t("TWIC expired"),
        soon: (days) => t("TWIC expires in {0, plural, one {# day} other {# days}}", days),
        missing: "",
      },
    }),
    datedCheck({
      key: "physical",
      at: profile.physicalDueDate,
      now,
      notice: QUALIFICATION_NOTICE_DAYS,
      required: false,
      t,
      names: {
        label: t("Physical"),
        expired: t("Physical overdue"),
        soon: (days) => t("Physical due in {0, plural, one {# day} other {# days}}", days),
        missing: "",
      },
    }),
    datedCheck({
      key: "mvr",
      at: profile.mvrDueDate,
      now,
      notice: QUALIFICATION_NOTICE_DAYS,
      required: false,
      t,
      names: {
        label: t("Motor vehicle record"),
        expired: t("Motor vehicle record overdue"),
        soon: (days) =>
          t("Motor vehicle record due in {0, plural, one {# day} other {# days}}", days),
        missing: "",
      },
    }),
  ];
  return worstFirst(checks.filter((check): check is DueCheck => check !== null));
}

/**
 * The worker's credentials beyond the dates on their profile (a forklift card,
 * a company course), worst first. Those that mirror a profile date are left
 * out, since the qualifications already read them; one that is neither held
 * nor required is left out too.
 */
export function otherCredentialChecks(
  items: readonly DeskWorkerCredential[],
  t: TranslateFn,
): DueCheck[] {
  const checks: DueCheck[] = [];
  for (const item of items) {
    if (item.credentialType.profileField) continue;
    if (item.health === "Missing" && !item.required) continue;
    const expiresAt = item.credential?.expiresAt ?? null;
    const days = item.daysUntilExpiry ?? null;
    const state: CardCheckState =
      item.health === "Expired"
        ? "bad"
        : item.health === "Missing"
          ? "bad"
          : item.health === "ExpiringSoon"
            ? "warn"
            : "ok";
    checks.push({
      key: item.credentialType.id,
      state,
      title: item.credentialType.name,
      detail:
        item.health === "Missing"
          ? t("Required, not on file")
          : expiresAt
            ? item.health === "Expired"
              ? t("Expired {0}", cardDate(expiresAt))
              : item.health === "ExpiringSoon" && days !== null
                ? t("Expires in {0, plural, one {# day} other {# days}}", days)
                : t("Through {0}", cardDate(expiresAt))
            : t("Does not expire"),
    });
  }
  return worstFirst(checks);
}

/** One of the driver's clocks: how much is left of how much, and whether that is getting short. */
export type HosClock = {
  key: "drive" | "shift" | "cycle" | "break";
  label: string;
  remainingMs: number;
  limitMs: number;
  /** Left of the limit, 0 to 1. */
  share: number;
  state: CardCheckState;
};

/** The driver's four clocks, each read against its own limit; short under an hour, out at none. */
export function hosClocks(state: NonNullable<DeskWorkerHos["state"]>, t: TranslateFn): HosClock[] {
  const clock = (
    key: HosClock["key"],
    label: string,
    remainingMs: number,
    limitMs: number,
  ): HosClock => {
    const remaining = Math.max(0, remainingMs);
    const share = limitMs > 0 ? Math.min(1, remaining / limitMs) : 0;
    return {
      key,
      label,
      remainingMs: remaining,
      limitMs,
      share,
      state: remaining <= 0 ? "bad" : remaining < HOUR_MS ? "warn" : "ok",
    };
  };
  return [
    clock("drive", t("Drive"), state.driveRemainingMs, state.driveLimitMs),
    clock("shift", t("Shift"), state.shiftRemainingMs, state.shiftLimitMs),
    clock("cycle", t("Cycle"), state.cycleRemainingMs, state.cycleLimitMs),
    clock("break", t("Until a break"), state.breakRemainingMs, state.breakLimitMs),
  ];
}

/** How the worker's overall standing reads on the card. */
export function standingTone(standing: DeskWorkerStanding["standing"]): CardTone {
  switch (standing) {
    case "Good":
      return "success";
    case "Watch":
      return "progress";
    case "AtRisk":
      return "warn";
    case "Blocked":
      return "warn";
  }
}

export function standingLabel(standing: DeskWorkerStanding["standing"], t: TranslateFn): string {
  switch (standing) {
    case "Good":
      return t("In good standing");
    case "Watch":
      return t("On watch");
    case "AtRisk":
      return t("At risk");
    case "Blocked":
      return t("Blocked");
  }
}

/**
 * The concerns the card shows are those it shows nowhere else: whether the
 * worker can be dispatched is the pill at its head, and their credentials are
 * the qualifications and other credentials below.
 */
const SHOWN_ELSEWHERE = new Set([
  "not_employed",
  "not_assignable",
  "credentials_expired",
  "credentials_missing",
  "credentials_expiring",
]);

/** What stands against the worker beyond what the card shows elsewhere, kept worst first. */
export function standingConcerns(
  concerns: DeskWorkerStanding["concerns"],
): DeskWorkerStanding["concerns"] {
  return concerns.filter((concern) => !SHOWN_ELSEWHERE.has(concern.code));
}

/** A concern's severity as a check's state. */
export function concernState(
  severity: DeskWorkerStanding["concerns"][number]["severity"],
): CardCheckState {
  return severity === "Critical" ? "bad" : severity === "Warning" ? "warn" : "ok";
}

/** The facts the card lists about the worker, in the order the office reads them. */
export function workerFacts(worker: DeskWorker, t: TranslateFn): CardFact[] {
  const home = [worker.city.trim(), worker.state?.abbreviation ?? ""].filter(Boolean).join(", ");
  return [
    { key: "phone", label: t("Phone"), value: worker.phoneNumber.trim() || t("None on file") },
    { key: "email", label: t("Email"), value: worker.email.trim() || t("None on file") },
    { key: "home", label: t("Home"), value: home || t("None on file") },
    {
      key: "hired",
      label: t("Hired"),
      value: worker.profile?.hireDate ? cardDate(worker.profile.hireDate) : t("None on file"),
    },
    { key: "manager", label: t("Manager"), value: worker.manager?.name ?? t("None") },
    { key: "fleet", label: t("Fleet"), value: worker.fleetCode?.code ?? t("None") },
    {
      key: "license",
      label: t("License number"),
      value: worker.profile?.licenseNumber.trim() || t("None on file"),
    },
    { key: "external", label: t("External ID"), value: worker.externalId.trim() || t("None") },
  ];
}
