import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const DocumentParsingRulePageContent = lazy(
  () => import("./_components/document-parsing-rule-page-content"),
);

export function DocumentParsingRulesPage() {
  const t = useT();

  return (
    <PageLayout
      fill
      pageHeaderProps={{
        title: t("Document parsing rules"),
        description: t(
          "Define provider-specific parsing rules, test with fixtures, and simulate extraction results",
        ),
      }}
    >
      <SuspenseLoader>
        <DocumentParsingRulePageContent />
      </SuspenseLoader>
    </PageLayout>
  );
}
