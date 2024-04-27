import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import {
  DataTableColumnHeader,
  DataTableTooltipColumnHeader,
} from "@/components/common/table/data-table-column-header";
import { EquipmentStatusBadge } from "@/components/common/table/data-table-components";
import { TractorDialog } from "@/components/tractor-table-dialog";
import { TractorTableEditSheet } from "@/components/tractor-table-edit-dialog";
import { equipmentStatusChoices, type Tractor } from "@/types/equipment";
import { type FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<Tractor>[] = [
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
    cell: ({ row }) => <EquipmentStatusBadge status={row.getValue("status")} />,
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
    accessorFn: (row) => `${row.edges?.equipmentType?.code}`,
    header: "Equipment Type",
    cell: ({ row }) => {
      if (row.original.edges?.equipmentType?.color) {
        return (
          <div className="text-foreground flex items-center space-x-2 text-sm font-medium">
            <div
              className={"mx-2 size-2 rounded-xl"}
              style={{
                backgroundColor: row.original.edges?.equipmentType?.color,
              }}
            />
            {row.original.edges?.equipmentType?.code}
          </div>
        );
      } else {
        return row.original.edges?.equipmentType?.code;
      }
    },
  },
  {
    id: "assignedTo",
    accessorFn: (row) =>
      `${row.edges?.primaryWorker?.firstName} ${row.edges?.primaryWorker?.lastName} `,
    header: () => (
      <DataTableTooltipColumnHeader
        title="Assigned Worker"
        tooltip="The Primary worker assigned to this tractor."
      />
    ),
    cell: ({ row }) => {
      return row.getValue("assignedTo");
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
];

const filters: FilterConfig<Tractor>[] = [
  {
    columnName: "status",
    title: "Status",
    options: equipmentStatusChoices,
  },
];

export default function TractorPage() {
  return (
    <DataTable
      queryKey="trailer-table-data"
      columns={columns}
      link="/tractors/"
      name="Tractor"
      exportModelName="tractors"
      filterColumn="code"
      tableFacetedFilters={filters}
      TableSheet={TractorDialog}
      TableEditSheet={TractorTableEditSheet}
      addPermissionName="create_tractor"
    />
  );
}
