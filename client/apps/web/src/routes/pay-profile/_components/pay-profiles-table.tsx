import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { runBulkAction } from "@/lib/bulk-run";
import {
  payProfileTableGraphQLConfig,
  updatePayProfile,
  type PayProfileRow,
} from "@/lib/graphql/driver-settlement";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns, payProfileStatusInput } from "./pay-profile-columns";
import { PayProfilePanel } from "./pay-profile-panel";

export default function PayProfilesTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);

  const handleBulkStatusUpdate = useCallback(
    async (rows: PayProfileRow[], status: string) => {
      const eligible = rows.filter((row) => row.status !== status);
      if (eligible.length === 0) {
        toast.info(t("Every selected pay profile already has that status."));
        return;
      }
      await runBulkAction(
        eligible,
        (row) => updatePayProfile(payProfileStatusInput(row, status as "Active" | "Inactive")),
        { noun: "pay profile", verb: status === "Active" ? "activated" : "deactivated" },
      );
      await queryClient.invalidateQueries({ queryKey: ["pay-profile-list"] });
    },
    [queryClient, t],
  );

  const dockActions = useMemo<DockAction<PayProfileRow>[]>(
    () => [
      {
        id: "status-update",
        type: "select",
        label: t("Update Status"),
        loadingLabel: t("Updating..."),
        icon: CircleCheckIcon,
        options: [
          {
            value: "Active",
            label: t("Activate"),
            color: "#15803d",
            description: t("Profiles become assignable to drivers again."),
          },
          {
            value: "Inactive",
            label: t("Deactivate"),
            color: "#dc2626",
            description: t("Profiles can no longer be assigned; existing assignments keep paying."),
          },
        ],
        onSelect: handleBulkStatusUpdate,
        clearSelectionOnSuccess: true,
      },
    ],
    [handleBulkStatusUpdate, t],
  );

  return (
    <DataTable<PayProfileRow>
      name="Pay Profile"
      queryKey="pay-profile-list"
      graphql={payProfileTableGraphQLConfig}
      resource={Resource.DriverPayProfile}
      columns={columns}
      dockActions={dockActions}
      enableRowSelection
      TablePanel={PayProfilePanel}
    />
  );
}
