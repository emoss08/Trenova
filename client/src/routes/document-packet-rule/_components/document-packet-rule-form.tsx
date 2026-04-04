import { DocumentTypeAutocompleteField } from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { resourceTypeChoices } from "@/lib/choices";
import type { DocumentPacketRule } from "@/types/document-packet-rule";
import { useFormContext, useWatch } from "react-hook-form";

export function DocumentPacketRuleForm({ disabled }: { disabled?: boolean }) {
  const { control } = useFormContext<DocumentPacketRule>();
  const expirationRequired = useWatch({ control, name: "expirationRequired" });

  return (
    <div className="flex flex-col gap-4">
      <FormSection
        title="Rule Target"
        description="Which resource type and document type does this rule apply to?"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="resourceType"
              label="Resource Type"
              placeholder="Select resource type"
              description="Shipment, trailer, tractor, or worker"
              options={resourceTypeChoices}
              isReadOnly={disabled}
            />
          </FormControl>
          <FormControl>
            <DocumentTypeAutocompleteField
              control={control}
              rules={{ required: true }}
              name="documentTypeId"
              label="Document Type"
              placeholder="Select document type"
              description="The document type required by this rule"
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <Separator />

      <FormSection
        title="Rule Behavior"
        description="Configure how this document requirement is enforced"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="required"
              label="Required"
              description="Mark this document as mandatory for compliance"
              disabled={disabled}
              position="left"
              outlined
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowMultiple"
              label="Allow Multiple"
              description="Allow more than one document of this type"
              disabled={disabled}
              position="left"
              outlined
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="displayOrder"
              label="Display Order"
              placeholder="0"
              description="Lower numbers appear first in the packet"
              disabled={disabled}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <Separator />

      <FormSection
        title="Expiration Tracking"
        description="Optionally require an expiration date and configure early warnings"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="expirationRequired"
              label="Expiration Required"
              description="Documents must include an expiration date"
              disabled={disabled}
              position="left"
              outlined
            />
          </FormControl>
          {expirationRequired && (
            <FormControl>
              <NumberField
                control={control}
                name="expirationWarningDays"
                label="Warning Days"
                placeholder="30"
                description="Days before expiration to flag as expiring soon"
                disabled={disabled}
              />
            </FormControl>
          )}
        </FormGroup>
      </FormSection>
    </div>
  );
}
