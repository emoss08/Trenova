import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { DistanceProfileRow } from "@/lib/graphql/distance-profile-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getColumns(t: TranslateFn): ColumnDef<DistanceProfileRow>[] {
  return [
    {
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => (
        <div className="flex min-w-0 items-center gap-2">
          <span className="truncate font-medium">{row.original.name}</span>
          {row.original.isDefault && <Badge variant="info">{t("Default")}</Badge>}
        </div>
      ),
      meta: {
        label: t("Name"),
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
      size: 240,
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <Badge variant={row.original.status === "Active" ? "active" : "inactive"}>
          {row.original.status}
        </Badge>
      ),
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
      },
      size: 120,
    },
    {
      accessorKey: "provider",
      header: t("Provider"),
      cell: ({ row }) => row.original.provider,
      meta: { label: t("Provider"), apiField: "provider", filterable: true, sortable: true },
      size: 120,
    },
    {
      accessorKey: "routingType",
      header: t("Routing"),
      cell: ({ row }) => row.original.routingType,
      meta: { label: t("Routing Type"), apiField: "routingType", filterable: true, sortable: true },
      size: 140,
    },
    {
      accessorKey: "distanceUnits",
      header: t("Units"),
      cell: ({ row }) => row.original.distanceUnits,
      meta: { label: t("Units"), apiField: "distanceUnits", filterable: true, sortable: true },
      size: 100,
    },
    {
      accessorKey: "dataVersion",
      header: t("Data Version"),
      cell: ({ row }) => row.original.dataVersion,
      meta: { label: t("Data Version"), apiField: "dataVersion", filterable: true, sortable: true },
      size: 140,
    },
    {
      accessorKey: "updatedAt",
      header: t("Updated"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.updatedAt} />,
      meta: {
        apiField: "updatedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 180,
    },
  ];
}
