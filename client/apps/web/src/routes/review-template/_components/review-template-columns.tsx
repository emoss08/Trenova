import { translate } from "@trenova/shared/i18n/runtime";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { statusChoices } from "@/lib/choices";
import type { ReviewTemplateRow } from "@/lib/graphql/performance-review";
import { StatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getColumns(): ColumnDef<ReviewTemplateRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
      size: 110,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "code",
      header: "Code",
      cell: ({ row }) => (
        <span className="flex items-center gap-2 font-medium">
          {row.original.code}
          {row.original.isDefault ? (
            <Badge variant="purple" className="px-1.5 py-0 text-[10px]">
              {translate("Default")}
            </Badge>
          ) : null}
        </span>
      ),
      meta: {
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <div className="flex flex-col">
          <span>{row.original.name}</span>
          {row.original.description ? (
            <span className="text-muted-foreground max-w-md truncate text-xs">
              {translate(row.original.description)}
            </span>
          ) : null}
        </div>
      ),
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "items",
      header: "Rates",
      cell: ({ row }) => (
        <span className="flex flex-wrap gap-1">
          {row.original.items.slice(0, 3).map((item) => (
            <Badge key={item.key} variant="outline" className="px-1.5 py-0 text-[10px]">
              {translate(item.label)} ×{item.weight}
            </Badge>
          ))}
          {row.original.items.length > 3 ? (
            <span className="text-muted-foreground text-xs">+{row.original.items.length - 3}</span>
          ) : null}
        </span>
      ),
      size: 260,
    },
    {
      accessorKey: "cadenceMonths",
      header: "Repeats",
      cell: ({ row }) =>
        row.original.cadenceMonths ? (
          `Every ${row.original.cadenceMonths} mo`
        ) : (
          <span className="text-muted-foreground">{translate("One-off")}</span>
        ),
      size: 110,
      meta: { apiField: "cadenceMonths", sortable: true },
    },
    {
      accessorKey: "openReviewCount",
      header: "Open reviews",
      cell: ({ row }) => <span className="tabular-nums">{row.original.openReviewCount}</span>,
      size: 110,
    },
    {
      accessorKey: "createdAt",
      header: "Created",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      meta: { apiField: "createdAt", sortable: true, filterable: true, filterType: "date" },
    },
  ];
}
