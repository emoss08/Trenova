import {
  EquipmentTypeAutocompleteField,
  ServiceTypeAutocompleteField,
  ShipmentTypeAutocompleteField,
} from "@/components/autocomplete-fields";
import { queries } from "@/lib/queries";
import {
  CapabilityFields,
  type FieldDescriptor,
} from "@trenova/shared/components/capability-form-section";
import { FormSection } from "@trenova/shared/components/ui/form";
import { getProfile } from "@trenova/shared/lib/capability";
import { equipmentClassSchema } from "@/types/equipment-type";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
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
  const { data: shipmentUIPolicy } = useQuery({ ...queries.shipment.uiPolicy() });

  const profile = getProfile(shipmentUIPolicy);

  const descriptors: FieldDescriptor[] = [
    {
      name: "serviceTypeId",
      render: () => (
        <ServiceTypeAutocompleteField
          control={control}
          name="serviceTypeId"
          rules={{ required: true }}
          label="Service Type"
          placeholder="Select Service Type"
          description="Select the service type for the shipment."
        />
      ),
    },
    {
      name: "shipmentTypeId",
      render: () => (
        <ShipmentTypeAutocompleteField
          control={control}
          name="shipmentTypeId"
          rules={{ required: true }}
          label="Shipment Type"
          placeholder="Select Shipment Type"
          description="Select the shipment type for the shipment."
        />
      ),
    },
    {
      name: "tractorTypeId",
      render: ({ required }) => (
        <EquipmentTypeAutocompleteField
          control={control}
          name="tractorTypeId"
          label="Tractor Type"
          placeholder="Select Tractor Type"
          description="Select the type of tractor used, considering any special requirements (e.g., refrigeration)."
          extraSearchParams={{
            classes: [equipmentClassSchema.enum.Tractor],
          }}
          rules={{ required }}
          clearable
        />
      ),
    },
    {
      // The deck-fit rule names trailerTypeId, so a dimensional-cargo profile
      // resolves this to required without the section restating it.
      name: "trailerTypeId",
      render: ({ required }) => (
        <EquipmentTypeAutocompleteField
          control={control}
          name="trailerTypeId"
          label="Trailer Type"
          placeholder="Select Trailer Type"
          description="Select the type of trailer used, considering any special requirements (e.g., refrigeration)."
          extraSearchParams={{
            classes: [equipmentClassSchema.enum.Trailer, equipmentClassSchema.enum.Container],
          }}
          rules={{ required }}
          clearable
        />
      ),
    },
  ];

  return <CapabilityFields descriptors={descriptors} profile={profile} />;
}
