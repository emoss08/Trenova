import { Badge } from "@/components/ui/badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { DistanceProfile } from "@/types/distance-profile";
import type { ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<DistanceProfile>[] {
  return [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <div className="flex min-w-0 items-center gap-2">
          <span className="truncate font-medium">{row.original.name}</span>
          {row.original.isDefault && <Badge variant="info">Default</Badge>}
        </div>
      ),
      meta: {
        label: "Name",
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
      size: 240,
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <Badge variant={row.original.status === "Active" ? "active" : "inactive"}>
          {row.original.status}
        </Badge>
      ),
      meta: {
        label: "Status",
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
      },
      size: 120,
    },
    {
      accessorKey: "provider",
      header: "Provider",
      cell: ({ row }) => row.original.provider,
      meta: { label: "Provider", apiField: "provider", filterable: true, sortable: true },
      size: 120,
    },
    {
      accessorKey: "routingType",
      header: "Routing",
      cell: ({ row }) => row.original.routingType,
      meta: { label: "Routing Type", apiField: "routingType", filterable: true, sortable: true },
      size: 140,
    },
    {
      accessorKey: "distanceUnits",
      header: "Units",
      cell: ({ row }) => row.original.distanceUnits,
      meta: { label: "Units", apiField: "distanceUnits", filterable: true, sortable: true },
      size: 100,
    },
    {
      accessorKey: "dataVersion",
      header: "Data Version",
      cell: ({ row }) => row.original.dataVersion,
      meta: { label: "Data Version", apiField: "dataVersion", filterable: true, sortable: true },
      size: 140,
    },
    {
      accessorKey: "updatedAt",
      header: "Updated",
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
