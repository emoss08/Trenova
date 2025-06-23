import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  CustomerAutocompleteField,
  EquipmentTypeAutocompleteField,
  LocationAutocompleteField,
  ServiceTypeAutocompleteField,
  ShipmentTypeAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { statusChoices } from "@/lib/choices";
import type { DedicatedLaneSchema } from "@/lib/schemas/dedicated-lane-schema";
import { EquipmentClass } from "@/types/equipment-type";
import { useFormContext } from "react-hook-form";

export function DedicatedLaneForm() {
  const { control } = useFormContext<DedicatedLaneSchema>();

  return (
    <>
      <FormGroup cols={2}>
        <FormControl cols="full">
          <SelectField
            control={control}
            options={statusChoices}
            rules={{ required: true }}
            name="status"
            label="Status"
            placeholder="Select Status"
            description="Current operational status of the lane"
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Lane Name"
            placeholder="Enter Lane Name"
            description="Unique identifier for the dedicated lane"
          />
        </FormControl>
        <FormControl cols="full">
          <CustomerAutocompleteField<DedicatedLaneSchema>
            name="customerId"
            control={control}
            label="Customer"
            rules={{ required: true }}
            placeholder="Select Customer"
            description="Customer account associated with this lane"
          />
        </FormControl>
      </FormGroup>
      <ShipmentInformationSection />
      <WorkerAssignmentSection />
    </>
  );
}
function ShipmentInformationSection() {
  const { control } = useFormContext<DedicatedLaneSchema>();

  return (
    <FormSection
      title="Route Details"
      description="Configure origin, destination, and equipment specifications"
      className="border-t pt-4"
    >
      <FormGroup cols={2}>
        <FormControl>
          <LocationAutocompleteField<DedicatedLaneSchema>
            name="originLocationId"
            control={control}
            label="Origin"
            placeholder="Select Origin"
            description="Starting location for the dedicated lane"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <LocationAutocompleteField<DedicatedLaneSchema>
            name="destinationLocationId"
            control={control}
            label="Destination"
            placeholder="Select Destination"
            description="Final delivery location for the dedicated lane"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <ShipmentTypeAutocompleteField<DedicatedLaneSchema>
            name="shipmentTypeId"
            control={control}
            label="Shipment Type"
            rules={{ required: true }}
            placeholder="Select Type"
            description="Classification of shipment for this lane"
          />
        </FormControl>
        <FormControl>
          <ServiceTypeAutocompleteField<DedicatedLaneSchema>
            name="serviceTypeId"
            control={control}
            label="Service Type"
            rules={{ required: true }}
            placeholder="Select Service"
            description="Required service level for this lane"
          />
        </FormControl>
        <FormControl>
          <EquipmentTypeAutocompleteField<DedicatedLaneSchema>
            name="tractorTypeId"
            control={control}
            label="Tractor Type"
            placeholder="Select Tractor"
            description="Required tractor specification"
            extraSearchParams={{
              classes: [EquipmentClass.Tractor],
            }}
          />
        </FormControl>
        <FormControl>
          <EquipmentTypeAutocompleteField<DedicatedLaneSchema>
            name="trailerTypeId"
            control={control}
            label="Trailer Type"
            placeholder="Select Trailer"
            description="Required trailer or container specification"
            extraSearchParams={{
              classes: [EquipmentClass.Trailer, EquipmentClass.Container],
            }}
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

function WorkerAssignmentSection() {
  const { control } = useFormContext<DedicatedLaneSchema>();

  return (
    <FormSection
      title="Worker Assignment"
      description="Designate primary and secondary workers for this lane"
      className="border-t pt-4"
    >
      <FormGroup cols={2}>
        <FormControl>
          <WorkerAutocompleteField<DedicatedLaneSchema>
            name="primaryWorkerId"
            control={control}
            label="Primary Worker"
            rules={{ required: true }}
            placeholder="Select Primary Worker"
            description="Main worker assigned to this lane"
          />
        </FormControl>
        <FormControl>
          <WorkerAutocompleteField<DedicatedLaneSchema>
            name="secondaryWorkerId"
            control={control}
            clearable
            label="Secondary Worker"
            placeholder="Select Secondary Worker"
            description="Backup worker for this lane"
          />
        </FormControl>
        <FormControl cols="full">
          <SwitchField
            control={control}
            name="autoAssign"
            label="Automatic Worker Assignment"
            description="Automatically assign designated workers when creating shipments for this lane"
            outlined
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}
