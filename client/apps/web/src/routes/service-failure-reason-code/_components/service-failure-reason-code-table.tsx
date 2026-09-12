import { useMemo } from "react";
import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import {
  serviceFailureReasonCodeTableGraphQLConfig,
  type ServiceFailureReasonCodeRow,
} from "@/lib/graphql/service-failure-reason-code-table";
import { apiService } from "@/services/api";
import type { RowAction, Row } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { ArchiveIcon, RotateCcwIcon } from "lucide-react";
import { toast } from "sonner";
import { getColumns } from "./service-failure-reason-code-columns";
import { ServiceFailureReasonCodePanel } from "./service-failure-reason-code-panel";

export default function ServiceFailureReasonCodeTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(t), [t]);

  const invalidate = () => {
    void queryClient.invalidateQueries({
      queryKey: ["service-failure-reason-code-list"],
    });
  };

  const handleArchive = async (row: Row<ServiceFailureReasonCodeRow>) => {
    await apiService.serviceFailureReasonCodeService.archive(row.original.id);
    toast.success(t("Reason code archived"));
    invalidate();
  };

  const handleActivate = async (row: Row<ServiceFailureReasonCodeRow>) => {
    await apiService.serviceFailureReasonCodeService.activate(row.original.id);
    toast.success(t("Reason code activated"));
    invalidate();
  };

  const contextMenuActions: RowAction<ServiceFailureReasonCodeRow>[] = [
    {
      id: "archive",
      label: t("Archive"),
      icon: ArchiveIcon,
      variant: "destructive",
      onClick: (row) => void handleArchive(row),
      hidden: (row) => !row.original.active,
    },
    {
      id: "activate",
      label: t("Reactivate"),
      icon: RotateCcwIcon,
      onClick: (row) => void handleActivate(row),
      hidden: (row) => row.original.active,
    },
  ];

  return (
    <DataTable<ServiceFailureReasonCodeRow>
      name="Service Failure Reason Code"
      queryKey="service-failure-reason-code-list"
      graphql={serviceFailureReasonCodeTableGraphQLConfig}
      resource={Resource.ServiceFailureReasonCode}
      columns={columns}
      contextMenuActions={contextMenuActions}
      TablePanel={ServiceFailureReasonCodePanel}
    />
  );
}
