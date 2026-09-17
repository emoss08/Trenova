import { useT } from "@trenova/shared/i18n/use-t";
import { useCarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { DataTable } from "@/components/data-table/data-table";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { CARRIER_INTELLIGENCE_KEY, setCarrierMonitoring } from "@/lib/graphql/carrier-intelligence";
import {
  CARRIER_MONITORING_ENROLLMENT_LIST_KEY,
  carrierMonitoringEnrollmentTableGraphQLConfig,
  type CarrierMonitoringEnrollmentRow,
} from "@/lib/graphql/carrier-monitoring-table";
import { queries } from "@/lib/queries";
import { useQueryClient } from "@tanstack/react-query";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import type { DataTableEmptyStateRenderProps, DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { EyeIcon, EyeOffIcon } from "lucide-react";
import { useQueryState } from "nuqs";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getEnrollmentColumns } from "./enrollment-columns";
import {
  enrollmentFilterForView,
  enrollmentViewParser,
  monitorableCarrierIds,
  type EnrollmentView,
} from "./enrollment-filter";

export type EnrollmentTableProps = {
  canUpdate: boolean;
};

export function EnrollmentTable({ canUpdate }: EnrollmentTableProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const queryClient = useQueryClient();
  const [view, setView] = useQueryState("enrollmentView", enrollmentViewParser);

  const graphql = useMemo(
    () => carrierMonitoringEnrollmentTableGraphQLConfig(enrollmentFilterForView(view)),
    [view],
  );
  const columns = useMemo(() => getEnrollmentColumns(t, labels), [labels, t]);

  const invalidate = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [CARRIER_MONITORING_ENROLLMENT_LIST_KEY] }),
      queryClient.invalidateQueries({ queryKey: [CARRIER_INTELLIGENCE_KEY] }),
      queryClient.invalidateQueries({
        queryKey: queries.carrierIntelSettings.monitoringStatus().queryKey,
      }),
    ]);
  }, [queryClient]);

  const applyMonitoring = useCallback(
    async (rows: CarrierMonitoringEnrollmentRow[], enabled: boolean) => {
      const carrierIds = monitorableCarrierIds(rows);
      if (carrierIds.length === 0) {
        toast.info(t("None of the selected rows are carriers"), {
          description: t(
            "Monitoring for your own authority and customer brokers is managed from their own settings.",
          ),
        });
        return;
      }

      let changed: number;
      try {
        changed = await setCarrierMonitoring(carrierIds, enabled);
      } catch (error) {
        handleMutationError({ error, resourceName: "Carrier monitoring" });
        throw error;
      }

      toast.success(
        enabled
          ? t(
              "{0, plural, one {# carrier set to monitor} other {# carriers set to monitor}}",
              changed,
            )
          : t(
              "{0, plural, one {# carrier removed from monitoring} other {# carriers removed from monitoring}}",
              changed,
            ),
        {
          description: t("The provider watchlist catches up on the next monitoring sync."),
        },
      );
      await invalidate();
    },
    [invalidate, t],
  );

  const dockActions = useMemo<DockAction<CarrierMonitoringEnrollmentRow>[]>(() => {
    if (!canUpdate) {
      return [];
    }
    return [
      {
        id: "enable-monitoring",
        label: t("Monitor"),
        loadingLabel: t("Enrolling..."),
        icon: EyeIcon,
        onClick: (rows) => applyMonitoring(rows, true),
        clearSelectionOnSuccess: true,
      },
      {
        id: "disable-monitoring",
        label: t("Stop monitoring"),
        loadingLabel: t("Unenrolling..."),
        icon: EyeOffIcon,
        variant: "destructive",
        onClick: (rows) => applyMonitoring(rows, false),
        clearSelectionOnSuccess: true,
      },
    ];
  }, [applyMonitoring, canUpdate, t]);

  const renderEmptyState = ({
    hasActiveFilters,
    onClearFilters,
  }: DataTableEmptyStateRenderProps) => (
    <EmptyTable
      className="py-10"
      title={
        hasActiveFilters
          ? t("Nothing matches")
          : view === "all"
            ? t("No carriers are enrolled")
            : t("Nothing in this view")
      }
      description={
        hasActiveFilters
          ? t("No enrollment fits the search and filters. Clear them to see every one.")
          : view === "all"
            ? t(
                "Carriers are enrolled by the enrollment policy in Integrations, when they are created, or by hand from the carrier's Intelligence tab.",
              )
            : t("No enrollment is in this state right now. Switch views to see the rest.")
      }
      columns={[
        { label: t("Carrier") },
        { label: t("Desired / Provider") },
        { label: t("Reason") },
        { label: t("Failures"), numeric: true },
      ]}
      onClearFilters={hasActiveFilters ? onClearFilters : undefined}
    />
  );

  return (
    <div className="flex flex-col gap-3">
      <SegmentedControl<EnrollmentView>
        items={[
          { value: "monitored", label: t("Monitored") },
          { value: "attention", label: t("Out of sync") },
          { value: "failed", label: t("Failed") },
          { value: "all", label: t("All") },
        ]}
        value={view}
        onValueChange={(next) => void setView(next === "monitored" ? null : next)}
        aria-label={t("Enrollment view")}
        className="self-start"
      />
      <DataTable<CarrierMonitoringEnrollmentRow>
        name="Carrier Monitoring Enrollment"
        queryKey={CARRIER_MONITORING_ENROLLMENT_LIST_KEY}
        resource={Resource.CarrierIntelligence}
        columns={columns}
        graphql={graphql}
        enableCreateAction={false}
        enableRowSelection={dockActions.length > 0}
        dockActions={dockActions}
        renderEmptyState={renderEmptyState}
      />
    </div>
  );
}
