import { useDeskSetting } from "@/stores/desk-settings-store";
import {
  DeskGreeting,
  DeskHomeComposer,
  useDeskHomeAsk,
  type DeskStartExtras,
} from "@/components/desk-chat/desk-home-ask";
import { DeskDropOverlay } from "@/components/desk-chat/composer/desk-uploads";
import { useAttentionSummary } from "@/hooks/use-attention";
import { usePermission } from "@/hooks/use-permission";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { briefing as briefingQueries } from "@/lib/queries/briefing";
import type { AssistantThread } from "@/types/assistant";
import { formatList } from "@trenova/shared/i18n/format";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  partOfDay,
  resolveUserTimezone,
  toUserWallClock,
  type PartOfDay,
} from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { BriefingSectionKey } from "@trenova/graphql/generated/graphql";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { usePendingDecisionSummary } from "./decisions/use-pending-decisions";
import { DeskToday } from "./today/desk-today";

export type { DeskStartExtras } from "@/components/desk-chat/desk-home-ask";

const nowInSeconds = () => Math.floor(Date.now() / 1000);

/** Each briefing section as the page titles it, in the language on screen. */
const BRIEFING_SECTION_LABELS: Record<BriefingSectionKey, string> = defineLabels({
  Attention: "Needs attention",
  Today: "Today",
  Coverage: "The next day",
  Exceptions: "Critical",
  Decisions: "Waiting on you",
  Compliance: "Credentials",
  Billing: "Billing queue",
  Cash: "Cash received yesterday",
});

export type DeskHomeProps = {
  agents: AgentChoice[];
  threads: AssistantThread[];
  isLoading: boolean;
  isStarting: boolean;
  onStart: (agentId: string, question?: string, extras?: DeskStartExtras) => void;
};

function greeting(t: TranslateFn, dayPart: PartOfDay, firstName: string): string {
  switch (dayPart) {
    case "morning":
      return firstName ? t("Good morning, {0}", firstName) : t("Good morning");
    case "afternoon":
      return firstName ? t("Good afternoon, {0}", firstName) : t("Good afternoon");
    default:
      return firstName ? t("Good evening, {0}", firstName) : t("Good evening");
  }
}

/**
 * What a briefing cut to the reader's access says about the cut: the sections
 * it leaves out, and those it shows only the part of they may open.
 */
function withheldNote(
  t: TranslateFn,
  withheld: readonly { key: BriefingSectionKey; partial: boolean }[],
): string {
  const names = (partial: boolean) =>
    formatList(
      withheld
        .filter((section) => section.partial === partial)
        .map((section) => BRIEFING_SECTION_LABELS[section.key]),
    );
  const left = withheld.some((section) => !section.partial);
  const partial = withheld.some((section) => section.partial);
  if (left && partial) {
    return t(
      "This morning's briefing leaves out {0} and shows only the part of {1} you can open.",
      names(false),
      names(true),
    );
  }
  if (partial) {
    return t("This morning's briefing shows only the part of {0} you can open.", names(true));
  }
  return t("This morning's briefing leaves out {0}, whose records you can't open.", names(false));
}

/**
 * The Desk's front page: the date, a greeting, one line on what the day holds,
 * and the box you talk into. When the line
 * is the count of decisions waiting, it is a link to them.
 *
 * Under the box the page tells before it asks: the day in figures from this
 * morning's briefing, what needs the reader soonest, what changed since they
 * last looked, what agents settled today and what they have in hand.
 *
 * The line under the greeting is the morning briefing's headline when one was
 * written — every figure in it was checked against the facts it was written
 * from — and otherwise a sentence composed from counts, never from a model.
 * A briefing cut to the sections the reader may open carries no headline, since
 * it was written about the whole page; a quiet line under the greeting names
 * the sections left out instead.
 * The composer's placeholder types out the chosen agent's own starter
 * questions; Tab takes the one on screen and Cmd/Ctrl+1–3 asks it outright.
 */
export function DeskHome({ agents, threads, isLoading, isStarting, onStart }: DeskHomeProps) {
  const t = useT();
  const [now] = useState(nowInSeconds);
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const { allowed: canDecide, isLoading: permissionsLoading } = usePermission(
    Resource.AgentProposal,
    Operation.Read,
  );
  const { data: attention } = useAttentionSummary();
  const { allowed: canWatch } = usePermission(Resource.Watchtower, Operation.Read);
  const briefingQuery = useQuery({ ...briefingQueries.today(), retry: false });
  const briefing = briefingQuery.data ?? null;
  const summaryQuery = usePendingDecisionSummary(canDecide);
  const waiting = summaryQuery.data?.total ?? attention?.agentDecisions ?? 0;
  const preferredAgent = useDeskSetting("agent");
  const ask = useDeskHomeAsk({ agents, threads, isLoading, preferId: preferredAgent, onStart });
  const { noAgents, drag } = ask;

  const dateline = useMemo(
    () =>
      new Intl.DateTimeFormat(undefined, {
        weekday: "long",
        month: "long",
        day: "numeric",
        timeZone: timezone,
      }).format(new Date(now * 1000)),
    [now, timezone],
  );
  const hour = toUserWallClock(now, timezone)?.getHours() ?? 9;
  const firstName = user?.name?.trim().split(/\s+/)[0] ?? "";

  // The decisions sentence is the way to them, so it opens Decisions.
  const headline =
    briefing?.headline ||
    (canDecide && waiting > 0 ? (
      <Link to="/desk/decisions" className="dk-headline-link">
        {t(
          "{0, plural, one {One decision is waiting on you.} other {# decisions are waiting on you.}}",
          waiting,
        )}
      </Link>
    ) : noAgents ? (
      t("Nothing is running here yet.")
    ) : (
      t("How can I help today?")
    ));
  const withheld = briefing?.withheldSections ?? [];
  const headlinePending =
    isLoading ||
    permissionsLoading ||
    briefingQuery.isPending ||
    (canDecide && summaryQuery.isPending);

  return (
    <div className="dk-home-w">
      <DeskDropOverlay show={drag.on} hot={drag.hot} count={drag.count} />
      <div className="dk-home dk-home-day dk-w-today">
        <div className="dk-home-in">
          <DeskGreeting
            date={dateline}
            title={greeting(t, partOfDay(hour), firstName)}
            line={headline}
            pending={headlinePending}
          />
          {!headlinePending && withheld.length > 0 && (
            <p className="dk-headline-note" data-slot="desk-briefing-withheld">
              {withheldNote(t, withheld)}
            </p>
          )}
          {!headlinePending && withheld.length === 0 && briefing?.headline && (
            <p className="dk-w-hl-note" data-slot="desk-headline-source">
              {briefing.narrated && <NovaBadge size="xs" />}
              {briefing.narrated
                ? t("Written by Nova from this morning's figures; every number was checked against them.")
                : t("Counted from this morning's figures.")}
            </p>
          )}
          <DeskHomeComposer ask={ask} isStarting={isStarting} />
          <DeskToday
            briefing={briefing}
            canWatch={canWatch}
            canDecide={canDecide}
            timezone={timezone}
          />
        </div>
      </div>
    </div>
  );
}
