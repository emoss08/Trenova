import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const UserRolesTable = lazy(() => import("./_components/user-roles-table"));

export function UsersPage() {
  return (
    <div className="flex flex-col gap-y-3 p-6">
      <PageHeader
        title="Users"
        description="Manage users and their role assignments"
        className="p-0 py-4"
      />
      <DataTableLazyComponent>
        <UserRolesTable />
      </DataTableLazyComponent>
    </div>
  );
}
