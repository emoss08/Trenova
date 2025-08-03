/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { RoleAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { statusChoices } from "@/lib/choices";
import { UserSchema } from "@/lib/schemas/user-schema";
import { TIMEZONES } from "@/lib/timezone/timezone";
import { useFormContext } from "react-hook-form";

export function UserForm() {
  const { control } = useFormContext<UserSchema>();

  return (
    <FormGroup cols={2}>
      <FormControl cols="full">
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="Account activation status"
          options={statusChoices}
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          rules={{ required: true }}
          name="username"
          label="Username"
          placeholder="Username"
          description="Unique login identifier"
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          rules={{ required: true }}
          name="emailAddress"
          label="Email Address"
          placeholder="Email Address"
          description="Primary contact email"
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Full Name"
          placeholder="Full Name"
          description="Legal first and last name"
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          name="timezone"
          options={TIMEZONES.map((timezone) => ({
            label: timezone.label,
            value: timezone.value,
            color: timezone.color,
            description: timezone.description,
          }))}
          rules={{ required: true }}
          label="Timezone"
          placeholder="Select timezone"
          description="Local time zone for scheduling and notifications"
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          name="timeFormat"
          options={[
            {
              label: "12-hour",
              value: "12-hour",
            },
            {
              label: "24-hour",
              value: "24-hour",
            },
          ]}
          label="Time Format"
          placeholder="Select time format"
          description="Preferred time display format"
        />
      </FormControl>
      <FormControl cols="full">
        <SwitchField
          size="xs"
          name="isLocked"
          control={control}
          label="Is Locked"
          outlined
          description="Account access restriction status"
        />
      </FormControl>
      <FormControl cols="full">
        <RoleAutocompleteField
          control={control}
          name="roles"
          label="Roles"
          description="System access permissions and privileges"
          placeholder="Select roles"
        />
      </FormControl>
    </FormGroup>
  );
}
