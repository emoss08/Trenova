import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  NumberFieldGroup,
  NumberFieldInput,
  NumberField as NumberFieldRoot,
} from "@/components/ui/number-field";
import { Separator } from "@/components/ui/separator";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn, formatCurrency } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { BillingQueueItem } from "@/types/billing-queue";
import type { AdditionalCharge } from "@/types/shipment";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckIcon,
  PencilIcon,
  PlusIcon,
  RepeatIcon,
  TrashIcon,
  XIcon,
} from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";
import {
  BillingQueueChargeDialog,
  type ChargeDialogResult,
} from "./billing-queue-charge-dialog";
import { BillingQueueRerateDialog } from "./billing-queue-rerate-dialog";

function getChargeWarnings(
  freightCharge: number,
  totalCharge: number,
  additionalCharges: AdditionalCharge[],
): string[] {
  const warnings: string[] = [];
  if (totalCharge === 0) warnings.push("Total charge is $0.00");
  if (freightCharge === 0 && additionalCharges.length > 0)
    warnings.push("Freight charge is $0.00 but accessorial charges exist");
  for (const charge of additionalCharges) {
    if (Number(charge.amount ?? 0) < 0) {
      warnings.push("One or more charges have a negative amount");
      break;
    }
  }
  if (freightCharge < 0) warnings.push("Freight charge is negative");
  return warnings;
}

function formatChargeBreakdown(charge: AdditionalCharge): string {
  const amount = Number(charge.amount ?? 0);
  const unit = charge.unit ?? 1;
  switch (charge.method) {
    case "PerUnit":
      return `${formatCurrency(amount)} × ${unit} units`;
    case "Percentage":
      return `${amount}% of line haul`;
    case "Flat":
      return unit > 1 ? `${formatCurrency(amount)} × ${unit}` : "Flat";
    default:
      return "Flat";
  }
}

function chargeLineTotal(charge: AdditionalCharge): number {
  const amount = Number(charge.amount ?? 0);
  const unit = charge.unit ?? 1;
  switch (charge.method) {
    case "PerUnit":
    case "Flat":
      return amount * unit;
    default:
      return amount;
  }
}

export function BillingQueueChargesTab({ item }: { item: BillingQueueItem }) {
  const shipment = item.shipment;
  const isEditable = item.status === "InReview";
  const queryClient = useQueryClient();

  const [chargeDialogOpen, setChargeDialogOpen] = useState(false);
  const [rerateDialogOpen, setRerateDialogOpen] = useState(false);
  const [editingFreight, setEditingFreight] = useState(false);
  const [freightDraft, setFreightDraft] = useState("");
  const [editingCharge, setEditingCharge] = useState<
    (Partial<ChargeDialogResult> & { index: number }) | null
  >(null);

  const { mutate: saveCharges, isPending } = useMutation({
    mutationFn: (payload: { baseRate?: string; additionalCharges?: any[] }) =>
      apiService.billingQueueService.updateCharges(item.id, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
      void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
      toast.success("Charges updated");
    },
    onError: () => {
      toast.error("Failed to update charges");
    },
  });

  const buildChargesPayload = useCallback(
    (charges: AdditionalCharge[]) =>
      charges.map((c) => ({
        id: c.id,
        accessorialChargeId: c.accessorialChargeId,
        method: c.method,
        amount: c.amount,
        unit: c.unit ?? 1,
      })),
    [],
  );

  const handleAddCharge = useCallback(
    (values: ChargeDialogResult) => {
      const currentCharges = shipment?.additionalCharges ?? [];
      saveCharges({
        additionalCharges: [
          ...buildChargesPayload(currentCharges),
          {
            accessorialChargeId: values.accessorialChargeId,
            method: values.method,
            amount: values.amount,
            unit: values.unit,
          },
        ],
      });
    },
    [shipment, saveCharges, buildChargesPayload],
  );

  const handleEditCharge = useCallback(
    (values: ChargeDialogResult) => {
      if (editingCharge === null) return;
      const currentCharges = [...(shipment?.additionalCharges ?? [])];
      const payload = buildChargesPayload(currentCharges);
      payload[editingCharge.index] = {
        id: values.id,
        accessorialChargeId: values.accessorialChargeId,
        method: values.method,
        amount: values.amount,
        unit: values.unit,
      };
      saveCharges({ additionalCharges: payload });
      setEditingCharge(null);
    },
    [editingCharge, shipment, saveCharges, buildChargesPayload],
  );

  const handleDeleteCharge = useCallback(
    (index: number) => {
      const currentCharges = [...(shipment?.additionalCharges ?? [])];
      currentCharges.splice(index, 1);
      saveCharges({ additionalCharges: buildChargesPayload(currentCharges) });
    },
    [shipment, saveCharges, buildChargesPayload],
  );

  if (!shipment) {
    return (
      <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
        Shipment details not available
      </div>
    );
  }

  const freightCharge = Number(shipment.freightChargeAmount ?? 0);
  const baseRate = Number(shipment.baseRate ?? 0);
  const otherCharge = Number(shipment.otherChargeAmount ?? 0);
  const totalCharge = Number(shipment.totalChargeAmount ?? 0);
  const additionalCharges = shipment.additionalCharges ?? [];
  const formulaTemplate = shipment.formulaTemplate;
  const warnings = getChargeWarnings(
    freightCharge,
    totalCharge,
    additionalCharges,
  );

  return (
    <div className="flex flex-col gap-3 p-4">
      {warnings.length > 0 && (
        <Alert variant="warning">
          <AlertTriangleIcon className="size-4" />
          <AlertDescription>
            <ul className="list-disc space-y-0.5 pl-4 text-xs">
              {warnings.map((w) => (
                <li key={w}>{w}</li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      )}

      {formulaTemplate && (
        <div className="flex items-center justify-between rounded-md border border-border bg-muted px-3 py-2">
          <div className="flex items-center gap-2 text-xs">
            <span className="text-muted-foreground">Rating:</span>
            <span className="font-medium">{formulaTemplate.name}</span>
            {formulaTemplate.expression && (
              <>
                <span className="text-muted-foreground/50">&middot;</span>
                <code className="font-mono text-muted-foreground">
                  {formulaTemplate.expression}
                </code>
              </>
            )}
          </div>
          {isEditable && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    size="icon-xs"
                    variant="ghost"
                    onClick={() => setRerateDialogOpen(true)}
                    disabled={isPending}
                  >
                    <RepeatIcon className="size-3" />
                  </Button>
                }
              />
              <TooltipContent side="top" sideOffset={10}>
                Change Template
              </TooltipContent>
            </Tooltip>
          )}
        </div>
      )}

      <div className="flex flex-col">
        <div className="group flex items-center justify-between gap-2 rounded-md p-2 hover:bg-muted">
          <div className="flex min-w-0 flex-col">
            <span className="text-sm">Base Rate</span>
            <span className="text-[11px] text-muted-foreground">
              Per-unit rate before formula
            </span>
          </div>
          {editingFreight ? (
            <div className="flex items-center gap-1">
              <NumberFieldRoot
                value={Number(freightDraft) || 0}
                onValueChange={(val) => setFreightDraft(String(val ?? 0))}
                step={0.01}
                min={0}
                size="sm"
                className="w-32"
              >
                <NumberFieldGroup>
                  <NumberFieldInput
                    autoFocus
                    className="text-right"
                    onKeyDown={(e) => {
                      if (e.key === "Escape") setEditingFreight(false);
                      if (e.key === "Enter") {
                        saveCharges({
                          baseRate: freightDraft,
                          additionalCharges:
                            buildChargesPayload(additionalCharges),
                        });
                        setEditingFreight(false);
                      }
                    }}
                  />
                </NumberFieldGroup>
              </NumberFieldRoot>
              <Button
                size="icon-xs"
                variant="ghostInvert"
                disabled={isPending}
                onClick={() => {
                  saveCharges({
                    baseRate: freightDraft,
                    additionalCharges: buildChargesPayload(additionalCharges),
                  });
                  setEditingFreight(false);
                }}
              >
                <CheckIcon className="size-3 text-green-600" />
              </Button>
              <Button
                size="icon-xs"
                variant="ghostInvert"
                onClick={() => setEditingFreight(false)}
              >
                <XIcon className="size-3 text-muted-foreground" />
              </Button>
            </div>
          ) : (
            <div className="relative flex min-w-[80px] items-center justify-end">
              <span
                className={cn(
                  "text-sm font-medium tabular-nums transition-opacity",
                  isEditable ? "group-hover:opacity-0" : "",
                )}
              >
                {formatCurrency(baseRate)}
              </span>
              {isEditable && (
                <div className="absolute inset-0 flex items-center justify-end gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <Button
                          size="icon-xs"
                          variant="ghostInvert"
                          disabled={isPending}
                          onClick={() => {
                            setFreightDraft(String(baseRate));
                            setEditingFreight(true);
                          }}
                        >
                          <PencilIcon className="size-3" />
                        </Button>
                      }
                    />
                    <TooltipContent side="top" sideOffset={10}>
                      Adjust Base Rate
                    </TooltipContent>
                  </Tooltip>
                </div>
              )}
            </div>
          )}
        </div>

        <div className="flex items-center justify-between gap-2 rounded-md p-2">
          <span className="text-sm">Line Haul</span>
          <span className="text-sm font-medium tabular-nums">
            {formatCurrency(freightCharge)}
          </span>
        </div>
        <Separator className="my-1" />
        <div className="flex items-center justify-between p-2">
          <span className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            Accessorials
          </span>
          {isEditable && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    size="icon-xs"
                    variant="ghost"
                    onClick={() => setChargeDialogOpen(true)}
                    disabled={isPending}
                  >
                    <PlusIcon className="size-3" />
                  </Button>
                }
              />
              <TooltipContent side="top" sideOffset={10}>
                Add Charge
              </TooltipContent>
            </Tooltip>
          )}
        </div>

        {additionalCharges.length === 0 && (
          <p className="px-2 pb-2 text-xs text-muted-foreground">
            No accessorial charges
          </p>
        )}

        {additionalCharges.map((charge, index) => {
          const name =
            charge.accessorialCharge?.description ??
            charge.accessorialCharge?.code ??
            "Charge";
          const breakdown = formatChargeBreakdown(charge);
          const total = chargeLineTotal(charge);

          return (
            <div
              key={charge.id ?? index}
              className="group flex items-center justify-between gap-2 rounded-md p-2 hover:bg-muted"
            >
              <div className="flex min-w-0 flex-col">
                <span className="truncate text-sm">{name}</span>
                <span className="text-[11px] text-muted-foreground">
                  {breakdown}
                </span>
              </div>
              <div className="relative flex min-w-[80px] items-center justify-end">
                <span
                  className={cn(
                    "text-sm font-medium tabular-nums transition-opacity",
                    isEditable ? "group-hover:opacity-0" : "",
                  )}
                >
                  {formatCurrency(total)}
                </span>
                {isEditable && (
                  <div className="absolute inset-0 flex items-center justify-end gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
                    <Tooltip>
                      <TooltipTrigger
                        render={
                          <Button
                            size="icon-xs"
                            variant="ghostInvert"
                            disabled={isPending}
                            onClick={() =>
                              setEditingCharge({
                                index,
                                id: charge.id,
                                accessorialChargeId: charge.accessorialChargeId,
                                method: charge.method,
                                amount: Number(charge.amount),
                                unit: charge.unit ?? 1,
                                accessorialCharge: charge.accessorialCharge,
                              })
                            }
                          >
                            <PencilIcon className="size-3" />
                          </Button>
                        }
                      />
                      <TooltipContent side="top" sideOffset={10}>
                        Edit
                      </TooltipContent>
                    </Tooltip>
                    <Tooltip>
                      <TooltipTrigger
                        render={
                          <Button
                            size="icon-xs"
                            variant="ghostInvert"
                            disabled={isPending || charge.isSystemGenerated}
                            onClick={() => handleDeleteCharge(index)}
                          >
                            <TrashIcon className="size-3" />
                          </Button>
                        }
                      />
                      <TooltipContent side="top" sideOffset={10}>
                        Delete
                      </TooltipContent>
                    </Tooltip>
                  </div>
                )}
              </div>
            </div>
          );
        })}

        {additionalCharges.length > 0 && (
          <div className="flex items-center justify-between p-2 text-muted-foreground">
            <span className="text-xs">Subtotal</span>
            <span className="text-xs font-medium tabular-nums">
              {formatCurrency(otherCharge)}
            </span>
          </div>
        )}
      </div>

      <div className="flex items-center justify-between rounded-md bg-muted/50 px-3 py-2.5">
        <span className="text-sm font-semibold">Total</span>
        <span className="text-base font-bold tabular-nums">
          {formatCurrency(totalCharge)}
        </span>
      </div>

      {chargeDialogOpen && (
        <BillingQueueChargeDialog
          open={chargeDialogOpen}
          onOpenChange={setChargeDialogOpen}
          onSave={handleAddCharge}
        />
      )}
      {editingCharge && (
        <BillingQueueChargeDialog
          open={!!editingCharge}
          onOpenChange={(open) => !open && setEditingCharge(null)}
          onSave={handleEditCharge}
          defaultValues={editingCharge}
        />
      )}
      {rerateDialogOpen && (
        <BillingQueueRerateDialog
          open={rerateDialogOpen}
          onOpenChange={setRerateDialogOpen}
          itemId={item.id}
          currentTemplateId={shipment.formulaTemplateId}
        />
      )}
    </div>
  );
}
