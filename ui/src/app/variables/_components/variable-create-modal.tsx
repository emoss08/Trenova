import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
  VariableContext,
  variableSchema,
  VariableValueType,
} from "@/lib/schemas/variable-schema";
import { TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { VariableForm } from "./variable-form";

export function CreateVariableModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(variableSchema),
    defaultValues: {
      isActive: true,
      description: "",
      appliesTo: VariableContext.enum.Customer,
      displayName: "",
      category: "",
      formatId: undefined,
      isValidated: false,
      tags: [],
      requiredParams: [],
      defaultValue: "",
      key: "",
      query: "",
      valueType: VariableValueType.enum.String,
      isSystem: false,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Variable"
      className="sm:max-w-[600px]"
      formComponent={<VariableForm />}
      form={form}
      url="/variables/"
      queryKey="variable-list"
    />
  );
}
