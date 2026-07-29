import type { DispatchAssignmentPreview } from "@/lib/graphql/dispatch-console";
import { dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import type { PreflightTarget } from "@/stores/dispatch-console-store";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  formatClockDurationMs,
  formatUnixDateTimeShort,
  formatUnixTime,
} from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import {
  CircleCheckIcon,
  CircleHelpIcon,
  MoveRightIcon,
  OctagonXIcon,
  TriangleAlertIcon,
} from "lucide-react";
import { motion, useReducedMotion } from "motion/react";
import {
  formatMiles,
  formatMinutesSpan,
  formatMinutesToPickup,
  hosStrategyMeta,
  verdictMeta,
  workerInitials,
} from "./dispatch-vocabulary";
import { FindingList } from "./finding-list";
import { ScoreBreakdown } from "./score-breakdown";

const VERDICT_BANNER: Record<
  string,
  { Icon: typeof CircleCheckIcon; panelClass: string; iconClass: string }
> = {
  feasible: {
    Icon: CircleCheckIcon,
    panelClass: "bg-green-500/10",
    iconClass: "text-green-600 dark:text-green-400",
  },
  tight: {
    Icon: TriangleAlertIcon,
    panelClass: "bg-amber-500/10",
    iconClass: "text-amber-600 dark:text-amber-400",
  },
  infeasible: {
    Icon: OctagonXIcon,
    panelClass: "bg-red-500/10",
    iconClass: "text-red-600 dark:text-red-400",
  },
  unknown: {
    Icon: CircleHelpIcon,
    panelClass: "bg-muted/60",
    iconClass: "text-muted-foreground",
  },
};

/**
 * The one sentence a dispatcher needs before anything else: the hard blocker when
 * there is one, otherwise the rest plan that makes the trip legal, otherwise a plain
 * all-clear. Whatever leads here is not repeated in the finding list below.
 */
function verdictLead(preview: DispatchAssignmentPreview): {
  message: string;
  promotedFinding: string | null;
} {
  const block = preview.score.findings.find((finding) => finding.severity === "Block");
  if (block) {
    return { message: block.message, promotedFinding: `${block.code}-${block.field}` };
  }

  const strategy = hosStrategyMeta(preview.score.hosStrategy);
  if (strategy && preview.score.hosStrategy !== "currentClocks") {
    const deadline = preview.score.hosRestStartDeadline;
    const message =
      deadline > 0
        ? `${strategy.phrase} — rest must start by ${formatUnixTime(deadline)}`
        : strategy.phrase;
    return { message, promotedFinding: null };
  }

  switch (preview.score.verdict) {
    case "feasible":
      return { message: "No blockers found for this pairing.", promotedFinding: null };
    case "tight":
      return { message: "Margins are tight — review before assigning.", promotedFinding: null };
    case "unknown":
      return {
        message: "No hours-of-service feed for this driver; judged on schedule alone.",
        promotedFinding: null,
      };
    default:
      return { message: "This pairing cannot run as planned.", promotedFinding: null };
  }
}

function Stat({ value, label, tone }: { value: string; label: string; tone?: "late" }) {
  return (
    <div className="flex flex-col items-start gap-0.5 px-3 py-2">
      <span
        className={cn(
          "text-sm leading-none font-semibold tabular-nums",
          tone === "late" && "text-red-600 dark:text-red-400",
        )}
      >
        {value}
      </span>
      <span className="text-[9.5px] font-medium tracking-wider text-muted-foreground uppercase">
        {label}
      </span>
    </div>
  );
}

/**
 * The pre-flight is what makes drag-and-drop safe: nothing commits until a dispatcher has
 * seen the verdict, the empty miles, the appointment margin, and anything the pairing
 * trips. Where the organization enforces compliance as a hard block, the action is
 * labelled as an override rather than pretending it is routine.
 */
export function PreflightDialog({
  target,
  onCancel,
  onConfirm,
  isAssigning,
}: {
  target: PreflightTarget;
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
  const { move, driver } = target;
  const reducedMotion = useReducedMotion();
  const { data, isLoading } = useQuery(
    dispatchConsoleQueries.assignmentPreview({
      moveId: move.moveId,
      workerId: driver.workerId,
    }),
  );

  const verdict = data ? verdictMeta(data.score.verdict) : null;
  const banner = data ? (VERDICT_BANNER[data.score.verdict] ?? VERDICT_BANNER.unknown) : null;
  const lead = data ? verdictLead(data) : null;
  const tractorId = data?.tractorId ?? driver.tractorId;
  const canAssign = Boolean(tractorId) && !isLoading;

  const minutesToPickup = Math.round((move.originWindowStart - Date.now() / 1000) / 60);
  const slack = data?.score.minutesOfSlack ?? 0;
  const listedFindings =
    data?.score.findings.filter(
      (finding) => `${finding.code}-${finding.field}` !== lead?.promotedFinding,
    ) ?? [];

  const section = (index: number) =>
    reducedMotion
      ? {}
      : {
          initial: { opacity: 0, y: 4 },
          animate: { opacity: 1, y: 0 },
          transition: { duration: 0.25, delay: 0.05 * index, ease: [0.22, 1, 0.36, 1] as const },
        };

  return (
    <Dialog open onOpenChange={(open) => !open && onCancel()}>
      <DialogContent className="max-w-lg gap-0 p-0">
        <DialogHeader className="gap-3 border-b border-border px-5 pt-5 pb-4">
          <div className="flex items-center gap-3">
            <span
              aria-hidden
              className="flex size-9 shrink-0 items-center justify-center rounded-full bg-brand/10 text-xs font-semibold text-brand"
            >
              {workerInitials(driver.firstName, driver.lastName)}
            </span>
            <div className="flex min-w-0 flex-col gap-1">
              <DialogTitle className="truncate text-sm leading-none font-semibold">
                {move.isCovered ? "Reassign" : "Assign"} {driver.firstName} {driver.lastName}
              </DialogTitle>
              <div className="flex items-center gap-1.5 text-[11px] leading-none text-muted-foreground">
                {tractorId ? (
                  <span className="font-mono">{driver.tractorCode || tractorId}</span>
                ) : (
                  <span className="text-red-600 dark:text-red-400">No tractor assigned</span>
                )}
                {data?.trailerId ? (
                  <>
                    <span aria-hidden>·</span>
                    <span>trailer continues</span>
                  </>
                ) : null}
              </div>
            </div>
          </div>

          <div className="flex flex-col gap-2 rounded-md border border-border bg-muted/30 px-3 py-2.5">
            <div className="flex items-center justify-between gap-2 text-[10.5px] leading-none">
              <span className="flex min-w-0 items-center gap-1.5 text-muted-foreground">
                <span className="truncate font-mono font-medium text-foreground/80">
                  {move.proNumber}
                </span>
                {move.distance ? (
                  <span className="shrink-0">· {formatMiles(move.distance)}</span>
                ) : null}
              </span>
              {move.originWindowStart > 0 && (
                <span
                  className={cn(
                    "shrink-0 font-medium tabular-nums",
                    minutesToPickup < 0
                      ? "text-red-600 dark:text-red-400"
                      : "text-muted-foreground",
                  )}
                >
                  pickup {formatMinutesToPickup(minutesToPickup)}
                </span>
              )}
            </div>

            <div className="grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-x-3">
              <div className="flex min-w-0 flex-col gap-0.5">
                <span className="truncate text-xs leading-none font-semibold">
                  {move.originCity}, {move.originState}
                </span>
                <span className="text-[10.5px] tabular-nums text-muted-foreground">
                  {move.originWindowStart > 0
                    ? formatUnixDateTimeShort(move.originWindowStart)
                    : "Unscheduled"}
                </span>
              </div>
              <div aria-hidden className="flex items-center text-muted-foreground/50">
                <span className="w-6 border-t border-dashed border-current" />
                <MoveRightIcon className="size-3.5" />
              </div>
              <div className="flex min-w-0 flex-col items-end gap-0.5 text-right">
                <span className="truncate text-xs leading-none font-semibold">
                  {move.destinationCity}, {move.destinationState}
                </span>
                {move.destinationWindowStart > 0 ? (
                  <span className="text-[10.5px] tabular-nums text-muted-foreground">
                    {formatUnixDateTimeShort(move.destinationWindowStart)}
                  </span>
                ) : null}
              </div>
            </div>
          </div>
        </DialogHeader>

        {isLoading || !data ? (
          <div className="flex flex-col gap-2 px-5 py-4">
            <Skeleton className="h-12 rounded-md" />
            <Skeleton className="h-14 rounded-md" />
            <Skeleton className="h-28 rounded-md" />
          </div>
        ) : (
          <ScrollArea className="max-h-[55vh]">
            <div className="flex flex-col gap-3.5 px-5 py-4">
              {banner && verdict && lead ? (
                <motion.div
                  {...section(0)}
                  className={cn("flex items-start gap-2.5 rounded-md p-3", banner.panelClass)}
                >
                  <banner.Icon
                    className={cn("mt-px size-4 shrink-0", banner.iconClass)}
                    aria-hidden
                  />
                  <div className="flex min-w-0 flex-col gap-0.5">
                    <span className={cn("text-xs leading-none font-semibold", banner.iconClass)}>
                      {verdict.label}
                    </span>
                    <span className="text-[11px] leading-snug text-foreground/80">
                      {lead.message}
                    </span>
                  </div>
                </motion.div>
              ) : null}

              <motion.div
                {...section(1)}
                className="grid grid-cols-3 divide-x divide-border rounded-md border border-border"
              >
                <Stat value={formatMiles(data.score.deadheadMiles)} label="Empty miles" />
                <Stat
                  value={formatClockDurationMs(data.score.driveRemainingMs)}
                  label="Drive left"
                />
                <Stat
                  value={formatMinutesSpan(slack)}
                  label={slack < 0 ? "Late" : "Margin"}
                  tone={slack < 0 ? "late" : undefined}
                />
              </motion.div>

              {!tractorId ? (
                <motion.p {...section(2)} className="text-[11px] text-red-600 dark:text-red-400">
                  This driver has no tractor assigned; assign one before dispatching.
                </motion.p>
              ) : null}

              {listedFindings.length > 0 ? (
                <motion.div {...section(2)}>
                  <FindingList findings={listedFindings} />
                </motion.div>
              ) : null}

              <motion.div {...section(3)} className="border-t border-border pt-3.5">
                <ScoreBreakdown score={data.score.score} factors={data.score.factors} />
              </motion.div>
            </div>
          </ScrollArea>
        )}

        <DialogFooter className="border-t border-border px-5 py-3.5">
          <Button variant="outline" size="sm" onClick={onCancel}>
            Cancel
          </Button>
          <Button
            size="sm"
            variant={data?.requiresOverride ? "destructive" : "default"}
            disabled={!canAssign || isAssigning || Boolean(data?.requiresOverride)}
            isLoading={isAssigning}
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
                ? "Reassign driver"
                : "Assign driver"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
