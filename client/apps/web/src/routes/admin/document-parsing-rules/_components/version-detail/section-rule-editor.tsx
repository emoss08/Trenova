import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { FormGroup, FormControl, FormSection } from "@trenova/shared/components/ui/form";
import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { ChevronDownIcon, PlusIcon, TrashIcon, LayoutListIcon } from "lucide-react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { TagInput } from "../shared/tag-input";
import type { RuleVersionFormValues } from "@/types/document-parsing-rule";

export function SectionRuleEditor() {
  const t = useT();

  const { control } = useFormContext<RuleVersionFormValues>();
  const { fields, append, remove } = useFieldArray({
    control,
    name: "ruleDocument.sections",
  });

  return (
    <FormSection
      title={t("Sections")}
      titleCount={fields.length}
      description={t("Sections divide the document into named regions. Fields and stops can optionally target specific sections.")}
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
          {t("Add Section")}
        </Button>
      }
    >
      <div className="space-y-3">
        {fields.length === 0 && (
          <div className="flex flex-col items-center justify-center gap-2 rounded-md border border-dashed py-8 text-center">
            <LayoutListIcon className="text-muted-foreground/50 size-8" />
            <div>
              <p className="text-muted-foreground text-sm font-medium">{t("No sections defined")}</p>
              <p className="text-muted-foreground/70 mt-0.5 text-xs">
                {t("Sections are optional. Add them to scope field and stop extraction to specific document regions.")}
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
  const t = useT();

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
        <CollapsibleTrigger className="hover:bg-muted/50 flex w-full items-center justify-between p-3 text-sm font-medium">
          <div className="flex items-center gap-2">
            <span>{defaultName || `Section ${index + 1}`}</span>
            {anchorCount > 0 && (
              <Badge variant="secondary">
                {t("{0} anchor{1}", anchorCount, anchorCount !== 1 ? "s" : "")}
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
              <TrashIcon className="text-destructive size-3.5" />
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
                  label={t("Name")}
                  placeholder={t("e.g. header, shipment_details")}
                />
              </FormControl>
              <FormControl>
                <TagInput
                  control={control}
                  name={`ruleDocument.sections.${index}.startAnchors`}
                  label={t("Start Anchors")}
                  description={t("Text strings that mark the beginning of this section")}
                  placeholder={t("Add anchor text...")}
                />
              </FormControl>
              <FormControl>
                <TagInput
                  control={control}
                  name={`ruleDocument.sections.${index}.endAnchors`}
                  label={t("End Anchors")}
                  description={t("Text strings that mark the end of this section")}
                  placeholder={t("Add anchor text...")}
                />
              </FormControl>
              <FormControl className="flex items-end gap-4">
                <SwitchField
                  control={control}
                  name={`ruleDocument.sections.${index}.captureBlankLine`}
                  label={t("Capture Blank Lines")}
                  description={t("Include blank lines within the section boundaries")}
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name={`ruleDocument.sections.${index}.allowMultiple`}
                  label={t("Allow Multiple")}
                  description={t("Allow this section to appear more than once in the document")}
                />
              </FormControl>
            </FormGroup>
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}
