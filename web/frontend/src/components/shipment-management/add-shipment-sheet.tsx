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

import { useCustomMutation } from "@/hooks/useCustomMutation";
import { cleanObject, cn } from "@/lib/utils";
import { useShipmentForm } from "@/lib/validations/ShipmentSchema";
import { useUserStore } from "@/stores/AuthStore";
import type { ShipmentFormValues } from "@/types/shipment";
import type { TableSheetProps } from "@/types/tables";
import { useState } from "react";
import { FormProvider } from "react-hook-form";
import { Button } from "../ui/button";
import { ComponentLoader } from "../ui/component-loader";
import { Form } from "../ui/form";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../ui/new/new-tabs";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "../ui/sheet";
import { BillingInfoTab } from "./add-shipment/billing-info-tab";
import { GeneralInfoTab } from "./add-shipment/general-info-tab";
import { StopInfoTab } from "./add-shipment/stop-info-tab";

function ShipmentForm() {
  const [activeTab, setActiveTab] = useState<string>("info");

  return (
    <Tabs
      defaultValue="info"
      value={activeTab}
      className="mt-10 w-full flex-1"
      onValueChange={setActiveTab}
    >
      <TabsList className="mx-auto space-x-4">
        <TabsTrigger value="info">General Information</TabsTrigger>
        <TabsTrigger value="stops">Stop Information</TabsTrigger>
        <TabsTrigger value="billing">Billing Information</TabsTrigger>
        <TabsTrigger value="comments">Comments</TabsTrigger>
        <TabsTrigger value="documents">Documents</TabsTrigger>
        <TabsTrigger value="edi">EDI Information</TabsTrigger>
      </TabsList>
      <TabsContent value="info">
        <GeneralInfoTab />
      </TabsContent>
      <TabsContent value="stops">
        <StopInfoTab />
      </TabsContent>
      <TabsContent value="billing">
        <BillingInfoTab />
      </TabsContent>
      <TabsContent value="comments">
        <div>coming soon</div>
      </TabsContent>
      <TabsContent value="contacts">
        <div>coming soon</div>
      </TabsContent>
      <TabsContent value="edi">
        <p>Coming Soon...</p>
      </TabsContent>
    </Tabs>
  );
}

export function ShipmentSheet({ onOpenChange, open }: TableSheetProps) {
  const [user] = useUserStore.use("user");

  const { shipmentForm, isShipmentControlLoading, shipmentControlData } =
    useShipmentForm({ user });
  const { control, handleSubmit, reset } = shipmentForm;

  const mutation = useCustomMutation<ShipmentFormValues>(control, {
    method: "POST",
    path: "/shipments/",
    successMessage: "Shipment created successfully.",
    queryKeysToInvalidate: "shipments",
    closeModal: true,
    reset,
    errorMessage: "Failed to create new shipment.",
  });

  const onSubmit = (values: ShipmentFormValues) => {
    const cleanedValues = cleanObject(values);
    mutation.mutate(cleanedValues);
  };

  if (isShipmentControlLoading && !shipmentControlData && !user) {
    return <ComponentLoader className="h-[60vh]" />;
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>Add New Shipment</SheetTitle>
          <SheetDescription>
            Use this form to add a new shipment to the system.
          </SheetDescription>
        </SheetHeader>
        <FormProvider {...shipmentForm}>
          <Form
            onSubmit={handleSubmit(onSubmit)}
            className="flex h-full flex-col overflow-y-auto"
          >
            <ShipmentForm />
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
          </Form>
        </FormProvider>
      </SheetContent>
    </Sheet>
  );
}
