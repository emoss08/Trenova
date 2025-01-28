import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import ShipmentTypesDataTable from "./_components/shipment-type-table";

export function ShipmentTypes() {
  return (
    <>
      <MetaTags title="Shipment Types" description="Shipment Types" />
      <SuspenseLoader>
        <ShipmentTypesDataTable />
      </SuspenseLoader>
    </>
  );
}
