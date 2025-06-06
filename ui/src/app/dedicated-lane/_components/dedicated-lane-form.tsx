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
            placeholder="Status"
            description="Indicates the current operational status of the dedicated lane."
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            placeholder="Name"
            description="A name identifying the dedicated lane."
          />
        </FormControl>
        <FormControl cols="full">
          <CustomerAutocompleteField<DedicatedLaneSchema>
            name="customerId"
            control={control}
            label="Customer"
            rules={{ required: true }}
            placeholder="Customer"
            description="The customer associated with the dedicated lane."
          />
        </FormControl>
        <FormControl>
          <LocationAutocompleteField<DedicatedLaneSchema>
            name="originLocationId"
            control={control}
            label="Origin Location"
            placeholder="Origin Location"
            description="The origin location associated with the dedicated lane."
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <LocationAutocompleteField<DedicatedLaneSchema>
            name="destinationLocationId"
            control={control}
            label="Destination Location"
            placeholder="Destination Location"
            description="The destination location associated with the dedicated lane."
            rules={{ required: true }}
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
      title="Shipment Information"
      description="The shipment information associated with the dedicated lane."
      className="border-t pt-4"
    >
      <FormGroup cols={2}>
        <FormControl>
          <ShipmentTypeAutocompleteField<DedicatedLaneSchema>
            name="shipmentTypeId"
            control={control}
            label="Shipment Type"
            rules={{ required: true }}
            placeholder="Shipment Type"
            description="The shipment type associated with the dedicated lane."
          />
        </FormControl>
        <FormControl>
          <ServiceTypeAutocompleteField<DedicatedLaneSchema>
            name="serviceTypeId"
            control={control}
            label="Service Type"
            rules={{ required: true }}
            placeholder="Service Type"
            description="The service type associated with the dedicated lane."
          />
        </FormControl>
        <FormControl>
          <EquipmentTypeAutocompleteField<DedicatedLaneSchema>
            name="tractorTypeId"
            control={control}
            label="Tractor Type"
            placeholder="Tractor Type"
            description="The tractor type associated with the dedicated lane."
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
            placeholder="Trailer Type"
            description="The trailer type associated with the dedicated lane."
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
    <FormSection title="Worker Assignment" className="border-t pt-4">
      <FormGroup cols={2}>
        <FormControl>
          <WorkerAutocompleteField<DedicatedLaneSchema>
            name="primaryWorkerId"
            control={control}
            label="Primary Worker"
            rules={{ required: true }}
            placeholder="Select Primary Worker"
            description="Select the primary worker for the assignment."
          />
        </FormControl>
        <FormControl>
          <WorkerAutocompleteField<DedicatedLaneSchema>
            name="secondaryWorkerId"
            control={control}
            clearable
            label="Secondary Worker"
            placeholder="Select Secondary Worker"
            description="Select the secondary worker for the assignment."
          />
        </FormControl>
        <FormControl cols="full">
          <SwitchField
            control={control}
            name="autoAssign"
            label="Auto Assign Workers"
            description="Auto assign the workers to the dedicated lane when a shipment is created."
            outlined
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}
