import { TabbedFormCreatePanel } from "@/components/tabbed-form-create-panel";
import { TabbedFormEditPanel } from "@/components/tabbed-form-edit-panel";
import { useEditRecordReset } from "@/hooks/use-edit-record-reset";
import { apiService } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { rateAgreementSchema, type RateAgreement } from "@trenova/shared/types/rate";
import {
  FileTextIcon,
  FlaskConicalIcon,
  FuelIcon,
  MapIcon,
  ReceiptTextIcon,
  UploadIcon,
} from "lucide-react";
import { useMemo } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { AccessorialScheduleEditor } from "./accessorial-schedule-editor";
import { FuelBindingForm } from "./fuel-binding-form";
import { LaneEditor } from "./lane-editor";
import { RateAgreementForm } from "./rate-agreement-form";
import { ImportPanel } from "./import-panel";
import { SimulationPanel } from "./simulation-panel";

const DEFAULT_AGREEMENT: Partial<RateAgreement> = {
  partyType: "Customer",
  customerId: null,
  carrierId: null,
  code: "",
  name: "",
  description: "",
  agreementType: "Contract",
  // A new agreement always starts in Draft, whatever this form says: creating
  // one already active would route around the review the organization asked for.
  status: "Draft",
  contractRef: "",
  priority: 0,
  effectiveFrom: Math.floor(Date.now() / 1000),
  effectiveTo: null,
  autoRenew: false,
  renewalNoticeDays: 30,
  currency: "USD",
  defaultMinCharge: null,
  defaultMaxCharge: null,
  roundingMode: "HalfUp",
  roundingPrecision: 2,
  billToCustomerId: null,
  marginFloorPercent: null,
  maxPayPercentOfSell: null,
  reviewComment: "",
  currentVersionNumber: 1,
  rules: [],
  accessorials: [],
  fuelBinding: null,
  versions: [],
};

export function RateAgreementPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<RateAgreement>) {
  const form = useForm<RateAgreement>({
    resolver: zodResolver(rateAgreementSchema) as Resolver<RateAgreement>,
    defaultValues: DEFAULT_AGREEMENT as RateAgreement,
    mode: "onChange",
  });

  // The table row carries only the header; without the full record every child
  // editor opens empty, and saving that emptiness would erase the contract's
  // lanes, accessorials and fuel terms.
  useEditRecordReset(form, {
    open,
    mode,
    queryKey: "rate-agreement",
    id: row?.id,
    version: row?.version,
    fetch: (id) => apiService.rateAgreementService.getById(id),
  });

  const formTabs = useMemo(
    () => [
      {
        value: "overview",
        label: "Overview",
        icon: FileTextIcon,
        content: <RateAgreementForm />,
      },
      {
        value: "lanes",
        label: "Lanes",
        icon: MapIcon,
        content: <LaneEditor />,
      },
      {
        value: "accessorials",
        label: "Accessorials",
        icon: ReceiptTextIcon,
        content: <AccessorialScheduleEditor />,
      },
      {
        value: "fuel",
        label: "Fuel",
        icon: FuelIcon,
        content: <FuelBindingForm />,
      },
      {
        value: "import",
        label: "Import",
        icon: UploadIcon,
        content: <ImportPanel rateAgreementId={row?.id} />,
      },
      {
        value: "simulation",
        label: "Simulation",
        icon: FlaskConicalIcon,
        content: <SimulationPanel rateAgreementId={row?.id} />,
      },
    ],
    [row?.id],
  );

  if (mode === "edit") {
    return (
      <TabbedFormEditPanel<RateAgreement, RateAgreement>
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        size="xl"
        queryKey="rate-agreement-list"
        title="Rate Agreement"
        fieldKey="name"
        formTabs={formTabs}
        mutationFn={(values, currentRow) => {
          if (!currentRow.id) {
            throw new Error("No Rate Agreement ID selected");
          }

          return apiService.rateAgreementService.update(currentRow.id, values);
        }}
      />
    );
  }

  return (
    <TabbedFormCreatePanel<RateAgreement, RateAgreement>
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      size="xl"
      queryKey="rate-agreement-list"
      title="Rate Agreement"
      description="Write the contract once, and every shipment on its lanes prices itself against it."
      formTabs={formTabs}
      mutationFn={(values) => apiService.rateAgreementService.create(values)}
    />
  );
}
