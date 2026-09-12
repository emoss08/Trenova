import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { PaymentStatsRow } from "./_components/payment-stats-row";

const Table = lazy(() => import("./_components/payments-table"));

export function CustomerPaymentsPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Customer Payments"),
        description: t("Record, apply, and reverse customer payments with full GL traceability."),
      }}
      className="p-0"
    >
      <div className="mx-4 mt-3 flex flex-col gap-4">
        <PaymentStatsRow />
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
