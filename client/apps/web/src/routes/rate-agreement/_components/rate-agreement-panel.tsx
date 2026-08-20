import { TabbedFormCreatePanel } from "@/components/tabbed-form-create-panel";
import { TabbedFormEditPanel } from "@/components/tabbed-form-edit-panel";
import { useEditRecordReset } from "@/hooks/use-edit-record-reset";
import { usePermission } from "@/hooks/use-permission";
import { apiService } from "@/services/api";
import type { RateAgreementReviewAction } from "@/services/rate";
import { zodResolver } from "@hookform/resolvers/zod";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { rateAgreementSchema, type RateAgreement } from "@trenova/shared/types/rate";
import {
  ArchiveIcon,
  CheckIcon,
  ClockIcon,
  FileTextIcon,
  FlaskConicalIcon,
  FuelIcon,
  MapIcon,
  PauseIcon,
  PlayIcon,
  ReceiptTextIcon,
  SendIcon,
  XIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { AccessorialScheduleEditor } from "./accessorial-schedule-editor";
import { FuelBindingForm } from "./fuel-binding-form";
import { LaneEditor } from "./lane-editor";
import { RateAgreementForm } from "./rate-agreement-form";
import { ReviewActionDialog } from "./review-action-dialog";
import { SimulationPanel } from "./simulation-panel";
import { VersionsTab } from "./versions-tab";

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

type ReviewHeaderActionsProps = {
  readonly agreement: RateAgreement;
  readonly onReviewAction: (action: RateAgreementReviewAction) => void;
};

/**
 * The review lifecycle, offered one legal step at a time.
 *
 * The status field itself is read-only in the form: an agreement moves between
 * states only through these actions, so an editor cannot activate a contract
 * by resending it.
 */
function ReviewHeaderActions({ agreement, onReviewAction }: ReviewHeaderActionsProps) {
  const { allowed: canSubmit } = usePermission(Resource.RateAgreement, Operation.Submit);
  const { allowed: canApprove } = usePermission(Resource.RateAgreement, Operation.Approve);
  const { allowed: canReject } = usePermission(Resource.RateAgreement, Operation.Reject);
  const { allowed: canUpdate } = usePermission(Resource.RateAgreement, Operation.Update);
  const { allowed: canArchive } = usePermission(Resource.RateAgreement, Operation.Archive);

  return (
    <div className="flex items-center gap-1">
      {agreement.status === "Draft" && canSubmit && (
        <Button
          type="button"
          variant="outline"
          size="xs"
          className="mr-1 gap-1.5"
          onClick={() => onReviewAction("submit")}
        >
          <SendIcon className="size-3" />
          Submit for Review
        </Button>
      )}
      {agreement.status === "InReview" && (
        <div className="mr-1 flex items-center gap-1">
          {canApprove && (
            <Button
              type="button"
              variant="outline"
              size="xs"
              className="gap-1.5 text-emerald-600 dark:text-emerald-400"
              onClick={() => onReviewAction("approve")}
            >
              <CheckIcon className="size-3" />
              Approve
            </Button>
          )}
          {canReject && (
            <Button
              type="button"
              variant="outline"
              size="xs"
              className="text-destructive gap-1.5"
              onClick={() => onReviewAction("reject")}
            >
              <XIcon className="size-3" />
              Reject
            </Button>
          )}
        </div>
      )}
      {agreement.status === "Active" && canUpdate && (
        <Button
          type="button"
          variant="outline"
          size="xs"
          className="mr-1 gap-1.5"
          onClick={() => onReviewAction("suspend")}
        >
          <PauseIcon className="size-3" />
          Suspend
        </Button>
      )}
      {agreement.status === "Suspended" && canUpdate && (
        <Button
          type="button"
          variant="outline"
          size="xs"
          className="mr-1 gap-1.5"
          onClick={() => onReviewAction("resume")}
        >
          <PlayIcon className="size-3" />
          Resume
        </Button>
      )}
      {(agreement.status === "Suspended" || agreement.status === "Expired") && canArchive && (
        <Button
          type="button"
          variant="outline"
          size="xs"
          className="mr-1 gap-1.5"
          onClick={() => onReviewAction("archive")}
        >
          <ArchiveIcon className="size-3" />
          Archive
        </Button>
      )}
      {agreement.currentVersionNumber ? (
        <Badge variant="outline" className="mr-1 font-mono text-xs">
          v{agreement.currentVersionNumber}
        </Badge>
      ) : null}
    </div>
  );
}

export function RateAgreementPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<RateAgreement>) {
  const [reviewAction, setReviewAction] = useState<RateAgreementReviewAction | null>(null);

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
        value: "simulation",
        label: "Simulation",
        icon: FlaskConicalIcon,
        content: <SimulationPanel rateAgreementId={row?.id} />,
      },
      {
        value: "versions",
        label: "Versions",
        icon: ClockIcon,
        content: <VersionsTab rateAgreementId={row?.id} />,
      },
    ],
    [row?.id],
  );

  if (mode === "edit") {
    return (
      <>
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
          headerActions={
            row ? <ReviewHeaderActions agreement={row} onReviewAction={setReviewAction} /> : null
          }
          mutationFn={(values, currentRow) => {
            if (!currentRow.id) {
              throw new Error("No Rate Agreement ID selected");
            }

            return apiService.rateAgreementService.update(currentRow.id, values);
          }}
        />

        {reviewAction && (
          <ReviewActionDialog
            open={reviewAction !== null}
            onOpenChange={(dialogOpen) => {
              if (!dialogOpen) setReviewAction(null);
            }}
            action={reviewAction}
            agreement={row ?? null}
          />
        )}
      </>
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
