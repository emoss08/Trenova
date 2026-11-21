import { SettingsPage } from "@/components/settings/settings-page";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { DocumentIntelligenceSkeleton } from "./skeleton";

const DocumentIntelligenceForm = lazy(() => import("./_components/document-intelligence-form"));

export function DocumentIntelligencePage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Document extraction")}
      description={t("Choose how uploaded documents are read, filed and drafted into shipments")}
      skeleton={<DocumentIntelligenceSkeleton />}
    >
      <DocumentIntelligenceForm />
    </SettingsPage>
  );
}
