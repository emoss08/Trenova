import { ResolveEventDialog } from "@/components/carrier-intelligence/resolve-event-dialog";
import { useCarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { useEventPresenter } from "@/components/carrier-intelligence/use-event-presenter";
import { useInfiniteScrollSentinel } from "@/hooks/use-infinite-scroll-sentinel";
import { useMediaQuery } from "@/hooks/use-media-query";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { isCarrierIntelNotFound } from "@/lib/carrier-intelligence";
import { fetchCarrierIntelEvent, type CarrierIntelEvent } from "@/lib/graphql/carrier-intelligence";
import { CARRIER_INTEL_EVENT_LIST_KEY } from "@/lib/graphql/carrier-monitoring-table";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@trenova/shared/components/ui/resizable";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { cn } from "@trenova/shared/lib/utils";
import { RefreshCwIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { useCallback, useMemo, useState, type ReactNode } from "react";
import { EventDetail } from "./event-detail";
import { EventList } from "./event-list";
import { groupInboxEvents } from "./group-events";
import {
  buildInboxQueryVariables,
  CLEARED_INBOX_FILTERS,
  hasInboxFilters,
  inboxSearchParams,
  type InboxFilterState,
  type InboxScope,
} from "./inbox-filters";
import { InboxToolbar, type InboxFilterPatch } from "./inbox-toolbar";
import { useAcknowledgeEvents, useInvalidateInbox } from "./use-event-actions";
import { useInboxEvents } from "./use-inbox-events";
import { useInboxKeyboard } from "./use-inbox-keyboard";

const WIDE_LAYOUT_QUERY = "(min-width: 1024px)";
const SKELETON_ROWS = 8;
const LOAD_AHEAD_ROWS = 5;
const DEEP_LINK_RETRIES = 2;

export type EventInboxProps = {
  canUpdate: boolean;
  scopeCounts: Partial<Record<InboxScope, number>>;
};

function InboxListSkeleton() {
  return (
    <div className="flex flex-col" aria-busy="true">
      {Array.from({ length: SKELETON_ROWS }, (_, index) => (
        <div key={index} className="flex h-14 items-center gap-3 border-b border-border/60 px-3">
          <Skeleton className="size-2 rounded-full" />
          <div className="flex flex-1 flex-col gap-1.5">
            <Skeleton className="h-3.5" style={{ width: `${40 + ((index * 17) % 45)}%` }} />
            <Skeleton className="h-3" style={{ width: `${25 + ((index * 11) % 30)}%` }} />
          </div>
        </div>
      ))}
    </div>
  );
}

function InboxSketch() {
  return (
    <div className="flex flex-col rounded-lg border">
      {[62, 48, 71].map((width) => (
        <div key={width} className="flex items-center gap-3 border-b px-3 py-3 last:border-b-0">
          <span className="bg-muted-foreground/20 size-2 rounded-full" />
          <div className="flex flex-1 flex-col gap-1.5">
            <GhostLine className={width > 60 ? "w-3/4" : "w-1/2"} />
            <GhostLine className="w-1/3" />
          </div>
        </div>
      ))}
    </div>
  );
}

function DetailSkeleton() {
  return (
    <div className="flex flex-col gap-3 px-5 py-4" aria-busy="true">
      <Skeleton className="h-3 w-24" />
      <Skeleton className="h-5 w-3/4" />
      <Skeleton className="h-4 w-1/2" />
      <div className="flex gap-2">
        <Skeleton className="h-8 w-28" />
        <Skeleton className="h-8 w-24" />
      </div>
    </div>
  );
}

function DetailPlaceholder({ children }: { children: ReactNode }) {
  return (
    <div className="text-muted-foreground flex h-full flex-col items-center justify-center gap-3 p-6 text-center text-xs">
      {children}
    </div>
  );
}

function KeyboardHints() {
  const t = useT();
  return (
    <div className="flex flex-wrap items-center justify-center gap-x-4 gap-y-2">
      <span className="flex items-center gap-1.5">
        <Kbd>J</Kbd>
        <Kbd>K</Kbd>
        {t("Move")}
      </span>
      <span className="flex items-center gap-1.5">
        <Kbd>Enter</Kbd>
        {t("Open")}
      </span>
      <span className="flex items-center gap-1.5">
        <Kbd>X</Kbd>
        {t("Select")}
      </span>
      <span className="flex items-center gap-1.5">
        <Kbd>E</Kbd>
        {t("Acknowledge")}
      </span>
    </div>
  );
}

export function EventInbox({ canUpdate, scopeCounts }: EventInboxProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const now = useNowSeconds();
  const isWide = useMediaQuery(WIDE_LAYOUT_QUERY);
  const [params, setParams] = useQueryStates(inboxSearchParams);

  const state = useMemo<InboxFilterState>(
    () => ({
      scope: params.scope,
      q: params.q,
      severity: params.severity,
      category: params.category,
      source: params.source,
      carrier: params.carrier,
    }),
    [params.carrier, params.category, params.q, params.scope, params.severity, params.source],
  );
  const variables = useMemo(() => buildInboxQueryVariables(state), [state]);
  const filtered = hasInboxFilters(state);

  const eventsQuery = useInboxEvents(variables);
  const { events, hasNextPage, isFetchingNextPage, fetchNextPage } = eventsQuery;

  const ruleCatalogQuery = useQuery({
    ...queries.carrierIntelSettings.settings(),
    select: (data) => data.carrierIntelRuleCatalog,
  });
  const ruleDescriptions = useMemo(
    () =>
      new Map((ruleCatalogQuery.data ?? []).map((rule) => [rule.code, rule.description] as const)),
    [ruleCatalogQuery.data],
  );
  const present = useEventPresenter();

  const groups = useMemo(
    () =>
      groupInboxEvents({
        events,
        grouping: params.group,
        now,
        labels: {
          today: t("Today"),
          yesterday: t("Yesterday"),
          usdot: (dotNumber) => t("USDOT {0}", dotNumber),
        },
      }),
    [events, now, params.group, t],
  );
  const ordered = useMemo(() => groups.flatMap((group) => group.events), [groups]);
  const byId = useMemo(() => new Map(ordered.map((event) => [event.id, event])), [ordered]);

  const [focusedCandidate, setFocusedId] = useState<string | null>(params.event);
  const [selectedCandidates, setSelectedIds] = useState<ReadonlySet<string>>(() => new Set());
  const [resolving, setResolving] = useState<CarrierIntelEvent | null>(null);

  const focusedId = focusedCandidate && byId.has(focusedCandidate) ? focusedCandidate : null;
  const selectedIds = useMemo<ReadonlySet<string>>(() => {
    const visible = new Set<string>();
    for (const id of selectedCandidates) {
      if (byId.has(id)) visible.add(id);
    }
    return visible;
  }, [byId, selectedCandidates]);

  const openId = params.event;
  const deepLinkId = openId && !eventsQuery.isPending && !byId.has(openId) ? openId : null;
  const deepLinkQuery = useQuery({
    queryKey: [CARRIER_INTEL_EVENT_LIST_KEY, "event", deepLinkId],
    queryFn: ({ signal }) => fetchCarrierIntelEvent(deepLinkId as string, { signal }),
    enabled: deepLinkId !== null,
    retry: (failureCount, error) =>
      !isCarrierIntelNotFound(error) && failureCount < DEEP_LINK_RETRIES,
  });
  const openEvent = openId
    ? (byId.get(openId) ?? (deepLinkId ? (deepLinkQuery.data ?? null) : null))
    : null;

  const invalidate = useInvalidateInbox();
  const { acknowledge, pending: acknowledging } = useAcknowledgeEvents();

  const setOpenEvent = useCallback(
    (id: string | null) => void setParams({ event: id }),
    [setParams],
  );

  const clearSelection = useCallback(() => setSelectedIds(new Set()), []);

  const updateFilters = useCallback(
    (patch: InboxFilterPatch) => {
      clearSelection();
      void setParams({
        ...(patch.scope !== undefined && {
          scope: patch.scope === "attention" ? null : patch.scope,
        }),
        ...(patch.q !== undefined && { q: patch.q.trim() === "" ? null : patch.q }),
        ...(patch.severity !== undefined && {
          severity: patch.severity.length > 0 ? [...patch.severity] : null,
        }),
        ...(patch.category !== undefined && {
          category: patch.category.length > 0 ? [...patch.category] : null,
        }),
        ...(patch.source !== undefined && {
          source: patch.source.length > 0 ? [...patch.source] : null,
        }),
        ...(patch.carrier !== undefined && { carrier: patch.carrier }),
      });
    },
    [clearSelection, setParams],
  );

  const clearFilters = useCallback(() => {
    clearSelection();
    void setParams(CLEARED_INBOX_FILTERS);
  }, [clearSelection, setParams]);

  const toggleSelected = useCallback((event: CarrierIntelEvent) => {
    setSelectedIds((current) => {
      const next = new Set(current);
      if (next.has(event.id)) {
        next.delete(event.id);
      } else {
        next.add(event.id);
      }
      return next;
    });
  }, []);

  const openRow = useCallback(
    (event: CarrierIntelEvent) => {
      setFocusedId(event.id);
      setOpenEvent(event.id);
    },
    [setOpenEvent],
  );

  const acknowledgeEvents = useCallback(
    async (targets: readonly CarrierIntelEvent[]) => {
      if (!canUpdate || targets.length === 0) return;
      const done = await acknowledge(targets);
      if (done) clearSelection();
    },
    [acknowledge, canUpdate, clearSelection],
  );

  const selectedEvents = useCallback(
    () => ordered.filter((event) => selectedIds.has(event.id)),
    [ordered, selectedIds],
  );

  useInboxKeyboard(resolving === null, {
    onMove: (delta) => {
      if (ordered.length === 0) return;
      const index = ordered.findIndex((event) => event.id === focusedId);
      const nextIndex = index === -1 ? 0 : Math.min(Math.max(index + delta, 0), ordered.length - 1);
      const next = ordered[nextIndex];
      setFocusedId(next.id);
      if (openId) setOpenEvent(next.id);
      if (hasNextPage && !isFetchingNextPage && nextIndex >= ordered.length - LOAD_AHEAD_ROWS) {
        void fetchNextPage();
      }
    },
    onToggleSelect: () => {
      const target = focusedId ? byId.get(focusedId) : undefined;
      if (target) toggleSelected(target);
    },
    onAcknowledge: () => {
      if (acknowledging) return;
      if (selectedIds.size > 0) {
        void acknowledgeEvents(selectedEvents());
        return;
      }
      const target = (focusedId ? byId.get(focusedId) : undefined) ?? openEvent;
      if (target) void acknowledgeEvents([target]);
    },
    onOpen: () => {
      if (focusedId) setOpenEvent(focusedId);
    },
    onClose: () => {
      if (openId) {
        setOpenEvent(null);
      } else if (selectedIds.size > 0) {
        clearSelection();
      }
    },
  });

  const sentinelRef = useInfiniteScrollSentinel<HTMLDivElement>({
    hasNextPage: Boolean(hasNextPage),
    isFetchingNextPage,
    onLoadMore: () => void fetchNextPage(),
  });

  const listFooter = isFetchingNextPage ? (
    <div className="text-muted-foreground flex h-12 items-center justify-center gap-2 text-xs">
      <Spinner className="size-3.5" />
      {t("Loading more")}
    </div>
  ) : hasNextPage ? (
    <div className="flex h-12 items-center justify-center">
      <Button
        type="button"
        variant="ghost"
        className="h-8 text-xs"
        onClick={() => void fetchNextPage()}
      >
        {t("Load more")}
      </Button>
    </div>
  ) : null;

  let listContent: ReactNode;
  if (eventsQuery.isPending) {
    listContent = <InboxListSkeleton />;
  } else if (eventsQuery.isError && events.length === 0) {
    listContent = (
      <div className="flex flex-col items-center gap-3 px-6 py-12 text-center">
        <p className="text-sm font-medium">{t("Changes could not be loaded")}</p>
        <p className="text-muted-foreground text-xs">
          {graphQLErrorMessage(eventsQuery.error, t("Try again in a moment."))}
        </p>
        <Button
          type="button"
          variant="outline"
          className="h-8 text-xs"
          onClick={() => void eventsQuery.refetch()}
        >
          <RefreshCwIcon className="size-3.5" />
          {t("Retry")}
        </Button>
      </div>
    );
  } else if (events.length === 0) {
    listContent = filtered ? (
      <EmptySheet
        title={t("Nothing matches")}
        description={t("No carrier change fits this search and these filters.")}
        sketch={<InboxSketch />}
        action={
          <Button type="button" variant="outline" className="h-8 text-xs" onClick={clearFilters}>
            {t("Clear filters")}
          </Button>
        }
      />
    ) : state.scope === "attention" ? (
      <EmptySheet
        title={t("You're all caught up")}
        description={t(
          "Authority, insurance and safety changes on monitored carriers land here as they are detected.",
        )}
        sketch={<InboxSketch />}
      />
    ) : (
      <EmptySheet
        title={state.scope === "all" ? t("No carrier changes yet") : t("Nothing here")}
        description={
          state.scope === "acknowledged"
            ? t("Changes someone has picked up but not resolved show up here.")
            : state.scope === "resolved"
              ? t("Changes that were resolved or dismissed show up here.")
              : t("Changes detected on monitored carriers show up here.")
        }
        sketch={<InboxSketch />}
      />
    );
  } else {
    listContent = (
      <EventList
        groups={groups}
        labels={labels}
        present={present}
        focusedId={focusedId}
        openId={openId}
        selectedIds={selectedIds}
        onRowClick={openRow}
        onToggleSelected={toggleSelected}
        sentinelRef={sentinelRef}
        footer={listFooter}
      />
    );
  }

  const closeDetail = (
    <Button
      type="button"
      variant="ghost"
      className="h-8 text-xs"
      onClick={() => setOpenEvent(null)}
    >
      {t("Close")}
    </Button>
  );

  let detail: ReactNode = null;
  if (openEvent) {
    detail = (
      <EventDetail
        key={openEvent.id}
        event={openEvent}
        presentation={present(openEvent)}
        labels={labels}
        ruleDescription={
          openEvent.ruleCode ? (ruleDescriptions.get(openEvent.ruleCode) ?? null) : null
        }
        canUpdate={canUpdate}
        acknowledging={acknowledging}
        onAcknowledge={(event) => void acknowledgeEvents([event])}
        onResolve={setResolving}
      />
    );
  } else if (deepLinkId) {
    if (deepLinkQuery.isPending) {
      detail = <DetailSkeleton />;
    } else if (deepLinkQuery.isError && !isCarrierIntelNotFound(deepLinkQuery.error)) {
      detail = (
        <DetailPlaceholder>
          <p className="text-foreground text-sm font-medium">
            {t("This change could not be loaded")}
          </p>
          <p>{graphQLErrorMessage(deepLinkQuery.error, t("Try again in a moment."))}</p>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              className="h-8 text-xs"
              onClick={() => void deepLinkQuery.refetch()}
            >
              <RefreshCwIcon className="size-3.5" />
              {t("Retry")}
            </Button>
            {closeDetail}
          </div>
        </DetailPlaceholder>
      );
    } else {
      detail = (
        <DetailPlaceholder>
          <p className="text-foreground text-sm font-medium">{t("Change not found")}</p>
          <p>{t("It may have been removed, or the link points to another organization.")}</p>
          {closeDetail}
        </DetailPlaceholder>
      );
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <InboxToolbar
        state={state}
        grouping={params.group}
        labels={labels}
        scopeCounts={scopeCounts}
        selectedCount={selectedIds.size}
        canUpdate={canUpdate}
        acknowledging={acknowledging}
        onChange={updateFilters}
        onGroupingChange={(group) => void setParams({ group: group === "day" ? null : group })}
        onClearFilters={clearFilters}
        onAcknowledgeSelected={() => void acknowledgeEvents(selectedEvents())}
        onClearSelection={clearSelection}
      />
      <div className="bg-background h-[calc(100dvh-22rem)] min-h-[30rem] overflow-hidden rounded-lg border">
        {isWide ? (
          <ResizablePanelGroup orientation="horizontal">
            <ResizablePanel defaultSize="44%" minSize="28%">
              <div
                className={cn(
                  "h-full overflow-y-auto transition-opacity",
                  eventsQuery.isPlaceholderData && "opacity-60",
                )}
              >
                {listContent}
              </div>
            </ResizablePanel>
            <ResizableHandle />
            <ResizablePanel defaultSize="56%" minSize="30%">
              <div className="h-full overflow-y-auto">
                {detail ??
                  (!openId && events.length > 0 ? (
                    <DetailPlaceholder>
                      <p>{t("Select a change to see what happened and act on it.")}</p>
                      <KeyboardHints />
                    </DetailPlaceholder>
                  ) : null)}
              </div>
            </ResizablePanel>
          </ResizablePanelGroup>
        ) : (
          <div
            className={cn(
              "h-full overflow-y-auto transition-opacity",
              eventsQuery.isPlaceholderData && "opacity-60",
            )}
          >
            {listContent}
          </div>
        )}
      </div>
      {!isWide ? (
        <Sheet
          open={detail !== null}
          onOpenChange={(open) => {
            if (!open) setOpenEvent(null);
          }}
        >
          <SheetContent className="w-full overflow-y-auto p-0 sm:max-w-lg">
            <SheetHeader className="sr-only">
              <SheetTitle>{openEvent ? present(openEvent).title : t("Carrier change")}</SheetTitle>
              <SheetDescription>
                {t("Details and actions for this carrier change")}
              </SheetDescription>
            </SheetHeader>
            {detail}
          </SheetContent>
        </Sheet>
      ) : null}
      <ResolveEventDialog
        event={resolving}
        title={resolving ? present(resolving).title : null}
        open={resolving !== null}
        onOpenChange={(open) => {
          if (!open) setResolving(null);
        }}
        onResolved={() => void invalidate()}
      />
    </div>
  );
}
