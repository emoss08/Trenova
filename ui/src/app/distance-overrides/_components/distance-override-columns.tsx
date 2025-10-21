import { EntityRefCell } from "@/components/data-table/_components/data-table-column-helpers";
import { HoverCardTimestamp } from "@/components/data-table/_components/data-table-components";
import type { CustomerSchema } from "@/lib/schemas/customer-schema";
import { DistanceOverrideSchema } from "@/lib/schemas/distance-override-schema";
import type { LocationSchema } from "@/lib/schemas/location-schema";
import { formatLocation } from "@/lib/utils";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<DistanceOverrideSchema>[] {
  return [
    {
      accessorKey: "originLocation",
      header: "Origin Location",
      cell: ({ row }) => {
        const { originLocation } = row.original;

        if (!originLocation) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<LocationSchema, DistanceOverrideSchema>
            entity={originLocation}
            config={{
              basePath: "/dispatch/configurations/locations",
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
    },
    {
      accessorKey: "destinationLocation",
      header: "Destination Location",
      cell: ({ row }) => {
        const { destinationLocation } = row.original;

        if (!destinationLocation) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<LocationSchema, DistanceOverrideSchema>
            entity={destinationLocation}
            config={{
              basePath: "/dispatch/configurations/locations",
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
    },
    {
      accessorKey: "distance",
      header: "Distance",
      cell: ({ row }) => {
        const { distance } = row.original;
        return <p>{distance?.toFixed(2)}</p>;
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
    },
    {
      accessorKey: "customer",
      header: "Customer",
      cell: ({ row }) => {
        const { customer } = row.original;

        if (!customer) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<CustomerSchema, DistanceOverrideSchema>
            entity={customer}
            config={{
              basePath: "/billing/configurations/customers",
              getId: (customer) => customer.id,
              getDisplayText: (customer) => customer.name,
              getHeaderText: "Customer",
            }}
            parent={row.original}
          />
        );
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => {
        return <HoverCardTimestamp timestamp={row.original.createdAt} />;
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "createdAt",
        label: "Created At",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
