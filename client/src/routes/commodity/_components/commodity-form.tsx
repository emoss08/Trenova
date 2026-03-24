import { HazardousMaterialAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { freightClassChoices, statusChoices } from "@/lib/choices";
import type { Commodity } from "@/types/commodity";
import { useFormContext } from "react-hook-form";

export function CommodityForm() {
  const { control } = useFormContext<Commodity>();

  return (
    <div className="space-y-6">
      <FormSection
        title="General Information"
        description="Basic identification for this commodity."
        className="border-b pb-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="status"
              label="Status"
              placeholder="Status"
              description="The current status of the commodity."
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
              description="The name of the commodity."
              maxLength={100}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              rules={{ required: true }}
              name="description"
              label="Description"
              placeholder="Description"
              description="A detailed description of the commodity."
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title="Classification"
        description="Freight classification and hazardous material linkage."
        className="border-b pb-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="freightClass"
              label="Freight Class"
              placeholder="Select freight class"
              description="The NMFC freight classification for this commodity."
              options={freightClassChoices}
            />
          </FormControl>
          <FormControl>
            <HazardousMaterialAutocompleteField
              control={control}
              name="hazardousMaterialId"
              label="Hazardous Material"
              placeholder="Search hazardous materials..."
              description="Link a hazardous material to this commodity if applicable."
              clearable
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title="Temperature"
        description="Temperature range requirements for shipping."
        className="border-b pb-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="minTemperature"
              label="Min Temperature"
              sideText="&deg;F"
              placeholder="Min Temperature"
              description="Minimum temperature for storing or shipping."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxTemperature"
              label="Max Temperature"
              sideText="&deg;F"
              placeholder="Max Temperature"
              description="Maximum temperature for storing or shipping."
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title="Physical Properties"
        description="Weight, dimensions, and quantity constraints."
        className="border-b pb-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="weightPerUnit"
              label="Weight Per Unit"
              sideText="lbs"
              placeholder="Weight Per Unit"
              description="The weight of a single unit of this commodity."
              step={0.01}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="linearFeetPerUnit"
              label="Linear Feet Per Unit"
              sideText="ft"
              placeholder="Linear Feet Per Unit"
              description="The linear feet occupied by a single unit."
              step={0.01}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxQuantityPerShipment"
              label="Max Qty Per Shipment"
              placeholder="Max Quantity Per Shipment"
              description="Maximum quantity allowed per shipment."
              step={0.01}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title="Handling"
        description="Loading instructions and handling requirements."
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="loadingInstructions"
              label="Loading Instructions"
              placeholder="Loading Instructions"
              description="Specific instructions for loading this commodity."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="stackable"
              label="Stackable"
              description="Whether this commodity can be stacked during transport."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="fragile"
              label="Fragile"
              description="Whether this commodity requires fragile handling."
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
