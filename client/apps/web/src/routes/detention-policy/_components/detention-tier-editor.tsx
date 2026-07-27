import { NumberField } from "@/components/fields/number-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl } from "@trenova/shared/components/ui/form";
import { formatDetentionMinutes } from "@trenova/shared/lib/detention";
import type { DetentionPolicy } from "@trenova/shared/types/detention";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

const RATE_UNIT_OPTIONS = [
  { label: "Per hour", value: "Hour" },
  { label: "Per day", value: "Day" },
  { label: "Flat", value: "Flat" },
];

/**
 * Editor for the graduated rate ladder.
 *
 * Bands are half-open [from, to) measured in minutes of billable detention, so
 * appending a rung always starts exactly where the previous one ended. The
 * server rejects gaps and overlaps; keeping the client contiguous by
 * construction means operators rarely see that error.
 */
export function DetentionTierEditor() {
  const { control } = useFormContext<DetentionPolicy>();
  const { fields, append, remove } = useFieldArray({ control, name: "tiers" });
  const tiers = useWatch({ control, name: "tiers" }) ?? [];

  const appendTier = () => {
    const previous = tiers.at(-1);
    const start = previous?.toMinute ?? (previous ? previous.fromMinute + 120 : 0);

    if (previous && !previous.toMinute) {
      return;
    }

    append({
      fromMinute: start,
      toMinute: null,
      rate: 0,
      rateUnit: "Hour",
      label: "",
      sortOrder: fields.length,
    });
  };

  const lastIsOpenEnded = tiers.length > 0 && !tiers.at(-1)?.toMinute;

  return (
    <div className="flex flex-col gap-3">
      {fields.length === 0 && (
        <p className="text-muted-foreground text-sm">
          No tiers yet. A ladder must start at minute 0 of billable detention and
          run contiguously; the final rung may be open-ended.
        </p>
      )}

      {fields.map((field, index) => (
        <div key={field.id} className="rounded-md border p-3">
          <div className="mb-2 flex items-center justify-between">
            <p className="text-xs font-medium">
              Tier {index + 1}
              {tiers[index] && (
                <span className="text-muted-foreground ml-2 font-normal">
                  {formatDetentionMinutes(tiers[index].fromMinute)} –{" "}
                  {tiers[index].toMinute
                    ? formatDetentionMinutes(tiers[index].toMinute)
                    : "beyond"}
                </span>
              )}
            </p>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              className="h-6 text-xs"
              onClick={() => remove(index)}
            >
              Remove
            </Button>
          </div>

          <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
            <FormControl>
              <NumberField
                control={control}
                name={`tiers.${index}.fromMinute`}
                label="From"
                sideText="min"
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name={`tiers.${index}.toMinute`}
                label="To"
                sideText="min"
                description="Leave empty for the final open-ended rung"
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name={`tiers.${index}.rate`}
                label="Rate"
                sideText="$"
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name={`tiers.${index}.rateUnit`}
                label="Unit"
                options={RATE_UNIT_OPTIONS}
              />
            </FormControl>
          </div>

          <FormControl className="mt-2">
            <InputField
              control={control}
              name={`tiers.${index}.label`}
              label="Label"
              placeholder="First 2 hours"
            />
          </FormControl>
        </div>
      ))}

      <div>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={appendTier}
          disabled={lastIsOpenEnded}
        >
          Add tier
        </Button>
        {lastIsOpenEnded && (
          <p className="text-muted-foreground mt-1 text-xs">
            Close the final tier before adding another; only the last rung may be
            open-ended.
          </p>
        )}
      </div>
    </div>
  );
}
