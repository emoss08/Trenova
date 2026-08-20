import type { DispatchBoardDriver, DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import type { TimeRange } from "@/lib/timeline/time-scale";
import { Suspense, lazy, useEffect } from "react";
import { CoverageKanbanSkeleton, DispatchTimelineSkeleton } from "./console-skeletons";
import type { TimelineZoomDays } from "./timeline-metrics";
import { useDispatchView, type CenterMode } from "./url-state";

// The two center views are mutually exclusive and neither is small — the kanban carries the
// urgency triage, the timeline carries the virtualizer and the whole time-scale renderer.
// Only the one being looked at is worth downloading.
const loadKanban = () => import("./coverage-kanban");
const loadTimeline = () => import("./dispatch-timeline");

const CoverageKanban = lazy(() => loadKanban().then((m) => ({ default: m.CoverageKanban })));
const DispatchTimeline = lazy(() => loadTimeline().then((m) => ({ default: m.DispatchTimeline })));

/** Warms the chunk on hover so the switch itself never waits on the network. */
export function preloadCenterView(mode: CenterMode): void {
  void (mode === "board" ? loadKanban() : loadTimeline());
}

export function ConsoleCenterPane({
  moves,
  drivers,
  isLoading,
  range,
  zoom,
  selectedMoveId,
  selectedWorkerId,
  onSelectMove,
  onSelectDriver,
}: {
  moves: readonly DispatchBoardMove[];
  drivers: readonly DispatchBoardDriver[];
  isLoading: boolean;
  range: TimeRange;
  zoom: TimelineZoomDays;
  selectedMoveId: string | null;
  selectedWorkerId: string | null;
  onSelectMove: (moveId: string) => void;
  onSelectDriver: (workerId: string) => void;
}) {
  const { mode, urgencyFocus } = useDispatchView();

  // The chunk downloads alongside the board query rather than after it, so a slow query
  // never hides a still-pending import.
  useEffect(() => preloadCenterView(mode), [mode]);

  return (
    <section className="bg-card flex min-h-0 flex-col overflow-hidden rounded-lg border">
      {mode === "board" ? (
        <div className="min-h-0 flex-1 p-2">
          <Suspense fallback={<CoverageKanbanSkeleton />}>
            <CoverageKanban
              moves={moves}
              isLoading={isLoading}
              urgencyFocus={urgencyFocus}
              selectedMoveId={selectedMoveId}
              onSelectMove={onSelectMove}
            />
          </Suspense>
        </div>
      ) : isLoading ? (
        // The timeline reads an empty driver list as "nothing to plan"; while the board is
        // still in flight that is a lie, so the skeleton stands in until it resolves.
        <DispatchTimelineSkeleton />
      ) : (
        <Suspense fallback={<DispatchTimelineSkeleton />}>
          <DispatchTimeline
            moves={moves}
            drivers={drivers}
            range={range}
            zoom={zoom}
            selectedMoveId={selectedMoveId}
            selectedWorkerId={selectedWorkerId}
            onSelectMove={onSelectMove}
            onSelectDriver={onSelectDriver}
          />
        </Suspense>
      )}
    </section>
  );
}
