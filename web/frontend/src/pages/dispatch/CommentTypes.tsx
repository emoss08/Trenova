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

import { CommentTypeDialog } from "@/components/comment-type-table-dialog";
import { CommentTypeEditSheet } from "@/components/comment-type-table-edit-dialog";
import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { Badge } from "@/components/ui/badge";
import { tableStatusChoices } from "@/lib/choices";
import { type CommentType } from "@/types/dispatch";
import { type FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";

const getSeverityColor = (severity: string) => {
  switch (severity) {
    case "High":
      return "inactive";
    case "Medium":
      return "info";
    default:
      return "active";
  }
};

const columns: ColumnDef<CommentType>[] = [
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
    accessorKey: "severity",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Severity" />
    ),
    cell: ({ row }) => {
      return (
        <Badge
          className="px-2.5 py-0.5 text-xs"
          variant={getSeverityColor(row.original.severity)}
        >
          {row.original.severity}
        </Badge>
      );
    },
  },
  {
    accessorKey: "description",
    header: "Description",
  },
];

const filters: FilterConfig<CommentType>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
  },
];

export default function CommentTypes() {
  return (
    <DataTable
      addPermissionName="commenttype.add"
      queryKey="commentTypes"
      columns={columns}
      link="/comment-types/"
      name="Comment Types"
      exportModelName="comment_types"
      filterColumn="name"
      tableFacetedFilters={filters}
      TableSheet={CommentTypeDialog}
      TableEditSheet={CommentTypeEditSheet}
    />
  );
}
