import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { CostControlSkeleton } from "./skeleton";

const CostControlForm = lazy(() => import("./_components/cost-control-form"));

export function CostControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Cost control"),
        description: t(
          "Configure the cost-per-mile model that powers shipment profitability estimates",
        ),
      }}
    >
      <LazyComponent fallback={<CostControlSkeleton />}>
        <CostControlForm />
      </LazyComponent>
    </PageLayout>
  );
}
