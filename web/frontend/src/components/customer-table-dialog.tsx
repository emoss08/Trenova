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
import { cn } from "@/lib/utils";
import { customerSchema } from "@/lib/validations/CustomerSchema";
import { useCustomerFormStore } from "@/stores/CustomerStore";
import { type CustomerFormValues as FormValues } from "@/types/customer";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { FormProvider, useForm } from "react-hook-form";
import { CustomerContactForm } from "./customer-contacts-form";
import { CustomerEmailProfileForm } from "./customer-email-profile-form";
import { CustomerInfoForm } from "./customer-info-form";
import { CustomerRuleProfileForm } from "./customer-rule-profile-form";
import { DeliverySlotForm } from "./delivery-slots-form";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "./ui/new/new-tabs";

export function CustomerForm({ open }: { open: boolean }) {
  const [activeTab, setActiveTab] = useCustomerFormStore.use("activeTab");

  return (
    <Tabs
      defaultValue="info"
      value={activeTab}
      className="mt-10 w-full flex-1"
      onValueChange={setActiveTab}
    >
      <TabsList className="mx-auto space-x-4">
        <TabsTrigger value="info">General Information</TabsTrigger>
        <TabsTrigger value="emailProfile">Email Profile</TabsTrigger>
        <TabsTrigger value="ruleProfile">Rule Profile</TabsTrigger>
        <TabsTrigger value="deliverySlots">Delivery Slots</TabsTrigger>
        <TabsTrigger value="contacts">Contacts</TabsTrigger>
        <TabsTrigger value="detentionPolicy">Detention Policy</TabsTrigger>
      </TabsList>
      <TabsContent value="info">
        <CustomerInfoForm open={open} />
      </TabsContent>
      <TabsContent value="emailProfile">
        <CustomerEmailProfileForm />
      </TabsContent>
      <TabsContent value="ruleProfile">
        <CustomerRuleProfileForm open={open} />
      </TabsContent>
      <TabsContent value="deliverySlots">
        <DeliverySlotForm open={open} />
      </TabsContent>
      <TabsContent value="contacts">
        <CustomerContactForm />
      </TabsContent>
      <TabsContent value="detentionPolicy">
        <p>Coming Soon...</p>
      </TabsContent>
    </Tabs>
  );
}

export function CustomerTableSheet({ onOpenChange, open }: TableSheetProps) {
  const customerForm = useForm<FormValues>({
    resolver: yupResolver(customerSchema),
    defaultValues: {
      status: "A",
      name: "",
      addressLine1: "",
      addressLine2: "",
      city: "",
      stateId: "",
      postalCode: "",
      hasCustomerPortal: false,
      autoMarkReadyToBill: false,
    },
  });

  const { control, reset, handleSubmit } = customerForm;

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/customers/",
    successMessage: "Customer created successfully.",
    queryKeysToInvalidate: "customers",
    closeModal: true,
    reset,
    errorMessage: "Failed to create new customer.",
  });

  function onSubmit(values: FormValues) {
    mutation.mutate(values);
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>Add New Customer</SheetTitle>
          <SheetDescription>
            Use this form to add a new customer to the system.
          </SheetDescription>
        </SheetHeader>
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
      </SheetContent>
    </Sheet>
  );
}
