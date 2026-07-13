import { ExpressionTestPanel } from "@/components/formula-editor/expression-test-panel";
import type { FormulaTemplateFormValues } from "@/types/formula-template";
import { useWatch, type UseFormReturn } from "react-hook-form";

export default function FormulaTemplateTestTab({
  form,
}: {
  form: UseFormReturn<FormulaTemplateFormValues>;
}) {
  const expression = useWatch({ control: form.control, name: "expression" });
  const schemaId = useWatch({ control: form.control, name: "schemaId" });
  const customVariables = useWatch({ control: form.control, name: "variableDefinitions" });
  const breakdowns = useWatch({ control: form.control, name: "breakdownDefinitions" });

  return (
    <ExpressionTestPanel
      expression={expression}
      schemaId={schemaId}
      customVariables={customVariables}
      breakdowns={breakdowns}
    />
  );
}
