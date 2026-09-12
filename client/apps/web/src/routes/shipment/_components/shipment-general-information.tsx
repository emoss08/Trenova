import { useT } from "@trenova/shared/i18n/use-t";
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
  const t = useT();

  return (
    <FormSection
      title={t("General Information")}
      description={t("Basic information about the shipment")}
      className="border-border border-t pt-4"
    >
      {children}
    </FormSection>
  );
}

export default function ShipmentGeneralInformation() {
  const t = useT();

  const { control } = useFormContext<Shipment>();
  const { data: shipmentUIPolicy } = useQuery({ ...queries.shipment.uiPolicy() });

  const profile = getProfile(shipmentUIPolicy);

  const descriptors: FieldDescriptor[] = [
    {
      name: "bol",
      cols: "full",
      // Ignores the resolved `required` on purpose. The only rule that names
      // `bol` is documentation.duplicateBol, which forbids reusing a number
      // rather than demanding one — so the mode profile has nothing to say
      // about whether a BOL is mandatory, and the catalog reflects that by
      // leaving `bol` out of the rule's requiredFields. What decides is the
      // customer's billing profile, which BOLField resolves for itself.
      render: () => <BOLField />,
    },
    {
      name: "temperatureMin",
      capability: CAPABILITIES.temperatureControl,
      render: ({ required }) => (
        <NumberField
          control={control}
          name="temperatureMin"
          description={t("The minimum temperature for the shipment.")}
          label={t("Temperature Min")}
          placeholder={t("Enter Temperature Min")}
          sideText={t("°F")}
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
          label={t("Temperature Max")}
          description={t("The maximum temperature for the shipment.")}
          placeholder={t("Enter Temperature Max")}
          sideText={t("°F")}
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
  const t = useT();

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
      label={t("BOL")}
      rules={{ required: bolRequired }}
      description={t("The BOL is the bill of lading number for the shipment.")}
      placeholder={t("Enter BOL")}
      maxLength={100}
    />
  );
}
