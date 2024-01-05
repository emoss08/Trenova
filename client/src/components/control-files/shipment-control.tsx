/*
 * COPYRIGHT(c) 2024 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { Button } from "@/components/ui/button";
import { useShipmentControl } from "@/hooks/useQueries";
import React from "react";
import { useForm } from "react-hook-form";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { CheckboxInput } from "@/components/common/fields/checkbox";
import { yupResolver } from "@hookform/resolvers/yup";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorLoadingData } from "@/components/common/table/data-table-components";
import { shipmentControlSchema } from "@/lib/validations/OrderSchema";
import {
  ShipmentControl as ShipmentControlType,
  ShipmentControlFormValues,
} from "@/types/order";

function ShipmentControlForm({
  shipmentControl,
}: {
  shipmentControl: ShipmentControlType;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, handleSubmit, reset } = useForm<ShipmentControlFormValues>({
    resolver: yupResolver(shipmentControlSchema),
    defaultValues: {
      autoRateShipment: shipmentControl.autoRateShipment,
      calculateDistance: shipmentControl.calculateDistance,
      enforceRevCode: shipmentControl.enforceRevCode,
      enforceVoidedComm: shipmentControl.enforceVoidedComm,
      generateRoutes: shipmentControl.generateRoutes,
      enforceCommodity: shipmentControl.enforceCommodity,
      autoSequenceStops: shipmentControl.autoSequenceStops,
      autoShipmentTotal: shipmentControl.autoShipmentTotal,
      enforceOriginDestination: shipmentControl.enforceOriginDestination,
      checkForDuplicateBol: shipmentControl.checkForDuplicateBol,
      removeShipment: shipmentControl.removeShipment,
    },
  });

  const mutation = useCustomMutation<ShipmentControlFormValues>(
    control,
    {
      method: "PUT",
      path: `/shipment_control/${shipmentControl.id}/`,
      successMessage: "Shipment Control updated successfully.",
      queryKeysToInvalidate: ["shipmentControl"],
      errorMessage: "Failed to update shipment control.",
    },
    () => setIsSubmitting(false),
  );

  const onSubmit = (values: ShipmentControlFormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);

    reset(values);
  };

  return (
    <form
      className="m-4 bg-background ring-1 ring-muted sm:rounded-xl md:col-span-2"
      onSubmit={handleSubmit(onSubmit)}
    >
      <div className="px-4 py-6 sm:p-8">
        <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-6">
          <div className="col-span-3">
            <CheckboxInput
              name="autoRateShipment"
              control={control}
              label="Auto Rate Shipment"
              description="Automate the rating of shipments based on pre-established contractual rates."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="calculateDistance"
              control={control}
              label="Calculate Distance"
              description="Enable automatic calculation of shipment distances for accurate logistics planning."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="enforceRevCode"
              control={control}
              label="Enforce Revenue code"
              description="Mandate the use of specific revenue codes for standardized shipment billing."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="enforceVoidedComm"
              control={control}
              label="Enforce Voided Comm"
              description="Implement checks against using voided commodity codes in shipment processing."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="generateRoutes"
              control={control}
              label="Generate Routes"
              description="Automatically generate optimal routes for each shipment to enhance efficiency."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="enforceCommodity"
              control={control}
              label="Enforce Commodity"
              description="Require the use of commodity codes for all shipments for consistent categorization."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="autoSequenceStops"
              control={control}
              label="Auto Sequence Stops"
              description="Automatically organize the sequence of stops in a shipment for optimal routing."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="autoShipmentTotal"
              control={control}
              label="Auto Calc. Shipment Total"
              description="Automatically calculate the total charge for shipments, ensuring billing accuracy."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="enforceOriginDestination"
              control={control}
              label="Compare Origin/Destination"
              description="Validate that the origin and destination of each shipment are distinct."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="checkForDuplicateBol"
              control={control}
              label="Check for Duplicate BOL"
              description="Check for and prevent duplicate Bill of Lading (BOL) numbers in the system."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="removeShipment"
              control={control}
              label="Remove Shipment"
              description="Grant the ability to remove shipments from the system, with restrictions on active movements and stops."
            />
          </div>
        </div>
      </div>
      <div className="flex items-center justify-end gap-x-6 border-t border-muted p-4 sm:px-8">
        <Button
          onClick={(e) => {
            e.preventDefault();
            reset();
          }}
          type="button"
          variant="ghost"
          disabled={isSubmitting}
        >
          Cancel
        </Button>
        <Button type="submit" isLoading={isSubmitting}>
          Save
        </Button>
      </div>
    </form>
  );
}

export default function ShipmentControl() {
  const { shipmentControlData, isLoading, isError } = useShipmentControl();

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
      <div className="px-4 sm:px-0">
        <h2 className="text-base font-semibold leading-7 text-foreground">
          Shipment Control
        </h2>
        <p className="mt-1 text-sm leading-6 text-muted-foreground">
          Revolutionize your shipment operations with our Shipment Management
          System. This module is built to streamline every aspect of shipment
          control, from routing to compliance enforcement, ensuring efficient
          and reliable management of your transport operations.
        </p>
      </div>
      {isLoading ? (
        <div className="m-4 bg-background ring-1 ring-muted sm:rounded-xl md:col-span-2">
          <Skeleton className="h-screen w-full" />
        </div>
      ) : isError ? (
        <div className="m-4 bg-background p-8 ring-1 ring-muted sm:rounded-xl md:col-span-2">
          <ErrorLoadingData message="Failed to load dispatch control." />
        </div>
      ) : (
        shipmentControlData && (
          <ShipmentControlForm shipmentControl={shipmentControlData} />
        )
      )}
    </div>
  );
}
