import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  locationSchema,
  type LocationSchema,
} from "@/lib/schemas/location-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { LocationForm } from "./location-form";

export function EditLocationModal({
  currentRecord,
}: EditTableSheetProps<LocationSchema>) {
  const form = useForm({
    resolver: zodResolver(locationSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      title="Location"
      formComponent={<LocationForm />}
      form={form}
      url="/locations/"
      queryKey="location-list"
      fieldKey="name"
    />
  );
}
