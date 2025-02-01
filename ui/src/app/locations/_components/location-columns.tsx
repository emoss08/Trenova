import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import {
  createCommonColumns,
  createEntityColumn,
  createEntityRefColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { type LocationSchema } from "@/lib/schemas/location-schema";
import { formatLocation } from "@/lib/utils";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<LocationSchema>[] {
  const columnHelper = createColumnHelper<LocationSchema>();
  const commonColumns = createCommonColumns(columnHelper);

  return [
    commonColumns.selection,
    {
      accessorKey: "status",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Status" />
      ),
      cell: ({ row }) => {
        const status = row.original.status;
        return <StatusBadge status={status} />;
      },
    },
    createEntityColumn(columnHelper, "name", {
      accessorKey: "name",
      getHeaderText: "Name",
      getId: (location) => location.id,
      getDisplayText: (location) => location.name,
    }),
    createEntityRefColumn<LocationSchema, "locationCategory">(
      columnHelper,
      "locationCategory",
      {
        basePath: "/dispatch/configurations/location-categories",
        getHeaderText: "Location Category",
        getId: (locationCategory) => locationCategory.id ?? undefined,
        getDisplayText: (locationCategory) => locationCategory.name,
        color: {
          getColor: (locationCategory) => locationCategory.color,
        },
      },
    ),
    {
      accessorKey: "description",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Description" />
      ),
      cell: ({ row }) => (
        <DataTableDescription description={row.original.description} />
      ),
    },
    {
      id: "addressLine",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Address Line" />
      ),
      cell: ({ row }) => {
        return <p>{formatLocation(row.original)}</p>;
      },
    },

    commonColumns.createdAt,
  ];
}
