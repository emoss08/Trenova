import AdminLayout from "@/components/admin-page/layout";
import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import {
  DataTableColumnHeader,
  DataTableTooltipColumnHeader,
} from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { TableChangeAlertEditSheet } from "@/components/table-change-edit-sheet";
import { TableChangeAlertSheet } from "@/components/table-change-sheet";
import {
  databaseActionChoices,
  sourceChoices,
  tableStatusChoices,
} from "@/lib/choices";
import { type TableChangeAlert } from "@/types/organization";
import { type FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";

const actionColor = (color: string) => {
  switch (color) {
    case "Insert":
      return "#15803d";
    case "Update":
      return "#2563eb";
    case "Delete":
      return "#b91c1c";
    default:
      return "#9c25eb";
  }
};

const columns: ColumnDef<TableChangeAlert>[] = [
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
    accessorKey: "databaseAction",
    header: "Database Action",
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
    cell: ({ row }) => {
      return (
        <div className="text-foreground flex items-center space-x-2 text-sm font-medium">
          <div
            className={"mx-2 size-2 rounded-xl"}
            style={{
              backgroundColor: actionColor(row.original.databaseAction),
            }}
          />
          {row.original.databaseAction}
        </div>
      );
    },
  },
  {
    accessorKey: "source",
    header: "Source",
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "tableName",
    header: () => (
      <DataTableTooltipColumnHeader
        title="Table Name"
        tooltip="Table is the name of the table that will be monitored for changes."
      />
    ),
  },
  {
    accessorKey: "topicName",
    header: () => (
      <DataTableTooltipColumnHeader
        title="Topic Name"
        tooltip="Topic is the name of the Kafka topic that will be used to publish the message."
      />
    ),
  },
];

const filters: FilterConfig<TableChangeAlert>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
  {
    columnName: "databaseAction",
    title: "Database Action",
    options: databaseActionChoices,
  },

  {
    columnName: "source",
    title: "Source",
    options: sourceChoices,
  },
];

export default function TableChangeAlerts() {
  return (
    <AdminLayout>
      <DataTable
        queryKey="table-change-alert-data"
        columns={columns}
        link="/table-change-alerts/"
        name="Table Change Alert"
        exportModelName="table_change_alerts"
        filterColumn="name"
        tableFacetedFilters={filters}
        TableSheet={TableChangeAlertSheet}
        TableEditSheet={TableChangeAlertEditSheet}
        addPermissionName="create_tablechangealert"
      />
    </AdminLayout>
  );
}
