import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/service-failure-reason-code-table"));

export function ServiceFailureReasonCodesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Service Failure Reason Codes"),
        description: t("Manage operational exception reasons and EDI 214 defaults"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
