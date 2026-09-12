import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import {
  distanceProfileDistanceUnitChoices,
  distanceProfileLocationGranularityChoices,
  distanceProfileProviderChoices,
  distanceProfileRegionChoices,
  distanceProfileRoutingTypeChoices,
  distanceProfileStatusChoices,
} from "@/lib/choices";
import type { DistanceProfile } from "@/types/distance-profile";
import { type Control, useFormContext } from "react-hook-form";

function ProfileDetailsSection({ control }: { control: Control<DistanceProfile> }) {
  const t = useT();

  return (
    <FormSection
      title={t("Profile Details")}
      description={t("Name, lifecycle state, and default selection for this business unit.")}
    >
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name="name"
            label={t("Name")}
            placeholder={t("Default PC*Miler")}
            description={t("A clear name dispatch and rating teams can recognize.")}
            rules={{ required: true }}
            maxLength={100}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="status"
            label={t("Status")}
            placeholder={t("Status")}
            description={t("Inactive profiles cannot be used as the default.")}
            options={distanceProfileStatusChoices}
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label={t("Description")}
            placeholder={t("Describe when this routing policy should be used.")}
            description={t("Optional notes that explain the operating policy behind this profile.")}
            minRows={3}
          />
        </FormControl>
        <FormControl cols="full">
          <SwitchField
            control={control}
            name="isDefault"
            label={t("Default profile")}
            description={t(
              "Use this profile when no distance override applies to a shipment move.",
            )}
            outlined
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

function ProviderPolicySection({ control }: { control: Control<DistanceProfile> }) {
  const t = useT();

  return (
    <FormSection
      title={t("Provider Policy")}
      description={t("PC*Miler dataset, geography, route type, units, and stop matching policy.")}
      className="border-t py-2"
    >
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            name="provider"
            label={t("Provider")}
            placeholder={t("Provider")}
            description={t("Distance profiles currently support PC*Miler routing policy.")}
            options={distanceProfileProviderChoices}
            rules={{ required: true }}
            isReadOnly
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="dataVersion"
            label={t("Data Version")}
            placeholder={t("Current")}
            description={t("PC*Miler data version sent with Route Reports requests.")}
            rules={{ required: true }}
            maxLength={50}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="region"
            label={t("Region")}
            placeholder={t("Region")}
            description={t("Routing data region used for all stops in this profile.")}
            options={distanceProfileRegionChoices}
            rules={{ required: true }}
            isReadOnly
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="routingType"
            label={t("Routing Type")}
            placeholder={t("Routing Type")}
            description={t("Controls how PC*Miler chooses roads for calculated distance.")}
            options={distanceProfileRoutingTypeChoices}
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="distanceUnits"
            label={t("Distance Units")}
            placeholder={t("Distance Units")}
            description={t("Unit stored on shipment moves and returned in calculation summaries.")}
            options={distanceProfileDistanceUnitChoices}
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="locationGranularity"
            label={t("Location Granularity")}
            placeholder={t("Location Granularity")}
            description={t("Determines which location fields are sent to PC*Miler stops.")}
            options={distanceProfileLocationGranularityChoices}
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            name="profileName"
            label={t("PC*Miler Profile Name")}
            placeholder={t("Optional Trimble vehicle profile")}
            description={t(
              "Optional PC*Miler vehicle profile name for account-specific routing settings.",
            )}
            maxLength={100}
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

function RouteBehaviorSection({ control }: { control: Control<DistanceProfile> }) {
  const t = useT();

  return (
    <FormSection
      title={t("Route Behavior")}
      description={t(
        "Road restrictions and supplemental reporting options sent with mileage requests.",
      )}
      className="border-t py-2"
    >
      <FormGroup cols={2}>
        <FormControl>
          <SwitchField
            control={control}
            name="highwayOnly"
            label={t("Highway only")}
            description={t("Restrict routing to highways where PC*Miler supports the option.")}
            outlined
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name="tollRoads"
            label={t("Allow toll roads")}
            description={t("Permit toll roads when PC*Miler selects the route.")}
            outlined
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name="bordersOpen"
            label={t("Borders open")}
            description={t("Allow cross-border routes when stops span supported regions.")}
            outlined
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name="includeTollData"
            label={t("Include toll data")}
            description={t("Request toll reporting details in the PC*Miler response.")}
            outlined
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

export function DistanceProfileForm() {
  const { control } = useFormContext<DistanceProfile>();

  return (
    <div className="flex flex-col">
      <ProfileDetailsSection control={control} />
      <ProviderPolicySection control={control} />
      <RouteBehaviorSection control={control} />
    </div>
  );
}
