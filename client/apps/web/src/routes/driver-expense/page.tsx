import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/expenses-table"));

export function DriverExpensesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Driver Expenses",
        description:
          "Driver-submitted out-of-pocket expenses — review receipts, approve to reimburse on the driver's open settlement, or reject with an explanation.",
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
