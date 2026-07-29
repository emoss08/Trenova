import { useDispatchConsoleStore } from "@/stores/dispatch-console-store";
import { DragOverlay } from "@dnd-kit/core";
import { Suspense, lazy } from "react";
import type { DispatchActions } from "./use-dispatch-actions";

// Neither dialog is on the path to a rendered board: one needs a drop, the other a plan.
// Both stay out of the initial download until the dispatcher does that.
const PreflightDialog = lazy(() =>
  import("./preflight-dialog").then((m) => ({ default: m.PreflightDialog })),
);
const PlanReviewDialog = lazy(() =>
  import("./plan-review-dialog").then((m) => ({ default: m.PlanReviewDialog })),
);

/**
 * Everything that floats above the grid: the drag label, the pre-flight a drop opens, and
 * the auto-assign proposal. They are grouped here because they all read the console's
 * transient state rather than the board layout.
 */
export function ConsoleOverlays({ actions }: { actions: DispatchActions }) {
  const dragPreview = useDispatchConsoleStore.use.dragPreview();
  const preflight = useDispatchConsoleStore.use.preflight();
  const closePreflight = useDispatchConsoleStore.use.closePreflight();

  return (
    <>
      <DragOverlay dropAnimation={null}>
        {dragPreview && (
          <div className="flex h-6.5 cursor-grabbing items-center rounded border border-brand/50 bg-brand/15 px-2 shadow-lg backdrop-blur-sm">
            <span className="font-table text-[10px] font-semibold tabular-nums">{dragPreview}</span>
          </div>
        )}
      </DragOverlay>

      {preflight && (
        <Suspense fallback={null}>
          <PreflightDialog
            target={preflight}
            onCancel={closePreflight}
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
              closePreflight();
            }}
          />
        </Suspense>
      )}

      {actions.plan && (
        <Suspense fallback={null}>
          <PlanReviewDialog
            plan={actions.plan}
            onClose={actions.clearPlan}
            isAssigning={actions.isAssigning}
            onApply={(input) => {
              actions.assign(input);
              actions.clearPlan();
            }}
          />
        </Suspense>
      )}
    </>
  );
}
