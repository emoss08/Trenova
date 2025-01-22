import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
  locationCategorySchema,
  type LocationCategorySchema,
} from "@/lib/schemas/location-category-schema";
import { type TableSheetProps } from "@/types/data-table";
import { FacilityType, LocationCategoryType } from "@/types/location-category";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { LocationCategoryForm } from "./location-category-form";

export function CreateLocationCategoryModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm<LocationCategorySchema>({
    resolver: yupResolver(locationCategorySchema),
    defaultValues: {
      name: "",
      description: "",
      type: LocationCategoryType.CustomerLocation,
      facilityType: FacilityType.None,
      hasRestroom: false,
      requiresAppointment: false,
      hasSecureParking: false,
      allowsOvernight: false,
      color: "",
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Location Category"
      formComponent={<LocationCategoryForm />}
      form={form}
      schema={locationCategorySchema}
      url="/location-categories/"
      queryKey="location-category-list"
      className="max-w-[450px]"
    />
  );
}
