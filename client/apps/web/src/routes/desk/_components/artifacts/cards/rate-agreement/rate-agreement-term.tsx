import { agreementWindow } from "@/lib/rate-agreement";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { RateAgreement } from "@trenova/shared/types/rate";
import { CardPath } from "../../record-card/card-path";
import { cardDate } from "../../record-card/record-card-values";
import {
  lifecycleSteps,
  termSpan,
  termState,
  type LifecycleStep,
  type TermState,
} from "./rate-agreement-card";

/** Where today sits on the track of a term with no end: well along, with the line running on past it. */
const OPEN_TODAY = 0.72;

type TermAgreement = Pick<
  RateAgreement,
  "status" | "effectiveFrom" | "effectiveTo" | "autoRenew" | "renewalNoticeDays"
>;

/** What the term's first chip says: how long it runs, or why it prices nothing. */
function termLine(agreement: TermAgreement, state: TermState, now: number, t: TranslateFn): string {
  switch (state) {
    case "held":
      return t("Prices nothing while suspended");
    case "ended":
      return agreement.status === "Active"
        ? agreementWindow(agreement, now, t).text
        : t("No longer prices freight");
    case "pending":
      return agreement.status === "Active"
        ? agreementWindow(agreement, now, t).text
        : t("Prices freight once approved");
    default:
      return agreementWindow(agreement, now, t).text;
  }
}

/**
 * The agreement's term as the card leads with it: its start and end set large
 * at either end, a track filled as far as today in the colour of where the
 * term stands, today marked on it, and the renewal notice hatched at its end.
 * A term with no end runs its line on past today.
 */
export function TermPanel({ agreement, now }: { agreement: TermAgreement; now: number }) {
  const t = useT();
  const state = termState(agreement, now);
  const span = termSpan(agreement, now);
  const end = agreement.effectiveTo ?? null;
  const begun = agreement.effectiveFrom <= now;
  const today = span?.today ?? (begun ? OPEN_TODAY : 0);
  const showsToday = span === null ? begun : span.when === "during";

  return (
    <div className={cn("dk-ra-now", `dk-ra-${state}`)}>
      <div className="dk-ra-ends">
        <div className="dk-ra-end">
          <span>{agreement.effectiveFrom > now ? t("Starts") : t("Started")}</span>
          <b>{cardDate(agreement.effectiveFrom)}</b>
        </div>
        <div className="dk-ra-end dk-to">
          <span>{end === null ? t("Ends") : end < now ? t("Ended") : t("Ends")}</span>
          <b>{end === null ? t("No end date") : cardDate(end)}</b>
        </div>
      </div>
      <div
        className={cn("dk-ra-track", span === null && "dk-open")}
        {...(span
          ? {
              role: "meter",
              "aria-label": t("Term elapsed"),
              "aria-valuemin": 0,
              "aria-valuemax": 100,
              "aria-valuenow": Math.round(span.today * 100),
            }
          : { "aria-hidden": true })}
      >
        <i className="dk-ra-done" style={{ width: `${today * 100}%` }} />
        {span && span.notice > 0 && (
          <i
            className={cn("dk-ra-notice-zone", agreement.autoRenew && "dk-renews")}
            style={{ width: `${span.notice * 100}%` }}
          />
        )}
        {showsToday && (
          <span className="dk-ra-today" style={{ left: `${today * 100}%` }}>
            <em>{t("Today")}</em>
          </span>
        )}
      </div>
      <div className="dk-ra-chips">
        <span className="dk-ra-chip dk-lead">
          <i aria-hidden />
          {termLine(agreement, state, now, t)}
        </span>
        {end === null ? (
          <span className="dk-ra-chip">{t("Runs on")}</span>
        ) : agreement.autoRenew ? (
          <span className="dk-ra-chip">{t("Renews by itself")}</span>
        ) : (
          <span className="dk-ra-chip">
            {t(
              "{0, plural, one {# day's notice} other {# days' notice}}",
              agreement.renewalNoticeDays,
            )}
          </span>
        )}
      </div>
    </div>
  );
}

/** What a step says under its name: when and by whom, or what it is waiting on. */
function stepLine(
  step: LifecycleStep,
  status: RateAgreement["status"],
  users: ReadonlyMap<string, string>,
  t: TranslateFn,
): string {
  if (step.at !== null) {
    const who = step.byId ? users.get(step.byId) : undefined;
    return [cardDate(step.at), who].filter(Boolean).join(" · ");
  }
  if (step.key === "ended") {
    return status === "Suspended" ? t("Held until it is resumed") : t("Closed");
  }
  if (step.key === "approved" && status === "InReview") {
    return t("Waiting on a reviewer");
  }
  return t("Not yet");
}

/**
 * The way the agreement came to price freight, down a rail: drafted,
 * submitted for review, approved, and how it ended, with a reviewer's comment
 * on the step it was left at.
 */
export function ApprovalPath({
  agreement,
  users,
}: {
  agreement: Pick<
    RateAgreement,
    | "status"
    | "createdAt"
    | "submittedAt"
    | "submittedById"
    | "approvedAt"
    | "approvedById"
    | "reviewComment"
  >;
  users: ReadonlyMap<string, string>;
}) {
  const t = useT();
  const steps = lifecycleSteps(agreement, t);
  const noteAt = steps.find((step) => step.state === "now")?.key ?? steps.at(-1)?.key;

  return (
    <CardPath
      label={t("Approval")}
      steps={steps.map((step) => ({
        key: step.key,
        label: step.label,
        line: stepLine(step, agreement.status, users, t),
        state: step.state,
        tone: step.tone,
        note: step.key === noteAt ? agreement.reviewComment : undefined,
      }))}
    />
  );
}
