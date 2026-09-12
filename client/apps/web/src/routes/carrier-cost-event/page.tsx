import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/cost-events-table"));

export function CarrierCostEventsPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Carrier Cost Events"),
        description: t(
          "Purchased-transportation cost accrued as carrier-covered shipments deliver — the source ledger behind every carrier settlement.",
        ),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
