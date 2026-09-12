import { useT } from "@trenova/shared/i18n/use-t";
import { HazardousMaterialAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { freightClassChoices, statusChoices } from "@/lib/choices";
import type { Commodity } from "@trenova/shared/types/commodity";
import { useFormContext } from "react-hook-form";

export function CommodityForm() {
  const t = useT();

  const { control } = useFormContext<Commodity>();

  return (
    <div className="space-y-6">
      <FormSection
        title={t("General Information")}
        description={t("Basic identification for this commodity.")}
        className="border-b pb-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="status"
              label={t("Status")}
              placeholder={t("Status")}
              description={t("The current status of the commodity.")}
              options={statusChoices}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label={t("Name")}
              placeholder={t("Name")}
              description={t("The name of the commodity.")}
              maxLength={100}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              rules={{ required: true }}
              name="description"
              label={t("Description")}
              placeholder={t("Description")}
              description={t("A detailed description of the commodity.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title={t("Classification")}
        description={t("Freight classification and hazardous material linkage.")}
        className="border-b pb-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="freightClass"
              label={t("Freight Class")}
              placeholder={t("Select freight class")}
              description={t("The NMFC freight classification for this commodity.")}
              options={freightClassChoices}
              isClearable
            />
          </FormControl>
          <FormControl>
            <HazardousMaterialAutocompleteField
              control={control}
              name="hazardousMaterialId"
              label={t("Hazardous Material")}
              placeholder={t("Search hazardous materials...")}
              description={t("Link a hazardous material to this commodity if applicable.")}
              clearable
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title={t("Temperature")}
        description={t("Temperature range requirements for shipping.")}
        className="border-b pb-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="minTemperature"
              label={t("Min Temperature")}
              sideText={t("°F")}
              placeholder={t("Min Temperature")}
              description={t("Minimum temperature for storing or shipping.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxTemperature"
              label={t("Max Temperature")}
              sideText={t("°F")}
              placeholder={t("Max Temperature")}
              description={t("Maximum temperature for storing or shipping.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title={t("Physical Properties")}
        description={t("Weight, dimensions, and quantity constraints.")}
        className="border-b pb-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="weightPerUnit"
              label={t("Weight Per Unit")}
              sideText="lbs"
              placeholder={t("Weight Per Unit")}
              description={t("The weight of a single unit of this commodity.")}
              step={0.01}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="linearFeetPerUnit"
              label={t("Linear Feet Per Unit")}
              sideText="ft"
              placeholder={t("Linear Feet Per Unit")}
              description={t("The linear feet occupied by a single unit.")}
              step={0.01}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxQuantityPerShipment"
              label={t("Max Qty Per Shipment")}
              placeholder={t("Max Quantity Per Shipment")}
              description={t("Maximum quantity allowed per shipment.")}
              step={0.01}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title={t("Handling")}
        description={t("Loading instructions and handling requirements.")}
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="loadingInstructions"
              label={t("Loading Instructions")}
              placeholder={t("Loading Instructions")}
              description={t("Specific instructions for loading this commodity.")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="stackable"
              label={t("Stackable")}
              description={t("Whether this commodity can be stacked during transport.")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="fragile"
              label={t("Fragile")}
              description={t("Whether this commodity requires fragile handling.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
