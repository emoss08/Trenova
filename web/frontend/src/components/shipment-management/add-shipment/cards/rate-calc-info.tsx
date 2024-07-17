/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { AsyncSelectInput } from "@/components/common/fields/async-select-input";
import { CheckboxInput } from "@/components/common/fields/checkbox";
import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TitleWithTooltip } from "@/components/ui/title-with-tooltip";
import { ratingMethodChoices } from "@/lib/choices";
import { ShipmentFormValues } from "@/types/shipment";
import { useFormContext } from "react-hook-form";
import { useTranslation } from "react-i18next";

export default function RateCalcInformation() {
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
              options={ratingMethodChoices}
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
              link="/rates/"
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
              link="/formula-templates/"
              control={control}
              label={t("card.rateCalcInfo.fields.formulaTemplate.label")}
              placeholder={t(
                "card.rateCalcInfo.fields.formulaTemplate.placeholder",
              )}
              description={t(
                "card.rateCalcInfo.fields.formulaTemplate.description",
              )}
              popoutLink="/shipments/formula-templates/"
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
