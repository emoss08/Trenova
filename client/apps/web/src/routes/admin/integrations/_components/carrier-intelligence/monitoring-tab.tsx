import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  carrierIntelProviderLabel,
  formatOptionalDecimalCurrency,
} from "@/lib/carrier-intelligence";
import {
  resumeCarrierIntelMonitoring,
  type CarrierIntelFeedState,
} from "@/lib/graphql/carrier-intel-settings";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeOrDash } from "@trenova/shared/lib/date";
import { RefreshCwIcon } from "lucide-react";
import { useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";
import {
  enrollmentPolicyChoices,
  type CarrierIntelSettingsFormValues,
} from "./carrier-intel-settings-schema";
import { SettingsSection } from "./settings-section";

const feedTypeLabels: Record<string, string> = {
  ChangeFeed: "Change feed",
  SnapshotRefresh: "Snapshot refresh",
};

export function CarrierIntelMonitoringTab({
  open,
  readOnly,
  canManage,
  providerConfigured,
}: {
  open: boolean;
  readOnly: boolean;
  canManage: boolean;
  providerConfigured: boolean;
}) {
  const t = useT();
  const { control } = useFormContext<CarrierIntelSettingsFormValues>();
  const enrollmentPolicy = useWatch({ control, name: "enrollmentPolicy" });

  return (
    <div className="space-y-6">
      <fieldset disabled={readOnly} className="m-0 min-w-0 space-y-6 border-0 p-0">
        <SettingsSection
          title={t("Enrollment")}
          description={t("Choose which carriers are enrolled for continuous monitoring.")}
        >
          <FormGroup cols={2}>
            <FormControl cols="full">
              <SelectField
                control={control}
                name="enrollmentPolicy"
                label={t("Enrollment policy")}
                description={t(
                  "Manual monitors only carriers you enroll. Recently used enrolls carriers with recent shipments. All active enrolls every active carrier.",
                )}
                options={enrollmentPolicyChoices.map((choice) => ({
                  value: choice.value,
                  label: t(choice.label),
                }))}
                isReadOnly={readOnly}
              />
            </FormControl>
            {enrollmentPolicy === "RecentlyUsed" ? (
              <>
                <FormControl>
                  <NumberField
                    control={control}
                    name="recentUsageDays"
                    label={t("Recent usage window")}
                    description={t("Carriers used within this many days are enrolled.")}
                    sideText={t("days")}
                    min={7}
                    max={365}
                    readOnly={readOnly}
                  />
                </FormControl>
                <FormControl className="min-h-0">
                  <SwitchField
                    control={control}
                    name="includeOpenTenders"
                    label={t("Include open tenders")}
                    description={t("Also enroll carriers with a tender that is still open.")}
                    position="left"
                    readOnly={readOnly}
                  />
                </FormControl>
              </>
            ) : null}
            {canManage && enrollmentPolicy !== "Manual" && providerConfigured ? (
              <FormControl cols="full" className="min-h-0">
                <CostEstimatePreview open={open} />
              </FormControl>
            ) : null}
            <FormControl cols="full" className="min-h-0">
              <SwitchField
                control={control}
                name="autoEnrollOnCreate"
                label={t("Enroll new carriers")}
                description={t("Enroll a carrier as soon as it is created.")}
                position="left"
                readOnly={readOnly}
              />
            </FormControl>
            <FormControl cols="full" className="min-h-0">
              <SwitchField
                control={control}
                name="autoUnenrollOnInactive"
                label={t("Unenroll inactive carriers")}
                description={t("Stop monitoring a carrier when it is marked inactive.")}
                position="left"
                readOnly={readOnly}
              />
            </FormControl>
            <FormControl cols="full" className="min-h-0">
              <SwitchField
                control={control}
                name="exclusiveWatchlist"
                label={t("Exclusive watchlist")}
                description={t(
                  "Remove carriers from the provider watchlist that Trenova did not enroll.",
                )}
                position="left"
                readOnly={readOnly}
              />
            </FormControl>
            <FormControl cols="full" className="min-h-0">
              <SwitchField
                control={control}
                name="selfMonitoringEnabled"
                label={t("Monitor our own authority")}
                description={t(
                  "Watch your organization's USDOT and MC record for changes and expirations.",
                )}
                position="left"
                readOnly={readOnly}
              />
            </FormControl>
          </FormGroup>
        </SettingsSection>
        <SettingsSection
          title={t("Refresh cadence")}
          description={t(
            "How often monitored carriers are polled and how long intelligence stays fresh.",
          )}
        >
          <FormGroup cols={3}>
            <FormControl>
              <NumberField
                control={control}
                name="pollIntervalMinutes"
                label={t("Poll interval")}
                sideText={t("minutes")}
                min={60}
                max={1440}
                readOnly={readOnly}
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name="snapshotTtlHours"
                label={t("Snapshot freshness")}
                sideText={t("hours")}
                min={1}
                max={720}
                readOnly={readOnly}
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name="fullProfileTtlDays"
                label={t("Full profile freshness")}
                sideText={t("days")}
                min={1}
                max={365}
                readOnly={readOnly}
              />
            </FormControl>
          </FormGroup>
        </SettingsSection>
      </fieldset>
      <MonitoringFeedStatus open={open} canManage={canManage} />
    </div>
  );
}

function CostEstimatePreview({ open }: { open: boolean }) {
  const t = useT();
  const { control } = useFormContext<CarrierIntelSettingsFormValues>();
  const [policy, recentUsageDays, includeOpenTenders] = useWatch({
    control,
    name: ["enrollmentPolicy", "recentUsageDays", "includeOpenTenders"],
  });
  const debouncedDays = useDebounce(recentUsageDays, 500);
  const validDays = typeof debouncedDays === "number" && debouncedDays >= 7 && debouncedDays <= 365;

  const estimateQuery = useQuery({
    ...queries.carrierIntelSettings.costEstimate({
      policy,
      recentUsageDays: policy === "RecentlyUsed" ? debouncedDays : undefined,
      includeOpenTenders: policy === "RecentlyUsed" ? includeOpenTenders : undefined,
    }),
    enabled: open && policy !== "Manual" && (policy !== "RecentlyUsed" || validDays),
  });

  return (
    <div
      className="border-border bg-muted/30 flex flex-wrap items-center justify-between gap-2 rounded-md border p-3 text-sm"
      data-testid="cost-estimate-preview"
    >
      <span className="text-muted-foreground">{t("Estimated monitoring cost")}</span>
      {estimateQuery.isLoading ? (
        <Skeleton className="h-4 w-40" />
      ) : estimateQuery.isError ? (
        <span className="text-destructive text-xs">{t("The estimate is unavailable.")}</span>
      ) : estimateQuery.data ? (
        <span className="font-medium">
          {t(
            "{0} carriers, about {1} per month",
            estimateQuery.data.subjectCount.toLocaleString(),
            formatOptionalDecimalCurrency(estimateQuery.data.monthlyMonitoring) ?? "-",
          )}
        </span>
      ) : (
        <span className="text-muted-foreground text-xs">{t("Enter a valid usage window.")}</span>
      )}
    </div>
  );
}

function MonitoringFeedStatus({ open, canManage }: { open: boolean; canManage: boolean }) {
  const t = useT();
  const queryClient = useQueryClient();
  const statusQuery = useQuery({
    ...queries.carrierIntelSettings.monitoringStatus(),
    enabled: open,
  });

  const resumeMutation = useApiMutation({
    mutationFn: resumeCarrierIntelMonitoring,
    resourceName: "Carrier monitoring",
    onSuccess: async () => {
      toast.success(t("Monitoring resumed"));
      await queryClient.invalidateQueries({
        queryKey: queries.carrierIntelSettings.monitoringStatus().queryKey,
      });
    },
  });

  const status = statusQuery.data;
  const hasPausedFeed = status?.feeds.some((feed) => Boolean(feed.pausedReason)) ?? false;

  return (
    <SettingsSection
      title={t("Feed status")}
      description={t("Enrollment and polling health for the primary provider.")}
      action={
        <div className="flex items-center gap-2">
          <Button
            type="button"
            size="sm"
            variant="ghost"
            onClick={() => void statusQuery.refetch()}
            disabled={statusQuery.isFetching}
            aria-label={t("Refresh feed status")}
          >
            <RefreshCwIcon className="size-3.5" />
          </Button>
          {hasPausedFeed && canManage ? (
            <Button
              type="button"
              size="sm"
              onClick={() => resumeMutation.mutate()}
              isLoading={resumeMutation.isPending}
              loadingText={t("Resuming...")}
            >
              {t("Resume monitoring")}
            </Button>
          ) : null}
        </div>
      }
    >
      {statusQuery.isLoading ? (
        <Skeleton className="h-32 w-full" />
      ) : statusQuery.isError || !status ? (
        <div className="border-border text-destructive rounded-md border p-3 text-sm">
          {t("Feed status could not be loaded.")}
        </div>
      ) : (
        <div className="space-y-3">
          <KpiStrip minItemWidth="7rem">
            <KpiStripItem
              label={t("Desired")}
              value={status.enrollmentCounts.desired.toLocaleString()}
            />
            <KpiStripItem
              label={t("Active")}
              value={status.enrollmentCounts.active.toLocaleString()}
            />
            <KpiStripItem
              label={t("Pending")}
              value={status.enrollmentCounts.pending.toLocaleString()}
            />
            <KpiStripItem
              label={t("Failed")}
              value={status.enrollmentCounts.failed.toLocaleString()}
              tone={status.enrollmentCounts.failed > 0 ? "danger" : undefined}
            />
            <KpiStripItem
              label={t("Review queue")}
              value={status.reviewQueueCount.toLocaleString()}
            />
          </KpiStrip>
          {status.feeds.length === 0 ? (
            <div className="border-border bg-muted/20 text-muted-foreground rounded-md border p-3 text-sm">
              {t(
                "No feeds have run yet. They start once a provider is connected and carriers are enrolled.",
              )}
            </div>
          ) : (
            <Table containerClassName="rounded-md border">
              <TableHeader>
                <TableRow>
                  <TableHead>{t("Feed")}</TableHead>
                  <TableHead>{t("Status")}</TableHead>
                  <TableHead>{t("Last success")}</TableHead>
                  <TableHead>{t("Next poll")}</TableHead>
                  <TableHead className="text-right">{t("Failures")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {status.feeds.map((feed) => (
                  <FeedRow key={`${feed.provider}-${feed.feedType}`} feed={feed} />
                ))}
              </TableBody>
            </Table>
          )}
        </div>
      )}
    </SettingsSection>
  );
}

function FeedRow({ feed }: { feed: CarrierIntelFeedState }) {
  const t = useT();

  return (
    <TableRow>
      <TableCell>
        <div className="flex flex-col">
          <span className="text-sm font-medium">
            {t(feedTypeLabels[feed.feedType] ?? feed.feedType)}
          </span>
          <span className="text-muted-foreground text-xs">
            {carrierIntelProviderLabel(feed.provider)}
          </span>
        </div>
      </TableCell>
      <TableCell className="whitespace-normal">
        {feed.pausedReason ? (
          <div className="space-y-0.5">
            <Badge variant="warning">{t("Paused")}</Badge>
            <p className="text-muted-foreground text-xs">{feed.pausedReason}</p>
            {feed.pausedAt ? (
              <p className="text-muted-foreground text-2xs">
                {t("Since {0}", formatUnixDateTimeOrDash(feed.pausedAt))}
              </p>
            ) : null}
          </div>
        ) : feed.lastError ? (
          <div className="space-y-0.5">
            <Badge variant="danger">{t("Failing")}</Badge>
            <p className="text-muted-foreground text-xs">{feed.lastError}</p>
          </div>
        ) : (
          <Badge variant="success">{t("Healthy")}</Badge>
        )}
      </TableCell>
      <TableCell className="text-xs">{formatUnixDateTimeOrDash(feed.lastSuccessAt)}</TableCell>
      <TableCell className="text-xs">{formatUnixDateTimeOrDash(feed.nextPollAfter)}</TableCell>
      <TableCell className="text-right text-xs">{feed.failureCount}</TableCell>
    </TableRow>
  );
}
