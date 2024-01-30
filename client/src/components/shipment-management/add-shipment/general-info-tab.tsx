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
import {
  useCommodities,
  useEquipmentTypes,
  useHazardousMaterial,
  useLocations,
  useRevenueCodes,
  useShipmentTypes,
} from "@/hooks/useQueries";
import { statusChoices } from "@/lib/choices";
import { Commodity } from "@/types/commodities";
import { useTranslation } from "react-i18next";

function DispatchDetails({ control, setValue, watch }: any) {
  const { t } = useTranslation(["shipment.addshipment", "common"]);

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

  const commodityValue = watch("commodity");

  if (commodityValue && hazardousMaterials) {
    const selectedCommodity = (commodities as Commodity[]).find(
      (commodity) => commodity.id === commodityValue,
    );

    if (selectedCommodity?.hazardousMaterial) {
      setValue("hazardousMaterial", selectedCommodity?.hazardousMaterial);
    }
  }

  return (
    <div className="grid grid-cols-1 gap-x-6 gap-y-4 md:grid-cols-2">
      <div className="col-span-1">
        <SelectInput
          name="equipmentType"
          control={control}
          options={selectEquipmentType}
          isLoading={isEquipmentTypesLoading}
          isFetchError={isEquipmentTypeError}
          label={t("fields.equipmentType.label")}
          placeholder={t("fields.equipmentType.placeholder")}
          description={t("fields.equipmentType.description")}
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
          name="bolNumber"
          control={control}
          rules={{ required: true }}
          label={t("fields.bolNumber.label")}
          placeholder={t("fields.bolNumber.placeholder")}
          description={t("fields.bolNumber.description")}
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
    </div>
  );
}

function DestinationCards({ control }: any) {
  const { t } = useTranslation(["shipment.addshipment", "common"]);

  const {
    selectLocationData,
    isError: isLocationError,
    isLoading: isLocationsLoading,
  } = useLocations();

  return (
    <div className="my-7 flex space-x-10">
      <div className="flex-1">
        <div className="flex flex-col">
          <div className="border-border rounded-md border">
            <div className="border-border border-b p-2">
              <h2 className="text-center text-lg font-semibold">
                {t("titles.origin.label")}
              </h2>
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
            <div className="border-border border-b p-2">
              <h2 className="text-center text-lg font-semibold">
                {t("titles.destination.label")}
              </h2>
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
                    label={t("fields.destinationAppointmentWindowStart.label")}
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
  );
}

export function ShipmentGeneralForm({ control, setValue, watch }: any) {
  const { t } = useTranslation(["shipment.addshipment", "common"]);

  const { selectRevenueCodes, isRevenueCodeLoading, isRevenueCodeError } =
    useRevenueCodes();

  const {
    selectShipmentType,
    isError: isShipmentTypeError,
    isLoading: isShipmentTypesLoading,
  } = useShipmentTypes();

  return (
    <div className="sm:p8 px-4 py-6">
      <div className="grid max-w-[1000px] grid-cols-1 gap-x-6 gap-y-8 lg:grid-cols-12">
        <div className="col-span-3">
          <SelectInput
            name="status"
            control={control}
            options={statusChoices}
            rules={{ required: true }}
            label={t("fields.status.label")}
            placeholder={t("fields.status.placeholder")}
            description={t("fields.status.description")}
          />
        </div>
        <div className="col-span-3">
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
        <div className="col-span-3">
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
      <DestinationCards control={control} />
      <DispatchDetails control={control} setValue={setValue} watch={watch} />
    </div>
  );
}
