import { ColorOptionValue } from "@/components/fields/select-components";
import { formulaTemplateStatusChoices } from "@/lib/choices";
import { useT } from "@trenova/shared/i18n/use-t";
import type { FormulaLibraryStatus } from "@trenova/shared/types/formula-library";

/** A library's or a template version's review state, as the tables show it. */
export function StatusLabel({ status }: { status: FormulaLibraryStatus }) {
  const t = useT();
  const choice = formulaTemplateStatusChoices.find((option) => option.value === status);
  return (
    <ColorOptionValue
      color={choice?.color}
      value={choice ? t(choice.label) : status}
      className="text-xs"
    />
  );
}
