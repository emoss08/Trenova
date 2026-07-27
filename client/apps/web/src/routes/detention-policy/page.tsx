import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/detention-policy-table"));

export function DetentionPolicyPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Detention Policies"
        description="Encode each contract's detention terms and see what they would charge before they touch a shipment"
      />
      <div className="p-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
