import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { hazardousClassChoices, packingGroupChoices, statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { HazardousMaterial } from "@/types/hazardous-material";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { useCallback } from "react";

function HazardousMaterialStatusCell({ row }: { row: HazardousMaterial }) {
  const queryClient = useQueryClient();

  const handleStatusChange = useCallback(
    async (newStatus: HazardousMaterial["status"]) => {
      if (!row.id) return;
      await apiService.hazardousMaterialService.patch(row.id, {
        status: newStatus,
      });

      await queryClient.invalidateQueries({
        queryKey: ["hazardous-material-list"],
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

export function getColumns(t: TranslateFn): ColumnDef<HazardousMaterial>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <HazardousMaterialStatusCell row={row.original} />,
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
      accessorKey: "code",
      header: t("Code"),
      meta: {
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "name",
      header: t("Name"),
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
      accessorKey: "class",
      header: t("Class"),
      cell: ({ row }) => {
        const classLabel = hazardousClassChoices.find((c) => c.value === row.original.class)?.label;
        return <span>{classLabel || row.original.class}</span>;
      },
      size: 250,
      minSize: 200,
      maxSize: 350,
      meta: {
        apiField: "class",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: hazardousClassChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "packingGroup",
      header: t("Packing Group"),
      cell: ({ row }) => {
        const pgLabel = packingGroupChoices.find(
          (c) => c.value === row.original.packingGroup,
        )?.label;
        return <span>{pgLabel || row.original.packingGroup || "-"}</span>;
      },
      size: 160,
      minSize: 120,
      maxSize: 200,
      meta: {
        apiField: "packingGroup",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: packingGroupChoices,
        defaultFilterOperator: "eq",
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
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
    },
  ];
}
