import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const Table = lazy(() => import("./_components/role-table"));

export function RolesPage() {
  const t = useT();

  return (
    <div className="flex flex-col p-6">
      <PageHeader
        title={t("Roles")}
        description={t("Manage roles and permissions for your organization")}
        className="p-0 py-4"
      />
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </div>
  );
}
