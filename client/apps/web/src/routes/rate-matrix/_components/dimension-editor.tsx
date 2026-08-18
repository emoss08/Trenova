import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { rateMatrixDimensionKindChoices, rateMatrixMatchModeChoices } from "@/lib/choices";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { MAX_MATRIX_DIMENSIONS } from "@trenova/shared/lib/rate-matrix";
import type { RateMatrix, RateMatrixDimension } from "@trenova/shared/types/rate";
import { PlusIcon, TriangleAlertIcon } from "lucide-react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

/**
 * The axes are the shape of the sheet, and changing them after rates exist
 * changes what every existing cell means. That is why they live on their own
 * tab rather than beside the rates: it should take a deliberate visit.
 */
export function DimensionEditor() {
  const { control } = useFormContext<RateMatrix>();
  const { fields, append, replace } = useFieldArray({ control, name: "dimensions" });
  const dimensions = (useWatch({ control, name: "dimensions" }) ?? []) as RateMatrixDimension[];

  const atCapacity = fields.length >= MAX_MATRIX_DIMENSIONS;

  // Positions have to stay contiguous. Removing a middle axis and leaving a
  // hole would let the next one appended land on a position already taken, and
  // two axes claiming one slot means every cell reads its bound from the wrong
  // one.
  const removeAxis = (index: number) => {
    replace(
      dimensions
        .filter((_, at) => at !== index)
        .map((dimension, at) => ({ ...dimension, position: at })),
    );
  };

  return (
    <div className="flex flex-col gap-3">
      {fields.length === 0 && (
        <p className="text-muted-foreground text-sm">
          No axes yet. A matrix with no axes can never be looked up, so every lane pointing at it
          would quietly price nothing.
        </p>
      )}

      {fields.length > 0 && (
        <Alert>
          <TriangleAlertIcon className="size-4" />
          <AlertDescription>
            Changing an axis after rates exist changes what every existing cell means. Re-upload the
            grid after any change here.
          </AlertDescription>
        </Alert>
      )}

      {fields.map((field, index) => {
        const matchMode = dimensions[index]?.matchMode;

        return (
          <div key={field.id} className="rounded-md border bg-card p-4">
            <div className="mb-3 flex items-center justify-between">
              <p className="text-sm font-medium">Axis {index + 1}</p>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-6 text-xs"
                onClick={() => removeAxis(index)}
              >
                Remove
              </Button>
            </div>

            <FormGroup cols={2}>
              <FormControl>
                <SelectField
                  control={control}
                  rules={{ required: true }}
                  name={`dimensions.${index}.kind` as never}
                  label="Dimension"
                  placeholder="Select dimension"
                  description="What about the shipment this axis reads"
                  options={rateMatrixDimensionKindChoices}
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={control}
                  rules={{ required: true }}
                  name={`dimensions.${index}.matchMode` as never}
                  label="Match Mode"
                  placeholder="Select match mode"
                  description={
                    matchMode === "Range"
                      ? "Bands, each covering from its floor up to but not including the next"
                      : "An exact key, like a zone code or a freight class"
                  }
                  options={rateMatrixMatchModeChoices}
                />
              </FormControl>
              <FormControl cols="full">
                <InputField
                  control={control}
                  name={`dimensions.${index}.label` as never}
                  label="Label"
                  placeholder="Origin zone"
                  description="What this axis is called in the grid — leave blank to use the kind"
                />
              </FormControl>
            </FormGroup>
          </div>
        );
      })}

      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={atCapacity}
        onClick={() =>
          append({
            position: fields.length,
            kind: "Zone",
            matchMode: "Exact",
            label: "",
          } as RateMatrixDimension)
        }
      >
        <PlusIcon className="mr-1 size-3.5" />
        Add axis
      </Button>

      {atCapacity && (
        <p className="text-muted-foreground text-xs">
          Four axes is the limit. Origin zone, destination zone, weight break and class covers every
          published tariff we have seen, and a fifth would make the grid unreadable.
        </p>
      )}
    </div>
  );
}
