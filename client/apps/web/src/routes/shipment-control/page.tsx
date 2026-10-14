import { useT } from "@trenova/shared/i18n/use-t";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { ShipmentControlSkeleton } from "./skeleton";

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
      <LazyComponent fallback={<ShipmentControlSkeleton />}>
        <ShipmentControlForm />
      </LazyComponent>
    </PageLayout>
  );
}
