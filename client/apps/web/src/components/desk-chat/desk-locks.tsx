import { apiService } from "@/services/api";
import type { AssistantThread, ThreadBudget } from "@/types/assistant";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixMonthDay, formatUnixTime } from "@trenova/shared/lib/date";
import { useCallback, type ReactNode } from "react";
import { toast } from "sonner";
import { DeskErrorLink } from "./desk-error-card";
import { DeskIcon } from "./desk-icons";
import { useRichT } from "@trenova/shared/i18n/rich";

/** What the composer shows instead of its text box, and the quiet line under it. */
export type DeskComposerLock = { lock: ReactNode; note?: ReactNode };

type RequestKind = "access" | "allowance" | "budget" | "daily_runs";

/**
 * Asks the people who run AI Control for what the person ran out of, and
 * says what happened: how many were asked, or that nobody can be.
 */
export function useRequestMore(threadId: string) {
  const t = useT();
  return useCallback(
    async (kind: RequestKind) => {
      try {
        const { sent } = await apiService.assistantService.requestMore(threadId, kind);
        if (sent > 0) {
          toast.success(t("{0, plural, one {Asked # admin} other {Asked # admins}}", sent), {
            description: t("They'll see it in their notifications."),
          });
        } else {
          toast.info(t("Your request was already sent today"), {
            description: t("Nobody new was asked. An admin can change limits in AI Control."),
          });
        }
      } catch {
        toast.error(t("The request didn't go through"), {
          description: t("Try again in a moment."),
        });
      }
    },
    [t, threadId],
  );
}

function monthName(at: number): string {
  return new Date(at * 1000).toLocaleDateString(undefined, { month: "long", timeZone: "UTC" });
}

function resetsAtTime(at: number, timezone: string, t: TranslateFn): string {
  const time = formatUnixTime(at, { timezone });
  const midnight = new Date(at * 1000).toLocaleTimeString("en-US", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
    timeZone: timezone,
  });
  return midnight === "00:00" || midnight === "24:00" ? t("midnight") : time;
}

function daysUntil(at: number): number {
  return Math.max(1, Math.ceil((at * 1000 - Date.now()) / 86_400_000));
}

/**
 * Why the composer cannot send, in the order a person can do something about
 * it: the conversation can no longer continue, no model is set up, their own
 * allowance is spent, then the agent's budget or daily limit. Each says which
 * limit, when it lifts, and the one thing to do about it.
 */
export function deskComposerLock({
  thread,
  agentName,
  budget,
  noModel,
  timezone,
  t,
  requestMore,
  startElsewhere,
  openAgentControl,
}: {
  thread: Pick<AssistantThread, "canContinue" | "cannotContinueReason">;
  agentName: string;
  budget: ThreadBudget | undefined;
  noModel: boolean;
  timezone: string;
  t: TranslateFn;
  requestMore: (kind: RequestKind) => void;
  startElsewhere: () => void;
  openAgentControl: () => void;
}): DeskComposerLock | null {
  const link = (label: string, onClick: () => void) => (
    <DeskErrorLink className="in-[.dk-ec-offmsg]:mt-px" onClick={onClick}>
      {label}
    </DeskErrorLink>
  );

  if (thread.canContinue === false) {
    switch (thread.cannotContinueReason) {
      case "NoAccess":
        return {
          lock: (
            <>
              <DeskIcon name="lock" size={13} stroke={2} />
              <span>
                <b>{t("You no longer have access to {0}.", agentName)}</b>{" "}
                {t("You can still read this conversation.")}
              </span>
              {link(t("Request access"), () => requestMore("access"))}
            </>
          ),
        };
      case "AgentDeleted":
        return {
          lock: (
            <>
              <DeskIcon name="lock" size={13} stroke={2} />
              <span>
                <b>{t("{0} was removed.", agentName)}</b>{" "}
                {t("You can still read this conversation.")}
              </span>
              {link(t("Start with another agent"), startElsewhere)}
            </>
          ),
        };
      case "AgentNotConversational":
        return {
          lock: (
            <>
              <DeskIcon name="lock" size={13} stroke={2} />
              <span>
                <b>{t("{0} now only runs on its own.", agentName)}</b>{" "}
                {t("You can still read this conversation.")}
              </span>
              {link(t("Start with another agent"), startElsewhere)}
            </>
          ),
        };
      default: {
        const when = budget?.disabledAt ? formatUnixMonthDay(budget.disabledAt, { timezone }) : "";
        const who = budget?.disabledBy ?? "";
        const headline =
          who !== "" && when !== ""
            ? t("{0} turned off {1} on {2}.", who, agentName, when)
            : when !== ""
              ? t("{0} was turned off on {1}.", agentName, when)
              : t("{0} has been turned off.", agentName);
        return {
          lock: (
            <>
              <DeskIcon name="lock" size={13} stroke={2} />
              <span>
                <b>{headline}</b> {t("You can still read this conversation.")}
              </span>
              {link(t("Start with another agent"), startElsewhere)}
            </>
          ),
        };
      }
    }
  }

  if (noModel) {
    return {
      lock: (
        <>
          <DeskIcon name="info" size={13} stroke={2} />
          <span>
            <b>{t("No AI model is set up for your organization yet.")}</b>{" "}
            {t("An admin can connect one in Agent Control.")}
          </span>
          {link(t("Open Agent Control"), openAgentControl)}
        </>
      ),
    };
  }

  if (!budget) {
    return null;
  }

  const person = budget.person;
  if (person && person.limit > 0 && person.used >= person.limit) {
    const days = daysUntil(person.resetsAt);
    return {
      lock: (
        <>
          <DeskIcon name="lock" size={13} stroke={2} />
          <span>
            <b>{t("You've used your AI allowance for this period.")}</b>{" "}
            {t("{0, plural, one {It refreshes in # day.} other {It refreshes in # days.}}", days)}
          </span>
          {link(t("Request more"), () => requestMore("allowance"))}
        </>
      ),
      note: t(
        "{0} of {1} messages · resets {2}",
        person.used.toLocaleString(),
        person.limit.toLocaleString(),
        formatUnixMonthDay(person.resetsAt, { timezone: "UTC" }),
      ),
    };
  }

  if (budget.budgetUsed) {
    return {
      lock: (
        <>
          <DeskIcon name="alert" size={13} stroke={2} />
          <span>
            <b>{t("{0}'s budget for {1} is used up.", monthName(budget.monthStart), agentName)}</b>{" "}
            {t(
              "It resets {0}, or an admin can raise it.",
              formatUnixMonthDay(budget.resetsAt, { timezone: "UTC" }),
            )}
          </span>
          {link(t("Ask an admin"), () => requestMore("budget"))}
        </>
      ),
    };
  }

  if (budget.dailyUsed) {
    return {
      lock: (
        <>
          <DeskIcon name="lock" size={13} stroke={2} />
          <span>
            <b>
              {t(
                "{0} has reached today's {1}-request limit.",
                agentName,
                budget.dailyRunLimit.toLocaleString(),
              )}
            </b>{" "}
            {t("It resets at {0}.", resetsAtTime(budget.dayResetsAt, timezone, t))}
          </span>
          {link(t("Ask another agent"), startElsewhere)}
        </>
      ),
    };
  }

  return null;
}

/**
 * The budget the agent has nearly spent, above the composer: the share in
 * words, the dollars against the budget, and a bar. Shown from nine tenths
 * on and gone once the budget is spent, when the composer itself says so.
 * The person's own allowance gets the same meter from eight tenths.
 */
export function DeskUsageMeter({ budget }: { budget: ThreadBudget }) {
  const rt = useRichT();
  const person = budget.person;
  if (person && person.limit > 0) {
    const share = person.used / person.limit;
    if (share >= 0.8 && share < 1) {
      return (
        <Meter
          share={share}
          figure={`${person.used.toLocaleString()} / ${person.limit.toLocaleString()}`}
        >
          {rt(
            "You've used <b>{0}%</b> of this month's AI allowance",
            { b: (c) => <b>{c}</b> },
            Math.floor(share * 100),
          )}
        </Meter>
      );
    }
  }
  if (budget.near && !budget.budgetUsed && budget.limitUsd !== "") {
    return (
      <Meter
        share={budget.share}
        figure={`${dollars(budget.spentUsd)} / ${dollars(budget.limitUsd)}`}
      >
        {rt(
          "{0} has used <b>{1}%</b> of {2}'s budget",
          { b: (c) => <b>{c}</b> },
          budget.agentName,
          Math.floor(budget.share * 100),
          monthName(budget.monthStart),
        )}
      </Meter>
    );
  }

  return null;
}

function dollars(value: string): string {
  const amount = Number(value);
  return Number.isFinite(amount)
    ? amount.toLocaleString("en-US", {
        style: "currency",
        currency: "USD",
        maximumFractionDigits: amount >= 100 ? 0 : 2,
      })
    : value;
}

function Meter({
  share,
  figure,
  children,
}: {
  share: number;
  figure: string;
  children: ReactNode;
}) {
  return (
    <div className="dk-ec-meter" role="status">
      <div className="dk-ec-mt">
        <span>{children}</span>
        <span className="dk-ax-num">{figure}</span>
      </div>
      <div className="dk-ec-mb">
        <i style={{ width: `${Math.min(100, Math.max(0, share * 100))}%` }} />
      </div>
    </div>
  );
}
