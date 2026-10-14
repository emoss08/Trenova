import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { useFactoringRecordRedirect } from "./_components/use-factoring-record-redirect";

const Table = lazy(() => import("./_components/factoring-company-table"));

export function FactoringCompaniesPage() {
  const t = useT();
  useFactoringRecordRedirect();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Factoring companies"),
        description: t(
          "The factors your receivables are sold to. Customers assigned to one are invoiced with its remit-to and notice of assignment.",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
