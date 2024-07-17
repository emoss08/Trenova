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

import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TitleWithTooltip } from "@/components/ui/title-with-tooltip";
import { useEquipmentTypes, useTrailers } from "@/hooks/useQueries";
import { Trailer } from "@/types/equipment";
import { ShipmentFormValues } from "@/types/shipment";
import { useEffect } from "react";
import { useFormContext } from "react-hook-form";
import { useTranslation } from "react-i18next";

export default function EquipmentInformationCard() {
  const { t } = useTranslation("shipment.addshipment");
  const { trailerData, isTrailerError, isTrailerLoading, selectTrailers } =
    useTrailers();
  const { control, setValue, watch } = useFormContext<ShipmentFormValues>();

  const {
    selectEquipmentType: selectTrailerTypes,
    isError: isTrailerTypeError,
    isLoading: isTrailerTypesLoading,
  } = useEquipmentTypes();

  const {
    selectEquipmentType: selectTractorTypes,
    isError: isTractorTypeError,
    isLoading: isTractorTypesLoading,
  } = useEquipmentTypes();

  useEffect(() => {
    const subscription = watch((value, { name }) => {
      if (name === "trailer" && trailerData && value.trailer) {
        const selectedTrailer = (trailerData as Trailer[]).find(
          (trailer: Trailer) => trailer.id === value.trailer,
        );

        if (selectedTrailer?.equipmentTypeId) {
          setValue("trailerType", selectedTrailer?.equipmentTypeId);
        }
      }
    });

    return () => subscription.unsubscribe();
  }, [setValue, watch, trailerData]);

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
