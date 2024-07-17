/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { CheckboxInput } from "@/components/common/fields/checkbox";
import { ErrorLoadingData } from "@/components/common/table/data-table-components";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useShipmentControl } from "@/hooks/useQueries";
import { shipmentControlSchema } from "@/lib/validations/ShipmentSchema";
import type {
  ShipmentControlFormValues,
  ShipmentControl as ShipmentControlType,
} from "@/types/shipment";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { ComponentLoader } from "./ui/component-loader";

function ShipmentControlForm({
  shipmentControl,
}: {
  shipmentControl: ShipmentControlType;
}) {
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
      sendPlacardInfo: shipmentControl.sendPlacardInfo,
      enforceHazmatSegRules: shipmentControl.enforceHazmatSegRules,
    },
  });

  const mutation = useCustomMutation<ShipmentControlFormValues>(control, {
    method: "PUT",
    path: `/shipment-control/${shipmentControl.id}/`,
    successMessage: t("formSuccessMessage"),
    queryKeysToInvalidate: "shipmentControl",
    reset,
    errorMessage: t("formErrorMessage"),
  });

  const onSubmit = (values: ShipmentControlFormValues) => {
    mutation.mutate(values);
    reset(values);
  };

  return (
    <form
      className="border-border bg-card m-4 border sm:rounded-xl md:col-span-2"
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
              name="sendPlacardInfo"
              control={control}
              label={t("fields.sendPlacardInfo.label")}
              description={t("fields.sendPlacardInfo.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="enforceHazmatSegRules"
              control={control}
              label={t("fields.enforceHazmatSegRules.label")}
              description={t("fields.enforceHazmatSegRules.description")}
            />
          </div>
        </div>
      </div>
      <div className="border-muted flex items-center justify-end gap-x-4 border-t p-4 sm:px-8">
        <Button
          onClick={(e) => {
            e.preventDefault();
            reset();
          }}
          type="button"
          variant="ghost"
          disabled={mutation.isPending}
        >
          {t("buttons.cancel", { ns: "common" })}
        </Button>
        <Button type="submit" isLoading={mutation.isPending}>
          {t("buttons.save", { ns: "common" })}
        </Button>
      </div>
    </form>
  );
}

export default function ShipmentControl() {
  const { data, isLoading, isError } = useShipmentControl();

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
      <div className="px-4 sm:px-0">
        <h2 className="text-foreground text-base font-semibold leading-7">
          Shipment Control
        </h2>
        <p className="text-muted-foreground mt-1 text-sm leading-6">
          Revolutionize your shipment operations with our Shipment Management
          System. This module is built to streamline every aspect of shipment
          control, from routing to compliance enforcement, ensuring efficient
          and reliable management of your transport operations.
        </p>
      </div>
      {isLoading ? (
        <div className="bg-background ring-muted m-4 ring-1 sm:rounded-xl md:col-span-2">
          <ComponentLoader className="h-[30em]" />
        </div>
      ) : isError ? (
        <ErrorLoadingData />
      ) : (
        data && <ShipmentControlForm shipmentControl={data} />
      )}
    </div>
  );
}
