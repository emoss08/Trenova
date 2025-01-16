import { MetaTags } from "@/components/meta-tags";
import ShipmentTypesDataTable from "./_components/shipment-type-table";

export function ShipmentTypes() {
  return (
    <>
      <MetaTags title="Shipment Types" description="Shipment Types" />
      <ShipmentTypesDataTable />
    </>
  );
}
