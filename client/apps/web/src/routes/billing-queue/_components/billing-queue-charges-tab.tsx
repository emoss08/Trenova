import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import { describeApiError } from "@/lib/api-error-message";
import {
  describeLinePayers,
  isChargeReassignable,
  staleAmountSplitMessages,
} from "@/lib/billing-queue-charges";
import { apiService } from "@/services/api";
import { useMutation } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import {
  NumberFieldGroup,
  NumberFieldInput,
  NumberField as NumberFieldRoot,
} from "@trenova/shared/components/ui/number-field";
import { Separator } from "@trenova/shared/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { chargeLineTotal } from "@trenova/shared/lib/charge-split";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type {
  BillingQueueItem,
  BillingQueueUpdateChargesInput,
  PayerShare,
  PayerShareLine,
} from "@trenova/shared/types/billing-queue";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { AdditionalCharge } from "@trenova/shared/types/shipment";
import {
  AlertTriangleIcon,
  CheckIcon,
  ChevronDownIcon,
  InfoIcon,
  PencilIcon,
  PlusIcon,
  RepeatIcon,
  TrashIcon,
  UsersRoundIcon,
  XIcon,
} from "lucide-react";
import { useCallback, useState, type ReactNode } from "react";
import { toast } from "sonner";
import { BillingQueueChargeDialog, type ChargeDialogResult } from "./billing-queue-charge-dialog";
import { BillingQueueReassignChargeDialog } from "./billing-queue-reassign-charge-dialog";
import { BillingQueueRerateDialog } from "./billing-queue-rerate-dialog";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";

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

function chargeName(charge: AdditionalCharge | undefined, fallback: string): string {
  return charge?.accessorialCharge?.description ?? charge?.accessorialCharge?.code ?? fallback;
}

type ChargeRowAction = {
  label: string;
  icon: ReactNode;
  onClick: () => void;
  disabled?: boolean;
};

/**
 * One line of the charge sheet. Its actions stay in the DOM with an accessible
 * name and fade in on hover, so they are reachable by keyboard and by tests.
 */
function ChargeRow({
  testId,
  name,
  details,
  amount,
  secondary,
  actions,
  muted = false,
}: {
  testId?: string;
  name: string;
  details?: ReactNode;
  amount: number | null | undefined;
  secondary?: string | null;
  actions: ChargeRowAction[];
  muted?: boolean;
}) {
  return (
    <div
      className="group hover:bg-muted flex items-center justify-between gap-2 rounded-md p-2"
      data-testid={testId}
    >
      <div className="flex min-w-0 flex-col">
        <span className={cn("truncate text-sm", muted && "text-muted-foreground")}>{name}</span>
        {details}
      </div>
      <div className="relative flex min-w-20 items-center justify-end">
        <span
          className={cn(
            "flex flex-col items-end transition-opacity",
            actions.length > 0 && "group-focus-within:opacity-0 group-hover:opacity-0",
          )}
        >
          <span
            className={cn(
              "text-sm font-medium tabular-nums",
              muted && "text-muted-foreground font-normal",
            )}
          >
            {formatCurrency(Number(amount ?? 0))}
          </span>
          {secondary ? (
            <span className="text-muted-foreground text-xs tabular-nums">{secondary}</span>
          ) : null}
        </span>
        {actions.length > 0 ? (
          <div className="absolute inset-0 flex items-center justify-end gap-0.5 opacity-0 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100">
            {actions.map((action) => (
              <Tooltip key={action.label}>
                <TooltipTrigger
                  render={
                    <Button
                      size="icon-xs"
                      variant="ghostInvert"
                      aria-label={action.label}
                      disabled={action.disabled}
                      onClick={action.onClick}
                    >
                      {action.icon}
                    </Button>
                  }
                />
                <TooltipContent side="top" sideOffset={10}>
                  {action.label}
                </TooltipContent>
              </Tooltip>
            ))}
          </div>
        ) : null}
      </div>
    </div>
  );
}

export function BillingQueueChargesTab({ item }: { item: BillingQueueItem }) {
  const t = useT();

  const shipment = item.shipment;
  const isEditable = item.status === "InReview";
  const invalidate = useInvalidateBillingQueue();
  const { allowed: canUpdate } = usePermission(Resource.BillingQueue, Operation.Update);
  const canReassign = canUpdate && isChargeReassignable(item.status);

  const [chargeDialogOpen, setChargeDialogOpen] = useState(false);
  const [rerateDialogOpen, setRerateDialogOpen] = useState(false);
  const [editingFreight, setEditingFreight] = useState(false);
  const [freightDraft, setFreightDraft] = useState("");
  const [editingCharge, setEditingCharge] = useState<
    (Partial<ChargeDialogResult> & { index: number }) | null
  >(null);
  const [staleSplit, setStaleSplit] = useState<{
    payload: BillingQueueUpdateChargesInput;
    messages: string[];
  } | null>(null);
  const [reassignLine, setReassignLine] = useState<PayerShareLine | null>(null);

  const { mutate: saveCharges, isPending } = useMutation({
    mutationFn: (payload: BillingQueueUpdateChargesInput) =>
      apiService.billingQueueService.updateCharges(item.id, payload),
    onSuccess: (_updated, payload) => {
      setStaleSplit(null);
      invalidate();
      toast.success(
        payload.convertAmountSplitsToPercent
          ? t("Charges updated and amount splits converted to percentages")
          : t("Charges updated"),
      );
    },
    onError: (error, payload) => {
      const messages = staleAmountSplitMessages(error);
      if (messages.length > 0 && !payload.convertAmountSplitsToPercent) {
        setStaleSplit({ payload, messages });
        return;
      }
      setStaleSplit(null);
      toast.error(t("Failed to update charges"), { description: describeApiError(error) });
    },
  });

  const buildChargesPayload = useCallback(
    (charges: AdditionalCharge[]) =>
      charges.map((c) => ({
        id: c.id,
        accessorialChargeId: c.accessorialChargeId,
        method: c.method,
        amount: c.amount ?? 0,
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
      <div className="text-muted-foreground flex items-center justify-center py-12 text-sm">
        {t("Shipment details not available")}
      </div>
    );
  }

  const freightCharge = Number(shipment.freightChargeAmount ?? 0);
  const baseRate = Number(shipment.baseRate ?? 0);
  const otherCharge = Number(shipment.otherChargeAmount ?? 0);
  const totalCharge = Number(shipment.totalChargeAmount ?? 0);
  const additionalCharges = shipment.additionalCharges ?? [];
  const formulaTemplate = shipment.formulaTemplate;
  const warnings = getChargeWarnings(freightCharge, totalCharge, additionalCharges);
  const resolutionError = item.payerShare?.resolutionError ?? null;
  const share: PayerShare | null = item.payerShare && !resolutionError ? item.payerShare : null;

  const saveBaseRate = () => {
    saveCharges({
      baseRate: freightDraft,
      additionalCharges: buildChargesPayload(additionalCharges),
    });
    setEditingFreight(false);
  };

  const chargeActions = (index: number, charge: AdditionalCharge): ChargeRowAction[] => {
    if (!isEditable) return [];
    const name = chargeName(charge, t("Charge"));
    return [
      {
        label: t("Edit {0}", name),
        icon: <PencilIcon className="size-3" />,
        disabled: isPending,
        onClick: () =>
          setEditingCharge({
            index,
            id: charge.id,
            accessorialChargeId: charge.accessorialChargeId,
            method: charge.method,
            amount: Number(charge.amount),
            unit: charge.unit ?? 1,
            accessorialCharge: charge.accessorialCharge,
          }),
      },
      {
        label: t("Delete {0}", name),
        icon: <TrashIcon className="size-3" />,
        disabled: isPending || charge.isSystemGenerated,
        onClick: () => handleDeleteCharge(index),
      },
    ];
  };

  const reassignAction = (line: PayerShareLine, name: string): ChargeRowAction[] =>
    canReassign
      ? [
          {
            label: t("Change payer for {0}", name),
            icon: <UsersRoundIcon className="size-3" />,
            disabled: isPending,
            onClick: () => setReassignLine(line),
          },
        ]
      : [];

  const lineName = (line: PayerShareLine) => {
    if (line.kind === "Freight") return t("Line Haul");
    const charge = additionalCharges.find((row) => row.id === line.additionalChargeId);
    return chargeName(charge, line.description);
  };

  const ofChargeTotal = (line: PayerShareLine) =>
    line.partial ? t("of {0}", formatCurrency(Number(line.chargeTotal ?? 0))) : null;

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

      {resolutionError ? (
        <Alert variant="warning">
          <AlertTriangleIcon className="size-4" />
          <AlertDescription className="text-xs">{resolutionError}</AlertDescription>
        </Alert>
      ) : null}

      {share && share.payers.length > 1 && (isEditable || canReassign) ? (
        <Alert variant="info">
          <InfoIcon className="size-4" />
          <AlertDescription className="text-xs">
            {t(
              "Charges belong to the shipment, so changing one here also changes the other payers' bills.",
            )}
          </AlertDescription>
        </Alert>
      ) : null}

      {formulaTemplate && (
        <div className="border-border bg-muted flex items-center justify-between rounded-md border px-3 py-2">
          <div className="flex items-center gap-2 text-xs">
            <span className="text-muted-foreground">{t("Rating:")}</span>
            <span className="font-medium">{formulaTemplate.name}</span>
            {formulaTemplate.expression && (
              <>
                <span className="text-muted-foreground/50">&middot;</span>
                <code className="text-muted-foreground font-mono">
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
                    aria-label={t("Change Template")}
                    onClick={() => setRerateDialogOpen(true)}
                    disabled={isPending}
                  >
                    <RepeatIcon className="size-3" />
                  </Button>
                }
              />
              <TooltipContent side="top" sideOffset={10}>
                {t("Change Template")}
              </TooltipContent>
            </Tooltip>
          )}
        </div>
      )}

      <div className="flex flex-col">
        <div className="group hover:bg-muted flex items-center justify-between gap-2 rounded-md p-2">
          <div className="flex min-w-0 flex-col">
            <span className="text-sm">{t("Base Rate")}</span>
            <span className="text-muted-foreground text-xs">
              {share ? t("Shipment rate before formula") : t("Per-unit rate before formula")}
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
                      if (e.key === "Enter") saveBaseRate();
                    }}
                  />
                </NumberFieldGroup>
              </NumberFieldRoot>
              <Button
                size="icon-xs"
                variant="ghostInvert"
                aria-label={t("Save base rate")}
                disabled={isPending}
                onClick={saveBaseRate}
              >
                <CheckIcon className="size-3 text-success-foreground" />
              </Button>
              <Button
                size="icon-xs"
                variant="ghostInvert"
                aria-label={t("Cancel")}
                onClick={() => setEditingFreight(false)}
              >
                <XIcon className="text-muted-foreground size-3" />
              </Button>
            </div>
          ) : (
            <div className="relative flex min-w-20 items-center justify-end">
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
                          aria-label={t("Adjust Base Rate")}
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
                      {t("Adjust Base Rate")}
                    </TooltipContent>
                  </Tooltip>
                </div>
              )}
            </div>
          )}
        </div>

        {share ? (
          <PayerBillLines
            share={share}
            additionalCharges={additionalCharges}
            lineName={lineName}
            ofChargeTotal={ofChargeTotal}
            chargeActions={chargeActions}
            reassignAction={reassignAction}
            isEditable={isEditable}
            isPending={isPending}
            onAddCharge={() => setChargeDialogOpen(true)}
          />
        ) : (
          <ShipmentChargeLines
            freightCharge={freightCharge}
            otherCharge={otherCharge}
            totalCharge={totalCharge}
            additionalCharges={additionalCharges}
            freightBasis={freightCharge}
            chargeActions={chargeActions}
            isEditable={isEditable}
            isPending={isPending}
            onAddCharge={() => setChargeDialogOpen(true)}
          />
        )}
      </div>

      {share ? (
        <div className="bg-muted/50 flex items-center justify-between rounded-md px-3 py-2.5">
          <span className="text-sm font-semibold">{t("Total")}</span>
          <span className="text-base font-semibold tabular-nums" data-testid="payer-bill-total">
            {formatCurrency(Number(share.totalAmount ?? 0))}
          </span>
        </div>
      ) : (
        <div className="bg-muted/50 flex items-center justify-between rounded-md px-3 py-2.5">
          <span className="text-sm font-semibold">{t("Total")}</span>
          <span className="text-base font-semibold tabular-nums">{formatCurrency(totalCharge)}</span>
        </div>
      )}

      <AlertDialog
        open={staleSplit !== null}
        onOpenChange={(open) => {
          if (!open) setStaleSplit(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("Convert amount splits to percentages?")}</AlertDialogTitle>
            <AlertDialogDescription render={<div />}>
              <ul className="list-disc space-y-1 pl-4 text-left text-sm">
                {staleSplit?.messages.map((message) => (
                  <li key={message}>{message}</li>
                ))}
              </ul>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <Button type="button" variant="outline" onClick={() => setStaleSplit(null)}>
              {t("Cancel")}
            </Button>
            <Button
              type="button"
              disabled={isPending}
              onClick={() => {
                if (!staleSplit) return;
                saveCharges({ ...staleSplit.payload, convertAmountSplitsToPercent: true });
              }}
            >
              {t("Convert to percentages")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

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
          currentTemplateId={shipment.formulaTemplateId ?? undefined}
        />
      )}
      {reassignLine ? (
        <BillingQueueReassignChargeDialog
          open={reassignLine !== null}
          onOpenChange={(open) => {
            if (!open) setReassignLine(null);
          }}
          item={item}
          line={reassignLine}
        />
      ) : null}
    </div>
  );
}

function AccessorialsHeader({
  isEditable,
  isPending,
  onAddCharge,
}: {
  isEditable: boolean;
  isPending: boolean;
  onAddCharge: () => void;
}) {
  const t = useT();
  return (
    <div className="flex items-center justify-between p-2">
      <span className="text-muted-foreground text-xs font-medium">
        {t("Accessorials")}
      </span>
      {isEditable && (
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                size="icon-xs"
                variant="ghost"
                aria-label={t("Add Charge")}
                onClick={onAddCharge}
                disabled={isPending}
              >
                <PlusIcon className="size-3" />
              </Button>
            }
          />
          <TooltipContent side="top" sideOffset={10}>
            {t("Add Charge")}
          </TooltipContent>
        </Tooltip>
      )}
    </div>
  );
}

/** The shipment's whole charge sheet, for items that bill a single payer. */
function ShipmentChargeLines({
  freightCharge,
  otherCharge,
  additionalCharges,
  freightBasis,
  chargeActions,
  isEditable,
  isPending,
  onAddCharge,
}: {
  freightCharge: number;
  otherCharge: number;
  totalCharge: number;
  additionalCharges: AdditionalCharge[];
  freightBasis: number;
  chargeActions: (index: number, charge: AdditionalCharge) => ChargeRowAction[];
  isEditable: boolean;
  isPending: boolean;
  onAddCharge: () => void;
}) {
  const t = useT();
  return (
    <>
      <ChargeRow name={t("Line Haul")} amount={freightCharge} actions={[]} />
      <Separator className="my-1" />
      <AccessorialsHeader isEditable={isEditable} isPending={isPending} onAddCharge={onAddCharge} />
      {additionalCharges.length === 0 && (
        <p className="text-muted-foreground px-2 pb-2 text-xs">{t("No accessorial charges")}</p>
      )}
      {additionalCharges.map((charge, index) => (
        <ChargeRow
          key={charge.id ?? index}
          name={chargeName(charge, t("Charge"))}
          details={
            <span className="text-muted-foreground text-xs">
              {formatChargeBreakdown(charge)}
            </span>
          }
          amount={chargeLineTotal(charge, freightBasis)}
          actions={chargeActions(index, charge)}
        />
      ))}
      {additionalCharges.length > 0 && (
        <div className="text-muted-foreground flex items-center justify-between p-2">
          <span className="text-xs">{t("Subtotal")}</span>
          <span className="text-xs font-medium tabular-nums">{formatCurrency(otherCharge)}</span>
        </div>
      )}
    </>
  );
}

/**
 * The item's own bill: only the charges its payer owes, each at the payer's
 * share, with everything owed by other payers set aside so a reviewer still sees
 * the whole shipment.
 */
function PayerBillLines({
  share,
  additionalCharges,
  lineName,
  ofChargeTotal,
  chargeActions,
  reassignAction,
  isEditable,
  isPending,
  onAddCharge,
}: {
  share: PayerShare;
  additionalCharges: AdditionalCharge[];
  lineName: (line: PayerShareLine) => string;
  ofChargeTotal: (line: PayerShareLine) => string | null;
  chargeActions: (index: number, charge: AdditionalCharge) => ChargeRowAction[];
  reassignAction: (line: PayerShareLine, name: string) => ChargeRowAction[];
  isEditable: boolean;
  isPending: boolean;
  onAddCharge: () => void;
}) {
  const t = useT();
  const freightLine = share.lines.find((line) => line.kind === "Freight");
  const accessorialLines = share.lines.filter((line) => line.kind === "Accessorial");

  const accessorialRow = (line: PayerShareLine, testIdPrefix: string, muted: boolean) => {
    const index = additionalCharges.findIndex((row) => row.id === line.additionalChargeId);
    const charge = index >= 0 ? additionalCharges[index] : undefined;
    const name = lineName(line);
    return (
      <ChargeRow
        key={`${testIdPrefix}-${line.additionalChargeId}`}
        testId={`${testIdPrefix}-${line.additionalChargeId}`}
        name={name}
        muted={muted}
        details={
          <>
            <span className="text-muted-foreground text-xs">
              {muted
                ? describeLinePayers(line)
                : charge
                  ? formatChargeBreakdown(charge)
                  : line.description}
            </span>
          </>
        }
        amount={muted ? line.chargeTotal : line.amount}
        secondary={muted ? null : ofChargeTotal(line)}
        actions={[
          ...reassignAction(line, name),
          ...(charge && !muted ? chargeActions(index, charge) : []),
        ]}
      />
    );
  };

  const otherLines = share.otherPayerLines;

  return (
    <>
      {freightLine ? (
        <ChargeRow
          testId="payer-line-freight"
          name={t("Line Haul")}
          amount={freightLine.amount}
          secondary={ofChargeTotal(freightLine)}
          actions={reassignAction(freightLine, t("Line Haul"))}
        />
      ) : null}
      <Separator className="my-1" />
      <AccessorialsHeader isEditable={isEditable} isPending={isPending} onAddCharge={onAddCharge} />
      {accessorialLines.length === 0 ? (
        <p className="text-muted-foreground px-2 pb-2 text-xs">
          {t("No accessorial charges for this payer")}
        </p>
      ) : (
        accessorialLines.map((line) => accessorialRow(line, "payer-line", false))
      )}
      {accessorialLines.length > 0 && (
        <div className="text-muted-foreground flex items-center justify-between p-2">
          <span className="text-xs">{t("Subtotal")}</span>
          <span className="text-xs font-medium tabular-nums">
            {formatCurrency(Number(share.accessorialAmount ?? 0))}
          </span>
        </div>
      )}
      {otherLines.length > 0 ? (
        <Collapsible className="mt-1">
          <CollapsibleTrigger
            render={
              <Button
                type="button"
                variant="ghost"
                size="xs"
                className="text-muted-foreground group/other w-full justify-between"
              />
            }
          >
            <span>{t("Billed to other payers ({0})", otherLines.length)}</span>
            <ChevronDownIcon className="size-3.5 transition-transform group-data-[panel-open]/other:rotate-180" />
          </CollapsibleTrigger>
          <CollapsibleContent>
            {otherLines.map((line) =>
              line.kind === "Freight" ? (
                <ChargeRow
                  key="other-payer-line-freight"
                  testId="other-payer-line-freight"
                  name={t("Line Haul")}
                  muted
                  details={
                    <span className="text-muted-foreground text-xs">
                      {describeLinePayers(line)}
                    </span>
                  }
                  amount={line.chargeTotal}
                  actions={reassignAction(line, t("Line Haul"))}
                />
              ) : (
                accessorialRow(line, "other-payer-line", true)
              ),
            )}
          </CollapsibleContent>
        </Collapsible>
      ) : null}
    </>
  );
}
