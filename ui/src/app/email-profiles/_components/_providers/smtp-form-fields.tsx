import { InputField } from "@/components/fields/input-field";
import { SensitiveInputField } from "@/components/fields/sensitive-input-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { EmailProfileSchema } from "@/lib/schemas/email-profile-schema";
import { useFormContext } from "react-hook-form";

export function SmtpFormFields() {
  const { control } = useFormContext<EmailProfileSchema>();

  return (
    <FormSection
      title="SMTP Configuration"
      description="Configure SMTP server settings for direct email delivery"
      className="pt-4"
    >
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name="host"
            label="Host"
            placeholder="Host"
            description="SMTP server hostname or IP address"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="port"
            rules={{ required: true }}
            label="Port"
            type="number"
            placeholder="Port"
            description="SMTP server port number (typically 25, 465, or 587)"
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            rules={{ required: true }}
            name="fromAddress"
            label="From Address"
            placeholder="From Address"
            description="Default sender email address for outgoing messages"
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            rules={{ required: true }}
            name="username"
            label="Username"
            placeholder="Username"
            description="SMTP authentication username for server access"
          />
        </FormControl>
        <FormControl cols="full">
          <SensitiveInputField
            control={control}
            rules={{ required: true }}
            name="password"
            label="Password"
            type="password"
            placeholder="Password"
            description="SMTP authentication password for server access"
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}
