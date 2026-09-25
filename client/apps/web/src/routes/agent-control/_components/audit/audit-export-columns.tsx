import { CopyIconButton } from "@/components/copy-icon-button";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { AIAuditExportRow } from "@/lib/graphql/ai-audit";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatRange } from "@trenova/shared/lib/date";
import { formatFileSize } from "@trenova/shared/lib/utils";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { DownloadIcon } from "lucide-react";
import { AuditExportStatusBadge } from "./audit-badges";
import { auditExportStatusAttrs } from "./audit-model";

const SHORT_HASH_LENGTH = 12;

export type AuditExportColumnOptions = {
  /** Whether the reader holds the right to download exports at all. */
  canExport: boolean;
  /** Which export's link is being fetched, so its button shows it. */
  downloadingId: string | null;
  onDownload: (row: AIAuditExportRow) => void;
};

function DownloadCell({
  row,
  t,
  options,
}: {
  row: AIAuditExportRow;
  t: TranslateFn;
  options: AuditExportColumnOptions;
}) {
  if (row.status !== "Succeeded") {
    return <span className="text-foreground-subtle">—</span>;
  }
  if (!options.canExport || !row.downloadable) {
    return (
      <span className="text-foreground-muted text-xs">
        {t("Only its requester can download it")}
      </span>
    );
  }

  return (
    <Button
      type="button"
      size="xs"
      variant="outline"
      isLoading={options.downloadingId === row.id}
      onClick={(event) => {
        event.stopPropagation();
        options.onDownload(row);
      }}
    >
      <DownloadIcon className="size-3.5" />
      {t("Download")}
    </Button>
  );
}

export function getAuditExportColumns(
  t: TranslateFn,
  options: AuditExportColumnOptions,
): ColumnDef<AIAuditExportRow>[] {
  const statuses = auditExportStatusAttrs(t);

  return [
    {
      accessorKey: "createdAt",
      header: t("Requested"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 160,
      meta: { label: t("Requested"), apiField: "createdAt", filterable: false, sortable: true },
    },
    {
      id: "requestedBy",
      accessorKey: "requestedByUserId",
      header: t("By"),
      cell: ({ row }) => <span>{row.original.requestedBy?.name ?? "—"}</span>,
      size: 160,
      meta: { label: t("By"), apiField: "requestedByUserId", filterable: false, sortable: false },
    },
    {
      accessorKey: "format",
      header: t("Format"),
      cell: ({ row }) => (
        <Badge variant={row.original.format === "CSV" ? "accent-teal" : "accent-violet"}>
          {row.original.format}
        </Badge>
      ),
      size: 100,
      meta: {
        label: t("Format"),
        apiField: "format",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: [
          { value: "CSV", label: t("CSV") },
          { value: "JSON", label: t("JSON") },
        ],
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "rangeFrom",
      header: t("Range"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {formatRange(row.original.rangeFrom, row.original.rangeTo)}
        </span>
      ),
      size: 190,
      meta: { label: t("Range"), apiField: "rangeFrom", filterable: false, sortable: true },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <span className="flex min-w-0 flex-col items-start gap-0.5">
          <AuditExportStatusBadge status={row.original.status} />
          {row.original.status === "Failed" && row.original.errorMessage ? (
            <span
              className="text-foreground-muted truncate text-xs"
              title={row.original.errorMessage}
            >
              {row.original.errorMessage}
            </span>
          ) : null}
        </span>
      ),
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
      accessorKey: "rowCount",
      header: t("Rows"),
      cell: ({ row }) => (
        <span className="tabular-nums">{row.original.rowCount.toLocaleString()}</span>
      ),
      size: 90,
      meta: {
        label: t("Rows"),
        apiField: "rowCount",
        filterable: true,
        sortable: true,
        filterType: "number",
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
      accessorKey: "sha256",
      header: t("SHA-256"),
      cell: ({ row }) =>
        row.original.sha256 ? (
          <span className="inline-flex items-center gap-1">
            <span title={row.original.sha256} className="text-foreground-muted font-mono text-xs">
              {row.original.sha256.slice(0, SHORT_HASH_LENGTH)}
            </span>
            <CopyIconButton value={row.original.sha256} label={t("Copy SHA-256")} size="icon-xxs" />
          </span>
        ) : (
          <span className="text-foreground-subtle">—</span>
        ),
      size: 170,
      meta: { label: t("SHA-256"), apiField: "sha256", filterable: false, sortable: false },
    },
    {
      accessorKey: "chainComplete",
      header: t("Chain"),
      cell: ({ row }) =>
        row.original.status !== "Succeeded" ? (
          <span className="text-foreground-subtle">—</span>
        ) : row.original.chainComplete ? (
          <Badge variant="success">{t("Complete")}</Badge>
        ) : (
          <Badge variant="neutral" appearance="outline">
            {t("Filtered")}
          </Badge>
        ),
      size: 110,
      meta: {
        label: t("Chain"),
        apiField: "chainComplete",
        filterable: true,
        sortable: true,
        filterType: "boolean",
      },
    },
    {
      accessorKey: "artifactExpiresAt",
      header: t("Expires"),
      cell: ({ row }) => (
        <HoverCardTimestamp timestamp={row.original.artifactExpiresAt ?? undefined} />
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
