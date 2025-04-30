import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  tractorSchema,
  type TractorSchema,
} from "@/lib/schemas/tractor-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { TractorForm } from "./tractor-form";

export function EditTractorModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<TractorSchema>) {
  const form = useForm({
    resolver: zodResolver(tractorSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      open={open}
      onOpenChange={onOpenChange}
      url="/tractors/"
      title="Tractor"
      queryKey="tractor-list"
      formComponent={<TractorForm />}
      fieldKey="code"
      form={form}
      className="max-w-[500px]"
    />
  );
}
