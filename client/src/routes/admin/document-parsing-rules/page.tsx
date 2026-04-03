import { SuspenseLoader } from "@/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const DocumentParsingRulePageContent = lazy(
  () => import("./_components/document-parsing-rule-page-content"),
);

export function DocumentParsingRulesPage() {
  return (
    <AdminPageLayout className="flex h-[calc(100vh-3rem)] flex-col">
      <PageHeader
        title="Document Parsing Rules"
        description="Define provider-specific parsing rules, test with fixtures, and simulate extraction results"
      />
      <SuspenseLoader>
        <DocumentParsingRulePageContent />
      </SuspenseLoader>
    </AdminPageLayout>
  );
}
