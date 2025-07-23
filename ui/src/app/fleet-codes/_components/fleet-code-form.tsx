/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { UserAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
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
          maxLength={100}
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
        <NumberField
          control={control}
          name="deadheadGoal"
          label="Deadhead Goal"
          placeholder="Deadhead Goal"
          description="The deadhead goal of the fleet code"
        />
      </FormControl>
      <FormControl>
        <NumberField
          control={control}
          name="revenueGoal"
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
          label="Manager"
          placeholder="Select Manager"
          description="Select the manager of the fleet code"
        />
      </FormControl>
    </FormGroup>
  );
}
