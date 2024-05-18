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
      <Tabs defaultValue="info" className="w-full flex-1">
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

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>{location && location.name}</SheetTitle>
          <SheetDescription>
            Last updated on{" "}
            {location && formatToUserTimezone(location.updatedAt)}
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
