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
