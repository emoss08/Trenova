import { SettingsPage } from "@/components/settings/settings-page";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { DistanceControlSkeleton } from "./skeleton";

const DistanceControlForm = lazy(() => import("./_components/distance-control-form"));

export function DistanceControlsPage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Distance control")}
      description={t("Set how mileage is routed, kept and reused")}
      skeleton={<DistanceControlSkeleton />}
    >
      <DistanceControlForm />
    </SettingsPage>
  );
}
