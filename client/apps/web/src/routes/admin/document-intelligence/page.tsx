import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const DocumentIntelligenceForm = lazy(() => import("./_components/document-intelligence-form"));

export function DocumentIntelligencePage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Document intelligence"),
        description: t("Configure OCR, classification, extraction, and shipment draft behavior"),
      }}
    >
      <SuspenseLoader>
        <DocumentIntelligenceForm />
      </SuspenseLoader>
    </PageLayout>
  );
}
