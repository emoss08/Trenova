import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { GeocodedBadge } from "@/components/geocode-badge";
import { DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { formatToUserTimezone } from "@/lib/date";
import { useAuthStore } from "@/stores/auth-store";
import { customerSchema, type Customer } from "@/types/customer";
import type { DataTablePanelProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { CustomerTabbedForm } from "./customer-tabbed-form";

const DEFAULT_VALUES: Customer = {
  status: "Active",
  code: "",
  name: "",
  addressLine1: "",
  addressLine2: null,
  city: "",
  stateId: "",
  postalCode: "",
  isGeocoded: false,
  longitude: null,
  latitude: null,
  placeId: null,
  externalId: null,
  allowConsolidation: true,
  exclusiveConsolidation: false,
  consolidationPriority: 1,
  billingProfile: {
    billingCycleType: "Immediate",
    billingCycleDayOfWeek: null,
    paymentTerm: "Net30",
    creditStatus: "Active",
    creditLimit: null,
    creditBalance: 0,
    enforceCreditLimit: false,
    autoCreditHold: false,
    creditHoldReason: "",
    hasBillingControlOverrides: false,
    invoiceMethod: "Individual",
    summaryTransmitOnGeneration: true,
    allowInvoiceConsolidation: false,
    consolidationPeriodDays: 7,
    consolidationGroupBy: "None",
    invoiceNumberFormat: "Default",
    customerInvoicePrefix: "",
    invoiceCopies: 1,
    revenueAccountId: null,
    arAccountId: null,
    applyLateCharges: false,
    lateChargeRate: null,
    gracePeriodDays: 0,
    taxExempt: false,
    taxExemptNumber: "",
    enforceCustomerBillingReq: true,
    validateCustomerRates: true,
    autoTransfer: true,
    autoMarkReadyToBill: true,
    autoBill: true,
    detentionBillingEnabled: false,
    detentionFreeMinutes: 120,
    detentionRatePerHour: null,
    countLateOnlyOnAppointmentStops: false,
    countDetentionOnlyOnAppointmentStops: false,
    autoApplyAccessorials: true,
    billingCurrency: "USD",
    requirePONumber: false,
    requireBOLNumber: false,
    requireDeliveryNumber: false,
    billingNotes: "",
    documentTypes: [],
  },
  emailProfile: {
    subject: "",
    comment: "",
    fromEmail: "",
    toRecipients: "",
    ccRecipients: "",
    bccRecipients: "",
    attachmentName: "",
    readReceipt: false,
    sendInvoiceOnGeneration: true,
    includeShipmentDetail: false,
  },
};

export function CustomerPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<Customer>) {
  const user = useAuthStore((s) => s.user);
  const panelDescription = row?.updatedAt
    ? `Last updated on ${formatToUserTimezone(
        row.updatedAt as number,
        {
          timeFormat: user?.timeFormat || "24-hour",
        },
        user?.timezone,
      )}`
    : undefined;

  const form = useForm({
    resolver: zodResolver(customerSchema),
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
        url="/customers/"
        queryKey="customer-list"
        title="Customer"
        fieldKey="name"
        size="lg"
        formComponent={<CustomerTabbedForm />}
        titleComponent={(currentRecord) => {
          return (
            <div className="flex flex-col gap-0.5">
              <DialogTitle className="flex items-center justify-start gap-x-1">
                <span className="truncate">{currentRecord.name}</span>
                {currentRecord.isGeocoded ? (
                  <GeocodedBadge
                    longitude={currentRecord.longitude as unknown as number}
                    latitude={currentRecord.latitude as unknown as number}
                    placeId={currentRecord.placeId ?? undefined}
                  />
                ) : null}
              </DialogTitle>
              <DialogDescription>{panelDescription}</DialogDescription>
            </div>
          );
        }}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/customers/"
      queryKey="customer-list"
      title="Customer"
      size="lg"
      formComponent={<CustomerTabbedForm />}
    />
  );
}
