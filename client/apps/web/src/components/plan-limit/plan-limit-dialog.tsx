import {
  PLAN_USAGE_PATH,
  formatPlanMeterValue,
  planCapabilityDefinition,
  planDisplayName,
  planMeterDefinition,
} from "@/lib/plan-meters";
import { usePlanLimitStore } from "@/stores/plan-limit-store";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Progress } from "@trenova/shared/components/ui/progress";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  SUBSCRIPTION_READ_ONLY_REASON,
  type PlanLimitNotice,
} from "@trenova/shared/lib/plan-limit";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useNavigate } from "react-router";

export type PlanLimitCopy = {
  title: string;
  description: string;
  guidance: string;
  usage: { used: number; limit: number; usedLabel: string; limitLabel: string } | null;
};

const PAID_PLANS_COMING = "Paid plans with higher limits are coming soon.";

/**
 * The words the dialog shows for one refusal. A quota names the meter, how much of it
 * is used and what frees room; a restriction names what the demo leaves out; the
 * read-only refusal explains that the trial is over rather than blaming the action.
 */
export function planLimitCopy(notice: PlanLimitNotice, t: TranslateFn): PlanLimitCopy {
  const planName = planDisplayName(notice.plan);

  if (notice.kind === "quota") {
    const meter = planMeterDefinition(notice.meter);
    const label = t(meter.label);
    const hasFigures = notice.limit !== null;
    const usage =
      meter.window !== "item" && notice.limit !== null && notice.limit > 0
        ? {
            used: Math.min(notice.used ?? notice.limit, notice.limit),
            limit: notice.limit,
            usedLabel: formatPlanMeterValue(notice.meter, notice.used ?? notice.limit),
            limitLabel: formatPlanMeterValue(notice.meter, notice.limit),
          }
        : null;

    const description =
      meter.window === "item" && hasFigures
        ? t(
            "{0} allows files up to {1}. This one is larger.",
            t(planName),
            formatPlanMeterValue(notice.meter, notice.limit ?? 0),
          )
        : usage
          ? t("{0}: {1} of {2} used ({3}).", label, usage.usedLabel, usage.limitLabel, t(planName))
          : t("Your organization has reached its {0} limit ({1}).", label, t(planName));

    const guidance =
      meter.window === "month"
        ? t("This limit resets at the start of next month. {0}", t(PAID_PLANS_COMING))
        : meter.window === "item"
          ? t("Upload a smaller file, or split it into parts. {0}", t(PAID_PLANS_COMING))
          : t(
              "The free demo is for trying Trenova, so it holds a small amount of data. Delete records you no longer need to free room. {0}",
              t(PAID_PLANS_COMING),
            );

    return { title: t("{0} limit reached", label), description, guidance, usage };
  }

  if (notice.reason === SUBSCRIPTION_READ_ONLY_REASON) {
    return {
      title: t("Your free demo has ended"),
      description: t(
        "The trial period is over, so this workspace is read-only: you can open and export everything, but nothing can be created or changed.",
      ),
      guidance: t(
        "The workspace is deleted when the read-only period ends. Paid plans are coming soon; until then, export anything you want to keep.",
      ),
      usage: null,
    };
  }

  const capability = planCapabilityDefinition(notice.capability);
  return {
    title: t("{0} not available", t(capability.label)),
    description: t(capability.explanation),
    guidance: t(
      "{0} is for trying Trenova and leaves this out. Paid plans that include it are coming soon.",
      t(planName),
    ),
    usage: null,
  };
}

export function PlanLimitDialog({
  notice,
  onClose,
}: {
  notice: PlanLimitNotice | null;
  onClose: () => void;
}) {
  const t = useT();
  const navigate = useNavigate();
  const canOpenPlan = usePermissionStore((state) =>
    state.hasPermission(Resource.Organization, Operation.Read),
  );
  const copy = notice ? planLimitCopy(notice, t) : null;

  return (
    <Dialog
      open={notice !== null}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      {copy ? (
        <DialogContent size="sm" data-testid="plan-limit-dialog">
          <DialogHeader>
            <DialogTitle>{copy.title}</DialogTitle>
            <DialogDescription>{copy.description}</DialogDescription>
          </DialogHeader>
          {copy.usage ? (
            <div className="flex flex-col gap-1.5">
              <Progress
                value={copy.usage.used}
                max={copy.usage.limit}
                size="sm"
                variant="warning"
                aria-label={t("Usage")}
              />
              <div className="text-muted-foreground flex justify-between text-xs tabular-nums">
                <span>{t("{0} used", copy.usage.usedLabel)}</span>
                <span>{t("{0} limit", copy.usage.limitLabel)}</span>
              </div>
            </div>
          ) : null}
          <p className="text-muted-foreground m-0 text-sm">{copy.guidance}</p>
          <DialogFooter>
            {canOpenPlan ? (
              <Button
                variant="outline"
                onClick={() => {
                  onClose();
                  void navigate(PLAN_USAGE_PATH);
                }}
              >
                {t("View plan & usage")}
              </Button>
            ) : null}
            <Button onClick={onClose}>{t("Got it")}</Button>
          </DialogFooter>
        </DialogContent>
      ) : null}
    </Dialog>
  );
}

/** Renders whatever refusal the transports last reported. Mounted once, at the root. */
export function PlanLimitDialogHost() {
  const notice = usePlanLimitStore((state) => state.notice);
  const dismiss = usePlanLimitStore((state) => state.dismiss);
  return <PlanLimitDialog notice={notice} onClose={dismiss} />;
}
