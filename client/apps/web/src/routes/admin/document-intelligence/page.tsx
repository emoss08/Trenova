import { useT } from "@trenova/shared/i18n/use-t";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { DocumentIntelligenceSkeleton } from "./skeleton";

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
      <LazyComponent fallback={<DocumentIntelligenceSkeleton />}>
        <DocumentIntelligenceForm />
      </LazyComponent>
    </PageLayout>
  );
}
