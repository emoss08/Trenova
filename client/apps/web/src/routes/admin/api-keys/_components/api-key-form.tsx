import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { useFormContext } from "react-hook-form";
import type { ApiKeyPanelFormValues } from "./api-key-panel";

export function APIKeyForm() {
  const t = useT();

  const { control } = useFormContext<ApiKeyPanelFormValues>();

  return (
    <FormSection
      title={t("Key details")}
      description={t(
        "Name the credential, document its intended use, and define an expiration if the integration should rotate automatically.",
      )}
    >
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name="name"
            label={t("Display name")}
            placeholder={t("Warehouse connector")}
            description={t("Used in the API keys table and audit trail.")}
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="expiresAtInput"
            type="datetime-local"
            label={t("Expiration")}
            description={t("Leave blank to keep the key active until revoked.")}
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label={t("Description")}
            placeholder={t(
              "Describe the partner system, deployment target, or workflow using this key.",
            )}
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}
