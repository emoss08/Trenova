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
          description="Current status of the user"
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
          description="The username of the user"
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          rules={{ required: true }}
          name="emailAddress"
          label="Email Address"
          placeholder="Email Address"
          description="The email address of the user"
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Full Name"
          placeholder="Full Name"
          description="The full name of the user"
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
          description="The timezone of the user"
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
          rules={{ required: true }}
          label="Time Format"
          placeholder="Select time format"
          description="The time format of the user"
        />
      </FormControl>
      <FormControl cols="full">
        <SwitchField
          size="xs"
          name="isLocked"
          control={control}
          label="Is Locked"
          outlined
          description="Specifies whether the user is locked."
        />
      </FormControl>
      <FormControl cols="full">
        <RoleAutocompleteField
          control={control}
          name="roles"
          label="Roles"
          description="The roles of the user"
          placeholder="Select roles"
          rules={{ required: true }}
        />
      </FormControl>
    </FormGroup>
  );
}
