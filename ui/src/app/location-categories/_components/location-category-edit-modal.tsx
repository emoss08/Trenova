import { FormEditModal } from "@/components/ui/form-edit-model";
import {
    locationCategorySchema,
    LocationCategorySchema,
} from "@/lib/schemas/location-category-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { LocationCategoryForm } from "./location-category-form";

export function EditLocationCategoryModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<LocationCategorySchema>) {
  const form = useForm<LocationCategorySchema>({
    resolver: yupResolver(locationCategorySchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      open={open}
      onOpenChange={onOpenChange}
      url="/location-categories/"
      title="Location Category"
      queryKey="location-category-list"
      formComponent={<LocationCategoryForm />}
      fieldKey="name"
      className="max-w-[450px]"
      form={form}
      schema={locationCategorySchema}
    />
  );
}
