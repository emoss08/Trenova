import { DistanceProfileAutocompleteField } from "@/components/autocomplete-fields";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import { distanceControlUnitChoices } from "@/lib/choices";
import type { DistanceControl, DistanceProfileAssignment } from "@/types/distance-control";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

type Assignment = {
  name: DistanceProfileAssignment;
  label: string;
  description: string;
};

const ASSIGNMENT_COPY = defineLabels({
  loadedMoveDistanceProfileId: "Loaded moves",
  loadedMoveDistanceProfileIdDescription: "Routes a move that carries freight.",
  emptyMoveDistanceProfileId: "Empty moves",
  emptyMoveDistanceProfileIdDescription: "Routes a move that repositions without freight.",
  etaOutOfRouteDistanceProfileId: "Arrival estimates",
  etaOutOfRouteDistanceProfileIdDescription:
    "Routes the legs arrival times and dispatch suggestions are estimated over.",
  payDistanceProfileId: "Pay",
  payDistanceProfileIdDescription:
    "Measures the miles drivers and carriers paid by the mile are paid for.",
  billingDistanceProfileId: "Billing",
  billingDistanceProfileIdDescription:
    "Measures the miles rating, fuel surcharges and invoices bill customers for.",
  fuelDistanceProfileId: "Fuel",
  fuelDistanceProfileIdDescription:
    "Measures the miles fuel tax returns are filed on, and routes the jurisdiction breakdown.",
});

const assignment = (name: DistanceProfileAssignment): Assignment => ({
  name,
  label: ASSIGNMENT_COPY[name],
  description: ASSIGNMENT_COPY[`${name}Description`],
});

const MOVES = [
  assignment("loadedMoveDistanceProfileId"),
  assignment("emptyMoveDistanceProfileId"),
  assignment("etaOutOfRouteDistanceProfileId"),
];

const SETTLEMENT = [
  assignment("payDistanceProfileId"),
  assignment("billingDistanceProfileId"),
  assignment("fuelDistanceProfileId"),
];

function ProfileRows({ rows }: { rows: readonly Assignment[] }) {
  const t = useT();
  const { control } = useFormContext<DistanceControl>();

  return rows.map((row) => (
    <DistanceProfileAutocompleteField<DistanceControl>
      key={row.name}
      control={control}
      name={row.name}
      label={row.label}
      description={row.description}
      placeholder={t("Select distance profile")}
      required
    />
  ));
}

/**
 * How Trenova measures distance: which distance profile routes each kind of mile, whether
 * lane mileage already calculated is kept and reused, whether each route is broken down by
 * jurisdiction, and the unit rating reads. City matching only matters while mileage is kept
 * or reused, so it opens under those two switches.
 */
export function MileageTab() {
  const t = useT();
  const { control } = useFormContext<DistanceControl>();
  const [reuse, keep] = useWatch({ control, name: ["storeMileage", "autoCreateStoredMileage"] });

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Routing")}
        description={t("The distance profile PC*Miler routes each kind of trip with.")}
      >
        <ProfileRows rows={MOVES} />
      </SettingsSection>
      <SettingsSection
        title={t("Pay, billing and fuel")}
        description={t("The distance profile each kind of mileage is measured with.")}
      >
        <ProfileRows rows={SETTLEMENT} />
      </SettingsSection>
      <SettingsSection
        title={t("Stored mileage")}
        description={t(
          "Mileage kept from earlier routes, so a lane already routed is not sent to PC*Miler again.",
        )}
      >
        <SwitchField
          control={control}
          name="storeMileage"
          label={t("Reuse stored mileage")}
          description={t("Look up a lane's stored mileage before asking PC*Miler.")}
        />
        <SwitchField
          control={control}
          name="autoCreateStoredMileage"
          label={t("Keep new mileage")}
          description={t("Save each PC*Miler result as stored mileage for the lane.")}
        />
        <SettingsNested show={reuse || keep}>
          <SwitchField
            control={control}
            name="postalCodeFallbackToCity"
            label={t("Match by city")}
            description={t(
              "A stop with no postal code is matched by its city and state instead of skipped.",
            )}
          />
        </SettingsNested>
      </SettingsSection>
      <SettingsSection
        title={t("Jurisdiction miles")}
        description={t("Miles by state or province, which fuel tax returns are filed from.")}
      >
        <SwitchField
          control={control}
          name="captureJurisdictionMiles"
          label={t("Capture jurisdiction miles")}
          description={t("Ask PC*Miler for each move's miles by state or province.")}
          tooltip={t(
            "PC*Miler may bill this as an extra transaction per route. Routes calculated before it is on have no breakdown until they are recalculated.",
          )}
        />
      </SettingsSection>
      <SettingsSection
        title={t("Rating")}
        description={t("How tariffs and rate tables read distance.")}
      >
        <SelectField
          control={control}
          name="ratingDistanceUnit"
          label={t("Distance unit")}
          description={t("The unit a tariff's distance and its rate table rows are in.")}
          options={distanceControlUnitChoices}
          required
        />
      </SettingsSection>
    </SettingsSections>
  );
}
