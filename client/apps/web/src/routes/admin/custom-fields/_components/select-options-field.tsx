import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { Button } from "@/components/ui/button";
import { FormSection } from "@/components/ui/form";
import type { CustomFieldDefinition } from "@/types/custom-field";
import { PlusIcon, TrashIcon } from "lucide-react";
import { type Control, useFieldArray } from "react-hook-form";

interface SelectOptionsFieldProps {
  control: Control<CustomFieldDefinition>;
}

export function SelectOptionsField({ control }: SelectOptionsFieldProps) {
  const { fields, append, remove } = useFieldArray({
    control,
    name: "options",
  });

  return (
    <FormSection
      title="Options"
      description="Define the available options for this select field"
      action={
        <Button
          type="button"
          variant="outline"
          size="xxs"
          onClick={() =>
            append({ value: "", label: "", color: "", description: "" })
          }
        >
          <PlusIcon className="size-3" />
          Add Option
        </Button>
      }
    >
      {fields.length === 0 && (
        <p className="text-sm text-muted-foreground">
          No options defined. Add at least one option for select fields.
        </p>
      )}
      <div className="space-y-2">
        {fields.map((field, index) => (
          <div
            key={field.id}
            className="grid grid-cols-[1fr_1fr_1fr_auto] items-end gap-2"
          >
            <InputField
              control={control}
              name={`options.${index}.value`}
              label="Value"
              placeholder="value"
            />
            <InputField
              control={control}
              name={`options.${index}.label`}
              label="Label"
              placeholder="Display Label"
            />
            <ColorField
              control={control}
              name={`options.${index}.color`}
              label="Color"
            />
            <div className="pb-0.5">
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => remove(index)}
              >
                <TrashIcon className="size-4 text-destructive" />
              </Button>
            </div>
          </div>
        ))}
      </div>
    </FormSection>
  );
}
