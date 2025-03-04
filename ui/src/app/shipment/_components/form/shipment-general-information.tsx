import { InputField } from "@/components/fields/input-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { useFormContext } from "react-hook-form";

export default function ShipmentGeneralInformation() {
  const { control } = useFormContext();

  return (
    <div className="flex flex-col gap-2">
      <h3 className="text-sm font-medium">General Information</h3>
      <FormGroup cols={2}>
        <FormControl cols="full">
          <InputField
            control={control}
            name="bol"
            label="BOL"
            rules={{ required: true }}
            description="The BOL is the bill of lading number for the shipment."
            placeholder="Enter BOL"
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="temperatureMin"
            label="Temperature Min"
            type="number"
            description="The minimum temperature for the shipment."
            placeholder="Enter Temperature Min"
            sideText="°F"
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="temperatureMax"
            label="Temperature Max"
            type="number"
            description="The maximum temperature for the shipment."
            placeholder="Enter Temperature Max"
            sideText="°F"
          />
        </FormControl>
      </FormGroup>
    </div>
  );
}
