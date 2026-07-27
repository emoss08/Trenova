import { TabbedFormCreatePanel } from "@/components/tabbed-form-create-panel";
import { TabbedFormEditPanel } from "@/components/tabbed-form-edit-panel";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  detentionPolicySchema,
  type DetentionPolicy,
} from "@trenova/shared/types/detention";
import { zodResolver } from "@hookform/resolvers/zod";
import { FlaskConicalIcon, HistoryIcon, ScaleIcon } from "lucide-react";
import { useMemo } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { DetentionBacktest } from "./detention-backtest";
import { DetentionPolicyForm } from "./detention-policy-form";
import { DetentionPolicyPreview } from "./detention-policy-preview";

const DEFAULT_POLICY: Partial<DetentionPolicy> = {
  name: "",
  code: "",
  description: "",
  status: "Draft",
  isOrgDefault: false,
  priority: 0,
  specificityScore: 0,
  customerId: null,
  locationId: null,
  shipmentTypeIds: null,
  serviceTypeIds: null,
  commodityIds: null,
  stopTypes: null,
  appointmentStopsOnly: false,
  effectiveStartDate: null,
  effectiveEndDate: null,
  clockStartBasis: "LaterOfArrivalOrAppointment",
  lateArrivalRule: "NoEffect",
  lateArrivalGraceMinutes: 0,
  billingFreeMinutes: 120,
  pickupFreeMinutes: null,
  deliveryFreeMinutes: null,
  payFreeMinutes: null,
  minimumBillableMinutes: 0,
  billingIncrementMinutes: 15,
  roundingMode: "Up",
  rateSource: "Accessorial",
  accessorialChargeId: "",
  tiers: [],
  maxBillableMinutesPerStop: null,
  maxChargePerStop: null,
  maxChargePerDay: null,
  maxChargePerShipment: null,
  dayBoundaryMode: "PerStop",
  convertToLayoverAtMinutes: null,
  layoverAccessorialChargeId: null,
  notificationRequirement: "None",
  notificationLeadMinutes: 30,
  notificationDeadlineMinutes: 0,
  unnotifiedBehavior: "Bill",
  autoSendNotice: false,
  sendDepartureSummary: false,
  requireApprovalOverAmount: null,
  autoApproveUnderAmount: null,
  currency: "USD",
  comments: "",
};

export function DetentionPolicyPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<DetentionPolicy>) {
  const form = useForm<DetentionPolicy>({
    resolver: zodResolver(detentionPolicySchema) as Resolver<DetentionPolicy>,
    defaultValues: DEFAULT_POLICY as DetentionPolicy,
    mode: "onChange",
  });

  const formTabs = useMemo(
    () => [
      {
        value: "terms",
        label: "Terms",
        icon: ScaleIcon,
        content: <DetentionPolicyForm />,
      },
      {
        value: "preview",
        label: "Live Preview",
        icon: FlaskConicalIcon,
        content: <DetentionPolicyPreview />,
      },
      {
        value: "backtest",
        label: "Backtest",
        icon: HistoryIcon,
        content: <DetentionBacktest />,
      },
    ],
    [],
  );

  if (mode === "edit") {
    return (
      <TabbedFormEditPanel<DetentionPolicy, DetentionPolicy>
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        queryKey="detention-policy-list"
        title="Detention Policy"
        fieldKey="name"
        formTabs={formTabs}
        size="xl"
        mutationFn={(values, currentRow) => {
          if (!currentRow.id) {
            throw new Error("No Detention Policy ID selected");
          }

          return apiService.detentionPolicyService.update(currentRow.id, values);
        }}
      />
    );
  }

  return (
    <TabbedFormCreatePanel<DetentionPolicy, DetentionPolicy>
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey="detention-policy-list"
      title="Detention Policy"
      description="Encode the contract's detention terms and see what they would charge before they touch a shipment."
      formTabs={formTabs}
      size="xl"
      mutationFn={(values) => apiService.detentionPolicyService.create(values)}
    />
  );
}
