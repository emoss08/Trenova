import { CarrierAutocompleteField } from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { carrierRateMethodChoices, offerTtlChoices, tenderChannelChoices } from "@/lib/choices";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl } from "@trenova/shared/components/ui/form";
import { formatDurationFromSeconds } from "@trenova/shared/lib/date";
import {
  emptyRoutingGuideEntry,
  type RoutingGuidePayloadInput,
} from "@trenova/shared/types/routing-guide";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

function ttlOptionsFor(valueSeconds: number | undefined) {
  if (valueSeconds == null || offerTtlChoices.some((choice) => choice.value === valueSeconds)) {
    return offerTtlChoices;
  }
  return [
    ...offerTtlChoices,
    { label: formatDurationFromSeconds(valueSeconds), value: valueSeconds },
  ];
}

/**
 * Editor for the guide's ranked carrier ladder. Rank 1 is offered first; each
 * carrier holds its offer for the configured expiry before the waterfall moves
 * to the next rank.
 */
export function RoutingGuideEntryEditor() {
  const { control } = useFormContext<RoutingGuidePayloadInput>();
  const { fields, append, remove } = useFieldArray({ control, name: "entries" });
  const entries = useWatch({ control, name: "entries" }) ?? [];

  const appendEntry = () => {
    let max = 0;
    for (const entry of entries) {
      const rank = typeof entry?.rank === "number" ? entry.rank : 0;
      if (rank > max) max = rank;
    }
    append(emptyRoutingGuideEntry(max + 1));
  };

  return (
    <div className="flex flex-col gap-3">
      {fields.length === 0 && (
        <p className="text-muted-foreground text-sm">
          No carriers yet. Add carriers in the order they should be offered the freight — rank 1
          goes first.
        </p>
      )}

      {fields.map((field, index) => (
        <div key={field.id} className="rounded-md border p-3">
          <div className="mb-2 flex items-center justify-between">
            <p className="text-xs font-medium">
              Rank {entries[index]?.rank ?? index + 1}
              {entries[index]?.channel === "EDI" && (
                <span className="text-muted-foreground ml-2 font-normal">
                  Requires a default EDI channel on the carrier
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

          <div className="grid gap-2 sm:grid-cols-2">
            <FormControl>
              <CarrierAutocompleteField
                control={control}
                name={`entries.${index}.carrierId`}
                label="Carrier"
                placeholder="Select carrier"
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name={`entries.${index}.rank`}
                label="Rank"
                placeholder="1"
                rules={{ required: true }}
                description="Offer order — rank 1 is tendered first."
              />
            </FormControl>
          </div>

          <div className="mt-2 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
            <FormControl>
              <SelectField
                control={control}
                name={`entries.${index}.rateMethod`}
                label="Rate Method"
                placeholder="Select method"
                rules={{ required: true }}
                options={carrierRateMethodChoices}
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name={`entries.${index}.rate`}
                label="Rate"
                placeholder="1500.00"
                sideText="$"
                decimalScale={2}
                rules={{ required: true }}
                description="Flat total or per-mile rate offered."
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name={`entries.${index}.offerTtlSeconds`}
                label="Offer Expiry"
                placeholder="Select expiry"
                rules={{ required: true }}
                options={ttlOptionsFor(
                  typeof entries[index]?.offerTtlSeconds === "number"
                    ? entries[index]?.offerTtlSeconds
                    : undefined,
                )}
                description="How long this carrier holds the offer."
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name={`entries.${index}.channel`}
                label="Channel"
                placeholder="Select channel"
                rules={{ required: true }}
                options={tenderChannelChoices}
                description="How the offer reaches the carrier."
              />
            </FormControl>
          </div>
        </div>
      ))}

      <div>
        <Button type="button" size="sm" variant="outline" onClick={appendEntry}>
          Add carrier
        </Button>
      </div>
    </div>
  );
}
