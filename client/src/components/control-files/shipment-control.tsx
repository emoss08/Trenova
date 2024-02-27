/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

import { CheckboxInput } from "@/components/common/fields/checkbox";
import { ErrorLoadingData } from "@/components/common/table/data-table-components";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useShipmentControl } from "@/hooks/useQueries";
import { shipmentControlSchema } from "@/lib/validations/ShipmentSchema";
import {
  ShipmentControl as ShipmentControlType,
  ShipmentControlFormValues,
} from "@/types/shipment";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";

function ShipmentControlForm({
  shipmentControl,
}: {
  shipmentControl: ShipmentControlType;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { t } = useTranslation(["admin.shipmentcontrol", "common"]); // Use the translation hook

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
      successMessage: t("formSuccessMessage"),
      queryKeysToInvalidate: ["shipmentControl"],
      errorMessage: t("formErrorMessage"),
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
      className="m-4 border border-border bg-card sm:rounded-xl md:col-span-2"
      onSubmit={handleSubmit(onSubmit)}
    >
      <div className="px-4 py-6 sm:p-8">
        <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-6">
          <div className="col-span-3">
            <CheckboxInput
              name="autoRateShipment"
              control={control}
              label={t("fields.autoRateShipment.label")}
              description={t("fields.autoRateShipment.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="calculateDistance"
              control={control}
              label={t("fields.calculateDistance.label")}
              description={t("fields.calculateDistance.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="enforceRevCode"
              control={control}
              label={t("fields.enforceRevCode.label")}
              description={t("fields.enforceRevCode.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="enforceVoidedComm"
              control={control}
              label={t("fields.enforceVoidedComm.label")}
              description={t("fields.enforceVoidedComm.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="generateRoutes"
              control={control}
              label={t("fields.generateRoutes.label")}
              description={t("fields.generateRoutes.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="enforceCommodity"
              control={control}
              label={t("fields.enforceCommodity.label")}
              description={t("fields.enforceCommodity.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="autoSequenceStops"
              control={control}
              label={t("fields.autoSequenceStops.label")}
              description={t("fields.autoSequenceStops.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="autoShipmentTotal"
              control={control}
              label={t("fields.autoShipmentTotal.label")}
              description={t("fields.autoShipmentTotal.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="enforceOriginDestination"
              control={control}
              label={t("fields.enforceOriginDestination.label")}
              description={t("fields.enforceOriginDestination.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="checkForDuplicateBol"
              control={control}
              label={t("fields.checkForDuplicateBol.label")}
              description={t("fields.checkForDuplicateBol.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="removeShipment"
              control={control}
              label={t("fields.removeShipment.label")}
              description={t("fields.removeShipment.description")}
            />
          </div>
        </div>
      </div>
      <div className="flex items-center justify-end gap-x-4 border-t border-muted p-4 sm:px-8">
        <Button
          onClick={(e) => {
            e.preventDefault();
            reset();
          }}
          type="button"
          variant="ghost"
          disabled={isSubmitting}
        >
          {t("buttons.cancel", { ns: "common" })}
        </Button>
        <Button type="submit" isLoading={isSubmitting}>
          {t("buttons.save", { ns: "common" })}
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
