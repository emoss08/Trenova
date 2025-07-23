/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

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
