import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import {
  CapabilityFields,
  type FieldDescriptor,
} from "@trenova/shared/components/capability-form-section";
import { FormSection } from "@trenova/shared/components/ui/form";
import { CAPABILITIES, getProfile } from "@trenova/shared/lib/capability";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { useFormContext, useWatch } from "react-hook-form";

function Inner({ children }: { children: React.ReactNode }) {
  return (
    <FormSection
      title="General Information"
      description="Basic information about the shipment"
      className="border-t border-border pt-4"
    >
      {children}
    </FormSection>
  );
}

export default function ShipmentGeneralInformation() {
  const { control } = useFormContext<Shipment>();
  const { data: shipmentUIPolicy } = useQuery({ ...queries.shipment.uiPolicy() });

  const profile = getProfile(shipmentUIPolicy);

  const descriptors: FieldDescriptor[] = [
    {
      name: "bol",
      cols: "full",
      // Deliberately ignores the resolved `required`. isFieldRequired reports
      // true whenever any *Block* rule names the field, and documentation.duplicateBol
      // names `bol` at Block by default — but that rule means "do not reuse a BOL",
      // not "a BOL is mandatory". Wiring it through would mark BOL required on
      // every profile that checks for duplicates. BOLField resolves its own
      // required state from the customer's billing profile, which is the only
      // thing that actually decides it.
      render: () => <BOLField />,
    },
    {
      name: "temperatureMin",
      capability: CAPABILITIES.temperatureControl,
      render: ({ required }) => (
        <NumberField
          control={control}
          name="temperatureMin"
          description="The minimum temperature for the shipment."
          label="Temperature Min"
          placeholder="Enter Temperature Min"
          sideText="°F"
          rules={{ required }}
        />
      ),
    },
    {
      name: "temperatureMax",
      capability: CAPABILITIES.temperatureControl,
      render: ({ required }) => (
        <NumberField
          control={control}
          name="temperatureMax"
          label="Temperature Max"
          description="The maximum temperature for the shipment."
          placeholder="Enter Temperature Max"
          sideText="°F"
          rules={{ required }}
        />
      ),
    },
  ];

  return (
    <Inner>
      <CapabilityFields descriptors={descriptors} profile={profile} />
    </Inner>
  );
}

export function BOLField() {
  const { control, setError, clearErrors, getFieldState } = useFormContext<Shipment>();
  const shipmentId = useWatch({ control, name: "id" });

  const customerId = useWatch({ control, name: "customerId" });
  const { data: billingProfile } = useQuery({
    ...queries.customer.getBillingProfile(customerId),
    enabled: !!customerId,
  });
  const { data: shipmentUIPolicy } = useQuery({
    ...queries.shipment.uiPolicy(),
  });

  const bol = useWatch({ control, name: "bol" });
  const bolCheckTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (bolCheckTimer.current != null) clearTimeout(bolCheckTimer.current);
    if (shipmentUIPolicy?.checkForDuplicateBols === false) {
      if (getFieldState("bol").error?.type === "manual") {
        clearErrors("bol");
      }
      return;
    }
    if (!bol || bol.length < 2) return;

    bolCheckTimer.current = setTimeout(async () => {
      try {
        await apiService.shipmentService.checkForDuplicateBOLs(bol, shipmentId);
        clearErrors("bol");
      } catch (err) {
        if (err instanceof ApiRequestError && err.data.errors?.length) {
          const bolError = err.data.errors.find((e) => e.field === "bol");
          if (bolError) {
            setError("bol", { type: "manual", message: bolError.message });
            return;
          }
        }
      }
    }, 500);

    return () => {
      if (bolCheckTimer.current != null) clearTimeout(bolCheckTimer.current);
    };
  }, [
    bol,
    shipmentId,
    shipmentUIPolicy?.checkForDuplicateBols,
    setError,
    clearErrors,
    getFieldState,
  ]);

  const bolRequired = billingProfile?.enforceCustomerBillingReq && billingProfile?.requireBOLNumber;

  // No FormControl wrapper: CapabilityFields supplies one per descriptor, and
  // nesting a second would offset the field from every sibling in the group.
  return (
    <InputField
      control={control}
      name="bol"
      label="BOL"
      rules={{ required: bolRequired }}
      description="The BOL is the bill of lading number for the shipment."
      placeholder="Enter BOL"
      maxLength={100}
    />
  );
}
