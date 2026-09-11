import { useT } from "@trenova/shared/i18n/use-t";
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
  const t = useT();

  return (
    <FormSection
      title={t("Service & Classification")}
      description={t("Shipment type, service level, and equipment requirements")}
    >
      {children}
    </FormSection>
  );
}

function ShipmentServiceDetailsForm() {
  const t = useT();

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
          label={t("Service Type")}
          placeholder={t("Select Service Type")}
          description={t("Select the service type for the shipment.")}
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
          label={t("Shipment Type")}
          placeholder={t("Select Shipment Type")}
          description={t("Select the shipment type for the shipment.")}
        />
      ),
    },
    {
      name: "tractorTypeId",
      render: ({ required }) => (
        <EquipmentTypeAutocompleteField
          control={control}
          name="tractorTypeId"
          label={t("Tractor Type")}
          placeholder={t("Select Tractor Type")}
          description={t("Select the type of tractor used, considering any special requirements (e.g., refrigeration).")}
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
          label={t("Trailer Type")}
          placeholder={t("Select Trailer Type")}
          description={t("Select the type of trailer used, considering any special requirements (e.g., refrigeration).")}
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
