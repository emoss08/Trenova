import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const Table = lazy(() => import("./_components/jurisdiction-rule-table"));

export function JurisdictionRulesPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Jurisdiction Rules"
        description="Oversize and overweight limits per state. These limits are shared across every organization; record a carrier override to hold your own fleet to something stricter."
      />
      <div className="p-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
