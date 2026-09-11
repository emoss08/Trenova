import { useT } from "@trenova/shared/i18n/use-t";
import { DocumentTypeAutocompleteField } from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { Separator } from "@trenova/shared/components/ui/separator";
import { resourceTypeChoices } from "@/lib/choices";
import type { DocumentPacketRule } from "@/types/document-packet-rule";
import { useFormContext, useWatch } from "react-hook-form";

export function DocumentPacketRuleForm({ disabled }: { disabled?: boolean }) {
  const t = useT();

  const { control } = useFormContext<DocumentPacketRule>();
  const expirationRequired = useWatch({ control, name: "expirationRequired" });

  return (
    <div className="flex flex-col gap-4">
      <FormSection
        title={t("Rule Target")}
        description={t("Which resource type and document type does this rule apply to?")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="resourceType"
              label={t("Resource Type")}
              placeholder={t("Select resource type")}
              description={t("Shipment, trailer, tractor, or worker")}
              options={resourceTypeChoices}
              isReadOnly={disabled}
            />
          </FormControl>
          <FormControl>
            <DocumentTypeAutocompleteField
              control={control}
              rules={{ required: true }}
              name="documentTypeId"
              label={t("Document Type")}
              placeholder={t("Select document type")}
              description={t("The document type required by this rule")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <Separator />

      <FormSection
        title={t("Rule Behavior")}
        description={t("Configure how this document requirement is enforced")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="required"
              label={t("Required")}
              description={t("Mark this document as mandatory for compliance")}
              disabled={disabled}
              position="left"
              outlined
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowMultiple"
              label={t("Allow Multiple")}
              description={t("Allow more than one document of this type")}
              disabled={disabled}
              position="left"
              outlined
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="displayOrder"
              label={t("Display Order")}
              placeholder="0"
              description={t("Lower numbers appear first in the packet")}
              disabled={disabled}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <Separator />

      <FormSection
        title={t("Expiration Tracking")}
        description={t("Optionally require an expiration date and configure early warnings")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="expirationRequired"
              label={t("Expiration Required")}
              description={t("Documents must include an expiration date")}
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
                label={t("Warning Days")}
                placeholder="30"
                description={t("Days before expiration to flag as expiring soon")}
                disabled={disabled}
              />
            </FormControl>
          )}
        </FormGroup>
      </FormSection>
    </div>
  );
}
