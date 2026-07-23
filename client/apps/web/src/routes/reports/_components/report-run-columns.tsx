import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { ReportRun } from "@/lib/graphql/reports";
import { formatFileSize } from "@/lib/utils";
import { REPORT_RUN_STATUS_LABELS, REPORT_RUN_TRIGGER_LABELS } from "@/types/report";
import type { ColumnDef } from "@tanstack/react-table";
import { CircleAlertIcon, ZapIcon } from "lucide-react";
import { ReportFormatBadge, ReportRunStatusBadge } from "./report-badges";

const runStatusChoices = Object.entries(REPORT_RUN_STATUS_LABELS).map(([value, label]) => ({
  value,
  label,
}));

function formatDuration(durationMs: number): string {
  if (durationMs <= 0) return "-";
  if (durationMs < 1000) return `${durationMs}ms`;
  const seconds = durationMs / 1000;
  if (seconds < 60) return `${seconds.toFixed(1)}s`;
  const minutes = Math.floor(seconds / 60);
  return `${minutes}m ${Math.round(seconds % 60)}s`;
}

function StatusCell({ run }: { run: ReportRun }) {
  return (
    <div className="flex items-center gap-1.5">
      <ReportRunStatusBadge status={run.status} />
      {run.error && (
        <Tooltip>
          <TooltipTrigger>
            <CircleAlertIcon className="size-4 text-destructive" />
          </TooltipTrigger>
          <TooltipContent className="max-w-sm">
            <p className="font-medium">{run.error.code}</p>
            <p>{run.error.message}</p>
          </TooltipContent>
        </Tooltip>
      )}
      {run.cacheHit && (
        <Tooltip>
          <TooltipTrigger>
            <ZapIcon className="size-4 text-yellow-500" />
          </TooltipTrigger>
          <TooltipContent>Served from the result cache</TooltipContent>
        </Tooltip>
      )}
    </div>
  );
}

export function getReportRunColumns(): ColumnDef<ReportRun>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <StatusCell run={row.original} />,
      size: 160,
      minSize: 130,
      maxSize: 200,
      meta: {
        label: "Status",
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: runStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "format",
      header: "Format",
      cell: ({ row }) => <ReportFormatBadge format={row.original.format} />,
      size: 90,
      minSize: 80,
      maxSize: 120,
      meta: {
        label: "Format",
        apiField: "format",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "trigger",
      header: "Trigger",
      cell: ({ row }) => (
        <p className="text-muted-foreground">
          {REPORT_RUN_TRIGGER_LABELS[row.original.trigger] ?? row.original.trigger}
        </p>
      ),
      size: 100,
      minSize: 90,
      maxSize: 130,
      meta: {
        label: "Trigger",
        apiField: "trigger",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "rowCount",
      header: "Rows",
      cell: ({ row }) => {
        const run = row.original;
        if (run.status !== "succeeded" && run.status !== "expired") {
          return <p className="text-muted-foreground">-</p>;
        }
        return (
          <p>
            {run.rowCount.toLocaleString()}
            {run.truncated && <span className="text-warning"> (truncated)</span>}
          </p>
        );
      },
      size: 110,
      minSize: 90,
      maxSize: 150,
      meta: {
        label: "Rows",
        apiField: "row_count",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "byteSize",
      header: "Size",
      cell: ({ row }) =>
        row.original.byteSize > 0 ? (
          <p>{formatFileSize(row.original.byteSize)}</p>
        ) : (
          <p className="text-muted-foreground">-</p>
        ),
      size: 100,
      minSize: 80,
      maxSize: 130,
      meta: {
        label: "Size",
        apiField: "byte_size",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "durationMs",
      header: "Duration",
      cell: ({ row }) => <p>{formatDuration(row.original.durationMs)}</p>,
      size: 100,
      minSize: 90,
      maxSize: 130,
      meta: {
        label: "Duration",
        apiField: "duration_ms",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "createdAt",
      header: "Requested At",
      cell: ({ row }) => (
        <HoverCardTimestamp className="shrink-0" timestamp={row.original.createdAt} />
      ),
      size: 180,
      minSize: 150,
      maxSize: 220,
      meta: {
        label: "Requested At",
        apiField: "createdAt",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "artifactExpiresAt",
      header: "Expires",
      cell: ({ row }) =>
        row.original.artifactExpiresAt ? (
          <HoverCardTimestamp className="shrink-0" timestamp={row.original.artifactExpiresAt} />
        ) : (
          <p className="text-muted-foreground">-</p>
        ),
      size: 170,
      minSize: 140,
      maxSize: 220,
      meta: {
        label: "Expires",
        apiField: "artifact_expires_at",
        filterable: false,
        sortable: false,
      },
    },
  ];
}
