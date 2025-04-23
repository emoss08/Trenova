import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { UserAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { statusChoices } from "@/lib/choices";
import { type FleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import { useFormContext } from "react-hook-form";

export function FleetCodeForm() {
  const { control } = useFormContext<FleetCodeSchema>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="The status of the fleet code"
          options={statusChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Name"
          description="The name of the fleet code"
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Description"
          description="The description of the fleet code"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="deadheadGoal"
          type="number"
          label="Deadhead Goal"
          placeholder="Deadhead Goal"
          description="The deadhead goal of the fleet code"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="revenueGoal"
          type="number"
          label="Revenue Goal"
          placeholder="Revenue Goal"
          description="The revenue goal of the fleet code"
        />
      </FormControl>
      <FormControl>
        <ColorField
          control={control}
          name="color"
          label="Color"
          description="The color of the fleet code"
        />
      </FormControl>
      <FormControl>
        <UserAutocompleteField<FleetCodeSchema>
          name="managerId"
          control={control}
          link="/users/"
          label="Manager"
          placeholder="Select Manager"
          description="Select the manager of the fleet code"
        />
      </FormControl>
    </FormGroup>
  );
}
