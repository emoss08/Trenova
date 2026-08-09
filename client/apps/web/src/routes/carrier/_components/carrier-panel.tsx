import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { carrierSchema, type Carrier } from "@trenova/shared/types/carrier";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { CarrierTabbedForm } from "./carrier-tabbed-form";

const DEFAULT_VALUES: Carrier = {
  status: "Active",
  code: "",
  name: "",
  dbaName: null,
  carrierType: "Common",
  dotNumber: null,
  mcNumber: null,
  scac: null,
  complianceStatus: "Pending",
  safetyRating: "NotRated",
  qualifiedAt: null,
  disqualifiedReason: null,
  taxId: null,
  taxIdType: null,
  w9OnFile: false,
  is1099Eligible: false,
  paymentMethod: "Check",
  paymentTermDays: 30,
  remitToName: null,
  remitAddressLine1: null,
  remitAddressLine2: null,
  remitCity: null,
  remitStateId: null,
  remitPostalCode: null,
  addressLine1: null,
  addressLine2: null,
  city: null,
  stateId: null,
  postalCode: null,
  phone: null,
  email: null,
  externalId: null,
  notes: null,
  contacts: [],
  insurancePolicies: [],
};

export function CarrierPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<Carrier>) {
  const form = useForm({
    resolver: zodResolver(carrierSchema),
    defaultValues: DEFAULT_VALUES,
    mode: "onChange",
  });

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/carriers/"
        queryKey="carrier-list"
        title="Carrier"
        fieldKey="name"
        size="lg"
        formComponent={<CarrierTabbedForm />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/carriers/"
      queryKey="carrier-list"
      title="Carrier"
      size="lg"
      formComponent={<CarrierTabbedForm />}
    />
  );
}
