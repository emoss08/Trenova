import {
  EDIConnectionAutocompleteField,
  EDIPartnerAutocompleteField,
  OrganizationAutocompleteField,
} from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { statusChoices } from "@/lib/choices";
import type { EDICommunicationProfile } from "@/types/edi";
import { useFormContext, useWatch } from "react-hook-form";
import { communicationProfileMethodOptions, type CommunicationProfileFormValues } from "../edi-schemas";
import {
  SecretProfileFields,
  TransportProfileFields,
  X12EnvelopeFields,
} from "./edi-communication-profile-fields";
import { EDIEmptyState } from "./edi-panel-primitives";

export function OverviewTab() {
  const { control } = useFormContext<CommunicationProfileFormValues>();
  const method = useWatch({ control, name: "method" });

  return (
    <div className="space-y-3">
      <FormSection title="Profile Identity">
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="name"
              label="Name"
              placeholder="Profile name"
              description="Unique name for this profile."
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="method"
              label="Method"
              options={communicationProfileMethodOptions}
              description="The method used to communicate with the trading partner."
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label="Status"
              options={statusChoices}
              description="The status of this profile."
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <EDIPartnerAutocompleteField
              control={control}
              name="ediPartnerId"
              label="Partner"
              placeholder="Select partner"
              description="Trading partner this transport profile delivers documents for."
              clearable
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="description"
              label="Description"
              placeholder="Operational notes for this profile"
              description="Additional notes about this profile."
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      {method === "Internal" && (
        <FormSection title="Internal Routing" className="rounded-md border bg-muted/20 p-3">
          <FormGroup cols={2}>
            <FormControl>
              <EDIConnectionAutocompleteField
                control={control}
                name="ediConnectionId"
                label="Connection"
                placeholder="Select connection"
                description="Accepted organization connection this profile routes through."
                clearable
              />
            </FormControl>
            <FormControl>
              <OrganizationAutocompleteField
                control={control}
                name="config.connectedOrganizationId"
                label="Connected Organization"
                placeholder="Select organization"
                description="Organization that receives documents delivered over this profile."
                clearable
              />
            </FormControl>
          </FormGroup>
        </FormSection>
      )}
    </div>
  );
}

export function TransportTab() {
  const { control } = useFormContext<CommunicationProfileFormValues>();
  const method = useWatch({ control, name: "method" });
  const authMode = useWatch({ control, name: "config.authMode" });

  return (
    <div className="space-y-3">
      <TransportProfileFields control={control} method={method} authMode={authMode} />
    </div>
  );
}

export function EnvelopeTab() {
  const { control } = useFormContext<CommunicationProfileFormValues>();
  const method = useWatch({ control, name: "method" });

  return (
    <div className="space-y-3">
      {method === "Internal" ? (
        <EDIEmptyState message="Internal profiles use organization routing and do not require X12 interchange identifiers." />
      ) : (
        <X12EnvelopeFields control={control} />
      )}
    </div>
  );
}

export function SecretsTab({ profile }: { profile: EDICommunicationProfile | null }) {
  const { control } = useFormContext<CommunicationProfileFormValues>();
  const method = useWatch({ control, name: "method" });
  const authMode = useWatch({ control, name: "config.authMode" });

  return (
    <div className="space-y-3">
      <SecretProfileFields control={control} method={method} profile={profile} authMode={authMode} />
    </div>
  );
}
