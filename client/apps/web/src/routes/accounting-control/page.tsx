import { SettingsPage } from "@/components/settings/settings-page";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { AccountingControlSkeleton } from "./skeleton";

const AccountingControlForm = lazy(() => import("./_components/accounting-control-form"));

export function AccountingControlPage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Accounting control")}
      description={t("Configure and manage your accounting control settings")}
      skeleton={<AccountingControlSkeleton />}
    >
      <AccountingControlForm />
    </SettingsPage>
  );
}
