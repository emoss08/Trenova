import { SettingsPage } from "@/components/settings/settings-page";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { CarrierSettlementControlSkeleton } from "./skeleton";

const CarrierSettlementControlForm = lazy(
  () => import("./_components/carrier-settlement-control-form"),
);

export function CarrierSettlementControlPage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Carrier settlement control")}
      description={t("When carriers are paid, how their invoices are matched, and where pay posts")}
      skeleton={<CarrierSettlementControlSkeleton />}
    >
      <CarrierSettlementControlForm />
    </SettingsPage>
  );
}
