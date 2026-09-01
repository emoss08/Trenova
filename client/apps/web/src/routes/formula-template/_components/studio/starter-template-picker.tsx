import { Button } from "@trenova/shared/components/ui/button";
import type { FormulaTemplateFormValues } from "@trenova/shared/types/formula-template";
import { LayoutTemplateIcon } from "lucide-react";
import { useFormContext } from "react-hook-form";
import { STARTER_TEMPLATES, type StarterTemplate } from "./starter-templates";

export function StarterTemplatePicker() {
  const { setValue } = useFormContext<FormulaTemplateFormValues>();

  const applyStarter = (starter: StarterTemplate) => {
    setValue("expression", starter.values.expression, { shouldDirty: true });
    setValue("variableDefinitions", starter.values.variableDefinitions ?? [], {
      shouldDirty: true,
    });
    setValue("breakdownDefinitions", starter.values.breakdownDefinitions ?? [], {
      shouldDirty: true,
    });
    setValue("minCharge", starter.values.minCharge ?? null, { shouldDirty: true });
    setValue("maxCharge", starter.values.maxCharge ?? null, { shouldDirty: true });
  };

  return (
    <div className="bg-muted/30 space-y-2 rounded-lg border p-3">
      <div className="text-muted-foreground flex items-center gap-2 text-xs font-medium tracking-wide uppercase">
        <LayoutTemplateIcon className="size-3" />
        Start from an example
      </div>
      <div className="grid grid-cols-2 gap-2">
        {STARTER_TEMPLATES.map((starter) => (
          <Button
            key={starter.id}
            type="button"
            variant="outline"
            onClick={() => applyStarter(starter)}
            className="h-auto flex-col items-start gap-0.5 px-3 py-2 text-left whitespace-normal"
          >
            <span className="text-xs font-semibold">{starter.name}</span>
            <span className="text-muted-foreground text-2xs font-normal">
              {starter.description}
            </span>
          </Button>
        ))}
      </div>
    </div>
  );
}
