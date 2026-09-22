import { KPI_VALUE_CLASS, KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useCallback, useMemo, useState } from "react";
import { useSearchParams } from "react-router";
import { InboxMessageDetail } from "./_components/message-detail";
import { InboxMessageRow } from "./_components/message-row";
import { LANE_ORDER, LANE_STATUSES, isLaneKey, type LaneKey } from "./_components/lanes";

const nowInSeconds = () => Math.floor(Date.now() / 1000);

/**
 * The inbox: what arrived on a monitored address, and what was made of it.
 *
 * The lane and the open message are both in the URL rather than in state, so a
 * message somebody is asking about can be linked — which is what the watchtower
 * items point at — and the back button does what it looks like it does.
 */
export function InboxPage() {
  const t = useT();
  const [now] = useState(nowInSeconds);
  const [searchParams, setSearchParams] = useSearchParams();

  const laneParam = searchParams.get("lane");
  const lane: LaneKey = isLaneKey(laneParam) ? laneParam : "waiting";
  const openId = searchParams.get("message");

  const filter = useMemo(() => ({ statuses: LANE_STATUSES[lane] }), [lane]);
  const messagesQuery = useQuery(queries.inbox.messages(filter));
  const countsQuery = useQuery(queries.inbox.counts());

  const setLane = useCallback(
    (next: LaneKey) => {
      const params = new URLSearchParams(searchParams);
      params.set("lane", next);
      // Switching lane closes whatever was open: the message a person was
      // reading is very likely not in the lane they just moved to.
      params.delete("message");
      setSearchParams(params);
    },
    [searchParams, setSearchParams],
  );

  const openMessage = useCallback(
    (id: string | null) => {
      const params = new URLSearchParams(searchParams);
      if (id === null) {
        params.delete("message");
      } else {
        params.set("message", id);
      }
      setSearchParams(params);
    },
    [searchParams, setSearchParams],
  );

  const messages = messagesQuery.data?.messages ?? [];
  const counts = countsQuery.data;

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Inbox"),
        description: t("Mail that arrived on a monitored address, and what was made of it"),
      }}
    >
      <div className="flex flex-col gap-4">
        <KpiStrip>
          <KpiStripItem
            label={t("Waiting on you")}
            value={
              counts ? (
                <span className={KPI_VALUE_CLASS}>{counts.waiting}</span>
              ) : (
                <Skeleton className="h-6 w-8" />
              )
            }
          />
          <KpiStripItem
            label={t("Handled")}
            value={
              counts ? (
                <span className={KPI_VALUE_CLASS}>{counts.handled}</span>
              ) : (
                <Skeleton className="h-6 w-8" />
              )
            }
          />
          <KpiStripItem
            label={t("Held back")}
            value={
              counts ? (
                <span className={KPI_VALUE_CLASS}>{counts.quarantined}</span>
              ) : (
                <Skeleton className="h-6 w-8" />
              )
            }
          />
          <KpiStripItem
            label={t("Received in total")}
            value={
              counts ? (
                <span className={KPI_VALUE_CLASS}>{counts.total}</span>
              ) : (
                <Skeleton className="h-6 w-8" />
              )
            }
          />
        </KpiStrip>

        <div className="border-border flex min-h-[32rem] overflow-hidden rounded-lg border">
          <div
            className={cn(
              "flex min-w-0 flex-col",
              openId === null ? "flex-1" : "hidden flex-1 lg:flex lg:max-w-md",
            )}
          >
            <div className="border-border flex flex-wrap items-center gap-1.5 border-b px-4 py-2.5">
              {LANE_ORDER.map((key) => (
                <LaneChip
                  key={key}
                  label={laneLabel(t, key)}
                  count={laneCount(counts, key)}
                  active={lane === key}
                  onClick={() => setLane(key)}
                />
              ))}
            </div>

            <ScrollArea className="min-h-0 flex-1">
              {messagesQuery.isLoading ? (
                <div className="flex flex-col gap-2 p-4">
                  <Skeleton className="h-16" />
                  <Skeleton className="h-16" />
                  <Skeleton className="h-16" />
                </div>
              ) : messages.length === 0 ? (
                <EmptySheet
                  className="my-10"
                  title={t("Nothing in this lane")}
                  description={t(
                    "Tenders, rate confirmations, proofs of delivery and status requests appear here as they arrive, with what the desk made of each one.",
                  )}
                  sketch={
                    <div className="flex flex-col gap-3">
                      <GhostLine className="w-2/3" />
                      <GhostLine className="w-1/2" />
                      <GhostLine className="w-3/5" />
                    </div>
                  }
                />
              ) : (
                <ul className="flex flex-col">
                  {messages.map((message) => (
                    <InboxMessageRow
                      key={message.id}
                      message={message}
                      active={message.id === openId}
                      now={now}
                      onOpen={(value) => openMessage(value.id)}
                    />
                  ))}
                </ul>
              )}
            </ScrollArea>
          </div>

          {openId !== null && (
            <div className="border-border flex min-w-0 flex-1 flex-col lg:border-l">
              <InboxMessageDetail messageId={openId} onClose={() => openMessage(null)} />
            </div>
          )}
        </div>
      </div>
    </PageLayout>
  );
}

function laneLabel(t: (value: string) => string, lane: LaneKey): string {
  switch (lane) {
    case "waiting":
      return t("Waiting on you");
    case "handled":
      return t("Handled");
    case "ignored":
      return t("Ignored");
    case "all":
      return t("Everything");
  }
}

function laneCount(
  counts: { waiting: number; handled: number; ignored: number; total: number } | undefined,
  lane: LaneKey,
): number | undefined {
  if (counts === undefined) {
    return undefined;
  }
  switch (lane) {
    case "waiting":
      return counts.waiting;
    case "handled":
      return counts.handled;
    case "ignored":
      return counts.ignored;
    case "all":
      return counts.total;
  }
}

function LaneChip({
  label,
  count,
  active,
  onClick,
}: {
  label: string;
  count?: number;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={cn(
        "ui-focus-ring flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs transition-colors",
        active
          ? "bg-foreground text-background"
          : "text-muted-foreground hover:text-foreground ring-foreground/10 ring-1",
      )}
    >
      {label}
      {count !== undefined && (
        <Badge
          variant="neutral"
          className={cn("h-4 px-1 tabular-nums", active && "bg-background/20 text-background")}
        >
          {count}
        </Badge>
      )}
    </button>
  );
}
