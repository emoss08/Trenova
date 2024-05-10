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
