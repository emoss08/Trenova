import type { TurnLimit } from "@/components/assistant/turn-stream";
import type { ThreadBudget } from "@/types/assistant";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort, formatUnixMonthDay } from "@trenova/shared/lib/date";
import type { ReactNode } from "react";
import { DeskErrorButton, DeskErrorCard } from "./desk-error-card";
import { DeskIcon } from "./desk-icons";

/** A person's allowance is worth a warning once this share of it is used. */
const allowanceNearShare = 0.8;

/**
 * When a cap lifts. Monthly caps roll over on the first in UTC, so the date
 * is read in UTC lest the evening before show in the Americas; the daily
 * run cap lifts at an hour, shown in the reader's own timezone.
 */
function resetLabel(kind: TurnLimit["kind"], at: number, timezone: string): string {
  if (at <= 0) {
    return "";
  }
  return kind === "daily_runs"
    ? formatUnixDateTimeShort(at, { timezone })
    : formatUnixMonthDay(at, { timezone: "UTC" });
}

function limitCopy(limit: TurnLimit, reset: string, t: TranslateFn) {
  switch (limit.kind) {
    case "person_allowance":
      return {
        title: t("You've used this month's AI allowance"),
        sub:
          reset !== ""
            ? t("{0} of {1} questions asked. It refreshes on {2}.", limit.used, limit.limit, reset)
            : t("{0} of {1} questions asked.", limit.used, limit.limit),
      };
    case "daily_runs":
      return {
        title: t("This agent has reached today's limit"),
        sub:
          reset !== ""
            ? t(
                "It has answered {0} questions today. It can answer again from {1}.",
                limit.limit,
                reset,
              )
            : t("It has answered {0} questions today.", limit.limit),
      };
    default:
      return {
        title: t("This agent has spent its monthly budget"),
        sub:
          reset !== ""
            ? t("${0} of ${1} used. It answers again on {2}.", limit.used, limit.limit, reset)
            : t("${0} of ${1} used.", limit.used, limit.limit),
      };
  }
}

/**
 * The card for a question a usage cap stopped. Nothing failed and trying
 * again would not help, so it is a boundary, not an error: it says which cap,
 * when it lifts and who can raise it, and offers only to put the words back.
 */
export function DeskLimitCard({
  limit,
  timezone,
  onDismiss,
}: {
  limit: TurnLimit;
  timezone: string;
  onDismiss: () => void;
}) {
  const t = useT();
  const copy = limitCopy(limit, resetLabel(limit.kind, limit.resetsAt, timezone), t);

  return (
    <DeskErrorCard
      tone="warn"
      icon="lock"
      compact
      title={copy.title}
      sub={
        <>
          {copy.sub} {t("An administrator can raise the limit in AI Control.")}
        </>
      }
      actions={<DeskErrorButton onClick={onDismiss}>{t("Dismiss")}</DeskErrorButton>}
    />
  );
}

/**
 * A line above the composer when a cap is close, so the person learns before
 * a question is refused rather than after. The person's own allowance is
 * named first: it is the one their next question spends.
 */
export function DeskLimitNote({ budget, timezone }: { budget: ThreadBudget; timezone: string }) {
  const t = useT();
  const person = budget.person;

  if (person && person.limit > 0) {
    const share = person.used / person.limit;
    if (share >= allowanceNearShare && person.used < person.limit) {
      const reset = resetLabel("person_allowance", person.resetsAt, timezone);
      return (
        <LimitLine share={share}>
          {t(
            "{0, plural, one {# question} other {# questions}} left in your allowance this month",
            person.limit - person.used,
          )}
          {reset !== "" && <> · {t("refreshes {0}", reset)}</>}
        </LimitLine>
      );
    }
  }

  if (budget.near && budget.share < 1) {
    const reset = resetLabel("monthly_budget", budget.resetsAt, timezone);
    return (
      <LimitLine share={budget.share}>
        {t(
          "{0} has used {1}% of its monthly budget",
          budget.agentName,
          Math.floor(budget.share * 100),
        )}
        {reset !== "" && <> · {t("resets {0}", reset)}</>}
      </LimitLine>
    );
  }

  return null;
}

function LimitLine({ share, children }: { share: number; children: ReactNode }) {
  const width = `${Math.min(100, Math.max(0, share * 100))}%`;
  return (
    <div className="dk-limit" role="status">
      <span className="dk-limit-ic">
        <DeskIcon name="info" size={13} stroke={2} />
      </span>
      <span className="dk-limit-t">{children}</span>
      <span className="dk-limit-bar" aria-hidden>
        <i style={{ width }} />
      </span>
    </div>
  );
}
