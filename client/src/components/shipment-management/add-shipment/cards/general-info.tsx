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

import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { Skeleton } from "@/components/ui/skeleton";
import { TitleWithTooltip } from "@/components/ui/title-with-tooltip";
import {
  useNextProNumber,
  useRevenueCodes,
  useShipmentTypes,
} from "@/hooks/useQueries";
import { shipmentSourceChoices, shipmentStatusChoices } from "@/lib/choices";
import { ShipmentFormValues } from "@/types/order";
import { useEffect } from "react";
import { Control, UseFormSetValue } from "react-hook-form";
import { useTranslation } from "react-i18next";

export function GeneralInformation({
  control,
  setValue,
}: {
  control: Control<ShipmentFormValues>;
  setValue: UseFormSetValue<ShipmentFormValues>;
}) {
  const { t } = useTranslation(["shipment.addshipment", "common"]);

  const { selectRevenueCodes, isRevenueCodeLoading, isRevenueCodeError } =
    useRevenueCodes();

  const {
    selectShipmentType,
    isError: isShipmentTypeError,
    isLoading: isShipmentTypesLoading,
  } = useShipmentTypes();

  const { proNumber, isProNumberLoading } = useNextProNumber();

  useEffect(() => {
    if (proNumber) {
      setValue("proNumber", proNumber as string);
    }
  }, [proNumber, setValue]);

  return (
    <div className="border-border bg-card rounded-md border">
      <div className="border-border bg-background flex justify-center rounded-t-md border-b p-2">
        <TitleWithTooltip
          title={t("card.generalInfo.label")}
          tooltip={t("card.generalInfo.description")}
        />
      </div>
      <div className="grid grid-cols-1 gap-x-6 gap-y-4 p-4 lg:grid-cols-4">
        <div className="col-span-1">
          <SelectInput
            name="status"
            control={control}
            options={shipmentStatusChoices}
            rules={{ required: true }}
            isReadOnly
            label={t("fields.status.label")}
            placeholder={t("fields.status.placeholder")}
            description={t("fields.status.description")}
          />
        </div>
        <div className="col-span-1">
          {isProNumberLoading ? (
            <Skeleton className="mt-6 h-9 w-60" />
          ) : (
            <InputField
              name="proNumber"
              control={control}
              rules={{ required: true }}
              readOnly
              label={t("fields.proNumber.label")}
              placeholder={t("fields.proNumber.placeholder")}
              description={t("fields.proNumber.description")}
            />
          )}
        </div>
        <div className="col-span-1">
          <SelectInput
            name="revenueCode"
            control={control}
            options={selectRevenueCodes}
            isLoading={isRevenueCodeLoading}
            isFetchError={isRevenueCodeError}
            label={t("fields.revenueCode.label")}
            placeholder={t("fields.revenueCode.placeholder")}
            description={t("fields.revenueCode.description")}
            hasPopoutWindow
            popoutLink="/accounting/revenue-codes"
            isClearable
            popoutLinkLabel="Revenue Code"
          />
        </div>
        <div className="col-span-1">
          <SelectInput
            name="shipmentType"
            control={control}
            options={selectShipmentType}
            isLoading={isShipmentTypesLoading}
            isFetchError={isShipmentTypeError}
            rules={{ required: true }}
            label={t("fields.shipmentType.label")}
            placeholder={t("fields.shipmentType.placeholder")}
            description={t("fields.shipmentType.description")}
            hasPopoutWindow
            popoutLink="/shipment-management/shipment-types/"
            isClearable
            popoutLinkLabel="Shipment Type"
          />
        </div>
        <div className="col-span-1">
          <SelectInput
            name="serviceType"
            menuPlacement="top"
            control={control}
            options={shipmentSourceChoices}
            rules={{ required: true }}
            label={t("fields.serviceType.label")}
            placeholder={t("fields.serviceType.placeholder")}
            description={t("fields.serviceType.description")}
          />
        </div>
      </div>
    </div>
  );
}
