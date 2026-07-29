import type {
  DispatchPlan,
  DispatchPlannedAssignment,
  DispatchUncoveredMove,
} from "@/lib/graphql/dispatch-console";
import type { DispatchAssignMoveInput } from "@trenova/graphql/generated/graphql";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { cn } from "@trenova/shared/lib/utils";
import { SearchXIcon, SparklesIcon, TriangleAlertIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { formatMiles, scoreTone, verdictMeta } from "./dispatch-vocabulary";
import { FindingList } from "./finding-list";

function assignable(assignment: DispatchPlannedAssignment): boolean {
  return Boolean(assignment.tractorId) && !assignment.score.blocked;
}

function PlannedAssignmentRow({
  assignment,
  checked,
  onCheckedChange,
}: {
  assignment: DispatchPlannedAssignment;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}) {
  const verdict = verdictMeta(assignment.score.verdict);
  const canAssign = assignable(assignment);

  return (
    <label
      className={cn(
        "flex cursor-pointer items-start gap-2 border-b px-3 py-2 transition-colors last:border-b-0",
        checked ? "bg-brand/[4%]" : "hover:bg-muted/50",
        !canAssign && "cursor-not-allowed opacity-60",
      )}
    >
      <Checkbox
        className="mt-0.5"
        checked={checked}
        disabled={!canAssign}
        onCheckedChange={(value) => onCheckedChange(value === true)}
        aria-label={`Include ${assignment.proNumber} → ${assignment.workerName}`}
      />
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <div className="flex items-center gap-1.5">
          <span className="truncate font-mono text-xs font-semibold">{assignment.proNumber}</span>
          <span className="text-[11px] text-muted-foreground">→</span>
          <span className="truncate text-xs">{assignment.workerName}</span>
          <span
            className={cn(
              "ml-auto shrink-0 text-xs font-semibold tabular-nums",
              scoreTone(assignment.score.score),
            )}
          >
            {assignment.score.score}
          </span>
          <Badge variant={verdict.variant} className="h-4 shrink-0 rounded px-1 text-[9px]">
            {verdict.label}
          </Badge>
        </div>
        <span className="text-[10.5px] text-muted-foreground">
          {assignment.rationale}
          {assignment.score.deadheadMiles != null &&
            ` · ${formatMiles(assignment.score.deadheadMiles)} empty`}
          {` · ${Math.round(assignment.confidence * 100)}% confidence`}
        </span>
        {!assignment.tractorId && (
          <span className="text-[10px] text-red-600 dark:text-red-400">
            No tractor available for this driver — cannot be applied.
          </span>
        )}
        {assignment.score.blocked && <FindingList findings={assignment.score.findings} limit={2} />}
      </div>
    </label>
  );
}

type UncoveredGroup = {
  reason: string;
  findings: DispatchUncoveredMove["bestBlockedFindings"];
  proNumbers: string[];
};

/**
 * Twenty moves blocked by the same exhausted driver is one problem, not twenty. Uncovered
 * moves are grouped by their shared cause so the dispatcher reads each reason once and
 * sees exactly which freight it strands.
 */
function groupUncovered(uncovered: readonly DispatchUncoveredMove[]): UncoveredGroup[] {
  const groups = new Map<string, UncoveredGroup>();
  for (const move of uncovered) {
    const signature =
      move.reason +
      "|" +
      move.bestBlockedFindings.map((finding) => `${finding.code}:${finding.message}`).join("|");
    const existing = groups.get(signature);
    if (existing) {
      existing.proNumbers.push(move.proNumber);
    } else {
      groups.set(signature, {
        reason: move.reason,
        findings: move.bestBlockedFindings,
        proNumbers: [move.proNumber],
      });
    }
  }
  return [...groups.values()].sort((a, b) => b.proNumbers.length - a.proNumbers.length);
}

const VISIBLE_PRO_CHIPS = 6;

function UncoveredGroupCard({ group }: { group: UncoveredGroup }) {
  const [showAll, setShowAll] = useState(false);
  const shown = showAll ? group.proNumbers : group.proNumbers.slice(0, VISIBLE_PRO_CHIPS);
  const hidden = group.proNumbers.length - shown.length;

  return (
    <div className="flex flex-col gap-1.5 px-3 py-2.5">
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-[11px] font-medium">{group.reason}</span>
        <span className="shrink-0 text-[10px] tabular-nums text-muted-foreground">
          {group.proNumbers.length} move{group.proNumbers.length === 1 ? "" : "s"}
        </span>
      </div>
      <FindingList findings={group.findings} limit={3} />
      <div className="flex flex-wrap items-center gap-1">
        {shown.map((proNumber) => (
          <span
            key={proNumber}
            className="rounded border border-border bg-muted/40 px-1.5 py-px font-mono text-[10px] leading-4"
          >
            {proNumber}
          </span>
        ))}
        {hidden > 0 ? (
          <button
            type="button"
            className="px-1 text-[10px] text-muted-foreground transition-colors hover:text-foreground"
            onClick={() => setShowAll(true)}
          >
            +{hidden} more
          </button>
        ) : null}
      </div>
    </div>
  );
}

/**
 * The optimizer's plan is a proposal, never a fait accompli: every pairing is reviewed,
 * individually deselectable, and applied through the same bulk-assign path a manual
 * assignment uses.
 */
export function PlanReviewDialog({
  plan,
  onClose,
  onApply,
  isAssigning,
}: {
  plan: DispatchPlan;
  onClose: () => void;
  onApply: (input: DispatchAssignMoveInput[]) => void;
  isAssigning: boolean;
}) {
  const [checkedMoveIds, setCheckedMoveIds] = useState<ReadonlySet<string>>(new Set());

  useEffect(() => {
    setCheckedMoveIds(
      new Set(plan.assignments.filter(assignable).map((assignment) => assignment.moveId)),
    );
  }, [plan]);

  const selected = useMemo(
    () => plan.assignments.filter((assignment) => checkedMoveIds.has(assignment.moveId)),
    [plan, checkedMoveIds],
  );

  const uncoveredGroups = useMemo(() => groupUncovered(plan.uncovered), [plan]);

  if (plan.shadowMode) return null;
  if (plan.assignments.length === 0 && plan.uncovered.length === 0) return null;

  const nothingToApply = !plan.assignments.some(assignable);

  const toggle = (moveId: string, checked: boolean) => {
    setCheckedMoveIds((previous) => {
      const next = new Set(previous);
      if (checked) {
        next.add(moveId);
      } else {
        next.delete(moveId);
      }
      return next;
    });
  };

  const apply = () => {
    onApply(
      selected.map((assignment) => ({
        moveId: assignment.moveId,
        primaryWorkerId: assignment.workerId,
        tractorId: assignment.tractorId as string,
        trailerId: assignment.trailerId ?? undefined,
        reassign: false,
      })),
    );
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="flex max-h-[80vh] max-w-2xl flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-1.5 text-sm">
            <SparklesIcon className="size-4 text-brand" aria-hidden />
            Auto-assign proposal
          </DialogTitle>
          <DialogDescription className="flex flex-wrap items-center gap-x-1.5 text-xs">
            <span className="font-medium text-foreground tabular-nums">
              {plan.assignments.length} pairing{plan.assignments.length === 1 ? "" : "s"}
            </span>
            {plan.uncovered.length > 0 && (
              <>
                <span aria-hidden>·</span>
                <span className="tabular-nums">{plan.uncovered.length} not covered</span>
              </>
            )}
            <span aria-hidden>·</span>
            <span>Nothing is written until you apply.</span>
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="min-h-0 flex-1 rounded-md border">
          {plan.assignments.length === 0 ? (
            <div className="flex flex-col items-center gap-1.5 px-6 py-8 text-center">
              <SearchXIcon className="size-5 text-muted-foreground" aria-hidden />
              <span className="text-xs font-medium">No pairings to propose</span>
              <span className="max-w-sm text-[11px] leading-snug text-muted-foreground">
                Every candidate was blocked or out of range for the moves in this window. The
                reasons are grouped below.
              </span>
            </div>
          ) : (
            plan.assignments.map((assignment) => (
              <PlannedAssignmentRow
                key={assignment.moveId}
                assignment={assignment}
                checked={checkedMoveIds.has(assignment.moveId)}
                onCheckedChange={(checked) => toggle(assignment.moveId, checked)}
              />
            ))
          )}
          {uncoveredGroups.length > 0 && (
            <div className="flex flex-col border-t">
              <span className="flex items-center gap-1 border-b bg-warning/[4%] px-3 py-1.5 text-[10.5px] font-semibold tracking-wide text-warning uppercase">
                <TriangleAlertIcon className="size-3" aria-hidden />
                Not covered
              </span>
              <div className="flex flex-col divide-y divide-border">
                {uncoveredGroups.map((group) => (
                  <UncoveredGroupCard key={group.reason + group.proNumbers[0]} group={group} />
                ))}
              </div>
            </div>
          )}
        </ScrollArea>

        <DialogFooter>
          <Button variant="outline" size="sm" onClick={onClose}>
            {nothingToApply ? "Close" : "Discard plan"}
          </Button>
          {!nothingToApply && (
            <Button
              size="sm"
              disabled={selected.length === 0 || isAssigning}
              onClick={apply}
              isLoading={isAssigning}
            >
              Assign {selected.length} move{selected.length === 1 ? "" : "s"}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
