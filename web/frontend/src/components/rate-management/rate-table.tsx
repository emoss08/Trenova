import { CommodityDialog } from "@/components/commodity-dialog";
import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { WorkerEditDialog } from "@/components/worker/worker-table-edit-dialog";
import { tableStatusChoices } from "@/lib/choices";
import { type Rate } from "@/types/dispatch";
import { FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "../ui/tooltip";

const columns: ColumnDef<Rate>[] = [
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
    accessorKey: "rateNumber",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Rate Number" />
    ),
  },
  {
    id: "customer",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Customer" />
    ),
    accessorFn: (row) => `${row.edges?.customer.name || "No Customer"}`,
    cell: ({ row, cell }) => {
      const code = row.original.edges?.customer.code;
      return (
        <TooltipProvider delayDuration={100}>
          <Tooltip>
            <TooltipTrigger asChild>
              <span>{cell.getValue() as string}</span>
            </TooltipTrigger>
            <TooltipContent sideOffset={10}>
              <span>Code: {code}</span>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      );
    },
  },
];

const filters: FilterConfig<Rate>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
];

export function RateTable() {
  return (
    <DataTable
      queryKey="rates"
      columns={columns}
      link="/rates/"
      name="Rate"
      exportModelName="rates"
      tableFacetedFilters={filters}
      filterColumn="rateNumber"
      TableSheet={CommodityDialog}
      TableEditSheet={WorkerEditDialog}
      addPermissionName="rate.add"
    />
  );
}
