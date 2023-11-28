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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { toast } from "@/components/ui/use-toast";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { cn } from "@/lib/utils";
import { locationSchema } from "@/lib/validations/location";
import { useTableStore } from "@/stores/TableStore";
import { LocationFormValues as FormValues, Location } from "@/types/location";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useState } from "react";
import { useForm } from "react-hook-form";
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
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(locationSchema),
    defaultValues: {
      status: location.status,
      code: location.code,
      locationCategory: location?.locationCategory || "",
      name: location.name,
      depot: location?.depot || "",
      addressLine1: location.addressLine1,
      addressLine2: location?.addressLine2 || "",
      city: location.city,
      state: location.state,
      zipCode: location.zipCode,
      locationComments: location.locationComments,
      locationContacts: location.locationContacts,
      description: location.description,
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    toast,
    {
      method: "PUT",
      path: `/locations/${location.id}/`,
      successMessage: "Location updated successfully.",
      queryKeysToInvalidate: ["locations-table-data"],
      closeModal: true,
      errorMessage: "Failed to update new location.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex flex-col h-full overflow-y-auto"
    >
      <Tabs defaultValue="info" className="flex-1 w-full">
        <TabsList>
          <TabsTrigger value="info">Information</TabsTrigger>
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

export function LocationTableEditSheet({
  onOpenChange,
  open,
}: TableSheetProps) {
  const [location] = useTableStore.use("currentRecord") as Location[];

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>{location && location.name}</SheetTitle>
          <SheetDescription>
            Last updated on {location && formatDate(location.modified)}
          </SheetDescription>
        </SheetHeader>
        {location && (
          <LocationEditForm
            location={location}
            open={open}
            onOpenChange={onOpenChange}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}
