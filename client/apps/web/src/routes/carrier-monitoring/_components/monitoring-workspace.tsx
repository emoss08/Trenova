import { StatStrip, type StatStripItem } from "@/components/carrier-intelligence/stat-strip";
import { StatusDot } from "@/components/carrier-intelligence/status-dot";
import { PageHeader } from "@/components/page-header";
import { usePermission } from "@/hooks/use-permission";
import { spendCapProgress } from "@/lib/carrier-intel-usage";
import { formatOptionalDecimalCurrency } from "@/lib/carrier-intelligence";
import { CARRIER_INTEL_INTEGRATIONS_PATH } from "@/lib/carrier-links";
import type { CarrierIntelMonitoringStatus } from "@/lib/graphql/carrier-intel-settings";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostBar, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useT } from "@trenova/shared/i18n/use-t";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PlugZapIcon, RefreshCwIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { useCallback, type ReactNode } from "react";
import { Link } from "react-router";
import { EventInbox } from "./inbox/event-inbox";
import { monitoringHealth } from "./monitoring-health";
import {
  CLEARED_TAB_STATE,
  isMonitoringTab,
  monitoringPageSearchParams,
  type MonitoringNavigation,
} from "./monitoring-tabs";
import { ProviderStatus } from "./provider-status";
import { ReviewQueue } from "./review/review-queue";
import { UsagePanel } from "./usage/usage-panel";
import { EnrollmentTable } from "./watchlist/enrollment-table";

const MONITORING_STATUS_REFRESH_MS = 60_000;

function MonitoringSketch() {
  return (
    <div className="flex flex-col gap-3">
      <div className="grid grid-cols-4 divide-x rounded-lg border">
        {[0, 1, 2, 3].map((index) => (
          <div key={index} className="flex flex-col gap-2 p-3">
            <GhostLine className="w-16" />
            <GhostLine className="w-8" />
          </div>
        ))}
      </div>
      <div className="flex flex-col gap-2 rounded-lg border p-3">
        <GhostBar share={60} className="w-full" />
        <GhostBar share={35} className="w-full" />
        <GhostBar share={80} className="w-full" />
      </div>
    </div>
  );
}

function TabCount({ value }: { value: number }) {
  if (value <= 0) {
    return null;
  }
  return (
    <span className="text-muted-foreground text-xs font-normal tabular-nums">
      {value.toLocaleString()}
    </span>
  );
}

function Notice({ children, action }: { children: ReactNode; action?: ReactNode }) {
  return (
    <div
      role="status"
      className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 rounded-lg border px-3 py-2"
    >
      <p className="flex items-center gap-2 text-sm">
        <StatusDot tone="medium" />
        {children}
      </p>
      {action}
    </div>
  );
}

function MonitoringLoading() {
  return (
    <div className="flex flex-col gap-4">
      <Skeleton className="h-20 w-full" />
      <Skeleton className="h-8 w-80" />
      <Skeleton className="h-[28rem] w-full" />
    </div>
  );
}

type MonitoringBodyProps = {
  status: CarrierIntelMonitoringStatus;
  canUpdate: boolean;
  canManage: boolean;
};

function MonitoringBody({ status, canUpdate, canManage }: MonitoringBodyProps) {
  const t = useT();
  const [{ tab }, setSearchParams] = useQueryStates(monitoringPageSearchParams);
  const health = monitoringHealth(status);

  const usageQuery = useQuery({
    ...queries.carrierIntelSettings.usage(),
    enabled: canManage,
  });

  const navigate = useCallback(
    ({ tab: next, scope }: MonitoringNavigation) => {
      void setSearchParams({
        ...CLEARED_TAB_STATE,
        tab: next === "inbox" ? null : next,
        scope: scope && scope !== "attention" ? scope : null,
      });
    },
    [setSearchParams],
  );

  const criticalOpen =
    status.eventCounts.bySeverity.find((entry) => entry.severity === "Critical")?.count ?? 0;
  const { enrollmentCounts } = status;

  const stats: StatStripItem[] = [
    {
      id: "attention",
      label: t("Needs attention"),
      value: status.eventCounts.open.toLocaleString(),
      tone: criticalOpen > 0 ? "critical" : undefined,
      hint:
        criticalOpen > 0
          ? t("{0, plural, one {# critical} other {# critical}}", criticalOpen)
          : status.eventCounts.acknowledged > 0
            ? t("{0} acknowledged", status.eventCounts.acknowledged.toLocaleString())
            : undefined,
      onClick: () => navigate({ tab: "inbox", scope: "attention" }),
    },
    {
      id: "review",
      label: t("Awaiting review"),
      value: status.reviewQueueCount.toLocaleString(),
      onClick: () => navigate({ tab: "review" }),
    },
    {
      id: "monitored",
      label: t("Monitored carriers"),
      value: enrollmentCounts.active.toLocaleString(),
      hint:
        enrollmentCounts.pending > 0 || enrollmentCounts.failed > 0
          ? [
              enrollmentCounts.pending > 0
                ? t("{0} pending", enrollmentCounts.pending.toLocaleString())
                : null,
              enrollmentCounts.failed > 0
                ? t("{0} failed", enrollmentCounts.failed.toLocaleString())
                : null,
            ]
              .filter(Boolean)
              .join(" · ")
          : undefined,
      onClick: () => navigate({ tab: "enrollments" }),
    },
  ];

  if (canManage) {
    const usage = usageQuery.data;
    const progress = usage
      ? spendCapProgress(usage.monthToDate, usage.cap, usage.softCapPercent)
      : null;
    stats.push({
      id: "spend",
      label: t("Month-to-date spend"),
      value: usage ? (
        (formatOptionalDecimalCurrency(usage.monthToDate) ?? "—")
      ) : usageQuery.isError ? (
        "—"
      ) : (
        <Skeleton className="mt-1 h-6 w-20" />
      ),
      tone:
        progress?.state === "exceeded"
          ? "critical"
          : progress?.state === "soft"
            ? "medium"
            : undefined,
      hint:
        usage && progress && progress.state !== "uncapped"
          ? t("of {0} cap", formatOptionalDecimalCurrency(usage.cap) ?? "—")
          : undefined,
      onClick: () => navigate({ tab: "usage" }),
    });
  }

  return (
    <div className="flex flex-col gap-4">
      {health.state === "paused" ? (
        <Notice
          action={
            <Button
              variant="ghost"
              className="h-8 text-xs"
              nativeButton={false}
              render={<Link to={CARRIER_INTEL_INTEGRATIONS_PATH} />}
            >
              {t("Open integrations")}
            </Button>
          }
        >
          {t(
            "Monitoring is paused, so changes on monitored carriers are not being detected until it resumes.",
          )}
        </Notice>
      ) : health.state === "disconnected" ? (
        <Notice
          action={
            <Button
              variant="ghost"
              className="h-8 text-xs"
              nativeButton={false}
              render={<Link to={CARRIER_INTEL_INTEGRATIONS_PATH} />}
            >
              <PlugZapIcon className="size-3.5" />
              {t("Connect a provider")}
            </Button>
          }
        >
          {t("No provider is connected, so nothing new is being detected.")}
        </Notice>
      ) : null}

      <StatStrip items={stats} />

      <Tabs
        value={tab}
        className="gap-4"
        onValueChange={(value) => {
          if (isMonitoringTab(value) && value !== tab) {
            navigate({ tab: value });
          }
        }}
      >
        <TabsList variant="underline">
          <TabsTab value="inbox">
            {t("Inbox")}
            <TabCount value={status.eventCounts.open} />
          </TabsTab>
          <TabsTab value="review">
            {t("Review")}
            <TabCount value={status.reviewQueueCount} />
          </TabsTab>
          <TabsTab value="enrollments">
            {t("Watchlist")}
            <TabCount value={enrollmentCounts.active} />
          </TabsTab>
          <TabsTab value="usage">{t("Usage")}</TabsTab>
        </TabsList>
        <TabsContent value="inbox">
          {tab === "inbox" ? (
            <EventInbox
              canUpdate={canUpdate}
              scopeCounts={{
                attention: status.eventCounts.open,
                acknowledged: status.eventCounts.acknowledged,
              }}
            />
          ) : null}
        </TabsContent>
        <TabsContent value="review">
          {tab === "review" ? <ReviewQueue canUpdate={canUpdate} /> : null}
        </TabsContent>
        <TabsContent value="enrollments">
          {tab === "enrollments" ? <EnrollmentTable canUpdate={canUpdate} /> : null}
        </TabsContent>
        <TabsContent value="usage">
          {tab === "usage" ? (
            canManage ? (
              <UsagePanel />
            ) : (
              <EmptySheet
                title={t("Provider spend is restricted")}
                description={t(
                  "Usage and spend are visible to users who can manage carrier intelligence.",
                )}
                sketch={<MonitoringSketch />}
              />
            )
          ) : null}
        </TabsContent>
      </Tabs>
    </div>
  );
}

export function MonitoringWorkspace() {
  const t = useT();
  const { allowed: canUpdate } = usePermission(Resource.CarrierIntelligence, Operation.Update);
  const { allowed: canManage } = usePermission(Resource.CarrierIntelligence, Operation.Manage);

  const statusQuery = useQuery({
    ...queries.carrierIntelSettings.monitoringStatus(),
    refetchInterval: MONITORING_STATUS_REFRESH_MS,
  });

  const status = statusQuery.data;
  const hasHistory = status
    ? status.eventCounts.open + status.eventCounts.acknowledged > 0 ||
      status.enrollmentCounts.desired > 0 ||
      status.reviewQueueCount > 0
    : false;

  let body: ReactNode;
  if (statusQuery.isPending) {
    body = <MonitoringLoading />;
  } else if (statusQuery.isError) {
    body = (
      <EmptySheet
        title={t("Carrier monitoring could not be loaded")}
        description={graphQLErrorMessage(statusQuery.error, t("Try again in a moment."))}
        sketch={<MonitoringSketch />}
        action={
          <Button
            type="button"
            variant="outline"
            className="h-8 text-xs"
            onClick={() => void statusQuery.refetch()}
          >
            <RefreshCwIcon className="size-3.5" />
            {t("Retry")}
          </Button>
        }
      />
    );
  } else if (!status.provider.configured && !hasHistory) {
    body = (
      <EmptySheet
        title={t("Connect a carrier intelligence provider")}
        description={t(
          "Connect CarrierOk or the free FMCSA QCMobile service to watch carriers for authority, insurance and safety changes. Changes, reviews and provider spend show up here once monitoring starts.",
        )}
        sketch={<MonitoringSketch />}
        action={
          <Button
            variant="outline"
            className="h-8 text-xs"
            nativeButton={false}
            render={<Link to={CARRIER_INTEL_INTEGRATIONS_PATH} />}
          >
            <PlugZapIcon className="size-3.5" />
            {t("Open integrations")}
          </Button>
        }
      />
    );
  } else {
    body = <MonitoringBody status={status} canUpdate={canUpdate} canManage={canManage} />;
  }

  return (
    <>
      <PageHeader
        title={t("Carrier Monitoring")}
        description={t("Authority, insurance and safety changes on the carriers you watch.")}
        actions={status ? <ProviderStatus status={status} canManage={canManage} /> : null}
      />
      <div className="flex flex-col gap-4 p-4">{body}</div>
    </>
  );
}
