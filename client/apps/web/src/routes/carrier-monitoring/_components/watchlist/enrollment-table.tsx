import { useCarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { DataTable } from "@/components/data-table/data-table";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { carrierPanelPath } from "@/lib/carrier-links";
import { CARRIER_INTELLIGENCE_KEY, setCarrierMonitoring } from "@/lib/graphql/carrier-intelligence";
import {
  CARRIER_MONITORING_ENROLLMENT_LIST_KEY,
  carrierMonitoringEnrollmentTableGraphQLConfig,
  type CarrierMonitoringEnrollmentRow,
} from "@/lib/graphql/carrier-monitoring-table";
import { queries } from "@/lib/queries";
import { useQueryClient } from "@tanstack/react-query";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { useT } from "@trenova/shared/i18n/use-t";
import type {
  DataTableEmptyStateRenderProps,
  DockAction,
  RowAction,
} from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { ExternalLinkIcon, EyeIcon, EyeOffIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { getEnrollmentColumns } from "./enrollment-columns";
import { monitorableCarrierIds } from "./enrollment-filter";

const ENROLLMENT_GRAPHQL = carrierMonitoringEnrollmentTableGraphQLConfig(null);

export type EnrollmentTableProps = {
  canUpdate: boolean;
};

export function EnrollmentTable({ canUpdate }: EnrollmentTableProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [pendingRowId, setPendingRowId] = useState<string | null>(null);

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
    async (rows: readonly CarrierMonitoringEnrollmentRow[], enabled: boolean) => {
      const carrierIds = monitorableCarrierIds(rows);
      if (carrierIds.length === 0) {
        toast.info(t("None of these rows are carriers"), {
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
        { description: t("The provider watchlist catches up on the next monitoring sync.") },
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

  const runRowAction = useCallback(
    async (row: CarrierMonitoringEnrollmentRow, enabled: boolean) => {
      setPendingRowId(row.id);
      try {
        await applyMonitoring([row], enabled);
      } catch {
        return;
      } finally {
        setPendingRowId(null);
      }
    },
    [applyMonitoring],
  );

  const contextMenuActions = useMemo<RowAction<CarrierMonitoringEnrollmentRow>[]>(
    () => [
      {
        id: "monitor",
        label: t("Monitor"),
        icon: EyeIcon,
        hidden: (row) =>
          !canUpdate || !row.original.carrierId || row.original.desiredState === "Enrolled",
        isPending: (row) => pendingRowId === row.original.id,
        onClick: (row) => runRowAction(row.original, true),
      },
      {
        id: "stop-monitoring",
        label: t("Stop monitoring"),
        icon: EyeOffIcon,
        variant: "destructive",
        hidden: (row) =>
          !canUpdate || !row.original.carrierId || row.original.desiredState !== "Enrolled",
        isPending: (row) => pendingRowId === row.original.id,
        onClick: (row) => runRowAction(row.original, false),
      },
      {
        id: "open-carrier",
        label: t("Open carrier"),
        icon: ExternalLinkIcon,
        hidden: (row) => !row.original.carrierId,
        onClick: (row) => {
          if (row.original.carrierId) {
            void navigate(carrierPanelPath(row.original.carrierId, "intelligence"));
          }
        },
      },
    ],
    [canUpdate, navigate, pendingRowId, runRowAction, t],
  );

  const renderEmptyState = ({
    hasActiveFilters,
    onClearFilters,
  }: DataTableEmptyStateRenderProps) => (
    <EmptyTable
      className="py-10"
      title={hasActiveFilters ? t("Nothing matches") : t("No carriers are on the watchlist")}
      description={
        hasActiveFilters
          ? t("No enrollment fits the search and filters. Clear them to see every one.")
          : t(
              "Carriers join the watchlist through the enrollment policy in Integrations, when they are created, or by hand from a carrier's Intelligence tab.",
            )
      }
      columns={[
        { label: t("Carrier") },
        { label: t("Desired") },
        { label: t("Provider") },
        { label: t("Failures"), numeric: true },
      ]}
      onClearFilters={hasActiveFilters ? onClearFilters : undefined}
    />
  );

  return (
    <DataTable<CarrierMonitoringEnrollmentRow>
      name="Carrier Monitoring Enrollment"
      queryKey={CARRIER_MONITORING_ENROLLMENT_LIST_KEY}
      resource={Resource.CarrierIntelligence}
      columns={columns}
      graphql={ENROLLMENT_GRAPHQL}
      enableCreateAction={false}
      enableRowSelection={dockActions.length > 0}
      dockActions={dockActions}
      contextMenuActions={contextMenuActions}
      renderEmptyState={renderEmptyState}
    />
  );
}
