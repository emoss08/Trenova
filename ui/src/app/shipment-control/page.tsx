import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import { lazy } from "react";

const ShipmentControlForm = lazy(
  () => import("./_components/shipment-control-form"),
);

export function ShipmentControl() {
  return (
    <>
      <MetaTags title="Shipment Control" description="Shipment Control" />
      <SuspenseLoader>
        <FormSaveProvider>
          <ShipmentControlForm />
        </FormSaveProvider>
      </SuspenseLoader>
    </>
  );
}
