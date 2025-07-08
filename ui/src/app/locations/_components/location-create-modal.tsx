import { FormCreateModal } from "@/components/ui/form-create-modal";
import { locationSchema } from "@/lib/schemas/location-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { LocationForm } from "./location-form";

export function CreateLocationModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(locationSchema),
    defaultValues: {
      status: Status.Active,
      name: "",
      code: "",
      description: "",
      addressLine1: "",
      addressLine2: "",
      city: "",
      postalCode: "",
      isGeocoded: false,
      locationCategoryId: "",
      stateId: "",
      state: undefined,
      locationCategory: undefined,
      latitude: undefined,
      longitude: undefined,
      placeId: undefined,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Location"
      formComponent={<LocationForm />}
      form={form}
      url="/locations/"
      queryKey="location-list"
    />
  );
}
