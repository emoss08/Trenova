/*
 * COPYRIGHT(c) 2023 MONTA
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
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { cn } from "@/lib/utils";
import { customerSchema } from "@/lib/validations/CustomerSchema";
import { useTableStore } from "@/stores/TableStore";
import { Customer, CustomerFormValues as FormValues } from "@/types/customer";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { CustomerForm } from "./customer-table-dialog";

export function CustomerEditForm({
  customer,
  open,
  onOpenChange,
}: {
  customer: Customer;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  if (!customer) return null;

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(customerSchema),
    defaultValues: {
      status: customer.status,
      code: customer.code,
      name: customer.name,
      addressLine1: customer.addressLine1,
      addressLine2: customer.addressLine2,
      city: customer.city,
      state: customer.state,
      zipCode: customer.zipCode,
      hasCustomerPortal: customer.hasCustomerPortal,
      autoMarkReadyToBill: customer.autoMarkReadyToBill,
      deliverySlots: customer?.deliverySlots || [],
      contacts: customer?.contacts || [],
      emailProfile: {
        subject: customer.emailProfile?.subject || "",
        comment: customer.emailProfile?.comment || "",
        fromAddress: customer.emailProfile?.fromAddress || "",
        blindCopy: customer.emailProfile?.blindCopy || "",
        readReceipt: customer.emailProfile?.readReceipt || false,
        readReceiptTo: customer.emailProfile?.readReceiptTo || "",
        attachmentName: customer.emailProfile?.attachmentName || "",
      },
      ruleProfile: {
        name: customer.ruleProfile?.name || "",
        documentClass: customer.ruleProfile?.documentClass || [],
      },
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/customers/${customer.id}/`,
      successMessage: "Customer updated successfully.",
      queryKeysToInvalidate: ["customers-table-data"],
      additionalInvalidateQueries: ["customers"],
      closeModal: true,
      errorMessage: "Failed to update existing customer.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  function onSubmit(values: FormValues) {
    setIsSubmitting(true);
    mutation.mutate(values);
  }

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex flex-col h-full overflow-y-auto"
    >
      <CustomerForm control={control} open={open} />
      <SheetFooter className="mb-12">
        <Button
          type="reset"
          variant="secondary"
          onClick={() => onOpenChange(false)}
          className="w-full"
        >
          Cancel
        </Button>
        <Button
          type="submit"
          isLoading={isSubmitting}
          loadingText="Saving Changes..."
          className="w-full"
        >
          Save
        </Button>
      </SheetFooter>
    </form>
  );
}

export function CustomerTableEditSheet({
  onOpenChange,
  open,
}: TableSheetProps) {
  const [customer] = useTableStore.use("currentRecord") as Customer[];

  if (!customer) return null;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>{customer && customer.name}</SheetTitle>
          <SheetDescription>
            Last updated on {customer && formatDate(customer.modified)}
          </SheetDescription>
        </SheetHeader>
        {customer && (
          <CustomerEditForm
            customer={customer}
            open={open}
            onOpenChange={onOpenChange}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}
