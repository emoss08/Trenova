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

import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { Checkbox } from "@/components/common/fields/checkbox";
import { tableStatusChoices, yesAndNoChoicesBoolean } from "@/lib/constants";
import { FilterConfig } from "@/types/tables";
import { ColumnDef } from "@tanstack/react-table";
import { truncateText } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { DelayCode } from "@/types/dispatch";
import { DelayCodeDialog } from "@/components/delay-codes/delay-code-table-dialog";
import { DelayCodeEditDialog } from "@/components/delay-codes/delay-code-edit-table-dialog";
import { StatusBadge } from "@/components/common/table/data-table-components";

function CarrierOrDriverBadge({
  carrierOrDriver,
}: {
  carrierOrDriver: boolean;
}) {
  return (
    <Badge variant={carrierOrDriver ? "default" : "destructive"}>
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
    accessorKey: "code",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Code" />
    ),
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
  {
    columnName: "fCarrierOrDriver",
    title: "Fault of Carrier Or Driver",
    options: yesAndNoChoicesBoolean,
  },
];

export default function DelayCodes() {
  return (
    <DataTable
      queryKey="delay-code-table-data"
      columns={columns}
      link="/delay_codes/"
      name="Delay Code"
      exportModelName="DelayCode"
      filterColumn="code"
      tableFacetedFilters={filters}
      TableSheet={DelayCodeDialog}
      TableEditSheet={DelayCodeEditDialog}
    />
  );
}
