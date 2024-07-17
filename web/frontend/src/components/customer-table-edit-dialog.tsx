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
