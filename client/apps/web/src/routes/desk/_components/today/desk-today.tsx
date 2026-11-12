import {
  DeskHomeComposer,
  useDeskHomeAsk,
  type DeskStartExtras,
} from "@/components/desk-chat/desk-home-ask";
import { DeskDropOverlay } from "@/components/desk-chat/composer/desk-uploads";
import { useAttentionSummary } from "@/hooks/use-attention";
import { usePermission } from "@/hooks/use-permission";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { briefing as briefingQueries } from "@/lib/queries/briefing";
import { deskActivity } from "@/lib/queries/desk-activity";
import { aiControl } from "@/lib/queries/ai-control";
import { deskAttention } from "@/lib/queries/desk-attention";
import { watchtower } from "@/lib/queries/watchtower";
import { useDeskSetting } from "@/stores/desk-settings-store";
import type { AssistantThread } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  formatUnixTime,
  getStartOfDay,
  partOfDay,
  resolveUserTimezone,
  toUserWallClock,
} from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { attentionPath } from "../attention/attention-record";
import { AttentionRow } from "../attention/attention-row";
import { usePendingDecisionSummary } from "../decisions/use-pending-decisions";
import { BriefingHeadline } from "./briefing-headline";
import { DayStrip } from "./day-strip";
import { NovaHandled } from "./nova-handled";
import { AgentsWorking, workingItems, workingRows } from "./agents-working";
import { SinceYouLeft } from "./since-you-left";
import { greeting, withheldNote } from "./today-headline";
import { TodayRowsSkeleton, TodaySection } from "./today-section";
import "./today.css";

export type { DeskStartExtras } from "@/components/desk-chat/desk-home-ask";

/** The records that need the reader soonest, as the prototype shows them under the box. */
const TOP_ROWS = 4;
/** The ledger lines read for the day; the first few show and the rest wait behind one line. */
const ACTIVITY_READ = 18;

const nowInSeconds = () => Math.floor(Date.now() / 1000);

export type DeskTodayProps = {
  agents: AgentChoice[];
  threads: AssistantThread[];
  isLoading: boolean;
  isStarting: boolean;
  onStart: (agentId: string, question?: string, extras?: DeskStartExtras) => void;
};

/**
 * The Desk's front page, as the prototype's `Today` draws it: the date, a
 * greeting, one line on what the day holds with where it came from, and the
 * box you talk into. Under the box it tells before it asks: the day in
 * figures, what needs the reader soonest, what changed since they last
 * looked, what agents settled and what they have in hand. Each block shows
 * only to someone who may read what it counts.
 *
 * The line under the greeting is the morning briefing's headline when one was
 * written (every figure in it was checked against the facts it was written
 * from), and otherwise a sentence composed from counts, never from a model. A
 * briefing cut to the sections the reader may open carries no headline, since
 * it was written about the whole page; a quiet line names what was left out.
 */
export function DeskToday({ agents, threads, isLoading, isStarting, onStart }: DeskTodayProps) {
  const t = useT();
  const navigate = useNavigate();
  const [opened] = useState(nowInSeconds);
  const now = useNow("minute");
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const [startOfDay] = useState(() => getStartOfDay(new Date(), timezone));

  const { allowed: canDecide, isLoading: permissionsLoading } = usePermission(
    Resource.AgentProposal,
    Operation.Read,
  );
  const { allowed: canWatch } = usePermission(Resource.Watchtower, Operation.Read);
  const { data: attentionSummary } = useAttentionSummary();
  const briefingQuery = useQuery({ ...briefingQueries.today(), retry: false });
  const briefing = briefingQuery.data ?? null;
  const summaryQuery = usePendingDecisionSummary(canDecide);
  const attentionQuery = useQuery({ ...deskAttention.list(), enabled: canWatch });
  // Its own read, so a run on a record far down Needs you is still listed.
  const workingQuery = useQuery({ ...deskAttention.working, enabled: canWatch });
  const { allowed: canSeeRuns } = usePermission(Resource.AgentRun, Operation.Read);
  const runsQuery = useQuery({ ...aiControl.workingRuns(), enabled: canWatch && canSeeRuns });
  const countsQuery = useQuery({ ...watchtower.counts(), enabled: canWatch });
  const seenAt = countsQuery.data?.seenAt ?? 0;
  const clearedQuery = useQuery({ ...watchtower.cleared(seenAt), enabled: canWatch && seenAt > 0 });
  const activityQuery = useQuery({
    ...deskActivity.since(startOfDay, ACTIVITY_READ),
    enabled: canDecide,
  });

  const preferredAgent = useDeskSetting("agent");
  const ask = useDeskHomeAsk({
    agents,
    threads,
    isLoading,
    preferId: preferredAgent,
    space: "desk-today",
    onStart,
  });

  const dateline = new Intl.DateTimeFormat(undefined, {
    weekday: "long",
    month: "long",
    day: "numeric",
    timeZone: timezone,
  }).format(new Date(opened * 1000));
  const hour = toUserWallClock(opened, timezone)?.getHours() ?? 9;
  const firstName = user?.name?.trim().split(/\s+/)[0] ?? "";

  const waiting = summaryQuery.data?.total ?? attentionSummary?.agentDecisions ?? 0;
  const headline = briefing?.headline ? (
    <BriefingHeadline headline={briefing.headline} links={briefing.headlineLinks} />
  ) : canDecide && waiting > 0 ? (
    <Link to="/desk/decisions" className="dk-dy-lnk ui-focus-ring">
      {t(
        "{0, plural, one {One decision is waiting on you.} other {# decisions are waiting on you.}}",
        waiting,
      )}
    </Link>
  ) : ask.noAgents ? (
    t("Nothing is running here yet.")
  ) : (
    t("How can I help today?")
  );

  const withheld = briefing?.withheldSections ?? [];
  const headlinePending =
    isLoading ||
    permissionsLoading ||
    briefingQuery.isPending ||
    (canDecide && summaryQuery.isPending);

  const groups = attentionQuery.data?.groups ?? [];
  const entries = activityQuery.data?.entries ?? [];
  const sinceEntries = entries.filter((entry) => entry.at > seenAt);
  const approvedByOthers = sinceEntries.filter(
    (entry) =>
      entry.outcome === "Approved" && entry.decidedBy !== null && entry.decidedBy.id !== user?.id,
  );
  const failedSince = sinceEntries.filter((entry) => entry.outcome === "Failed").length;
  const needsCount = attentionQuery.data?.counts.everything ?? 0;
  const handledCount = activityQuery.data?.hasMore
    ? t("{0}+", entries.length)
    : String(entries.length);
  const pathOfWork = (itemId: string) => {
    const group = workingQuery.data?.groups.find((candidate) =>
      candidate.items.some((item) => item.id === itemId),
    );
    return group ? attentionPath(group) : "/desk/needs";
  };
  // The runs AI control lists as working are the truth; the items say which of
  // them is on something in Needs you. A reader who may not read runs sees the
  // items alone.
  const working = workingRows(
    canSeeRuns ? (runsQuery.data ?? []) : null,
    workingItems(workingQuery.data?.groups ?? []),
    (item) => pathOfWork(item.id),
  );
  const workingPending = workingQuery.isPending || (canSeeRuns && runsQuery.isPending);
  const workingFailed = canSeeRuns ? runsQuery.isError : workingQuery.isError;

  return (
    <div className="dk-dy-w">
      <DeskDropOverlay show={ask.drag.on} hot={ask.drag.hot} count={ask.drag.count} />
      <div className="dk-dy-scroll">
        <div className="dk-dy">
          <div className="dk-dy-date">{dateline}</div>
          <h1 className="dk-dy-greet">{greeting(t, partOfDay(hour), firstName)}</h1>
          <p className="dk-dy-hl" data-slot="desk-headline">
            {headlinePending ? (
              <span className="dk-dy-hl-sk ui-shimmer" data-slot="skeleton" />
            ) : (
              headline
            )}
          </p>
          {!headlinePending && withheld.length > 0 && (
            <p className="dk-dy-hl-note" data-slot="desk-briefing-withheld">
              {withheldNote(t, withheld)}
            </p>
          )}
          {!headlinePending && withheld.length === 0 && briefing?.headline && (
            <p className="dk-dy-hl-note" data-slot="desk-headline-source">
              {briefing.narrated && <NovaBadge size="xs" />}
              {briefing.narrated
                ? t(
                    "Written by Nova from this morning's figures; every number was checked against them.",
                  )
                : t("Counted from this morning's figures.")}
            </p>
          )}

          <div className="dk-dy-cmp">
            <DeskHomeComposer ask={ask} isStarting={isStarting} />
          </div>

          {briefing && <DayStrip sections={briefing.sections} />}

          {canWatch && (
            <TodaySection
              id="dk-dy-needs"
              title={t("Needs you")}
              count={needsCount > 0 ? String(needsCount) : undefined}
              more={{ to: "/desk/needs", label: t("All of them"), shortcut: "G N" }}
            >
              {attentionQuery.isPending ? (
                <TodayRowsSkeleton rows={TOP_ROWS} />
              ) : attentionQuery.isError ? (
                <p className="dk-dy-quiet">{t("What needs you could not be read.")}</p>
              ) : groups.length === 0 ? (
                <p className="dk-dy-quiet">{t("Nothing needs you right now.")}</p>
              ) : (
                <div className="dk-dy-rows">
                  {groups.slice(0, TOP_ROWS).map((group) => (
                    <AttentionRow
                      key={group.key}
                      group={group}
                      now={now}
                      onOpen={() => void navigate(attentionPath(group))}
                    />
                  ))}
                </div>
              )}
            </TodaySection>
          )}

          {canWatch && seenAt > 0 && (
            <TodaySection
              id="dk-dy-since"
              title={t("Since you left at {0}", formatUnixTime(seenAt, { timezone }))}
            >
              <SinceYouLeft
                unseen={countsQuery.data?.unseen ?? 0}
                cleared={clearedQuery.data ?? 0}
                approvedByOthers={approvedByOthers}
                failed={failedSince}
              />
            </TodaySection>
          )}

          {canDecide && (
            <TodaySection
              id="dk-dy-handled"
              title={t("Agents handled")}
              count={entries.length > 0 ? handledCount : undefined}
              aside={t("since midnight")}
            >
              {activityQuery.isPending ? (
                <TodayRowsSkeleton rows={3} />
              ) : activityQuery.isError ? (
                <p className="dk-dy-quiet">{t("What agents did today could not be read.")}</p>
              ) : entries.length === 0 ? (
                <p className="dk-dy-quiet">{t("No agent has changed anything today.")}</p>
              ) : (
                <NovaHandled entries={entries} timezone={timezone} />
              )}
            </TodaySection>
          )}

          {canWatch && (
            <TodaySection
              id="dk-dy-working"
              title={t("Agents are working on")}
              count={working.length > 0 ? String(working.length) : undefined}
            >
              {workingPending ? (
                <TodayRowsSkeleton rows={2} />
              ) : workingFailed ? (
                <p className="dk-dy-quiet">{t("What agents are working on could not be read.")}</p>
              ) : working.length === 0 ? (
                <p className="dk-dy-quiet">
                  {t("No agent is working on anything right now. Hand an item off from Needs you.")}
                </p>
              ) : (
                <AgentsWorking rows={working} />
              )}
            </TodaySection>
          )}
        </div>
      </div>
    </div>
  );
}
