import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const Table = lazy(() => import("./_components/document-packet-rule-table"));

export function DocumentPacketRulesPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Document Packet Rules",
        description:
          "Configure which document types are required for each resource type",
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
