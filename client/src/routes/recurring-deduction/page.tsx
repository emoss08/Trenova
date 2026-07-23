import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/deductions-table"));

export function RecurringDeductionsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Recurring Deductions",
        description:
          "Standing per-settlement deductions: insurance, lease payments, escrow contributions, and loan repayments with caps.",
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
