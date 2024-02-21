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
import {
  AsyncSelectInput,
  SelectInput,
} from "@/components/common/fields/select-input";
import { Skeleton } from "@/components/ui/skeleton";
import { TitleWithTooltip } from "@/components/ui/title-with-tooltip";
import { useServiceTypes } from "@/hooks/useQueries";
import { shipmentStatusChoices } from "@/lib/choices";
import { ShipmentControl, ShipmentFormValues } from "@/types/order";
import { useEffect } from "react";
import { useFormContext } from "react-hook-form";
import { useTranslation } from "react-i18next";

export default function GeneralInformation({
  shipmentControlData,
  isShipmentControlLoading,
  proNumber,
  isProNumberLoading,
}: {
  shipmentControlData: ShipmentControl;
  isShipmentControlLoading: boolean;
  proNumber: string | undefined;
  isProNumberLoading: boolean;
}) {
  const { t } = useTranslation("shipment.addshipment");
  const { control, setValue } = useFormContext<ShipmentFormValues>();
  const { selectServiceTypes, isServiceTypeError, isServiceTypeLoading } =
    useServiceTypes();

  useEffect(() => {
    if (proNumber) {
      setValue("proNumber", proNumber as string);
    }
  }, [proNumber, setValue]);

  if (isShipmentControlLoading || isProNumberLoading) {
    return <Skeleton className="h-[30vh]" />;
  }

  return (
    <div className="rounded-md border border-border bg-card">
      <div className="flex justify-center rounded-t-md border-b border-border bg-background p-2">
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
            label={t("card.generalInfo.fields.status.label")}
            placeholder={t("card.generalInfo.fields.status.placeholder")}
            description={t("card.generalInfo.fields.status.description")}
          />
        </div>
        <div className="col-span-1">
          <InputField
            name="proNumber"
            control={control}
            rules={{ required: true }}
            readOnly
            label={t("card.generalInfo.fields.proNumber.label")}
            placeholder={t("card.generalInfo.fields.proNumber.placeholder")}
            description={t("card.generalInfo.fields.proNumber.description")}
          />
        </div>
      </div>
      <div className="grid grid-cols-1 gap-x-6 gap-y-4 px-4 pb-4 lg:grid-cols-4">
        <div className="col-span-1">
          <AsyncSelectInput
            name="revenueCode"
            valueKey="code"
            link="revenue_codes"
            control={control}
            rules={{ required: shipmentControlData.enforceRevCode || false }}
            label={t("card.generalInfo.fields.revenueCode.label")}
            placeholder={t("card.generalInfo.fields.revenueCode.placeholder")}
            description={t("card.generalInfo.fields.revenueCode.description")}
            hasPopoutWindow
            popoutLink="/accounting/revenue-codes"
            isClearable
            popoutLinkLabel="Revenue Code"
          />
        </div>
        <div className="col-span-1">
          <AsyncSelectInput
            name="shipmentType"
            link="shipment_types"
            valueKey="code"
            control={control}
            rules={{ required: true }}
            label={t("card.generalInfo.fields.shipmentType.label")}
            placeholder={t("card.generalInfo.fields.shipmentType.placeholder")}
            description={t("card.generalInfo.fields.shipmentType.description")}
            hasPopoutWindow
            popoutLink="/shipment-management/shipment-types/"
            isClearable
            popoutLinkLabel="Shipment Type"
          />
        </div>
        <div className="col-span-1">
          <AsyncSelectInput
            name="serviceType"
            link="service_types"
            valueKey="code"
            control={control}
            options={selectServiceTypes}
            isFetchError={isServiceTypeError}
            isLoading={isServiceTypeLoading}
            rules={{ required: true }}
            label={t("card.generalInfo.fields.serviceType.label")}
            placeholder={t("card.generalInfo.fields.serviceType.placeholder")}
            description={t("card.generalInfo.fields.serviceType.description")}
            popoutLink="/shipment-management/service-types/"
            hasPopoutWindow
            popoutLinkLabel="Service Type"
          />
        </div>
      </div>
    </div>
  );
}
