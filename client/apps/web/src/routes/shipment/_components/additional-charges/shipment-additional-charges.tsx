"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { ChargePayerControl } from "@/components/billing/charge-payer-control";
import { EmptyState } from "@/components/empty-state";
import { queries } from "@/lib/queries";
import { OccurrenceDetailSheet } from "@/routes/detention-desk/_components/occurrence-detail-sheet";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { FormSection } from "@trenova/shared/components/ui/form";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { chargeLineTotal } from "@trenova/shared/lib/charge-split";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { AccessorialCharge } from "@trenova/shared/types/accessorial-charge";
import type { DetentionOccurrence } from "@trenova/shared/types/detention";
import type { Shipment } from "@trenova/shared/types/shipment";
import {
  BoxesIcon,
  FuelIcon,
  LockIcon,
  PencilIcon,
  PlusIcon,
  ReceiptIcon,
  TrashIcon,
  TriangleAlertIcon,
  TruckIcon,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import {
  DetentionChargeAction,
  DetentionChargeLabel,
  DetentionChargeUnit,
} from "./detention-charge-cell";
import { FuelSurchargeAuditPopover } from "./fuel-surcharge-audit-popover";
import { useShipmentDefaultPayer } from "../use-shipment-default-payer";
import { AdditionalChargeDialog } from "./shipment-additional-charges-dialog";

const NO_OCCURRENCES: DetentionOccurrence[] = [];

function isDetentionCharge(charge: Shipment["additionalCharges"][number] | undefined) {
  return !!charge?.isSystemGenerated && !!charge?.isDetention;
}

/**
 * Flattens react-hook-form's error tree for one charge into the messages the
 * row tooltip lists. A split that does not add up is reported on the
 * allocation array itself (`allocations.root`), and a row missing its payer on
 * that row, so both nested shapes are walked rather than only the top level.
 */
export function collectChargeErrorMessages(chargeErrors: Record<string, unknown>): string[] {
  const messages: string[] = [];
  const visit = (node: unknown) => {
    if (!node || typeof node !== "object") return;
    if (Array.isArray(node)) {
      node.forEach(visit);
      return;
    }
    const record = node as Record<string, unknown>;
    if (typeof record.message === "string" && record.message) {
      messages.push(record.message);
      return;
    }
    for (const [key, value] of Object.entries(record)) {
      if (key === "ref") continue;
      visit(value);
    }
  };
  for (const [key, value] of Object.entries(chargeErrors)) {
    if (key === "ref") continue;
    visit(value);
  }
  return messages.length > 0 ? messages : ["Invalid"];
}

/**
 * One detention charge bills every detained stop on the shipment, so the
 * occurrences are grouped by the charge they point at, in the order the truck
 * reached the stops.
 */
function groupOccurrencesByCharge(occurrences: DetentionOccurrence[] | undefined) {
  const byCharge = new Map<string, DetentionOccurrence[]>();
  for (const occurrence of occurrences ?? []) {
    if (!occurrence.additionalChargeId) continue;
    const list = byCharge.get(occurrence.additionalChargeId) ?? [];
    list.push(occurrence);
    byCharge.set(occurrence.additionalChargeId, list);
  }
  for (const list of byCharge.values()) {
    list.sort((a, b) => (a.arrivedAt ?? 0) - (b.arrivedAt ?? 0));
  }
  return byCharge;
}

export default function AdditionalChargesSection() {
  const t = useT();

  const {
    control,
    setValue,
    formState: { errors },
  } = useFormContext<Shipment>();
  const shipmentId = useWatch({ control, name: "id" });
  const fuelSurchargeLocked = useWatch({ control, name: "fuelSurchargeLocked" });
  const { fields, append, update, remove } = useFieldArray({
    control,
    name: "additionalCharges",
    keyName: "fieldId",
  });

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [claimFileId, setClaimFileId] = useState<string | null>(null);
  const charges = useWatch({ control, name: "additionalCharges" }) ?? [];

  const hasDetentionCharges = charges.some(isDetentionCharge);
  const defaultPayer = useShipmentDefaultPayer();

  const { data: occurrences, refetch: refetchOccurrences } = useQuery({
    ...queries.detention.byShipment(shipmentId as string),
    enabled: Boolean(shipmentId) && hasDetentionCharges,
  });

  // Every save re-runs the detention engine server-side, so the derivation
  // behind these rows is only trustworthy if it is re-read when the shipment
  // version moves.
  const shipmentVersion = useWatch({ control, name: "version" });
  const readOccurrencesAtVersion = useRef(shipmentVersion);

  useEffect(() => {
    if (readOccurrencesAtVersion.current === shipmentVersion) return;
    readOccurrencesAtVersion.current = shipmentVersion;

    if (hasDetentionCharges) void refetchOccurrences();
  }, [shipmentVersion, hasDetentionCharges, refetchOccurrences]);

  const occurrencesByChargeId = useMemo(() => groupOccurrencesByCharge(occurrences), [occurrences]);

  function handleAdd() {
    const newIndex = fields.length;
    append({
      accessorialChargeId: "",
      isSystemGenerated: false,
      method: "Flat",
      amount: 0,
      unit: 1,
      allocations: [],
    });
    setEditingIndex(newIndex);
    setIsEditing(false);
    setDialogOpen(true);
  }

  function handleEdit(index: number) {
    setEditingIndex(index);
    setIsEditing(true);
    setDialogOpen(true);
  }

  function handleDialogCancel() {
    if (editingIndex !== null && !isEditing) {
      remove(editingIndex);
    }
    setDialogOpen(false);
    setEditingIndex(null);
  }

  function handleDialogSave() {
    setDialogOpen(false);
    setEditingIndex(null);
  }

  return (
    <>
      <FormSection
        title={t("Additional charges")}
        description={t(
          "Additional fees charged for services such as detention, fuel surcharge, and more.",
        )}
        action={
          fields.length > 0 && (
            <Button type="button" variant="outline" size="xxs" onClick={handleAdd}>
              <PlusIcon className="size-3" />
              {t("Add charge")}
            </Button>
          )
        }
      >
        {fields.length > 0 ? (
          <div className="rounded-lg border">
            <div className="border-border text-xs text-muted-foreground grid grid-cols-10 gap-2 border-b px-4 py-2">
              <span className="col-span-4">{t("Charge")}</span>
              <span className="col-span-2">{t("Unit")}</span>
              <span className="col-span-2">{t("Amount")}</span>
              <span className="col-span-2" />
            </div>
            <div className="divide-y">
              {fields.map((field, index) => {
                const charge = charges[index];
                const chargeObj = (charge as any)?.accessorialCharge as
                  | AccessorialCharge
                  | undefined;
                const isFuelSurcharge =
                  !!charge?.isSystemGenerated && !!charge?.fuelSurchargeProgramId;
                const isDetention = isDetentionCharge(charge);
                const chargeOccurrences =
                  isDetention && charge?.id
                    ? (occurrencesByChargeId.get(charge.id) ?? NO_OCCURRENCES)
                    : NO_OCCURRENCES;
                const displayName = isFuelSurcharge
                  ? (chargeObj?.code ??
                    charge?.fuelSurchargeDetail?.programCode ??
                    "Fuel Surcharge")
                  : (chargeObj?.code ??
                    chargeObj?.description ??
                    (isDetention ? "Detention" : "—"));
                const amt = Number(charge?.amount) || 0;
                // A percentage accessorial is priced against the freight on the
                // server, so only a percent split can be checked against it here.
                const percentOnly = charge?.method === "Percentage";
                const lineTotal = percentOnly
                  ? null
                  : chargeLineTotal({
                      method: charge?.method,
                      amount: charge?.amount,
                      unit: charge?.unit,
                    });

                const chargeErrors = errors.additionalCharges?.[index];
                const hasErrors = !!(chargeErrors && Object.keys(chargeErrors).length > 0);
                const errorMessages = hasErrors ? collectChargeErrorMessages(chargeErrors) : [];

                return (
                  <div
                    key={field.fieldId}
                    className={cn(
                      "grid grid-cols-10 items-center gap-2 px-4 py-2",
                      hasErrors && "bg-danger-subtle ring-destructive ring-1 ring-inset",
                    )}
                  >
                    <span className="col-span-4 flex items-center gap-1.5 truncate text-xs font-medium">
                      {isFuelSurcharge && <FuelIcon className="text-primary size-3 shrink-0" />}
                      {isDetention ? (
                        <DetentionChargeLabel code={displayName} occurrences={chargeOccurrences} />
                      ) : (
                        displayName
                      )}
                      <ChargePayerControl
                        name={`additionalCharges.${index}.allocations`}
                        chargeAmount={lineTotal}
                        percentOnly={percentOnly}
                        defaultPayer={defaultPayer}
                        splitTitle={t("Split {0}", displayName)}
                        splitDescription={
                          lineTotal == null
                            ? t(
                                "Divide this charge between the customers who pay for it. Each payer receives an invoice for their share.",
                              )
                            : t(
                                "Divide the {0} charge between the customers who pay for it. Each payer receives an invoice for their share.",
                                formatCurrency(lineTotal),
                              )
                        }
                      />
                      {isFuelSurcharge && !fuelSurchargeLocked && (
                        <span className="bg-primary/10 text-2xs text-primary rounded-md px-1 py-0.5">
                          {t("Auto")}
                        </span>
                      )}
                      {isFuelSurcharge && fuelSurchargeLocked && (
                        <Tooltip>
                          <TooltipTrigger>
                            <button
                              type="button"
                              onClick={() =>
                                setValue("fuelSurchargeLocked", false, { shouldDirty: true })
                              }
                              className="text-2xs flex items-center gap-1 rounded-md bg-warning-subtle px-1 py-0.5 text-warning-foreground"
                            >
                              <LockIcon className="size-2.5" />
                              {t("Locked")}
                            </button>
                          </TooltipTrigger>
                          <TooltipContent side="top" sideOffset={6}>
                            <p className="max-w-56 text-xs">
                              {t(
                                "Kept at its original amount — shipment changes won't re-rate it. Click to unlock and re-rate automatically.",
                              )}
                            </p>
                          </TooltipContent>
                        </Tooltip>
                      )}
                    </span>
                    <span className="text-muted-foreground col-span-2 text-xs">
                      {isDetention ? (
                        <DetentionChargeUnit
                          unit={charge?.unit ?? 1}
                          occurrences={chargeOccurrences}
                        />
                      ) : (
                        (charge?.unit ?? 1)
                      )}
                    </span>
                    <span className="text-muted-foreground col-span-2 text-xs">
                      ${amt.toFixed(2)}
                    </span>
                    <div className="col-span-2 flex items-center justify-end gap-1">
                      {isFuelSurcharge && charge?.fuelSurchargeDetail ? (
                        <FuelSurchargeAuditPopover detail={charge.fuelSurchargeDetail} />
                      ) : isDetention ? (
                        <DetentionChargeAction
                          occurrences={chargeOccurrences}
                          onOpenClaimFile={setClaimFileId}
                        />
                      ) : (
                        <>
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            className="size-7"
                            onClick={() => handleEdit(index)}
                          >
                            <PencilIcon className="text-muted-foreground size-3.5" />
                          </Button>
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            className="size-7"
                            onClick={() => remove(index)}
                          >
                            <TrashIcon className="text-muted-foreground size-3.5" />
                          </Button>
                        </>
                      )}
                      {hasErrors && (
                        <Tooltip>
                          <TooltipTrigger>
                            <TriangleAlertIcon className="text-destructive size-3.5 cursor-help" />
                          </TooltipTrigger>
                          <TooltipContent side="top" sideOffset={10}>
                            <div className="space-y-1">
                              {errorMessages.map((msg, idx) => (
                                <p key={idx} className="text-xs">
                                  {msg}
                                </p>
                              ))}
                            </div>
                          </TooltipContent>
                        </Tooltip>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        ) : (
          <EmptyState
            title={t("No additional charges")}
            description={t("Shipment has no associated additional charges")}
            icons={[ReceiptIcon, BoxesIcon, TruckIcon]}
            className="border-bg-sidebar-border max-h-50 rounded-lg border p-4"
            action={{
              label: t("Add first charge"),
              onClick: handleAdd,
              icon: PlusIcon,
            }}
          />
        )}
      </FormSection>
      {editingIndex !== null && (
        <AdditionalChargeDialog
          open={dialogOpen}
          onCancel={handleDialogCancel}
          onSave={handleDialogSave}
          index={editingIndex}
          isEditing={isEditing}
          update={update}
        />
      )}
      <OccurrenceDetailSheet
        occurrenceId={claimFileId}
        onOpenChange={(open) => {
          if (!open) setClaimFileId(null);
        }}
      />
    </>
  );
}
