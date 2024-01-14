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

import { Button } from "@/components/ui/button";
import { useBillingControl } from "@/hooks/useQueries";
import React from "react";
import { useForm } from "react-hook-form";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { SelectInput } from "@/components/common/fields/select-input";
import { CheckboxInput } from "@/components/common/fields/checkbox";
import { yupResolver } from "@hookform/resolvers/yup";
import { Skeleton } from "@/components/ui/skeleton";
import {
  BillingControl as BillingControlType,
  BillingControlFormValues,
} from "@/types/billing";
import { billingControlSchema } from "@/lib/validations/BillingSchema";
import {
  autoBillingCriteriaChoices,
  shipmentTransferCriteriaChoices,
} from "@/utils/apps/billing";
import { useTranslation } from "react-i18next";

function BillingControlForm({
  billingControl,
}: {
  billingControl: BillingControlType;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
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

  const mutation = useCustomMutation<BillingControlFormValues>(
    control,
    {
      method: "PUT",
      path: `/billing_control/${billingControl.id}/`,
      successMessage: t("formSuccessMessage"),
      queryKeysToInvalidate: ["billingControl"],
      errorMessage: t("formErrorMessage"),
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: BillingControlFormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
    reset(values);
  };

  return (
    <form
      className="m-4 border bg-background sm:rounded-xl md:col-span-2"
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
      <div className="flex items-center justify-end gap-x-6 border-t border-muted p-4 sm:px-8">
        <Button
          onClick={(e) => {
            e.preventDefault();
            reset();
          }}
          type="button"
          variant="ghost"
          disabled={isSubmitting}
        >
          {t("buttons.cancel", { ns: "common" })}
        </Button>
        <Button type="submit" isLoading={isSubmitting}>
          {t("buttons.save", { ns: "common" })}
        </Button>
      </div>
    </form>
  );
}

export default function BillingControl() {
  const { billingControlData, isLoading } = useBillingControl();
  const { t } = useTranslation("admin.billingcontrol");

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
      <div className="px-4 sm:px-0">
        <h2 className="text-base font-semibold leading-7 text-foreground">
          {t("title")}
        </h2>
        <p className="mt-1 text-sm leading-6 text-muted-foreground">
          {t("subTitle")}
        </p>
      </div>
      {isLoading ? (
        <div className="m-4 bg-background ring-1 ring-muted sm:rounded-xl md:col-span-2">
          <Skeleton className="h-screen w-full" />
        </div>
      ) : (
        billingControlData && (
          <BillingControlForm billingControl={billingControlData} />
        )
      )}
    </div>
  );
}
