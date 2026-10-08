import { useT } from "@trenova/shared/i18n/use-t";
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
    // i18n-ignore: form field name
    city: `${prefix}City`,
    radiusMeters: `${prefix}RadiusMeters`,
    // i18n-ignore: form field name
    latitude: `${prefix}Latitude`,
    // i18n-ignore: form field name
    longitude: `${prefix}Longitude`,
  };
}

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
  const t = useT();

  const names = fieldsFor(namePrefix, side);
  const scopeType = useWatch({
    control,
    name: names.scopeType as never,
  }) as unknown as RateScopeType;
  const isOrigin = side === "origin";

  return (
    <FormGroup cols={2}>
      <FormControl cols={scopeType === "Any" ? "full" : 1}>
        <SelectField
          control={control}
          rules={{ required: true }}
          name={names.scopeType as never}
          label={isOrigin ? t("Origin Scope") : t("Destination Scope")}
          placeholder={t("Select scope")}
          description={t(
            "How narrowly this end of the lane is written. A narrower scope beats a wider one covering the same load.",
          )}
          options={rateScopeTypeChoices}
        />
      </FormControl>

      {scopeType === "Zone" && (
        <FormControl>
          <RateZoneAutocompleteField
            control={control}
            rules={{ required: true }}
            name={names.scopeValue as never}
            label={isOrigin ? t("Origin Zone") : t("Destination Zone")}
            placeholder={t("Select zone")}
            description={t("The market area this end covers")}
          />
        </FormControl>
      )}

      {scopeType === "State" && (
        <FormControl>
          <UsStateAutocompleteField
            control={control}
            rules={{ required: true }}
            name={names.scopeValue as never}
            label={isOrigin ? t("Origin State") : t("Destination State")}
            placeholder={t("Select state")}
            description={t("The state this end covers")}
          />
        </FormControl>
      )}

      {scopeType === "Location" && (
        <FormControl>
          <LocationAutocompleteField
            control={control}
            rules={{ required: true }}
            name={names.scopeValue as never}
            label={isOrigin ? t("Origin Location") : t("Destination Location")}
            placeholder={t("Select location")}
            description={t("The single facility this end covers")}
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
              label={isOrigin ? t("Origin State") : t("Destination State")}
              placeholder={t("Select state")}
              description={t("The state the city sits in")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name={names.city as never}
              label={isOrigin ? t("Origin City") : t("Destination City")}
              placeholder={t("City")}
              description={t(
                "Spelling and case do not matter — the city is folded before it is matched",
              )}
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
            label={
              scopeType === "Zip3"
                ? isOrigin
                  ? t("Origin Postal Prefix")
                  : t("Destination Postal Prefix")
                : isOrigin
                  ? t("Origin Postal Code")
                  : t("Destination Postal Code")
            }
            placeholder={scopeType === "Zip3" ? "606" : "60601"}
            description={
              scopeType === "Zip3"
                ? t("The first three digits, which is how most tariffs are written")
                : t("The full postal code")
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
            label={isOrigin ? t("Origin Country") : t("Destination Country")}
            placeholder={t("USA")}
            description={t("Three letter country code")}
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
              label={isOrigin ? t("Origin Radius") : t("Destination Radius")}
              placeholder="80000"
              sideText="m"
              description={t("How far from the centre point this end reaches")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              rules={{ required: true }}
              name={names.latitude as never}
              label={isOrigin ? t("Origin Latitude") : t("Destination Latitude")}
              placeholder="32.7767"
              description={t("The centre point this end measures from")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              rules={{ required: true }}
              name={names.longitude as never}
              label={isOrigin ? t("Origin Longitude") : t("Destination Longitude")}
              placeholder="-96.7970"
              description={t("The centre point this end measures from")}
            />
          </FormControl>
        </>
      )}
    </FormGroup>
  );
}
