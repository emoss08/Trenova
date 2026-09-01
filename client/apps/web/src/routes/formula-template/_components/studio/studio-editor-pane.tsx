import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { ExpressionEditor } from "@/components/formula-editor/expression-editor";
import type { KnownIdentifiers } from "@/components/formula-editor/known-identifiers";
import { VariableDefinitionEditor } from "@/components/formula-editor/variable-definition-editor";
import { formulaTemplateTypeChoices } from "@/lib/choices";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Separator } from "@trenova/shared/components/ui/separator";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import type { FormulaTemplateFormValues } from "@trenova/shared/types/formula-template";
import {
  ChevronDownIcon,
  CodeIcon,
  FileCode2,
  ShieldCheckIcon,
  SparklesIcon,
  WandSparklesIcon,
} from "lucide-react";
import { useState, type Ref } from "react";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import { BreakdownDefinitionEditor } from "../breakdown-definition-editor";
import { AiExplainPanel } from "./ai/ai-explain-panel";
import { StarterTemplatePicker } from "./starter-template-picker";

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
      <div className="bg-primary/10 text-primary flex size-8 shrink-0 items-center justify-center rounded-lg">
        <Icon className="size-4" />
      </div>
      <div>
        <h3 className="text-sm leading-none font-semibold tracking-tight">{title}</h3>
        <p className="text-muted-foreground mt-1 text-xs">{description}</p>
      </div>
    </div>
  );
}

type StudioEditorPaneProps = {
  mode: "create" | "edit";
  known: KnownIdentifiers;
  editorRef: Ref<ReactCodeMirrorRef>;
  onOpenAiGenerate: () => void;
};

export function StudioEditorPane({
  mode,
  known,
  editorRef,
  onOpenAiGenerate,
}: StudioEditorPaneProps) {
  const { control, register } = useFormContext<FormulaTemplateFormValues>();
  const [detailsOpen, setDetailsOpen] = useState(true);

  const expression = useWatch({ control, name: "expression" });
  const schemaId = useWatch({ control, name: "schemaId" });
  const status = useWatch({ control, name: "status" });

  return (
    <ScrollArea className="h-full">
      <div className="space-y-4 p-4">
        <Collapsible open={detailsOpen} onOpenChange={setDetailsOpen}>
          <CollapsibleTrigger
            render={
              <button type="button" className="flex w-full items-center justify-between">
                <SectionHeader
                  icon={FileCode2}
                  title="Template Details"
                  description="Name, type, and description"
                />
                <ChevronDownIcon
                  className={`text-muted-foreground size-4 transition-transform ${
                    detailsOpen ? "rotate-180" : ""
                  }`}
                />
              </button>
            }
          />
          <CollapsibleContent>
            <FormGroup cols={2} className="pt-3">
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
                  placeholder="Describe when this template applies and how it prices"
                  rows={2}
                />
              </FormControl>
            </FormGroup>
          </CollapsibleContent>
        </Collapsible>

        <Controller
          name="schemaId"
          control={control}
          render={({ field }) => <Input type="hidden" {...field} />}
        />
        <Controller
          name="status"
          control={control}
          render={({ field }) => <Input type="hidden" {...field} value={field.value ?? "Draft"} />}
        />

        <Separator />

        <div className="flex items-center justify-between gap-2">
          <SectionHeader
            icon={CodeIcon}
            title="Expression"
            description="The formula that computes the charge"
          />
          <div className="flex items-center gap-1.5">
            {mode === "edit" && status && (
              <Badge variant="outline" className="text-2xs">
                {status}
              </Badge>
            )}
            <Button
              type="button"
              variant="outline"
              size="xs"
              onClick={onOpenAiGenerate}
              className="gap-1.5"
            >
              <WandSparklesIcon className="size-3" />
              Generate with AI
            </Button>
          </div>
        </div>

        {mode === "create" && !expression?.trim() && <StarterTemplatePicker />}

        <ExpressionEditor
          name="expression"
          control={control}
          rules={{ required: true }}
          placeholder="e.g., round(baseRate * totalDistance, 2)"
          height="240px"
          knownIdentifiers={known}
          editorRef={editorRef}
        />
        <div className="flex items-start justify-between gap-3">
          <p className="text-muted-foreground text-2xs flex items-center gap-1">
            <SparklesIcon className="size-3" />
            Ctrl+Space for autocomplete. Click a variable in the reference to insert it.
          </p>
        </div>
        <AiExplainPanel expression={expression ?? ""} schemaId={schemaId || "shipment"} />

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
        <VariableDefinitionEditor control={control as never} register={register as never} />

        <Separator />
        <BreakdownDefinitionEditor
          control={control as never}
          register={register as never}
          knownIdentifiers={known}
        />
      </div>
    </ScrollArea>
  );
}
