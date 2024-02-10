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
import {
  AsyncSelectInput,
  SelectInput,
} from "@/components/common/fields/select-input";
import { TitleWithTooltip } from "@/components/ui/title-with-tooltip";
import { ratingMethodChoies } from "@/lib/choices";
import { ShipmentFormValues } from "@/types/order";
import { useFormContext } from "react-hook-form";
import { useTranslation } from "react-i18next";

export function RateCalcInformation() {
  const { control, watch } = useFormContext<ShipmentFormValues>();
  const { t } = useTranslation("shipment.addshipment");
  const ratingMethodValue = watch("rateMethod");

  return (
    <div className="border-border bg-card rounded-md border">
      <div className="border-border bg-background flex justify-center rounded-t-md border-b p-2">
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
              label={t("card.rateCalcInfo.fields.ratingMethod.label")}
              placeholder={t(
                "card.rateCalcInfo.fields.ratingMethod.placeholder",
              )}
              description={t(
                "card.rateCalcInfo.fields.ratingMethod.description",
              )}
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="ratingUnits"
              type="number"
              control={control}
              rules={{ required: true }}
              readOnly={ratingMethodValue !== "O"}
              label={t("card.rateCalcInfo.fields.ratingUnits.label")}
              placeholder={t(
                "card.rateCalcInfo.fields.ratingUnits.placeholder",
              )}
              description={t(
                "card.rateCalcInfo.fields.ratingUnits.description",
              )}
            />
          </div>
          <div className="col-span-3">
            <AsyncSelectInput
              name="rate"
              link="rates"
              valueKey="rate_number"
              control={control}
              label={t("card.rateCalcInfo.fields.rate.label")}
              placeholder={t("card.rateCalcInfo.fields.rate.placeholder")}
              description={t("card.rateCalcInfo.fields.rate.description")}
              popoutLink="/dispatch/rate-management/"
              hasPopoutWindow
              popoutLinkLabel="Rate"
            />
          </div>
          <div className="col-span-3">
            <AsyncSelectInput
              name="formulaTemplate"
              link="formula_templates"
              control={control}
              label={t("card.rateCalcInfo.fields.formulaTemplate.label")}
              placeholder={t(
                "card.rateCalcInfo.fields.formulaTemplate.placeholder",
              )}
              description={t(
                "card.rateCalcInfo.fields.formulaTemplate.description",
              )}
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
              label={t("card.rateCalcInfo.fields.autoRate.label")}
              description={t("card.rateCalcInfo.fields.autoRate.description")}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
