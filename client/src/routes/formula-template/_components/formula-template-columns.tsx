import { EditableStatusBadge } from "@/components/editable-status-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@/components/ui/badge";
import {
  formulaTemplateStatusChoices,
  formulaTypeChoices,
} from "@/lib/choices";
import { patchFormulaTemplate } from "@/lib/formula-template-api";
import type {
  FormulaTemplate,
  FormulaTemplateStatus,
} from "@/types/formula-template";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import { FileCode2 } from "lucide-react";
import { useCallback } from "react";

// eslint-disable-next-line react-refresh/only-export-components
function FormulaTemplateStatusCell({
  id,
  status,
}: {
  id: FormulaTemplate["id"];
  status: FormulaTemplateStatus;
}) {
  "use no memo";
  const queryClient = useQueryClient();

  const handleStatusChange = useCallback(
    async (newStatus: FormulaTemplate["status"]) => {
      if (!id) return;
      await patchFormulaTemplate(id, {
        status: newStatus,
      });
      await queryClient.invalidateQueries({
        queryKey: ["formula-template-list"],
      });
    },
    [id, queryClient],
  );

  return (
    <EditableStatusBadge
      status={status}
      options={formulaTemplateStatusChoices}
      onStatusChange={handleStatusChange}
    />
  );
}

const TYPE_BADGE_VARIANT: Record<string, "info" | "purple"> = {
  FreightCharge: "info",
  AccessorialCharge: "purple",
};

export function getColumns(): ColumnDef<FormulaTemplate>[] {
  return [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <div className="flex items-center gap-3">
          <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary/10">
            <FileCode2 className="size-4 text-primary" />
          </div>
          <div className="min-w-0">
            <span className="text-sm font-medium">{row.original.name}</span>
            {row.original.description && (
              <p className="line-clamp-1 text-2xs text-muted-foreground">
                {row.original.description}
              </p>
            )}
          </div>
        </div>
      ),
      meta: {
        label: "Name",
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      size: 280,
      minSize: 220,
      maxSize: 400,
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <FormulaTemplateStatusCell
          id={row.original.id}
          status={row.original.status}
        />
      ),
      meta: {
        label: "Status",
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: formulaTemplateStatusChoices,
        defaultFilterOperator: "eq",
      },
      size: 120,
      minSize: 100,
      maxSize: 150,
    },
    {
      accessorKey: "type",
      header: "Type",
      cell: ({ row }) => {
        const typeLabel = formulaTypeChoices.find(
          (c) => c.value === row.original.type,
        )?.label;
        const variant = TYPE_BADGE_VARIANT[row.original.type] ?? "info";
        return <Badge variant={variant}>{typeLabel || row.original.type}</Badge>;
      },
      meta: {
        label: "Type",
        apiField: "type",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: formulaTypeChoices,
        defaultFilterOperator: "eq",
      },
      size: 160,
      minSize: 140,
      maxSize: 200,
    },
    {
      accessorKey: "version",
      header: "Version",
      cell: ({ row }) => (
        <Badge variant="outline" className="font-mono text-xs">
          v{row.original.version}
        </Badge>
      ),
      meta: {
        label: "Version",
        apiField: "version",
        filterable: false,
        sortable: true,
        filterType: "number",
      },
      size: 80,
      minSize: 60,
      maxSize: 100,
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => (
        <HoverCardTimestamp timestamp={row.original.createdAt} />
      ),
      meta: {
        label: "Created At",
        apiField: "createdAt",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "gt",
      },
      size: 180,
      minSize: 160,
      maxSize: 220,
    },
  ];
}
