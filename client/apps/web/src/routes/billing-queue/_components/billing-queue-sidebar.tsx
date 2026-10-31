import { useT } from "@trenova/shared/i18n/use-t";
import { BillingListEmpty } from "@/components/billing/billing-empty";
import { usePermission } from "@/hooks/use-permission";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SelectionDock } from "@/components/selection-dock";
import { type VirtualRow, VirtualRows, type VirtualRowsHandle } from "@/components/virtual-rows";
import { useListSelection } from "@/hooks/use-list-selection";
import { billingQueueItemLabel } from "@/lib/billing-queue-label";
import type { DockAction } from "@trenova/shared/types/data-table";
import {
  BILLING_QUEUE_OPEN_LANES,
  type BillingQueueLane,
  type BillingQueueStats,
} from "@trenova/shared/types/billing-queue";
import type { BillingQueueRow } from "@/lib/graphql/billing-queue";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import {
  CheckCircleIcon,
  CheckDone01Icon,
  ChevronDownIcon,
  File06Icon,
  PauseIcon,
  Send01Icon,
  UserPlus01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { billingQueue } from "@/lib/queries/billing-queue";
import { useQueryStates } from "nuqs";
import { useDeferredValue, useEffect, useRef, useState } from "react";
import {
  type BillingQueueListFilters,
  billingQueueLaneQuery,
  billingQueueStatsQuery,
  laneItems,
  laneTotal,
} from "../billing-queue-queries";
import { queueSidebarSearchParamsParser } from "../use-billing-queue-state";
import { BillingQueueApproveConfirm } from "./billing-queue-approve-confirm";
import { BillingQueueAssignDialog, type BillingQueueTarget } from "./billing-queue-assign-dialog";
import { bulkEligibility } from "./billing-queue-bulk";
import { BillingQueueCancelDialog } from "./billing-queue-cancel-dialog";
import {
  activeQueueFilterCount,
  BillingQueueFilters,
  CLEARED_QUEUE_FILTERS,
} from "./billing-queue-filters";
import { BillingQueueHoldDialog } from "./billing-queue-hold-dialog";
import { BillingQueueItemCard } from "./billing-queue-item-card";
import { LANE_DOT, LANES_FOLDED_BY_DEFAULT, laneLabel } from "./billing-queue-lanes";
import { BillingQueueLaneSkeleton, BillingQueueListSkeleton } from "./billing-queue-list-skeleton";
import { BillingQueueSearch, isSentence } from "./billing-queue-search";
import { BillingQueueSuggestions } from "./billing-queue-suggestions";
import { useBulkRun } from "./use-bulk-run";

const itemId = (item: BillingQueueRow) => item.id;

const toTarget = (item: BillingQueueRow): BillingQueueTarget => ({
  id: item.id,
  label: billingQueueItemLabel(item),
});

type HoldRequest = { targets: BillingQueueTarget[]; note: string };

type OpenLane = (typeof BILLING_QUEUE_OPEN_LANES)[number];

function useLaneQuery(lane: BillingQueueLane, filters: BillingQueueListFilters, enabled: boolean) {
  return useInfiniteQuery({ ...billingQueueLaneQuery(lane, filters), enabled });
}

/**
 * The queue as lanes in working order: what needs the biller, what is ready to
 * approve, what waits on someone else, what is still being checked, what is held,
 * and what is approved but not posted. One virtualized list carries every lane,
 * each lane's heading pinned while its rows scroll under it; each lane pages on its
 * own, a folded lane fetches nothing until a filter needs its count, and a selection
 * spans lanes.
 */
export function BillingQueueSidebar({
  selectedItemId,
  onSelectItem,
  onOrderChange,
}: {
  selectedItemId: string | null;
  onSelectItem: (id: string) => void;
  /** The ids on screen, in order, so the detail pane can move to the next one. */
  onOrderChange?: (itemIds: string[]) => void;
}) {
  const t = useT();
  const { allowed: canPost } = usePermission(Resource.Invoice, Operation.Post);
  const [params, setParams] = useQueryStates(queueSidebarSearchParamsParser);
  const deferredSearch = useDeferredValue(params.query);
  const { data: assist } = useQuery(billingQueue.assist());
  const { data: stats } = useQuery(billingQueueStatsQuery());

  // A sentence waits for Enter, when Nova turns it into filters; searching the
  // server for it as typed would only ever empty the list.
  const search = assist?.search && isSentence(deferredSearch) ? "" : deferredSearch;
  const filters: BillingQueueListFilters = {
    status: params.status,
    billers: params.billers,
    billType: params.billType,
    payer: params.payer,
    search,
    includePosted: params.includePosted,
    minAmount: params.minAmount,
    maxAmount: params.maxAmount,
    createdBefore: params.createdBefore,
    shipmentCustomer: params.shipmentCustomer,
    delivered: params.delivered,
    pickedUp: params.pickedUp,
    queued: params.queued,
  };
  const filtered = activeQueueFilterCount(params) > 0 || search.trim() !== "";

  const [folded, setFolded] = useState<ReadonlySet<BillingQueueLane>>(
    () => new Set(LANES_FOLDED_BY_DEFAULT),
  );
  const visibleLanes = BILLING_QUEUE_OPEN_LANES.filter(
    (lane) => params.lanes.length === 0 || params.lanes.includes(lane),
  );
  // A lane asked for by name opens, whatever it was left as.
  const isOpen = (lane: OpenLane) => params.lanes.includes(lane) || !folded.has(lane);
  const shouldFetch = (lane: OpenLane) => visibleLanes.includes(lane) && (isOpen(lane) || filtered);

  const lanes = {
    NeedsYou: useLaneQuery("NeedsYou", filters, shouldFetch("NeedsYou")),
    Ready: useLaneQuery("Ready", filters, shouldFetch("Ready")),
    Waiting: useLaneQuery("Waiting", filters, shouldFetch("Waiting")),
    Checking: useLaneQuery("Checking", filters, shouldFetch("Checking")),
    Held: useLaneQuery("Held", filters, shouldFetch("Held")),
    Approved: useLaneQuery("Approved", filters, shouldFetch("Approved")),
  } satisfies Record<OpenLane, unknown>;

  const items = visibleLanes.filter(isOpen).flatMap((lane) => laneItems(lanes[lane].data));
  const selection = useListSelection(items, itemId);
  const eligible = bulkEligibility(selection.selected);
  const clearSelection = selection.clear;
  const bulk = useBulkRun();

  const [assignTargets, setAssignTargets] = useState<BillingQueueTarget[] | null>(null);
  const [cancelTargets, setCancelTargets] = useState<BillingQueueTarget[] | null>(null);
  const [holdRequest, setHoldRequest] = useState<HoldRequest | null>(null);
  const [approveConfirm, setApproveConfirm] = useState<BillingQueueRow[] | null>(null);

  const listHandle = useRef<VirtualRowsHandle>(null);
  useEffect(() => {
    if (selectedItemId) {
      listHandle.current?.scrollToKey(selectedItemId);
    }
  }, [selectedItemId]);

  const orderKey = items.map(itemId).join(",");
  useEffect(() => {
    onOrderChange?.(orderKey ? orderKey.split(",") : []);
  }, [orderKey, onOrderChange]);

  const approve = async (targets: BillingQueueRow[], assignMe: boolean) => {
    await bulk.start("Approve", targets.map(itemId), { assignApprover: assignMe });
    setApproveConfirm(null);
    clearSelection();
  };

  const dockActions: DockAction<BillingQueueRow>[] = [];
  if (selection.allState !== "all") {
    dockActions.push({
      id: "select-all",
      label: t("Select all {0}", items.length),
      icon: CheckDone01Icon,
      onClick: () => selection.setAll(true),
    });
  }
  if (eligible.approvable.length > 0) {
    dockActions.push({
      id: "approve",
      label: t("Approve {0}", eligible.approvable.length),
      icon: CheckCircleIcon,
      onClick: async () => {
        if (eligible.approvable.some((item) => !item.assignedBillerId)) {
          setApproveConfirm(eligible.approvable);
          return;
        }
        await approve(eligible.approvable, false);
      },
    });
  }
  if (canPost && eligible.postable.length > 0) {
    dockActions.push({
      id: "post",
      label: t("Post {0}", eligible.postable.length),
      icon: Send01Icon,
      onClick: async () => {
        await bulk.start("Post", eligible.postable.map(itemId));
        clearSelection();
      },
    });
  }
  if (eligible.holdable.length > 0) {
    dockActions.push({
      id: "hold",
      label: t("Hold"),
      icon: PauseIcon,
      onClick: () => setHoldRequest({ targets: eligible.holdable.map(toTarget), note: "" }),
    });
  }
  if (eligible.open.length > 0) {
    dockActions.push(
      {
        id: "assign",
        label: t("Assign"),
        icon: UserPlus01Icon,
        onClick: () => setAssignTargets(eligible.open.map(toTarget)),
      },
      {
        id: "paperwork",
        label: t("Ask for paperwork"),
        icon: File06Icon,
        onClick: async () => {
          await bulk.start("RequestPaperwork", eligible.open.map(itemId));
          clearSelection();
        },
      },
      {
        id: "cancel",
        label: t("Cancel {0}", eligible.open.length),
        icon: XCloseIcon,
        variant: "destructive",
        onClick: () => setCancelTargets(eligible.open.map(toTarget)),
      },
    );
  }
  const selectedTotal = selection.selected.reduce(
    (sum, item) => sum + Number(item.allocatedTotalAmount ?? 0),
    0,
  );

  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      if (event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement)
        return;
      if (items.length === 0 || (event.key !== "j" && event.key !== "k")) return;
      if (!selectedItemId) {
        onSelectItem(items[0].id);
        return;
      }
      const current = items.findIndex((item) => item.id === selectedItemId);
      const next =
        event.key === "j" ? Math.min(current + 1, items.length - 1) : Math.max(current - 1, 0);
      onSelectItem(items[next].id);
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [items, onSelectItem, selectedItemId]);

  const toggleLane = (lane: OpenLane) => {
    if (params.lanes.includes(lane)) {
      void setParams({ lanes: params.lanes.filter((value) => value !== lane) });
      return;
    }
    setFolded((current) => {
      const next = new Set(current);
      if (next.has(lane)) next.delete(lane);
      else next.add(lane);
      return next;
    });
  };

  function laneCount(lane: OpenLane): number | null {
    const loaded = laneTotal(lanes[lane].data);
    if (filtered) return loaded ?? null;
    // The stats leave out empty lanes, so a lane they don't name is empty.
    if (stats) return statFor(stats, lane)?.count ?? 0;
    return loaded ?? null;
  }

  const fetchedLanes = visibleLanes.filter(shouldFetch);
  const firstLoad = fetchedLanes.length > 0 && fetchedLanes.every((lane) => lanes[lane].isPending);
  const settled = fetchedLanes.every((lane) => lanes[lane].isSuccess);
  const emptyByFilter = filtered && settled && fetchedLanes.every((lane) => laneCount(lane) === 0);

  const suggestions = filtered ? null : (
    <BillingQueueSuggestions
      run={bulk.start}
      isStarting={bulk.isStarting}
      onOpenItem={onSelectItem}
      onHoldItems={(ids, note) =>
        setHoldRequest({ targets: ids.map((id) => ({ id, label: id })), note })
      }
    />
  );

  // One flat list for the virtualizer: each lane's heading stays pinned while its
  // rows scroll under it, and a lane with more to read ends in a row that loads it.
  const rows: VirtualRow[] = [];
  if (suggestions) {
    rows.push({ key: "suggestions", render: () => suggestions });
  }
  for (const lane of visibleLanes) {
    const count = laneCount(lane);
    if (count === 0 && (filtered || params.lanes.length === 0)) continue;
    const query = lanes[lane];
    const open = isOpen(lane);
    const total = filtered ? null : statFor(stats, lane)?.total;
    rows.push({
      key: `lane-${lane}`,
      sticky: true,
      render: () => (
        <div className="bg-card pt-1.5">
          <button
            type="button"
            onClick={() => toggleLane(lane)}
            aria-expanded={open}
            className="ui-inset-focus-ring hover:bg-surface-hover text-foreground flex w-full items-center gap-2 px-3.5 py-2 text-left text-sm"
          >
            <ChevronDownIcon
              className={cn(
                "text-foreground-subtle size-3.5 transition-transform",
                !open && "-rotate-90",
              )}
            />
            <span aria-hidden className={cn("size-1.5 rounded-full", LANE_DOT[lane])} />
            <span className="font-semibold">{laneLabel(lane, t)}</span>
            {count != null ? (
              <span className="text-foreground-muted font-mono text-xs">{count}</span>
            ) : null}
            <span className="flex-1" />
            {total != null ? (
              <span className="text-foreground-muted font-mono text-xs">
                {formatCurrency(Number(total))}
              </span>
            ) : null}
          </button>
        </div>
      ),
    });
    if (!open) continue;
    if (query.isPending) {
      rows.push({
        key: `loading-${lane}`,
        render: () => <BillingQueueLaneSkeleton rows={Math.min(count ?? 3, 3)} />,
      });
      continue;
    }
    const laneRows = laneItems(query.data);
    for (const item of laneRows) {
      rows.push({
        key: item.id,
        render: () => (
          <BillingQueueItemCard
            item={item}
            isSelected={item.id === selectedItemId}
            onClick={() => onSelectItem(item.id)}
            onAssignBiller={() => setAssignTargets([toTarget(item)])}
            onHold={() => setHoldRequest({ targets: [toTarget(item)], note: "" })}
            onCancel={() => setCancelTargets([toTarget(item)])}
            checked={selection.isSelected(item.id)}
            onCheckedChange={(checked) => selection.setSelected(item.id, checked)}
          />
        ),
      });
    }
    if (query.hasNextPage) {
      rows.push({
        key: `more-${lane}`,
        render: () => (
          <LaneMore
            hasNextPage={query.hasNextPage}
            isFetchingNextPage={query.isFetchingNextPage}
            isFetchNextPageError={query.isFetchNextPageError}
            fetchNextPage={query.fetchNextPage}
            remaining={count != null ? count - laneRows.length : null}
          />
        ),
      });
    }
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex shrink-0 items-start gap-1.5 border-b p-2.5">
        <BillingQueueSearch />
        <BillingQueueFilters />
      </div>
      {emptyByFilter || firstLoad ? (
        <ScrollArea maskVariant="card" className="min-h-0 flex-1">
          {suggestions}
          {emptyByFilter ? (
            <BillingListEmpty
              title="Nothing matches"
              description="No item in the queue fits the search and filters. Widen them, or clear them to see everything waiting."
              onClearFilters={() => void setParams(CLEARED_QUEUE_FILTERS)}
            />
          ) : (
            <BillingQueueListSkeleton />
          )}
        </ScrollArea>
      ) : (
        <VirtualRows
          rows={rows}
          estimateSize={56}
          initialHeight={720}
          mask="card"
          className="min-h-0 flex-1"
          aria-label={t("Billing queue")}
          handleRef={listHandle}
        />
      )}
      {assignTargets && (
        <BillingQueueAssignDialog
          open
          onOpenChange={(open) => !open && setAssignTargets(null)}
          targets={assignTargets}
          run={bulk.start}
          onDone={clearSelection}
        />
      )}
      {cancelTargets && (
        <BillingQueueCancelDialog
          open
          onOpenChange={(open) => !open && setCancelTargets(null)}
          targets={cancelTargets}
          run={bulk.start}
          onDone={clearSelection}
        />
      )}
      {holdRequest && (
        <BillingQueueHoldDialog
          targets={holdRequest.targets}
          defaultNote={holdRequest.note}
          run={bulk.start}
          onOpenChange={(open) => !open && setHoldRequest(null)}
          onDone={clearSelection}
        />
      )}
      {approveConfirm && (
        <BillingQueueApproveConfirm
          items={approveConfirm}
          isPending={bulk.isStarting}
          onOpenChange={(open) => !open && setApproveConfirm(null)}
          onApprove={(targets, assignMe) => void approve(targets, assignMe)}
        />
      )}
      <SelectionDock
        selectedCount={selection.selected.length}
        totals={[
          { id: "amount", label: t("Selected amount"), text: formatCurrency(selectedTotal) },
        ]}
        actions={dockActions}
        getSelected={() => selection.selected}
        onClearSelection={clearSelection}
      />
    </div>
  );
}

function statFor(stats: BillingQueueStats | undefined, lane: BillingQueueLane) {
  return stats?.lanes.find((entry) => entry.lane === lane);
}

/**
 * The end of a lane that has more to read. The list mounts only the rows near the
 * viewport, so this mounting means the biller has scrolled to the lane's end: it reads
 * the next page then, and again while it stays in view, so every open lane fills in as
 * it is reached rather than only the last. Its rows wait in the lane's own skeleton; a
 * page that fails waits on a retry instead of asking again on its own.
 */
function LaneMore({
  hasNextPage,
  isFetchingNextPage,
  isFetchNextPageError,
  fetchNextPage,
  remaining,
}: {
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  isFetchNextPageError: boolean;
  fetchNextPage: () => Promise<unknown>;
  remaining: number | null;
}) {
  const t = useT();

  useEffect(() => {
    if (hasNextPage && !isFetchingNextPage && !isFetchNextPageError) {
      void fetchNextPage();
    }
  }, [hasNextPage, isFetchingNextPage, isFetchNextPageError, fetchNextPage]);

  if (isFetchNextPageError) {
    return (
      <Button
        variant="ghost"
        size="xs"
        className="text-foreground-muted mt-0.5 mb-1 ml-7.5"
        onClick={() => void fetchNextPage()}
      >
        {t("Couldn't load more · Try again")}
      </Button>
    );
  }

  return <BillingQueueLaneSkeleton rows={Math.max(1, Math.min(remaining ?? 3, 3))} />;
}
