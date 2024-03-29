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
import { HazardousMaterialEditDialog } from "@/components/hazmat-seg-rules-edit-dialog";
import { HazmatSegRulesDialog } from "@/components/hazmat-seg-rules-table-dialog";
import { Badge } from "@/components/ui/badge";
import { segregationTypeChoices } from "@/lib/choices";
import { type HazardousMaterialSegregationRule } from "@/types/shipment";
import { type FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";

const readableSegType = (type: string) => {
  switch (type) {
    case "NotAllowed":
      return <Badge variant="inactive">Not Allowed</Badge>;
    default:
      return <Badge variant="active">Allowed With Conditions</Badge>;
  }
};

const columns: ColumnDef<HazardousMaterialSegregationRule>[] = [
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
    accessorKey: "classA",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Class A" />
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "classB",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Class B" />
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "segregationType",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Segregation Type" />
    ),
    cell: ({ row }) => readableSegType(row.original.segregationType),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
];

const filters: FilterConfig<HazardousMaterialSegregationRule>[] = [
  {
    columnName: "segregationType",
    title: "Segregation Type",
    options: segregationTypeChoices,
  },
];

export default function HazardousMaterialSegregation() {
  return (
    <AdminLayout>
      <DataTable
        queryKey="hazardous-material-segregation-table-data"
        columns={columns}
        link="/hazardous-material-segregations/"
        name="Hazmat Seg. Rules"
        exportModelName="HazardousMaterialSegregation"
        filterColumn="classA"
        tableFacetedFilters={filters}
        TableSheet={HazmatSegRulesDialog}
        TableEditSheet={HazardousMaterialEditDialog}
        addPermissionName="add_hazardousmaterialsegregation"
      />
    </AdminLayout>
  );
}
