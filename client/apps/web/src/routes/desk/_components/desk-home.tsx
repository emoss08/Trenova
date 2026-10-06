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
import { queries } from "@/lib/queries";
import type { AssistantThread } from "@/types/assistant";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  partOfDay,
  resolveUserTimezone,
  toUserWallClock,
  type PartOfDay,
} from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { usePendingDecisionSummary } from "./decisions/use-pending-decisions";

export type { DeskStartExtras } from "@/components/desk-chat/desk-home-ask";

const nowInSeconds = () => Math.floor(Date.now() / 1000);

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
 * The Desk's front page: the date, a greeting, one line on what the day holds,
 * and the box you talk into, centred in the room and nothing else.
 *
 * The line under the greeting is the morning briefing's headline when one was
 * written — every figure in it was checked against the facts it was written
 * from — and otherwise a sentence composed from counts, never from a model.
 * The composer's placeholder types out the chosen agent's own starter
 * questions; Tab takes the one on screen and ⌘1–3 asks it outright.
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
  const briefingQuery = useQuery({ ...queries.briefing.today(), retry: false });
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

  const headline =
    briefing?.headline ||
    (canDecide && waiting > 0
      ? t(
          "{0, plural, one {One decision is waiting on you.} other {# decisions are waiting on you.}}",
          waiting,
        )
      : noAgents
        ? t("Nothing is running here yet.")
        : t("How can I help today?"));
  const headlinePending =
    isLoading ||
    permissionsLoading ||
    briefingQuery.isPending ||
    (canDecide && summaryQuery.isPending);

  return (
    <div className="dk-home-w">
      <DeskDropOverlay show={drag.on} hot={drag.hot} count={drag.count} />
      <div className="dk-home">
        <div className="dk-home-in">
          <DeskGreeting
            date={dateline}
            title={greeting(t, partOfDay(hour), firstName)}
            line={headline}
            pending={headlinePending}
          />
          <DeskHomeComposer ask={ask} isStarting={isStarting} />
        </div>
      </div>
    </div>
  );
}
