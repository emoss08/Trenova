/* eslint-disable react-refresh/only-export-components */
import { EntityRefCell } from "@/components/data-table/_components/entity-ref-link";
import { EditableDateField } from "@/components/editable-date-field";
import { EditableEquipmentStatusBadge } from "@/components/editable-equipment-status-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { equipmentStatusChoices } from "@/lib/choices";
import { generateDateOnlyString, toDateFromUnixSeconds } from "@/lib/date";
import { apiService } from "@/services/api";
import type { EquipmentManufacturer } from "@/types/equipment-manufacturer";
import type { EquipmentType } from "@/types/equipment-type";
import type { FleetCode } from "@/types/fleet-code";
import type { Location } from "@/types/location";
import type { Trailer } from "@/types/trailer";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import { useCallback } from "react";
import { toast } from "sonner";

function StatusCell({ row }: { row: Trailer }) {
  const queryClient = useQueryClient();

  const handleStatusChange = useCallback(
    async (newStatus: Trailer["status"]) => {
      if (!row.id) return;
      await apiService.trailerService.patch(row.id, {
        status: newStatus,
      });

      await queryClient.invalidateQueries({
        queryKey: ["trailer-list"],
      });

      toast.success("Trailer status updated successfully");
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

function LastInspectionDateCell({ row }: { row: Trailer }) {
  const queryClient = useQueryClient();

  const handleDateChange = useCallback(
    async (newDate: number) => {
      if (!row.id) return;
      await apiService.trailerService.patch(row.id, {
        lastInspectionDate: newDate,
      });

      await queryClient.invalidateQueries({
        queryKey: ["trailer-list"],
      });

      toast.success(
        `Updated last inspection data to ${generateDateOnlyString(toDateFromUnixSeconds(newDate))}`,
      );
    },
    [row.id, queryClient],
  );

  return (
    <EditableDateField
      date={row.lastInspectionDate}
      onDateChange={handleDateChange}
    />
  );
}

export function getColumns(): ColumnDef<Trailer>[] {
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
      accessorKey: "equipmentType",
      header: "Equip. Type",
      cell: ({ row }) => {
        const { equipmentType } = row.original;

        if (!equipmentType) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<EquipmentType, Trailer>
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
          <EntityRefCell<EquipmentManufacturer, Trailer>
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
          <EntityRefCell<FleetCode, Trailer>
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
      accessorKey: "lastKnownLocationName",
      header: "Last Known Location",
      cell: ({ row }) => {
        const { lastKnownLocationId, lastKnownLocationName } = row.original;
        if (!lastKnownLocationId || !lastKnownLocationName) {
          return <p className="text-muted-foreground">-</p>;
        }

        const locationRef = {
          id: lastKnownLocationId,
          name: lastKnownLocationName,
        } as Pick<Location, "id" | "name">;

        return (
          <EntityRefCell<Pick<Location, "id" | "name">, Trailer>
            entity={locationRef}
            config={{
              basePath: "/dispatch/locations",
              getId: (location) => location.id,
              getDisplayText: (location) => location.name,
              getHeaderText: "Last Known Location",
            }}
            parent={row.original}
          />
        );
      },
      size: 220,
      minSize: 180,
      maxSize: 320,
      meta: {
        label: "Last Known Location",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "lastInspectionDate",
      header: "Last Inspection Date",
      cell: ({ row }) => {
        return <LastInspectionDateCell row={row.original} />;
      },
      size: 250,
      minSize: 250,
      maxSize: 300,
      meta: {
        apiField: "lastInspectionDate",
        label: "Last Inspection Date",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "eq",
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
