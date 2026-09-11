import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const Table = lazy(() => import("./_components/hazmat-segregation-rule-table"));

export function HazmatSegregationRulesPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Hazmat Segregation Rules")}
        description={t("Manage and configure hazmat segregation rules for your organization")}
      />
      <DataTableLazyComponent>
        <div className="px-4">
          <Table />
        </div>
      </DataTableLazyComponent>
    </AdminPageLayout>
  );
}
