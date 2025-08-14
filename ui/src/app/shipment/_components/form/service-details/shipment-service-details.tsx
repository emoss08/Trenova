/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import {
  EquipmentTypeAutocompleteField,
  ServiceTypeAutocompleteField,
  ShipmentTypeAutocompleteField,
} from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { EquipmentClass } from "@/types/equipment-type";
import { useFormContext } from "react-hook-form";

export default function ShipmentServiceDetails() {
  return (
    <ShipmentServiceDetailsInner>
      <ShipmentServiceDetailsForm />
    </ShipmentServiceDetailsInner>
  );
}

function ShipmentServiceDetailsInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-2">
      <h3 className="text-sm font-medium font-table">Service Information</h3>
      {children}
    </div>
  );
}

function ShipmentServiceDetailsForm() {
  const { control } = useFormContext<ShipmentSchema>();
  return (
    <FormGroup cols={2}>
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
          clearable
        />
      </FormControl>
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
          clearable
        />
      </FormControl>
    </FormGroup>
  );
}
