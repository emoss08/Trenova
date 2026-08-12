import { LocationAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { statusChoices, usStateAbbreviationChoices } from "@/lib/choices";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import {
  deriveLaneMatchMode,
  type LaneMatchMode,
  type RoutingGuidePayloadInput,
} from "@trenova/shared/types/routing-guide";
import { useCallback, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { RoutingGuideEntryEditor } from "./routing-guide-entry-editor";

const LANE_MODE_ITEMS: { value: LaneMatchMode; label: string; caption: string }[] = [
  {
    value: "locations",
    label: "Exact locations",
    caption: "Matches one facility pair",
  },
  {
    value: "cityState",
    label: "City + state",
    caption: "Matches a metro lane",
  },
  {
    value: "stateOnly",
    label: "State only",
    caption: "Matches a regional lane",
  },
];

/**
 * The guide builder. The lane predicate is tiered exactly like the server's
 * matcher: exact locations beat city+state, which beats state-only, and both
 * ends of the lane must be described at the same tier.
 */
export function RoutingGuideForm() {
  const { control, setValue, clearErrors } = useFormContext<RoutingGuidePayloadInput>();

  const originLocationId = useWatch({ control, name: "originLocationId" });
  const destinationLocationId = useWatch({ control, name: "destinationLocationId" });
  const originCity = useWatch({ control, name: "originCity" });
  const destinationCity = useWatch({ control, name: "destinationCity" });
  const originState = useWatch({ control, name: "originState" });
  const destinationState = useWatch({ control, name: "destinationState" });

  const [modeOverride, setModeOverride] = useState<LaneMatchMode | null>(null);

  const hasLaneValue = Boolean(
    originLocationId ||
    destinationLocationId ||
    originCity ||
    destinationCity ||
    originState ||
    destinationState,
  );
  const laneMode =
    modeOverride ??
    (hasLaneValue
      ? deriveLaneMatchMode({
          originLocationId,
          destinationLocationId,
          originCity,
          destinationCity,
        })
      : "locations");

  const switchLaneMode = useCallback(
    (mode: LaneMatchMode) => {
      setModeOverride(mode);
      // Fields belonging to the other tiers are cleared so the payload the
      // server sees describes exactly one tier per side — mixed criteria are a
      // validation error there.
      if (mode !== "locations") {
        setValue("originLocationId", null, { shouldDirty: true });
        setValue("destinationLocationId", null, { shouldDirty: true });
      }
      if (mode !== "cityState") {
        setValue("originCity", "", { shouldDirty: true });
        setValue("destinationCity", "", { shouldDirty: true });
      }
      if (mode === "locations") {
        setValue("originState", "", { shouldDirty: true });
        setValue("destinationState", "", { shouldDirty: true });
      }
      clearErrors([
        "originLocationId",
        "destinationLocationId",
        "originCity",
        "originState",
        "destinationCity",
        "destinationState",
      ]);
    },
    [clearErrors, setValue],
  );

  return (
    <div className="flex flex-col gap-6">
      <FormSection title="Identity" description="How this guide is referenced across dispatch">
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="name"
              label="Name"
              placeholder="Dallas → Atlanta Dry Van"
              rules={{ required: true }}
              maxLength={255}
              description="Shown wherever a tender references this guide."
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label="Status"
              placeholder="Select status"
              rules={{ required: true }}
              options={statusChoices}
              description="Only Active guides are matched when a move is tendered."
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="description"
              label="Description"
              placeholder="Primary waterfall for the Dallas–Atlanta contract freight"
              description="Context for the next person who has to understand this ladder."
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Lane"
        description="Which lane this guide covers. The most specific matching guide wins: exact locations beat city+state, which beats state-only."
      >
        <div className="flex flex-col gap-3">
          <SegmentedControl<LaneMatchMode>
            items={LANE_MODE_ITEMS}
            value={laneMode}
            onValueChange={switchLaneMode}
            fullWidth
            aria-label="Lane match level"
          />

          <FormGroup cols={2}>
            {laneMode === "locations" && (
              <>
                <FormControl>
                  <LocationAutocompleteField
                    control={control}
                    name="originLocationId"
                    label="Origin Location"
                    placeholder="Select origin facility"
                    rules={{ required: true }}
                  />
                </FormControl>
                <FormControl>
                  <LocationAutocompleteField
                    control={control}
                    name="destinationLocationId"
                    label="Destination Location"
                    placeholder="Select destination facility"
                    rules={{ required: true }}
                  />
                </FormControl>
              </>
            )}

            {laneMode === "cityState" && (
              <>
                <FormControl>
                  <InputField
                    control={control}
                    name="originCity"
                    label="Origin City"
                    placeholder="Dallas"
                    rules={{ required: true }}
                    maxLength={100}
                  />
                </FormControl>
                <FormControl>
                  <SelectField
                    control={control}
                    name="originState"
                    label="Origin State"
                    placeholder="Select state"
                    rules={{ required: true }}
                    options={usStateAbbreviationChoices}
                  />
                </FormControl>
                <FormControl>
                  <InputField
                    control={control}
                    name="destinationCity"
                    label="Destination City"
                    placeholder="Atlanta"
                    rules={{ required: true }}
                    maxLength={100}
                  />
                </FormControl>
                <FormControl>
                  <SelectField
                    control={control}
                    name="destinationState"
                    label="Destination State"
                    placeholder="Select state"
                    rules={{ required: true }}
                    options={usStateAbbreviationChoices}
                  />
                </FormControl>
              </>
            )}

            {laneMode === "stateOnly" && (
              <>
                <FormControl>
                  <SelectField
                    control={control}
                    name="originState"
                    label="Origin State"
                    placeholder="Select state"
                    rules={{ required: true }}
                    options={usStateAbbreviationChoices}
                  />
                </FormControl>
                <FormControl>
                  <SelectField
                    control={control}
                    name="destinationState"
                    label="Destination State"
                    placeholder="Select state"
                    rules={{ required: true }}
                    options={usStateAbbreviationChoices}
                  />
                </FormControl>
              </>
            )}
          </FormGroup>
        </div>
      </FormSection>

      <FormSection
        title="Carrier Waterfall"
        description="Ranked carriers with the rate, offer expiry, and channel each is tendered on"
      >
        <RoutingGuideEntryEditor />
      </FormSection>
    </div>
  );
}
