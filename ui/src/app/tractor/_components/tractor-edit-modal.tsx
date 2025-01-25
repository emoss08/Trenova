import { FormEditModal } from "@/components/ui/form-edit-model";
import {
  tractorSchema,
  type TractorSchema,
} from "@/lib/schemas/tractor-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { TractorForm } from "./tractor-form";

export function EditTractorModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<TractorSchema>) {
  const form = useForm<TractorSchema>({
    resolver: yupResolver(tractorSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      open={open}
      onOpenChange={onOpenChange}
      url="/tractors/"
      title="Tractor"
      queryKey={["tractor"]}
      formComponent={<TractorForm />}
      fieldKey="code"
      form={form}
      schema={tractorSchema}
    />
  );
}
