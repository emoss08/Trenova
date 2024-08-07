/**
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
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { cn } from "@/lib/utils";
import { customerSchema } from "@/lib/validations/CustomerSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  Customer,
  CustomerFormValues as FormValues,
} from "@/types/customer";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { FormProvider, useForm } from "react-hook-form";
import { CustomerForm } from "./customer-table-dialog";
import { Badge } from "./ui/badge";

function CustomerEditForm({
  customer,
  open,
  onOpenChange,
}: {
  customer: Customer;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const customerForm = useForm<FormValues>({
    resolver: yupResolver(customerSchema),
    defaultValues: customer,
  });

  const { control, reset, handleSubmit } = customerForm;

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/customers/${customer.id}/`,
    successMessage: "Customer updated successfully.",
    queryKeysToInvalidate: "customers",
    closeModal: true,
    reset,
    errorMessage: "Failed to update existing customer.",
  });

  if (!customer) return null;

  function onSubmit(values: FormValues) {
    mutation.mutate(values);
  }

  return (
    <FormProvider {...customerForm}>
      <form
        onSubmit={handleSubmit(onSubmit)}
        className="flex h-full flex-col overflow-y-auto"
      >
        <CustomerForm open={open} />
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
            isLoading={mutation.isPending}
            className="w-full"
          >
            Save
          </Button>
        </SheetFooter>
      </form>
    </FormProvider>
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
          <SheetTitle className="flex">
            <span>{customer.name}</span>
            <Badge className="ml-5" variant="purple">
              {customer.id}
            </Badge>
          </SheetTitle>
          <SheetDescription>
            Last updated on {formatToUserTimezone(customer.updatedAt)}
          </SheetDescription>
        </SheetHeader>
        <CustomerEditForm
          customer={customer}
          open={open}
          onOpenChange={onOpenChange}
        />
      </SheetContent>
    </Sheet>
  );
}
