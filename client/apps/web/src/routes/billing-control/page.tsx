import { SettingsPage } from "@/components/settings/settings-page";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { BillingControlSkeleton } from "./skeleton";

const BillingControlForm = lazy(() => import("./_components/billing-control-form"));

export function BillingControlPage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Billing control")}
      description={t("Configure and manage your billing control settings")}
      skeleton={<BillingControlSkeleton />}
    >
      <BillingControlForm />
    </SettingsPage>
  );
}
