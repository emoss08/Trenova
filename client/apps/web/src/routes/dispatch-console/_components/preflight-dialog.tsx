import type { DispatchBoardDriver, DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { queries } from "@/lib/queries";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatClockDurationMs, formatUnixDateTime } from "@trenova/shared/lib/date";
import { useQuery } from "@tanstack/react-query";
import { formatMiles, verdictMeta } from "./dispatch-vocabulary";
import { FindingList } from "./finding-list";
import { ScoreBreakdown } from "./score-breakdown";

export type PreflightTarget = {
  move: DispatchBoardMove;
  driver: DispatchBoardDriver;
};

/**
 * The pre-flight is what makes drag-and-drop safe: nothing commits until a dispatcher has
 * seen the score, the empty miles, the appointment margin, and anything the pairing
 * trips. Where the organization enforces compliance as a hard block, the action is
 * labelled as an override rather than pretending it is routine.
 */
export function PreflightDialog({
  target,
  onCancel,
  onConfirm,
  isAssigning,
}: {
  target: PreflightTarget | null;
  onCancel: () => void;
  onConfirm: (params: {
    moveId: string;
    workerId: string;
    tractorId: string;
    trailerId?: string;
    reassign: boolean;
  }) => void;
  isAssigning: boolean;
}) {
  const { data, isLoading } = useQuery({
    ...queries.dispatchConsole.assignmentPreview({
      moveId: target?.move.moveId ?? "",
      workerId: target?.driver.workerId ?? "",
    }),
    enabled: Boolean(target),
  });

  if (!target) return null;

  const { move, driver } = target;
  const verdict = data ? verdictMeta(data.score.verdict) : null;
  const tractorId = data?.tractorId ?? driver.tractorId;
  const canAssign = Boolean(tractorId) && !isLoading;

  return (
    <Dialog open onOpenChange={(open) => !open && onCancel()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="text-sm">
            Assign {driver.firstName} {driver.lastName} to {move.proNumber}
          </DialogTitle>
          <DialogDescription className="text-xs">
            {move.originCity}, {move.originState} → {move.destinationCity},{" "}
            {move.destinationState} · pickup {formatUnixDateTime(move.originWindowStart)}
          </DialogDescription>
        </DialogHeader>

        {isLoading || !data ? (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-20 rounded-md" />
            <Skeleton className="h-24 rounded-md" />
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            <div className="flex flex-wrap items-center gap-2">
              {verdict ? (
                <Badge variant={verdict.variant} className="h-5 rounded px-1.5 text-[10px]">
                  {verdict.label}
                </Badge>
              ) : null}
              <span className="text-[11px] text-muted-foreground">
                {formatMiles(data.score.deadheadMiles)} empty ·{" "}
                {formatClockDurationMs(data.score.driveRemainingMs)} drive left ·{" "}
                {data.score.minutesOfSlack < 0
                  ? `${Math.abs(data.score.minutesOfSlack)}m late`
                  : `${data.score.minutesOfSlack}m margin`}
              </span>
            </div>

            <div className="flex flex-wrap items-center gap-1.5">
              {tractorId ? (
                <Badge variant="outline" className="h-5 rounded px-1.5 font-mono text-[10px]">
                  Tractor {driver.tractorCode || tractorId}
                </Badge>
              ) : (
                <span className="text-[11px] text-red-600 dark:text-red-400">
                  This driver has no tractor assigned; assign one before dispatching.
                </span>
              )}
              {data.trailerId ? (
                <Badge variant="outline" className="h-5 rounded px-1.5 font-mono text-[10px]">
                  Trailer continues
                </Badge>
              ) : null}
            </div>

            <FindingList findings={data.score.findings} />

            <ScoreBreakdown score={data.score.score} factors={data.score.factors} />
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" size="sm" onClick={onCancel}>
            Cancel
          </Button>
          <Button
            size="sm"
            variant={data?.requiresOverride ? "destructive" : "default"}
            disabled={!canAssign || isAssigning || Boolean(data?.requiresOverride)}
            onClick={() =>
              onConfirm({
                moveId: move.moveId,
                workerId: driver.workerId,
                tractorId: tractorId ?? "",
                trailerId: data?.trailerId ?? undefined,
                reassign: move.isCovered,
              })
            }
          >
            {data?.requiresOverride
              ? "Blocked by policy"
              : move.isCovered
                ? "Reassign"
                : "Assign"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
