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
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { EmailProfileTableEditDialog } from "@/components/email-profile/email-profile-table-edit-dialog";
import { TableChangeAlertSheet } from "@/components/table-change-alerts/table-change-sheet";
import { Card, CardContent } from "@/components/ui/card";
import { TableChangeAlert } from "@/types/organization";
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
  },
  {
    accessorKey: "source",
    header: "Source",
  },
  {
    accessorKey: "table",
    header: "Table",
  },
];

export default function TableChangeAlerts() {
  return (
    <AdminLayout>
      <Card>
        <CardContent>
          <DataTable
            queryKey="table-change-alert-data"
            columns={columns}
            link="/table_change_alerts/"
            name="Table Change Alert"
            exportModelName="TableChangeAlert"
            filterColumn="name"
            TableSheet={TableChangeAlertSheet}
            TableEditSheet={EmailProfileTableEditDialog}
            addPermissionName="view_tablechangealert"
          />
        </CardContent>
      </Card>
    </AdminLayout>
  );
}
