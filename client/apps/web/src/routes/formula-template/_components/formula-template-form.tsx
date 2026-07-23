import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { ExpressionEditor } from "@/components/formula-editor/expression-editor";
import { VariableDefinitionEditor } from "@/components/formula-editor/variable-definition-editor";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Input } from "@trenova/shared/components/ui/input";
import { Separator } from "@trenova/shared/components/ui/separator";
import { formulaTemplateStatusChoices, formulaTemplateTypeChoices } from "@/lib/choices";
import type { FormulaTemplateFormValues } from "@trenova/shared/types/formula-template";
import { CodeIcon, FileCode2, ShieldCheckIcon } from "lucide-react";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import { BreakdownDefinitionEditor } from "./breakdown-definition-editor";

const statusSelectChoices = formulaTemplateStatusChoices.map((choice) =>
  choice.value === "InReview" ? { ...choice, disabled: true } : choice,
);

function SectionHeader({
  icon: Icon,
  title,
  description,
}: {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  description: string;
}) {
  return (
    <div className="flex items-center gap-3">
      <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
        <Icon className="size-4" />
      </div>
      <div>
        <h3 className="text-sm leading-none font-semibold tracking-tight">{title}</h3>
        <p className="mt-1 text-xs text-muted-foreground">{description}</p>
      </div>
    </div>
  );
}

export function FormulaTemplateForm() {
  const { control, register } = useFormContext<FormulaTemplateFormValues>();

  const customVariables = useWatch({ control, name: "variableDefinitions" });

  return (
    <div className="space-y-4">
      <SectionHeader
        icon={FileCode2}
        title="Template Details"
        description="Basic information about the formula template"
      />
      <FormGroup cols={3}>
        <FormControl>
          <SelectField
            label="Status"
            name="status"
            control={control}
            rules={{ required: true }}
            options={statusSelectChoices}
          />
        </FormControl>
        <FormControl>
          <InputField
            label="Name"
            name="name"
            control={control}
            rules={{ required: true }}
            placeholder="Enter template name"
          />
        </FormControl>
        <FormControl>
          <SelectField
            label="Type"
            name="type"
            control={control}
            rules={{ required: true }}
            options={formulaTemplateTypeChoices}
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            label="Description"
            name="description"
            control={control}
            rules={{ required: true }}
            placeholder="Describe the formula template"
            rows={3}
          />
        </FormControl>
      </FormGroup>
      <Controller
        name="schemaId"
        control={control}
        render={({ field }) => <Input type="hidden" {...field} />}
      />
      <Separator />
      <SectionHeader
        icon={CodeIcon}
        title="Expression"
        description="Define the formula logic using variables and functions"
      />
      <ExpressionEditor
        name="expression"
        control={control}
        rules={{ required: true }}
        placeholder="e.g., totalDistance * 2.5"
        height="220px"
        customVariables={customVariables}
      />
      <p className="text-xs text-muted-foreground">
        Press Ctrl+Space for autocomplete. Use the Testing tab to validate.
      </p>

      <Separator />
      <SectionHeader
        icon={ShieldCheckIcon}
        title="Guardrails"
        description="Clamp the calculated charge to a minimum and maximum amount"
      />
      <FormGroup cols={2}>
        <FormControl>
          <NumberField
            label="Minimum Charge"
            name="minCharge"
            control={control}
            placeholder="No minimum"
            sideText="$"
            decimalScale={2}
            thousandSeparator
          />
        </FormControl>
        <FormControl>
          <NumberField
            label="Maximum Charge"
            name="maxCharge"
            control={control}
            placeholder="No maximum"
            sideText="$"
            decimalScale={2}
            thousandSeparator
          />
        </FormControl>
      </FormGroup>

      <Separator />
      <VariableDefinitionEditor control={control as any} register={register as any} />

      <Separator />
      <BreakdownDefinitionEditor control={control as any} register={register as any} />
    </div>
  );
}
