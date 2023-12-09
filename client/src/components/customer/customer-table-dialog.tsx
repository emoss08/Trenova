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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { cn } from "@/lib/utils";
import { customerSchema } from "@/lib/validations/CustomerSchema";
import { useCustomerFormStore } from "@/stores/CustomerStore";
import { CustomerFormValues as FormValues } from "@/types/customer";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useState } from "react";
import { Control, useForm } from "react-hook-form";
import { CustomerContactForm } from "./customer-contacts-form";
import { CustomerEmailProfileForm } from "./customer-email-profile-form";
import { CustomerInfoForm } from "./customer-info-form";
import { CustomerRuleProfileForm } from "./customer-rule-profile-form";
import { DeliverySlotForm } from "./delivery-slots-form";

export function CustomerForm({
  control,
  open,
}: {
  control: Control<FormValues>;
  open: boolean;
}) {
  const [activeTab, setActiveTab] = useCustomerFormStore.use("activeTab");

  return (
    <Tabs
      defaultValue="info"
      value={activeTab}
      className="flex-1 w-full"
      onValueChange={setActiveTab}
    >
      <TabsList>
        <TabsTrigger value="info">Information</TabsTrigger>
        <TabsTrigger value="emailProfile">Email Profile</TabsTrigger>
        <TabsTrigger value="ruleProfile">Rule Profile</TabsTrigger>
        <TabsTrigger value="deliverySlots">Delivery Slots</TabsTrigger>
        <TabsTrigger value="contacts">Contacts</TabsTrigger>
        <TabsTrigger value="detentionPolicy">Detention Policy</TabsTrigger>
      </TabsList>
      <TabsContent value="info">
        <CustomerInfoForm control={control} open={open} />
      </TabsContent>
      <TabsContent value="emailProfile">
        <CustomerEmailProfileForm control={control} />
      </TabsContent>
      <TabsContent value="ruleProfile">
        <CustomerRuleProfileForm control={control} open={open} />
      </TabsContent>
      <TabsContent value="deliverySlots">
        <DeliverySlotForm control={control} open={open} />
      </TabsContent>
      <TabsContent value="contacts">
        <CustomerContactForm control={control} />
      </TabsContent>
      <TabsContent value="detentionPolicy">
        <p>Work in progress</p>
      </TabsContent>
    </Tabs>
  );
}

export function CustomerTableSheet({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(customerSchema),
    defaultValues: {
      status: "A",
      code: "",
      name: "",
      addressLine1: "",
      addressLine2: "",
      city: "",
      state: "",
      zipCode: "",
      hasCustomerPortal: false,
      autoMarkReadyToBill: false,
      deliverySlots: [],
      contacts: [],
      emailProfile: {
        subject: "",
        comment: "",
        fromAddress: "",
        blindCopy: "",
        readReceipt: false,
        readReceiptTo: "",
        attachmentName: "",
      },
      ruleProfile: {
        name: "",
        documentClass: [],
      },
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/customers/",
      successMessage: "Customer created successfully.",
      queryKeysToInvalidate: ["customers-table-data"],
      closeModal: true,
      errorMessage: "Failed to create new customer.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  function onSubmit(values: FormValues) {
    setIsSubmitting(true);
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
      </SheetContent>
    </Sheet>
  );
}
