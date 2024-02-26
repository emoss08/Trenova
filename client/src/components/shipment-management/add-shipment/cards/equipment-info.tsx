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
  useCommodities,
  useEquipmentTypes,
  useHazardousMaterial,
  useTrailers,
} from "@/hooks/useQueries";
import { Commodity } from "@/types/commodities";
import { Trailer } from "@/types/equipment";
import { ShipmentControl, ShipmentFormValues } from "@/types/shipment";
import { useEffect } from "react";
import { useFormContext } from "react-hook-form";
import { useTranslation } from "react-i18next";

export default function EquipmentInformation({
  shipmentControlData,
  isShipmentControlLoading,
}: {
  shipmentControlData: ShipmentControl;
  isShipmentControlLoading: boolean;
}) {
  const { t } = useTranslation("shipment.addshipment");
  const { data: commodities } = useCommodities();
  const { trailerData, isTrailerError, isTrailerLoading, selectTrailers } =
    useTrailers();
  const { control, setValue, watch } = useFormContext<ShipmentFormValues>();

  const {
    selectEquipmentType: selectTrailerTypes,
    isError: isTrailerTypeError,
    isLoading: isTrailerTypesLoading,
  } = useEquipmentTypes("TRAILER");

  const {
    selectEquipmentType: selectTractorTypes,
    isError: isTractorTypeError,
    isLoading: isTractorTypesLoading,
  } = useEquipmentTypes("TRACTOR");

  const {
    selectCommodityData,
    isLoading: isCommoditiesLoading,
    isError: isCommodityError,
  } = useCommodities();

  const {
    selectHazardousMaterials,
    isLoading: isHazmatLoading,
    isError: isHazmatError,
  } = useHazardousMaterial();

  useEffect(() => {
    const subscription = watch((value, { name }) => {
      if (name === "commodity" && commodities && value.commodity) {
        const selectedCommodity = (commodities as Commodity[]).find(
          (commodity) => commodity.id === value.commodity,
        );

        if (selectedCommodity?.minTemp && selectedCommodity?.maxTemp) {
          setValue("temperatureMin", selectedCommodity?.minTemp);
          setValue("temperatureMax", selectedCommodity?.maxTemp);
        }

        if (selectedCommodity?.hazardousMaterial) {
          setValue("hazardousMaterial", selectedCommodity?.hazardousMaterial);
        }
      }

      if (name === "trailer" && trailerData && value.trailer) {
        const selectedTrailer = (trailerData as Trailer[]).find(
          (trailer: Trailer) => trailer.id === value.trailer,
        );

        if (selectedTrailer?.equipmentType) {
          setValue("trailerType", selectedTrailer?.equipmentType);
        }
      }
    });

    return () => subscription.unsubscribe();
  }, [commodities, setValue, watch, trailerData]);

  if (isShipmentControlLoading) {
    return <Skeleton className="h-[40vh]" />;
  }

  return (
    <div className="rounded-md border border-border bg-card">
      <div className="flex justify-center rounded-t-md border-b border-border bg-background p-2">
        <TitleWithTooltip
          title={t("card.equipmentInfo.label")}
          tooltip={t("card.equipmentInfo.description")}
        />
      </div>
      <div className="grid grid-cols-1 gap-x-6 gap-y-4 p-4 md:grid-cols-2">
        <div className="col-span-1">
          <SelectInput
            name="trailer"
            options={selectTrailers}
            isLoading={isTrailerLoading}
            isFetchError={isTrailerError}
            control={control}
            rules={{ required: true }}
            label={t("card.equipmentInfo.fields.trailer.label")}
            placeholder={t("card.equipmentInfo.fields.trailer.placeholder")}
            description={t("card.equipmentInfo.fields.trailer.description")}
            hasPopoutWindow
            popoutLink="/equipment/trailer/"
            popoutLinkLabel="Trailer"
          />
        </div>
        <div className="col-span-1">
          <SelectInput
            options={selectTrailerTypes}
            isFetchError={isTrailerTypeError}
            isLoading={isTrailerTypesLoading}
            name="trailerType"
            control={control}
            rules={{ required: true }}
            label={t("card.equipmentInfo.fields.trailerType.label")}
            placeholder={t("card.equipmentInfo.fields.trailerType.placeholder")}
            description={t("card.equipmentInfo.fields.trailerType.description")}
            hasPopoutWindow
            popoutLink="/equipment/equipment-types/"
            popoutLinkLabel="Equipment Type"
          />
        </div>
        <div className="col-span-1">
          <SelectInput
            options={selectTractorTypes}
            isFetchError={isTractorTypeError}
            isLoading={isTractorTypesLoading}
            name="tractorType"
            control={control}
            label={t("card.equipmentInfo.fields.tractorType.label")}
            placeholder={t("card.equipmentInfo.fields.tractorType.placeholder")}
            description={t("card.equipmentInfo.fields.tractorType.description")}
            hasPopoutWindow
            popoutLink="/equipment/equipment-types/"
            popoutLinkLabel="Equipment Type"
          />
        </div>
        <div className="col-span-1">
          <SelectInput
            name="commodity"
            options={selectCommodityData}
            isLoading={isCommoditiesLoading}
            isFetchError={isCommodityError}
            control={control}
            rules={{ required: shipmentControlData.enforceCommodity || false }}
            label={t("card.equipmentInfo.fields.commodity.label")}
            placeholder={t("card.equipmentInfo.fields.commodity.placeholder")}
            description={t("card.equipmentInfo.fields.commodity.description")}
            hasPopoutWindow
            popoutLink="/shipment-management/commodity-codes/"
            popoutLinkLabel="Commodity Code"
            isClearable
          />
        </div>
        <div className="col-span-2">
          <SelectInput
            options={selectHazardousMaterials}
            isLoading={isHazmatLoading}
            isFetchError={isHazmatError}
            name="hazardousMaterial"
            control={control}
            label={t("card.equipmentInfo.fields.hazardousMaterial.label")}
            placeholder={t(
              "card.equipmentInfo.fields.hazardousMaterial.placeholder",
            )}
            description={t(
              "card.equipmentInfo.fields.hazardousMaterial.description",
            )}
            hasPopoutWindow
            popoutLink="/shipment-management/hazardous-materials/"
            popoutLinkLabel="Hazardous Material"
            isClearable
          />
        </div>
        <div className="col-span-1">
          <InputField
            name="temperatureMin"
            type="number"
            control={control}
            label={t("card.equipmentInfo.fields.temperatureMin.label")}
            placeholder={t(
              "card.equipmentInfo.fields.temperatureMin.placeholder",
            )}
            description={t(
              "card.equipmentInfo.fields.temperatureMin.description",
            )}
          />
        </div>
        <div className="col-span-1">
          <InputField
            name="temperatureMax"
            control={control}
            type="number"
            label={t("card.equipmentInfo.fields.temperatureMax.label")}
            placeholder={t(
              "card.equipmentInfo.fields.temperatureMax.placeholder",
            )}
            description={t(
              "card.equipmentInfo.fields.temperatureMax.description",
            )}
          />
        </div>
      </div>
    </div>
  );
}
