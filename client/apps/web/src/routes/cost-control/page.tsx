import { SettingsPage } from "@/components/settings/settings-page";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { CostControlSkeleton } from "./skeleton";

const CostControlForm = lazy(() => import("./_components/cost-control-form"));

export function CostControlPage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Cost control")}
      description={t(
        "Configure the cost-per-mile model that powers shipment profitability estimates",
      )}
      skeleton={<CostControlSkeleton />}
    >
      <CostControlForm />
    </SettingsPage>
  );
}
