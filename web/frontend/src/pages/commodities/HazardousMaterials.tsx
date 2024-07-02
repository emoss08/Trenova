import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { HazardousMaterialDialog } from "@/components/hazardous-material-dialog";
import { HazardousMaterialEditDialog } from "@/components/hazardous-material-edit-dialog";
import { tableStatusChoices } from "@/lib/choices";
import { truncateText } from "@/lib/utils";
import { type HazardousMaterial } from "@/types/commodities";
import { type FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<HazardousMaterial>[] = [
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
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "description",
    header: "Description",
    cell: ({ row }) => truncateText(row.original.description as string, 25),
  },
  {
    accessorKey: "packingGroup",
    header: "Packing Group",
  },
];

const filters: FilterConfig<HazardousMaterial>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
];

export default function HazardousMaterials() {
  return (
    <DataTable
      queryKey="hazardousMaterials"
      columns={columns}
      link="/hazardous-materials/"
      name="Hazardous Material"
      exportModelName="HazardousMaterial"
      filterColumn="name"
      tableFacetedFilters={filters}
      TableSheet={HazardousMaterialDialog}
      TableEditSheet={HazardousMaterialEditDialog}
      addPermissionName="hazardousmaterial.add"
    />
  );
}
