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

import { CheckboxInput } from "@/components/common/fields/checkbox";
import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TitleWithTooltip } from "@/components/ui/title-with-tooltip";
import { useFormulaTemplates, useRates } from "@/hooks/useQueries";
import { ratingMethodChoies } from "@/lib/choices";
import { ShipmentFormValues } from "@/types/order";
import { Control, UseFormWatch } from "react-hook-form";
import { useTranslation } from "react-i18next";

export function RateCalcInformation({
  control,
  watch,
}: {
  control: Control<ShipmentFormValues>;
  watch: UseFormWatch<ShipmentFormValues>;
}) {
  const { t } = useTranslation(["shipment.addshipment", "common"]);
  const { selectRates, isRateError, isRatesLoading } = useRates();

  const { selectFormulaTemplates, isFormulaError, isFormulaLoading } =
    useFormulaTemplates();

  // If rating method is not other than "RateUnit" needs to be readonly

  const ratingMethod = watch("rateMethod");

  return (
    <div className="border-border bg-card rounded-md border">
      <div className="border-border bg-accent flex justify-center rounded-t-md border-b p-2">
        <TitleWithTooltip
          title={t("card.rateCalcInfo.label")}
          tooltip={t("card.rateCalcInfo.description")}
        />
      </div>
      <div className="p-4">
        <div className="grid max-w-3xl grid-cols-1 gap-4 sm:grid-cols-4">
          <div className="col-span-3">
            <SelectInput
              name="rateMethod"
              options={ratingMethodChoies}
              control={control}
              rules={{ required: true }}
              label={t("fields.ratingMethod.label")}
              placeholder={t("fields.ratingMethod.placeholder")}
              description={t("fields.ratingMethod.description")}
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="ratingUnits"
              type="number"
              control={control}
              rules={{ required: true }}
              readOnly={ratingMethod !== "O"}
              label={t("fields.ratingUnits.label")}
              placeholder={t("fields.ratingUnits.placeholder")}
              description={t("fields.ratingUnits.description")}
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="rate"
              options={selectRates}
              isLoading={isRatesLoading}
              isFetchError={isRateError}
              control={control}
              label={t("fields.rate.label")}
              placeholder={t("fields.rate.placeholder")}
              description={t("fields.rate.description")}
              popoutLink="/dispatch/rate-management/"
              hasPopoutWindow
              popoutLinkLabel="Rate"
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="formulaTemplate"
              options={selectFormulaTemplates}
              isLoading={isFormulaLoading}
              isFetchError={isFormulaError}
              control={control}
              label={t("fields.formulaTemplate.label")}
              placeholder={t("fields.formulaTemplate.placeholder")}
              description={t("fields.formulaTemplate.description")}
              popoutLink="/shipment-management/formula-templates/"
              hasPopoutWindow
              popoutLinkLabel="Formula Template"
            />
          </div>
          <div className="col-span-full">
            <CheckboxInput
              name="autoRate"
              control={control}
              rules={{ required: true }}
              label={t("fields.autoRate.label")}
              description={t("fields.autoRate.description")}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
