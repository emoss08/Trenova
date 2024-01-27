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
import { FileField, InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useInvoiceControl } from "@/hooks/useQueries";
import { dateFormatChoices } from "@/lib/choices";
import { cleanObject } from "@/lib/utils";
import { invoiceControlSchema } from "@/lib/validations/InvoicingSchema";
import {
  InvoiceControlFormValues,
  InvoiceControl as InvoiceControlType,
} from "@/types/invoicing";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";

function InvoiceControlForm({
  invoiceControl,
}: {
  invoiceControl: InvoiceControlType;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { t } = useTranslation(["admin.invoicecontrol", "common"]);

  const { control, handleSubmit, reset } = useForm<InvoiceControlFormValues>({
    resolver: yupResolver(invoiceControlSchema),
    defaultValues: {
      invoiceNumberPrefix: invoiceControl.invoiceNumberPrefix,
      creditMemoNumberPrefix: invoiceControl.creditMemoNumberPrefix,
      invoiceDueAfterDays: invoiceControl.invoiceDueAfterDays,
      invoiceDateFormat: invoiceControl.invoiceDateFormat,
      invoiceLogo: invoiceControl.invoiceLogo,
      invoiceLogoWidth: invoiceControl.invoiceLogoWidth,
      attachPdf: invoiceControl.attachPdf,
      showAmountDue: invoiceControl.showAmountDue,
      showInvoiceDueDate: invoiceControl.showInvoiceDueDate,
      invoiceTerms: invoiceControl.invoiceTerms,
      invoiceFooter: invoiceControl.invoiceFooter,
    },
  });

  const mutation = useCustomMutation<InvoiceControlFormValues>(
    control,
    {
      method: "PUT",
      path: `/invoice_control/${invoiceControl.id}/`,
      successMessage: t("formSuccessMessage"),
      queryKeysToInvalidate: ["invoiceControl"],
      errorMessage: t("formErrorMessage"),
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: InvoiceControlFormValues) => {
    const cleanedValues = cleanObject(values);

    setIsSubmitting(true);
    mutation.mutate(cleanedValues);
  };

  return (
    <form
      className="bg-background m-4 border sm:rounded-xl md:col-span-2"
      onSubmit={handleSubmit(onSubmit)}
    >
      <div className="px-4 py-6 sm:p-8">
        <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-6">
          <div className="col-span-3">
            <InputField
              name="invoiceNumberPrefix"
              rules={{ required: true }}
              control={control}
              label={t("fields.invoiceNumberPrefix.label")}
              placeholder={t("fields.invoiceNumberPrefix.placeholder")}
              description={t("fields.invoiceNumberPrefix.description")}
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="creditMemoNumberPrefix"
              rules={{ required: true }}
              control={control}
              label={t("fields.creditMemoNumberPrefix.label")}
              placeholder={t("fields.creditMemoNumberPrefix.placeholder")}
              description={t("fields.creditMemoNumberPrefix.description")}
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="invoiceDueAfterDays"
              control={control}
              rules={{ required: true }}
              label={t("fields.invoiceDueAfterDays.label")}
              placeholder={t("fields.invoiceDueAfterDays.placeholder")}
              description={t("fields.invoiceDueAfterDays.description")}
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="invoiceDateFormat"
              control={control}
              options={dateFormatChoices}
              rules={{ required: true }}
              label={t("fields.invoiceDateFormat.label")}
              placeholder={t("fields.invoiceDateFormat.placeholder")}
              description={t("fields.invoiceDateFormat.description")}
            />
          </div>
          <div className="col-span-3">
            <FileField
              name="invoiceLogo"
              control={control}
              label={t("fields.invoiceLogo.label")}
              placeholder={t("fields.invoiceLogo.placeholder")}
              description={t("fields.invoiceLogo.description")}
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="invoiceLogoWidth"
              control={control}
              rules={{ required: true }}
              label={t("fields.invoiceLogoWidth.label")}
              placeholder={t("fields.invoiceLogoWidth.placeholder")}
              description={t("fields.invoiceLogoWidth.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="attachPdf"
              control={control}
              label={t("fields.attachPdf.label")}
              description={t("fields.attachPdf.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="showAmountDue"
              control={control}
              label={t("fields.showAmountDue.label")}
              description={t("fields.showAmountDue.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="showInvoiceDueDate"
              control={control}
              label={t("fields.showInvoiceDueDate.label")}
              description={t("fields.showInvoiceDueDate.description")}
            />
          </div>
          <div className="col-span-full">
            <TextareaField
              name="invoiceTerms"
              control={control}
              label={t("fields.invoiceTerms.label")}
              placeholder={t("fields.invoiceTerms.placeholder")}
              description={t("fields.invoiceTerms.description")}
            />
          </div>
          <div className="col-span-full">
            <TextareaField
              name="invoiceFooter"
              control={control}
              label={t("fields.invoiceFooter.label")}
              placeholder={t("fields.invoiceFooter.placeholder")}
              description={t("fields.invoiceFooter.description")}
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

export default function InvoiceControl() {
  const { invoiceControlData, isLoading } = useInvoiceControl();
  const { t } = useTranslation("admin.invoicecontrol");

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
          <Skeleton className="h-screen w-full" />
        </div>
      ) : (
        invoiceControlData && (
          <InvoiceControlForm invoiceControl={invoiceControlData} />
        )
      )}
    </div>
  );
}
