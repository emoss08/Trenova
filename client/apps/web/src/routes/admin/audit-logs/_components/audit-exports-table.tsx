import { DataTable } from "@/components/data-table/data-table";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import {
  AUDIT_EXPORT_LIST_KEY,
  auditExportTableGraphQLConfig,
  fetchAuditExportDownload,
  type AuditExportRow,
} from "@/lib/graphql/audit-log-admin";
import { Download01Icon, PlusIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { toast } from "@trenova/shared/components/ui/toast";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { downloadFromUrl } from "@trenova/shared/lib/utils";
import type { ColumnDef, RowAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useState } from "react";

const EXPORT_RANGE_FORMAT = { showTime: false, showTimeZone: false };
import { RequestAuditExportDialog } from "./request-audit-export-dialog";

const EXPORTS_REFRESH_MS = 30_000;

const STATUS_LABELS = defineLabels({
  Pending: "Queued",
  Running: "Writing",
  Succeeded: "Ready",
  Failed: "Failed",
  Expired: "Expired",
} as const);

const STATUS_VARIANTS = {
  Pending: "neutral",
  Running: "info",
  Succeeded: "success",
  Failed: "danger",
  Expired: "neutral",
} as const;

function exportColumns(
  t: TranslateFn,
  downloadingId: string | null,
  onDownload: (row: AuditExportRow) => void,
): ColumnDef<AuditExportRow>[] {
  return [
    {
      accessorKey: "createdAt",
      header: t("Requested"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      meta: { label: t("Requested"), apiField: "createdAt", sortable: true, filterable: false },
    },
    {
      id: "range",
      header: t("Range"),
      cell: ({ row }) =>
        t(
          "{0} to {1}",
          formatToUserTimezone(row.original.rangeFrom, EXPORT_RANGE_FORMAT),
          formatToUserTimezone(row.original.rangeTo, EXPORT_RANGE_FORMAT),
        ),
      meta: { label: t("Range"), sortable: false, filterable: false },
    },
    {
      accessorKey: "format",
      header: t("Format"),
      cell: ({ row }) => (row.original.format === "JSONL" ? t("JSON Lines") : t("CSV")),
      meta: { label: t("Format"), apiField: "format", sortable: true, filterable: false },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <Badge variant={STATUS_VARIANTS[row.original.status]}>
          {STATUS_LABELS[row.original.status]}
        </Badge>
      ),
      meta: { label: t("Status"), apiField: "status", sortable: true, filterable: false },
    },
    {
      accessorKey: "rowCount",
      header: t("Entries"),
      cell: ({ row }) => row.original.rowCount.toLocaleString(),
      meta: { label: t("Entries"), apiField: "rowCount", sortable: true, filterable: false },
    },
    {
      id: "download",
      header: "",
      cell: ({ row }) =>
        row.original.downloadable ? (
          <Button
            type="button"
            variant="outline"
            size="xs"
            isLoading={downloadingId === row.original.id}
            onClick={() => onDownload(row.original)}
          >
            <Download01Icon aria-hidden />
            {t("Download")}
          </Button>
        ) : row.original.status === "Failed" ? (
          <span className="text-muted-foreground text-xs">{row.original.errorMessage}</span>
        ) : null,
      meta: { label: t("Download"), sortable: false, filterable: false },
    },
  ];
}

export default function AuditExportsTable() {
  const t = useT();
  const [requestOpen, setRequestOpen] = useState(false);
  const [downloadingId, setDownloadingId] = useState<string | null>(null);
  const download = async (row: AuditExportRow) => {
    if (downloadingId !== null) {
      return;
    }
    setDownloadingId(row.id);
    try {
      const file = await fetchAuditExportDownload(row.id);
      downloadFromUrl(file.url, file.fileName);
    } catch (error) {
      toast.error(t("The file could not be downloaded"), {
        description: graphQLErrorMessage(error, t("Try again shortly.")),
      });
    } finally {
      setDownloadingId(null);
    }
  };

  const columns = exportColumns(t, downloadingId, (row) => void download(row));

  const contextMenuActions: RowAction<AuditExportRow>[] = [
    {
      id: "download",
      label: t("Download"),
      icon: Download01Icon,
      onClick: (row) => void download(row.original),
      hidden: (row) => !row.original.downloadable,
    },
  ];

  return (
    <>
      <DataTable<AuditExportRow>
        name="Audit Export"
        emptyTitle={t("No exports yet")}
        queryKey={AUDIT_EXPORT_LIST_KEY}
        graphql={auditExportTableGraphQLConfig}
        resource={Resource.AuditLog}
        columns={columns}
        contextMenuActions={contextMenuActions}
        enableExport={false}
        enableCreateAction={false}
        refetchIntervalMs={EXPORTS_REFRESH_MS}
        toolbar={{
          trailing: (
            <Button type="button" onClick={() => setRequestOpen(true)}>
              <PlusIcon aria-hidden />
              {t("New export")}
            </Button>
          ),
        }}
      />
      <RequestAuditExportDialog open={requestOpen} onOpenChange={setRequestOpen} />
    </>
  );
}
