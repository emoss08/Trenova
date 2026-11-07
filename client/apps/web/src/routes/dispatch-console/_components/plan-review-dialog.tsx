import { formatCentsWhole } from "@trenova/shared/lib/utils";
import { useT } from "@trenova/shared/i18n/use-t";
import type {
  DispatchFinding,
  DispatchPlan,
  DispatchPlannedAssignment,
  DispatchUncoveredMove,
} from "@/lib/graphql/dispatch-console";
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
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { AlertTriangleIcon, SearchXIcon } from "@trenova/shared/components/icons";
import { useMemo, useState } from "react";
import { verdictMeta } from "./dispatch-vocabulary";
import { UNCOVERED_REASON_LABELS } from "./cost-vocabulary";
import { FindingList } from "./finding-list";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";

function PlannedAssignmentRow({ assignment }: { assignment: DispatchPlannedAssignment }) {
  const t = useT();

  const candidate = assignment.candidate;
  const verdict = verdictMeta(candidate.verdict);
  const next = assignment.alternatives[0];

  return (
    <div className="flex items-start gap-2 border-b px-3 py-2 last:border-b-0">
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <div className="flex items-center gap-1.5">
          <span className="truncate font-mono text-xs font-semibold">{assignment.proNumber}</span>
          <span className="text-muted-foreground text-xs" aria-hidden>
            →
          </span>
          <span className="truncate text-xs">{assignment.workerName}</span>
          <span className="ml-auto shrink-0 font-mono text-xs tabular-nums">
            {formatCentsWhole(assignment.planCostCents)}
          </span>
          <Badge variant={verdict.variant} className="h-4 shrink-0 rounded-md px-1 text-3xs">
            {t(verdict.label)}
          </Badge>
        </div>
        <span className="text-muted-foreground text-2xs">
          {next
            ? t(
                "Worth {0} to cover · {1} instead adds {2} · {3}% confidence",
                formatCentsWhole(assignment.coveragePriceCents),
                next.workerName,
                formatCentsWhole(next.extraCents),
                Math.round(assignment.confidence * 100),
              )
            : t(
                "Worth {0} to cover · {1}% confidence",
                formatCentsWhole(assignment.coveragePriceCents),
                Math.round(assignment.confidence * 100),
              )}
        </span>
        {candidate.blocked && <FindingList findings={candidate.findings} limit={2} />}
      </div>
    </div>
  );
}

type UncoveredGroup = {
  reason: DispatchUncoveredMove["reason"];
  findings: DispatchFinding[];
  proNumbers: string[];
};

/**
 * Twenty moves blocked by the same exhausted driver is one problem, not twenty. Uncovered
 * moves are grouped by why they went uncovered so the dispatcher reads each reason once and
 * sees exactly which freight it strands.
 */
function groupUncovered(uncovered: readonly DispatchUncoveredMove[]): UncoveredGroup[] {
  const groups = new Map<string, UncoveredGroup>();
  for (const move of uncovered) {
    const blocking = (move.bestCandidate?.findings ?? []).filter(
      (finding) => finding.severity === "Block",
    );
    const existing = groups.get(move.reason);
    if (existing) {
      existing.proNumbers.push(move.proNumber);
      for (const finding of blocking) {
        if (!existing.findings.some((known) => known.code === finding.code)) {
          existing.findings.push(finding);
        }
      }
    } else {
      groups.set(move.reason, {
        reason: move.reason,
        findings: [...blocking],
        proNumbers: [move.proNumber],
      });
    }
  }
  return [...groups.values()].sort((a, b) => b.proNumbers.length - a.proNumbers.length);
}

const VISIBLE_PRO_CHIPS = 6;

function UncoveredGroupCard({ group }: { group: UncoveredGroup }) {
  const t = useT();

  const [showAll, setShowAll] = useState(false);
  const shown = showAll ? group.proNumbers : group.proNumbers.slice(0, VISIBLE_PRO_CHIPS);
  const hidden = group.proNumbers.length - shown.length;

  return (
    <div className="flex flex-col gap-1.5 px-3 py-2.5">
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-xs font-medium">{UNCOVERED_REASON_LABELS[group.reason]}</span>
        <span className="text-muted-foreground shrink-0 text-2xs tabular-nums">
          {t("{0, plural, one {# move} other {# moves}}", group.proNumbers.length)}
        </span>
      </div>
      <FindingList findings={group.findings} limit={3} />
      <div className="flex flex-wrap items-center gap-1">
        {shown.map((proNumber) => (
          <span
            key={proNumber}
            className="border-border bg-muted/40 rounded-md border px-1.5 py-px font-mono text-2xs leading-4"
          >
            {proNumber}
          </span>
        ))}
        {hidden > 0 ? (
          <button
            type="button"
            className="text-muted-foreground hover:text-foreground px-1 text-2xs transition-colors"
            onClick={() => setShowAll(true)}
          >
            {t("+{0} more", hidden)}
          </button>
        ) : null}
      </div>
    </div>
  );
}

/**
 * A shadow plan: what Nova would have proposed while the organization watches it without
 * letting it act. Nothing was written, so nothing here writes; it is kept for review rather
 * than reduced to a toast, so the dispatcher can judge the pairings before trusting them.
 */
export function PlanReviewDialog({ plan, onClose }: { plan: DispatchPlan; onClose: () => void }) {
  const t = useT();

  const uncoveredGroups = useMemo(() => groupUncovered(plan.uncovered), [plan]);

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent size="lg" className="flex max-h-[80vh] flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-1.5">
            <NovaBadge size="md" />
            {t("What Nova would have proposed")}
          </DialogTitle>
          <DialogDescription className="flex flex-wrap items-center gap-x-1.5">
            <span className="text-foreground font-medium tabular-nums">
              {t("{0, plural, one {# pairing} other {# pairings}}", plan.assignments.length)}
            </span>
            {plan.uncovered.length > 0 && (
              <>
                <span aria-hidden>·</span>
                <span className="tabular-nums">{t("{0} not covered", plan.uncovered.length)}</span>
              </>
            )}
            <span aria-hidden>·</span>
            <span>{t("Shadow mode is on, so nothing was proposed or assigned.")}</span>
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="min-h-0 flex-1 rounded-md border">
          {plan.assignments.length === 0 ? (
            <div className="flex flex-col items-center gap-1.5 px-6 py-8 text-center">
              <SearchXIcon className="text-muted-foreground size-5" aria-hidden />
              <span className="text-xs font-medium">{t("No pairings to propose")}</span>
              <span className="text-muted-foreground max-w-sm text-xs leading-snug">
                {plan.uncovered.length > 0
                  ? t(
                      "Every candidate was blocked or out of range for the moves in this window. The reasons are grouped below.",
                    )
                  : t("No uncovered moves in this window.")}
              </span>
            </div>
          ) : (
            plan.assignments.map((assignment) => (
              <PlannedAssignmentRow key={assignment.moveId} assignment={assignment} />
            ))
          )}
          {uncoveredGroups.length > 0 && (
            <div className="flex flex-col border-t">
              <span className="bg-warning/[4%] text-warning flex items-center gap-1 border-b px-3 py-1.5 text-xs font-semibold">
                <AlertTriangleIcon className="size-3" aria-hidden />
                {t("Not covered")}
              </span>
              <div className="divide-border flex flex-col divide-y">
                {uncoveredGroups.map((group) => (
                  <UncoveredGroupCard key={group.reason + group.proNumbers[0]} group={group} />
                ))}
              </div>
            </div>
          )}
        </ScrollArea>

        <DialogFooter>
          <Button onClick={onClose}>{t("Close")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
