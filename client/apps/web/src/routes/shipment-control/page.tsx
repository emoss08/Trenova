import { SettingsPage } from "@/components/settings/settings-page";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { ShipmentControlSkeleton } from "./skeleton";

const ShipmentControlForm = lazy(() => import("./_components/shipment-control-form"));

export function ShipmentControlPage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Shipment control")}
      description={t("Configure and manage your shipment control settings")}
      skeleton={<ShipmentControlSkeleton />}
    >
      <ShipmentControlForm />
    </SettingsPage>
  );
}
