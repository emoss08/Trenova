import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { useFormContext } from "react-hook-form";
import type { ApiKeyPanelFormValues } from "./api-key-panel";

export function APIKeyForm() {
  const { control } = useFormContext<ApiKeyPanelFormValues>();

  return (
    <section className="space-y-4">
      <div className="space-y-1">
        <h3 className="text-sm font-semibold">Key Details</h3>
        <p className="text-sm text-muted-foreground">
          Name the credential, document its intended use, and define an
          expiration if the integration should rotate automatically.
        </p>
      </div>

      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name="name"
            label="Display Name"
            placeholder="Warehouse connector"
            description="Used in the API keys table and audit trail."
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="expiresAtInput"
            type="datetime-local"
            label="Expiration"
            description="Leave blank to keep the key active until revoked."
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label="Description"
            placeholder="Describe the partner system, deployment target, or workflow using this key."
          />
        </FormControl>
      </FormGroup>
    </section>
  );
}
