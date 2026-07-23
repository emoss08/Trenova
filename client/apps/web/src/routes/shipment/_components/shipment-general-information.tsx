import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { ApiRequestError } from "@/lib/api";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { Shipment } from "@/types/shipment";
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

  return (
    <Inner>
      <FormGroup cols={2}>
        <BOLField />
        <FormControl>
          <NumberField
            control={control}
            name="temperatureMin"
            description="The minimum temperature for the shipment."
            label="Temperature Min"
            placeholder="Enter Temperature Min"
            sideText="°F"
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="temperatureMax"
            label="Temperature Max"
            description="The maximum temperature for the shipment."
            placeholder="Enter Temperature Max"
            sideText="°F"
          />
        </FormControl>
      </FormGroup>
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
  }, [bol, shipmentId, shipmentUIPolicy?.checkForDuplicateBols, setError, clearErrors, getFieldState]);

  const bolRequired = billingProfile?.enforceCustomerBillingReq && billingProfile?.requireBOLNumber;

  return (
    <FormControl cols="full">
      <InputField
        control={control}
        name="bol"
        label="BOL"
        rules={{ required: bolRequired }}
        description="The BOL is the bill of lading number for the shipment."
        placeholder="Enter BOL"
        maxLength={100}
      />
    </FormControl>
  );
}
