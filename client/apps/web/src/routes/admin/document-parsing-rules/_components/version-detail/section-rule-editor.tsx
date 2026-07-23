import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { FormGroup, FormControl, FormSection } from "@/components/ui/form";
import { InputField } from "@/components/fields/input-field";
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
  LayoutListIcon,
} from "lucide-react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { TagInput } from "../shared/tag-input";
import type { RuleVersionFormValues } from "@/types/document-parsing-rule";

export function SectionRuleEditor() {
  const { control } = useFormContext<RuleVersionFormValues>();
  const { fields, append, remove } = useFieldArray({
    control,
    name: "ruleDocument.sections",
  });

  return (
    <FormSection
      title="Sections"
      titleCount={fields.length}
      description="Sections divide the document into named regions. Fields and stops can optionally target specific sections."
      action={
        <Button
          type="button"
          variant="outline"
          size="xxs"
          className="gap-1"
          onClick={() =>
            append({
              name: "",
              startAnchors: [],
              endAnchors: [],
              captureBlankLine: false,
              allowMultiple: false,
            })
          }
        >
          <PlusIcon className="size-3" />
          Add Section
        </Button>
      }
    >
      <div className="space-y-3">
        {fields.length === 0 && (
          <div className="flex flex-col items-center justify-center gap-2 rounded-md border border-dashed py-8 text-center">
            <LayoutListIcon className="size-8 text-muted-foreground/50" />
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                No sections defined
              </p>
              <p className="mt-0.5 text-xs text-muted-foreground/70">
                Sections are optional. Add them to scope field and stop
                extraction to specific document regions.
              </p>
            </div>
          </div>
        )}
        {fields.map((field, index) => (
          <SectionItem
            key={field.id}
            index={index}
            defaultName={field.name}
            onRemove={() => remove(index)}
          />
        ))}
      </div>
    </FormSection>
  );
}

function SectionItem({
  index,
  defaultName,
  onRemove,
}: {
  index: number;
  defaultName: string;
  onRemove: () => void;
}) {
  const { control } = useFormContext<RuleVersionFormValues>();

  const startAnchors = useWatch({
    control,
    name: `ruleDocument.sections.${index}.startAnchors`,
  });
  const endAnchors = useWatch({
    control,
    name: `ruleDocument.sections.${index}.endAnchors`,
  });

  const anchorCount =
    (Array.isArray(startAnchors) ? startAnchors.length : 0) +
    (Array.isArray(endAnchors) ? endAnchors.length : 0);

  return (
    <Collapsible defaultOpen={!defaultName}>
      <div className="rounded-md border">
        <CollapsibleTrigger className="flex w-full items-center justify-between p-3 text-sm font-medium hover:bg-muted/50">
          <div className="flex items-center gap-2">
            <span>{defaultName || `Section ${index + 1}`}</span>
            {anchorCount > 0 && (
              <Badge variant="secondary">
                {anchorCount} anchor{anchorCount !== 1 ? "s" : ""}
              </Badge>
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
                  name={`ruleDocument.sections.${index}.name`}
                  label="Name"
                  placeholder="e.g. header, shipment_details"
                />
              </FormControl>
              <FormControl>
                <TagInput
                  control={control}
                  name={`ruleDocument.sections.${index}.startAnchors`}
                  label="Start Anchors"
                  description="Text strings that mark the beginning of this section"
                  placeholder="Add anchor text..."
                />
              </FormControl>
              <FormControl>
                <TagInput
                  control={control}
                  name={`ruleDocument.sections.${index}.endAnchors`}
                  label="End Anchors"
                  description="Text strings that mark the end of this section"
                  placeholder="Add anchor text..."
                />
              </FormControl>
              <FormControl className="flex items-end gap-4">
                <SwitchField
                  control={control}
                  name={`ruleDocument.sections.${index}.captureBlankLine`}
                  label="Capture Blank Lines"
                  description="Include blank lines within the section boundaries"
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name={`ruleDocument.sections.${index}.allowMultiple`}
                  label="Allow Multiple"
                  description="Allow this section to appear more than once in the document"
                />
              </FormControl>
            </FormGroup>
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}
