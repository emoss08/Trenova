import { FormEditModal } from "@/components/ui/form-edit-model";
import {
  hazmatSegregationRuleSchema,
  type HazmatSegregationRuleSchema,
} from "@/lib/schemas/hazmat-segregation-rule-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { HazmatSegregationRuleForm } from "./hazmat-segregation-rule-form";

export function EditHazmatSegregationRuleModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<HazmatSegregationRuleSchema>) {
  const form = useForm<HazmatSegregationRuleSchema>({
    resolver: yupResolver(hazmatSegregationRuleSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      open={open}
      onOpenChange={onOpenChange}
      url="/hazmat-segregation-rules/"
      title="Hazmat Segregation Rule"
      queryKey="hazmat-segregation-rule-list"
      formComponent={<HazmatSegregationRuleForm />}
      fieldKey="name"
      className="max-w-[500px]"
      form={form}
      schema={hazmatSegregationRuleSchema}
    />
  );
}
