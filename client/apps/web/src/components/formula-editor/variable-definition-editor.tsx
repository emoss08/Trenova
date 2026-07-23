import { SelectField } from "@/components/fields/select-field";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import type {
  VariableDefinition,
  VariableValueType,
} from "@/types/formula-template";
import { Plus, Trash2, Variable } from "lucide-react";
import { useCallback } from "react";
import {
  useFieldArray,
  type Control,
  type UseFormRegister,
} from "react-hook-form";

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
          <div className="flex size-8 items-center justify-center rounded-lg bg-primary/10">
            <Variable className="size-4 text-primary" />
          </div>
          <div>
            <CardTitle className="text-sm font-medium">
              Custom Variables
            </CardTitle>
            <p className="text-xs text-muted-foreground">
              Define additional variables for your formula
            </p>
          </div>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={handleAdd}
          className="gap-1.5"
        >
          <Plus className="size-3.5" />
          Add
        </Button>
      </CardHeader>
      <CardContent className="p-4">
        {fields.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <div className="flex size-12 items-center justify-center rounded-full bg-muted">
              <Variable className="size-5 text-muted-foreground" />
            </div>
            <p className="mt-3 text-sm font-medium">No custom variables</p>
            <p className="mt-1 text-xs text-muted-foreground">
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
              <div
                key={field.id}
                className="group relative grid grid-cols-12 gap-3 rounded-lg border bg-muted/30 p-3 transition-colors hover:bg-muted/50"
              >
                <div className="col-span-3">
                  <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                    Name
                  </label>
                  <Input
                    {...register(`variableDefinitions.${index}.name`)}
                    placeholder="myVariable"
                    className="h-8 font-mono text-sm"
                  />
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
                  <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                    Default
                  </label>
                  <Input
                    {...register(`variableDefinitions.${index}.defaultValue`)}
                    placeholder="0"
                    className="h-8 text-sm"
                  />
                </div>

                <div className="col-span-4">
                  <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                    Description
                  </label>
                  <Input
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
                    onClick={() => remove(index)}
                    className="size-8 p-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100 hover:bg-destructive/10 hover:text-destructive"
                  >
                    <Trash2 className="size-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
