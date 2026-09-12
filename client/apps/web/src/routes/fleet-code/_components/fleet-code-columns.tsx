import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  DataTableColorColumn,
  DataTableDescription,
} from "@/components/data-table/_components/data-table-components";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { FleetCodeRow } from "@/lib/graphql/fleet-code-table";
import type { FleetCode } from "@trenova/shared/types/fleet-code";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { useCallback } from "react";

// eslint-disable-next-line react-refresh/only-export-components
function StatusCell({ row }: { row: FleetCodeRow }) {
  const queryClient = useQueryClient();

  const handleStatusChange = useCallback(
    async (newStatus: FleetCode["status"]) => {
      if (!row.id) return;
      await apiService.fleetCodeService.patch(row.id, {
        status: newStatus,
      });

      await queryClient.invalidateQueries({
        queryKey: ["fleet-code-list"],
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
export function getColumns(t: TranslateFn): ColumnDef<FleetCodeRow>[] {
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
      cell: ({ row }) => <StatusCell row={row.original} />,
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
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => (
        <DataTableDescription description={t(row.original.description)} truncateLength={100} />
      ),
      size: 100,
      minSize: 100,
      maxSize: 500,
      meta: {
        apiField: "description",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "manager",
      header: t("Manager"),
      cell: ({ row }) => {
        const { manager } = row.original;
        if (!manager) return <p className="text-muted-foreground">-</p>;
        return <p>{manager.name}</p>;
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "manager.name",
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
        return <HoverCardTimestamp className="shrink-0" timestamp={row.original.createdAt} />;
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "createdAt",
        label: t("Created At"),
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
