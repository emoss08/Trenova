/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { FormCreateModal } from "@/components/ui/form-create-modal";
import { locationCategorySchema } from "@/lib/schemas/location-category-schema";
import { type TableSheetProps } from "@/types/data-table";
import { LocationCategoryType } from "@/types/location-category";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { LocationCategoryForm } from "./location-category-form";

export function CreateLocationCategoryModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(locationCategorySchema),
    defaultValues: {
      name: "",
      description: "",
      type: LocationCategoryType.CustomerLocation,
      facilityType: undefined,
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
      url="/location-categories/"
      queryKey="location-category-list"
      className="max-w-[450px]"
    />
  );
}
