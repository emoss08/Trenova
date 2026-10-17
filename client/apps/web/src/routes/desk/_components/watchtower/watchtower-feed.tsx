import { usePermission } from "@/hooks/use-permission";
import type { WatchtowerItem } from "@/lib/graphql/watchtower";
import { watchtower, watchtowerQueueQuery } from "@/lib/queries/watchtower";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { MessageChatSquareIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { getStartOfDay, resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router";
import { QueueClear, QueueHeader, QueueLayout } from "../queue/queue";
import {
  QUEUE_NAV_CLASS,
  QueueBar,
  QueueBarSpacer,
  QueueButton,
  QueueNav,
} from "../queue/queue-bar";
import { QueueGroupHeader, QueueList, type QueueSection } from "../queue/queue-list";
import { WatchtowerHandOff } from "./hand-off-menu";
import { WatchtowerItemCard } from "./item-card";
import { WatchtowerItemRow } from "./item-row";
import { groupByLane, laneLabel } from "./lanes";
import { OnItStrip } from "./on-it-strip";
import { useWatchtowerActions } from "./use-watchtower-actions";
import { WatchtowerSkeleton } from "./watchtower-skeleton";

/** The address parameters a link lands on the queue with: an item, a severity, a kind. */
const FOCUS_PARAMS = ["item", "severities", "kinds"] as const;

function focusMatch(searchParams: URLSearchParams): ((item: WatchtowerItem) => boolean) | null {
  const itemId = searchParams.get("item");
  if (itemId) {
    return (item) => item.id === itemId;
  }
  const severities = searchParams.get("severities")?.split(",").filter(Boolean) ?? [];
  if (severities.length > 0) {
    return (item) => severities.includes(item.severity);
  }
  const kinds = searchParams.get("kinds")?.split(",").filter(Boolean) ?? [];
  if (kinds.length > 0) {
    return (item) => kinds.includes(item.sourceKind);
  }
  return null;
}

/**
 * The Watchtower, worked one item at a time, soonest deadline first.
 *
 * The queue down the left is grouped by how long is left — act now, today,
 * coming up, and nothing due — with the agents already on something in a
 * strip above it. The card in the middle is the item in front of the person,
 * and the bar under it walks the queue and acts on the item: snooze, done, or
 * hand it to an agent. A link that names an item, a severity or a kind (a
 * notification, the briefing's "Critical" block) lands on the first item it
 * names.
 */
export function WatchtowerFeed() {
  const t = useT();
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const now = useNow("minute");
  const [since] = useState(() => getStartOfDay(new Date(), timezone));
  const { allowed: canAct } = usePermission(Resource.Watchtower, Operation.Update);
  const [searchParams, setSearchParams] = useSearchParams();

  const queue = useInfiniteQuery(watchtowerQueueQuery());
  const clearedQuery = useQuery(watchtower.cleared(since));
  const countsQuery = useQuery(watchtower.counts());

  const lanes = useMemo(() => {
    const byId = new Map<string, WatchtowerItem>();
    for (const page of queue.data?.pages ?? []) {
      for (const item of page.items) {
        if (!byId.has(item.id)) {
          byId.set(item.id, item);
        }
      }
    }
    return groupByLane([...byId.values()], now);
  }, [now, queue.data]);
  const ordered = useMemo(() => lanes.flatMap((lane) => lane.items), [lanes]);

  const actions = useWatchtowerActions({
    items: ordered,
    now,
    timezone,
    since,
    keysEnabled: canAct,
  });
  const { current, select } = actions;

  // A link names what to land on; the queue opens on the first item it
  // names, reading on until it is loaded, and the address then forgets it so
  // going back does not land there again.
  const match = focusMatch(searchParams);
  const focusing = match !== null;
  const targetId = match === null ? null : (ordered.find(match)?.id ?? null);
  const { fetchNextPage, hasNextPage, isFetchingNextPage, isPending } = queue;
  useEffect(() => {
    if (!focusing || isPending) {
      return;
    }
    if (targetId !== null) {
      select(targetId);
    } else if (hasNextPage) {
      if (!isFetchingNextPage) {
        void fetchNextPage();
      }
      return;
    }
    setSearchParams(
      (params) => {
        const next = new URLSearchParams(params);
        FOCUS_PARAMS.forEach((name) => next.delete(name));
        return next;
      },
      { replace: true },
    );
  }, [
    fetchNextPage,
    focusing,
    hasNextPage,
    isFetchingNextPage,
    isPending,
    select,
    setSearchParams,
    targetId,
  ]);

  if (queue.isPending) {
    return <WatchtowerSkeleton />;
  }

  const cleared = clearedQuery.data ?? 0;
  const open = countsQuery.data?.active ?? ordered.length;
  const total = cleared + open;
  const working = ordered.filter((item) => item.agentWork?.status === "Working");

  const sections: QueueSection[] = lanes.map(({ lane, items }) => ({
    key: lane,
    className: `dk-w3-l-${lane}`,
    header: (
      <QueueGroupHeader
        label={<b className="dk-w3-lt">{laneLabel(t, lane)}</b>}
        count={items.length}
      />
    ),
    rows: items.map((item) => ({
      key: item.id,
      render: () => (
        <WatchtowerItemRow
          item={item}
          now={now}
          selected={current?.id === item.id}
          onSelect={() => select(item.id)}
        />
      ),
    })),
  }));

  return (
    <QueueLayout
      label={t("Watchtower")}
      className="dk-w3"
      aside={
        <>
          <QueueHeader
            title={t("Watchtower")}
            progressLabel={t("{0} of {1} cleared", cleared, total)}
            done={cleared}
            total={total}
          />
          {working.length > 0 && (
            <OnItStrip working={working} selectedId={current?.id ?? null} onSelect={select} />
          )}
          <QueueList
            label={t("Watchtower queue")}
            sections={sections}
            selectedKey={current?.id ?? null}
            empty={<div className="dk-dc2-qe">{t("All clear")}</div>}
            onEndReached={
              hasNextPage && !isFetchingNextPage ? () => void fetchNextPage() : undefined
            }
          />
        </>
      }
    >
      {queue.isError ? (
        <div className="dk-dc2-clear">
          <b>{t("The queue could not be read")}</b>
          <Button size="sm" variant="outline" onClick={() => void queue.refetch()}>
            {t("Try again")}
          </Button>
        </div>
      ) : current ? (
        <WatchtowerItemCard
          key={current.id}
          item={current}
          timezone={timezone}
          leaving={actions.leaving}
          onReviewAndApprove={actions.reviewAndApprove}
        />
      ) : (
        <QueueClear
          title={t("All clear")}
          body={t(
            "{0, plural, one {You cleared # today.} other {You cleared # today.}} New signals show up here the moment they happen.",
            cleared,
          )}
        />
      )}
      {current && (
        <QueueBar>
          <QueueNav onPrevious={actions.previous} onNext={actions.next} />
          <Button
            variant="quiet"
            size="bare"
            className={cn(QUEUE_NAV_CLASS, "dk-w3-ask ml-1.5 inline-flex h-7.5 items-center")}
            onClick={actions.ask}
            disabled={!actions.canAsk}
            title={t("Ask about this")}
          >
            <MessageChatSquareIcon aria-hidden className="size-3.5" />
            <span>{t("Ask about this")}</span>
          </Button>
          <QueueBarSpacer />
          {canAct ? (
            <>
              <QueueButton shortcut="S" onClick={actions.snooze}>
                {t("Snooze")}
              </QueueButton>
              <QueueButton shortcut="E" onClick={actions.done}>
                {t("Done")}
              </QueueButton>
              {current.agentWork === null && current.subjectId !== null && (
                <WatchtowerHandOff
                  suggested={current.suggestedAgent}
                  open={actions.menuOpen}
                  onOpenChange={actions.setMenuOpen}
                  onHandOff={actions.handOff}
                  disabled={actions.handingOff}
                />
              )}
            </>
          ) : (
            <span className="dk-dc2-qe">{t("You can read these, but not act on them.")}</span>
          )}
        </QueueBar>
      )}
    </QueueLayout>
  );
}
