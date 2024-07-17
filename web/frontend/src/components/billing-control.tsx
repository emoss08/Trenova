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

import { CheckboxInput } from "@/components/common/fields/checkbox";
import { SelectInput } from "@/components/common/fields/select-input";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useBillingControl } from "@/hooks/useQueries";
import { billingControlSchema } from "@/lib/validations/BillingSchema";
import type {
  BillingControlFormValues,
  BillingControl as BillingControlType,
} from "@/types/billing";
import {
  autoBillingCriteriaChoices,
  shipmentTransferCriteriaChoices,
} from "@/utils/apps/billing";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { ErrorLoadingData } from "./common/table/data-table-components";
import { ComponentLoader } from "./ui/component-loader";

function BillingControlForm({
  billingControl,
}: {
  billingControl: BillingControlType;
}) {
  const { t } = useTranslation(["admin.billingcontrol", "common"]);

  const { control, handleSubmit, reset } = useForm<BillingControlFormValues>({
    resolver: yupResolver(billingControlSchema),
    defaultValues: {
      removeBillingHistory: billingControl.removeBillingHistory,
      autoBillShipment: billingControl.autoBillShipment,
      autoMarkReadyToBill: billingControl.autoMarkReadyToBill,
      validateCustomerRates: billingControl.validateCustomerRates,
      autoBillCriteria: billingControl.autoBillCriteria,
      shipmentTransferCriteria: billingControl.shipmentTransferCriteria,
      enforceCustomerBilling: billingControl.enforceCustomerBilling,
    },
  });

  const mutation = useCustomMutation<BillingControlFormValues>(control, {
    method: "PUT",
    path: `/billing-control/${billingControl.id}/`,
    successMessage: t("formSuccessMessage"),
    queryKeysToInvalidate: "billingControl",
    reset,
    errorMessage: t("formErrorMessage"),
  });

  const onSubmit = (values: BillingControlFormValues) => {
    mutation.mutate(values);
    reset(values);
  };

  return (
    <form
      className="border-border bg-card m-4 border sm:rounded-xl md:col-span-2"
      onSubmit={handleSubmit(onSubmit)}
    >
      <div className="px-4 py-6 sm:p-8">
        <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-6">
          <div className="col-span-3">
            <CheckboxInput
              name="removeBillingHistory"
              control={control}
              label={t("fields.removeBillingHistory.label")}
              description={t("fields.removeBillingHistory.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="autoBillShipment"
              control={control}
              label={t("fields.autoBillShipment.label")}
              description={t("fields.autoBillShipment.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="autoMarkReadyToBill"
              control={control}
              label={t("fields.autoMarkReadyToBill.label")}
              description={t("fields.autoMarkReadyToBill.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="validateCustomerRates"
              control={control}
              label={t("fields.validateCustomerRates.label")}
              description={t("fields.validateCustomerRates.description")}
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="autoBillCriteria"
              control={control}
              options={autoBillingCriteriaChoices}
              rules={{ required: true }}
              label={t("fields.autoBillCriteria.label")}
              placeholder={t("fields.autoBillCriteria.placeholder")}
              description={t("fields.autoBillCriteria.description")}
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="shipmentTransferCriteria"
              control={control}
              options={shipmentTransferCriteriaChoices}
              rules={{ required: true }}
              label={t("fields.shipmentTransferCriteria.label")}
              placeholder={t("fields.shipmentTransferCriteria.placeholder")}
              description={t("fields.shipmentTransferCriteria.description")}
            />
          </div>
          <div className="col-span-full">
            <CheckboxInput
              name="enforceCustomerBilling"
              control={control}
              label={t("fields.enforceCustomerBilling.label")}
              description={t("fields.enforceCustomerBilling.description")}
            />
          </div>
        </div>
      </div>
      <div className="border-muted flex items-center justify-end gap-x-4 border-t p-4 sm:px-8">
        <Button
          onClick={(e) => {
            e.preventDefault();
            reset();
          }}
          type="button"
          variant="outline"
          disabled={mutation.isPending}
        >
          {t("buttons.cancel", { ns: "common" })}
        </Button>
        <Button type="submit" isLoading={mutation.isPending}>
          {t("buttons.save", { ns: "common" })}
        </Button>
      </div>
    </form>
  );
}

export default function BillingControl() {
  const { data, isLoading, isError } = useBillingControl();
  const { t } = useTranslation("admin.billingcontrol");

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
      <div className="px-4 sm:px-0">
        <h2 className="text-foreground text-base font-semibold leading-7">
          {t("title")}
        </h2>
        <p className="text-muted-foreground mt-1 text-sm leading-6">
          {t("subTitle")}
        </p>
      </div>
      {isLoading ? (
        <div className="bg-background ring-muted m-4 ring-1 sm:rounded-xl md:col-span-2">
          <ComponentLoader className="h-[30em]" />
        </div>
      ) : isError ? (
        <ErrorLoadingData />
      ) : (
        data && <BillingControlForm billingControl={data} />
      )}
    </div>
  );
}
