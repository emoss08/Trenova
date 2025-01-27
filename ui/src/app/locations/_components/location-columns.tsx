import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import {
  DataTableColorColumn,
  DataTableDescription,
} from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { type LocationSchema } from "@/lib/schemas/location-schema";
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
    {
      accessorKey: "code",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Code" />
      ),
    },
    {
      accessorKey: "name",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Name" />
      ),
    },
    {
      accessorKey: "locationCategory",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Location Category" />
      ),
      cell: ({ row }) => {
        const locationCategory = row.original.locationCategory;
        const isLocationCategory = !!locationCategory;

        return isLocationCategory ? (
          <DataTableColorColumn
            color={locationCategory?.color}
            text={locationCategory?.name ?? ""}
          />
        ) : (
          <p>No location category</p>
        );
      },
    },
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
        const state = row.original?.state;
        const addressLine =
          row.original.addressLine1 +
          (row.original.addressLine2 ? `, ${row.original.addressLine2}` : "");
        const cityStateZip = `${row.original.city} ${state?.abbreviation}, ${row.original.postalCode}`;

        return (
          <p>
            {addressLine} {cityStateZip}
          </p>
        );
      },
    },
  ];
}
