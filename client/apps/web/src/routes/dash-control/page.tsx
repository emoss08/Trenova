import { SettingsPage } from "@/components/settings/settings-page";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { DashControlSkeleton } from "./skeleton";

const DashControlForm = lazy(() => import("./_components/dash-control-form"));

export function DashControlPage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Dash control")}
      description={t("Choose what drivers can see and do in the Dash driver portal")}
      skeleton={<DashControlSkeleton />}
    >
      <DashControlForm />
    </SettingsPage>
  );
}
