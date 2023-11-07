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
import { tableStatusChoices, yesAndNoChoices } from "@/lib/constants";
import { FilterConfig } from "@/types/tables";
import { ColumnDef } from "@tanstack/react-table";
import { truncateText } from "@/lib/utils";
import { Commodity } from "@/types/commodities";
import { Badge } from "@/components/ui/badge";
import { CommodityDialog } from "@/components/commodities/commodity-dialog";
import { CommodityEditDialog } from "@/components/commodities/commodity-edit-table-dialog";

function HazmatBadge({ isHazmat }: { isHazmat: string }) {
  return (
    <Badge variant={isHazmat === "Y" ? "default" : "destructive"}>
      {isHazmat === "Y" ? "Yes" : "No"}
    </Badge>
  );
}

const columns: ColumnDef<Commodity>[] = [
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
    cell: ({ row }) => truncateText(row.original.description as string, 25),
  },
  {
    id: "temp_range",
    accessorFn: (row) => `${row.minTemp} - ${row.maxTemp}`,
    header: "Temperature Range",
    cell: ({ row, column }) => {
      if (row.original.minTemp === null && row.original.maxTemp === null) {
        return "N/A";
      }
      return row.getValue(column.id);
    },
  },
  {
    accessorKey: "isHazmat",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Is Hazmat" />
    ),
    cell: ({ row }) => <HazmatBadge isHazmat={row.original.isHazmat} />,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
];

const filters: FilterConfig<Commodity>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
  {
    columnName: "isHazmat",
    title: "Is Hazmat",
    options: yesAndNoChoices,
  },
];

export default function DelayCodes() {
  return (
    <Card>
      <CardContent>
        <DataTable
          queryKey="delay-code-table-data"
          columns={columns}
          link="/delay_codes/"
          name="DelayCode"
          exportModelName="DelayCode"
          filterColumn="name"
          tableFacetedFilters={filters}
          TableSheet={CommodityDialog}
          TableEditSheet={CommodityEditDialog}
        />
      </CardContent>
    </Card>
  );
}
