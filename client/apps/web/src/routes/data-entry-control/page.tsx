import { SettingsPage } from "@/components/settings/settings-page";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { DataEntryControlSkeleton } from "./skeleton";

const DataEntryControlForm = lazy(() => import("./_components/data-entry-control-form"));

export function DataEntryControlPage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Data entry control")}
      description={t("Set how entered text is capitalized across the system")}
      skeleton={<DataEntryControlSkeleton />}
    >
      <DataEntryControlForm />
    </SettingsPage>
  );
}
