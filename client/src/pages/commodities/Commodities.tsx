import { CommodityDialog } from "@/components/commodity-dialog";
import { CommodityEditDialog } from "@/components/commodity-edit-table-dialog";
import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { Badge } from "@/components/ui/badge";
import { tableStatusChoices, yesAndNoChoices } from "@/lib/choices";
import { truncateText } from "@/lib/utils";
import { type Commodity } from "@/types/commodities";
import { type FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";

function HazmatBadge({ isHazmat }: { isHazmat: boolean }) {
  return (
    <Badge variant={isHazmat ? "active" : "inactive"}>
      {isHazmat ? "Yes" : "No"}
    </Badge>
  );
}

const columns: ColumnDef<Commodity>[] = [
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
    accessorKey: "status",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => <StatusBadge status={row.original.status} />,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
  },
  {
    accessorKey: "description",
    header: "Description",
    cell: ({ row }) => truncateText(row.original.description as string, 25),
  },
  {
    id: "temp_range",
    accessorFn: (row) => `${row.minTemp} - ${row.maxTemp}`,
    header: "Temperature Range",
    cell: ({ row, column }) => {
      return row.original?.minTemp && row.original?.maxTemp
        ? row.getValue(column.id)
        : "N/A";
    },
  },
  {
    accessorKey: "isHazmat",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Is Hazmat" />
    ),
    cell: ({ row }) => <HazmatBadge isHazmat={row.original.isHazmat} />,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
];

const filters: FilterConfig<Commodity>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
  {
    columnName: "isHazmat",
    title: "Is Hazmat",
    options: yesAndNoChoices,
  },
];

export default function CommodityPage() {
  return (
    <DataTable
      queryKey="commodities"
      columns={columns}
      link="/commodities/"
      name="Commodity"
      exportModelName="commodities"
      filterColumn="name"
      tableFacetedFilters={filters}
      TableSheet={CommodityDialog}
      TableEditSheet={CommodityEditDialog}
      addPermissionName="commodity.add"
    />
  );
}
