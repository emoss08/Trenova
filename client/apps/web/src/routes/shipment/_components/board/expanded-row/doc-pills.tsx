import { useT } from "@trenova/shared/i18n/use-t";
import { CheckIcon, ClockIcon, PlusIcon } from "@trenova/shared/components/icons";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { queries } from "@/lib/queries";
import type { Shipment, ShipmentBillingRequirement } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { useShipmentRecordActions } from "../record-actions";

type DocState = "done" | "waiting" | "due";

function docState(requirement: ShipmentBillingRequirement): DocState {
  if (requirement.satisfied) return "done";
  return requirement.documentCount > 0 ? "waiting" : "due";
}

const STATE_CLASS: Record<DocState, string> = {
  done: "border-success-border bg-success-subtle text-success-subtle-foreground",
  waiting: "border-warning-border bg-warning-subtle text-warning-subtle-foreground",
  due: "border-border text-muted-foreground",
};

/**
 * The paperwork billing will ask for, one pill each: on file, uploaded but not
 * yet accepted, or still due. Upload files the next document that is due.
 */
export function DocPills({ shipment }: { shipment: Shipment }) {
  const t = useT();
  const { uploadDocument } = useShipmentRecordActions();
  const shipmentId = shipment.id ?? "";
  const { data, isLoading, isError } = useQuery({
    ...queries.shipment.billingReadiness(shipmentId),
    enabled: !!shipmentId,
  });
  const requirements = data?.requirements ?? [];
  const nextDue = requirements.find((requirement) => !requirement.satisfied);

  return (
    <div className="flex min-w-0 flex-wrap items-center gap-1.5">
      <span className="text-muted-foreground mr-0.5 text-xs">{t("Docs")}</span>
      {isLoading ? (
        <Skeleton className="h-5 w-40" />
      ) : isError ? (
        <span className="text-muted-foreground text-xs">{t("Documents unavailable")}</span>
      ) : requirements.length === 0 ? (
        <span className="text-muted-foreground text-xs">{t("None required")}</span>
      ) : (
        requirements.map((requirement) => {
          const state = docState(requirement);
          return (
            <span
              key={requirement.documentTypeId}
              title={t(requirement.documentTypeName)}
              className={cn(
                "inline-flex h-5 items-center gap-1 rounded-full border px-2 text-xs font-medium",
                STATE_CLASS[state],
              )}
            >
              {state === "done" ? (
                <CheckIcon className="size-3" aria-hidden />
              ) : state === "waiting" ? (
                <ClockIcon className="size-3" aria-hidden />
              ) : (
                <span aria-hidden className="size-1.5 rounded-full border border-current" />
              )}
              {requirement.documentTypeCode || t(requirement.documentTypeName)}
            </span>
          );
        })
      )}
      <button
        type="button"
        className="ui-focus-ring border-border text-muted-foreground hover:text-foreground hover:border-border-strong inline-flex h-5 items-center gap-1 rounded-full border border-dashed px-2 text-xs"
        onClick={() =>
          uploadDocument(
            shipment,
            nextDue
              ? {
                  documentTypeId: nextDue.documentTypeId,
                  documentTypeName: nextDue.documentTypeName,
                }
              : undefined,
          )
        }
      >
        <PlusIcon className="size-3" aria-hidden />
        {t("Upload")}
      </button>
    </div>
  );
}
