import { SettingsPage } from "@/components/settings/settings-page";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { SettlementControlSkeleton } from "./skeleton";

const SettlementControlForm = lazy(() => import("./_components/settlement-control-form"));

export function SettlementControlPage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Settlement control")}
      description={t(
        "Configure pay periods, accrual triggers, exception policies, and escrow interest",
      )}
      skeleton={<SettlementControlSkeleton />}
    >
      <SettlementControlForm />
    </SettingsPage>
  );
}
