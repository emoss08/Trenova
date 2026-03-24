import {
  EquipmentTypeAutocompleteField,
  ServiceTypeAutocompleteField,
  ShipmentTypeAutocompleteField,
} from "@/components/autocomplete-fields";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { equipmentClassSchema } from "@/types/equipment-type";
import type { Shipment } from "@/types/shipment";
import { useFormContext } from "react-hook-form";

export default function ShipmentServiceDetails() {
  return (
    <ShipmentServiceDetailsInner>
      <ShipmentServiceDetailsForm />
    </ShipmentServiceDetailsInner>
  );
}

function ShipmentServiceDetailsInner({ children }: { children: React.ReactNode }) {
  return (
    <FormSection
      title="Service & Classification"
      description="Shipment type, service level, and equipment requirements"
    >
      {children}
    </FormSection>
  );
}

function ShipmentServiceDetailsForm() {
  const { control } = useFormContext<Shipment>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <ServiceTypeAutocompleteField
          control={control}
          name="serviceTypeId"
          rules={{ required: true }}
          label="Service Type"
          placeholder="Select Service Type"
          description="Select the service type for the shipment."
        />
      </FormControl>
      <FormControl>
        <ShipmentTypeAutocompleteField
          control={control}
          name="shipmentTypeId"
          rules={{ required: true }}
          label="Shipment Type"
          placeholder="Select Shipment Type"
          description="Select the shipment type for the shipment."
        />
      </FormControl>
      <FormControl>
        <EquipmentTypeAutocompleteField
          control={control}
          name="tractorTypeId"
          label="Tractor Type"
          placeholder="Select Tractor Type"
          description="Select the type of tractor used, considering any special requirements (e.g., refrigeration)."
          extraSearchParams={{
            classes: [equipmentClassSchema.enum.Tractor],
          }}
          clearable
        />
      </FormControl>
      <FormControl>
        <EquipmentTypeAutocompleteField
          control={control}
          name="trailerTypeId"
          label="Trailer Type"
          placeholder="Select Trailer Type"
          description="Select the type of trailer used, considering any special requirements (e.g., refrigeration)."
          extraSearchParams={{
            classes: [equipmentClassSchema.enum.Trailer, equipmentClassSchema.enum.Container],
          }}
          clearable
        />
      </FormControl>
    </FormGroup>
  );
}
