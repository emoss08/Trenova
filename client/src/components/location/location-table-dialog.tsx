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
import { cn } from "@/lib/utils";
import { LocationSchema } from "@/lib/validations/LocationSchema";
import { type LocationFormValues as FormValues } from "@/types/location";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { LocationCommentForm } from "./location-comments-form";

export function LocationTableSheet({ onOpenChange, open }: TableSheetProps) {
  const { control, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(LocationSchema),
    defaultValues: {
      status: "A",
      code: "",
      locationCategoryId: "",
      name: "",
      addressLine1: "",
      addressLine2: "",
      city: "",
      stateId: "",
      postalCode: "",
      description: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/locations/",
    successMessage: "Location created successfully.",
    queryKeysToInvalidate: "locations",
    closeModal: true,
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
