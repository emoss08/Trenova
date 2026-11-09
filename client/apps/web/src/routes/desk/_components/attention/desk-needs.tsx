import { usePermission } from "@/hooks/use-permission";
import type { DeskAttentionGroup, DeskLens } from "@/lib/graphql/desk-attention";
import { deskAttention } from "@/lib/queries/desk-attention";
import { watchtower } from "@/lib/queries/watchtower";
import { isTypingTarget } from "@/lib/dom";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { getStartOfDay, resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router";
import { QueueClear, QueueHeader, QueueLayout } from "../queue/queue";
import { QueueGroupHeader, QueueList, type QueueSection } from "../queue/queue-list";
import { WatchtowerItemBar } from "../watchtower/item-bar";
import { WatchtowerItemCard } from "../watchtower/item-card";
import { LANES, laneLabel, laneOf, minutesLeft, type Lane } from "../watchtower/lanes";
import { useWatchtowerActions } from "../watchtower/use-watchtower-actions";
import { WatchtowerSkeleton } from "../watchtower/watchtower-skeleton";
import { AttentionRecordHead } from "./attention-record";
import { AttentionRow } from "./attention-row";
import { DESK_LENSES, attentionItems, lensFromParam, lensLabel, lensParam } from "./lens";

function byLane(groups: readonly DeskAttentionGroup[], now: number) {
  const lanes = new Map<Lane, DeskAttentionGroup[]>();
  for (const group of groups) {
    const lane = laneOf(minutesLeft(group.dueAt, now));
    lanes.set(lane, [...(lanes.get(lane) ?? []), group]);
  }
  return LANES.filter((lane) => lanes.has(lane)).map((lane) => ({
    lane,
    groups: lanes.get(lane) ?? [],
  }));
}

/**
 * Needs you: everything that waits on people, one row per record, soonest
 * deadline first. Watchtower, Triage and Decisions are lenses on it, not
 * separate places: a shipment's late warning, the customer's email about it
 * and the plan to recover it are one row, and each lens still opens its own
 * page for the deep work. The card is the signal in front of the person and
 * the bar acts on it, with the Watchtower's own keys.
 */
export function DeskNeeds() {
  const t = useT();
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const now = useNow("minute");
  const [since] = useState(() => getStartOfDay(new Date(), timezone));
  const { allowed: canAct } = usePermission(Resource.Watchtower, Operation.Update);
  const [searchParams, setSearchParams] = useSearchParams();
  const lens = lensFromParam(searchParams.get("lens"));

  const query = useQuery(deskAttention.list({ lens }));
  const clearedQuery = useQuery(watchtower.cleared(since));
  const groups = useMemo(() => query.data?.groups ?? [], [query.data]);
  const items = useMemo(() => attentionItems(groups), [groups]);
  const lanes = useMemo(() => byLane(groups, now), [groups, now]);

  const actions = useWatchtowerActions({ items, now, timezone, since, keysEnabled: canAct });
  const { current, select } = actions;
  const currentGroup = current
    ? (groups.find((group) => group.items.some((item) => item.id === current.id)) ?? null)
    : null;

  const setLens = (next: DeskLens) =>
    setSearchParams(
      (params) => {
        const updated = new URLSearchParams(params);
        const value = lensParam(next);
        if (value === "") {
          updated.delete("lens");
        } else {
          updated.set("lens", value);
        }
        return updated;
      },
      { replace: true },
    );

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (
        event.defaultPrevented ||
        event.metaKey ||
        event.ctrlKey ||
        event.altKey ||
        isTypingTarget(event.target)
      ) {
        return;
      }
      const index = ["1", "2", "3", "4"].indexOf(event.key);
      if (index >= 0) {
        event.preventDefault();
        setLens(DESK_LENSES[index]);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  if (query.isPending) {
    return <WatchtowerSkeleton label={t("Reading what needs you")} lenses={DESK_LENSES.length} />;
  }

  const counts = query.data?.counts;
  const cleared = clearedQuery.data ?? 0;
  const open = counts?.everything ?? items.length;
  const total = cleared + open;
  const countOf = (value: DeskLens) =>
    counts
      ? {
          Everything: counts.everything,
          Watchtower: counts.watchtower,
          Triage: counts.triage,
          Decisions: counts.decisions,
        }[value]
      : 0;

  const sections: QueueSection[] = lanes.map(({ lane, groups: laneGroups }) => ({
    key: lane,
    className: `dk-w3-l-${lane}`,
    header: (
      <QueueGroupHeader
        label={<b className="dk-w3-lt">{laneLabel(t, lane)}</b>}
        count={laneGroups.length}
      />
    ),
    rows: laneGroups.map((group) => ({
      key: group.key,
      render: () => (
        <AttentionRow
          group={group}
          now={now}
          selected={currentGroup?.key === group.key}
          onSelect={() => select(group.items[0].id)}
        />
      ),
    })),
  }));

  return (
    <QueueLayout
      label={t("Needs you")}
      className="dk-w3 dk-at"
      aside={
        <>
          <QueueHeader
            title={t("Needs you")}
            progressLabel={t("{0} of {1} cleared", cleared, total)}
            done={cleared}
            total={total}
          />
          <div className="dk-at-lens" role="tablist" aria-label={t("Lens")}>
            {DESK_LENSES.map((value, index) => (
              <button
                key={value}
                type="button"
                role="tab"
                aria-selected={lens === value}
                className={cn("dk-at-chip ui-focus-ring", lens === value && "dk-on")}
                onClick={() => setLens(value)}
                title={t("Press {0}", index + 1)}
              >
                {lensLabel(t, value)}
                <i>{countOf(value)}</i>
              </button>
            ))}
          </div>
          <QueueList
            label={t("What needs you")}
            sections={sections}
            selectedKey={currentGroup?.key ?? null}
            empty={<div className="dk-dc2-qe">{t("All clear")}</div>}
            footer={
              query.data?.hasMore ? (
                <Link className="dk-at-more ui-focus-ring" to={lens === "Decisions" ? "/desk/decisions" : "/desk/watchtower"}>
                  {t("More than this waits. Open the full queue")}
                </Link>
              ) : undefined
            }
          />
        </>
      }
    >
      {query.isError ? (
        <div className="dk-dc2-clear">
          <b>{t("What needs you could not be read")}</b>
          <Button size="sm" variant="outline" onClick={() => void query.refetch()}>
            {t("Try again")}
          </Button>
        </div>
      ) : current && currentGroup ? (
        <>
          <AttentionRecordHead group={currentGroup} currentId={current.id} onSelect={select} />
          <WatchtowerItemCard
            key={current.id}
            item={current}
            timezone={timezone}
            leaving={actions.leaving}
            onReviewAndApprove={actions.reviewAndApprove}
          />
        </>
      ) : (
        <QueueClear
          title={t("Nothing needs you")}
          body={t(
            "{0, plural, one {You cleared # today.} other {You cleared # today.}} Agents ask here before they change anything.",
            cleared,
          )}
        />
      )}
      {current && <WatchtowerItemBar item={current} actions={actions} canAct={canAct} />}
    </QueueLayout>
  );
}
