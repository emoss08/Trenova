import {
  LocationAutocompleteField,
  RateZoneAutocompleteField,
  UsStateAutocompleteField,
} from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { rateScopeTypeChoices } from "@/lib/choices";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import type { RateScopeType } from "@trenova/shared/types/rate";
import type { Control, FieldValues } from "react-hook-form";
import { useWatch } from "react-hook-form";

type LaneSide = "origin" | "destination";

type LaneScopeFieldsProps<T extends FieldValues> = {
  control: Control<T>;
  side: LaneSide;
  /**
   * Where the lane sits in the form. Lanes are edited inside an array on the
   * agreement, so the field names have to carry the index with them.
   */
  namePrefix: string;
};

/** The field names one side of a lane writes to, so the two sides share a form. */
function fieldsFor(namePrefix: string, side: LaneSide) {
  const prefix = `${namePrefix}${side}`;

  return {
    scopeType: `${prefix}ScopeType`,
    scopeValue: `${prefix}ScopeValue`,
    city: `${prefix}City`,
    radiusMeters: `${prefix}RadiusMeters`,
    latitude: `${prefix}Latitude`,
    longitude: `${prefix}Longitude`,
  };
}

const sideLabel: Record<LaneSide, string> = {
  origin: "Origin",
  destination: "Destination",
};

/**
 * The value fields for one end of a lane.
 *
 * Which fields appear is driven by the scope, because the scopes name places in
 * genuinely different ways: a zone is a record, a state is a record, a city is
 * a name paired with a state, a postal code is a string, and a radius is a
 * point with a distance. Showing all of them at once would invite a lane that
 * carries a postal code and a zone and matches on neither.
 */
export function LaneScopeFields<T extends FieldValues>({
  control,
  side,
  namePrefix,
}: LaneScopeFieldsProps<T>) {
  const names = fieldsFor(namePrefix, side);
  const scopeType = useWatch({
    control,
    name: names.scopeType as never,
  }) as unknown as RateScopeType;
  const label = sideLabel[side];

  return (
    <FormGroup cols={2}>
      <FormControl cols={scopeType === "Any" ? "full" : 1}>
        <SelectField
          control={control}
          rules={{ required: true }}
          name={names.scopeType as never}
          label={`${label} scope`}
          placeholder="Scope"
          description="How narrowly this end of the lane is written. A narrower scope beats a wider one covering the same load."
          options={rateScopeTypeChoices}
        />
      </FormControl>

      {scopeType === "Zone" && (
        <FormControl>
          <RateZoneAutocompleteField
            control={control}
            rules={{ required: true }}
            name={names.scopeValue as never}
            label={`${label} zone`}
            placeholder="Zone"
            description="The market area this end covers"
          />
        </FormControl>
      )}

      {scopeType === "State" && (
        <FormControl>
          <UsStateAutocompleteField
            control={control}
            rules={{ required: true }}
            name={names.scopeValue as never}
            label={`${label} state`}
            placeholder="State"
            description="The state this end covers"
          />
        </FormControl>
      )}

      {scopeType === "Location" && (
        <FormControl>
          <LocationAutocompleteField
            control={control}
            rules={{ required: true }}
            name={names.scopeValue as never}
            label={`${label} location`}
            placeholder="Location"
            description="The single facility this end covers"
          />
        </FormControl>
      )}

      {scopeType === "CityState" && (
        <>
          <FormControl>
            <UsStateAutocompleteField
              control={control}
              rules={{ required: true }}
              name={names.scopeValue as never}
              label={`${label} state`}
              placeholder="State"
              description="The state the city sits in"
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name={names.city as never}
              label={`${label} city`}
              placeholder="City"
              description="Spelling and case do not matter — the city is folded before it is matched"
            />
          </FormControl>
        </>
      )}

      {(scopeType === "Zip3" || scopeType === "Zip5") && (
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name={names.scopeValue as never}
            label={scopeType === "Zip3" ? `${label} postal prefix` : `${label} postal code`}
            placeholder={scopeType === "Zip3" ? "606" : "60601"}
            description={
              scopeType === "Zip3"
                ? "The first three digits, which is how most tariffs are written"
                : "The full postal code"
            }
          />
        </FormControl>
      )}

      {scopeType === "Country" && (
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name={names.scopeValue as never}
            label={`${label} country`}
            placeholder="USA"
            description="Three letter country code"
          />
        </FormControl>
      )}

      {scopeType === "Radius" && (
        <>
          <FormControl>
            <NumberField
              control={control}
              rules={{ required: true }}
              name={names.radiusMeters as never}
              label={`${label} radius (meters)`}
              placeholder="80000"
              description="How far from the centre point this end reaches"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              rules={{ required: true }}
              name={names.latitude as never}
              label={`${label} latitude`}
              placeholder="32.7767"
              description="The centre point this end measures from"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              rules={{ required: true }}
              name={names.longitude as never}
              label={`${label} longitude`}
              placeholder="-96.7970"
              description="The centre point this end measures from"
            />
          </FormControl>
        </>
      )}
    </FormGroup>
  );
}
