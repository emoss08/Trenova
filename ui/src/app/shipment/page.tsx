import { QueryLazyComponent } from "@/components/error-boundary";
import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { lazy, memo } from "react";

const ShipmentTable = lazy(() => import("./_components/shipment-table"));
const ShipmentAnalytics = lazy(
  () => import("./_components/analytics/shipment-analytics"),
);
export function Shipment() {
  return (
    <FormSaveProvider>
      <div className="space-y-6 p-6">
        <MetaTags title="Shipments" description="Shipments" />
        <Header />
        <QueryLazyComponent queryKey={["shipment-list"]}>
          {/* <ShipmentAnalytics /> */}
          <ShipmentTable />
        </QueryLazyComponent>
      </div>
    </FormSaveProvider>
  );
}

const Header = memo(() => {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Shipments</h1>
        <p className="text-muted-foreground">
          Manage and track all shipments in your system
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";
