import { edition } from "@/lib/edition";
import { planLimitCopy } from "@/lib/plan-limit-copy";
import { usePlanLimitStore } from "@/stores/plan-limit-store";
import type { EditionPlan, PlanLimitCopy } from "@trenova/edition";
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
import type { PlanLimitNotice } from "@trenova/shared/lib/plan-limit";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useNavigate } from "react-router";

/** The edition's words for a refusal when it has its own, the host's otherwise. */
function dialogCopy(
  notice: PlanLimitNotice,
  t: TranslateFn,
  plan: Pick<EditionPlan, "limitCopy"> = edition.plan,
): PlanLimitCopy {
  return plan.limitCopy?.(notice, t) ?? planLimitCopy(notice, t);
}

export function PlanLimitDialog({
  notice,
  onClose,
  plan = edition.plan,
}: {
  notice: PlanLimitNotice | null;
  onClose: () => void;
  plan?: Pick<EditionPlan, "limitCopy" | "usagePath">;
}) {
  const t = useT();
  const navigate = useNavigate();
  const canOpenPlan = usePermissionStore((state) =>
    state.hasPermission(Resource.Organization, Operation.Read),
  );
  const copy = notice ? dialogCopy(notice, t, plan) : null;
  const usagePath = plan.usagePath;

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
            {canOpenPlan && usagePath ? (
              <Button
                variant="outline"
                onClick={() => {
                  onClose();
                  void navigate(usagePath);
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
export function PlanLimitDialogHost({
  plan,
}: {
  plan?: Pick<EditionPlan, "limitCopy" | "usagePath">;
}) {
  const notice = usePlanLimitStore((state) => state.notice);
  const dismiss = usePlanLimitStore((state) => state.dismiss);
  return <PlanLimitDialog notice={notice} onClose={dismiss} plan={plan} />;
}
