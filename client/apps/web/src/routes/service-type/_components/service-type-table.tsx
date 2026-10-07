import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import {
  serviceTypeTableGraphQLConfig,
  type ServiceTypeRow,
} from "@/lib/graphql/service-type-table";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { CellEditCommitFn } from "@trenova/shared/lib/cell-editing-feature";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import type { ServiceType } from "@/types/service-type";
import { useQueryClient } from "@tanstack/react-query";
import { CheckCircleIcon } from "@trenova/shared/components/icons";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns } from "./service-type-columns";
import { ServiceTypePanel } from "./service-type-panel";
import { translate } from "@trenova/shared/i18n/runtime";

const INLINE_EDITABLE_FIELDS = new Set<keyof ServiceTypeRow>(["code", "description"]);

export default function EquipmentTypeTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);

  const handleBulkStatusUpdate = useCallback(
    async (rows: ServiceTypeRow[], status: string) => {
      const ids = rows.map((r) => r.id);
      toast.promise(
        apiService.serviceTypeService.bulkUpdateStatus({
          serviceTypeIds: ids as string[],
          status: status as ServiceType["status"],
        }),
        {
          loading: translate("Updating status..."),
          success: translate("Status updated successfully"),
          error: translate("Failed to update status"),
          finally: async () => {
            await queryClient.invalidateQueries({
              queryKey: ["service-type-list"],
              refetchType: "all",
            });
          },
        },
      );
    },
    [queryClient],
  );

  const handleCellEditCommit = useCallback<CellEditCommitFn<ServiceTypeRow>>(
    async ({ rowId, columnId, value }) => {
      const field = columnId as keyof ServiceTypeRow;
      if (!INLINE_EDITABLE_FIELDS.has(field)) return;
      if (field === "code" && (value === null || value === "")) {
        throw new Error("Code is required.");
      }

      await apiService.serviceTypeService.patch(rowId, { [field]: value });
      await queryClient.invalidateQueries({
        queryKey: ["service-type-list"],
        refetchType: "all",
      });
      toast.success(t("Service type updated"));
    },
    [queryClient, t],
  );

  const dockActions = useMemo<DockAction<ServiceTypeRow>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: t("Update status"),
        loadingLabel: t("Updating..."),
        icon: CheckCircleIcon,
        options: statusChoices,
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate, t],
  );

  return (
    <DataTable<ServiceTypeRow>
      name="Service Type"
      queryKey="service-type-list"
      graphql={serviceTypeTableGraphQLConfig}
      resource={Resource.ServiceType}
      columns={columns}
      dockActions={dockActions}
      TablePanel={ServiceTypePanel}
      enableRowSelection
      onCellEditCommit={handleCellEditCommit}
    />
  );
}
