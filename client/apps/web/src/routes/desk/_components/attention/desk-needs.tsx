import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { usePermission } from "@/hooks/use-permission";
import type { DeskAttentionGroup, DeskLens } from "@/lib/graphql/desk-attention";
import { isTypingTarget } from "@/lib/dom";
import { deskAttention } from "@/lib/queries/desk-attention";
import { watchtower } from "@/lib/queries/watchtower";
import { useDeskMarginStore } from "@/stores/desk-margin-store";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { getStartOfDay, resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { WatchtowerSeverity, WatchtowerSourceKind } from "@trenova/graphql/generated/graphql";
import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { LANES, laneLabel, laneOf, minutesLeft, type Lane } from "../watchtower/lanes";
import { useWatchtowerActions } from "../watchtower/use-watchtower-actions";
import { casePath } from "./attention-record";
import { AttentionRow } from "./attention-row";
import { ItemDetail } from "./item-detail";
import { DESK_LENSES, attentionItems, lensFromParam, lensLabel, lensParam } from "./lens";
import { NeedsSkeleton } from "./needs-skeleton";
import { useItemDecision } from "./use-item-decision";

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

function listParam<T extends string>(value: string | null): T[] {
  return value ? (value.split(",").filter((part) => part !== "") as T[]) : [];
}

/**
 * Needs you: everything that waits on people, one row per record, soonest
 * deadline first, in lanes by how long is left. Watchtower, Triage and
 * Decisions are lenses on the one list, not separate places: a shipment's
 * late warning, the customer's email about it and the plan to recover it are
 * one row. The item chosen is read in full beside the list and acted on from
 * the bar under it, with the same keys everywhere.
 */
export function DeskNeeds() {
  const t = useT();
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const now = useNow("minute");
  const [since] = useState(() => getStartOfDay(new Date(), timezone));
  const { allowed: canAct } = usePermission(Resource.Watchtower, Operation.Update);
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Update);
  const [searchParams, setSearchParams] = useSearchParams();
  const lens = lensFromParam(searchParams.get("lens"));
  const kinds = listParam<WatchtowerSourceKind>(searchParams.get("kinds"));
  const severities = listParam<WatchtowerSeverity>(searchParams.get("severities"));
  const askedItem = searchParams.get("item");

  const query = useQuery(deskAttention.list({ lens, kinds, severities }));
  const clearedQuery = useQuery(watchtower.cleared(since));
  const groups = useMemo(() => query.data?.groups ?? [], [query.data]);
  const items = useMemo(() => attentionItems(groups), [groups]);
  const lanes = useMemo(() => byLane(groups, now), [groups, now]);

  const [declineOpen, setDeclineOpen] = useState(false);
  const actions = useWatchtowerActions({
    items,
    now,
    timezone,
    since,
    keysEnabled: canAct && !declineOpen,
  });
  const { current, select } = actions;
  const currentGroup = current
    ? (groups.find((group) => group.items.some((item) => item.id === current.id)) ?? null)
    : null;
  const currentItem = currentGroup?.items.find((item) => item.id === current?.id) ?? null;
  const decision = useItemDecision(currentItem?.deskLens === "Decisions" ? currentItem : null);
  const [showList, setShowList] = useState(askedItem === null);

  // An address that names an item opens on it, once it is in the list.
  const [opened, setOpened] = useState<string | null>(null);
  if (askedItem !== null && askedItem !== opened && items.some((item) => item.id === askedItem)) {
    setOpened(askedItem);
    select(askedItem);
  }

  // What the open item is about rides on ⌘J and Ask.
  const setPageSubject = useDeskMarginStore((state) => state.setPageSubject);
  const subjectRecord =
    currentGroup?.record?.readable && currentGroup.record.label !== "" ? currentGroup.record : null;
  useEffect(() => {
    setPageSubject(
      subjectRecord
        ? { kind: subjectRecord.kind, id: subjectRecord.id, label: subjectRecord.label }
        : null,
    );
  }, [setPageSubject, subjectRecord]);
  useEffect(() => () => setPageSubject(null), [setPageSubject]);

  const setParam = (name: string, value: string | null) =>
    setSearchParams(
      (params) => {
        const updated = new URLSearchParams(params);
        if (value === null || value === "") {
          updated.delete(name);
        } else {
          updated.set(name, value);
        }
        return updated;
      },
      { replace: true },
    );
  const setLens = (next: DeskLens) => setParam("lens", lensParam(next));
  const openGroup = (group: DeskAttentionGroup) => {
    select(group.items[0].id);
    setShowList(false);
  };

  // The page's own keys: the lens by number, Enter for the case, A and D for
  // a decision. A decision is never cleared with E or S: it is decided.
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (
        event.defaultPrevented ||
        event.metaKey ||
        event.ctrlKey ||
        event.altKey ||
        declineOpen ||
        isTypingTarget(event.target)
      ) {
        return;
      }
      const key = event.key.toLowerCase();
      const index = ["1", "2", "3", "4"].indexOf(event.key);
      if (index >= 0) {
        event.preventDefault();
        setLens(DESK_LENSES[index]);
        return;
      }
      const isDecision = currentItem?.deskLens === "Decisions";
      if (isDecision && (key === "e" || key === "s")) {
        event.preventDefault();
        return;
      }
      if (
        event.key === "Enter" &&
        currentGroup?.record?.readable &&
        currentGroup.record.label !== ""
      ) {
        event.preventDefault();
        void navigate(casePath(currentGroup.record.kind, currentGroup.record.id));
        return;
      }
      if (isDecision && canDecide && decision.phase === "idle" && key === "a") {
        event.preventDefault();
        if (decision.tally.loading + decision.tally.failed + decision.tally.stale === 0) {
          void decision.approve();
        }
        return;
      }
      if (isDecision && canDecide && decision.phase === "idle" && key === "d") {
        event.preventDefault();
        setDeclineOpen(true);
      }
    };
    window.addEventListener("keydown", onKey, true);
    return () => window.removeEventListener("keydown", onKey, true);
  });

  if (query.isPending) {
    return <NeedsSkeleton />;
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
  const filtered = kinds.length > 0 || severities.length > 0;

  return (
    <div className={cn("dk-w-nq", !showList && currentItem && "dk-has-sel")}>
      <div className="dk-w-nq-l">
        <div>
          <div className="dk-w-nq-h">
            <b>{t("Needs you")}</b>
            <span className="dk-w-nq-c">{t("{0} of {1} cleared today", cleared, total)}</span>
          </div>
          <div className="dk-w-nq-bar" aria-hidden>
            <i style={{ transform: `scaleX(${total === 0 ? 0 : cleared / total})` }} />
          </div>
        </div>
        <div className="dk-w-lens" role="tablist" aria-label={t("Lens")}>
          {DESK_LENSES.map((value, index) => (
            <button
              key={value}
              type="button"
              role="tab"
              aria-selected={lens === value}
              className="dk-w-chip ui-focus-ring"
              onClick={() => setLens(value)}
              title={t("Press {0}", index + 1)}
            >
              {lensLabel(t, value)}
              <span className="dk-w-chip-n">{countOf(value)}</span>
            </button>
          ))}
          {filtered && (
            <button
              type="button"
              className="dk-w-chip ui-focus-ring"
              aria-pressed
              onClick={() => {
                setParam("kinds", null);
                setParam("severities", null);
              }}
            >
              {t("Filtered")}
              <span className="dk-w-chip-n">×</span>
            </button>
          )}
        </div>
        {(lens === "Triage" || lens === "Decisions") && (
          <div className="dk-w-lens-deep">
            {lens === "Triage" ? (
              <Link className="dk-w-alnk ui-focus-ring" to="/desk/triage">
                {t("Read and file messages in the Triage room")}
              </Link>
            ) : (
              <Link className="dk-w-alnk ui-focus-ring" to="/desk/decisions">
                {t("Approve several at once in Decisions")}
              </Link>
            )}
          </div>
        )}
        <div
          className="dk-w-nq-list"
          role="listbox"
          aria-label={t("Things that need you")}
          aria-activedescendant={currentGroup ? `dk-w-row-${currentGroup.key}` : undefined}
        >
          {query.isError ? (
            <div className="dk-w-empty">
              <div>
                <b>{t("What needs you could not be read")}</b>
                <Button size="sm" variant="outline" onClick={() => void query.refetch()}>
                  {t("Try again")}
                </Button>
              </div>
            </div>
          ) : groups.length === 0 ? (
            <div className="dk-w-empty">
              <div>
                <div className="dk-w-okwell">
                  <DeskIcon name="check" size={18} />
                </div>
                <b>{t("Nothing here needs you")}</b>
                {t("Agents ask here before they change anything.")}
              </div>
            </div>
          ) : (
            lanes.map(({ lane, groups: laneGroups }) => (
              <div key={lane} role="group" aria-label={laneLabel(t, lane)}>
                <div className="dk-w-nq-g">{laneLabel(t, lane)}</div>
                {laneGroups.map((group) => (
                  <AttentionRow
                    key={group.key}
                    id={`dk-w-row-${group.key}`}
                    group={group}
                    now={now}
                    selected={currentGroup?.key === group.key}
                    onOpen={() => openGroup(group)}
                  />
                ))}
              </div>
            ))
          )}
          {query.data?.hasMore && (
            <p className="dk-w-quiet dk-w-nq-more">
              {t(
                "These are the soonest records. More wait after them; clear these to see the rest.",
              )}
            </p>
          )}
        </div>
      </div>
      <div className="dk-w-nq-d">
        {currentItem && currentGroup ? (
          <ItemDetail
            key={currentItem.id}
            item={currentItem}
            group={currentGroup}
            now={now}
            timezone={timezone}
            actions={actions}
            decision={decision}
            canAct={canAct}
            canDecide={canDecide}
            declining={declineOpen}
            onDecliningChange={setDeclineOpen}
            onBack={() => setShowList(true)}
          />
        ) : (
          <div className="dk-w-empty">
            <div>{t("Pick something on the left, or press J.")}</div>
          </div>
        )}
      </div>
    </div>
  );
}
