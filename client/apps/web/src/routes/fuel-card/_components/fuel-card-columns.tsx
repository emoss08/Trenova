import { EntityRefCell } from "@/components/data-table/_components/entity-ref-link";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { fuelCardProviderChoices, fuelCardStatusChoices } from "@/lib/choices";
import type { TractorRow } from "@/lib/graphql/equipment-table";
import type { FuelCardRow } from "@/lib/graphql/fuel-card";
import type { WorkerRow } from "@/lib/graphql/worker-table";
import { FuelCardStatusBadge } from "@trenova/shared/components/status-badge";
import { formatUnixDateOrDash } from "@trenova/shared/lib/date";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { FUEL_CARD_PROVIDER_LABELS } from "@trenova/shared/types/fuel-ifta-enums";

export function maskedCardNumber(lastFour: string): string {
  return `•••• ${lastFour}`;
}

export function getColumns(): ColumnDef<FuelCardRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <FuelCardStatusBadge status={row.original.status} />,
      size: 120,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: fuelCardStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "label",
      header: "Label",
      cell: ({ row }) => <span className="font-medium">{row.original.label}</span>,
      meta: {
        apiField: "label",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "provider",
      header: "Provider",
      cell: ({ row }) => FUEL_CARD_PROVIDER_LABELS[row.original.provider],
      size: 110,
      meta: {
        apiField: "provider",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: fuelCardProviderChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "lastFour",
      header: "Card",
      cell: ({ row }) => (
        <span className="font-table tabular-nums">{maskedCardNumber(row.original.lastFour)}</span>
      ),
      size: 110,
      meta: {
        apiField: "lastFour",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "eq",
        exportValue: (row: FuelCardRow) => maskedCardNumber(row.lastFour),
      },
    },
    {
      id: "assignedWorker",
      header: "Worker",
      cell: ({ row }) => {
        const { assignedWorker } = row.original;

        if (!assignedWorker) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<NonNullable<FuelCardRow["assignedWorker"]>, WorkerRow>
            entity={assignedWorker}
            parent={assignedWorker as WorkerRow}
            config={{
              basePath: "/workers",
              getId: (w) => w.id,
              getDisplayText: (w) => `${w.firstName} ${w.lastName}`,
            }}
          />
        );
      },
      enableSorting: false,
      meta: {
        apiField: "assignedWorkerId",
        filterable: false,
        sortable: false,
        exportValue: (row: FuelCardRow) => row.assignedWorker?.wholeName ?? "",
      },
    },
    {
      id: "assignedTractor",
      header: "Tractor",
      cell: ({ row }) => {
        const { assignedTractor } = row.original;

        if (!assignedTractor) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<NonNullable<FuelCardRow["assignedTractor"]>, TractorRow>
            entity={assignedTractor}
            parent={assignedTractor as TractorRow}
            config={{
              basePath: "/equipment/tractors",
              getId: (t) => t.id,
              getDisplayText: (t) => t.code,
            }}
          />
        );
      },
      enableSorting: false,
      size: 110,
      meta: {
        apiField: "assignedTractorId",
        filterable: false,
        sortable: false,
        exportValue: (row: FuelCardRow) => row.assignedTractor?.code ?? "",
      },
    },
    {
      accessorKey: "expiresAt",
      header: "Expires",
      cell: ({ row }) => (
        <span className="font-table tabular-nums">
          {formatUnixDateOrDash(row.original.expiresAt)}
        </span>
      ),
      size: 120,
      meta: {
        apiField: "expiresAt",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      accessorKey: "updatedAt",
      header: "Updated",
      cell: ({ row }) => (
        <HoverCardTimestamp
          className="font-table tracking-tight"
          timestamp={row.original.updatedAt}
        />
      ),
      meta: {
        apiField: "updatedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
