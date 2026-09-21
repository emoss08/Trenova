import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const UserRolesTable = lazy(() => import("./_components/user-roles-table"));

export function UsersPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Users"),
        description: t("Manage users and their role assignments"),
      }}
    >
      <DataTableLazyComponent>
        <UserRolesTable />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
