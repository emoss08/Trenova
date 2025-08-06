/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { ShipmentDetailsHeader } from "./shipment-details-header";

export function ShipmentFormContent({
  children,
  selectedShipment,
}: {
  children: React.ReactNode;
  selectedShipment?: ShipmentSchema | null;
}) {
  return (
    <>
      <ShipmentDetailsHeader selectedShipment={selectedShipment} />
      {children}
    </>
  );
}
