import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  DataTableColorColumn,
  DataTableDescription,
} from "@/components/data-table/_components/data-table-components";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { ServiceTypeRow } from "@/lib/graphql/service-type-table";
import type { ServiceType } from "@/types/service-type";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { useCallback } from "react";

function ServiceTypeStatusCell({ row }: { row: ServiceTypeRow }) {
  const queryClient = useQueryClient();

  const handleStatusChange = useCallback(
    async (newStatus: ServiceType["status"]) => {
      if (!row.id) return;
      await apiService.serviceTypeService.patch(row.id, {
        status: newStatus,
      });

      await queryClient.invalidateQueries({
        queryKey: ["service-type-list"],
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

export function getColumns(t: TranslateFn): ColumnDef<ServiceTypeRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <ServiceTypeStatusCell row={row.original} />,
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        apiField: "status",
        label: t("Status"),
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "code",
      header: t("Code"),
      enableCellEditing: true,
      cell: ({ row }) => {
        const { code, color } = row.original;
        return <DataTableColorColumn text={code} color={color ?? undefined} />;
      },
      meta: {
        apiField: "code",
        label: t("Code"),
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "description",
      header: t("Description"),
      enableCellEditing: true,
      cell: ({ row }) => (
        <DataTableDescription description={t(row.original.description)} truncateLength={100} />
      ),
      size: 400,
      minSize: 300,
      maxSize: 500,
      meta: {
        apiField: "description",
        label: t("Description"),
        filterable: true,
        sortable: true,
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
        label: t("Created At"),
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
