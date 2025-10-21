import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
  variableFormatSchema,
  VariableValueType,
} from "@/lib/schemas/variable-schema";
import { TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { FormatForm } from "./format-form";

export function CreateFormatModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(variableFormatSchema),
    defaultValues: {
      isActive: true,
      name: "",
      description: "",
      formatSql: "",
      valueType: VariableValueType.enum.String,
      isSystem: false,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Format"
      className="sm:max-w-[600px]"
      formComponent={<FormatForm />}
      form={form}
      url="/variable-formats/"
      queryKey="format-list"
    />
  );
}
