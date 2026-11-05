import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { hazmatSegregationRuleService } from "@/services/hazmat-segregation-rule";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
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
  const t = useT();

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

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        queryKey="hazmat-segregation-rule-list"
        title={t("Hazmat segregation rule")}
        fieldKey="name"
        formComponent={<HazmatSegregationRuleForm />}
        mutationFn={(values, currentRow) => {
          if (!currentRow.id) {
            throw new Error("No Hazmat Segregation Rule ID selected");
          }

          return hazmatSegregationRuleService.update(currentRow.id, values);
        }}
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
      queryKey="hazmat-segregation-rule-list"
      title={t("Hazmat segregation rule")}
      formComponent={<HazmatSegregationRuleForm />}
      mutationFn={(values) => hazmatSegregationRuleService.create(values)}
    />
  );
}
