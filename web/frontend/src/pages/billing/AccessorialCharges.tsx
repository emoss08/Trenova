/**
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

import { ACDialog } from "@/components/accessorial-charge-table-dialog";
import { AccessorialChargeTableEditDialog } from "@/components/accessorial-charge-table-edit-dialog";
import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { Badge } from "@/components/ui/badge";
import { tableStatusChoices } from "@/lib/choices";
import { ConvertDecimalToUSD, truncateText } from "@/lib/utils";
import { AccessorialCharge } from "@/types/billing";
import { type FilterConfig } from "@/types/tables";
import { fuelMethodChoices } from "@/utils/apps/billing";
import { type ColumnDef } from "@tanstack/react-table";

function DetentionBadge({ isDetention }: { isDetention: boolean }) {
  return (
    <Badge variant={isDetention ? "active" : "inactive"}>
      {isDetention ? "Yes" : "No"}
    </Badge>
  );
}

const columns: ColumnDef<AccessorialCharge>[] = [
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
    accessorKey: "method",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Method" />
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "isDetention",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Is Detention" />
    ),
    cell: ({ row }) => (
      <DetentionBadge isDetention={row.original.isDetention} />
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
    accessorKey: "amount",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Charge Amount" />
    ),
    cell: ({ row }) => ConvertDecimalToUSD(row.original.amount),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
];

const filters: FilterConfig<AccessorialCharge>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
  {
    columnName: "isDetention",
    title: "Detention",
    options: [
      { label: "Yes", value: true },
      { label: "No", value: false },
    ],
  },
  { columnName: "method", title: "Method", options: fuelMethodChoices },
];

export default function AccessorialCharges() {
  return (
    <DataTable
      queryKey="accessorialCharges"
      columns={columns}
      link="/accessorial-charges/"
      name="Accessorial Charge"
      exportModelName="accessorial_charges"
      filterColumn="code"
      tableFacetedFilters={filters}
      TableSheet={ACDialog}
      TableEditSheet={AccessorialChargeTableEditDialog}
      addPermissionName="accessorial_charge:create"
    />
  );
}
