import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  resetDispatchStyle,
  type DispatchStyle,
  type DispatchStyleWeight,
} from "@/lib/graphql/dispatch-style";
import { dispatchStyle } from "@/lib/queries/dispatch-style";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { formatNumber } from "@trenova/shared/i18n/format";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useRef, useState } from "react";
import { LearnedStyleSkeleton } from "./learned-style-skeleton";

const SAME_AS_DEFAULT = 0.05;

const COMPONENT_LABELS = defineLabels({
  homeTime: "Time away from home",
  fairness: "Miles above the fleet median",
  customerExperience: "Customer experience",
  onTimeHistory: "On-time history",
  safety: "Safety history",
  acceptance: "Acceptance",
  trailerContinuity: "Trailer continuity",
  haulFit: "Haul fit",
  timeOffBuffer: "Time-off buffer",
});

/**
 * How the organization's dispatchers weigh each preference against its costs, learned from
 * the drivers they assign against the plan, with a way to start over. It draws no frame of
 * its own: it sits in the Learning tab's panel, under the switch that turns learning on.
 */
export function LearnedStyle() {
  const t = useT();
  const { data, isPending, isError, refetch, isRefetching } = useQuery(dispatchStyle.get());

  if (isPending) {
    return <LearnedStyleSkeleton />;
  }

  if (isError) {
    return (
      <Alert size="sm" variant="destructive">
        <AlertDescription className="flex items-center justify-between gap-3">
          <span>{t("The learned dispatch style could not be loaded.")}</span>
          <Button
            type="button"
            variant="outline"
            size="sm"
            isLoading={isRefetching}
            onClick={() => void refetch()}
          >
            {t("Try again")}
          </Button>
        </AlertDescription>
      </Alert>
    );
  }

  return <LearnedStyleBody style={data} />;
}

function LearnedStyleBody({ style }: { style: DispatchStyle }) {
  const t = useT();
  const learned = style.decisions > 0 || style.weights.length > 0;

  return (
    <>
      {!style.enabled && (
        <p className="text-sm text-muted-foreground">
          {learned
            ? t(
                "Learning is off, so the planner uses your own costs. What it learned is kept and comes back if you turn learning on.",
              )
            : t("Learning is off. Turn on Learn our dispatch style to start learning.")}
        </p>
      )}

      {learned ? (
        <LearnedWeights style={style} />
      ) : (
        style.enabled && (
          <p className="text-sm text-muted-foreground">
            {t(
              "Nothing learned yet. Each driver a dispatcher assigns against the plan teaches the planner a little, and the weights appear here after the first decisions.",
            )}
          </p>
        )
      )}

      <ResetStyle disabled={!learned} />
    </>
  );
}

function LearnedWeights({ style }: { style: DispatchStyle }) {
  const t = useT();

  return (
    <div className="flex flex-col gap-3">
      <p className="text-sm">
        {t(
          "Learned from {0, plural, one {# decision} other {# decisions}}: {1} took the plan's driver, {2} chose a driver it priced and {3} chose one it did not.",
          style.decisions,
          formatNumber(style.approvals),
          formatNumber(style.overrides),
          formatNumber(style.declines),
        )}
      </p>
      {style.weights.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          {t("Your dispatchers' choices so far match your costs; no weight has moved.")}
        </p>
      ) : (
        <ul className="flex flex-col divide-y divide-border" aria-label={t("Learned weights")}>
          {style.weights.map((weight) => (
            <WeightRow key={weight.component} weight={weight} />
          ))}
        </ul>
      )}
    </div>
  );
}

function WeightRow({ weight }: { weight: DispatchStyleWeight }) {
  const t = useT();
  const label = COMPONENT_LABELS[weight.component];
  const same = Math.abs(weight.multiplier - 1) < SAME_AS_DEFAULT;
  const multiplier = formatNumber(weight.multiplier, {
    minimumFractionDigits: 1,
    maximumFractionDigits: 2,
  });

  return (
    <li className="flex items-baseline justify-between gap-4 py-2">
      <span className="text-sm font-medium">{label}</span>
      <span className="flex flex-col items-end text-right">
        <span className="text-sm tabular-nums">
          {same ? t("As your cost") : t("{0}× your cost", multiplier)}
        </span>
        <span className="text-xs text-muted-foreground">
          {t("learned from {0, plural, one {# decision} other {# decisions}}", weight.samples)}
        </span>
      </span>
    </li>
  );
}

function ResetStyle({ disabled }: { disabled: boolean }) {
  const t = useT();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const resetting = useRef(false);
  const { allowed: canReset } = usePermission(Resource.DispatchControl, Operation.Update);

  const reset = useApiMutation({
    scope: { id: "dispatch-style-reset" },
    resourceName: "Dispatch style",
    mutationFn: () => resetDispatchStyle(),
    onSuccess: (style) => {
      queryClient.setQueryData(dispatchStyle.get().queryKey, style);
      setOpen(false);
    },
  });

  const confirm = async () => {
    if (resetting.current) {
      return;
    }
    resetting.current = true;
    try {
      await reset.mutateAsync();
    } catch {
      return;
    } finally {
      resetting.current = false;
    }
  };

  if (!canReset) {
    return null;
  }

  return (
    <>
      <div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={disabled}
          onClick={() => setOpen(true)}
        >
          {t("Reset learned style")}
        </Button>
      </div>
      <AlertDialog open={open} onOpenChange={(next) => !reset.isPending && setOpen(next)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("Reset the learned dispatch style?")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "The planner forgets every weight it learned and goes back to your own costs. Learning starts again from the next driver a dispatcher assigns; decisions already made are not learned again.",
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={reset.isPending}>{t("Keep it")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              isLoading={reset.isPending}
              onClick={() => void confirm()}
            >
              {t("Reset")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
