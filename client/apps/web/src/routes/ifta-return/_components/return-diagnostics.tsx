import { useT } from "@trenova/shared/i18n/use-t";
import { handleMutationError } from "@/hooks/use-api-mutation";
import {
  backfillJurisdictionMiles,
  type JurisdictionMilesBackfillResult,
} from "@/lib/graphql/ifta-return";
import {
  formatIftaMeasure,
  IFTA_MILES_DISPLAY_SCALE,
  problemTone,
  problemTotals,
  type IftaProblemTone,
  type IftaReturnView,
} from "@/lib/ifta-return";
import { useMutation } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { pluralize } from "@trenova/shared/lib/utils";
import type { BadgeVariant } from "@trenova/shared/types/badge";
import { IFTA_FUEL_TYPE_LABELS } from "@trenova/shared/types/fuel-ifta-enums";
import { CheckCircle2Icon, RouteIcon, SettingsIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";

export const DISTANCE_CONTROLS_PATH = "/admin/distance-controls";

const TONE_BADGE: Record<IftaProblemTone, BadgeVariant> = {
  danger: "inactive",
  warn: "warning",
  info: "outline",
};

const TONE_LABEL: Record<IftaProblemTone, string> = {
  danger: "Blocks filing",
  warn: "Check",
  info: "Note",
};

function ProblemDetail({ jurisdictionCode, fuelType, amount }: IftaReturnView["problems"][number]) {
  const parts: string[] = [];
  if (jurisdictionCode) parts.push(jurisdictionCode);
  if (fuelType) parts.push(IFTA_FUEL_TYPE_LABELS[fuelType]);
  if (amount) parts.push(formatIftaMeasure(amount, IFTA_MILES_DISPLAY_SCALE));
  return parts.length === 0 ? null : (
    <span className="text-muted-foreground text-xs">{parts.join(" · ")}</span>
  );
}

function CountFigure({
  label,
  count,
  miles,
  hint,
}: {
  label: string;
  count: number;
  miles: string;
  hint: string;
}) {
  const t = useT();

  return (
    <div className="bg-muted/30 rounded-lg border p-3" title={hint}>
      <p className="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
        {label}
      </p>
      <p className="mt-1 text-sm font-semibold tabular-nums">
        {count} {pluralize("move", count)}
      </p>
      <p className="text-muted-foreground mt-0.5 text-[11px] tabular-nums">
        {t("{0} miles", formatIftaMeasure(miles, IFTA_MILES_DISPLAY_SCALE))}
      </p>
    </div>
  );
}

type ReturnDiagnosticsProps = {
  ret: IftaReturnView;
  canBackfill: boolean;
};

export function ReturnDiagnostics({ ret, canBackfill }: ReturnDiagnosticsProps) {
  const t = useT();

  const [backfillOpen, setBackfillOpen] = useState(false);
  const mismatch = problemTotals(ret.problems, "MileageMismatch");
  const hasUnattributed = ret.unattributedMoveCount > 0;

  return (
    <Card className="rounded-md">
      <CardHeader className="flex flex-row items-start justify-between gap-3">
        <div className="flex flex-col gap-1">
          <CardTitle className="text-sm font-semibold">{t("What the figures leave out")}</CardTitle>
          <p className="text-muted-foreground text-xs">
            {t("Miles the return could not place, and everything the computation flagged while it ran.")}
          </p>
        </div>
        {canBackfill ? (
          <Button variant="outline" size="sm" onClick={() => setBackfillOpen(true)}>
            <RouteIcon className="size-3.5" />
            {t("Backfill jurisdiction miles…")}
          </Button>
        ) : null}
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-3">
          <CountFigure
            label={t("Unattributed")}
            count={ret.unattributedMoveCount}
            miles={ret.unattributedMiles}
            hint={t("Completed moves with routed distance but no jurisdiction breakdown. They are on no line, so the return is understated by these miles.")}
          />
          <CountFigure
            label={t("No tractor")}
            count={ret.noTractorMoveCount}
            miles={ret.noTractorMiles}
            hint={t("Attributed miles on moves with no tractor assignment, which cannot be placed on a fuel type.")}
          />
          <CountFigure
            label={t("Mileage mismatch")}
            count={mismatch.count}
            miles={mismatch.amount}
            hint={t("Where the jurisdiction rows do not add up to the move's own distance.")}
          />
        </div>

        {ret.problems.length === 0 ? (
          <div className="text-muted-foreground flex items-center gap-2 text-xs">
            <CheckCircle2Icon className="size-3.5 text-green-600 dark:text-green-400" />
            {t("The computation flagged nothing on this quarter.")}
          </div>
        ) : (
          <ul className="divide-border/60 divide-y rounded-md border">
            {ret.problems.map((problem, index) => {
              const tone = problemTone(problem.code);
              return (
                <li
                  key={`${problem.code}-${problem.jurisdictionCode ?? ""}-${problem.fuelType ?? ""}-${index}`}
                  className="flex flex-wrap items-center justify-between gap-2 px-3 py-2"
                >
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant={TONE_BADGE[tone]}>{TONE_LABEL[tone]}</Badge>
                    <span className="text-xs">{problem.message}</span>
                  </div>
                  <ProblemDetail {...problem} />
                </li>
              );
            })}
          </ul>
        )}

        {hasUnattributed ? (
          <Alert variant="warning">
            <SettingsIcon className="size-4" />
            <AlertTitle>{t("Some moves were never broken down by jurisdiction")}</AlertTitle>
            <AlertDescription>
              {t("Routed miles are only split state by state while")}{" "}
              <span className="font-medium">{t("Capture jurisdiction miles")}</span> {t("is on in")}{" "}
              <Link to={DISTANCE_CONTROLS_PATH} className="text-brand font-medium hover:underline">
                {t("distance controls")}
              </Link>
              {t(". Switch it on for future routes, and backfill the moves already run — each one is a billable distance request, so size the job with the dry run first.")}
            </AlertDescription>
          </Alert>
        ) : null}
      </CardContent>

      <BackfillMilesDialog open={backfillOpen} onOpenChange={setBackfillOpen} ret={ret} />
    </Card>
  );
}

type BackfillMilesDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  ret: IftaReturnView;
};

export function BackfillMilesDialog({ open, onOpenChange, ret }: BackfillMilesDialogProps) {
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("Backfill jurisdiction miles")}</DialogTitle>
          <DialogDescription>
            {t("Every completed move in {0} that has distance but no jurisdiction breakdown is re-routed for its state-by-state report. Each move is a billable distance request, so the quarter is counted first.", ret.period.label)}
          </DialogDescription>
        </DialogHeader>
        {open ? <BackfillSession ret={ret} onOpenChange={onOpenChange} /> : null}
      </DialogContent>
    </Dialog>
  );
}

function BackfillSession({
  ret,
  onOpenChange,
}: {
  ret: IftaReturnView;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  const [dryRun, setDryRun] = useState<JurisdictionMilesBackfillResult | null>(null);

  const count = useMutation({
    mutationFn: () =>
      backfillJurisdictionMiles({
        periodStart: ret.periodStart,
        periodEnd: ret.periodEnd,
        dryRun: true,
      }),
    onSuccess: (result) => setDryRun(result),
    onError: (error) => handleMutationError({ error, resourceName: "IFTA Return" }),
  });

  const start = useMutation({
    mutationFn: () =>
      backfillJurisdictionMiles({
        periodStart: ret.periodStart,
        periodEnd: ret.periodEnd,
        dryRun: false,
      }),
    onSuccess: (result) => {
      toast.success(t("Backfill started"), {
        description: result.workflowId
          ? `Workflow ${result.workflowId}. Recompute the return once it finishes.`
          : "Recompute the return once it finishes.",
      });
      onOpenChange(false);
    },
    onError: (error) => handleMutationError({ error, resourceName: "IFTA Return" }),
  });

  const { mutate: runCount } = count;
  useEffect(() => {
    runCount();
  }, [runCount]);

  const nothingToDo = dryRun !== null && dryRun.unattributedMoves === 0;

  return (
    <>
      <div className="rounded-md border p-3 text-sm">
        {count.isPending || dryRun === null ? (
          <span className="text-muted-foreground flex items-center gap-2 text-xs">
            <Spinner className="size-3.5" />
            {t("Counting the moves this would re-route…")}
          </span>
        ) : nothingToDo ? (
          <span className="text-xs">
            {t("Every completed move in the quarter already has a jurisdiction breakdown. There is nothing to backfill.")}
          </span>
        ) : (
          <div className="flex flex-col gap-1">
            <span className="font-semibold tabular-nums">
              {dryRun.unattributedMoves} {pluralize("move", dryRun.unattributedMoves)}
            </span>
            <span className="text-muted-foreground text-xs tabular-nums">
              {t("{0} miles would be attributed, at one billable distance request per move.", formatIftaMeasure(dryRun.unattributedMiles, IFTA_MILES_DISPLAY_SCALE))}
            </span>
          </div>
        )}
      </div>
      <DialogFooter className="mt-4">
        <Button
          type="button"
          variant="outline"
          onClick={() => onOpenChange(false)}
          disabled={start.isPending}
        >
          {t("Cancel")}
        </Button>
        <Button
          type="button"
          onClick={() => start.mutate()}
          disabled={count.isPending || dryRun === null || nothingToDo || start.isPending}
        >
          {start.isPending ? "Starting..." : "Start the backfill"}
        </Button>
      </DialogFooter>
    </>
  );
}
