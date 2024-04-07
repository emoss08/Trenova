import { CommodityDialog } from "@/components/commodity-dialog";
import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { WorkerEditDialog } from "@/components/worker/worker-table-edit-dialog";
import { type Worker } from "@/types/worker";
import { type ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<Worker>[] = [
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
  },
  {
    accessorKey: "workerType",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Worker Type" />
    ),
  },
  {
    accessorKey: "name",
    accessorFn: (row) => `${row.firstName} ${row.lastName}`,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
  },
];

export default function WorkerPage() {
  return (
    <DataTable
      queryKey="worker-table-data"
      columns={columns}
      link="/workers/"
      name="Worker"
      exportModelName="Worker"
      filterColumn="name"
      TableSheet={CommodityDialog}
      TableEditSheet={WorkerEditDialog}
      addPermissionName="add_worker"
    />
  );
}
