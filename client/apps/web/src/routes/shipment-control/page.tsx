import { SuspenseLoader } from "@/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const ShipmentControlForm = lazy(() => import("./_components/shipment-control-form"));

export function ShipmentControlPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Shipment Control"
        description="Configure and manage your shipment control settings"
      />
      <div className="p-4">
        <SuspenseLoader>
          <ShipmentControlForm />
        </SuspenseLoader>
      </div>
    </AdminPageLayout>
  );
}
