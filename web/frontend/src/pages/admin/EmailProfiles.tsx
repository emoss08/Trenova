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
import {
  DataTableColumnHeader,
  DataTableTooltipColumnHeader,
} from "@/components/common/table/data-table-column-header";
import { BoolStatusBadge } from "@/components/common/table/data-table-components";
import { EmailProfileDialog } from "@/components/email-profile-table-dialog";
import { EmailProfileTableEditDialog } from "@/components/email-profile-table-edit-dialog";
import { type EmailProfile } from "@/types/organization";
import { type ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<EmailProfile>[] = [
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
    accessorKey: "isDefault",
    header: () => (
      <DataTableTooltipColumnHeader
        title="Default"
        tooltip="Is this the default email profile for the organization?"
      />
    ),
    cell: ({ row }) => <BoolStatusBadge status={row.getValue("isDefault")} />,
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
    accessorKey: "email",
    header: "Email Address",
  },
  {
    accessorKey: "host",
    header: "Host",
  },
  {
    accessorKey: "port",
    header: "Port",
  },
];

export default function EmailProfiles() {
  return (
    <AdminLayout>
      <DataTable
        queryKey="emailProfiles"
        columns={columns}
        link="/email-profiles/"
        name="Email Profile"
        exportModelName="email_profiles"
        filterColumn="name"
        TableSheet={EmailProfileDialog}
        TableEditSheet={EmailProfileTableEditDialog}
        addPermissionName="emailprofile.add"
      />
    </AdminLayout>
  );
}
