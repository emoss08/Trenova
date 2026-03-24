import { ExpressionTestPanel } from "@/components/formula-editor/expression-test-panel";
import type { FormulaTemplateFormValues } from "@/types/formula-template";
import type { UseFormReturn } from "react-hook-form";

export default function FormulaTemplateTestTab({
  form,
}: {
  form: UseFormReturn<FormulaTemplateFormValues>;
}) {
  const expression = form.watch("expression");
  const schemaId = form.watch("schemaId");
  const customVariables = form.watch("variableDefinitions");

  return (
    <ExpressionTestPanel
      expression={expression}
      schemaId={schemaId}
      customVariables={customVariables}
    />
  );
}
