import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { Button } from "@/components/ui/button";
import { FormControl, FormGroup } from "@/components/ui/form";
import type { TCASubscriptionFormValues } from "@/types/table-change-alert";
import { PlusIcon, TrashIcon } from "lucide-react";
import { type Control, useFieldArray, useWatch } from "react-hook-form";

const OPERATOR_OPTIONS = [
  { value: "eq", label: "Equals" },
  { value: "neq", label: "Not equals" },
  { value: "gt", label: "Greater than" },
  { value: "gte", label: "Greater than or equal" },
  { value: "lt", label: "Less than" },
  { value: "lte", label: "Less than or equal" },
  { value: "is_null", label: "Is empty" },
  { value: "is_not_null", label: "Is not empty" },
  { value: "contains", label: "Contains" },
  { value: "not_contains", label: "Does not contain" },
  { value: "changed_to", label: "Changed to" },
  { value: "changed_from", label: "Changed from" },
  { value: "changed", label: "Was modified" },
];

const MATCH_OPTIONS = [
  { value: "all", label: "Match ALL conditions" },
  { value: "any", label: "Match ANY condition" },
];

const UNARY_OPERATORS = new Set(["is_null", "is_not_null", "changed"]);

function ConditionRow({
  control,
  index,
  onRemove,
}: {
  control: Control<TCASubscriptionFormValues>;
  index: number;
  onRemove: () => void;
}) {
  const operator = useWatch({
    control,
    name: `conditions.${index}.operator`,
  });

  const isUnary = UNARY_OPERATORS.has(operator ?? "");

  return (
    <div className="flex items-start gap-2">
      <div className={`grid flex-1 gap-2 ${isUnary ? "grid-cols-2" : "grid-cols-3"}`}>
        <FormControl>
          <InputField<TCASubscriptionFormValues>
            control={control}
            name={`conditions.${index}.field`}
            placeholder="e.g., status"
          />
        </FormControl>
        <FormControl>
          <SelectField<TCASubscriptionFormValues>
            control={control}
            name={`conditions.${index}.operator`}
            options={OPERATOR_OPTIONS}
            placeholder="Operator"
          />
        </FormControl>
        {!isUnary && (
          <FormControl>
            <InputField<TCASubscriptionFormValues>
              control={control}
              name={`conditions.${index}.value`}
              placeholder="Value"
            />
          </FormControl>
        )}
      </div>
      <Button type="button" variant="ghost" size="icon-xs" onClick={onRemove} aria-label="Remove condition">
        <TrashIcon className="size-3.5" />
      </Button>
    </div>
  );
}

export function ConditionBuilder({
  control,
}: {
  control: Control<TCASubscriptionFormValues>;
}) {
  const { fields, append, remove } = useFieldArray({
    control,
    name: "conditions",
  });

  return (
    <div className="space-y-3">
      <FormGroup cols={1}>
        <FormControl>
          <SelectField<TCASubscriptionFormValues>
            control={control}
            name="conditionMatch"
            label="Condition Matching"
            options={MATCH_OPTIONS}
            description="How multiple conditions are evaluated together."
          />
        </FormControl>
      </FormGroup>

      {fields.length > 0 && (
        <div className="space-y-2">
          {fields.map((field, index) => (
            <ConditionRow
              key={field.id}
              control={control}
              index={index}
              onRemove={() => remove(index)}
            />
          ))}
        </div>
      )}

      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => append({ field: "", operator: "eq", value: "" })}
      >
        <PlusIcon className="mr-1 size-3.5" />
        Add condition
      </Button>
    </div>
  );
}
