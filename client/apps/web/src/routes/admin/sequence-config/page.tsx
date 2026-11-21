import { SettingsPage } from "@/components/settings/settings-page";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { SequenceConfigSkeleton } from "./skeleton";

const SequenceConfigForm = lazy(() => import("./_components/sequence-config-form"));

export function SequenceConfigPage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Sequence configuration")}
      description={t("How the numbers on shipments, invoices, journals and settlements are written")}
      skeleton={<SequenceConfigSkeleton />}
    >
      <SequenceConfigForm />
    </SettingsPage>
  );
}
