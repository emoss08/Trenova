import { SuspenseLoader } from "@/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const DocumentIntelligenceForm = lazy(
  () => import("./_components/document-intelligence-form"),
);

export function DocumentIntelligencePage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Document Intelligence"
        description="Configure OCR, classification, extraction, and shipment draft behavior"
      />
      <SuspenseLoader>
        <DocumentIntelligenceForm />
      </SuspenseLoader>
    </AdminPageLayout>
  );
}
