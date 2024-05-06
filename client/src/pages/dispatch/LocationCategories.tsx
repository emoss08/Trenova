import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { LocationCategoryDialog } from "@/components/location-category-table-dialog";
import { LocationCategoryEditDialog } from "@/components/location-category-table-edit-dialog";
import { truncateText } from "@/lib/utils";
import { type LocationCategory } from "@/types/location";
import { type ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<LocationCategory>[] = [
  {
    id: "select",
    header: ({ table }) => (
      <Checkbox
        checked={table.getIsAllPageRowsSelected()}
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all"
        className="translate-y-[2px]"
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label="Select row"
        className="translate-y-[2px]"
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
    cell: ({ row }) => {
      if (row.original.color) {
        return (
          <div className="flex items-center space-x-2 text-sm font-medium text-foreground">
            <div
              className={"mx-2 size-2 rounded-xl"}
              style={{ backgroundColor: row.original.color }}
            />
            {row.original.name}
          </div>
        );
      } else {
        return row.original.name;
      }
    },
  },
  {
    accessorKey: "description",
    header: "Description",
    cell: ({ row }) => truncateText(row.original.description as string, 50),
  },
];

export default function LocationCategories() {
  return (
    <DataTable
      queryKey="locationCategories"
      columns={columns}
      link="/location-categories/"
      name="Location Category"
      exportModelName="location_categories"
      filterColumn="name"
      TableSheet={LocationCategoryDialog}
      TableEditSheet={LocationCategoryEditDialog}
      addPermissionName="locationcategory.add"
    />
  );
}
