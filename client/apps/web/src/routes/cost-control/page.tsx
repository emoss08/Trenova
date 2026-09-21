import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy, Suspense } from "react";
import { PageSkeleton } from "./skeleton";

const CostControlForm = lazy(() => import("./_components/cost-control-form"));

export function CostControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Cost Control"),
        description: t(
          "Configure the cost-per-mile model that powers shipment profitability estimates",
        ),
      }}
    >
      <Suspense fallback={<PageSkeleton />}>
        <div className="p-4">
          <CostControlForm />
        </div>
      </Suspense>
    </PageLayout>
  );
}
