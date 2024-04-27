import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { DelayCodeEditDialog } from "@/components/delay-code-edit-table-dialog";
import { DelayCodeDialog } from "@/components/delay-code-table-dialog";
import { Badge } from "@/components/ui/badge";
import { tableStatusChoices } from "@/lib/choices";
import { truncateText } from "@/lib/utils";
import { DelayCode } from "@/types/dispatch";
import { type FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";

function CarrierOrDriverBadge({
  carrierOrDriver,
}: {
  carrierOrDriver: boolean;
}) {
  return (
    <Badge variant={carrierOrDriver ? "active" : "inactive"}>
      {carrierOrDriver ? "Yes" : "No"}
    </Badge>
  );
}

const columns: ColumnDef<DelayCode>[] = [
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
    id: "code",
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
    cell: ({ row }) => {
      if (row.original.color) {
        return (
          <div className="text-foreground flex items-center space-x-2 text-sm font-medium">
            <div
              className={"mx-2 size-2 rounded-xl"}
              style={{ backgroundColor: row.original.color }}
            />
            {row.original.code}
          </div>
        );
      } else {
        return row.original.code;
      }
    },
  },
  {
    accessorKey: "description",
    header: "Description",
    cell: ({ row }) => truncateText(row.original.description as string, 25),
  },
  {
    accessorKey: "fCarrierOrDriver",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Fault of Carrier Or Driver"
      />
    ),
    cell: ({ row }) => (
      <CarrierOrDriverBadge carrierOrDriver={row.original.fCarrierOrDriver} />
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
];

const filters: FilterConfig<DelayCode>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
];

export default function DelayCodes() {
  return (
    <DataTable
      queryKey="delay-code-table-data"
      columns={columns}
      link="/delay-codes/"
      name="Delay Code"
      exportModelName="delay_codes"
      filterColumn="code"
      tableFacetedFilters={filters}
      TableSheet={DelayCodeDialog}
      TableEditSheet={DelayCodeEditDialog}
      addPermissionName="create_delaycode"
    />
  );
}
