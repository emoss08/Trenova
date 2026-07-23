import { Button } from "@trenova/shared/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { Input } from "@trenova/shared/components/ui/input";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { MAX_BREAKDOWN_DEFINITIONS, type BreakdownDefinitionInput } from "@trenova/shared/types/formula-template";
import { ListTreeIcon, Plus, Trash2 } from "lucide-react";
import { useCallback } from "react";
import { useFieldArray, useFormState, type Control, type UseFormRegister } from "react-hook-form";

type FormWithBreakdowns = {
  breakdownDefinitions: BreakdownDefinitionInput[];
};

type BreakdownDefinitionEditorProps = {
  control: Control<FormWithBreakdowns>;
  register: UseFormRegister<FormWithBreakdowns>;
  className?: string;
};

function FieldError({ message }: { message?: string }) {
  if (!message) return null;

  return <p className="mt-1 text-2xs text-destructive">{message}</p>;
}

export function BreakdownDefinitionEditor({
  control,
  register,
  className,
}: BreakdownDefinitionEditorProps) {
  const { fields, append, remove } = useFieldArray({
    control,
    name: "breakdownDefinitions",
  });
  const { errors } = useFormState({ control, name: "breakdownDefinitions" });

  const handleAdd = useCallback(() => {
    append({
      name: "",
      label: "",
      expression: "",
    });
  }, [append]);

  const atLimit = fields.length >= MAX_BREAKDOWN_DEFINITIONS;

  return (
    <Card className={className}>
      <CardHeader className="flex flex-row items-center justify-between border-b py-3">
        <div className="flex items-center gap-2">
          <div className="flex size-8 items-center justify-center rounded-lg bg-primary/10">
            <ListTreeIcon className="size-4 text-primary" />
          </div>
          <div>
            <CardTitle className="text-sm font-medium">Charge Breakdown</CardTitle>
            <p className="text-xs text-muted-foreground">
              Itemize the total into named components for invoices and audit
            </p>
          </div>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={handleAdd}
          disabled={atLimit}
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
              <ListTreeIcon className="size-5 text-muted-foreground" />
            </div>
            <p className="mt-3 text-sm font-medium">No breakdown items</p>
            <p className="mt-1 text-xs text-muted-foreground">
              Add items to break the calculated charge into labeled amounts
            </p>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleAdd}
              className="mt-4 gap-1.5"
            >
              <Plus className="size-3.5" />
              Add Item
            </Button>
          </div>
        ) : (
          <div className="space-y-3">
            {fields.map((field, index) => {
              const rowErrors = errors.breakdownDefinitions?.[index];

              return (
                <div
                  key={field.id}
                  className="group relative grid grid-cols-12 gap-3 rounded-lg border bg-muted/30 p-3 transition-colors hover:bg-muted/50"
                >
                  <div className="col-span-3">
                    <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                      Name
                    </label>
                    <Input
                      {...register(`breakdownDefinitions.${index}.name`)}
                      placeholder="fuelSurcharge"
                      className="h-8 font-mono text-sm"
                    />
                    <FieldError message={rowErrors?.name?.message} />
                  </div>

                  <div className="col-span-3">
                    <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                      Label
                    </label>
                    <Input
                      {...register(`breakdownDefinitions.${index}.label`)}
                      placeholder="Fuel Surcharge"
                      className="h-8 text-sm"
                    />
                    <FieldError message={rowErrors?.label?.message} />
                  </div>

                  <div className="col-span-5">
                    <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                      Expression
                    </label>
                    <Textarea
                      {...register(`breakdownDefinitions.${index}.expression`)}
                      placeholder="totalDistance * 0.35"
                      minRows={1}
                      maxRows={4}
                      className="min-h-8 py-1.5 font-mono text-sm"
                    />
                    <FieldError message={rowErrors?.expression?.message} />
                  </div>

                  <div className="col-span-1 flex items-start justify-end pt-6">
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
              );
            })}
            {atLimit && (
              <p className="text-xs text-muted-foreground">
                Maximum of {MAX_BREAKDOWN_DEFINITIONS} breakdown items reached.
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
