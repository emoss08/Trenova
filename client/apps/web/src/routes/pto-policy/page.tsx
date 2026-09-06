import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/pto-policy-table"));

export function PTOPoliciesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "PTO Policies",
        description:
          "Accrual rules for paid time off — how days are earned, capped, and carried over, and which drivers each policy governs.",
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
