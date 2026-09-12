import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/deductions-table"));

export function RecurringDeductionsPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Recurring Deductions"),
        description: t(
          "Standing per-settlement deductions: insurance, lease payments, escrow contributions, and loan repayments with caps.",
        ),
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
