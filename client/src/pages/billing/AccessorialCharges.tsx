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

import { ACDialog } from "@/components/accessorial-charges/ac-table-dialog";
import { ACTableEditDialog } from "@/components/accessorial-charges/ac-table-edit-dialog";
import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable, StatusBadge } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { Badge } from "@/components/ui/badge";
import { tableStatusChoices } from "@/lib/constants";
import { truncateText, USDollarFormatString } from "@/lib/utils";
import { AccessorialCharge } from "@/types/billing";
import { FilterConfig } from "@/types/tables";
import {
  fuelMethodChoices,
  FuelMethodChoicesProps,
} from "@/utils/apps/billing";
import { ColumnDef } from "@tanstack/react-table";

function DetentionBadge({ isDetention }: { isDetention: boolean }) {
  return (
    <Badge variant={isDetention ? "default" : "destructive"}>
      {isDetention ? "Yes" : "No"}
    </Badge>
  );
}

function methodText(fuelMethod: FuelMethodChoicesProps) {
  return fuelMethod === "P"
    ? "Percentage"
    : fuelMethod === "D"
    ? "Distance"
    : fuelMethod === "F"
    ? "Fuel"
    : "N/A";
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
    cell: ({ row }) => methodText(row.original.method),
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
    accessorKey: "chargeAmount",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Charge Amount" />
    ),
    cell: ({ row }) => USDollarFormatString(row.original.chargeAmount),
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
      queryKey="accessorial-charges-table-data"
      columns={columns}
      link="/accessorial_charges/"
      name="Accessorial Charge"
      exportModelName="AccessorialCharge"
      filterColumn="code"
      tableFacetedFilters={filters}
      TableSheet={ACDialog}
      TableEditSheet={ACTableEditDialog}
    />
  );
}
