import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  DataTableColorColumn,
  DataTableDescription,
} from "@/components/data-table/_components/data-table-components";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { ShipmentTypeRow } from "@/lib/graphql/shipment-type-table";
import type { ShipmentType } from "@/types/shipment-type";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { useCallback } from "react";

function ShipmentTypeStatusCell({ row }: { row: ShipmentTypeRow }) {
  const queryClient = useQueryClient();

  const handleStatusChange = useCallback(
    async (newStatus: ShipmentType["status"]) => {
      if (!row.id) return;
      await apiService.shipmentTypeService.patch(row.id, {
        status: newStatus,
      });

      await queryClient.invalidateQueries({
        queryKey: ["shipment-type-list"],
      });
    },
    [row.id, queryClient],
  );

  return (
    <EditableStatusBadge
      status={row.status}
      options={statusChoices}
      onStatusChange={handleStatusChange}
    />
  );
}

export function getColumns(t: TranslateFn): ColumnDef<ShipmentTypeRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <ShipmentTypeStatusCell row={row.original} />,
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        label: t("Status"),
        filterType: "select",
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "code",
      header: t("Code"),
      cell: ({ row }) => {
        const { code, color } = row.original;
        return <DataTableColorColumn text={code} color={color ?? undefined} />;
      },
      enableCellEditing: true,
      meta: {
        apiField: "code",
        filterable: true,
        sortable: true,
        label: t("Code"),
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => (
        <DataTableDescription description={t(row.original.description)} truncateLength={100} />
      ),
      size: 400,
      minSize: 300,
      maxSize: 500,
      enableCellEditing: true,
      meta: {
        apiField: "description",
        filterable: true,
        sortable: true,
        label: t("Description"),
        filterType: "text",
        defaultFilterOperator: "contains",
      },
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
        label: t("Created At"),
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
    },
  ];
}
