import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { BooleanBadge } from "@/components/status-badge";
import { freightClassChoices, statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { Commodity } from "@/types/commodity";
import { useQueryClient } from "@tanstack/react-query";
import { type ColumnDef } from "@tanstack/react-table";
import { useCallback } from "react";

function CommodityStatusCell({ row }: { row: Commodity }) {
  const queryClient = useQueryClient();

  const handleStatusChange = useCallback(
    async (newStatus: Commodity["status"]) => {
      if (!row.id) return;
      await apiService.commodityService.patch(row.id, {
        status: newStatus,
      });

      await queryClient.invalidateQueries({
        queryKey: ["commodity-list"],
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

export function getColumns(): ColumnDef<Commodity>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <CommodityStatusCell row={row.original} />,
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "freightClass",
      header: "Freight Class",
      cell: ({ row }) => {
        const classLabel = freightClassChoices.find(
          (c) => c.value === row.original.freightClass,
        )?.label;
        return <span>{classLabel || row.original.freightClass || "-"}</span>;
      },
      size: 160,
      minSize: 120,
      maxSize: 200,
      meta: {
        apiField: "freightClass",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: freightClassChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "hazardousMaterialId",
      header: "Hazmat",
      cell: ({ row }) => <BooleanBadge value={!!row.original.hazardousMaterialId} />,
      size: 100,
      minSize: 80,
      maxSize: 120,
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => (
        <DataTableDescription description={row.original.description} truncateLength={100} />
      ),
      size: 250,
      minSize: 200,
      maxSize: 400,
      meta: {
        apiField: "description",
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
