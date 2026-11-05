import type { DispatchBoardDriver, DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import { watchtower } from "@/lib/queries/watchtower";
import { useDispatchConsoleStore, type PreflightTarget } from "@/stores/dispatch-console-store";
import { DndContext, KeyboardSensor, PointerSensor, useSensor, useSensors } from "@dnd-kit/core";
import type { DragEndEvent, DragStartEvent } from "@dnd-kit/core";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  hasOrganizationCapability,
  OrganizationCapability,
} from "@trenova/shared/types/organization-capability";
import { Suspense, lazy, useCallback, useEffect, useMemo, useState } from "react";
import { DispatchTimeline, TimelinePane } from "./console-center-pane";
import { ConsoleOverlays } from "./console-overlays";
import { InspectorSkeleton } from "./console-skeletons";
import { ConsoleToolbar } from "./console-toolbar";
import { groupMovesByUrgency, URGENCY_ORDER } from "./dispatch-vocabulary";
import { QueuePane } from "./queue-pane";
import { SummaryStrip } from "./summary-strip";
import type { DispatchActions } from "./use-dispatch-actions";
import { focusQueueRow, stepMove, useDispatchHotkeys } from "./use-dispatch-hotkeys";
import {
  toDispatchBoardInput,
  useDispatchRail,
  useDispatchSelection,
  useDispatchView,
  useDispatchWindow,
} from "./url-state";

// The inspector only answers a question once something is selected, and it drags in the
// candidate scoring UI to do it, none of which the first paint of the queue needs.
const Inspector = lazy(() => import("./inspector").then((m) => ({ default: m.Inspector })));

const BOARD_REFETCH_MS = 60_000;
const EXCEPTIONS_REFETCH_MS = 60_000;
// Long enough for the collapse to play out before the row stops being drawn.
const DEPART_MS = 220;
// A move approved inside its undo window stays uncovered on the board until the approval
// runs; it is kept out of the queue that long, and no longer, so a write that never lands
// brings it back.
const DEPARTED_TTL_MS = 120_000;

// Stable identities for the empty case, so the memos below do not recompute every render.
const NO_MOVES: readonly DispatchBoardMove[] = [];
const NO_DRIVERS: readonly DispatchBoardDriver[] = [];

export function DispatchConsoleContent({ actions }: { actions: DispatchActions }) {
  const t = useT();
  const { range, filters, includeCovered, search, setSearch, shiftWindow, goToday, setAnchor } =
    useDispatchWindow();
  const { selectedMoveId, selectedWorkerId, selectMove, selectDriver, clearSelection } =
    useDispatchSelection();
  const { mode, tab, setTab, urgencyFocus, setUrgencyFocus } = useDispatchView();
  const { setCapacityFilter } = useDispatchRail();
  const capabilities = useOrgCapabilities();
  // Drivers are only listed for an organization that employs them.
  const showsDrivers = hasOrganizationCapability(
    capabilities,
    OrganizationCapability.AssetOperations,
  );

  const setDragPreview = useDispatchConsoleStore((s) => s.setDragPreview);
  const openPreflight = useDispatchConsoleStore((s) => s.openPreflight);
  const preflight = useDispatchConsoleStore((s) => s.preflight);
  const carrierAssignTarget = useDispatchConsoleStore((s) => s.carrierAssignTarget);
  const carrierCancelTarget = useDispatchConsoleStore((s) => s.carrierCancelTarget);
  const tenderTarget = useDispatchConsoleStore((s) => s.tenderTarget);
  const resetConsole = useDispatchConsoleStore((s) => s.reset);

  // A drag or a pending pre-flight belongs to the visit, not to the store's lifetime.
  useEffect(() => resetConsole, [resetConsole]);

  const [queueScope, setQueueScope] = useState<HTMLDivElement | null>(null);

  const boardInput = useMemo(() => toDispatchBoardInput(range, filters), [filters, range]);
  // The last board stays on screen while a new search or window loads, and while a poll
  // fails: the status line says it is not current rather than the queue going blank.
  const boardQuery = useQuery({
    ...dispatchConsoleQueries.board(boardInput),
    refetchInterval: BOARD_REFETCH_MS,
    placeholderData: keepPreviousData,
  });
  const board = boardQuery.data;
  const boardRead = {
    isPending: boardQuery.isPending,
    // Only a board that never arrived is drawn as a failure; a failed poll keeps the last
    // good board and turns the status line.
    isError: boardQuery.isError && !board,
    error: boardQuery.error,
    refetch: () => void boardQuery.refetch(),
  };

  const exceptionsQuery = useQuery({
    ...watchtower.dispatchExceptions(),
    refetchInterval: EXCEPTIONS_REFETCH_MS,
  });
  const exceptionItems = exceptionsQuery.data?.items ?? [];
  const exceptionsMore = exceptionsQuery.data?.hasNextPage ?? false;

  const moves = board?.moves ?? NO_MOVES;
  const drivers = board?.drivers ?? NO_DRIVERS;

  // Moves just covered here collapse out at once rather than waiting on the next read.
  const [departing, setDeparting] = useState<ReadonlySet<string>>(() => new Set());
  const [departed, setDeparted] = useState<ReadonlyMap<string, number>>(() => new Map());

  // A new board settles what was departing: a move it no longer holds needs no hiding, and
  // one hidden past its time comes back.
  const [prevMoves, setPrevMoves] = useState(moves);
  if (prevMoves !== moves) {
    setPrevMoves(moves);
    const present = new Set(moves.map((move) => move.moveId));
    const now = Date.now();
    if (departed.size > 0) {
      const kept = new Map(
        [...departed].filter(([id, at]) => present.has(id) && now - at < DEPARTED_TTL_MS),
      );
      if (kept.size !== departed.size) setDeparted(kept);
    }
  }

  const shownMoves = useMemo(
    () =>
      moves.filter(
        (move) =>
          !departed.has(move.moveId) && (urgencyFocus === null || move.urgency === urgencyFocus),
      ),
    [departed, moves, urgencyFocus],
  );

  // The order the queue draws, which is the order J and K walk; a move on its way out is
  // already behind the dispatcher.
  const orderedMoves = useMemo(() => {
    const grouped = groupMovesByUrgency(shownMoves);
    return URGENCY_ORDER.flatMap((bucket) => grouped.get(bucket) ?? []).filter(
      (move) => !departing.has(move.moveId),
    );
  }, [departing, shownMoves]);

  const boardMove = useMemo(
    () => moves.find((move) => move.moveId === selectedMoveId) ?? null,
    [moves, selectedMoveId],
  );
  // A selection can point off the board: a best-fit match, a link, a timeline bar, a
  // window that moved. It is read on its own and explained rather than dropped.
  const offBoard = Boolean(selectedMoveId) && !boardMove && boardQuery.isSuccess;
  const moveQuery = useQuery({
    ...dispatchConsoleQueries.move(selectedMoveId ?? ""),
    enabled: offBoard,
  });
  const selectedMove = boardMove ?? (offBoard ? (moveQuery.data ?? null) : null);
  const outsideWindow =
    offBoard &&
    selectedMove !== null &&
    (selectedMove.originWindowStart < range.start || selectedMove.originWindowStart >= range.end);

  const selectedDriver = useMemo(
    () => drivers.find((driver) => driver.workerId === selectedWorkerId) ?? null,
    [drivers, selectedWorkerId],
  );

  const inspectorOpen = Boolean(selectedMoveId) || selectedDriver !== null;

  const onCovered = useCallback(
    (moveId: string) => {
      const next =
        stepMove(orderedMoves, moveId, 1) === moveId
          ? stepMove(orderedMoves, moveId, -1)
          : stepMove(orderedMoves, moveId, 1);
      setDeparting((current) => new Set(current).add(moveId));
      window.setTimeout(() => {
        setDeparted((current) => new Map(current).set(moveId, Date.now()));
        setDeparting((current) => {
          const rest = new Set(current);
          rest.delete(moveId);
          return rest;
        });
      }, DEPART_MS);
      if (next && next !== moveId) {
        selectMove(next);
        focusQueueRow(next, queueScope);
      } else {
        clearSelection();
      }
    },
    [clearSelection, orderedMoves, queueScope, selectMove],
  );

  const undo = useCallback(() => {
    // Whatever undo reverses comes back into the queue from the next read, so nothing
    // stays hidden on the strength of a write that was taken back.
    setDeparted(new Map());
    setDeparting(new Set());
    void actions.undo();
  }, [actions]);

  const nextMove = useCallback(() => {
    const next = stepMove(orderedMoves, selectedMoveId, 1);
    if (next) {
      selectMove(next);
      focusQueueRow(next, queueScope);
    }
  }, [orderedMoves, queueScope, selectMove, selectedMoveId]);

  const handleDragStart = useCallback(
    (event: DragStartEvent) => {
      const data = event.active.data.current;
      if (data?.type === "move") {
        const move = data.move as DispatchBoardMove;
        setDragPreview(t("{0} · {1} → {2}", move.proNumber, move.originCity, move.destinationCity));
      }
    },
    [setDragPreview, t],
  );

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      setDragPreview(null);

      const activeData = event.active.data.current;
      const overData = event.over?.data.current;
      if (!activeData || !overData) return;

      // A drop on the timeline opens the pre-flight rather than committing: the
      // dispatcher sees the score, the empty miles and anything the pairing trips first,
      // including why a blocked driver can't take it.
      if (activeData.type !== "move" || overData.type !== "driver-target") return;
      const target: PreflightTarget = {
        move: activeData.move as DispatchBoardMove,
        driver: overData.driver as DispatchBoardDriver,
      };

      // A carrier-covered move cannot take a driver until its carrier assignment is
      // canceled; selecting it shows the coverage card with that action instead of a
      // pre-flight that is bound to fail.
      if (target.move.coverageType === "carrier") {
        selectMove(target.move.moveId);
        return;
      }

      openPreflight(target);
      selectMove(target.move.moveId);
    },
    [openPreflight, selectMove, setDragPreview],
  );

  const sensors = useSensors(
    // A small activation distance keeps a click on a row from being read as a drag.
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    // Space picks a move up; Enter stays the bar's own click, which opens it.
    useSensor(KeyboardSensor, {
      keyboardCodes: { start: ["Space"], cancel: ["Escape"], end: ["Space", "Enter"] },
    }),
  );

  const hasDialogOpen =
    Boolean(preflight) ||
    Boolean(carrierAssignTarget) ||
    Boolean(carrierCancelTarget) ||
    Boolean(tenderTarget) ||
    Boolean(actions.shadowPlan);

  useDispatchHotkeys({
    enabled: !hasDialogOpen,
    orderedMoves,
    selectedMoveId,
    canUndo: actions.canUndo,
    queueScope,
    onSelectMove: selectMove,
    onClearSelection: clearSelection,
    onShiftWindow: shiftWindow,
    onToday: goToday,
    onUndo: undo,
  });

  const proposalsWaiting = useMemo(
    () => moves.filter((move) => !move.isCovered && move.pendingProposal).length,
    [moves],
  );

  return (
    <DndContext
      sensors={sensors}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
      onDragCancel={() => setDragPreview(null)}
    >
      {/* A full-screen workspace, edge to edge as the design draws it: the toolbar and
          the figures are bars across the top, and the queue and the inspector fill the
          rest with a rule between them. */}
      <div className="flex min-h-135 flex-1 flex-col">
        <ConsoleToolbar
          freshness={{
            generatedAt: board?.generatedAt ?? 0,
            isError: boardQuery.isError,
            isFetching: boardQuery.isFetching,
            refetch: () => void boardQuery.refetch(),
          }}
        />

        <SummaryStrip
          summary={board?.summary}
          proposalsWaiting={proposalsWaiting}
          exceptions={
            exceptionsQuery.isSuccess
              ? { count: exceptionItems.length, more: exceptionsMore }
              : null
          }
          isPending={boardQuery.isPending}
          isError={boardRead.isError}
          urgencyFocus={urgencyFocus}
          onUrgencyFocus={(bucket) => {
            setTab("moves");
            setUrgencyFocus(bucket);
          }}
          onShowExceptions={() => setTab("exceptions")}
          onShowOpenDrivers={() => {
            setTab("drivers");
            setCapacityFilter("open");
          }}
          showsDrivers={showsDrivers}
        />

        <div className="flex min-h-0 flex-1">
          {mode === "queue" ? (
            <QueuePane
              tab={tab}
              onTab={setTab}
              moves={shownMoves}
              moveCount={board ? shownMoves.length - departing.size : null}
              read={boardRead}
              narrowed={
                search.trim() !== "" ||
                filters.fleetCodeIds.length > 0 ||
                filters.customerIds.length > 0
              }
              includeCovered={includeCovered}
              onClearSearch={() => setSearch("")}
              selectedMoveId={selectedMoveId}
              departingMoveIds={departing}
              onSelectMove={selectMove}
              outcome={actions.outcome}
              canUndo={actions.canUndo}
              onUndo={undo}
              exceptions={exceptionItems}
              driversTruncated={board?.driversTruncated ?? false}
              exceptionsRead={{
                isPending: exceptionsQuery.isPending,
                isError: exceptionsQuery.isError,
                error: exceptionsQuery.error,
                refetch: () => void exceptionsQuery.refetch(),
              }}
              exceptionsMore={exceptionsMore}
              drivers={drivers}
              selectedWorkerId={selectedWorkerId}
              onSelectDriver={selectDriver}
              showsDrivers={showsDrivers}
              listRef={setQueueScope}
            />
          ) : (
            <TimelinePane isLoading={boardQuery.isPending}>
              {(sheet) => (
                <DispatchTimeline
                  moves={moves}
                  drivers={drivers}
                  scale={sheet.scale}
                  today={sheet.today}
                  selectedMoveId={selectedMoveId}
                  selectedWorkerId={selectedWorkerId}
                  onSelectMove={selectMove}
                  onSelectDriver={selectDriver}
                />
              )}
            </TimelinePane>
          )}

          {/* Opens and closes by width, so the queue gives up and takes back the room
              rather than being covered. */}
          <aside
            aria-label={
              selectedMove
                ? t("Move {0}", selectedMove.proNumber)
                : selectedDriver
                  ? t("Driver {0} {1}", selectedDriver.firstName, selectedDriver.lastName)
                  : t("Inspector")
            }
            aria-hidden={!inspectorOpen}
            inert={!inspectorOpen}
            className={cn(
              "bg-background min-h-0 shrink-0 overflow-hidden border-l transition-[width,opacity,border-color] duration-320 ease-settle motion-reduce:transition-none",
              inspectorOpen
                ? "border-border w-100 opacity-100"
                : "w-0 border-transparent opacity-0",
            )}
          >
            <div className="flex h-full w-100 flex-col">
              {inspectorOpen ? (
                <Suspense fallback={<InspectorSkeleton />}>
                  <Inspector
                    move={selectedMove}
                    moveRead={
                      offBoard
                        ? {
                            isPending: moveQuery.isPending,
                            isError: moveQuery.isError,
                            error: moveQuery.error,
                            refetch: () => void moveQuery.refetch(),
                          }
                        : boardQuery.isPending
                          ? { isPending: true, isError: false, error: null, refetch: () => {} }
                          : null
                    }
                    outsideWindow={outsideWindow}
                    driver={selectedMove ? null : selectedDriver}
                    range={range}
                    hotkeysEnabled={!hasDialogOpen}
                    queueScope={queueScope}
                    onClose={clearSelection}
                    onSelectMove={selectMove}
                    onNextMove={nextMove}
                    onCovered={onCovered}
                    onJumpTo={(move) => setAnchor(new Date(move.originWindowStart * 1000))}
                    actions={actions}
                  />
                </Suspense>
              ) : null}
            </div>
          </aside>
        </div>
      </div>
      <ConsoleOverlays actions={actions} />
    </DndContext>
  );
}
