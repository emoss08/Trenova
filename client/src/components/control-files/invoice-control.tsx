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
import { useInvoiceControl } from "@/hooks/useQueries";
import React from "react";
import { useForm } from "react-hook-form";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { CheckboxInput } from "@/components/common/fields/checkbox";
import { yupResolver } from "@hookform/resolvers/yup";
import { Skeleton } from "@/components/ui/skeleton";
import {
  InvoiceControl as InvoiceControlType,
  InvoiceControlFormValues,
} from "@/types/invoicing";
import { invoiceControlSchema } from "@/lib/validations/InvoicingSchema";
import { SelectInput } from "@/components/common/fields/select-input";
import { dateFormatChoices } from "@/lib/choices";
import { FileField, InputField } from "@/components/common/fields/input";
import { TextareaField } from "@/components/common/fields/textarea";
import { cleanObject } from "@/lib/utils";

function InvoiceControlForm({
  invoiceControl,
}: {
  invoiceControl: InvoiceControlType;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

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
      successMessage: "Invoice Control updated successfully.",
      queryKeysToInvalidate: ["invoiceControl"],
      errorMessage: "Failed to update invoice control.",
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
      className="m-4 bg-background ring-1 ring-muted sm:rounded-xl md:col-span-2"
      onSubmit={handleSubmit(onSubmit)}
    >
      <div className="px-4 py-6 sm:p-8">
        <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-6">
          <div className="col-span-3">
            <InputField
              name="invoiceNumberPrefix"
              rules={{ required: true }}
              control={control}
              label="Invoice Number Prefix"
              placeholder="Invoice Number Prefix"
              description="Set a specific prefix for invoice numbers to maintain a consistent numbering system."
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="creditMemoNumberPrefix"
              rules={{ required: true }}
              control={control}
              label="Credit Memo Number Prefix"
              placeholder="Credit Memo Number Prefix"
              description="Define a prefix for credit memo numbers to easily distinguish them from regular invoices."
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="invoiceDueAfterDays"
              control={control}
              rules={{ required: true }}
              label="Invoice Due After Days"
              placeholder="Credit Memo Number Prefix"
              description="Specify the default payment due period for invoices in days."
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="invoiceDateFormat"
              control={control}
              options={dateFormatChoices}
              rules={{ required: true }}
              label="Invoice Date Format"
              placeholder="Invoice Date Format"
              description="Enter the prefix to be used for the credit memo number."
            />
          </div>
          <div className="col-span-3">
            <FileField
              name="invoiceLogo"
              control={control}
              label="Invoice Logo"
              placeholder="Invoice Logo"
              description="Upload your company logo to personalize invoices and enhance brand presence."
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="invoiceLogoWidth"
              control={control}
              rules={{ required: true }}
              label="Invoice Logo Width"
              placeholder="Invoice Logo Width"
              description="Determine the display width of the logo on invoices, ensuring optimal visibility. (In pixels)"
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="attachPdf"
              control={control}
              label="Attach PDF"
              description="Option to attach a PDF version of the invoice with customer emails for easy access."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="showAmountDue"
              control={control}
              label="Show Amount Due"
              description="Choose to display the total amount due on the invoice for clear communication."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="showInvoiceDueDate"
              control={control}
              label="Show Invoice Due Date"
              description="Include the due date on invoices to remind customers of the payment timeline."
            />
          </div>
          <div className="col-span-full">
            <TextareaField
              name="invoiceTerms"
              control={control}
              label="Invoice Terms"
              placeholder="Invoice Terms"
              description="Enter custom terms and conditions for your invoices to inform clients about payment policies."
            />
          </div>
          <div className="col-span-full">
            <TextareaField
              name="invoiceFooter"
              control={control}
              label="Invoice Footer"
              placeholder="Invoice Footer"
              description="Add a custom footer to invoices for additional notes or company information."
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

export default function InvoiceControl() {
  const { invoiceControlData, isLoading } = useInvoiceControl();

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
      <div className="px-4 sm:px-0">
        <h2 className="text-base font-semibold leading-7 text-foreground">
          Invoice Control
        </h2>
        <p className="mt-1 text-sm leading-6 text-muted-foreground">
          Enhance your billing accuracy and efficiency with our Invoice
          Customization Panel. Tailor every aspect of your invoicing process,
          from numbering to presentation, ensuring a seamless fit with your
          company's branding and operational requirements in the transportation
          sector.
        </p>
      </div>
      {isLoading ? (
        <div className="m-4 bg-background ring-1 ring-muted sm:rounded-xl md:col-span-2">
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
