import { useT } from "@trenova/shared/i18n/use-t";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy, Suspense } from "react";
import { PageSkeleton } from "./skeleton";

const CostControlForm = lazy(() => import("./_components/cost-control-form"));

export function CostControlPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Cost Control")}
        description={t("Configure the cost-per-mile model that powers shipment profitability estimates")}
      />
      <Suspense fallback={<PageSkeleton />}>
        <div className="p-4">
          <CostControlForm />
        </div>
      </Suspense>
    </AdminPageLayout>
  );
}
