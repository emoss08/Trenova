import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { Badge } from "@/components/ui/badge";
import {
  type ExecutionStatusType,
  type WorkflowExecutionSchema,
} from "@/lib/schemas/workflow-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";
import { formatDistanceToNow } from "date-fns";

const executionStatusConfig: Record<
  ExecutionStatusType,
  { label: string; variant: "default" | "success" | "warning" | "destructive" }
> = {
  pending: { label: "Pending", variant: "default" },
  running: { label: "Running", variant: "warning" },
  completed: { label: "Completed", variant: "success" },
  failed: { label: "Failed", variant: "destructive" },
  cancelled: { label: "Cancelled", variant: "default" },
  timeout: { label: "Timeout", variant: "destructive" },
};

export function getExecutionColumns(): ColumnDef<WorkflowExecutionSchema>[] {
  const columnHelper = createColumnHelper<WorkflowExecutionSchema>();
  const commonColumns = createCommonColumns<WorkflowExecutionSchema>();

  return [
    columnHelper.display({
      id: "status",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.status;
        const config = executionStatusConfig[status];
        return <Badge variant={config.variant}>{config.label}</Badge>;
      },
    }),
    columnHelper.display({
      id: "workflowName",
      header: "Workflow",
      cell: ({ row }) => (
        <p className="font-medium">{row.original.workflowId}</p>
      ),
    }),
    columnHelper.display({
      id: "triggeredBy",
      header: "Triggered By",
      cell: ({ row }) => (
        <p className="text-sm">{row.original.triggeredBy || "System"}</p>
      ),
    }),
    columnHelper.display({
      id: "duration",
      header: "Duration",
      cell: ({ row }) => {
        const { startedAt, completedAt } = row.original;
        if (!startedAt) return <span className="text-muted-foreground">-</span>;

        if (!completedAt) {
          return (
            <span className="text-sm">
              {formatDistanceToNow(new Date(startedAt), { addSuffix: false })}
            </span>
          );
        }

        const duration =
          new Date(completedAt).getTime() - new Date(startedAt).getTime();
        const seconds = Math.floor(duration / 1000);
        const minutes = Math.floor(seconds / 60);

        if (minutes > 0) {
          return (
            <span className="text-sm">
              {minutes}m {seconds % 60}s
            </span>
          );
        }
        return <span className="text-sm">{seconds}s</span>;
      },
    }),
    columnHelper.display({
      id: "error",
      header: "Error",
      cell: ({ row }) => {
        if (row.original.error) {
          return (
            <p className="max-w-xs truncate text-sm text-destructive">
              {row.original.error}
            </p>
          );
        }
        return <span className="text-muted-foreground">-</span>;
      },
    }),
    commonColumns.createdAt,
  ];
}
