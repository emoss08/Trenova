import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/hazmat-segregation-rule-table"));

export function HazmatSegregationRulesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Hazmat Segregation Rules"),
        description: t("Manage and configure hazmat segregation rules for your organization"),
      }}
    >
      <DataTableLazyComponent>
        <div className="px-4">
          <Table />
        </div>
      </DataTableLazyComponent>
    </PageLayout>
  );
}
