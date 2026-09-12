import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  DataTableColorColumn,
  DataTableDescription,
} from "@/components/data-table/_components/data-table-components";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { equipmentClassChoices, statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { EquipmentType } from "@/types/equipment-type";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { useCallback } from "react";

// eslint-disable-next-line react-refresh/only-export-components
function EquipmentTypeStatusCell({ row }: { row: EquipmentType }) {
  const queryClient = useQueryClient();

  const handleStatusChange = useCallback(
    async (newStatus: EquipmentType["status"]) => {
      if (!row.id) return;
      await apiService.equipmentTypeService.patch(row.id, {
        status: newStatus,
      });

      await queryClient.invalidateQueries({
        queryKey: ["equipment-type-list"],
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

export function getColumns(t: TranslateFn): ColumnDef<EquipmentType>[] {
  return [
    {
      accessorKey: "code",
      header: t("Code"),
      cell: ({ row }) => {
        const { color, code } = row.original;
        return <DataTableColorColumn text={code} color={color} />;
      },
      meta: {
        label: t("Code"),
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <EquipmentTypeStatusCell row={row.original} />,
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "class",
      header: t("Equip. Class"),
      meta: {
        label: t("Equip. Class"),
        apiField: "class",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: equipmentClassChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => (
        <DataTableDescription description={t(row.original.description)} truncateLength={50} />
      ),
      meta: {
        label: t("Description"),
        apiField: "description",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      size: 250,
      minSize: 150,
      maxSize: 400,
    },
    {
      accessorKey: "createdAt",
      header: t("Created At"),
      cell: ({ row }) => {
        return <HoverCardTimestamp timestamp={row.original.createdAt} />;
      },
      meta: {
        label: t("Created At"),
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
