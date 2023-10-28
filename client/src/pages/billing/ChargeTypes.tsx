/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import { DataTable, StatusBadge } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/common/fields/checkbox";
import { tableStatusChoices } from "@/lib/constants";
import { FilterConfig } from "@/types/tables";
import { ColumnDef } from "@tanstack/react-table";
import { ChargeType } from "@/types/billing";
import { ChargeTypeDialog } from "@/components/charge-types/charge-type-dialog";
import { ChargeTypeEditSheet } from "@/components/charge-types/charge-type-edit-dialog";

const columns: ColumnDef<ChargeType>[] = [
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
  },
];

const filters: FilterConfig<ChargeType>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
];

export default function ChargeTypes() {
  return (
    <Card>
      <CardContent>
        <DataTable
          queryKey="charge-type-table-data"
          columns={columns}
          link="/charge_types/"
          name="Charge Type"
          exportModelName="ChargeType"
          filterColumn="name"
          tableFacetedFilters={filters}
          TableSheet={ChargeTypeDialog}
          TableEditSheet={ChargeTypeEditSheet}
        />
      </CardContent>
    </Card>
  );
}
