import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { useFormContext } from "react-hook-form";
import {
  emailProfileStatusChoices,
  emailProviderChoices,
  type EmailProfileFormValues,
} from "./email-profile-constants";

export function EmailProfileForm() {
  const { control } = useFormContext<EmailProfileFormValues>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Profile Name"
          placeholder="Billing sender"
          description="Internal label used when assigning this sender profile."
          maxLength={100}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="Inactive profiles cannot be assigned to purposes."
          options={emailProfileStatusChoices}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="provider"
          label="Provider"
          placeholder="Provider"
          description="Email service provider used for this sender identity."
          options={emailProviderChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="senderName"
          label="Sender Name"
          placeholder="Trenova Billing"
          description="Display name recipients see in their inbox."
          maxLength={100}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="senderEmail"
          label="Sender Email"
          placeholder="billing@example.com"
          description="Verified sender email address for this profile."
          type="email"
          maxLength={320}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="replyToEmail"
          label="Reply-To Email"
          placeholder="replies@example.com"
          description="Optional reply destination. Blank uses the sender email."
          type="email"
          maxLength={320}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Usage notes for this profile"
          description="Operational notes for admins choosing sender identities."
        />
      </FormControl>
    </FormGroup>
  );
}
