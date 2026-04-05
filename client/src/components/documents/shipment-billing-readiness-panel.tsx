import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
  UploadIcon,
} from "lucide-react";

interface ShipmentBillingReadinessPanelProps {
  readiness: ShipmentBillingReadiness;
  shipment?: Shipment;
  onUploadRequired: (requirement: ShipmentBillingRequirement) => void;
  onMarkReadyToInvoice: () => void;
  isMarkingReady: boolean;
  disabled?: boolean;
}

function getProgressVariant(
  completed: number,
  total: number,
  canMarkReady: boolean,
): "error" | "warning" | "success" | "default" {
  if (total === 0) return canMarkReady ? "success" : "default";
  const pct = completed / total;
  if (pct === 1) return "success";
  if (pct >= 0.5) return "warning";
  return "error";
}

function getStatusHint(readiness: ShipmentBillingReadiness, shipment?: Shipment) {
  if (shipment?.status === "Invoiced") return "Invoiced";
  if (shipment?.status === "ReadyToInvoice") return "Ready to invoice";
  if (shipment?.status !== "Completed") return "Awaiting shipment completion";
  if (readiness.canMarkReadyToInvoice) {
    return readiness.shouldAutoMarkReadyToInvoice ? "Auto-ready enabled" : "Ready to mark";
  }
  return "Documents needed";
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
  const done = requirement.satisfied;

  return (
    <div
      className={[
        "flex items-center justify-between gap-3 rounded-lg border px-3 py-2.5 transition-colors",
        done
          ? "border-green-500/20 bg-green-500/[0.03]"
          : isNext
            ? "border-primary/25 bg-primary/[0.03]"
            : "border-transparent bg-muted/40",
      ].join(" ")}
    >
      <div className="flex min-w-0 items-center gap-2.5">
        {done ? (
          <div className="flex size-5 shrink-0 items-center justify-center rounded-full bg-green-500/15">
            <CheckCircle2Icon className="size-3 text-green-600 dark:text-green-400" />
          </div>
        ) : (
          <div className="flex size-5 shrink-0 items-center justify-center rounded-full bg-muted">
            <UploadIcon className="size-3 text-muted-foreground" />
          </div>
        )}
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">{requirement.documentTypeName}</p>
          <p className="text-2xs text-muted-foreground">{requirement.documentTypeCode}</p>
        </div>
      </div>

      <Button
        variant={isNext && !done ? "default" : "outline"}
        size="xxs"
        onClick={onUpload}
        disabled={disabled}
        className="shrink-0"
      >
        {done ? "Replace" : "Upload"}
      </Button>
    </div>
  );
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
  const progressVariant = getProgressVariant(
    completedRequirements,
    totalRequirements,
    readiness.canMarkReadyToInvoice,
  );
  const statusHint = getStatusHint(readiness, shipment);
  const canShowManualReady =
    shipment?.status === "Completed" &&
    readiness.canMarkReadyToInvoice &&
    !readiness.shouldAutoMarkReadyToInvoice;

  return (
    <div className="overflow-hidden rounded-xl border bg-card">
      {/* Header */}
      <div className="border-b px-4 py-3">
        <div className="flex items-center justify-between gap-4">
          <span className="text-sm font-semibold">Billing Readiness</span>
          <span className="text-2xs text-muted-foreground">{statusHint}</span>
        </div>

        <div className="mt-3 space-y-1.5">
          <Progress value={progressValue} max={progressMax} variant={progressVariant} />
          <p className="text-xs tabular-nums text-muted-foreground">
            {totalRequirements > 0
              ? `${completedRequirements} of ${totalRequirements} documents uploaded`
              : "No documents required"}
          </p>
        </div>
      </div>

      {/* Body */}
      <div className="px-4 py-3">
        {readiness.validationFailures.length > 0 && (
          <Alert variant="destructive" className="mb-3">
            <AlertCircleIcon className="size-4" />
            <AlertTitle>Validation issues</AlertTitle>
            <AlertDescription>
              <ul className="list-inside list-disc space-y-0.5">
                {readiness.validationFailures.map((failure) => (
                  <li key={`${failure.field}-${failure.code}`} className="text-xs">
                    {failure.message}
                  </li>
                ))}
              </ul>
            </AlertDescription>
          </Alert>
        )}

        {totalRequirements > 0 ? (
          <div className="space-y-1.5">
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
        ) : (
          <div className="flex items-center justify-center gap-2 py-6 text-muted-foreground">
            <Clock3Icon className="size-4" />
            <p className="text-sm">No billing documents required.</p>
          </div>
        )}

        {canShowManualReady && (
          <div className="mt-3 border-t pt-3">
            <Button
              className="w-full"
              size="sm"
              onClick={onMarkReadyToInvoice}
              disabled={disabled}
              isLoading={isMarkingReady}
              loadingText="Marking..."
            >
              <CheckCircle2Icon className="size-4" />
              Mark Ready To Invoice
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
