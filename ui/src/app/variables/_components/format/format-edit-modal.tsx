import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  variableFormatSchema,
  VariableFormatSchema,
} from "@/lib/schemas/variable-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { FormatForm } from "./format-form";

export function EditFormatModal({
  currentRecord,
}: EditTableSheetProps<VariableFormatSchema>) {
  const form = useForm({
    resolver: zodResolver(variableFormatSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/variable-formats/"
      title="Format"
      className="sm:max-w-[600px]"
      queryKey="format-list"
      formComponent={<FormatForm />}
      fieldKey="name"
      form={form}
    />
  );
}
