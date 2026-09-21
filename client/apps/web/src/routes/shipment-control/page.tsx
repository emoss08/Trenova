import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const ShipmentControlForm = lazy(() => import("./_components/shipment-control-form"));

export function ShipmentControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Shipment control"),
        description: t("Configure and manage your shipment control settings"),
      }}
    >
      <SuspenseLoader>
        <ShipmentControlForm />
      </SuspenseLoader>
    </PageLayout>
  );
}
