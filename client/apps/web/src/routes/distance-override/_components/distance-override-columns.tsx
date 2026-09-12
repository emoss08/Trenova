import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { EntityRefCell } from "@/components/data-table/_components/entity-ref-link";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { formatLocation } from "@trenova/shared/lib/utils";
import type { DistanceOverrideRow } from "@/lib/graphql/distance-override-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getColumns(t: TranslateFn): ColumnDef<DistanceOverrideRow>[] {
  return [
    {
      accessorKey: "originLocationId",
      header: t("Origin Location"),
      cell: ({ row }) => {
        const { originLocation } = row.original;
        if (!originLocation) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell
            entity={originLocation}
            config={{
              basePath: "/dispatch/configuration-files/locations",
              getId: (location) => location.id,
              getDisplayText: (location) => location.name,
              getSecondaryInfo: (location) => {
                return {
                  entity: location,
                  displayText: formatLocation(location),
                  clickable: false,
                };
              },
              getHeaderText: "Origin Location",
            }}
            parent={row.original}
          />
        );
      },
      meta: {
        label: t("Origin Location"),
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "destinationLocationId",
      header: t("Destination Location"),
      cell: ({ row }) => {
        const { destinationLocation } = row.original;
        if (!destinationLocation) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell
            entity={destinationLocation}
            config={{
              basePath: "/dispatch/configuration-files/locations",
              getId: (location) => location.id,
              getDisplayText: (location) => location.name,
              getSecondaryInfo: (location) => {
                return {
                  entity: location,
                  displayText: formatLocation(location),
                  clickable: false,
                };
              },
              getHeaderText: "Destination Location",
            }}
            parent={row.original}
          />
        );
      },
      meta: {
        label: t("Destination Location"),
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "distance",
      header: t("Distance"),
      cell: ({ row }) => row.original.distance,
      meta: {
        label: t("Distance"),
        apiField: "distance",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
      size: 120,
      minSize: 80,
      maxSize: 180,
    },
    {
      accessorKey: "customerId",
      header: t("Customer"),
      cell: ({ row }) => {
        const { customer } = row.original;

        if (!customer) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell
            entity={customer}
            config={{
              basePath: "/billing/configuration-files/customers",
              getId: (customer) => customer.id,
              getDisplayText: (customer) => customer.name,
              getHeaderText: "Customer",
            }}
            parent={row.original}
          />
        );
      },
      size: 200,
      minSize: 100,
      maxSize: 250,
      meta: {
        label: t("Customer"),
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "intermediateStops",
      header: t("Stops"),
      cell: ({ row }) => row.original.intermediateStops?.length ?? 0,
      meta: {
        label: t("Stops"),
        filterable: false,
        sortable: false,
      },
      size: 80,
      minSize: 70,
      maxSize: 100,
    },
    {
      accessorKey: "createdAt",
      header: t("Created At"),
      cell: ({ row }) => {
        return <HoverCardTimestamp timestamp={row.original.createdAt} />;
      },
      meta: {
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
    },
  ];
}
