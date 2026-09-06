import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const BenefitsConsole = lazy(() => import("./_components/benefits-console"));

export function BenefitsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Benefits",
        description:
          "What the carrier offers and who is on it. An employee contribution is taken through an ordinary settlement deduction, so it caps, pauses and reverses exactly like every other one.",
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent>
          <BenefitsConsole />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
