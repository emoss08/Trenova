import { useCloudPlan } from "../../hooks/use-cloud-plan";
import { demoBannerMessages, type CloudTrialStatus } from "../../lib/cloud-trial";
import { PLAN_USAGE_PATH } from "../../lib/free-demo";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ClockIcon, Lock01Icon } from "@trenova/shared/components/icons";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";
import { DemoBanner } from "./demo-banner";

const WARNING_DAYS = 5;

export function trialBannerMessage(trial: CloudTrialStatus, t: TranslateFn): string {
  switch (trial.kind) {
    case "trialing":
      if (trial.daysRemaining === null) {
        return t("You're on the free demo. Paid plans are coming soon.");
      }
      if (trial.daysRemaining <= 0) {
        return t("Your free demo ends today. After that the workspace becomes read-only.");
      }
      return trial.daysRemaining === 1
        ? t("Free demo: 1 day left. After that the workspace becomes read-only.")
        : t(
            "Free demo: {0} days left. After that the workspace becomes read-only.",
            String(trial.daysRemaining),
          );
    case "read_only":
      return trial.readOnlyUntil
        ? t(
            "Your free demo has ended. The workspace is read-only until {0}, then it is deleted.",
            formatUnixDateMedium(trial.readOnlyUntil),
          )
        : t("Your free demo has ended. The workspace is read-only and will be deleted.");
    case "expired":
      return t("Your free demo has expired and this workspace is scheduled for deletion.");
  }
}

/**
 * The free demo's banner above the app header. While the trial runs it is the
 * animated countdown of days and shipments left; once writes have stopped it is
 * the plain notice saying why every save is refused. Never dismissible.
 */
export function CloudTrialBanner() {
  const t = useT();
  const { trial, summary } = useCloudPlan();
  const canOpenPlan = usePermissionStore((state) =>
    state.hasPermission(Resource.Organization, Operation.Read),
  );

  if (!trial) {
    return null;
  }

  const messages = demoBannerMessages(trial, summary);
  if (messages.length > 0) {
    return (
      <DemoBanner
        key={messages.map((message) => message.kind).join("|")}
        messages={messages}
        showPlanLink={canOpenPlan}
      />
    );
  }

  const ended = trial.kind !== "trialing";
  const urgent = ended || (trial.daysRemaining !== null && trial.daysRemaining <= WARNING_DAYS);
  const Icon = ended ? Lock01Icon : ClockIcon;

  return (
    <div
      role={ended ? "alert" : "status"}
      data-trial-state={trial.kind}
      className={cn(
        "flex min-h-8 shrink-0 items-center gap-2 border-b px-4 py-1.5 text-xs",
        urgent
          ? "border-warning-border bg-warning-subtle text-warning-subtle-foreground"
          : "border-info-border bg-info-subtle text-info-subtle-foreground",
      )}
    >
      <Icon className="size-3.5 shrink-0" aria-hidden="true" />
      <span className="min-w-0 flex-1">{trialBannerMessage(trial, t)}</span>
      {canOpenPlan ? (
        <Link
          to={PLAN_USAGE_PATH}
          className="shrink-0 font-medium underline underline-offset-[3px]"
        >
          {t("Plan & usage")}
        </Link>
      ) : null}
    </div>
  );
}
