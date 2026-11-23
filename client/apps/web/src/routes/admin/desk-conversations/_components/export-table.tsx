import { DataTable } from "@/components/data-table/data-table";
import {
  DESK_TRANSCRIPT_EXPORT_LIST_KEY,
  deskTranscriptExportTableGraphQLConfig,
  type DeskTranscriptExportRow,
} from "@/lib/graphql/desk-transcripts";
import { downloadDeskTranscriptExport } from "@/lib/desk-transcripts";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { toast } from "@trenova/shared/components/ui/toast";
import { Download01Icon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import type { DataTableEmptyStateRenderProps, RowAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { getExportColumns } from "./export-columns";

/** A file is written in the background; realtime moves the table, this is the fallback. */
const EXPORTS_REFRESH_MS = 30_000;

const EMPTY_COLUMNS = [
  { label: "Requested" },
  { label: "Asked by" },
  { label: "Covers" },
  { label: "Conversations", numeric: true },
  { label: "Status" },
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
              "Export a timeframe, or select conversations and download them. Each file can be downloaded by the person who asked for it for seven days.",
            )
      }
      columns={EMPTY_COLUMNS}
      onClearFilters={hasActiveFilters ? onClearFilters : undefined}
    />
  );
}

/**
 * Every export of Desk conversations the organization has asked for: who
 * asked, when, what it covered in words, how it went, how big it is and until
 * when it can be downloaded. Only the person who asked can download a file.
 */
export default function ExportTable({ canExport }: { canExport: boolean }) {
  const t = useT();
  const [downloadingId, setDownloadingId] = useState<string | null>(null);

  const download = async (row: DeskTranscriptExportRow) => {
    if (downloadingId !== null) {
      return;
    }
    setDownloadingId(row.id);
    try {
      await downloadDeskTranscriptExport(row.id);
    } catch (error) {
      toast.error(t("The file could not be downloaded"), {
        description: graphQLErrorMessage(error, t("Try again shortly.")),
      });
    } finally {
      setDownloadingId(null);
    }
  };

  const columns = getExportColumns(t, {
    canExport,
    downloadingId,
    onDownload: (row) => void download(row),
  });

  const contextMenuActions: RowAction<DeskTranscriptExportRow>[] = [
    {
      id: "download",
      label: t("Download"),
      icon: Download01Icon,
      onClick: (row) => download(row.original),
      hidden: (row) =>
        !canExport || row.original.status !== "Succeeded" || !row.original.downloadable,
    },
  ];

  return (
    <DataTable<DeskTranscriptExportRow>
      name="Desk transcript export"
      queryKey={DESK_TRANSCRIPT_EXPORT_LIST_KEY}
      graphql={deskTranscriptExportTableGraphQLConfig}
      resource={Resource.DeskTranscript}
      columns={columns}
      contextMenuActions={contextMenuActions}
      enableExport={false}
      enableCreateAction={false}
      refetchIntervalMs={EXPORTS_REFRESH_MS}
      renderEmptyState={(state) => <ExportsEmpty {...state} />}
    />
  );
}
