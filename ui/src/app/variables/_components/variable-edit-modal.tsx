import { FormEditModal } from "@/components/ui/form-edit-modal";
import { variableSchema, VariableSchema } from "@/lib/schemas/variable-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { VariableForm } from "./variable-form";

export function EditVariableModal({
  currentRecord,
}: EditTableSheetProps<VariableSchema>) {
  const form = useForm({
    resolver: zodResolver(variableSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/variables/"
      title="Variable"
      className="sm:max-w-[600px]"
      queryKey="variable-list"
      formComponent={<VariableForm />}
      fieldKey="displayName"
      form={form}
    />
  );
}
