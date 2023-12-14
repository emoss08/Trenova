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

import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { TrailerDialog } from "@/components/trailers/trailer-table-dialog";
import { TrailerEditDialog } from "@/components/trailers/trailer-table-edit-dialog";
import { Badge } from "@/components/ui/badge";
import {
  Tractor,
  trailerStatusChoices,
  TrailerStatuses,
} from "@/types/equipment";
import { FilterConfig } from "@/types/tables";
import { ColumnDef } from "@tanstack/react-table";

function LastInspectionDate({ lastInspection }: { lastInspection?: string }) {
  return (
    <Badge variant={lastInspection ? "default" : "destructive"}>
      {lastInspection || "Never"}
    </Badge>
  );
}

export function TrailerStatusBadge({ status }: { status: TrailerStatuses }) {
  const mapToStatus = {
    A: "Available",
    OOS: "Out of Service",
    AM: "At Maintenance",
    S: "Sold",
    L: "Lost",
  };

  return (
    <Badge variant={status === "A" ? "default" : "destructive"}>
      {mapToStatus[status]}
    </Badge>
  );
}

const columns: ColumnDef<Tractor>[] = [
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
    cell: ({ row }) => <TrailerStatusBadge status={row.getValue("status")} />,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "code",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Code" />
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "equipTypeName",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Equipment Type" />
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "lastInspection",
    header: "Last Inspection Date",
    cell: ({ row }) => (
      <LastInspectionDate lastInspection={row.getValue("lastInspection")} />
    ),
  },
];

const filters: FilterConfig<Tractor>[] = [
  {
    columnName: "status",
    title: "Status",
    options: trailerStatusChoices,
  },
];

export default function TrailerPage() {
  return (
    <DataTable
      queryKey="trailer-table-data"
      columns={columns}
      link="/tractors/"
      name="Tractor"
      exportModelName="Tractor"
      filterColumn="code"
      tableFacetedFilters={filters}
      TableSheet={TrailerDialog}
      TableEditSheet={TrailerEditDialog}
      addPermissionName="add_trailer"
    />
  );
}