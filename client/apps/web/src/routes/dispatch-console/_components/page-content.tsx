import type {
  DispatchBoardDriver,
  DispatchBoardMove,
  DispatchCandidate,
} from "@/lib/graphql/dispatch-console";
import { dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import { useDispatchConsoleStore, type PreflightTarget } from "@/stores/dispatch-console-store";
import { DndContext, PointerSensor, useSensor, useSensors } from "@dnd-kit/core";
import type { DragEndEvent, DragStartEvent } from "@dnd-kit/core";
import { useQuery } from "@tanstack/react-query";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import { cn } from "@trenova/shared/lib/utils";
import {
  hasOrganizationCapability,
  OrganizationCapability,
} from "@trenova/shared/types/organization-capability";
import { Suspense, lazy, useCallback, useEffect, useMemo } from "react";
import { CapacityRail } from "./capacity-rail";
import { ConsoleCenterPane } from "./console-center-pane";
import { ConsoleOverlays } from "./console-overlays";
import { InspectorSkeleton } from "./console-skeletons";
import { ConsoleToolbar } from "./console-toolbar";
import { groupMovesByUrgency, URGENCY_ORDER } from "./dispatch-vocabulary";
import { SummaryStrip } from "./summary-strip";
import { useDispatchActions } from "./use-dispatch-actions";
import { useDispatchHotkeys } from "./use-dispatch-hotkeys";
import {
  useDispatchRail,
  useDispatchSelection,
  useDispatchView,
  useDispatchWindow,
} from "./url-state";

// The inspector only answers a question once something is selected, and it drags in the
// candidate scoring UI to do it — none of which the first paint of the board needs.
const Inspector = lazy(() => import("./inspector").then((m) => ({ default: m.Inspector })));

const BOARD_REFETCH_MS = 60_000;

// Stable identities for the empty case. Without them every render produces a new array and
// every memo below it recomputes, which on a board this size is not free.
const NO_MOVES: readonly DispatchBoardMove[] = [];
const NO_DRIVERS: readonly DispatchBoardDriver[] = [];

export function DispatchConsoleContent() {
  const { range, zoom, includeCovered, shiftWindow, goToday } = useDispatchWindow();
  const { selectedMoveId, selectedWorkerId, selectMove, selectDriver, clearSelection } =
    useDispatchSelection();
  const { urgencyFocus, setUrgencyFocus } = useDispatchView();
  const { setCapacityFilter } = useDispatchRail();
  const capabilities = useOrgCapabilities();
  // The rail is a column of driver rows with HOS gauges and drag handles. With no
  // drivers to put in it there is nothing to rail, so the column is removed
  // rather than left standing empty beside the board.
  const showsCapacityRail = hasOrganizationCapability(
    capabilities,
    OrganizationCapability.AssetOperations,
  );

  const setDragPreview = useDispatchConsoleStore.use.setDragPreview();
  const openPreflight = useDispatchConsoleStore.use.openPreflight();
  const preflight = useDispatchConsoleStore.use.preflight();
  const carrierAssignTarget = useDispatchConsoleStore.use.carrierAssignTarget();
  const carrierCancelTarget = useDispatchConsoleStore.use.carrierCancelTarget();
  const tenderTarget = useDispatchConsoleStore.use.tenderTarget();
  const resetConsole = useDispatchConsoleStore.use.reset();

  // A drag or a pending pre-flight belongs to the visit, not to the store's lifetime.
  useEffect(() => resetConsole, [resetConsole]);

  const boardInput = useMemo(
    () => ({ includeCovered, windowStart: range.start, windowEnd: range.end }),
    [includeCovered, range],
  );

  const { data: board, isLoading } = useQuery({
    ...dispatchConsoleQueries.board(boardInput),
    refetchInterval: BOARD_REFETCH_MS,
  });

  const actions = useDispatchActions();

  const moves = board?.moves ?? NO_MOVES;
  const drivers = board?.drivers ?? NO_DRIVERS;

  // The visual order of the kanban, reused by j/k navigation so the keyboard walks the
  // board in the same order the eye does.
  const orderedMoves = useMemo(() => {
    const grouped = groupMovesByUrgency(moves);
    return URGENCY_ORDER.flatMap((bucket) => grouped.get(bucket) ?? []);
  }, [moves]);

  const selectedMove = useMemo(
    () => moves.find((move) => move.moveId === selectedMoveId) ?? null,
    [moves, selectedMoveId],
  );
  const selectedDriver = useMemo(
    () => drivers.find((driver) => driver.workerId === selectedWorkerId) ?? null,
    [drivers, selectedWorkerId],
  );

  const assignCandidate = useCallback(
    (candidate: DispatchCandidate) => {
      if (!candidate.tractorId) return;
      const move = moves.find((item) => item.moveId === candidate.moveId);
      actions.assign([
        {
          moveId: candidate.moveId,
          primaryWorkerId: candidate.workerId,
          tractorId: candidate.tractorId,
          trailerId: candidate.trailerId ?? undefined,
          reassign: move?.isCovered ?? false,
        },
      ]);
    },
    [actions, moves],
  );

  const handleDragStart = useCallback(
    (event: DragStartEvent) => {
      const data = event.active.data.current;
      if (data?.type === "driver") {
        const driver = data.driver as DispatchBoardDriver;
        setDragPreview(
          `${driver.firstName} ${driver.lastName}${driver.tractorCode ? ` · ${driver.tractorCode}` : ""}`,
        );
        return;
      }
      if (data?.type === "move") {
        const move = data.move as DispatchBoardMove;
        setDragPreview(`${move.proNumber} · ${move.originCity} → ${move.destinationCity}`);
      }
    },
    [setDragPreview],
  );

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      setDragPreview(null);

      const activeData = event.active.data.current;
      const overData = event.over?.data.current;
      if (!activeData || !overData) return;

      // A drop opens the pre-flight rather than committing. The dispatcher sees the score,
      // the empty miles, and anything the pairing trips before anything is written.
      let target: PreflightTarget | null = null;
      if (activeData.type === "driver" && overData.type === "move-target") {
        target = {
          move: overData.move as DispatchBoardMove,
          driver: activeData.driver as DispatchBoardDriver,
        };
      } else if (activeData.type === "move" && overData.type === "driver-target") {
        target = {
          move: activeData.move as DispatchBoardMove,
          driver: overData.driver as DispatchBoardDriver,
        };
      }
      if (!target) return;

      // A carrier-covered move cannot take a driver until its carrier assignment is
      // canceled; selecting it surfaces the coverage card with that action instead of
      // opening a pre-flight that is guaranteed to fail.
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
  );

  const planForWindow = useCallback(
    () => actions.planAutoAssign({ windowStart: range.start, windowEnd: range.end }),
    [actions, range],
  );

  const hasDialogOpen =
    Boolean(preflight) ||
    Boolean(carrierAssignTarget) ||
    Boolean(carrierCancelTarget) ||
    Boolean(tenderTarget) ||
    Boolean(actions.plan && !actions.plan.shadowMode);

  useDispatchHotkeys({
    enabled: !hasDialogOpen,
    orderedMoves,
    selectedMoveId,
    canUndo: actions.canUndo,
    onSelectMove: selectMove,
    onClearSelection: clearSelection,
    onShiftWindow: shiftWindow,
    onToday: goToday,
    onUndo: actions.undo,
  });

  return (
    <DndContext sensors={sensors} onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
      <div className="flex h-[calc(100vh-9.5rem)] min-h-135 flex-col gap-3">
        <SummaryStrip
          summary={board?.summary}
          isLoading={isLoading}
          urgencyFocus={urgencyFocus}
          onUrgencyFocus={setUrgencyFocus}
          onCapacityFilter={setCapacityFilter}
        />

        <ConsoleToolbar
          canUndo={actions.canUndo}
          isAssigning={actions.isAssigning}
          isPlanning={actions.isPlanning}
          onUndo={actions.undo}
          onPlan={planForWindow}
        />

        <div
          className={cn(
            "grid min-h-0 flex-1 grid-cols-1 gap-3",
            showsCapacityRail
              ? "lg:grid-cols-[300px_minmax(0,1fr)_320px]"
              : "lg:grid-cols-[minmax(0,1fr)_320px]",
          )}
        >
          {showsCapacityRail && (
            <CapacityRail
              drivers={drivers}
              isLoading={isLoading}
              selectedWorkerId={selectedWorkerId}
              onSelectDriver={selectDriver}
            />
          )}

          <ConsoleCenterPane
            moves={moves}
            drivers={drivers}
            isLoading={isLoading}
            range={range}
            zoom={zoom}
            selectedMoveId={selectedMoveId}
            selectedWorkerId={selectedWorkerId}
            onSelectMove={selectMove}
            onSelectDriver={selectDriver}
          />

          <Suspense fallback={<InspectorSkeleton />}>
            <Inspector
              selectedMove={selectedMove}
              selectedDriver={selectedDriver}
              onAssign={assignCandidate}
              onSelectMove={selectMove}
              isAssigning={actions.isAssigning}
              hotkeysEnabled={!hasDialogOpen}
              actions={actions}
            />
          </Suspense>
        </div>
      </div>
      <ConsoleOverlays actions={actions} />
    </DndContext>
  );
}
