/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import AdminLayout from "@/components/admin-page/layout";
import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { HazardousMaterialEditDialog } from "@/components/hazmat-seg-rules-edit-dialog";
import { HazmatSegRulesDialog } from "@/components/hazmat-seg-rules-table-dialog";
import { Badge } from "@/components/ui/badge";
import {
  getHazardousClassLabel,
  hazardousClassChoices,
  segregationTypeChoices,
} from "@/lib/choices";
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
    cell: ({ row }) => getHazardousClassLabel(row.original.classA),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "classB",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Class B" />
    ),
    cell: ({ row }) => getHazardousClassLabel(row.original.classB),
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
  {
    columnName: "classA",
    title: "Class A",
    options: hazardousClassChoices,
  },
  {
    columnName: "classB",
    title: "Class B",
    options: hazardousClassChoices,
  },
];

export default function HazardousMaterialSegregation() {
  return (
    <AdminLayout>
      <DataTable
        queryKey="hazardousMaterialsSegregations"
        columns={columns}
        link="/hazardous-material-segregations/"
        name="Hazmat Seg. Rules"
        exportModelName="hazardous_material_segregations"
        filterColumn="classA"
        tableFacetedFilters={filters}
        TableSheet={HazmatSegRulesDialog}
        TableEditSheet={HazardousMaterialEditDialog}
        addPermissionName="hazardousmaterialsegregation.add"
      />
    </AdminLayout>
  );
}
