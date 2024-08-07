/**
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
    defaultValues: shipmentControl,
  });
  const mutation = useCustomMutation<ShipmentControlFormValues>(control, {
    method: "PUT",
    path: "/shipment-control/",
    successMessage: t("formSuccessMessage"),
    queryKeysToInvalidate: "shipmentControl",
    reset,
    errorMessage: t("formErrorMessage"),
    onSettled: (response) => {
      reset(response?.data);
    },
  });

  const onSubmit = (values: ShipmentControlFormValues) => {
    mutation.mutate(values);
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
              name="autoTotalShipment"
              control={control}
              label={t("fields.autoShipmentTotal.label")}
              description={t("fields.autoShipmentTotal.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="compareOriginDestination"
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
    <div className="grid grid-cols-1 gap-8 md:grid-cols-3 xl:grid-cols-4">
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
