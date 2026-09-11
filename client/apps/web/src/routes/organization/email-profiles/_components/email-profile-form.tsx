import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useFormContext } from "react-hook-form";
import {
  emailProfileStatusChoices,
  emailProviderChoices,
  type EmailProfileFormValues,
} from "./email-profile-constants";

export function EmailProfileForm() {
  const t = useT();

  const { control } = useFormContext<EmailProfileFormValues>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label={t("Profile Name")}
          placeholder={t("Billing sender")}
          description={t("Internal label used when assigning this sender profile.")}
          maxLength={100}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label={t("Status")}
          placeholder={t("Status")}
          description={t("Inactive profiles cannot be assigned to purposes.")}
          options={emailProfileStatusChoices}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="provider"
          label={t("Provider")}
          placeholder={t("Provider")}
          description={t("Email service provider used for this sender identity.")}
          options={emailProviderChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="senderName"
          label={t("Sender Name")}
          placeholder={t("Trenova Billing")}
          description={t("Display name recipients see in their inbox.")}
          maxLength={100}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="senderEmail"
          label={t("Sender Email")}
          placeholder={t("billing@example.com")}
          description={t("Verified sender email address for this profile.")}
          type="email"
          maxLength={320}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="replyToEmail"
          label={t("Reply-To Email")}
          placeholder={t("replies@example.com")}
          description={t("Optional reply destination. Blank uses the sender email.")}
          type="email"
          maxLength={320}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label={t("Description")}
          placeholder={t("Usage notes for this profile")}
          description={t("Operational notes for admins choosing sender identities.")}
        />
      </FormControl>
    </FormGroup>
  );
}
