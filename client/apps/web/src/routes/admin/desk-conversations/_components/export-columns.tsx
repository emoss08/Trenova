import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type {
  DeskTranscriptExportRow,
  DeskTranscriptExportStatus,
} from "@/lib/graphql/desk-transcripts";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { phaseTone, type BadgeAttrProps } from "@trenova/shared/lib/status-phase";
import { formatFileSize } from "@trenova/shared/lib/utils";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { Download01Icon } from "@trenova/shared/components/icons";
import { describeCoverage, describeOwners } from "./export-coverage";

export function exportStatusAttrs(
  t: TranslateFn,
): Record<DeskTranscriptExportStatus, BadgeAttrProps> {
  return {
    Pending: { phase: "queued", text: t("Queued") },
    Running: { phase: "active", text: t("Writing") },
    Succeeded: { phase: "complete", text: t("Ready") },
    Failed: { phase: "failed", text: t("Failed") },
    Expired: { phase: "closed", text: t("Expired") },
  };
}

export type ExportColumnOptions = {
  canExport: boolean;
  downloadingId: string | null;
  onDownload: (row: DeskTranscriptExportRow) => void;
};

function DownloadCell({
  row,
  t,
  options,
}: {
  row: DeskTranscriptExportRow;
  t: TranslateFn;
  options: ExportColumnOptions;
}) {
  if (row.status !== "Succeeded") {
    return <span className="text-foreground-subtle">—</span>;
  }
  if (!options.canExport || !row.downloadable) {
    return (
      <span className="text-foreground-muted text-xs">
        {t("Only the person who asked can download it")}
      </span>
    );
  }

  return (
    <Button
      type="button"
      size="xs"
      isLoading={options.downloadingId === row.id}
      onClick={(event) => {
        event.stopPropagation();
        options.onDownload(row);
      }}
    >
      <Download01Icon className="size-3.5" />
      {t("Download")}
    </Button>
  );
}

export function getExportColumns(
  t: TranslateFn,
  options: ExportColumnOptions,
): ColumnDef<DeskTranscriptExportRow>[] {
  const statuses = exportStatusAttrs(t);

  return [
    {
      accessorKey: "createdAt",
      header: t("Requested"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 160,
      meta: { label: t("Requested"), apiField: "createdAt", filterable: false, sortable: true },
    },
    {
      accessorKey: "requestedByName",
      header: t("Asked by"),
      cell: ({ row }) => <span>{row.original.requestedByName}</span>,
      size: 160,
      enableSorting: false,
      meta: { label: t("Asked by"), filterable: false, sortable: false },
    },
    {
      id: "coverage",
      header: t("Covers"),
      cell: ({ row }) => {
        const owners = describeOwners(t, row.original.coverage);
        return (
          <span className="flex min-w-0 flex-col">
            <span className="truncate">{describeCoverage(t, row.original.coverage)}</span>
            {owners ? (
              <span className="text-foreground-muted truncate text-xs" title={owners}>
                {owners}
              </span>
            ) : null}
          </span>
        );
      },
      size: 300,
      enableSorting: false,
      meta: { label: t("Covers"), filterable: false, sortable: false },
    },
    {
      accessorKey: "conversationCount",
      header: t("Conversations"),
      cell: ({ row }) => (
        <span className="tabular-nums">{row.original.conversationCount.toLocaleString()}</span>
      ),
      size: 120,
      meta: {
        label: t("Conversations"),
        apiField: "conversationCount",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => {
        const entry = statuses[row.original.status];
        return (
          <span className="flex min-w-0 flex-col items-start gap-0.5">
            <Badge variant={phaseTone(entry.phase)}>{entry.text}</Badge>
            {row.original.status === "Failed" && row.original.errorMessage ? (
              <span
                className="text-foreground-muted truncate text-xs"
                title={row.original.errorMessage}
              >
                {row.original.errorMessage}
              </span>
            ) : null}
          </span>
        );
      },
      size: 170,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: (["Pending", "Running", "Succeeded", "Failed", "Expired"] as const).map(
          (status) => ({ value: status, label: statuses[status].text }),
        ),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "byteSize",
      header: t("Size"),
      cell: ({ row }) =>
        row.original.byteSize > 0 ? (
          <span className="tabular-nums">{formatFileSize(row.original.byteSize)}</span>
        ) : (
          <span className="text-foreground-subtle">—</span>
        ),
      size: 90,
      meta: { label: t("Size"), apiField: "byteSize", filterable: false, sortable: true },
    },
    {
      accessorKey: "artifactExpiresAt",
      header: t("Expires"),
      cell: ({ row }) =>
        row.original.artifactExpiresAt ? (
          <HoverCardTimestamp timestamp={row.original.artifactExpiresAt} />
        ) : (
          <span className="text-foreground-subtle">—</span>
        ),
      size: 160,
      meta: {
        label: t("Expires"),
        apiField: "artifactExpiresAt",
        filterable: false,
        sortable: true,
      },
    },
    {
      id: "download",
      header: t("File"),
      cell: ({ row }) => <DownloadCell row={row.original} t={t} options={options} />,
      size: 200,
      enableSorting: false,
      meta: { label: t("File"), filterable: false, sortable: false },
    },
  ];
}
