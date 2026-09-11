import { useT } from "@trenova/shared/i18n/use-t";
import { OrganizationAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { Form, FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import type { Control, UseFormReturn } from "react-hook-form";
import type { CreateInternalPartnerPairFormValues } from "../edi-schemas";

type InternalPartnerPairFormProps = {
  id: string;
  form: UseFormReturn<CreateInternalPartnerPairFormValues>;
  onSubmit: (values: CreateInternalPartnerPairFormValues) => void;
  onTargetOrganizationChange: (organization: GraphQLSelectOption | null) => void;
};

export function InternalPartnerPairForm({
  id,
  form,
  onSubmit,
  onTargetOrganizationChange,
}: InternalPartnerPairFormProps) {
  const t = useT();

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
        title={t("Organization Pairing")}
        description={t("Choose the organization to connect with and confirm the reciprocal partner records that will be created.")}
      >
        <FormGroup cols={2} className="gap-x-5 gap-y-3">
          <FormControl cols="full">
            <OrganizationAutocompleteField
              control={control}
              name="targetOrganizationId"
              label={t("Target Organization")}
              placeholder={t("Select organization")}
              description={t("Organization that will receive the connection request. The current organization is excluded from this list.")}
              rules={{ required: true }}
              extraSearchParams={{
                scope: "business-unit",
                excludeCurrent: "true",
              }}
              onOptionChange={onTargetOrganizationChange}
            />
          </FormControl>
          <PartnerSideFields
            title={t("Current Organization View")}
            description={t("Partner record created in your current organization to represent the selected organization.")}
            prefix="source"
            control={control}
          />
          <PartnerSideFields
            title={t("Target Organization View")}
            description={t("Partner record created in the selected organization to represent your current organization.")}
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
  const t = useT();

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
      className="bg-muted/20 rounded-md border p-3"
    >
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name={codeName}
            label={t("Partner Code")}
            placeholder={t("Partner code")}
            description={t("Stable code used to identify this organization in internal EDI routing and connection records.")}
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name={partnerName}
            label={t("Partner Name")}
            placeholder={t("Partner name")}
            description={t("Display name shown on the reciprocal partner record after the connection is accepted.")}
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name={contactName}
            label={t("Contact Name")}
            placeholder={t("Contact name")}
            description={t("Operational owner for questions about this side of the internal connection.")}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name={contactEmail}
            label={t("Contact Email")}
            placeholder={t("ops@example.com")}
            description={t("Email address used for coordination if the internal connection needs attention.")}
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            name={contactPhone}
            label={t("Contact Phone")}
            placeholder={t("Contact phone")}
            description={t("Phone number for urgent operational follow-up about this connection.")}
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name={inboundName}
            label={t("Inbound Enabled")}
            description={t("Allow this partner record to receive load tenders from the connected organization.")}
            outlined
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name={outboundName}
            label={t("Outbound Enabled")}
            description={t("Allow this partner record to send load tenders to the connected organization.")}
            outlined
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}
