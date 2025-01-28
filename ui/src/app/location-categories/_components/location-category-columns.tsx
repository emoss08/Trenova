import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import {
  createCommonColumns,
  createEntityColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
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
    createEntityColumn(columnHelper, "name", {
      accessorKey: "name",
      getHeaderText: "Name",
      getId: (locationCategory) => locationCategory.id,
      getDisplayText: (locationCategory) => locationCategory.name,
      getColor: (locationCategory) => locationCategory.color,
    }),
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
