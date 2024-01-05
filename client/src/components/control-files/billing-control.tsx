/*
 * COPYRIGHT(c) 2024 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

function BillingControlForm({
  billingControl,
}: {
  billingControl: BillingControlType;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

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
      successMessage: "Billing Control updated successfully.",
      queryKeysToInvalidate: ["billingControl"],
      errorMessage: "Failed to update billing control.",
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
      className="m-4 bg-background ring-1 ring-muted sm:rounded-xl md:col-span-2"
      onSubmit={handleSubmit(onSubmit)}
    >
      <div className="px-4 py-6 sm:p-8">
        <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-6">
          <div className="col-span-3">
            <CheckboxInput
              name="removeBillingHistory"
              control={control}
              label="Remove Billing History"
              description="Enable users to delete records from the billing history."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="autoBillShipment"
              control={control}
              label="Auto Bill Shipments"
              description="Automate the process of billing shipments directly to customers upon completion."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="autoMarkReadyToBill"
              control={control}
              label="Auto Mark Ready to Bill"
              description="Automatically mark shipments as 'Ready to Bill' upon delivery and meeting customer billing criteria."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="validateCustomerRates"
              control={control}
              label="Validate Customer Rates"
              description="Ensure billing rates align with customer contracts before processing, to maintain accuracy and compliance."
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="autoBillCriteria"
              control={control}
              options={autoBillingCriteriaChoices}
              rules={{ required: true }}
              label="Auto Bill Criteria"
              placeholder="Auto Bill Criteria"
              description="Set specific conditions under which shipments are automatically billed."
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="shipmentTransferCriteria"
              control={control}
              options={shipmentTransferCriteriaChoices}
              rules={{ required: true }}
              label="Shipment Transfer Criteria"
              placeholder="Shipment Transfer Criteria"
              description="Establish guidelines for transferring shipments to the billing phase."
            />
          </div>
          <div className="col-span-full">
            <CheckboxInput
              name="enforceCustomerBilling"
              control={control}
              label="Enforce Customer Billing Requirements"
              description="Mandate adherence to customer billing requirements during the billing process."
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
          Cancel
        </Button>
        <Button type="submit" isLoading={isSubmitting}>
          Save
        </Button>
      </div>
    </form>
  );
}

export default function BillingControl() {
  const { billingControlData, isLoading } = useBillingControl();

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
      <div className="px-4 sm:px-0">
        <h2 className="text-base font-semibold leading-7 text-foreground">
          Billing Control
        </h2>
        <p className="mt-1 text-sm leading-6 text-muted-foreground">
          Optimize and automate your billing processes for enhanced accuracy and
          efficiency. This module consolidates essential billing
          functionalities, ensuring precise invoicing, effective communication,
          and customized financial handling specifically tailored for the
          dynamics of the transportation industry.
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
