"use no memo";
import { EmptyState } from "@/components/empty-state";
import { queries } from "@/lib/queries";
import { OccurrenceDetailSheet } from "@/routes/detention-desk/_components/occurrence-detail-sheet";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { FormSection } from "@trenova/shared/components/ui/form";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn } from "@trenova/shared/lib/utils";
import type { AccessorialCharge } from "@trenova/shared/types/accessorial-charge";
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
import { AdditionalChargeDialog } from "./shipment-additional-charges-dialog";

function detentionOccurrenceId(charge: Shipment["additionalCharges"][number] | undefined) {
  return charge?.isSystemGenerated ? (charge.detentionOccurrenceId ?? null) : null;
}

export default function AdditionalChargesSection() {
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

  const hasDetentionCharges = charges.some((charge) => detentionOccurrenceId(charge));

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

  const occurrenceById = useMemo(
    () => new Map((occurrences ?? []).map((occurrence) => [occurrence.id, occurrence])),
    [occurrences],
  );

  function handleAdd() {
    const newIndex = fields.length;
    append({
      accessorialChargeId: "",
      isSystemGenerated: false,
      method: "Flat",
      amount: 0,
      unit: 1,
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
        title="Additional Charges"
        description="Additional fees charged for services such as detention, fuel surcharge, and more."
        action={
          fields.length > 0 && (
            <Button type="button" variant="outline" size="xxs" onClick={handleAdd}>
              <PlusIcon className="size-3" />
              Add Charge
            </Button>
          )
        }
      >
        {fields.length > 0 ? (
          <div className="rounded-lg border">
            <div className="border-border text-2xs text-muted-foreground grid grid-cols-10 gap-2 border-b px-4 py-2 uppercase">
              <span className="col-span-4">Charge</span>
              <span className="col-span-2">Unit</span>
              <span className="col-span-2">Amount</span>
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
                const occurrenceId = detentionOccurrenceId(charge);
                const occurrence = occurrenceId ? occurrenceById.get(occurrenceId) : undefined;
                const displayName = isFuelSurcharge
                  ? (chargeObj?.code ??
                    charge?.fuelSurchargeDetail?.programCode ??
                    "Fuel Surcharge")
                  : (chargeObj?.code ??
                    chargeObj?.description ??
                    (occurrenceId ? "Detention" : "—"));
                const amt = Number(charge?.amount) || 0;

                const chargeErrors = errors.additionalCharges?.[index];
                const hasErrors = !!(chargeErrors && Object.keys(chargeErrors).length > 0);
                const errorMessages = hasErrors
                  ? Object.entries(chargeErrors as Record<string, { message?: string }>)
                      .filter(([key]) => key !== "ref" && key !== "root")
                      .map(([, err]) => err?.message ?? "Invalid")
                  : [];

                return (
                  <div
                    key={field.fieldId}
                    className={cn(
                      "grid grid-cols-10 items-center gap-2 px-4 py-2",
                      hasErrors && "bg-destructive/10 ring-destructive ring-1 ring-inset",
                    )}
                  >
                    <span className="col-span-4 flex items-center gap-1.5 truncate text-xs font-medium">
                      {isFuelSurcharge && <FuelIcon className="text-primary size-3 shrink-0" />}
                      {occurrenceId ? (
                        <DetentionChargeLabel code={displayName} occurrence={occurrence} />
                      ) : (
                        displayName
                      )}
                      {isFuelSurcharge && !fuelSurchargeLocked && (
                        <span className="bg-primary/10 text-2xs text-primary rounded px-1 py-0.5">
                          Auto
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
                              className="text-2xs flex items-center gap-1 rounded bg-amber-500/10 px-1 py-0.5 text-amber-600 dark:text-amber-400"
                            >
                              <LockIcon className="size-2.5" />
                              Locked
                            </button>
                          </TooltipTrigger>
                          <TooltipContent side="top" sideOffset={6}>
                            <p className="max-w-56 text-xs">
                              Kept at its original amount — shipment changes won&apos;t re-rate it.
                              Click to unlock and re-rate automatically.
                            </p>
                          </TooltipContent>
                        </Tooltip>
                      )}
                    </span>
                    <span className="text-muted-foreground col-span-2 text-xs">
                      {occurrenceId ? (
                        <DetentionChargeUnit unit={charge?.unit ?? 1} occurrence={occurrence} />
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
                      ) : occurrenceId ? (
                        <DetentionChargeAction
                          occurrence={occurrence}
                          onOpenClaimFile={() => setClaimFileId(occurrenceId)}
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
            title="No Additional Charges"
            description="Shipment has no associated additional charges"
            icons={[ReceiptIcon, BoxesIcon, TruckIcon]}
            className="border-bg-sidebar-border max-h-[200px] rounded-lg border p-4"
            action={{
              label: "Add First Charge",
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
