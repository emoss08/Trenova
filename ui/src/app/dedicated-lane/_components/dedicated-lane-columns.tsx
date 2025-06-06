import {
  createCommonColumns,
  EntityRefCell,
} from "@/components/data-table/_components/data-table-column-helpers";
import { BooleanBadge, StatusBadge } from "@/components/status-badge";
import type { CustomerSchema } from "@/lib/schemas/customer-schema";
import type { DedicatedLaneSchema } from "@/lib/schemas/dedicated-lane-schema";
import type { LocationSchema } from "@/lib/schemas/location-schema";
import type { WorkerSchema } from "@/lib/schemas/worker-schema";
import { formatLocation, truncateText } from "@/lib/utils";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<DedicatedLaneSchema>[] {
  const commonColumns = createCommonColumns<DedicatedLaneSchema>();

  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const { status } = row.original;
        return <StatusBadge status={status} />;
      },
    },
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => {
        const { name } = row.original;
        return <p>{truncateText(name, 20)}</p>;
      },
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
          <EntityRefCell<CustomerSchema, DedicatedLaneSchema>
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
      accessorKey: "originLocation",
      header: "Origin Location",
      cell: ({ row }) => {
        const { originLocation } = row.original;

        if (!originLocation) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<LocationSchema, DedicatedLaneSchema>
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
          <EntityRefCell<LocationSchema, DedicatedLaneSchema>
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
      accessorKey: "primaryWorker",
      header: "Assigned Worker(s)",
      cell: ({ row }) => {
        const { primaryWorker } = row.original;

        if (!primaryWorker) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<WorkerSchema, DedicatedLaneSchema>
            entity={primaryWorker}
            config={{
              basePath: "/dispatch/configurations/workers",
              getId: (worker) => worker.id,
              getDisplayText: (worker) =>
                `${worker.firstName} ${worker.lastName}`,
              getHeaderText: "Assigned Workers",
              getSecondaryInfo: (_, dedicatedLane) =>
                dedicatedLane.secondaryWorker
                  ? {
                      label: "Co-Driver",
                      entity: dedicatedLane.secondaryWorker,
                      displayText: `${dedicatedLane.secondaryWorker.firstName} ${dedicatedLane.secondaryWorker.lastName}`,
                    }
                  : null,
            }}
            parent={row.original}
          />
        );
      },
    },
    {
      accessorKey: "autoAssign",
      header: "Auto Assign",
      cell: ({ row }) => {
        const { autoAssign } = row.original;
        return <BooleanBadge value={autoAssign} />;
      },
    },
    commonColumns.createdAt,
  ];
}
