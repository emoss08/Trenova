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
import { AsyncSelectInput } from "@/components/common/fields/select-input";
import { Skeleton } from "@/components/ui/skeleton";
import { TitleWithTooltip } from "@/components/ui/title-with-tooltip";
import {
  useCommodities,
  useHazardousMaterial,
  useTrailers,
} from "@/hooks/useQueries";
import { ShipmentControl, ShipmentFormValues } from "@/types/order";
import { useFormContext } from "react-hook-form";
import { useTranslation } from "react-i18next";

export function EquipmentInformation({
  shipmentControlData,
  isShipmentControlLoading,
}: {
  shipmentControlData: ShipmentControl;
  isShipmentControlLoading: boolean;
}) {
  const { t } = useTranslation("shipment.addshipment");
  const { data: commodities } = useCommodities();
  const { data: hazardousMaterials } = useHazardousMaterial();
  const { selectTrailers, trailerData } = useTrailers();
  const { control, setValue, watch } = useFormContext<ShipmentFormValues>();

  const commodityValue = watch("commodity");
  const trailerValue = watch("trailer");

  // TODO(WOLFRED): Rewrite this code to subscribe and unsubscribe.
  // // Use useEffect to respond to changes in originLocation and destinationLocation
  // useEffect(() => {
  //   if (commodityValue) {
  //     const selectedCommodity = (commodities as Commodity[]).find(
  //       (commodity) => commodity.id === commodityValue,
  //     );

  //     if (selectedCommodity?.minTemp && selectedCommodity?.maxTemp) {
  //       setValue("temperatureMin", selectedCommodity?.minTemp);
  //       setValue("temperatureMax", selectedCommodity?.maxTemp);
  //     }

  //     if (selectedCommodity?.hazardousMaterial) {
  //       setValue("hazardousMaterial", selectedCommodity?.hazardousMaterial);
  //     }
  //   }

  //   if (trailerValue) {
  //     const selectedTrailer = (trailerData as Trailer[]).find(
  //       (trailer: Trailer) => trailer.id === trailerValue,
  //     );

  //     if (selectedTrailer?.equipmentType) {
  //       setValue("trailerType", selectedTrailer?.equipmentType);
  //     }
  //   }
  // }, [
  //   commodityValue,
  //   hazardousMaterials,
  //   commodities,
  //   setValue,
  //   trailerValue,
  //   selectTrailers,
  //   trailerData,
  // ]);

  if (isShipmentControlLoading) {
    return <Skeleton className="h-[40vh]" />;
  }

  return (
    <div className="border-border bg-card rounded-md border">
      <div className="border-border bg-background flex justify-center rounded-t-md border-b p-2">
        <TitleWithTooltip
          title={t("card.equipmentInfo.label")}
          tooltip={t("card.equipmentInfo.description")}
        />
      </div>
      <div className="grid grid-cols-1 gap-x-6 gap-y-4 p-4 md:grid-cols-2">
        <div className="col-span-1">
          <AsyncSelectInput
            name="trailer"
            link="/trailers/"
            valueKey="code"
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
          <AsyncSelectInput
            name="trailerType"
            link="/equipment_types/"
            valueKey="name"
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
          <AsyncSelectInput
            name="tractorType"
            link="/equipment_types/"
            valueKey="name"
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
          <AsyncSelectInput
            name="commodity"
            link="/commodities/"
            valueKey="name"
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
          <AsyncSelectInput
            name="hazardousMaterial"
            link="/hazardous_materials/"
            valueKey="name"
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
