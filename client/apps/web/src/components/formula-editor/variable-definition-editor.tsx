import { SelectField } from "@/components/fields/select-field";
import { Button } from "@trenova/shared/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { Input } from "@trenova/shared/components/ui/input";
import type { VariableDefinition, VariableValueType } from "@trenova/shared/types/formula-template";
import { Plus, Trash2, Variable } from "lucide-react";
import { useCallback, useEffect } from "react";
import {
  useController,
  useFieldArray,
  useFormState,
  useWatch,
  type Control,
  type UseFormRegister,
} from "react-hook-form";

export function coerceVariableDefaultValue(type: VariableValueType, raw: unknown): unknown {
  if (raw === "" || raw === null || raw === undefined) return undefined;

  switch (type) {
    case "Number": {
      if (typeof raw === "number") return raw;
      if (typeof raw !== "string") return raw;
      const parsed = Number(raw);
      return Number.isNaN(parsed) ? raw : parsed;
    }
    case "Boolean": {
      if (typeof raw === "boolean") return raw;
      if (raw === "true") return true;
      if (raw === "false") return false;
      return raw;
    }
    case "String":
      if (typeof raw === "string") return raw;
      if (typeof raw === "number" || typeof raw === "boolean") return String(raw);
      return raw;
    default:
      return raw;
  }
}

type FormWithVariables = {
  variableDefinitions: VariableDefinition[];
};

type VariableDefinitionEditorProps = {
  control: Control<FormWithVariables>;
  register: UseFormRegister<FormWithVariables>;
  className?: string;
};

const VARIABLE_TYPES: { value: VariableValueType; label: string }[] = [
  { value: "Number", label: "Number" },
  { value: "String", label: "String" },
  { value: "Boolean", label: "Boolean" },
  { value: "Date", label: "Date" },
  { value: "Array", label: "Array" },
  { value: "Object", label: "Object" },
  { value: "Any", label: "Any" },
];

export function VariableDefinitionEditor({
  control,
  register,
  className,
}: VariableDefinitionEditorProps) {
  const { fields, append, remove } = useFieldArray({
    control,
    name: "variableDefinitions",
  });

  const handleAdd = useCallback(() => {
    append({
      name: "",
      type: "Number",
      description: "",
      required: false,
      defaultValue: undefined,
    });
  }, [append]);

  return (
    <Card className={className}>
      <CardHeader className="flex flex-row items-center justify-between border-b py-3">
        <div className="flex items-center gap-2">
          <div className="bg-primary/10 flex size-8 items-center justify-center rounded-lg">
            <Variable className="text-primary size-4" />
          </div>
          <div>
            <CardTitle className="text-sm font-medium">Custom Variables</CardTitle>
            <p className="text-muted-foreground text-xs">
              Define additional variables for your formula
            </p>
          </div>
        </div>
        <Button type="button" variant="outline" size="sm" onClick={handleAdd} className="gap-1.5">
          <Plus className="size-3.5" />
          Add
        </Button>
      </CardHeader>
      <CardContent className="p-4">
        {fields.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <div className="bg-muted flex size-12 items-center justify-center rounded-full">
              <Variable className="text-muted-foreground size-5" />
            </div>
            <p className="mt-3 text-sm font-medium">No custom variables</p>
            <p className="text-muted-foreground mt-1 text-xs">
              Add custom variables to use in your formula expression
            </p>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleAdd}
              className="mt-4 gap-1.5"
            >
              <Plus className="size-3.5" />
              Add Variable
            </Button>
          </div>
        ) : (
          <div className="space-y-3">
            {fields.map((field, index) => (
              <VariableDefinitionRow
                key={field.id}
                index={index}
                control={control}
                register={register}
                onRemove={() => remove(index)}
              />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function VariableDefinitionRow({
  index,
  control,
  register,
  onRemove,
}: {
  index: number;
  control: Control<FormWithVariables>;
  register: UseFormRegister<FormWithVariables>;
  onRemove: () => void;
}) {
  const variableType = useWatch({
    control,
    name: `variableDefinitions.${index}.type`,
  });
  const { errors } = useFormState({ control, name: `variableDefinitions.${index}` });
  const nameError = errors.variableDefinitions?.[index]?.name?.message;
  const { field: defaultField } = useController({
    control,
    name: `variableDefinitions.${index}.defaultValue`,
  });
  const { onChange: onDefaultChange, value: defaultValue } = defaultField;

  // A default typed as "5" while the type was Number is the number 5; if the
  // type then becomes String it should be the text "5", and vice versa, so
  // the stored default always matches the declared type.
  useEffect(() => {
    const coerced = coerceVariableDefaultValue(variableType ?? "Number", defaultValue);
    if (coerced !== defaultValue) {
      onDefaultChange(coerced);
    }
  }, [variableType, defaultValue, onDefaultChange]);

  const nameId = `variable-${index}-name`;
  const defaultId = `variable-${index}-default`;
  const descriptionId = `variable-${index}-description`;

  return (
    <div className="group bg-muted/30 hover:bg-muted/50 relative grid grid-cols-12 gap-3 rounded-lg border p-3 transition-colors">
      <div className="col-span-3">
        <label htmlFor={nameId} className="text-muted-foreground mb-1.5 block text-xs font-medium">
          Name
        </label>
        <Input
          id={nameId}
          {...register(`variableDefinitions.${index}.name`)}
          placeholder="myVariable"
          aria-invalid={nameError ? "true" : undefined}
          className="h-8 font-mono text-sm"
        />
        {nameError && <p className="text-2xs text-destructive mt-1">{nameError}</p>}
      </div>

      <div className="col-span-2">
        <SelectField
          label="Type"
          name={`variableDefinitions.${index}.type` as any}
          control={control as any}
          options={VARIABLE_TYPES}
        />
      </div>

      <div className="col-span-2">
        <label
          htmlFor={defaultId}
          className="text-muted-foreground mb-1.5 block text-xs font-medium"
        >
          Default
        </label>
        <Input
          id={defaultId}
          name={defaultField.name}
          ref={defaultField.ref}
          value={defaultValue === undefined || defaultValue === null ? "" : String(defaultValue)}
          onChange={(event) =>
            onDefaultChange(
              coerceVariableDefaultValue(variableType ?? "Number", event.target.value),
            )
          }
          onBlur={defaultField.onBlur}
          placeholder="0"
          className="h-8 text-sm"
        />
      </div>

      <div className="col-span-4">
        <label
          htmlFor={descriptionId}
          className="text-muted-foreground mb-1.5 block text-xs font-medium"
        >
          Description
        </label>
        <Input
          id={descriptionId}
          {...register(`variableDefinitions.${index}.description`)}
          placeholder="Optional description"
          className="h-8 text-sm"
        />
      </div>

      <div className="col-span-1 flex items-end justify-end pb-0.5">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onRemove}
          aria-label="Remove variable"
          className="text-muted-foreground hover:bg-destructive/10 hover:text-destructive size-8 p-0 opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
        >
          <Trash2 className="size-4" />
        </Button>
      </div>
    </div>
  );
}
