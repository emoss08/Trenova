import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { FormSection, FormGroup, FormControl } from "@trenova/shared/components/ui/form";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { Separator } from "@trenova/shared/components/ui/separator";
import { ChevronDownIcon, PlusIcon, TrashIcon, MapPinIcon } from "lucide-react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { TagInput } from "../shared/tag-input";
import type { RuleVersionFormValues } from "@/types/document-parsing-rule";

const ROLE_OPTIONS = [
  { value: "pickup", label: "Pickup" },
  { value: "delivery", label: "Delivery" },
  { value: "stop", label: "Stop" },
];

const ROLE_BADGE_VARIANT = {
  pickup: "active",
  delivery: "info",
  stop: "secondary",
} as const;

const FIELD_KEY_OPTIONS = [
  { value: "name", label: "Name" },
  { value: "addressLine1", label: "Address Line 1" },
  { value: "addressLine2", label: "Address Line 2" },
  { value: "city", label: "City" },
  { value: "state", label: "State" },
  { value: "postalCode", label: "Postal Code" },
  { value: "date", label: "Date" },
  { value: "timeWindow", label: "Time Window" },
];

export function StopRuleEditor() {
  const t = useT();

  const { control } = useFormContext<RuleVersionFormValues>();
  const { fields, append, remove } = useFieldArray({
    control,
    name: "ruleDocument.stops",
  });

  return (
    <FormSection
      title={t("Stops")}
      titleCount={fields.length}
      description={t("Stops represent physical locations in a shipment — pickup points, delivery destinations, or intermediate stops. Each stop defines how to extract address and scheduling details.")}
      action={
        <Button
          type="button"
          variant="outline"
          size="xxs"
          className="gap-1"
          onClick={() =>
            append({
              role: "pickup",
              required: false,
              sectionNames: [],
              startAnchors: [],
              endAnchors: [],
              allowMultiple: false,
              sequenceStart: 0,
              extractors: [
                {
                  fieldKey: "name",
                  aliases: [],
                  patterns: [],
                  normalizer: "",
                  confidence: 0,
                  required: false,
                },
              ],
              appointmentPatterns: [],
            })
          }
        >
          <PlusIcon className="size-3" />
          {t("Add Stop")}
        </Button>
      }
    >
      {fields.length === 0 && (
        <div className="flex flex-col items-center justify-center gap-2 rounded-md border border-dashed py-8 text-center">
          <MapPinIcon className="text-muted-foreground/50 size-8" />
          <div>
            <p className="text-muted-foreground text-sm font-medium">{t("No stop rules defined")}</p>
            <p className="text-muted-foreground/70 mt-0.5 text-xs">
              {t("Add stops to extract pickup and delivery locations from the document.")}
            </p>
          </div>
        </div>
      )}
      {fields.map((field, index) => (
        <StopItem
          key={field.id}
          index={index}
          defaultRole={field.role}
          onRemove={() => remove(index)}
        />
      ))}
    </FormSection>
  );
}

function StopItem({
  index,
  defaultRole,
  onRemove,
}: {
  index: number;
  defaultRole: string;
  onRemove: () => void;
}) {
  const t = useT();

  const { control } = useFormContext<RuleVersionFormValues>();

  const role = useWatch({
    control,
    name: `ruleDocument.stops.${index}.role`,
  });

  const currentRole = role || defaultRole || "stop";
  const badgeVariant =
    ROLE_BADGE_VARIANT[currentRole as keyof typeof ROLE_BADGE_VARIANT] ?? "secondary";

  return (
    <Collapsible defaultOpen>
      <div className="rounded-md border">
        <CollapsibleTrigger className="hover:bg-muted/50 flex w-full items-center justify-between p-3 text-sm font-medium">
          <div className="flex items-center gap-2">
            <Badge variant={badgeVariant} className="capitalize">
              {currentRole}
            </Badge>
            <span>{t("Stop {0}", index + 1)}</span>
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
          <div className="space-y-4 border-t p-3">
            <FormGroup cols={2}>
              <FormControl>
                <SelectField
                  control={control}
                  name={`ruleDocument.stops.${index}.role`}
                  label={t("Role")}
                  options={ROLE_OPTIONS}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name={`ruleDocument.stops.${index}.sequenceStart`}
                  label={t("Sequence Start")}
                  description={t("Starting sequence number for this stop type")}
                />
              </FormControl>
              <FormControl>
                <TagInput
                  control={control}
                  name={`ruleDocument.stops.${index}.sectionNames`}
                  label={t("Section Names")}
                  description={t("Restrict stop extraction to these document sections")}
                  placeholder={t("Add section...")}
                />
              </FormControl>
              <FormControl>
                <TagInput
                  control={control}
                  name={`ruleDocument.stops.${index}.startAnchors`}
                  label={t("Start Anchors")}
                  description={t("Text that marks the beginning of stop data")}
                  placeholder={t("Add anchor...")}
                />
              </FormControl>
              <FormControl>
                <TagInput
                  control={control}
                  name={`ruleDocument.stops.${index}.endAnchors`}
                  label={t("End Anchors")}
                  description={t("Text that marks the end of stop data")}
                  placeholder={t("Add anchor...")}
                />
              </FormControl>
              <FormControl>
                <TagInput
                  control={control}
                  name={`ruleDocument.stops.${index}.appointmentPatterns`}
                  label={t("Appointment Patterns")}
                  description={t("Regex patterns to extract appointment windows")}
                  placeholder={t("Add regex...")}
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name={`ruleDocument.stops.${index}.required`}
                  label={t("Required")}
                  description={t("Flag for review if this stop is missing")}
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name={`ruleDocument.stops.${index}.allowMultiple`}
                  label={t("Allow Multiple")}
                  description={t("Allow multiple instances of this stop type")}
                />
              </FormControl>
            </FormGroup>

            <Separator />

            <div className="bg-muted/30 rounded-md p-3">
              <StopExtractorEditor stopIndex={index} />
            </div>
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}

function StopExtractorEditor({ stopIndex }: { stopIndex: number }) {
  const t = useT();

  const { control } = useFormContext<RuleVersionFormValues>();
  const { fields, append, remove } = useFieldArray({
    control,
    name: `ruleDocument.stops.${stopIndex}.extractors`,
  });

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h4 className="text-sm font-medium">{t("Extractors ({0})", fields.length)}</h4>
          <p className="text-muted-foreground mt-0.5 text-xs">
            {t("Each extractor pulls a specific piece of data (name, address, date) from this stop.")}
          </p>
        </div>
        <Button
          type="button"
          variant="outline"
          size="xxs"
          className="gap-1"
          onClick={() =>
            append({
              fieldKey: "name",
              aliases: [],
              patterns: [],
              normalizer: "",
              confidence: 0,
              required: false,
            })
          }
        >
          <PlusIcon className="size-3" />
          {t("Add Extractor")}
        </Button>
      </div>
      {fields.map((field, extIdx) => (
        <div key={field.id} className="bg-background rounded border p-3">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
              {field.fieldKey || t("Extractor {0}", extIdx + 1)}
            </span>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="size-6"
              onClick={() => remove(extIdx)}
            >
              <TrashIcon className="text-destructive size-3" />
            </Button>
          </div>
          <FormGroup cols={2}>
            <FormControl>
              <SelectField
                control={control}
                name={`ruleDocument.stops.${stopIndex}.extractors.${extIdx}.fieldKey`}
                label={t("Field Key")}
                options={FIELD_KEY_OPTIONS}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name={`ruleDocument.stops.${stopIndex}.extractors.${extIdx}.normalizer`}
                label={t("Normalizer")}
                placeholder={t("e.g. trim")}
              />
            </FormControl>
            <FormControl>
              <TagInput
                control={control}
                name={`ruleDocument.stops.${stopIndex}.extractors.${extIdx}.aliases`}
                label={t("Aliases")}
                placeholder={t("Add alias...")}
              />
            </FormControl>
            <FormControl>
              <TagInput
                control={control}
                name={`ruleDocument.stops.${stopIndex}.extractors.${extIdx}.patterns`}
                label={t("Patterns")}
                placeholder={t("Add regex...")}
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name={`ruleDocument.stops.${stopIndex}.extractors.${extIdx}.confidence`}
                label={t("Min Confidence")}
                description={t("0.0 to 1.0")}
              />
            </FormControl>
            <FormControl>
              <SwitchField
                control={control}
                name={`ruleDocument.stops.${stopIndex}.extractors.${extIdx}.required`}
                label={t("Required")}
              />
            </FormControl>
          </FormGroup>
        </div>
      ))}
    </div>
  );
}
