import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/rate-agreement-table"));

export function RateAgreementPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Rate Agreements"),
        description: t(
          "The contracts that decide what a shipment costs, and the lanes each one prices",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
