import type { Briefing } from "@/lib/graphql/briefing";
import { deskActivity } from "@/lib/queries/desk-activity";
import { deskAttention } from "@/lib/queries/desk-attention";
import { watchtower } from "@/lib/queries/watchtower";
import { useQuery } from "@tanstack/react-query";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime, getStartOfDay } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useState } from "react";
import { useNavigate } from "react-router";
import { AttentionRow } from "../attention/attention-row";
import { attentionPath } from "../attention/attention-record";
import { DeskDayStrip } from "./day-strip";
import { NovaHandled } from "./nova-handled";
import { NovaWorking } from "./nova-working";
import { SinceYouLeft } from "./since-you-left";
import { TodaySection } from "./today-section";

const TOP_ROWS = 4;
const ACTIVITY_LINES = 6;

function SectionSkeleton({ rows }: { rows: number }) {
  return (
    <div className="dk-w-sk" aria-hidden>
      {Array.from({ length: rows }, (_, index) => (
        <i key={index} className="dk-sk" />
      ))}
    </div>
  );
}

/**
 * Today under the box, telling before it asks: the day in figures, what
 * needs the reader soonest (one row per record, its signals under it), what
 * changed since they last looked, what agents settled and what they have in
 * hand. Each block shows only to someone who may read what it counts.
 */
export function DeskToday({
  briefing,
  canWatch,
  canDecide,
  timezone,
}: {
  briefing: Briefing | null;
  canWatch: boolean;
  canDecide: boolean;
  timezone: string;
}) {
  const t = useT();
  const navigate = useNavigate();
  const now = useNow("minute");
  const userId = useAuthStore((state) => state.user?.id ?? null);
  const [startOfDay] = useState(() => getStartOfDay(new Date(), timezone));

  const attentionQuery = useQuery({ ...deskAttention.list(), enabled: canWatch });
  const countsQuery = useQuery({ ...watchtower.counts(), enabled: canWatch });
  const seenAt = countsQuery.data?.seenAt ?? 0;
  const clearedQuery = useQuery({ ...watchtower.cleared(seenAt), enabled: canWatch && seenAt > 0 });
  const activityQuery = useQuery({
    ...deskActivity.since(startOfDay, ACTIVITY_LINES * 3),
    enabled: canDecide,
  });

  const groups = attentionQuery.data?.groups ?? [];
  const working = groups
    .flatMap((group) => group.items)
    .filter((item) => item.agentWork?.status === "Working");
  const entries = activityQuery.data?.entries ?? [];
  const sinceEntries = entries.filter((entry) => entry.at > seenAt);
  const approvedByOthers = sinceEntries.filter(
    (entry) =>
      entry.outcome === "Approved" && entry.decidedBy !== null && entry.decidedBy.id !== userId,
  );
  const failedSince = sinceEntries.filter((entry) => entry.outcome === "Failed").length;
  const groupOf = (itemId: string) =>
    groups.find((group) => group.items.some((item) => item.id === itemId)) ?? null;

  return (
    <>
      {briefing && <DeskDayStrip sections={briefing.sections} />}

      {canWatch && (
        <TodaySection
          id="dk-w-needs"
          title={t("Needs you")}
          count={attentionQuery.data?.counts.everything}
          more={{ to: "/desk/needs", label: t("All of them"), shortcut: "G N" }}
        >
          {attentionQuery.isPending ? (
            <SectionSkeleton rows={TOP_ROWS} />
          ) : attentionQuery.isError ? (
            <p className="dk-w-quiet">{t("What needs you could not be read.")}</p>
          ) : groups.length === 0 ? (
            <p className="dk-w-quiet">{t("Nothing needs you right now.")}</p>
          ) : (
            <div className="dk-w-rows">
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
          id="dk-w-since"
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
          id="dk-w-handled"
          title={t("Nova handled today")}
          count={entries.length}
          aside={t("since midnight")}
        >
          {activityQuery.isPending ? (
            <SectionSkeleton rows={3} />
          ) : activityQuery.isError ? (
            <p className="dk-w-quiet">{t("What agents did today could not be read.")}</p>
          ) : entries.length === 0 ? (
            <p className="dk-w-quiet">{t("No agent has changed anything today.")}</p>
          ) : (
            <NovaHandled entries={entries.slice(0, ACTIVITY_LINES)} timezone={timezone} />
          )}
        </TodaySection>
      )}

      {canWatch && (
        <TodaySection id="dk-w-working" title={t("Nova is working on")} count={working.length}>
          {attentionQuery.isPending ? (
            <SectionSkeleton rows={2} />
          ) : working.length === 0 ? (
            <p className="dk-w-quiet">
              {t("No agent is working on anything open. Hand an item off from Needs you.")}
            </p>
          ) : (
            <NovaWorking
              items={working}
              pathOf={(item) => {
                const group = groupOf(item.id);
                return group ? attentionPath(group) : "/desk/needs";
              }}
            />
          )}
        </TodaySection>
      )}
    </>
  );
}
