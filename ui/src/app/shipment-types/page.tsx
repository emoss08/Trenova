import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const ShipmentTypesDataTable = lazy(
  () => import("./_components/shipment-type-table"),
);

export function ShipmentTypes() {
  return (
    <>
      <MetaTags title="Shipment Types" description="Shipment Types" />
      <LazyComponent>
        <ShipmentTypesDataTable />
      </LazyComponent>
    </>
  );
}
