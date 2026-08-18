import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const Table = lazy(() => import("./_components/rate-agreement-table"));

export function RateAgreementPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Rate Agreements"
        description="The contracts that decide what a shipment costs, and the lanes each one prices"
      />
      <div className="p-4">
        <DataTableLazyComponent>
          <Table />
        </DataTableLazyComponent>
      </div>
    </AdminPageLayout>
  );
}
