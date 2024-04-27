import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { ServiceTypeEditDialog } from "@/components/service-type-edit-table-dialog";
import { ServiceTypeDialog } from "@/components/service-type-table-dialog";
import { tableStatusChoices } from "@/lib/choices";
import { truncateText } from "@/lib/utils";
import { type ServiceType } from "@/types/shipment";
import { type FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<ServiceType>[] = [
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
    accessorKey: "code",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Code" />
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "description",
    header: "Description",
    cell: ({ row }) => truncateText(row.original.description as string, 30),
  },
];

const filters: FilterConfig<ServiceType>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
];

export default function ServiceTypes() {
  return (
    <DataTable
      queryKey="service-type-table-data"
      columns={columns}
      link="/service-types/"
      name="Service Types"
      exportModelName="service_types"
      filterColumn="code"
      tableFacetedFilters={filters}
      TableSheet={ServiceTypeDialog}
      TableEditSheet={ServiceTypeEditDialog}
      addPermissionName="servicetype.add"
    />
  );
}
