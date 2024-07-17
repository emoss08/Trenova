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

import { LocationContactForm } from "@/components/location/location-contacts-form";
import { LocationInfoForm } from "@/components/location/location-info-form";
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
import { LocationSchema } from "@/lib/validations/LocationSchema";
import { type LocationFormValues as FormValues } from "@/types/location";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../ui/new/new-tabs";
import { LocationCommentForm } from "./location-comments-form";

export function LocationTableSheet({ onOpenChange, open }: TableSheetProps) {
  const { control, handleSubmit, reset } = useForm<FormValues>({
    resolver: yupResolver(LocationSchema),
    defaultValues: {
      status: "Active",
      code: undefined,
      locationCategoryId: undefined,
      name: "",
      addressLine1: "",
      addressLine2: "",
      city: "",
      stateId: undefined,
      postalCode: "",
      description: "",
      comments: [],
      contacts: [],
    },
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/locations/",
    successMessage: "Location created successfully.",
    queryKeysToInvalidate: "locations",
    closeModal: true,
    reset,
    errorMessage: "Failed to create new location.",
  });

  const onSubmit = (values: FormValues) => mutation.mutate(values);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>Add New Location</SheetTitle>
          <SheetDescription>
            Use this form to add a new location to the system.
          </SheetDescription>
        </SheetHeader>
        <form
          onSubmit={handleSubmit(onSubmit)}
          className="flex h-full flex-col overflow-y-auto"
        >
          <Tabs defaultValue="info" className="mt-10 w-full flex-1">
            <TabsList className="mx-auto space-x-4">
              <TabsTrigger value="info">General Information</TabsTrigger>
              <TabsTrigger value="comments">Comments</TabsTrigger>
              <TabsTrigger value="contacts">Contacts</TabsTrigger>
            </TabsList>
            <TabsContent value="info">
              <LocationInfoForm control={control} open={open} />
            </TabsContent>
            <TabsContent value="comments">
              <LocationCommentForm control={control} />
            </TabsContent>
            <TabsContent value="contacts">
              <LocationContactForm control={control} />
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
              isLoading={mutation.isPending}
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
