import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { PaymentStatsRow } from "./_components/payment-stats-row";

const Table = lazy(() => import("./_components/payments-table"));

export function CustomerPaymentsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Customer Payments",
        description: "Record, apply, and reverse customer payments with full GL traceability.",
      }}
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
