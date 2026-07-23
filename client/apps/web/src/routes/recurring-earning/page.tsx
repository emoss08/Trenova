import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/earnings-table"));

export function RecurringEarningsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Recurring Earnings",
        description:
          "Standing per-settlement earnings: per diem, safety and performance bonuses, stipends, and equipment rental payments with caps.",
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
