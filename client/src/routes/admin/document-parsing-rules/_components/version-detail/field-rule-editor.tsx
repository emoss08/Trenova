import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { FormSection, FormGroup, FormControl } from "@/components/ui/form";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  ChevronDownIcon,
  PlusIcon,
  TrashIcon,
  TextCursorInputIcon,
} from "lucide-react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { TagInput } from "../shared/tag-input";
import type { RuleVersionFormValues } from "@/types/document-parsing-rule";

export function FieldRuleEditor() {
  const { control } = useFormContext<RuleVersionFormValues>();
  const { fields, append, remove } = useFieldArray({
    control,
    name: "ruleDocument.fields",
  });

  return (
    <FormSection
      title="Fields"
      titleCount={fields.length}
      description="Fields define individual data points to extract from the document."
      action={
        <Button
          type="button"
          variant="outline"
          size="xxs"
          className="gap-1"
          onClick={() =>
            append({
              key: "",
              label: "",
              sectionNames: [],
              aliases: [],
              patterns: [],
              normalizer: "",
              required: false,
              confidence: 0,
            })
          }
        >
          <PlusIcon className="size-3" />
          Add Field
        </Button>
      }
    >
      {fields.length === 0 && (
        <div className="flex flex-col items-center justify-center gap-2 rounded-md border border-dashed py-8 text-center">
          <TextCursorInputIcon className="size-8 text-muted-foreground/50" />
          <div>
            <p className="text-sm font-medium text-muted-foreground">
              No field rules defined
            </p>
            <p className="mt-0.5 text-xs text-muted-foreground/70">
              Add fields to extract specific values like PRO numbers,
              reference IDs, or dates from documents.
            </p>
          </div>
        </div>
      )}
      {fields.map((field, index) => (
        <FieldItem
          key={field.id}
          index={index}
          defaultKey={field.key}
          defaultLabel={field.label}
          onRemove={() => remove(index)}
        />
      ))}
    </FormSection>
  );
}

function FieldItem({
  index,
  defaultKey,
  defaultLabel,
  onRemove,
}: {
  index: number;
  defaultKey: string;
  defaultLabel: string;
  onRemove: () => void;
}) {
  const { control } = useFormContext<RuleVersionFormValues>();

  const isRequired = useWatch({
    control,
    name: `ruleDocument.fields.${index}.required`,
  });
  const confidence = useWatch({
    control,
    name: `ruleDocument.fields.${index}.confidence`,
  });

  return (
    <Collapsible defaultOpen={!defaultKey}>
      <div className="rounded-md border">
        <CollapsibleTrigger className="flex w-full items-center justify-between p-3 text-sm font-medium hover:bg-muted/50">
          <div className="flex items-center gap-2">
            <span>
              {defaultKey
                ? `${defaultKey} — ${defaultLabel}`
                : `Field ${index + 1}`}
            </span>
            {isRequired && <Badge variant="active">Required</Badge>}
            {typeof confidence === "number" && confidence > 0 && (
              <Badge variant="outline">{Math.round(confidence * 100)}%</Badge>
            )}
          </div>
          <div className="flex items-center gap-1">
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="size-7"
              onClick={(e) => {
                e.stopPropagation();
                onRemove();
              }}
            >
              <TrashIcon className="size-3.5 text-destructive" />
            </Button>
            <ChevronDownIcon className="size-4 transition-transform [[data-state=open]>&]:rotate-180" />
          </div>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="border-t p-3">
            <FormGroup cols={2}>
              <FormControl>
                <InputField
                  control={control}
                  name={`ruleDocument.fields.${index}.key`}
                  label="Key"
                  placeholder="e.g. pro_number"
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={control}
                  name={`ruleDocument.fields.${index}.label`}
                  label="Label"
                  placeholder="e.g. PRO Number"
                />
              </FormControl>
              <FormControl>
                <TagInput
                  control={control}
                  name={`ruleDocument.fields.${index}.sectionNames`}
                  label="Section Names"
                  description="Restrict extraction to these document sections"
                  placeholder="Limit to sections..."
                />
              </FormControl>
              <FormControl>
                <TagInput
                  control={control}
                  name={`ruleDocument.fields.${index}.aliases`}
                  label="Aliases"
                  description="Alternative labels the parser will also recognize"
                  placeholder="Add alias..."
                />
              </FormControl>
              <FormControl cols={2}>
                <TagInput
                  control={control}
                  name={`ruleDocument.fields.${index}.patterns`}
                  label="Patterns"
                  description="Regex patterns for extraction. The first capture group is used as the value."
                  placeholder="Add regex pattern..."
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={control}
                  name={`ruleDocument.fields.${index}.normalizer`}
                  label="Normalizer"
                  description="Post-extraction transformation (e.g. trim, uppercase)"
                  placeholder="e.g. trim, uppercase"
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name={`ruleDocument.fields.${index}.confidence`}
                  label="Min Confidence"
                  description="Minimum confidence threshold (0.0 to 1.0)"
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name={`ruleDocument.fields.${index}.required`}
                  label="Required"
                  description="Flag for review if this field is missing from the result"
                />
              </FormControl>
            </FormGroup>
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}
