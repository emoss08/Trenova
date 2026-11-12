import { DeskIcon } from "@/components/desk-chat/desk-icons";
import type { VirtualRow, VirtualRowsHandle } from "@/components/virtual-rows";
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
import { useEffect, useMemo, useRef, useState } from "react";
import { parseAsArrayOf, parseAsString, useQueryStates } from "nuqs";
import { Link, useNavigate } from "react-router";
import { LANES, laneLabel, laneOf, minutesLeft, type Lane } from "../watchtower/lanes";
import { useWatchtowerActions } from "../watchtower/use-watchtower-actions";
import { casePath, START_CASE } from "./attention-record";
import { itemsWorkedElsewhere } from "./case-work";
import { AttentionRow } from "./attention-row";
import { ItemDetail } from "./item-detail";
import { DeskList, DeskListHeading, DeskListPane, DeskListStill } from "../list/desk-list";
import {
  NEEDS_LENSES,
  attentionItems,
  lensFromParam,
  lensLabel,
  lensParam,
  type NeedsLens,
} from "./lens";
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

/** A row's fixed height (`h-14` on `AttentionRow`); lane headings are measured. */
const ROW_HEIGHT = 56;
/** The list's height before it is measured: a tall window, so the first paint is full. */
const INITIAL_LIST_HEIGHT = 900;
/** Longer than a row's entrance (`dk-w-rise`), after which rows scrolled into view mount still. */
const ENTRANCE_MS = 480;

function rowId(key: string) {
  return `dk-w-row-${key}`;
}

/** What Needs you keeps in the address, in the shape it has always had there. */
const NEEDS_ADDRESS = {
  lens: parseAsString,
  kinds: parseAsArrayOf(parseAsString, ",").withDefault([]),
  severities: parseAsArrayOf(parseAsString, ",").withDefault([]),
  item: parseAsString,
};

/**
 * Needs you: everything that waits on people, one row per record, soonest
 * deadline first, in lanes by how long is left. Watchtower and Decisions are
 * lenses on the one list, not separate places: a shipment's late warning, the
 * customer's email about it and the plan to recover it are one row. The item
 * chosen is read in full beside the list and acted on from the bar under it,
 * with the same keys everywhere. A message is the exception: it shows here as
 * a signal and is worked in Triage, the one place mail is read and filed.
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
  // What is being looked at lives in the address: the lens, the filters and
  // the item open beside the list. So a link sent to someone opens on the
  // same item, and a reload comes back to it.
  const [address, setAddress] = useQueryStates(NEEDS_ADDRESS, { history: "replace" });
  const lens = lensFromParam(address.lens);
  const kinds = address.kinds as WatchtowerSourceKind[];
  const severities = address.severities as WatchtowerSeverity[];
  const askedItem = address.item;

  const query = useQuery(deskAttention.list({ lens, kinds, severities }));
  const clearedQuery = useQuery(watchtower.cleared(since));
  const groups = useMemo(() => query.data?.groups ?? [], [query.data]);
  const items = useMemo(() => attentionItems(groups), [groups]);
  const lanes = useMemo(() => byLane(groups, now), [groups, now]);

  const [declineOpen, setDeclineOpen] = useState(false);
  const workedElsewhere = useMemo(() => itemsWorkedElsewhere(groups), [groups]);
  const actions = useWatchtowerActions({
    items,
    held: workedElsewhere,
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
  // And the item in front is the one the address names, as the person moves
  // through the list. An address naming one that is not here (cleared since
  // the link was sent, or not the reader's to see) falls to the one shown.
  const currentItemId = current?.id ?? null;
  useEffect(() => {
    if (currentItemId !== null && currentItemId !== askedItem) {
      void setAddress({ item: currentItemId });
    }
  }, [askedItem, currentItemId, setAddress]);

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

  const setLens = (next: NeedsLens) => void setAddress({ lens: lensParam(next) || null });
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
      const index = ["1", "2", "3"].indexOf(event.key);
      if (index >= 0) {
        event.preventDefault();
        setLens(NEEDS_LENSES[index]);
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
        void navigate(casePath(currentGroup.record.kind, currentGroup.record.id), {
          state: START_CASE,
          viewTransition: true,
        });
        return;
      }
      if (event.key === "Enter" && currentItem?.deskLens === "Triage" && currentItem.path !== "") {
        event.preventDefault();
        void navigate(currentItem.path);
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

  // Only the rows near the viewport mount, however many records wait: a lane
  // heading, then its records, as one flat list.
  const listRef = useRef<VirtualRowsHandle>(null);
  const listRows: VirtualRow[] = [];
  let position = 0;
  for (const { lane, groups: laneGroups } of lanes) {
    listRows.push({
      key: `lane:${lane}`,
      render: () => <DeskListHeading>{laneLabel(t, lane)}</DeskListHeading>,
    });
    for (const group of laneGroups) {
      const index = position;
      position += 1;
      listRows.push({
        key: group.key,
        render: () => (
          <AttentionRow
            id={rowId(group.key)}
            group={group}
            now={now}
            lens={lens}
            selected={currentGroup?.key === group.key}
            position={{ index, size: groups.length }}
            onOpen={() => openGroup(group)}
          />
        ),
      });
    }
  }

  // The chosen record stays in view as J and K move through rows not mounted yet.
  const currentKey = currentGroup?.key ?? null;
  useEffect(() => {
    if (currentKey !== null) {
      listRef.current?.scrollToKey(currentKey);
    }
  }, [currentKey]);

  // Rows rise in when a lens first draws; a row scrolled into view later mounts still.
  const [settledLens, setSettledLens] = useState<DeskLens | null>(null);
  const settled = settledLens === lens;
  const hasRows = groups.length > 0;
  useEffect(() => {
    if (!hasRows) {
      return;
    }
    const timer = window.setTimeout(() => setSettledLens(lens), ENTRANCE_MS);
    return () => window.clearTimeout(timer);
  }, [lens, hasRows]);

  if (query.isPending) {
    return <NeedsSkeleton />;
  }

  const counts = query.data?.counts;
  const cleared = clearedQuery.data ?? 0;
  const open = counts?.everything ?? items.length;
  const total = cleared + open;
  const countOf = (value: NeedsLens) =>
    counts
      ? {
          Everything: counts.everything,
          Watchtower: counts.watchtower,
          Decisions: counts.decisions,
        }[value]
      : 0;
  const filtered = kinds.length > 0 || severities.length > 0;

  return (
    <div className={cn("dk-w-nq", !showList && currentItem && "dk-has-sel")}>
      <DeskListPane className="dk-w-nq-l">
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
          {NEEDS_LENSES.map((value, index) => (
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
              onClick={() => void setAddress({ kinds: null, severities: null })}
            >
              {t("Filtered")}
              <span className="dk-w-chip-n">×</span>
            </button>
          )}
        </div>
        {lens === "Decisions" && (
          <div className="dk-w-lens-deep">
            <Link className="dk-w-alnk ui-focus-ring" to="/desk/decisions">
              {t("Approve several at once in Decisions")}
            </Link>
          </div>
        )}
        {query.isError || groups.length === 0 ? (
          <DeskListStill>
            {query.isError ? (
              <div className="dk-w-empty">
                <div>
                  <b>{t("What needs you could not be read")}</b>
                  <Button size="sm" variant="outline" onClick={() => void query.refetch()}>
                    {t("Try again")}
                  </Button>
                </div>
              </div>
            ) : (
              <div className="dk-w-empty">
                <div>
                  <div className="dk-w-okwell">
                    <DeskIcon name="check" size={18} />
                  </div>
                  <b>{t("Nothing here needs you")}</b>
                  {t("Agents ask here before they change anything.")}
                </div>
              </div>
            )}
          </DeskListStill>
        ) : (
          <DeskList
            rows={listRows}
            handleRef={listRef}
            estimateSize={ROW_HEIGHT}
            initialHeight={INITIAL_LIST_HEIGHT}
            className={cn(settled && "dk-w-nq-settled")}
            role="listbox"
            aria-label={t("Things that need you")}
            aria-activedescendant={currentGroup ? rowId(currentGroup.key) : undefined}
            footer={
              query.data?.hasMore && (
                <p className="dk-w-quiet dk-w-nq-more">
                  {t(
                    "These are the soonest records. More wait after them; clear these to see the rest.",
                  )}
                </p>
              )
            }
          />
        )}
      </DeskListPane>
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
