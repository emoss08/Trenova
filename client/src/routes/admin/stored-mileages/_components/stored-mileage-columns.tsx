import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@/components/ui/badge";
import type { StoredMileage } from "@/types/stored-mileage";
import type { ColumnDef } from "@tanstack/react-table";

function stopLabel(key: StoredMileage["originKey"]) {
  if (key.postalCode) {
    return key.postalCode;
  }
  if (key.city || key.state) {
    return [key.city, key.state].filter(Boolean).join(", ");
  }
  return key.key;
}

export function getColumns(): ColumnDef<StoredMileage>[] {
  return [
    {
      accessorKey: "routeSignature",
      header: "Lane",
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="truncate font-medium">
            {stopLabel(row.original.originKey)} {"->"} {stopLabel(row.original.destinationKey)}
          </div>
          <div className="truncate text-xs text-muted-foreground">
            {row.original.intermediateKeys.length > 0
              ? `${row.original.intermediateKeys.length} intermediate stops`
              : row.original.routeHash}
          </div>
        </div>
      ),
      meta: { label: "Route", apiField: "routeSignature", filterable: true, sortable: true },
      size: 320,
    },
    {
      accessorKey: "distance",
      header: "Distance",
      cell: ({ row }) => `${row.original.distance.toFixed(2)} ${row.original.distanceUnits}`,
      meta: { label: "Distance", apiField: "distance", filterable: true, sortable: true },
      size: 140,
    },
    {
      accessorKey: "routingType",
      header: "Routing",
      cell: ({ row }) => row.original.routingType,
      meta: { label: "Routing", apiField: "routingType", filterable: true, sortable: true },
      size: 120,
    },
    {
      accessorKey: "method",
      header: "Method",
      cell: ({ row }) => row.original.method,
      meta: { label: "Method", apiField: "method", filterable: true, sortable: true },
      size: 140,
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <Badge variant={row.original.status === "Active" ? "active" : "inactive"}>
          {row.original.status}
        </Badge>
      ),
      meta: { label: "Status", apiField: "status", filterable: true, sortable: true },
      size: 120,
    },
    {
      accessorKey: "hitCount",
      header: "Hits",
      cell: ({ row }) => row.original.hitCount,
      meta: { label: "Hits", apiField: "hitCount", filterable: true, sortable: true },
      size: 90,
    },
    {
      accessorKey: "lastCalculatedAt",
      header: "Calculated",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.lastCalculatedAt} />,
      meta: { label: "Calculated", apiField: "lastCalculatedAt", sortable: true },
      size: 180,
    },
  ];
}
