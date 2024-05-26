import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { Skeleton } from "@/components/ui/skeleton";
import { TitleWithTooltip } from "@/components/ui/title-with-tooltip";
import {
  useRevenueCodes,
  useServiceTypes,
  useShipmentTypes,
} from "@/hooks/useQueries";
import { shipmentStatusChoices } from "@/lib/choices";
import { ShipmentControl, ShipmentFormValues } from "@/types/shipment";
import { useEffect } from "react";
import { useFormContext } from "react-hook-form";
import { useTranslation } from "react-i18next";

export function GeneralInformationCard({
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
  const { selectRevenueCodes, isRevenueCodeError, isRevenueCodeLoading } =
    useRevenueCodes();
  const {
    selectShipmentType,
    isLoading: isShipmentTypesLoading,
    isError: isShipmentTypeError,
  } = useShipmentTypes();

  useEffect(() => {
    if (proNumber) {
      setValue("proNumber", proNumber as string);
    }
  }, [proNumber, setValue]);

  if (isShipmentControlLoading || isProNumberLoading) {
    return <Skeleton className="h-[30vh]" />;
  }

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
          <SelectInput
            name="revenueCode"
            options={selectRevenueCodes}
            isFetchError={isRevenueCodeError}
            isLoading={isRevenueCodeLoading}
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
          <SelectInput
            name="shipmentType"
            options={selectShipmentType}
            isFetchError={isShipmentTypeError}
            isLoading={isShipmentTypesLoading}
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
          <SelectInput
            name="serviceType"
            control={control}
            options={selectServiceTypes}
            isFetchError={isServiceTypeError}
            isLoading={isServiceTypeLoading}
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
