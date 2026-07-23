import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import {
  hazmatSegregationRuleSchema,
  type HazmatSegregationRule,
} from "@/types/hazmat-segregation-rule";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { HazmatSegregationRuleForm } from "./hazmat-segregation-rule-form";

export function HazmatSegregationRulePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<HazmatSegregationRule>) {
  const form = useForm({
    resolver: zodResolver(hazmatSegregationRuleSchema),
    defaultValues: {
      status: "Active",
      name: "",
      description: "",
      classA: "HazardClass3",
      classB: "HazardClass8",
      segregationType: "Prohibited",
      hasExceptions: false,
      exceptionNotes: "",
      referenceCode: "",
      regulationSource: "",
    },
  });

  console.info("Form", form.formState.errors);

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        url="/hazmat-segregation-rules/"
        queryKey="hazmat-segregation-rule-list"
        title="Hazmat Segregation Rule"
        fieldKey="name"
        formComponent={<HazmatSegregationRuleForm />}
        row={row}
        form={form}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/hazmat-segregation-rules/"
      queryKey="hazmat-segregation-rule-list"
      title="Hazmat Segregation Rule"
      formComponent={<HazmatSegregationRuleForm />}
    />
  );
}
