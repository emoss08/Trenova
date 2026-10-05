import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { useCloudPlan } from "../../../hooks/use-cloud-plan";
import { isFreeDemoPlan, trialEndingLimit, type CloudTrialStatus } from "../../../lib/cloud-trial";
import {
  comparePlanMeters,
  formatPlanMeterValue,
  planMeterDefinition,
  type PlanMeterWindow,
} from "@/lib/plan-meters";
import { PLAN_CAPABILITIES, planDisplayName } from "../../../lib/free-demo";
import type { BillingUsageSummary } from "../../../types/platform-billing";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Progress } from "@trenova/shared/components/ui/progress";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  AlertCircleIcon,
  InfoCircleIcon,
  Lock01Icon,
  RefreshCcw02Icon,
} from "@trenova/shared/components/icons";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { useMemo } from "react";

const WINDOW_LABEL: Record<PlanMeterWindow, string> = {
  lifetime: "Total held",
  month: "Resets monthly",
  item: "Per file",
};

export type UsageLevel = "ok" | "near" | "full" | "unlimited";

/** How close a meter is to its limit; a limit of zero or less means the meter is uncapped. */
export function usageLevel(usage: Pick<BillingUsageSummary, "limit" | "used">): UsageLevel {
  if (usage.limit <= 0) {
    return "unlimited";
  }
  const ratio = usage.used / usage.limit;
  if (ratio >= 1) return "full";
  if (ratio >= 0.8) return "near";
  return "ok";
}

const PROGRESS_VARIANT: Record<
  Exclude<UsageLevel, "unlimited">,
  "default" | "warning" | "error"
> = {
  ok: "default",
  near: "warning",
  full: "error",
};

function subscriptionStatusLabel(
  trial: CloudTrialStatus | null,
  status: string | undefined,
  t: TranslateFn,
) {
  if (trial?.kind === "trialing") return t("Trial");
  if (trial?.kind === "read_only") return t("Read-only");
  if (trial?.kind === "expired") return t("Expired");
  if (status === "active") return t("Active");
  return t("Unlimited");
}

export function PlanUsagePage() {
  const t = useT();
  const { isCloud, summary, trial, isLoading, isError, refetch, isFetching } = useCloudPlan();

  const usage = useMemo(
    () => [...(summary?.usage ?? [])].sort((a, b) => comparePlanMeters(a.meterKey, b.meterKey)),
    [summary?.usage],
  );
  const freeDemo = isFreeDemoPlan(summary);
  const planName = summary?.plan
    ? planDisplayName(summary.plan.key, summary.plan.name)
    : t("Unlimited");
  const shipmentAllowance = trialEndingLimit(summary);

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Plan & usage"),
        description: t(
          "Your organization's plan, where its trial stands and how much of each limit it has used.",
        ),
        actions: isCloud ? (
          <Button variant="outline" size="sm" onClick={refetch} disabled={isFetching}>
            <RefreshCcw02Icon className="size-3.5" />
            {t("Refresh")}
          </Button>
        ) : undefined,
      }}
    >
      {!isCloud ? (
        <Alert variant="info">
          <InfoCircleIcon />
          <AlertTitle>{t("No plan limits on this installation")}</AlertTitle>
          <AlertDescription>
            {t(
              "Plans and usage limits apply only on Trenova Cloud. Every organization here is unlimited.",
            )}
          </AlertDescription>
        </Alert>
      ) : isLoading ? (
        <PlanUsageSkeleton />
      ) : isError || !summary ? (
        <Alert variant="destructive">
          <AlertCircleIcon />
          <AlertTitle>{t("Unable to load your plan")}</AlertTitle>
          <AlertDescription>
            {t("The plan and usage summary could not be loaded. Try refreshing.")}
          </AlertDescription>
        </Alert>
      ) : (
        <>
          <KpiStrip aria-label={t("Plan summary")}>
            <KpiStripItem label={t("Plan")} value={t(planName)} />
            <KpiStripItem
              label={t("Status")}
              value={subscriptionStatusLabel(trial, summary.subscription?.status, t)}
              tone={trial?.kind === "trialing" ? "info" : trial ? "warning" : undefined}
            />
            {trial?.kind === "trialing" ? (
              <KpiStripItem
                label={t("Trial ends")}
                value={formatUnixDateMedium(trial.trialEndsAt, { fallback: "—" })}
                sub={
                  trial.daysRemaining === null
                    ? undefined
                    : trial.daysRemaining === 1
                      ? t("1 day left")
                      : t("{0} days left", String(trial.daysRemaining))
                }
              />
            ) : null}
            {trial?.kind === "read_only" ? (
              <KpiStripItem
                label={t("Deleted on")}
                value={formatUnixDateMedium(trial.readOnlyUntil, { fallback: "—" })}
                sub={
                  trial.daysUntilPurge === null
                    ? undefined
                    : t("{0} days left", String(trial.daysUntilPurge))
                }
              />
            ) : null}
          </KpiStrip>

          {trial?.kind === "trialing" ? (
            <Alert variant="info" size="sm">
              <InfoCircleIcon />
              <AlertDescription>
                {shipmentAllowance === null
                  ? t("The trial ends on the date above.")
                  : t(
                      "The trial ends on the date above, or as soon as all {0} shipments are used, whichever comes first.",
                      String(shipmentAllowance),
                    )}
              </AlertDescription>
            </Alert>
          ) : null}

          {trial?.kind === "read_only" || trial?.kind === "expired" ? (
            <Alert variant="warning" size="sm">
              <Lock01Icon />
              <AlertDescription>
                {t(
                  "The free demo's trial period is over, so this workspace is read-only: everything can be opened and exported, but nothing can be created or changed. Paid plans are coming soon.",
                )}
              </AlertDescription>
            </Alert>
          ) : freeDemo ? (
            <Alert variant="info" size="sm">
              <InfoCircleIcon />
              <AlertDescription>
                {t(
                  "This is Trenova Cloud's free demo: the whole product with small limits, for trying it on your own freight. Paid plans with higher limits are coming soon.",
                )}
              </AlertDescription>
            </Alert>
          ) : null}

          <SectionPanel title={t("Usage")} count={usage.length}>
            {usage.length === 0 ? (
              <SectionPanelQuiet>{t("This plan has no usage limits.")}</SectionPanelQuiet>
            ) : (
              <ul className="divide-border-subtle m-0 flex list-none flex-col divide-y p-0">
                {usage.map((row) => (
                  <UsageRow key={row.meterKey} usage={row} />
                ))}
              </ul>
            )}
          </SectionPanel>

          {freeDemo ? (
            <SectionPanel
              title={t("Not included in the free demo")}
              help={t("Each of these returns a plan notice when tried. They come with paid plans.")}
            >
              <div className="p-3">
                <DescriptionList layout="inline">
                  {Object.entries(PLAN_CAPABILITIES).map(([key, capability]) => (
                    <DescriptionItem key={key} label={t(capability.label)}>
                      {t(capability.explanation)}
                    </DescriptionItem>
                  ))}
                </DescriptionList>
              </div>
            </SectionPanel>
          ) : null}
        </>
      )}
    </PageLayout>
  );
}

function UsageRow({ usage }: { usage: BillingUsageSummary }) {
  const t = useT();
  const meter = planMeterDefinition(usage.meterKey);
  const level = usageLevel(usage);

  return (
    <li className="grid gap-2 px-3 py-2.5 sm:grid-cols-[minmax(0,14rem)_minmax(0,1fr)_8rem] sm:items-center sm:gap-4">
      <div className="min-w-0">
        <p className="m-0 truncate text-sm font-medium">{t(meter.label)}</p>
        <p className="text-muted-foreground m-0 truncate text-xs">
          {meter.description ? t(meter.description) : t(WINDOW_LABEL[meter.window])}
        </p>
      </div>
      {level === "unlimited" ? (
        <span className="text-muted-foreground text-xs">{t("Unlimited")}</span>
      ) : meter.window === "item" ? (
        <span className="text-muted-foreground text-xs">{t(WINDOW_LABEL.item)}</span>
      ) : (
        <Progress
          value={Math.min(usage.used, usage.limit)}
          max={usage.limit}
          size="sm"
          variant={PROGRESS_VARIANT[level]}
          aria-label={t(meter.label)}
        />
      )}
      <div className="text-right text-sm tabular-nums">
        {level === "unlimited" ? (
          formatPlanMeterValue(usage.meterKey, usage.used)
        ) : meter.window === "item" ? (
          t("Up to {0}", formatPlanMeterValue(usage.meterKey, usage.limit))
        ) : (
          <>
            {formatPlanMeterValue(usage.meterKey, usage.used)}
            <span className="text-muted-foreground">
              {" / "}
              {formatPlanMeterValue(usage.meterKey, usage.limit)}
            </span>
          </>
        )}
        {meter.window === "month" && usage.windowEnd > 0 ? (
          <span className="text-muted-foreground block text-xs">
            {t("Resets {0}", formatUnixDateMedium(usage.windowEnd))}
          </span>
        ) : null}
      </div>
    </li>
  );
}

function PlanUsageSkeleton() {
  return (
    <>
      <Skeleton className="h-16" />
      <Skeleton className="h-96" />
    </>
  );
}
