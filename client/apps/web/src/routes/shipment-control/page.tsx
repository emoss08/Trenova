import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy } from "react";

const ShipmentControlForm = lazy(() => import("./_components/shipment-control-form"));

export function ShipmentControlPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Shipment Control")}
        description={t("Configure and manage your shipment control settings")}
      />
      <div className="p-4">
        <SuspenseLoader>
          <ShipmentControlForm />
        </SuspenseLoader>
      </div>
    </AdminPageLayout>
  );
}
