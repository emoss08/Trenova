import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import { downloadAIAuditExport } from "@/lib/ai-audit-exports";
import {
  AI_AUDIT_EXPORT_LIST_KEY,
  aiAuditExportTableGraphQLConfig,
  type AIAuditExportRow,
} from "@/lib/graphql/ai-audit";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { useT } from "@trenova/shared/i18n/use-t";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import type { DataTableEmptyStateRenderProps, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { DownloadIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getAuditExportColumns } from "./audit-export-columns";

/** Exports change as a background file is written; realtime moves the table, this is the fallback. */
const EXPORTS_REFRESH_MS = 30_000;

const EMPTY_COLUMNS = [
  { label: "Requested" },
  { label: "By" },
  { label: "Format" },
  { label: "Status" },
  { label: "Rows", numeric: true },
] as const;

function ExportsEmpty({ hasActiveFilters, onClearFilters }: DataTableEmptyStateRenderProps) {
  const t = useT();

  return (
    <EmptyTable
      className="py-10"
      title={hasActiveFilters ? t("Nothing matches") : t("No exports yet")}
      description={
        hasActiveFilters
          ? t("No export fits the filters. Clear them to see every one.")
          : t(
              "Export the trail from the Trail view. Each file is signed row by row, keeps its SHA-256 here, and can be downloaded by the person who asked for it until it expires.",
            )
      }
      columns={EMPTY_COLUMNS}
      onClearFilters={hasActiveFilters ? onClearFilters : undefined}
    />
  );
}

/**
 * Every file the trail has been exported to: who asked, for which range,
 * how it went, how many rows and bytes it holds, its SHA-256, whether its
 * chain can be checked end to end, and until when it can be downloaded.
 * Only the person who asked for a file can download it.
 */
export default function AuditExportsView() {
  const t = useT();
  const { allowed: canExport } = usePermission(Resource.AIAuditTrail, Operation.Export);
  const [downloadingId, setDownloadingId] = useState<string | null>(null);

  const download = useCallback(
    async (row: AIAuditExportRow) => {
      setDownloadingId(row.id);
      try {
        await downloadAIAuditExport(row.id);
      } catch (error) {
        toast.error(t("The file could not be downloaded"), {
          description: graphQLErrorMessage(error, t("Try again shortly.")),
        });
      } finally {
        setDownloadingId(null);
      }
    },
    [t],
  );

  const columns = useMemo(
    () =>
      getAuditExportColumns(t, {
        canExport,
        downloadingId,
        onDownload: (row) => void download(row),
      }),
    [canExport, download, downloadingId, t],
  );

  const contextMenuActions: RowAction<AIAuditExportRow>[] = [
    {
      id: "download",
      label: t("Download"),
      icon: DownloadIcon,
      onClick: (row) => void download(row.original),
      hidden: (row) =>
        !canExport || row.original.status !== "Succeeded" || !row.original.downloadable,
    },
  ];

  return (
    <DataTable<AIAuditExportRow>
      name="AI Audit Export"
      queryKey={AI_AUDIT_EXPORT_LIST_KEY}
      graphql={aiAuditExportTableGraphQLConfig}
      resource={Resource.AIAuditTrail}
      columns={columns}
      contextMenuActions={contextMenuActions}
      enableExport={false}
      enableCreateAction={false}
      refetchIntervalMs={EXPORTS_REFRESH_MS}
      renderEmptyState={(state) => <ExportsEmpty {...state} />}
    />
  );
}
