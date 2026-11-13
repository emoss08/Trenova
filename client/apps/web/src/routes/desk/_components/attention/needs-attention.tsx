import { useDeskSetting } from "@/stores/desk-settings-store";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import type { VirtualRow, VirtualRowsHandle } from "@/components/virtual-rows";
import { usePermission } from "@/hooks/use-permission";
import type { DeskAttentionGroup } from "@/lib/graphql/desk-attention";
import { isTypingTarget } from "@/lib/dom";
import { useDeskMarginStore } from "@/stores/desk-margin-store";
import { Button } from "@trenova/shared/components/ui/button";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useMemo, useRef, useState } from "react";
import { useQueryStates } from "nuqs";
import { useNavigate } from "react-router";
import { LANES, laneLabel, laneOf, minutesLeft, type Lane } from "../watchtower/lanes";
import { useWatchtowerActions } from "../watchtower/use-watchtower-actions";
import { casePath, START_CASE } from "./attention-record";
import { itemsWorkedElsewhere } from "./case-work";
import { AttentionRow } from "./attention-row";
import { ItemDetail } from "./item-detail";
import { NEEDS_DETAIL_TRANSITION_CLASS, NEEDS_LIST_TRANSITION_CLASS } from "../desk-motion";
import { DeskList, DeskListHeading, DeskListStill } from "../list/desk-list";
import { DeskListSkeleton } from "../list/desk-list-skeleton";
import { DeskViewDetail, DeskViewList } from "../list/desk-views";
import { LIST_SETTLED_CLASS, useListEntrance } from "../list/use-list-entrance";
import { attentionItems, type RecordLens } from "./lens";
import {
  NEEDS_ITEM_ADDRESS,
  NO_FILTERS,
  filterCount,
  isNarrowed,
  type NeedsFilters,
} from "./needs-filters";
import { useAttentionList } from "./use-attention-list";
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

function rowId(key: string) {
  return `dk-w-row-${key}`;
}

/**
 * Needs you by record, the list of the Everything and Watchtower lenses:
 * everything that waits on people, one row per record, soonest deadline
 * first, in lanes by how long is left. A shipment's late warning, the
 * customer's email about it and the plan to recover it are one row. The item
 * chosen is read in full beside the list and acted on from the bar under it,
 * with the same keys everywhere. A message is the exception: it shows here as
 * a signal and is worked in Triage, the one place mail is read and filed.
 *
 * It fills the two cells a lens owns in the page's frame (list/desk-views):
 * the list under the page's head, and the detail beside it. The head, the
 * search and the filters are the page's, which hands down what the list is
 * narrowed to.
 */
export function NeedsAttention({
  lens,
  shown,
  filters,
  onFilters,
  since,
  keysPaused,
}: {
  lens: RecordLens;
  /** Whether this lens is the one in front; it is kept, hidden, while Decisions is. */
  shown: boolean;
  /** What the page's head narrows the list to. */
  filters: NeedsFilters;
  onFilters: (filters: NeedsFilters) => void;
  /** The start of the reader's day, which "cleared today" counts from. */
  since: number;
  /** Set while the head's filter menu is open: the list's keys stand down. */
  keysPaused: boolean;
}) {
  const t = useT();
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const now = useNow("minute");
  const { allowed: canAct } = usePermission(Resource.Watchtower, Operation.Update);
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Update);
  // The item open beside the list lives in the address, as the lens, the
  // search and the filters do. So a link sent to someone opens on the same
  // item in the same list, and a reload comes back to it.
  const [address, setAddress] = useQueryStates(NEEDS_ITEM_ADDRESS, { history: "replace" });
  const narrowed = isNarrowed(filters);
  const askedItem = address.item;

  // The page's head reads the same list for what waits in each lens; this is
  // that read, not a second one.
  const query = useAttentionList(lens, useDeskSetting("cases") === "mine", filters);
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
    keysEnabled: canAct && !declineOpen && !keysPaused,
    deferSeen: filters.unseen,
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

  const openGroup = (group: DeskAttentionGroup) => {
    select(group.items[0].id);
    setShowList(false);
  };

  // The page's own keys: Enter for the case, A and D for a decision. A decision is never cleared with E or S: it is decided.
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (
        event.defaultPrevented ||
        event.metaKey ||
        event.ctrlKey ||
        event.altKey ||
        declineOpen ||
        keysPaused ||
        isTypingTarget(event.target)
      ) {
        return;
      }
      const key = event.key.toLowerCase();
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

  // Rows rise in when a lens is drawn; a row scrolled into view later mounts still.
  const settled = useListEntrance(lens, groups.length > 0, shown);
  const reading = shown && !showList && currentItem !== null;

  return (
    <>
      <DeskViewList className={NEEDS_LIST_TRANSITION_CLASS[lens]}>
        {query.isPending ? (
          <DeskListSkeleton label={t("Reading what needs you")} />
        ) : query.isError || groups.length === 0 ? (
          <DeskListStill>
            {query.isError ? (
              <div className="dk-w-empty">
                <div>
                  <b>{t("What needs you could not be read")}</b>
                  <Button size="sm" variant="secondary" onClick={() => void query.refetch()}>
                    {t("Try again")}
                  </Button>
                </div>
              </div>
            ) : narrowed ? (
              <div className="dk-w-empty">
                <div>
                  <b>
                    {filters.search !== ""
                      ? t("Nothing matches “{0}”", filters.search)
                      : t("Nothing matches these filters")}
                  </b>
                  {filters.search !== ""
                    ? t("Search reads what each item says and the record it is about.")
                    : t("Take a filter off to see more of what waits.")}
                  <div className="mt-3 flex justify-center">
                    <Button size="sm" variant="secondary" onClick={() => onFilters(NO_FILTERS)}>
                      {filterCount(filters) > 0 && filters.search !== ""
                        ? t("Clear the search and filters")
                        : filters.search !== ""
                          ? t("Clear the search")
                          : t("Clear filters")}
                    </Button>
                  </div>
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
            className={cn(
              settled && LIST_SETTLED_CLASS,
              "transition-opacity",
              query.isPlaceholderData && "opacity-60",
            )}
            role="listbox"
            aria-label={t("Things that need you")}
            aria-activedescendant={currentGroup ? rowId(currentGroup.key) : undefined}
            footer={
              query.data?.hasMore && (
                <p className="dk-w-quiet dk-w-nq-more">
                  {narrowed
                    ? t(
                        "These are the soonest records that match. More match after them; search or filter further to reach them.",
                      )
                    : t(
                        "These are the soonest records. More wait after them; clear these to see the rest.",
                      )}
                </p>
              )
            }
          />
        )}
      </DeskViewList>
      <DeskViewDetail reading={reading} className={NEEDS_DETAIL_TRANSITION_CLASS[lens]}>
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
          !query.isPending && (
            <div className="dk-w-empty">
              <div>{t("Pick something on the left, or press J.")}</div>
            </div>
          )
        )}
      </DeskViewDetail>
    </>
  );
}
