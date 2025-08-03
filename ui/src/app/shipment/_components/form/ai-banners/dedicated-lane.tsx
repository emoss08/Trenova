/* eslint-disable react/display-name */
/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { queries } from "@/lib/queries";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { ShipmentLocations } from "@/lib/shipment/utils";
import { useQuery } from "@tanstack/react-query";
import { memo, useMemo } from "react";
import { useFormContext, useWatch } from "react-hook-form";

export const DedicatedLaneBanner = memo(() => {
  const { control } = useFormContext<ShipmentSchema>();

  // Only watch specific fields needed for the dedicated lane query
  const customerId = useWatch({ control, name: "customerId" });
  const serviceTypeId = useWatch({ control, name: "serviceTypeId" });
  const shipmentTypeId = useWatch({ control, name: "shipmentTypeId" });
  const tractorTypeId = useWatch({ control, name: "tractorTypeId" });
  const trailerTypeId = useWatch({ control, name: "trailerTypeId" });
  const moves = useWatch({ control, name: "moves" });

  // Create a minimal shipment object with only the fields we need
  const shipmentForLocations = useMemo(
    () =>
      ({
        moves: moves || [],
      }) as ShipmentSchema,
    [moves],
  );

  const formValuesLocations =
    ShipmentLocations.useLocations(shipmentForLocations);
  const { destination, origin } = formValuesLocations;

  const hasRequiredFields = Boolean(
    customerId &&
      serviceTypeId &&
      shipmentTypeId &&
      origin?.id &&
      destination?.id,
  );

  const { data: dedicatedLane } = useQuery({
    ...queries.dedicatedLane.getByShipment({
      customerId: customerId || "",
      serviceTypeId: serviceTypeId || "",
      shipmentTypeId: shipmentTypeId || "",
      originLocationId: origin?.id || null,
      destinationLocationId: destination?.id || null,
      tractorTypeId: tractorTypeId || null,
      trailerTypeId: trailerTypeId || null,
    }),
    enabled: hasRequiredFields,
  });

  if (!dedicatedLane) {
    return null;
  }

  return (
    <div className="flex bg-purple-500/10 border border-purple-600/50 p-3 rounded-lg justify-between items-center w-full selection:bg-purple-500/50">
      <div className="flex items-center gap-4 w-full text-purple-600">
        <div className="flex flex-col">
          <p className="text-sm font-semibold">Contractual Dedicated Lane</p>
          <p className="text-xs dark:text-purple-200 text-purple-700">
            {dedicatedLane.autoAssign ? (
              <>
                This shipment is associated with a contractual dedicated lane
                agreement and will be automatically assigned to the designated
                worker.
                <br />
                <a
                  target="_blank"
                  href={`/shipments/configurations/dedicated-lanes?entityId=${dedicatedLane.id}&modalType=edit`}
                  className="text-purple-600 underline hover:text-purple-700 transition-colors mt-1 inline-block"
                  rel="noreferrer"
                >
                  View Dedicated Lane Details
                </a>
              </>
            ) : (
              <>
                This shipment matches a contractual dedicated lane agreement but
                automatic assignment is currently disabled. Enable
                auto-assignment to ensure proper worker allocation according to
                the contract terms.
              </>
            )}
          </p>
        </div>
      </div>
    </div>
  );
});
