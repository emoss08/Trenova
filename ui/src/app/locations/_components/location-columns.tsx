import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import {
  DataTableColorColumn,
  DataTableDescription,
} from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { Checkbox } from "@/components/ui/checkbox";
import { type LocationSchema } from "@/lib/schemas/location-schema";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<LocationSchema>[] {
  return [
    {
      accessorKey: "select",
      id: "select",
      header: ({ table }) => {
        return (
          <Checkbox
            checked={
              table.getIsAllPageRowsSelected() ||
              (table.getIsSomePageRowsSelected() && "indeterminate")
            }
            onCheckedChange={(checked) =>
              table.toggleAllPageRowsSelected(!!checked)
            }
            aria-label="Select all"
          />
        );
      },
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(checked) => row.toggleSelected(!!checked)}
          aria-label="Select row"
        />
      ),
      enableSorting: false,
      enableHiding: false,
    },
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
