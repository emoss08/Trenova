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
import { TitleWithTooltip } from "@/components/ui/title-with-tooltip";
import { useCommodities, useEquipmentTypes, useHazardousMaterial, useTrailers } from "@/hooks/useQueries";
import { Commodity } from "@/types/commodities";
import { Trailer } from "@/types/equipment";
import { ShipmentFormProps } from "@/types/order";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";

export function EquipmentInformation({ control, setValue, watch }: ShipmentFormProps) {
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
        isLoading: isHazmatLoading,
    } = useHazardousMaterial();

    const { selectTrailers, isTrailerError, isTrailerLoading, trailerData } =
        useTrailers();

    const commodityValue = watch("commodity");
    const trailerValue = watch("trailer");

    // Use useEffect to respond to changes in originLocation and destinationLocation
    useEffect(() => {
        if (commodityValue) {
            const selectedCommodity = (commodities as Commodity[]).find(
                (commodity) => commodity.id === commodityValue,
            );

            if (selectedCommodity?.minTemp && selectedCommodity?.maxTemp) {
                setValue("temperatureMin", selectedCommodity?.minTemp);
                setValue("temperatureMax", selectedCommodity?.maxTemp);
            }

            if (selectedCommodity?.hazardousMaterial) {
                setValue("hazardousMaterial", selectedCommodity?.hazardousMaterial);
            }
        }

        if (trailerValue) {
            const selectedTrailer = (trailerData as Trailer[]).find(
                (trailer: Trailer) => trailer.id === trailerValue,
            );

            if (selectedTrailer?.equipmentType) {
                setValue("trailerType", selectedTrailer?.equipmentType);
            }
        }
    }, [
        commodityValue,
        hazardousMaterials,
        commodities,
        setValue,
        trailerValue,
        selectTrailers,
        trailerData,
    ]);

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
                <div className="col-span-2">
                    <SelectInput
                        name="hazardousMaterial"
                        control={control}
                        options={selectHazardousMaterials}
                        isLoading={isHazmatLoading}
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
            </div>
        </div>
    );
}
