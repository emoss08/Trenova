import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { notifyBulkOutcome, settleAll } from "@/lib/bulk-outcome";
import {
  payProfileTableGraphQLConfig,
  updatePayProfile,
  type PayProfileRow,
} from "@/lib/graphql/driver-settlement";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CheckCircleIcon } from "@trenova/shared/components/icons";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { getColumns, payProfileStatusInput } from "./pay-profile-columns";
import { PayProfilePanel } from "./pay-profile-panel";
import { translate } from "@trenova/shared/i18n/runtime";

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
      const outcome = await settleAll(eligible, (row) =>
        updatePayProfile(payProfileStatusInput(row, status as "Active" | "Inactive")),
      );
      notifyBulkOutcome(
        outcome,
        status === "Active"
          ? {
              succeeded: (count) =>
                translate(
                  "{0, plural, one {# pay profile activated} other {# pay profiles activated}}",
                  count,
                ),
              partial: (succeeded, failed) =>
                translate(
                  "{0, plural, one {# pay profile activated} other {# pay profiles activated}}, {1} failed",
                  succeeded,
                  failed,
                ),
              allFailed: (failed) =>
                translate(
                  "{0, plural, one {The selected pay profile failed} other {All # selected pay profiles failed}}",
                  failed,
                ),
            }
          : {
              succeeded: (count) =>
                translate(
                  "{0, plural, one {# pay profile deactivated} other {# pay profiles deactivated}}",
                  count,
                ),
              partial: (succeeded, failed) =>
                translate(
                  "{0, plural, one {# pay profile deactivated} other {# pay profiles deactivated}}, {1} failed",
                  succeeded,
                  failed,
                ),
              allFailed: (failed) =>
                translate(
                  "{0, plural, one {The selected pay profile failed} other {All # selected pay profiles failed}}",
                  failed,
                ),
            },
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
        label: t("Update status"),
        loadingLabel: t("Updating..."),
        icon: CheckCircleIcon,
        options: [
          {
            value: "Active",
            label: t("Activate"),
            color: "var(--success)",
            description: t("Profiles become assignable to drivers again."),
          },
          {
            value: "Inactive",
            label: t("Deactivate"),
            color: "var(--danger)",
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
      emptyTitle={t("No pay profiles yet")}
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
