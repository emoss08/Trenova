import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/document-packet-rule-table"));

export function DocumentPacketRulesPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Document Packet Rules"),
        description: t("Configure which document types are required for each resource type"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
