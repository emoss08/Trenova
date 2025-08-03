/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { providerTypeChoices, statusChoices } from "@/lib/choices";
import {
  ProviderType,
  type EmailProfileSchema,
} from "@/lib/schemas/email-profile-schema";
import { useFormContext, useWatch } from "react-hook-form";
import { SmtpFormFields } from "./_providers/smtp-form-fields";

const renderProviderForm = (provider: EmailProfileSchema["providerType"]) => {
  switch (provider) {
    case ProviderType.enum.SMTP:
      return <SmtpFormFields />;
    default:
      return <div>No provider form found</div>;
  }
};

export function EmailProfileForm() {
  const { control } = useFormContext();

  const provider = useWatch({ name: "providerType" });

  return (
    <>
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            options={statusChoices}
            rules={{ required: true }}
            name="status"
            label="Status"
            placeholder="Status"
            description="Current operational status of this email configuration"
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            placeholder="Name"
            description="Unique identifier for this email profile configuration"
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label="Description"
            placeholder="Description"
            description="Description of this email profile configuration"
          />
        </FormControl>
        <FormControl cols="full">
          <SelectField
            control={control}
            options={providerTypeChoices}
            rules={{ required: true }}
            name="providerType"
            label="Provider Type"
            placeholder="Provider Type"
            description="Email service provider to handle message delivery"
          />
        </FormControl>
        <FormControl cols="full">
          <SwitchField
            control={control}
            name="isDefault"
            label="Default Profile?"
            outlined
            description="Whether this is the default email profile"
          />
        </FormControl>
      </FormGroup>
      {renderProviderForm(provider)}
    </>
  );
}
