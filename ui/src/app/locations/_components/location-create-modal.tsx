import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
  locationSchema,
  type LocationSchema,
} from "@/lib/schemas/location-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { LocationForm } from "./location-form";

export function CreateLocationModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm<LocationSchema>({
    resolver: yupResolver(locationSchema),
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
      state: null,
      // locationCategory: null,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Location"
      formComponent={<LocationForm />}
      form={form}
      schema={locationSchema}
      url="/locations/"
      queryKey="location-list"
    />
  );
}
