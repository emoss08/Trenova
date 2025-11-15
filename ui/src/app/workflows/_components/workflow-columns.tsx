/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { Badge } from "@/components/ui/badge";
import {
  type WorkflowSchema,
  type WorkflowStatusType,
  type TriggerTypeType,
} from "@/lib/schemas/workflow-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

const workflowStatusConfig: Record<
  WorkflowStatusType,
  { label: string; variant: "default" | "success" | "warning" | "destructive" }
> = {
  draft: { label: "Draft", variant: "default" },
  active: { label: "Active", variant: "success" },
  inactive: { label: "Inactive", variant: "warning" },
  archived: { label: "Archived", variant: "destructive" },
};

const triggerTypeLabels: Record<TriggerTypeType, string> = {
  manual: "Manual",
  scheduled: "Scheduled",
  shipment_status: "Shipment Status",
  document_uploaded: "Document Upload",
  entity_created: "Entity Created",
  entity_updated: "Entity Updated",
  webhook: "Webhook",
};

export function getColumns(): ColumnDef<WorkflowSchema>[] {
  const columnHelper = createColumnHelper<WorkflowSchema>();
  const commonColumns = createCommonColumns<WorkflowSchema>();

  return [
    columnHelper.display({
      id: "status",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.status;
        const config = workflowStatusConfig[status];
        return <Badge variant={config.variant}>{config.label}</Badge>;
      },
    }),
    columnHelper.display({
      id: "name",
      header: "Name",
      cell: ({ row }) => <p className="font-medium">{row.original.name}</p>,
    }),
    commonColumns.description,
    columnHelper.display({
      id: "triggerType",
      header: "Trigger Type",
      cell: ({ row }) => (
        <Badge variant="outline">
          {triggerTypeLabels[row.original.triggerType]}
        </Badge>
      ),
    }),
    columnHelper.display({
      id: "versioning",
      header: "Version",
      cell: ({ row }) => {
        const hasPublished = !!row.original.publishedVersionId;
        return (
          <div className="flex flex-col gap-1">
            <span className="text-sm">
              {hasPublished ? "Published" : "Not Published"}
            </span>
          </div>
        );
      },
    }),
    commonColumns.createdAt,
    commonColumns.updatedAt,
  ];
}
