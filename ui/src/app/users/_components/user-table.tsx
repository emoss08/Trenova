import { DataTable } from "@/components/data-table/data-table";
import type { UserSchema } from "@/lib/schemas/user-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./user-columns";
import { EditUserModal } from "./user-edit-modal";

export default function UserTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<UserSchema>
      resource={Resource.User}
      name="User"
      link="/users/"
      queryKey="user-list"
      exportModelName="user"
      extraSearchParams={{
        includeRoles: true,
      }}
      // TableModal={CreateUserModal}
      TableEditModal={EditUserModal}
      columns={columns}
    />
  );
}
