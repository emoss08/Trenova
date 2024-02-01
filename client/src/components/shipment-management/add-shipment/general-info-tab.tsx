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

import { InputField, TimeField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { Skeleton } from "@/components/ui/skeleton";
import { TitleWithTooltip } from "@/components/ui/title-with-tooltip";
import {
  getLatestProNumber,
  useCommodities,
  useEquipmentTypes,
  useHazardousMaterial,
  useLocations,
  useRevenueCodes,
  useShipmentTypes,
  useTrailers,
  useUsers,
} from "@/hooks/useQueries";
import { shipmentSourceChoices, shipmentStatusChoices } from "@/lib/choices";
import { useUserStore } from "@/stores/AuthStore";
import { Commodity } from "@/types/commodities";
import { Trailer } from "@/types/equipment";
import { Location } from "@/types/location";
import { ShipmentFormProps } from "@/types/order";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";

function DispatchDetails({ control, setValue, watch }: ShipmentFormProps) {
  const { t } = useTranslation(["shipment.addshipment", "common"]);
  const [user] = useUserStore.use("user");

  const {
    selectEquipmentType,
    isError: isEquipmentTypeError,
    isLoading: isEquipmentTypesLoading,
  } = useEquipmentTypes();

  const {
    selectCommodityData,
    data: commodities,
    isError: isCommodityError,
    isLoading: isCommoditiesLoading,
  } = useCommodities();

  const {
    selectHazardousMaterials,
    data: hazardousMaterials,
    isError: isHazmatError,
    isLoading: isHazmatsLoading,
  } = useHazardousMaterial();

  const { selectTrailers, isTrailerError, isTrailerLoading, trailerData } =
    useTrailers();

  const {
    selectUsersData,
    isError: isUserError,
    isLoading: isUsersLoading,
  } = useUsers();

  const commodityValue = watch("commodity");
  const trailerValue = watch("trailer");

  // Use useEffect to respond to changes in originLocation and destinationLocation
  useEffect(() => {
    if (commodityValue && hazardousMaterials) {
      const selectedCommodity = (commodities as Commodity[]).find(
        (commodity) => commodity.id === commodityValue,
      );

      if (selectedCommodity?.hazardousMaterial) {
        setValue("hazardousMaterial", selectedCommodity?.hazardousMaterial);
      }
    }

    if (trailerValue) {
      const selectedTrailer = (trailerData as Trailer[]).find(
        (trailer: any) => trailer.id === trailerValue,
      );

      if (selectedTrailer?.equipmentType) {
        setValue("trailerType", selectedTrailer?.equipmentType);
      }
    }

    if (user) {
      setValue("enteredBy", user.id);
    }
  }, [
    commodityValue,
    hazardousMaterials,
    commodities,
    setValue,
    trailerValue,
    selectTrailers,
    trailerData,
    user,
  ]);

  return (
    <div className="border-border bg-card rounded-md border">
      <div className="border-border rounded-md border-b">
        <div className="border-border bg-background flex justify-center rounded-t-md border-b p-2">
          <TitleWithTooltip
            title={t("card.additionalInfo.label")}
            tooltip={t("card.additionalInfo.description")}
          />
        </div>
        <div className="grid grid-cols-1 gap-x-6 gap-y-4 p-4 md:grid-cols-2">
          <div className="col-span-1">
            <SelectInput
              name="trailer"
              control={control}
              options={selectTrailers}
              rules={{ required: true }}
              isLoading={isTrailerLoading}
              isFetchError={isTrailerError}
              label={t("fields.trailer.label")}
              placeholder={t("fields.trailer.placeholder")}
              description={t("fields.trailer.description")}
              hasPopoutWindow
              popoutLink="/equipment/trailer/"
              popoutLinkLabel="Trailer"
            />
          </div>
          <div className="col-span-1">
            <SelectInput
              name="trailerType"
              control={control}
              options={selectEquipmentType}
              rules={{ required: true }}
              isLoading={isEquipmentTypesLoading}
              isFetchError={isEquipmentTypeError}
              label={t("fields.trailerType.label")}
              placeholder={t("fields.trailerType.placeholder")}
              description={t("fields.trailerType.description")}
              hasPopoutWindow
              popoutLink="/equipment/equipment-types/"
              popoutLinkLabel="Equipment Type"
            />
          </div>
          <div className="col-span-1">
            <SelectInput
              name="tractorType"
              control={control}
              options={selectEquipmentType}
              isLoading={isEquipmentTypesLoading}
              isFetchError={isEquipmentTypeError}
              label={t("fields.tractorType.label")}
              placeholder={t("fields.tractorType.placeholder")}
              description={t("fields.tractorType.description")}
              hasPopoutWindow
              popoutLink="/equipment/equipment-types/"
              popoutLinkLabel="Equipment Type"
            />
          </div>
          <div className="col-span-1">
            <SelectInput
              name="commodity"
              control={control}
              options={selectCommodityData}
              isLoading={isCommoditiesLoading}
              isFetchError={isCommodityError}
              rules={{ required: true }}
              label={t("fields.commodity.label")}
              placeholder={t("fields.commodity.placeholder")}
              description={t("fields.commodity.description")}
              hasPopoutWindow
              popoutLink="/shipment-management/commodity-codes/"
              isClearable
              popoutLinkLabel="Commodity Code"
            />
          </div>
          <div className="col-span-1">
            <SelectInput
              name="hazardousMaterial"
              control={control}
              options={selectHazardousMaterials}
              isLoading={isHazmatsLoading}
              isFetchError={isHazmatError}
              label={t("fields.hazardousMaterial.label")}
              placeholder={t("fields.hazardousMaterial.placeholder")}
              description={t("fields.hazardousMaterial.description")}
              hasPopoutWindow
              popoutLink="/shipment-management/hazardous-materials/"
              isClearable
              popoutLinkLabel="Hazardous Material"
            />
          </div>

          <div className="col-span-1">
            <InputField
              name="temperatureMin"
              control={control}
              label={t("fields.temperatureMin.label")}
              placeholder={t("fields.temperatureMin.placeholder")}
              description={t("fields.temperatureMin.description")}
            />
          </div>
          <div className="col-span-1">
            <InputField
              name="temperatureMax"
              control={control}
              label={t("fields.temperatureMax.label")}
              placeholder={t("fields.temperatureMax.placeholder")}
              description={t("fields.temperatureMax.description")}
            />
          </div>
          <div className="col-span-1">
            <InputField
              name="consigneeRefNumber"
              control={control}
              label={t("fields.consigneeRefNumber.label")}
              placeholder={t("fields.consigneeRefNumber.placeholder")}
              description={t("fields.consigneeRefNumber.description")}
            />
          </div>
          <div className="col-span-1">
            <InputField
              name="bolNumber"
              control={control}
              rules={{ required: true }}
              label={t("fields.bolNumber.label")}
              placeholder={t("fields.bolNumber.placeholder")}
              description={t("fields.bolNumber.description")}
            />
          </div>
          <div className="col-span-1">
            <SelectInput
              name="enteredBy"
              options={selectUsersData}
              isLoading={isUsersLoading}
              isFetchError={isUserError}
              control={control}
              isReadOnly
              rules={{ required: true }}
              label={t("fields.enteredBy.label")}
              placeholder={t("fields.enteredBy.placeholder")}
              description={t("fields.enteredBy.description")}
            />
          </div>
          <div className="col-span-1">
            <SelectInput
              name="entryMethod"
              control={control}
              options={shipmentSourceChoices}
              isReadOnly
              rules={{ required: true }}
              label={t("fields.entryMethod.label")}
              placeholder={t("fields.entryMethod.placeholder")}
              description={t("fields.entryMethod.description")}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

function DestinationCards({ control, watch, setValue }: ShipmentFormProps) {
  const { t } = useTranslation(["shipment.addshipment", "common"]);

  const {
    selectLocationData,
    locations,
    isError: isLocationError,
    isLoading: isLocationsLoading,
  } = useLocations();

  const originLocation = watch("originLocation");
  const destinationLocation = watch("destinationLocation");

  // Use useEffect to respond to changes in originLocation and destinationLocation
  useEffect(() => {
    if (originLocation) {
      const selectedLocation = (locations as Location[]).find(
        (location) => location.id === originLocation,
      );

      console.info("selectedLocation", selectedLocation);

      if (selectedLocation) {
        setValue(
          "originAddress",
          `${selectedLocation.addressLine1}, ${selectedLocation.city}, ${selectedLocation.state} ${selectedLocation.zipCode}`,
        );
      }
    }

    if (destinationLocation) {
      const selectedDestinationLocation = (locations as Location[]).find(
        (location) => location.id === destinationLocation,
      );

      if (selectedDestinationLocation) {
        setValue(
          "destinationAddress",
          `${selectedDestinationLocation.addressLine1}, ${selectedDestinationLocation.city}, ${selectedDestinationLocation.state} ${selectedDestinationLocation.zipCode}`,
        );
      }
    }
  }, [originLocation, destinationLocation, locations, setValue]);

  return (
    <div className="border-border bg-card rounded-md border p-4">
      <div className="flex space-x-10">
        <div className="flex-1">
          <div className="flex flex-col">
            <div className="border-border rounded-md border">
              <div className="border-border bg-background flex justify-center rounded-t-md border-b p-2">
                <TitleWithTooltip
                  title={t("card.origin.label")}
                  tooltip={t("card.origin.description")}
                />
              </div>
              <div className="grid grid-cols-1 gap-y-4 p-4">
                <div className="col-span-3">
                  <SelectInput
                    name="originLocation"
                    control={control}
                    options={selectLocationData}
                    isLoading={isLocationsLoading}
                    isFetchError={isLocationError}
                    label={t("fields.originLocation.label")}
                    placeholder={t("fields.originLocation.placeholder")}
                    description={t("fields.originLocation.description")}
                    hasPopoutWindow
                    popoutLink="/dispatch/locations/"
                    isClearable
                    popoutLinkLabel="Location"
                  />
                </div>
                <div className="col-span-3">
                  <InputField
                    control={control}
                    name="originAddress"
                    autoCapitalize="none"
                    autoCorrect="off"
                    type="text"
                    label={t("fields.originAddress.label")}
                    placeholder={t("fields.originAddress.placeholder")}
                    description={t("fields.originAddress.description")}
                  />
                </div>
                <div className="grid grid-cols-2 gap-x-4">
                  <div className="col-span-1">
                    <TimeField
                      control={control}
                      rules={{ required: true }}
                      name="originAppointmentWindowStart"
                      label={t("fields.originAppointmentWindowStart.label")}
                      placeholder={t(
                        "fields.originAppointmentWindowStart.placeholder",
                      )}
                      description={t(
                        "fields.originAppointmentWindowStart.description",
                      )}
                    />
                  </div>
                  <div className="col-span-1">
                    <TimeField
                      control={control}
                      rules={{ required: true }}
                      name="originAppointmentWindowEnd"
                      label={t("fields.originAppointmentWindowEnd.label")}
                      placeholder={t(
                        "fields.originAppointmentWindowEnd.placeholder",
                      )}
                      description={t(
                        "fields.originAppointmentWindowEnd.description",
                      )}
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div className="flex-1">
          <div className="flex flex-col">
            <div className="border-border rounded-md border">
              <div className="border-border bg-background flex justify-center rounded-t-md border-b p-2">
                <TitleWithTooltip
                  title={t("card.destination.label")}
                  tooltip={t("card.destination.description")}
                />
              </div>
              <div className="grid grid-cols-1 gap-y-4 p-4">
                <div className="col-span-3">
                  <SelectInput
                    name="destinationLocation"
                    control={control}
                    options={selectLocationData}
                    isLoading={isLocationsLoading}
                    isFetchError={isLocationError}
                    label={t("fields.destinationLocation.label")}
                    placeholder={t("fields.destinationLocation.placeholder")}
                    description={t("fields.destinationLocation.description")}
                    hasPopoutWindow
                    popoutLink="/dispatch/locations/"
                    isClearable
                    popoutLinkLabel="Location"
                  />
                </div>
                <div className="col-span-3">
                  <InputField
                    control={control}
                    name="destinationAddress"
                    autoCapitalize="none"
                    autoCorrect="off"
                    type="text"
                    label={t("fields.destinationAddress.label")}
                    placeholder={t("fields.destinationAddress.placeholder")}
                    description={t("fields.destinationAddress.description")}
                  />
                </div>
                <div className="grid grid-cols-2 gap-x-4">
                  <div className="col-span-1">
                    <TimeField
                      control={control}
                      rules={{ required: true }}
                      name="destinationAppointmentWindowStart"
                      label={t(
                        "fields.destinationAppointmentWindowStart.label",
                      )}
                      placeholder={t(
                        "fields.destinationAppointmentWindowStart.placeholder",
                      )}
                      description={t(
                        "fields.destinationAppointmentWindowStart.description",
                      )}
                    />
                  </div>
                  <div className="col-span-1">
                    <TimeField
                      control={control}
                      rules={{ required: true }}
                      name="destinationAppointmentWindowEnd"
                      label={t("fields.destinationAppointmentWindowEnd.label")}
                      placeholder={t(
                        "fields.destinationAppointmentWindowEnd.placeholder",
                      )}
                      description={t(
                        "fields.destinationAppointmentWindowEnd.description",
                      )}
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function DetailCard({ control, setValue }: ShipmentFormProps) {
  const { t } = useTranslation(["shipment.addshipment", "common"]);

  const { selectRevenueCodes, isRevenueCodeLoading, isRevenueCodeError } =
    useRevenueCodes();

  const {
    selectShipmentType,
    isError: isShipmentTypeError,
    isLoading: isShipmentTypesLoading,
  } = useShipmentTypes();

  const { proNumber, isProNumberLoading } = getLatestProNumber();

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
      <div className="grid grid-cols-1 gap-x-6 gap-y-8 p-4 lg:grid-cols-4">
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
      </div>
    </div>
  );
}

export function ShipmentGeneralForm({
  control,
  setValue,
  watch,
}: ShipmentFormProps) {
  return (
    <div className="grid grid-cols-1 gap-y-8">
      <DetailCard control={control} setValue={setValue} watch={watch} />
      <DestinationCards control={control} watch={watch} setValue={setValue} />
      <DispatchDetails control={control} setValue={setValue} watch={watch} />
    </div>
  );
}
