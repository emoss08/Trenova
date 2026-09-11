import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const DocumentParsingRulePageContent = lazy(
  () => import("./_components/document-parsing-rule-page-content"),
);

export function DocumentParsingRulesPage() {
  const t = useT();

  return (
    <AdminPageLayout className="flex h-[calc(100vh-3rem)] flex-col">
      <PageHeader
        title={t("Document Parsing Rules")}
        description={t("Define provider-specific parsing rules, test with fixtures, and simulate extraction results")}
      />
      <SuspenseLoader>
        <DocumentParsingRulePageContent />
      </SuspenseLoader>
    </AdminPageLayout>
  );
}
