import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const UserRolesTable = lazy(() => import("./_components/user-roles-table"));

export function UsersPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader title={t("Users")} description={t("Manage users and their role assignments")} />
      <div className="p-4">
        <DataTableLazyComponent>
          <UserRolesTable />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
