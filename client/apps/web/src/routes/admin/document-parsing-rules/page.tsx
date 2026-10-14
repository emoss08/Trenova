import { useT } from "@trenova/shared/i18n/use-t";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { DocumentParsingRulesSkeleton } from "./skeleton";

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
      <LazyComponent fallback={<DocumentParsingRulesSkeleton />}>
        <DocumentParsingRulePageContent />
      </LazyComponent>
    </PageLayout>
  );
}
