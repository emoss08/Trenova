import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

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
      <SuspenseLoader>
        <CarrierSettlementControlForm />
      </SuspenseLoader>
    </PageLayout>
  );
}
