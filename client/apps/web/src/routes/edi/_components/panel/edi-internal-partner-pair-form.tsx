import { OrganizationAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { Form, FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { OrganizationSelectOption } from "@trenova/shared/types/organization";
import type { Control, UseFormReturn } from "react-hook-form";
import type { CreateInternalPartnerPairFormValues } from "../edi-schemas";

type InternalPartnerPairFormProps = {
  id: string;
  form: UseFormReturn<CreateInternalPartnerPairFormValues>;
  onSubmit: (values: CreateInternalPartnerPairFormValues) => void;
  onTargetOrganizationChange: (organization: OrganizationSelectOption | null) => void;
};

export function InternalPartnerPairForm({
  id,
  form,
  onSubmit,
  onTargetOrganizationChange,
}: InternalPartnerPairFormProps) {
  const { control, handleSubmit } = form;

  return (
    <Form
      id={id}
      className="flex flex-col gap-6"
      onSubmit={(event) => {
        event.stopPropagation();
        void handleSubmit(onSubmit)(event);
      }}
    >
      <FormSection
        title="Organization Pairing"
        description="Choose the organization to connect with and confirm the reciprocal partner records that will be created."
      >
        <FormGroup cols={2} className="gap-x-5 gap-y-3">
          <FormControl cols="full">
            <OrganizationAutocompleteField
              control={control}
              name="targetOrganizationId"
              label="Target Organization"
              placeholder="Select organization"
              description="Organization that will receive the connection request. The current organization is excluded from this list."
              rules={{ required: true }}
              extraSearchParams={{
                scope: "business-unit",
                excludeCurrent: "true",
              }}
              onOptionChange={onTargetOrganizationChange}
            />
          </FormControl>
          <PartnerSideFields
            title="Current Organization View"
            description="Partner record created in your current organization to represent the selected organization."
            prefix="source"
            control={control}
          />
          <PartnerSideFields
            title="Target Organization View"
            description="Partner record created in the selected organization to represent your current organization."
            prefix="target"
            control={control}
          />
        </FormGroup>
      </FormSection>
    </Form>
  );
}

function PartnerSideFields({
  title,
  description,
  prefix,
  control,
}: {
  title: string;
  description: string;
  prefix: "source" | "target";
  control: Control<CreateInternalPartnerPairFormValues>;
}) {
  const codeName = `${prefix}Code` as const;
  const partnerName = `${prefix}Name` as const;
  const contactName = `${prefix}ContactName` as const;
  const contactEmail = `${prefix}ContactEmail` as const;
  const contactPhone = `${prefix}ContactPhone` as const;
  const inboundName = `${prefix}EnabledForInbound` as const;
  const outboundName = `${prefix}EnabledForOutbound` as const;

  return (
    <FormSection
      title={title}
      description={description}
      className="rounded-md border bg-muted/20 p-3"
    >
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name={codeName}
            label="Partner Code"
            placeholder="Partner code"
            description="Stable code used to identify this organization in internal EDI routing and connection records."
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name={partnerName}
            label="Partner Name"
            placeholder="Partner name"
            description="Display name shown on the reciprocal partner record after the connection is accepted."
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name={contactName}
            label="Contact Name"
            placeholder="Contact name"
            description="Operational owner for questions about this side of the internal connection."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name={contactEmail}
            label="Contact Email"
            placeholder="ops@example.com"
            description="Email address used for coordination if the internal connection needs attention."
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            name={contactPhone}
            label="Contact Phone"
            placeholder="Contact phone"
            description="Phone number for urgent operational follow-up about this connection."
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name={inboundName}
            label="Inbound Enabled"
            description="Allow this partner record to receive load tenders from the connected organization."
            outlined
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name={outboundName}
            label="Outbound Enabled"
            description="Allow this partner record to send load tenders to the connected organization."
            outlined
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}
