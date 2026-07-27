import type {
  DispatchBoardDriver,
  DispatchBoardMove,
  DispatchCandidate,
} from "@/lib/graphql/dispatch-console";
import { queries } from "@/lib/queries";
import { DndContext, DragOverlay, PointerSensor, useSensor, useSensors } from "@dnd-kit/core";
import type { DragEndEvent, DragStartEvent } from "@dnd-kit/core";
import { Button } from "@trenova/shared/components/ui/button";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useQuery } from "@tanstack/react-query";
import { LayoutListIcon, SparklesIcon, Undo2Icon } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { CapacityRail } from "./capacity-rail";
import { CoverageBoard } from "./coverage-board";
import { Inspector } from "./inspector";
import { MetricsStrip } from "./metrics-strip";
import { PlanningBoard, type TimelineZoom } from "./planning-board";
import { PreflightDialog, type PreflightTarget } from "./preflight-dialog";
import { useDispatchActions } from "./use-dispatch-actions";

const BOARD_REFETCH_MS = 60_000;

// Stable identities for the empty case. Without them every render produces a new array and
// every memo below it recomputes, which on a board this size is not free.
const NO_MOVES: readonly DispatchBoardMove[] = [];
const NO_DRIVERS: readonly DispatchBoardDriver[] = [];

export function DispatchConsoleContent() {
  const [selectedMoveId, setSelectedMoveId] = useState<string | null>(null);
  const [selectedWorkerId, setSelectedWorkerId] = useState<string | null>(null);
  const [includeCovered, setIncludeCovered] = useState(false);
  const [showTimeline, setShowTimeline] = useState(true);
  const [zoom, setZoom] = useState<TimelineZoom>(12);
  const [preflight, setPreflight] = useState<PreflightTarget | null>(null);
  const [draggingDriver, setDraggingDriver] = useState<DispatchBoardDriver | null>(null);

  const boardInput = useMemo(() => ({ includeCovered }), [includeCovered]);

  const { data: board, isLoading } = useQuery({
    ...queries.dispatchConsole.board(boardInput),
    refetchInterval: BOARD_REFETCH_MS,
  });

  const actions = useDispatchActions();

  const moves = board?.moves ?? NO_MOVES;
  const drivers = board?.drivers ?? NO_DRIVERS;

  const selectedMove = useMemo(
    () => moves.find((move) => move.moveId === selectedMoveId) ?? null,
    [moves, selectedMoveId],
  );
  const selectedDriver = useMemo(
    () => drivers.find((driver) => driver.workerId === selectedWorkerId) ?? null,
    [drivers, selectedWorkerId],
  );

  const selectMove = useCallback((moveId: string) => {
    setSelectedMoveId(moveId);
    setSelectedWorkerId(null);
  }, []);

  const selectDriver = useCallback((workerId: string) => {
    setSelectedWorkerId(workerId);
    setSelectedMoveId(null);
  }, []);

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

  const handleDragStart = useCallback((event: DragStartEvent) => {
    const data = event.active.data.current;
    if (data?.type === "driver") {
      setDraggingDriver(data.driver as DispatchBoardDriver);
    }
  }, []);

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      setDraggingDriver(null);

      const driver = event.active.data.current?.driver as DispatchBoardDriver | undefined;
      const move = event.over?.data.current?.move as DispatchBoardMove | undefined;
      if (!driver || !move) return;

      // A drop opens the pre-flight rather than committing. The dispatcher sees the score,
      // the empty miles, and anything the pairing trips before anything is written.
      setPreflight({ move, driver });
      setSelectedMoveId(move.moveId);
      setSelectedWorkerId(null);
    },
    [],
  );

  const sensors = useSensors(
    // A small activation distance keeps a click on a driver row from being read as a drag.
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
  );

  // Dispatchers cover a board faster on the keyboard than with a mouse. Drag-and-drop is
  // the discoverable path; these are the fast one.
  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      const target = event.target as HTMLElement | null;
      if (target && ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName)) return;
      if (event.metaKey || event.ctrlKey || event.altKey) return;

      if (event.key === "u" && actions.canUndo) {
        event.preventDefault();
        actions.undo();
        return;
      }

      if (event.key === "j" || event.key === "k") {
        if (moves.length === 0) return;
        event.preventDefault();
        const index = moves.findIndex((move) => move.moveId === selectedMoveId);
        const delta = event.key === "j" ? 1 : -1;
        const next = index === -1 ? 0 : Math.min(Math.max(index + delta, 0), moves.length - 1);
        selectMove(moves[next].moveId);
      }
    }

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [actions, moves, selectMove, selectedMoveId]);

  return (
    <DndContext sensors={sensors} onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
      <div className="flex flex-col gap-3">
        <MetricsStrip summary={board?.summary} isLoading={isLoading} />

        <div className="flex flex-wrap items-center gap-3">
          <label className="flex items-center gap-1.5 text-[11px]">
            <Switch checked={includeCovered} onCheckedChange={setIncludeCovered} />
            Show covered moves
          </label>
          <label className="flex items-center gap-1.5 text-[11px]">
            <Switch checked={showTimeline} onCheckedChange={setShowTimeline} />
            Planning board
          </label>

          <div className="ml-auto flex items-center gap-1.5">
            <Button
              size="sm"
              variant="outline"
              className="h-7 gap-1 px-2 text-[11px]"
              disabled={!actions.canUndo || actions.isAssigning}
              onClick={actions.undo}
            >
              <Undo2Icon className="size-3" aria-hidden />
              Undo
              <kbd className="ml-0.5 rounded border border-border px-1 font-mono text-[9px]">u</kbd>
            </Button>
            <Button
              size="sm"
              variant="outline"
              className="h-7 gap-1 px-2 text-[11px]"
              disabled={actions.isPlanning}
              onClick={() => actions.planAutoAssign({})}
            >
              <SparklesIcon className="size-3" aria-hidden />
              {actions.isPlanning ? "Planning…" : "Auto-assign"}
            </Button>
          </div>
        </div>

        {actions.plan && actions.plan.uncovered.length > 0 ? (
          <div className="flex items-start gap-2 rounded-md border border-amber-500/40 bg-amber-500/5 p-2">
            <LayoutListIcon className="mt-0.5 size-3.5 shrink-0 text-amber-600" aria-hidden />
            <div className="flex flex-col gap-0.5">
              <span className="text-[11px] font-medium">
                {actions.plan.uncovered.length} move(s) could not be covered
              </span>
              <span className="text-[10px] text-muted-foreground">
                {actions.plan.uncovered[0].proNumber}: {actions.plan.uncovered[0].reason}
              </span>
            </div>
          </div>
        ) : null}

        <div className="grid min-h-[32rem] gap-3 lg:grid-cols-[18rem_minmax(0,1fr)_20rem]">
          <CapacityRail
            drivers={drivers}
            isLoading={isLoading}
            selectedWorkerId={selectedWorkerId}
            onSelectDriver={selectDriver}
          />
          <CoverageBoard
            moves={moves}
            isLoading={isLoading}
            selectedMoveId={selectedMoveId}
            onSelectMove={selectMove}
          />
          <Inspector
            selectedMove={selectedMove}
            selectedDriver={selectedDriver}
            onAssign={assignCandidate}
            onSelectMove={selectMove}
            isAssigning={actions.isAssigning}
          />
        </div>

        {showTimeline && board ? (
          <PlanningBoard
            drivers={drivers}
            windowStart={board.windowStart}
            zoom={zoom}
            onZoomChange={setZoom}
            onSelectMove={selectMove}
          />
        ) : null}
      </div>

      <DragOverlay>
        {draggingDriver ? (
          <div className="rounded-md border border-brand bg-card px-2 py-1 text-xs shadow-lg">
            {draggingDriver.firstName} {draggingDriver.lastName}
            {draggingDriver.tractorCode ? (
              <span className="ml-1 font-mono text-[10px] text-muted-foreground">
                {draggingDriver.tractorCode}
              </span>
            ) : null}
          </div>
        ) : null}
      </DragOverlay>

      <PreflightDialog
        target={preflight}
        onCancel={() => setPreflight(null)}
        isAssigning={actions.isAssigning}
        onConfirm={(params) => {
          actions.assign([
            {
              moveId: params.moveId,
              primaryWorkerId: params.workerId,
              tractorId: params.tractorId,
              trailerId: params.trailerId,
              reassign: params.reassign,
            },
          ]);
          setPreflight(null);
        }}
      />
    </DndContext>
  );
}
