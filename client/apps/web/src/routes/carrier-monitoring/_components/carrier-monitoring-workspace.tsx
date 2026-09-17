import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import { CARRIER_INTEL_INTEGRATIONS_PATH } from "@/lib/carrier-links";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostBar, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PlugZapIcon, RefreshCwIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { useCallback } from "react";
import { Link } from "react-router";
import { EnrollmentTable } from "./enrollment-table";
import { EventInbox } from "./event-inbox";
import { MonitoringSummary } from "./monitoring-summary";
import {
  CLEARED_TAB_STATE,
  MONITORING_TABS,
  monitoringPageSearchParams,
  type MonitoringTab,
} from "./monitoring-tabs";
import { ReviewQueue } from "./review-queue";
import { UsagePanel } from "./usage-panel";

const MONITORING_STATUS_REFRESH_MS = 60_000;

function MonitoringSketch() {
  return (
    <div className="flex flex-col gap-3">
      <div className="grid grid-cols-4 gap-2">
        {[0, 1, 2, 3].map((index) => (
          <div key={index} className="flex flex-col gap-2 rounded-lg border p-3">
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

function isMonitoringTab(value: unknown): value is MonitoringTab {
  return typeof value === "string" && (MONITORING_TABS as readonly string[]).includes(value);
}

export function CarrierMonitoringWorkspace() {
  const t = useT();
  const [{ tab }, setSearchParams] = useQueryStates(monitoringPageSearchParams);
  const { allowed: canUpdate } = usePermission(Resource.CarrierIntelligence, Operation.Update);
  const { allowed: canManage } = usePermission(Resource.CarrierIntelligence, Operation.Manage);

  const statusQuery = useQuery({
    ...queries.carrierIntelSettings.monitoringStatus(),
    refetchInterval: MONITORING_STATUS_REFRESH_MS,
  });

  const changeTab = useCallback(
    (next: MonitoringTab) => {
      if (next === tab) {
        return;
      }
      void setSearchParams({ ...CLEARED_TAB_STATE, tab: next === "inbox" ? null : next });
    },
    [setSearchParams, tab],
  );

  if (statusQuery.isPending) {
    return (
      <div className="flex flex-col gap-3">
        <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-5">
          {[0, 1, 2, 3, 4].map((index) => (
            <Skeleton key={index} className="h-24 w-full" />
          ))}
        </div>
        <Skeleton className="h-10 w-80" />
        <Skeleton className="h-96 w-full" />
      </div>
    );
  }

  if (statusQuery.isError) {
    return (
      <div className="text-destructive flex items-center justify-between gap-2 rounded-lg border border-dashed p-3 text-sm">
        <span>
          {t(
            "Carrier monitoring could not be loaded. {0}",
            graphQLErrorMessage(statusQuery.error, t("Try again in a moment.")),
          )}
        </span>
        <Button
          type="button"
          size="xs"
          variant="outline"
          onClick={() => void statusQuery.refetch()}
        >
          <RefreshCwIcon />
          {t("Retry")}
        </Button>
      </div>
    );
  }

  const status = statusQuery.data;
  const hasHistory =
    status.eventCounts.open + status.eventCounts.acknowledged > 0 ||
    status.enrollmentCounts.desired > 0 ||
    status.reviewQueueCount > 0;

  if (!status.provider.configured && !hasHistory) {
    return (
      <EmptySheet
        title={t("No carrier intelligence provider is connected")}
        description={t(
          "Connect CarrierOK or the free FMCSA QCMobile service to watch carriers for authority, insurance and safety changes. Changes, review requests and provider spend all show up here once monitoring starts.",
        )}
        sketch={<MonitoringSketch />}
        action={
          <Button
            size="sm"
            variant="outline"
            nativeButton={false}
            render={<Link to={CARRIER_INTEL_INTEGRATIONS_PATH} />}
          >
            <PlugZapIcon className="size-3.5" />
            {t("Open integrations")}
          </Button>
        }
      />
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {!status.provider.configured ? (
        <p className="text-muted-foreground rounded-lg border border-dashed px-3 py-2 text-xs">
          {t(
            "No provider is connected right now, so nothing new is being detected. The events and enrollments below are what was recorded before.",
          )}{" "}
          <Link to={CARRIER_INTEL_INTEGRATIONS_PATH} className="underline">
            {t("Open integrations")}
          </Link>
        </p>
      ) : null}
      <MonitoringSummary status={status} canManage={canManage} onNavigate={changeTab} />
      <Tabs
        value={tab}
        onValueChange={(value) => {
          if (isMonitoringTab(value)) {
            changeTab(value);
          }
        }}
      >
        <TabsList variant="underline">
          <TabsTab value="inbox">
            {t("Inbox")}
            {status.eventCounts.open > 0 ? (
              <Badge variant="warning" className="max-h-5 tabular-nums">
                {status.eventCounts.open}
              </Badge>
            ) : null}
          </TabsTab>
          <TabsTab value="review">
            {t("Review queue")}
            {status.reviewQueueCount > 0 ? (
              <Badge variant="warning" className="max-h-5 tabular-nums">
                {status.reviewQueueCount}
              </Badge>
            ) : null}
          </TabsTab>
          <TabsTab value="enrollments">
            {t("Enrollments")}
            {status.enrollmentCounts.failed > 0 ? (
              <Badge variant="inactive" className="max-h-5 tabular-nums">
                {status.enrollmentCounts.failed}
              </Badge>
            ) : null}
          </TabsTab>
          <TabsTab value="usage">{t("Usage")}</TabsTab>
        </TabsList>
        <TabsContent value="inbox" className="pt-3">
          {tab === "inbox" ? <EventInbox canUpdate={canUpdate} /> : null}
        </TabsContent>
        <TabsContent value="review" className="pt-3">
          {tab === "review" ? <ReviewQueue canUpdate={canUpdate} /> : null}
        </TabsContent>
        <TabsContent value="enrollments" className="pt-3">
          {tab === "enrollments" ? <EnrollmentTable canUpdate={canUpdate} /> : null}
        </TabsContent>
        <TabsContent value="usage" className="pt-3">
          {tab === "usage" ? (
            canManage ? (
              <UsagePanel />
            ) : (
              <EmptySheet
                title={t("Provider spend is restricted")}
                description={t(
                  "Usage and spend are visible to users who can manage carrier intelligence. Ask an administrator for manage access to Carrier Intelligence.",
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
