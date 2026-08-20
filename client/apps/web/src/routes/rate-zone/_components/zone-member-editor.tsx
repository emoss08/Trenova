import {
  LocationAutocompleteField,
  UsStateAutocompleteField,
} from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import type { GenericSelectOption } from "@trenova/shared/types/fields";
import type { RateZone, RateZoneMember } from "@trenova/shared/types/rate";
import { PlusIcon } from "lucide-react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

/**
 * A zone is a union of primitive places, and deliberately cannot contain
 * another zone: nesting would turn membership from an indexed lookup into a
 * graph walk, and the rating path cannot afford that.
 */
const MEMBER_SCOPE_CHOICES = [
  { label: "State", value: "State" },
  { label: "City", value: "CityState" },
  { label: "Postal prefix", value: "Zip3" },
  { label: "Postal code", value: "Zip5" },
  { label: "Location", value: "Location" },
  { label: "Country", value: "Country" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

const NEW_MEMBER = {
  scopeType: "State",
  scopeValue: "",
  city: "",
} as unknown as RateZoneMember;

export function ZoneMemberEditor() {
  const { control } = useFormContext<RateZone>();
  const { fields, append, remove } = useFieldArray({ control, name: "members" });
  const members = (useWatch({ control, name: "members" }) ?? []) as RateZoneMember[];

  return (
    <div className="flex flex-col gap-3">
      {fields.length === 0 && (
        <p className="text-muted-foreground text-sm">
          No places yet. A zone with no members matches nothing, so every lane written against it
          would quietly never apply.
        </p>
      )}

      {fields.map((field, index) => {
        const scopeType = members[index]?.scopeType;

        return (
          <div key={field.id} className="bg-card rounded-md border p-4">
            <div className="mb-3 flex items-center justify-between">
              <p className="text-sm font-medium">Place {index + 1}</p>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-6 text-xs"
                onClick={() => remove(index)}
              >
                Remove
              </Button>
            </div>

            <FormGroup cols={2}>
              <FormControl>
                <SelectField
                  control={control}
                  rules={{ required: true }}
                  name={`members.${index}.scopeType` as never}
                  label="Place Type"
                  placeholder="Select type"
                  description="How this place is named"
                  options={MEMBER_SCOPE_CHOICES}
                />
              </FormControl>

              {scopeType === "State" || scopeType === "CityState" ? (
                <FormControl>
                  <UsStateAutocompleteField
                    control={control}
                    rules={{ required: true }}
                    name={`members.${index}.scopeValue` as never}
                    label="State"
                    placeholder="State"
                    description="The state this place sits in"
                  />
                </FormControl>
              ) : scopeType === "Location" ? (
                <FormControl>
                  <LocationAutocompleteField
                    control={control}
                    rules={{ required: true }}
                    name={`members.${index}.scopeValue` as never}
                    label="Location"
                    placeholder="Location"
                    description="A single facility"
                  />
                </FormControl>
              ) : (
                <FormControl>
                  <InputField
                    control={control}
                    rules={{ required: true }}
                    name={`members.${index}.scopeValue` as never}
                    label={scopeType === "Country" ? "Country" : "Postal"}
                    placeholder={scopeType === "Zip3" ? "606" : "60601"}
                    description={
                      scopeType === "Zip3"
                        ? "The first three digits, which is how most tariffs are written"
                        : "The value this place is named by"
                    }
                  />
                </FormControl>
              )}

              {scopeType === "CityState" && (
                <FormControl cols="full">
                  <InputField
                    control={control}
                    rules={{ required: true }}
                    name={`members.${index}.city` as never}
                    label="City"
                    placeholder="Chicago"
                    description="Spelling and case do not matter — the city is folded before it is matched"
                  />
                </FormControl>
              )}
            </FormGroup>
          </div>
        );
      })}

      <Button type="button" variant="outline" size="sm" onClick={() => append(NEW_MEMBER)}>
        <PlusIcon className="mr-1 size-3.5" />
        Add place
      </Button>
    </div>
  );
}
