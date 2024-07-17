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
import { formatToUserTimezone } from "@/lib/date";
import { cn } from "@/lib/utils";
import { LocationSchema } from "@/lib/validations/LocationSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  LocationFormValues as FormValues,
  Location,
} from "@/types/location";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { Badge } from "../ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../ui/new/new-tabs";
import { LocationCommentForm } from "./location-comments-form";

export function LocationEditForm({
  location,
  open,
  onOpenChange,
}: {
  location: Location;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { control, handleSubmit, reset } = useForm<FormValues>({
    resolver: yupResolver(LocationSchema),
    defaultValues: location,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/locations/${location.id}/`,
    successMessage: "Location updated successfully.",
    queryKeysToInvalidate: "locations",
    closeModal: true,
    reset,
    errorMessage: "Failed to update new location.",
  });

  const onSubmit = (values: FormValues) => {
    // For each comment append the location id value
    values.comments = values.comments ?? [];
    values.comments = values.comments.map((comment) => ({
      ...comment,
      locationId: location.id,
    }));

    values.contacts = values.contacts ?? [];
    values.contacts = values.contacts.map((contact) => ({
      ...contact,
      locationId: location.id,
    }));

    console.info("LocationEditForm.onSubmit", values);

    mutation.mutate(values);
  };

  return (
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
        <Button type="submit" isLoading={mutation.isPending} className="w-full">
          Save
        </Button>
      </SheetFooter>
    </form>
  );
}

export function LocationTableEditSheet({
  onOpenChange,
  open,
}: TableSheetProps) {
  const [location] = useTableStore.use("currentRecord") as Location[];

  if (!location) return null;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>
            <span>{location.name}</span>
            <Badge className="ml-5" variant="purple">
              {location.id}
            </Badge>
          </SheetTitle>
          <SheetDescription>
            Last updated on {formatToUserTimezone(location.updatedAt)}
          </SheetDescription>
        </SheetHeader>
        <LocationEditForm
          location={location}
          open={open}
          onOpenChange={onOpenChange}
        />
      </SheetContent>
    </Sheet>
  );
}
