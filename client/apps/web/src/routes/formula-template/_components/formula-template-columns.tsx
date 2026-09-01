import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { formulaTemplateStatusChoices, formulaTemplateTypeChoices } from "@/lib/choices";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import type { FormulaTemplate } from "@trenova/shared/types/formula-template";

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
          <div className="min-w-0">
            <span className="text-sm font-medium">{row.original.name}</span>
            {row.original.description && (
              <p className="text-2xs text-muted-foreground line-clamp-1">
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
      cell: ({ row }) => {
        const choice = formulaTemplateStatusChoices.find(
          (option) => option.value === row.original.status,
        );

        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.status
        );
      },
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
        const typeLabel = formulaTemplateTypeChoices.find(
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
        filterOptions: formulaTemplateTypeChoices,
        defaultFilterOperator: "eq",
      },
      size: 160,
      minSize: 140,
      maxSize: 200,
    },
    {
      accessorKey: "currentVersionNumber",
      header: "Version",
      cell: ({ row }) => (
        <Badge variant="outline" className="font-mono text-xs">
          {row.original.currentVersionNumber ? `v${row.original.currentVersionNumber}` : "—"}
        </Badge>
      ),
      meta: {
        label: "Version",
        apiField: "currentVersionNumber",
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
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
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
