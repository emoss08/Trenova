/* eslint-disable react-refresh/only-export-components */
import { EntityRefCell } from "@/components/data-table/_components/entity-ref-link";
import { EditableEquipmentStatusBadge } from "@/components/editable-equipment-status-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { equipmentStatusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { EquipmentManufacturer } from "@/types/equipment-manufacturer";
import type { EquipmentType } from "@/types/equipment-type";
import type { FleetCode } from "@/types/fleet-code";
import type { Tractor } from "@/types/tractor";
import type { Worker } from "@/types/worker";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import { useCallback } from "react";
import { toast } from "sonner";

function StatusCell({ row }: { row: Tractor }) {
  const queryClient = useQueryClient();

  const handleStatusChange = useCallback(
    async (newStatus: Tractor["status"]) => {
      if (!row.id) return;
      await apiService.tractorService.patch(row.id, {
        status: newStatus,
      });

      await queryClient.invalidateQueries({
        queryKey: ["tractor-list"],
      });

      toast.success("Tractor status updated successfully");
    },
    [row.id, queryClient],
  );

  return (
    <EditableEquipmentStatusBadge
      status={row.status}
      options={equipmentStatusChoices}
      onStatusChange={handleStatusChange}
    />
  );
}

export function getColumns(): ColumnDef<Tractor>[] {
  return [
    {
      accessorKey: "code",
      header: "Code",
      cell: ({ row }) => <p>{row.original.code}</p>,
      meta: {
        label: "Code",
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <StatusCell row={row.original} />,
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        label: "Status",
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: equipmentStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "primaryWorker",
      header: "Primary Worker",
      cell: ({ row }) => {
        const { primaryWorker } = row.original;

        if (!primaryWorker) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<Worker, Tractor>
            entity={primaryWorker}
            config={{
              basePath: "/workers",
              getId: (worker) => worker.id,
              getDisplayText: (worker) =>
                `${worker.firstName} ${worker.lastName}`,
              getHeaderText: "Primary Worker",
            }}
            parent={row.original}
          />
        );
      },
      meta: {
        apiField: "primaryWorker.wholeName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
        label: "Primary Worker",
      },
    },
    {
      accessorKey: "equipmentType",
      header: "Equip. Type",
      cell: ({ row }) => {
        const { equipmentType } = row.original;

        if (!equipmentType) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<EquipmentType, Tractor>
            entity={equipmentType}
            config={{
              basePath: "/equipment/configuration-files/equipment-types",
              getId: (equipmentType) => equipmentType.id,
              getDisplayText: (equipmentType) => equipmentType.code,
              getHeaderText: "Equip. Type",
              color: {
                getColor: (equipmentType) => equipmentType.color,
              },
            }}
            parent={row.original}
          />
        );
      },
      meta: {
        apiField: "equipmentType.code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
        label: "Equip. Type",
      },
    },
    {
      accessorKey: "equipmentManufacturer",
      header: "Equip. Manufacturer",
      cell: ({ row }) => {
        const { equipmentManufacturer } = row.original;
        if (!equipmentManufacturer) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<EquipmentManufacturer, Tractor>
            entity={equipmentManufacturer}
            config={{
              basePath:
                "/equipment/configuration-files/equipment-manufacturers",
              getId: (equipmentManufacturer) => equipmentManufacturer.id,
              getDisplayText: (equipmentManufacturer) =>
                equipmentManufacturer.name,
              getHeaderText: "Equip. Manufacturer",
            }}
            parent={row.original}
          />
        );
      },
    },
    {
      accessorKey: "fleetCode",
      header: "Fleet Code",
      cell: ({ row }) => {
        const { fleetCode } = row.original;
        if (!fleetCode) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<FleetCode, Tractor>
            entity={fleetCode}
            config={{
              basePath: "/dispatch/configuration-files/fleet-codes",
              getId: (fleetCode) => fleetCode.id,
              getDisplayText: (fleetCode) => fleetCode.code,
              color: {
                getColor: (fleetCode) => fleetCode.color,
              },
              getHeaderText: "Fleet Code",
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
        return (
          <HoverCardTimestamp
            className="shrink-0"
            timestamp={row.original.createdAt}
          />
        );
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
