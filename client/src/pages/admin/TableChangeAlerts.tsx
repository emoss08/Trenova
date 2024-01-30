/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import AdminLayout from "@/components/admin-page/layout";
import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import {
  DataTableColumnHeader,
  DataTableTooltipColumnHeader,
} from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { TableChangeAlertEditSheet } from "@/components/table-change-alerts/table-change-edit-sheet";
import { TableChangeAlertSheet } from "@/components/table-change-alerts/table-change-sheet";
import { databaseActionChoices, sourceChoices } from "@/lib/choices";
import { tableStatusChoices } from "@/lib/constants";
import { TableChangeAlert } from "@/types/organization";
import { FilterConfig } from "@/types/tables";
import { ColumnDef } from "@tanstack/react-table";

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
  },
  {
    accessorKey: "source",
    header: "Source",
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "table",
    header: () => (
      <DataTableTooltipColumnHeader
        title="Table"
        tooltip="Table is the name of the table that will be monitored for changes."
      />
    ),
  },
  {
    accessorKey: "topic",
    header: () => (
      <DataTableTooltipColumnHeader
        title="Topic"
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
        link="/table_change_alerts/"
        name="Table Change Alert"
        exportModelName="TableChangeAlert"
        filterColumn="name"
        tableFacetedFilters={filters}
        TableSheet={TableChangeAlertSheet}
        TableEditSheet={TableChangeAlertEditSheet}
        addPermissionName="view_tablechangealert"
      />
    </AdminLayout>
  );
}
