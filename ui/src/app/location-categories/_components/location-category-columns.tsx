import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { type LocationCategorySchema } from "@/lib/schemas/location-category-schema";
import {
  mapToFacilityType,
  mapToLocationCategoryType,
} from "@/types/location-category";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<LocationCategorySchema>[] {
  const columnHelper = createColumnHelper<LocationCategorySchema>();
  const commonColumns = createCommonColumns(columnHelper);

  return [
    commonColumns.selection,
    {
      accessorKey: "name",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Name" />
      ),
      cell: ({ row }) => {
        const isColor = !!row.original.color;
        return isColor ? (
          <div className="flex items-center gap-x-1.5 text-sm font-medium text-foreground">
            <div
              className="size-2 rounded-full"
              style={{
                backgroundColor: row.original.color,
              }}
            />
            <p>{row.original.name}</p>
          </div>
        ) : (
          <p>{row.original.name}</p>
        );
      },
    },
    {
      accessorKey: "type",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Type" />
      ),
      cell: ({ row }) => <p>{mapToLocationCategoryType(row.original.type)}</p>,
    },
    {
      accessorKey: "facilityType",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Facility Type" />
      ),
      cell: ({ row }) => (
        <p>
          {row.original.facilityType
            ? mapToFacilityType(row.original.facilityType)
            : ""}
        </p>
      ),
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
  ];
}
