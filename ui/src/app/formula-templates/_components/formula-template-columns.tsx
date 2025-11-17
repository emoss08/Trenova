import { HoverCardTimestamp } from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { statusChoices } from "@/lib/choices";
import type { FormulaTemplateSchema } from "@/lib/schemas/formula-template-schema";
import type { ColumnDef } from "@tanstack/react-table";

const categoryLabels: Record<string, string> = {
  BaseRate: "Base Rate",
  DistanceBased: "Distance Based",
  WeightBased: "Weight Based",
  DimensionalWeight: "Dimensional Weight",
  FuelSurcharge: "Fuel Surcharge",
  Accessorial: "Accessorial",
  TimeBasedRate: "Time Based",
  ZoneBased: "Zone Based",
  Custom: "Custom",
};

export function getColumns(): ColumnDef<FormulaTemplateSchema>[] {
  return [
    {
      accessorKey: "isActive",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.isActive ? "Active" : "Inactive";
        return <StatusBadge status={status} />;
      },
      meta: {
        apiField: "isActive",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "name",
      header: "Name",
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      cell: ({ row }) => (
        <div className="flex flex-col gap-0.5 leading-tight">
          <div className="flex items-center gap-2">
            <p className="text-sm font-medium">{row.original.name}</p>
            {row.original.isDefault && (
              <Badge variant="secondary" className="text-xs">
                Default
              </Badge>
            )}
          </div>
          {row.original.description && (
            <p className="text-xs text-muted-foreground line-clamp-1">
              {row.original.description}
            </p>
          )}
        </div>
      ),
    },
    {
      accessorKey: "category",
      header: "Category",
      meta: {
        apiField: "category",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: Object.keys(categoryLabels).map((key) => ({
          value: key,
          label: categoryLabels[key as keyof typeof categoryLabels],
        })),
        defaultFilterOperator: "eq",
      },
      cell: ({ row }) => {
        const category = row.original.category;
        return (
          <Badge variant="outline">
            {categoryLabels[category as keyof typeof categoryLabels] || category}
          </Badge>
        );
      },
    },
    {
      accessorKey: "expression",
      header: "Expression",
      size: 300,
      minSize: 200,
      maxSize: 400,
      cell: ({ row }) => {
        const expression = row.original.expression;
        return (
          <code className="text-xs bg-muted px-2 py-1 rounded block truncate max-w-[300px]">
            {expression}
          </code>
        );
      },
    },
    {
      id: "rateRange",
      header: "Rate Range",
      cell: ({ row }) => {
        const { minRate, maxRate } = row.original;
        if (!minRate && !maxRate) return "-";

        return (
          <div className="text-sm">
            {minRate && (
              <span className="text-muted-foreground">
                Min: ${parseFloat(minRate).toFixed(2)}
              </span>
            )}
            {minRate && maxRate && <span className="mx-1">•</span>}
            {maxRate && (
              <span className="text-muted-foreground">
                Max: ${parseFloat(maxRate).toFixed(2)}
              </span>
            )}
          </div>
        );
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created",
      meta: {
        apiField: "createdAt",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      cell: ({ row }) => {
        if (!row.original.createdAt) return "-";

        return (
          <HoverCardTimestamp
            className="font-table tracking-tight"
            timestamp={row.original.createdAt}
          />
        );
      },
    },
  ];
}
