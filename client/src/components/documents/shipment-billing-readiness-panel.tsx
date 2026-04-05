import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import type {
  Shipment,
  ShipmentBillingReadiness,
  ShipmentBillingRequirement,
} from "@/types/shipment";
import {
  AlertCircleIcon,
  CheckCircle2Icon,
  Clock3Icon,
  FileTextIcon,
  ReceiptIcon,
} from "lucide-react";

interface ShipmentBillingReadinessPanelProps {
  readiness: ShipmentBillingReadiness;
  shipment?: Shipment;
  onUploadRequired: (requirement: ShipmentBillingRequirement) => void;
  onMarkReadyToInvoice: () => void;
  isMarkingReady: boolean;
  disabled?: boolean;
}

function RequirementRow({
  requirement,
  isNext,
  onUpload,
  disabled,
}: {
  requirement: ShipmentBillingRequirement;
  isNext: boolean;
  onUpload: () => void;
  disabled?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-xl border bg-background/70 px-3 py-3">
      <div className="min-w-0 space-y-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm font-medium">{requirement.documentTypeName}</span>
          <Badge variant="outline" className="text-2xs">
            {requirement.documentTypeCode}
          </Badge>
          {requirement.satisfied ? (
            <Badge variant="teal" className="text-2xs">
              Uploaded
            </Badge>
          ) : isNext ? (
            <Badge variant="warning" className="text-2xs">
              Next Required
            </Badge>
          ) : null}
        </div>
        <p className="text-xs text-muted-foreground">
          {requirement.satisfied
            ? `${requirement.documentCount} document${requirement.documentCount === 1 ? "" : "s"} uploaded`
            : "Required before the shipment can be invoiced"}
        </p>
      </div>
      <Button
        variant={requirement.satisfied ? "outline" : "secondary"}
        size="sm"
        onClick={onUpload}
        disabled={disabled}
      >
        <FileTextIcon className="size-4" />
        {requirement.satisfied ? "Replace" : "Upload"} {requirement.documentTypeCode}
      </Button>
    </div>
  );
}

function getStatusMessage(readiness: ShipmentBillingReadiness, shipment?: Shipment) {
  const completedCount = readiness.requirements.length - readiness.missingRequirements.length;

  if (shipment?.status === "Invoiced") {
    return {
      title: "Shipment has already been invoiced",
      description:
        "All billing requirements are satisfied and this shipment has already moved through invoicing.",
      variant: "default" as const,
      icon: CheckCircle2Icon,
    };
  }

  if (shipment?.status === "ReadyToInvoice") {
    return {
      title: "Shipment is ready to invoice",
      description:
        "All billing requirements are satisfied and the shipment has already been marked Ready To Invoice.",
      variant: "default" as const,
      icon: CheckCircle2Icon,
    };
  }

  if (shipment?.status !== "Completed") {
    if (readiness.missingRequirements.length === 0 && readiness.validationFailures.length === 0) {
      return {
        title: "Billing requirements are complete",
        description:
          "The required billing documents are already uploaded. This shipment will move to Ready To Invoice after it is marked Completed.",
        variant: "warning" as const,
        icon: Clock3Icon,
      };
    }

    return {
      title: "Billing requirements are still in progress",
      description:
        readiness.requirements.length > 0
          ? `${completedCount}/${readiness.requirements.length} required documents are uploaded. The shipment also must be marked Completed before it can be invoiced.`
          : "This shipment must be marked Completed before it can be invoiced.",
      variant: "warning" as const,
      icon: Clock3Icon,
    };
  }

  if (readiness.canMarkReadyToInvoice) {
    return {
      title: readiness.shouldAutoMarkReadyToInvoice
        ? "Shipment qualifies for automatic billing readiness"
        : "Shipment is ready to invoice",
      description: readiness.shouldAutoMarkReadyToInvoice
        ? "New required-document uploads or completion updates will automatically move this shipment to Ready To Invoice."
        : "All billing requirements are satisfied. You can mark this shipment Ready To Invoice from this panel.",
      variant: "default" as const,
      icon: CheckCircle2Icon,
    };
  }

  return {
    title: "Billing requirements still need attention",
    description:
      "Finish the remaining required documents and validations to make this shipment invoice-ready.",
    variant: "warning" as const,
    icon: AlertCircleIcon,
  };
}

export function ShipmentBillingReadinessPanel({
  readiness,
  shipment,
  onUploadRequired,
  onMarkReadyToInvoice,
  isMarkingReady,
  disabled,
}: ShipmentBillingReadinessPanelProps) {
  const nextMissing = readiness.missingRequirements[0];
  const totalRequirements = readiness.requirements.length;
  const completedRequirements = totalRequirements - readiness.missingRequirements.length;
  const progressValue =
    totalRequirements > 0 ? completedRequirements : readiness.canMarkReadyToInvoice ? 1 : 0;
  const progressMax = totalRequirements > 0 ? totalRequirements : 1;
  const statusMessage = getStatusMessage(readiness, shipment);
  const StatusIcon = statusMessage.icon;
  const canShowManualReady =
    shipment?.status === "Completed" &&
    readiness.canMarkReadyToInvoice &&
    !readiness.shouldAutoMarkReadyToInvoice;

  return (
    <div className="overflow-hidden rounded-xl border bg-card">
      <div className="border-b bg-muted/40 px-4 py-4">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="min-w-0 flex-1 space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <div className="flex size-9 items-center justify-center rounded-lg bg-background text-foreground shadow-sm">
                <ReceiptIcon className="size-4" />
              </div>
              <div className="min-w-0">
                <p className="text-sm font-semibold">Billing Readiness</p>
                <p className="text-xs text-muted-foreground">
                  Track required customer billing documents before invoicing
                </p>
              </div>
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between gap-3 text-xs text-muted-foreground">
                <span>
                  {totalRequirements > 0
                    ? `${completedRequirements} of ${totalRequirements} required documents uploaded`
                    : "No required customer documents configured"}
                </span>
                <span className="tabular-nums">
                  {totalRequirements > 0
                    ? `${Math.round((progressValue / progressMax) * 100)}%`
                    : "0%"}
                </span>
              </div>
              <Progress
                value={progressValue}
                max={progressMax}
                size="sm"
                variant={readiness.canMarkReadyToInvoice ? "success" : "warning"}
              />
            </div>
          </div>

          {canShowManualReady && (
            <Button size="sm" onClick={onMarkReadyToInvoice} disabled={disabled || isMarkingReady}>
              <CheckCircle2Icon className="size-4" />
              {isMarkingReady ? "Marking..." : "Mark Ready To Invoice"}
            </Button>
          )}
        </div>
      </div>

      <div className="space-y-3 px-4 py-4">
        <Alert variant={statusMessage.variant}>
          <StatusIcon className="size-4" />
          <AlertTitle>{statusMessage.title}</AlertTitle>
          <AlertDescription>{statusMessage.description}</AlertDescription>
        </Alert>

        {readiness.validationFailures.length > 0 && (
          <Alert variant="warning">
            <AlertCircleIcon className="size-4" />
            <AlertTitle>Billing validations still need attention</AlertTitle>
            <AlertDescription>
              <div className="space-y-1">
                {readiness.validationFailures.map((failure) => (
                  <p key={`${failure.field}-${failure.code}`} className="text-xs">
                    {failure.message}
                  </p>
                ))}
              </div>
            </AlertDescription>
          </Alert>
        )}

        {readiness.requirements.length > 0 && (
          <div className="space-y-2">
            {readiness.requirements.map((requirement) => (
              <RequirementRow
                key={requirement.documentTypeId}
                requirement={requirement}
                isNext={nextMissing?.documentTypeId === requirement.documentTypeId}
                onUpload={() => onUploadRequired(requirement)}
                disabled={disabled}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
