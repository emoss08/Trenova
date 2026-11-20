import { SettingsPage } from "@/components/settings/settings-page";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { DispatchControlSkeleton } from "./skeleton";

const DispatchControlForm = lazy(() => import("./_components/dispatch-control-form"));

export function DispatchControlPage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Dispatch control")}
      description={t("Configure and manage your dispatch control settings")}
      skeleton={<DispatchControlSkeleton />}
    >
      <DispatchControlForm />
    </SettingsPage>
  );
}
