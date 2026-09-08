import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { statusChoices } from "@/lib/choices";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import type { ReviewTemplateFormValues } from "@trenova/shared/types/performance-review";
import { InfoIcon, PlusIcon, Trash2Icon } from "lucide-react";
import { useMemo } from "react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

type ReviewTemplateFormProps = {
  isEdit: boolean;
  openReviewCount?: number;
};

export function ReviewTemplateForm({ isEdit, openReviewCount = 0 }: ReviewTemplateFormProps) {
  const { control } = useFormContext<ReviewTemplateFormValues>();
  const items = useFieldArray({ control, name: "items" });
  const watched = useWatch({ control, name: "items" });
  const totalWeight = useMemo(
    () => (watched ?? []).reduce((sum, item) => sum + (item?.weight ?? 0), 0),
    [watched],
  );

  return (
    <div className="flex flex-col gap-6">
      <section className="flex flex-col gap-3">
        <SectionTitle
          title="General"
          hint="Name and code identify the template when someone starts a review."
        />
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="code"
              label="Code"
              placeholder="e.g. DRIVER-ANNUAL"
              rules={{ required: true }}
              description="Short unique identifier. Letters, digits, dashes and underscores."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="name"
              label="Name"
              placeholder="e.g. Annual Driver Review"
              rules={{ required: true }}
              description="Shown when someone picks a template to start a review."
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label="Status"
              options={statusChoices}
              rules={{ required: true }}
              placeholder="Select a status"
              description={
                isEdit && openReviewCount > 0
                  ? `${openReviewCount} open review${openReviewCount === 1 ? "" : "s"} use this template; close those first to deactivate.`
                  : "Inactive templates cannot be used for new reviews."
              }
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="cadenceMonths"
              label="Repeat every"
              sideText="months"
              min={1}
              placeholder="12"
              description="Closing a review schedules the next one this far out. Leave empty for one-off reviews."
            />
          </FormControl>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="isDefault"
              label="Default template"
              description="Pre-selected when someone starts a review."
              position="left"
              outlined
            />
          </FormControl>
          <FormControl className="col-span-2">
            <TextareaField
              control={control}
              name="description"
              label="Description"
              placeholder="Who this review is for and what it covers"
              maxLength={1000}
              description="Optional notes on when and for whom this template should be used."
            />
          </FormControl>
        </FormGroup>
      </section>

      <section className="flex flex-col gap-3">
        <div className="flex items-start justify-between gap-2">
          <SectionTitle
            title="Rating items"
            hint={`Each item is scored 1–5. Weights decide the share of the overall score — currently ${totalWeight} in total.`}
          />
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => items.append({ key: "", label: "", description: null, weight: 1 })}
          >
            <PlusIcon className="size-3.5" />
            Add item
          </Button>
        </div>
        <Alert variant="default">
          <InfoIcon className="size-4" />
          <AlertTitle>Items are copied when a review starts</AlertTitle>
          <AlertDescription>
            Changes to the rating items only affect reviews started after you save. Reviews already
            in progress keep the items and weights they were started with.
          </AlertDescription>
        </Alert>
        <div className="flex flex-col gap-3">
          {items.fields.map((field, index) => (
            <div key={field.id} className="bg-muted/30 rounded-lg border p-3">
              <div className="mb-2 flex items-center justify-between">
                <p className="text-muted-foreground text-[11px] font-medium uppercase">
                  Item {index + 1}
                </p>
                {items.fields.length > 1 ? (
                  <Button
                    type="button"
                    size="sm"
                    variant="ghost"
                    className="size-7"
                    aria-label={`Remove item ${index + 1}`}
                    onClick={() => items.remove(index)}
                  >
                    <Trash2Icon className="size-3.5" />
                  </Button>
                ) : null}
              </div>
              <FormGroup cols={3}>
                <FormControl>
                  <InputField
                    control={control}
                    name={`items.${index}.key`}
                    label="Key"
                    placeholder="e.g. safety"
                    rules={{ required: true }}
                    description="Stable identifier kept on every review; must be unique within the template."
                  />
                </FormControl>
                <FormControl>
                  <InputField
                    control={control}
                    name={`items.${index}.label`}
                    label="Label"
                    placeholder="e.g. Safe driving"
                    rules={{ required: true }}
                    description="Shown to the reviewer as the thing being scored."
                  />
                </FormControl>
                <FormControl>
                  <NumberField
                    control={control}
                    name={`items.${index}.weight`}
                    label="Weight"
                    min={1}
                    max={10}
                    placeholder="1"
                    description="Relative share of the overall score; the score is the weighted average of items."
                  />
                </FormControl>
                <FormControl className="col-span-3">
                  <InputField
                    control={control}
                    name={`items.${index}.description`}
                    label="Hint"
                    placeholder="What the reviewer should be looking at"
                    maxLength={255}
                    description="Optional guidance for whoever scores this item."
                  />
                </FormControl>
              </FormGroup>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}

function SectionTitle({ title, hint }: { title: string; hint: string }) {
  return (
    <div>
      <h3 className="text-sm font-semibold">{title}</h3>
      <p className="text-muted-foreground text-xs">{hint}</p>
    </div>
  );
}
