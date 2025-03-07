import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import { lazy, memo } from "react";

const ShipmentControlForm = lazy(
  () => import("./_components/shipment-control-form"),
);

export function ShipmentControl() {
  return (
    <div className="flex flex-col space-y-6">
      <MetaTags title="Shipment Control" description="Shipment Control" />
      <Header />
      <SuspenseLoader>
        <FormSaveProvider>
          <ShipmentControlForm />
        </FormSaveProvider>
      </SuspenseLoader>
    </div>
  );
}

const Header = memo(() => {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Shipment Control</h1>
        <p className="text-muted-foreground">
          Configure and manage your shipment control settings
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";
