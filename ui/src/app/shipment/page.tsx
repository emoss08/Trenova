import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import ShipmentTable from "./_components/shipment-table";

export function Shipment() {
  return (
    <>
      <MetaTags title="Shipments" description="Shipments" />
      <SuspenseLoader>
        <ShipmentTable />
      </SuspenseLoader>
    </>
  );
}
