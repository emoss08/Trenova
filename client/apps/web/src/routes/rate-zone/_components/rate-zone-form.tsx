import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { rateZoneKindChoices, statusChoices } from "@/lib/choices";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import type { RateZone } from "@trenova/shared/types/rate";
import { useFormContext } from "react-hook-form";
import { ZoneMemberEditor } from "./zone-member-editor";

export function RateZoneForm() {
  const { control } = useFormContext<RateZone>();

  return (
    <div className="flex flex-col gap-4">
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            rules={{ required: true }}
            name="status"
            label="Status"
            placeholder="Status"
            description="An inactive zone stops matching, and every lane written against it stops with it"
            options={statusChoices}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            rules={{ required: true }}
            name="kind"
            label="Kind"
            placeholder="Kind"
            description="What sort of area this is, which is how somebody else reads it later"
            options={rateZoneKindChoices}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="code"
            label="Code"
            placeholder="SE"
            description="The short name lanes and tariffs refer to"
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            placeholder="Southeast"
            description="What this area is called out loud"
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label="Description"
            placeholder="Atlantic and Gulf states from Virginia through Louisiana"
            description="What the zone actually covers, for the next person deciding whether to reuse it"
          />
        </FormControl>
      </FormGroup>

      <div>
        <h4 className="mb-2 text-sm font-medium">Places</h4>
        <ZoneMemberEditor />
      </div>
    </div>
  );
}
