import { DataTableLazyComponent } from "@/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const UserRolesTable = lazy(() => import("./_components/user-roles-table"));

export function UsersPage() {
  return (
    <AdminPageLayout>
      <PageHeader title="Users" description="Manage users and their role assignments" />
      <div className="p-4">
        <DataTableLazyComponent>
          <UserRolesTable />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
