import { EntityRefCell } from "@/components/data-table/_components/data-table-column-helpers";
import { HoverCardTimestamp } from "@/components/data-table/_components/data-table-components";
import { EquipmentStatusBadge } from "@/components/status-badge";
import { equipmentStatusChoices } from "@/lib/choices";
import { EquipmentManufacturerSchema } from "@/lib/schemas/equipment-manufacturer-schema";
import { EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import { FleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import type { TractorSchema } from "@/lib/schemas/tractor-schema";
import { WorkerSchema } from "@/lib/schemas/worker-schema";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<TractorSchema>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.status;
        return <EquipmentStatusBadge status={status} />;
      },
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: equipmentStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "code",
      accessorKey: "code",
      header: "Code",
      cell: ({ row }) => {
        const { code } = row.original;
        return <p>{code}</p>;
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "equipmentType",
      accessorKey: "equipmentType",
      header: "Equipment Type",
      cell: ({ row }) => {
        const { equipmentType } = row.original;
        if (!equipmentType) {
          return <p className="text-muted-foreground">-</p>;
        }
        return (
          <EntityRefCell<EquipmentTypeSchema, TractorSchema>
            entity={equipmentType}
            config={{
              basePath: "/dispatch/configurations/equipment-types",
              getId: (equipmentType) => equipmentType.id ?? undefined,
              getDisplayText: (equipmentType) => equipmentType.code,
              color: {
                getColor: (equipmentType) => equipmentType.color,
              },
              getHeaderText: "Equipment Type",
            }}
            parent={row.original}
          />
        );
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "equipmentType.code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "equipmentManufacturer",
      accessorKey: "equipmentManufacturer",
      header: "Equipment Manufacturer",
      cell: ({ row }) => {
        const { equipmentManufacturer } = row.original;
        if (!equipmentManufacturer) {
          return <p className="text-muted-foreground">-</p>;
        }
        return (
          <EntityRefCell<EquipmentManufacturerSchema, TractorSchema>
            entity={equipmentManufacturer}
            config={{
              basePath: "/dispatch/configurations/equipment-manufacturers",
              getId: (equipmentManufacturer) =>
                equipmentManufacturer.id ?? undefined,
              getDisplayText: (equipmentManufacturer) =>
                equipmentManufacturer.name,
              getHeaderText: "Equipment Manufacturer",
            }}
            parent={row.original}
          />
        );
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "equipmentManufacturer.name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "primaryWorker",
      accessorKey: "primaryWorker",
      header: "Primary Worker",
      cell: ({ row }) => {
        const { primaryWorker } = row.original;
        if (!primaryWorker) {
          return <p className="text-muted-foreground">-</p>;
        }
        return (
          <EntityRefCell<WorkerSchema, TractorSchema>
            entity={primaryWorker}
            config={{
              basePath: "/dispatch/configurations/workers",
              getId: (primaryWorker) => primaryWorker.id ?? undefined,
              getDisplayText: (primaryWorker) =>
                `${primaryWorker.firstName} ${primaryWorker.lastName}`,
              getHeaderText: "Primary Worker",
            }}
            parent={row.original}
          />
        );
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "primaryWorker.name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "fleetCode",
      accessorKey: "fleetCode",
      header: "Fleet Code",
      cell: ({ row }) => {
        const { fleetCode } = row.original;
        if (!fleetCode) {
          return <p className="text-muted-foreground">-</p>;
        }
        return (
          <EntityRefCell<FleetCodeSchema, TractorSchema>
            entity={fleetCode}
            config={{
              basePath: "/dispatch/configurations/fleet-codes",
              getId: (fleetCode) => fleetCode.id ?? undefined,
              getDisplayText: (fleetCode) => fleetCode.code,
              getHeaderText: "Fleet Code",
            }}
            parent={row.original}
          />
        );
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "fleetCode.name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
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
