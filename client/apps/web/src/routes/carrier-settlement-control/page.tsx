import { useT } from "@trenova/shared/i18n/use-t";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { CarrierSettlementControlSkeleton } from "./skeleton";

const CarrierSettlementControlForm = lazy(
  () => import("./_components/carrier-settlement-control-form"),
);

export function CarrierSettlementControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Carrier settlement control"),
        description: t(
          "Configure carrier pay periods, cost accrual triggers, batch automation, invoice-match tolerance, and AP posting accounts",
        ),
      }}
    >
      <LazyComponent fallback={<CarrierSettlementControlSkeleton />}>
        <CarrierSettlementControlForm />
      </LazyComponent>
    </PageLayout>
  );
}
