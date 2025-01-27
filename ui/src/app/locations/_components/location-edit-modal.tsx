import { FormEditModal } from "@/components/ui/form-edit-model";
import {
  locationSchema,
  type LocationSchema,
} from "@/lib/schemas/location-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { LocationForm } from "./location-form";

export function EditLocationModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<LocationSchema>) {
  const form = useForm<LocationSchema>({
    resolver: yupResolver(locationSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      open={open}
      onOpenChange={onOpenChange}
      title="Location"
      formComponent={<LocationForm />}
      form={form}
      schema={locationSchema}
      url="/locations/"
      queryKey="location-list"
      fieldKey="name"
    />
  );
}
