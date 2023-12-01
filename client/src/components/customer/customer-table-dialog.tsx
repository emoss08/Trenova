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
import { toast } from "@/components/ui/use-toast";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { cn } from "@/lib/utils";
import { customerSchema } from "@/lib/validations/CustomerSchema";
import { CustomerFormValues as FormValues } from "@/types/customer";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { CustomerInfoForm } from "./customer-info-form";
import { DeliverySlotForm } from "./delivery-slots-form";

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
      customerContacts: [],
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
    toast,
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

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

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
          <Tabs defaultValue="info" className="flex-1 w-full">
            <TabsList>
              <TabsTrigger value="info">Information</TabsTrigger>
              <TabsTrigger value="email_profile">Email Profile</TabsTrigger>
              <TabsTrigger value="rule_profile">Rule Profile</TabsTrigger>
              <TabsTrigger value="delivery_slots">Delivery Slots</TabsTrigger>
              <TabsTrigger value="contacts">Contacts</TabsTrigger>
            </TabsList>
            <TabsContent value="info">
              <CustomerInfoForm control={control} open={open} />
            </TabsContent>
            <TabsContent value="email_profile">
              {/* <LocationContactForm control={control} /> */}
            </TabsContent>
            <TabsContent value="rule_profile">
              {/* <LocationCommentForm control={control} /> */}
            </TabsContent>
            <TabsContent value="delivery_slots">
              <DeliverySlotForm control={control} open={open} />
            </TabsContent>
            <TabsContent value="contacts">
              {/* <LocationContactForm control={control} /> */}
            </TabsContent>
          </Tabs>
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
