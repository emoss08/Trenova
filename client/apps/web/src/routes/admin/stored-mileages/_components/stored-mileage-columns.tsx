import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { StoredMileageRow } from "@/lib/graphql/stored-mileage-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";

function stopLabel(key: StoredMileageRow["originKey"]) {
  if (key.postalCode) {
    return key.postalCode;
  }
  if (key.city || key.state) {
    return [key.city, key.state].filter(Boolean).join(", ");
  }
  return key.key;
}

export function getColumns(t: TranslateFn): ColumnDef<StoredMileageRow>[] {
  return [
    {
      accessorKey: "routeSignature",
      header: t("Lane"),
      cell: ({ row }) => {
        const intermediateStopCount = row.original.intermediateKeys?.length ?? 0;
        return (
          <div className="min-w-0">
            <div className="truncate font-medium">
              {stopLabel(row.original.originKey)} {"->"} {stopLabel(row.original.destinationKey)}
            </div>
            <div className="text-muted-foreground truncate text-xs">
              {intermediateStopCount > 0
                ? t("{0} intermediate stops", intermediateStopCount)
                : row.original.routeHash}
            </div>
          </div>
        );
      },
      meta: { label: t("Route"), apiField: "routeSignature", filterable: true, sortable: true },
      size: 320,
    },
    {
      accessorKey: "distance",
      header: t("Distance"),
      cell: ({ row }) => `${row.original.distance.toFixed(2)} ${row.original.distanceUnits}`,
      meta: { label: t("Distance"), apiField: "distance", filterable: true, sortable: true },
      size: 140,
    },
    {
      accessorKey: "routingType",
      header: t("Routing"),
      cell: ({ row }) => row.original.routingType,
      meta: { label: t("Routing"), apiField: "routingType", filterable: true, sortable: true },
      size: 120,
    },
    {
      accessorKey: "method",
      header: t("Method"),
      cell: ({ row }) => row.original.method,
      meta: { label: t("Method"), apiField: "method", filterable: true, sortable: true },
      size: 140,
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <Badge variant={row.original.status === "Active" ? "active" : "inactive"}>
          {row.original.status}
        </Badge>
      ),
      meta: { label: t("Status"), apiField: "status", filterable: true, sortable: true },
      size: 120,
    },
    {
      accessorKey: "hitCount",
      header: t("Hits"),
      cell: ({ row }) => row.original.hitCount,
      meta: { label: t("Hits"), apiField: "hitCount", filterable: true, sortable: true },
      size: 90,
    },
    {
      accessorKey: "lastCalculatedAt",
      header: t("Calculated"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.lastCalculatedAt} />,
      meta: { label: t("Calculated"), apiField: "lastCalculatedAt", sortable: true },
      size: 180,
    },
  ];
}
