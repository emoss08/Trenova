import { DataTable } from "@/components/data-table/data-table";
import {
  downloadReportRun,
  isReportRunActive,
  REPORT_RUN_LIST_QUERY_KEY,
  useCancelReportRun,
} from "@/hooks/use-reports";
import { usePermission } from "@/hooks/use-permission";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { reportRunsTableGraphQLConfig, type ReportRun } from "@/lib/graphql/reports";
import type { RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { BanIcon, DownloadIcon } from "lucide-react";
import { useMemo } from "react";
import { toast } from "sonner";
import { getReportRunColumns } from "./report-run-columns";

function isDownloadable(run: ReportRun): boolean {
  if (run.status !== "succeeded") return false;
  return !run.artifactExpiresAt || run.artifactExpiresAt > Math.floor(Date.now() / 1000);
}

export default function ReportRunsTable({ definitionId }: { definitionId?: string }) {
  const cancelRun = useCancelReportRun();
  const { allowed: canExport } = usePermission(Resource.Report, Operation.Export);

  const columns = useMemo(() => getReportRunColumns(), []);
  const graphql = useMemo(
    () => reportRunsTableGraphQLConfig(definitionId ? { definitionId } : undefined),
    [definitionId],
  );

  const contextMenuActions = useMemo<RowAction<ReportRun>[]>(
    () => [
      {
        id: "download",
        label: "Download",
        icon: DownloadIcon,
        hidden: (row) => !canExport || !isDownloadable(row.original),
        onClick: (row) => downloadReportRun(row.original),
      },
      {
        id: "cancel",
        label: "Cancel Run",
        icon: BanIcon,
        variant: "destructive",
        hidden: (row) => !isReportRunActive(row.original.status),
        onClick: (row) => {
          cancelRun.mutate(row.original.id, {
            onSuccess: () => toast.success("Report run canceled"),
            onError: (error) => toast.error(graphQLErrorMessage(error, "Failed to cancel the run")),
          });
        },
      },
    ],
    [canExport, cancelRun],
  );

  return (
    <DataTable<ReportRun>
      name="Report Run"
      queryKey={REPORT_RUN_LIST_QUERY_KEY}
      graphql={graphql}
      resource={Resource.Report}
      columns={columns}
      contextMenuActions={contextMenuActions}
      enableCreateAction={false}
      onRowClick={(row) => {
        if (canExport && isDownloadable(row.original)) {
          downloadReportRun(row.original);
        }
      }}
    />
  );
}
