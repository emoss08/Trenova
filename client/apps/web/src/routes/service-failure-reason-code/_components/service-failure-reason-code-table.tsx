import { DataTable } from "@/components/data-table/data-table";
import { serviceFailureReasonCodeTableGraphQLConfig } from "@/lib/graphql/service-failure-reason-code-table";
import { apiService } from "@/services/api";
import type { RowAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import type { ServiceFailureReasonCode } from "@/types/service-failure-reason-code";
import { useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import { ArchiveIcon, RotateCcwIcon } from "lucide-react";
import { toast } from "sonner";
import { getColumns } from "./service-failure-reason-code-columns";
import { ServiceFailureReasonCodePanel } from "./service-failure-reason-code-panel";

export default function ServiceFailureReasonCodeTable() {
  const queryClient = useQueryClient();
  const columns = getColumns();

  const invalidate = () => {
    void queryClient.invalidateQueries({
      queryKey: ["service-failure-reason-code-list"],
    });
  };

  const handleArchive = async (row: Row<ServiceFailureReasonCode>) => {
    await apiService.serviceFailureReasonCodeService.archive(row.original.id ?? "");
    toast.success("Reason code archived");
    invalidate();
  };

  const handleActivate = async (row: Row<ServiceFailureReasonCode>) => {
    await apiService.serviceFailureReasonCodeService.activate(row.original.id ?? "");
    toast.success("Reason code activated");
    invalidate();
  };

  const contextMenuActions: RowAction<ServiceFailureReasonCode>[] = [
    {
      id: "archive",
      label: "Archive",
      icon: ArchiveIcon,
      variant: "destructive",
      onClick: (row) => void handleArchive(row),
      hidden: (row) => !row.original.active,
    },
    {
      id: "activate",
      label: "Reactivate",
      icon: RotateCcwIcon,
      onClick: (row) => void handleActivate(row),
      hidden: (row) => row.original.active,
    },
  ];

  return (
    <DataTable<ServiceFailureReasonCode>
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
