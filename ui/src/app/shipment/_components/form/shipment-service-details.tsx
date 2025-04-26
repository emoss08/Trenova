import {
  EquipmentTypeAutocompleteField,
  ServiceTypeAutocompleteField,
  ShipmentTypeAutocompleteField,
} from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { EquipmentClass } from "@/types/equipment-type";
import { memo } from "react";
import { useFormContext } from "react-hook-form";

const ShipmentServiceDetailsComponent = () => {
  const { control } = useFormContext<ShipmentSchema>();

  return (
    <div className="flex flex-col gap-2">
      <h3 className="text-sm font-medium">Service Information</h3>
      <FormGroup cols={2}>
        <ShipmentTypeField control={control} />
        <ServiceTypeField control={control} />
        <TractorTypeField control={control} />
        <TrailerTypeField control={control} />
      </FormGroup>
    </div>
  );
};

ShipmentServiceDetailsComponent.displayName = "ShipmentServiceDetails";
export default memo(ShipmentServiceDetailsComponent);

// Individual field components
const ShipmentTypeField = memo(({ control }: { control: any }) => (
  <FormControl>
    <ShipmentTypeAutocompleteField<ShipmentSchema>
      name="shipmentTypeId"
      control={control}
      label="Shipment Type"
      rules={{ required: true }}
      placeholder="Select Shipment Type"
      description="Select the shipment type for the shipment."
    />
  </FormControl>
));

ShipmentTypeField.displayName = "ShipmentTypeField";

const ServiceTypeField = memo(({ control }: { control: any }) => (
  <FormControl>
    <ServiceTypeAutocompleteField<ShipmentSchema>
      name="serviceTypeId"
      control={control}
      label="Service Type"
      rules={{ required: true }}
      placeholder="Select Service Type"
      description="Select the service type for the shipment."
    />
  </FormControl>
));

ServiceTypeField.displayName = "ServiceTypeField";

const TractorTypeField = memo(({ control }: { control: any }) => (
  <FormControl>
    <EquipmentTypeAutocompleteField<ShipmentSchema>
      name="tractorTypeId"
      control={control}
      label="Tractor Type"
      placeholder="Select Tractor Type"
      description="Select the type of tractor used, considering any special requirements (e.g., refrigeration)."
      extraSearchParams={{
        classes: [EquipmentClass.Tractor],
      }}
    />
  </FormControl>
));

TractorTypeField.displayName = "TractorTypeField";

const TrailerTypeField = memo(({ control }: { control: any }) => (
  <FormControl>
    <EquipmentTypeAutocompleteField<ShipmentSchema>
      name="trailerTypeId"
      control={control}
      label="Trailer Type"
      placeholder="Select Trailer Type"
      description="Select the type of trailer used, considering any special requirements (e.g., refrigeration)."
      extraSearchParams={{
        classes: [EquipmentClass.Trailer, EquipmentClass.Container],
      }}
    />
  </FormControl>
));

TrailerTypeField.displayName = "TrailerTypeField";
