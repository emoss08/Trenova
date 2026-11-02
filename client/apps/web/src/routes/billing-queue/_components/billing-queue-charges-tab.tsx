import { translate } from "@trenova/shared/i18n/runtime";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { describeApiError } from "@/lib/api-error-message";
import { isSuggestion } from "@/lib/billing-queue-checks";
import { shipmentMargin } from "@/lib/billing-queue-margin";
import {
  AccessorialChargeAutocompleteField,
  CustomerAutocompleteField,
} from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import {
  accessorialOptionDefaults,
  describeLinePayers,
  isChargeReassignable,
  staleAmountSplitMessages,
} from "@/lib/billing-queue-charges";
import { billingQueue } from "@/lib/queries/billing-queue";
import { shipment as shipmentQueries } from "@/lib/queries/shipment";
import { apiService } from "@/services/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
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
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useRules } from "@trenova/shared/hooks/use-rules";
import { chargeTotals } from "@trenova/shared/rules/allocation";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type {
  BillingIssue,
  BillingQueueItem,
  BillingQueueUpdateChargesInput,
  PayerShare,
  PayerShareLine,
  ReassignChargeInput,
} from "@trenova/shared/types/billing-queue";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { AdditionalCharge } from "@trenova/shared/types/shipment";
import {
  accessorialChargeMethodSchema,
  type AccessorialChargeMethod,
} from "@trenova/shared/types/accessorial-charge";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { humanizeTariffName } from "@trenova/shared/lib/tariff-inputs";
import { chargeLineBasis } from "@trenova/shared/lib/tariff-lines";
import { formatTariffValue } from "@trenova/shared/lib/tariff-value";
import type { TariffWireValue } from "@trenova/shared/types/tariff";
import {
  AlertTriangleIcon,
  Edit02Icon,
  InfoCircleIcon,
  PlusIcon,
  XCloseIcon,
  Users01Icon,
} from "@trenova/shared/components/icons";
import { AnimatePresence, m, useIsPresent, type Variants } from "motion/react";
import { useRef, useState, type ReactNode } from "react";
import { toast } from "@trenova/shared/components/ui/toast";
import { BillingQueueReassignChargeDialog } from "./billing-queue-reassign-charge-dialog";
import { BillingQueueRerateDialog } from "./billing-queue-rerate-dialog";
import { useAfterReassignment } from "./use-after-reassignment";
import { BillingQueueRatedBy, ratedByName } from "./billing-queue-rated-by";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";

const GRID = "grid grid-cols-[minmax(0,1fr)_4.5rem_7rem_11rem_8rem_1.875rem] items-center gap-2";

/** The most a line bills by hand: up to 999 units at up to $99,999.99 each. */
const MAX_LINE_QUANTITY = 999;
const MAX_LINE_RATE = 99_999.99;

function getChargeWarnings(
  freightCharge: number,
  totalCharge: number,
  additionalCharges: AdditionalCharge[],
): string[] {
  const warnings: string[] = [];
  if (totalCharge === 0) warnings.push(translate("Total charge is $0.00"));
  if (freightCharge === 0 && additionalCharges.length > 0)
    warnings.push(translate("Freight charge is $0.00 but accessorial charges exist"));
  if (additionalCharges.some((charge) => Number(charge.amount ?? 0) < 0)) {
    warnings.push(translate("One or more charges have a negative amount"));
  }
  if (freightCharge < 0) warnings.push(translate("Freight charge is negative"));
  return warnings;
}

function chargeBasis(charge: AdditionalCharge): string {
  // A charge the tariff billed states its own quantity and rate, as the bill wrote them.
  if (charge.isTariff) {
    return chargeLineBasis(charge, translate) ?? translate("Billed by the tariff");
  }
  switch (charge.method) {
    case "PerUnit":
      return translate("Per unit");
    case "Percentage":
      return translate("{0}% of line haul", Number(charge.amount ?? 0));
    default:
      return translate("Flat");
  }
}

function chargeName(charge: AdditionalCharge | undefined, fallback: string): string {
  // A tariff's charge is named by its label, the customer's text as the tariff wrote it.
  if (charge?.isTariff && charge.label) return charge.label;
  return charge?.accessorialCharge?.description ?? charge?.accessorialCharge?.code ?? fallback;
}

function quantityText(value: unknown): string {
  if (value == null || value === "") return "—";
  const n = Number(value);
  return Number.isFinite(n) ? n.toLocaleString() : String(value);
}

type LineRow = {
  key: string;
  testId?: string;
  name: string;
  source: string;
  basis: string;
  qty: string;
  rate: string;
  payer: string;
  amount: number;
  secondary?: string | null;
  /** The shipment's additional charge this line bills, when it bills one. */
  chargeId?: string;
  /** The accessorial this row edits in place, with its index on the shipment. */
  editable?: { index: number; charge: AdditionalCharge };
  reassign?: PayerShareLine;
  muted?: boolean;
};

/**
 * The item's charges as one sheet: each line with its quantity, rate, payer and
 * amount, an accessorial edited in place, receipts the shipment carries but does
 * not bill offered as lines to add, and the totals against the carrier's cost.
 */
export function BillingQueueChargesTab({ item }: { item: BillingQueueItem }) {
  const t = useT();
  const rt = useRichT();
  const rules = useRules();
  const shipment = item.shipment;
  const isEditable = item.status === "InReview";
  const invalidate = useInvalidateBillingQueue();
  const queryClient = useQueryClient();
  const { allowed: canUpdate } = usePermission(Resource.BillingQueue, Operation.Update);
  const canReassign = canUpdate && isChargeReassignable(item.status);

  const [adding, setAdding] = useState(false);
  const [rerateOpen, setRerateOpen] = useState(false);
  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [editingFrom, setEditingFrom] = useState(0);
  // The line whose editor was just canceled, and the height the editor stood at: the
  // line folds back down from there, the reverse of the editor opening.
  const [foldingRow, setFoldingRow] = useState<{ key: string; from: number } | null>(null);
  // The charges the sheet held when a new line was saved. A line outside them has
  // just arrived: it opens onto the sheet in the same render its editor folds away,
  // whichever lands first, the save's state or the cache write that carries the line.
  const [addBaseline, setAddBaseline] = useState<string[] | null>(null);
  const [staleSplit, setStaleSplit] = useState<{
    payload: BillingQueueUpdateChargesInput;
    messages: string[];
  } | null>(null);
  const [reassignLine, setReassignLine] = useState<PayerShareLine | null>(null);

  const { data: profitability } = useQuery({
    ...shipmentQueries.profitability(item.shipmentId ?? ""),
    enabled: Boolean(item.shipmentId),
  });

  const {
    mutate: saveCharges,
    mutateAsync: saveChargesAsync,
    isPending,
  } = useMutation({
    mutationFn: (payload: BillingQueueUpdateChargesInput) =>
      apiService.billingQueueService.updateCharges(item.id, {
        ...payload,
        expandShipmentDetails: true,
      }),
    // The save answers with the item as its read does, so the sheet shows a line
    // added or removed in the same render the editor closes, not after a refetch.
    onSuccess: (saved) => {
      queryClient.setQueryData(billingQueue.get(item.id).queryKey, saved);
      setStaleSplit(null);
      setEditingKey(null);
      // Only one line is open at a time, so a save while a new line is open is that
      // line's, including its retry after converting a stale split.
      setAdding(false);
      invalidate();
    },
    onError: (error, payload) => {
      const messages = staleAmountSplitMessages(error);
      if (messages.length > 0 && !payload.convertAmountSplitsToPercent) {
        setStaleSplit({ payload, messages });
        return;
      }
      setStaleSplit(null);
      setAddBaseline(null);
      toast.error(t("Failed to update charges"), { description: describeApiError(error) });
    },
  });

  const resolve = useApiMutation({
    mutationFn: ({ issueId, optionKey }: { issueId: string; optionKey: string }) =>
      apiService.billingQueueService.resolveIssue(item.id, issueId, optionKey),
    resourceName: "BillingQueueItem",
    onSuccess: (next) => {
      if (next.review) {
        queryClient.setQueryData(billingQueue.get(item.id).queryKey, next);
      }
      invalidate();
    },
  });

  const afterReassignment = useAfterReassignment(item);
  const reassign = useApiMutation({
    mutationFn: ({ input }: { input: ReassignChargeInput; chargeName: string }) =>
      apiService.billingQueueService.reassignCharge(item.id, input),
    resourceName: "billing queue charge",
    onSuccess: (result, { chargeName }) => {
      setEditingKey(null);
      afterReassignment(result, chargeName);
    },
  });

  if (!shipment) {
    return (
      <p className="text-foreground-muted py-12 text-center text-sm">
        {t("Shipment details not available")}
      </p>
    );
  }

  const additionalCharges = shipment.additionalCharges ?? [];
  const freightCharge = Number(shipment.freightChargeAmount ?? 0);
  const totalCharge = Number(shipment.totalChargeAmount ?? 0);
  const warnings = getChargeWarnings(freightCharge, totalCharge, additionalCharges);
  const resolutionError = item.payerShare?.resolutionError ?? null;
  const share: PayerShare | null = item.payerShare && !resolutionError ? item.payerShare : null;
  const payerName = item.billToCustomer?.name ?? shipment.customer?.name ?? "";
  const shipmentPayerId = shipment.billToCustomerId || shipment.customerId || "";
  // A new line is billed to this item's payer when it can be moved there, and
  // otherwise lands with the shipment's payer.
  const newLinePayerId =
    canReassign && share ? (item.billToCustomerId ?? shipmentPayerId) : shipmentPayerId;
  const freightShare = share?.lines.find((line) => line.kind === "Freight") ?? null;
  const busy = isPending || reassign.isPending;
  const price = (line: Omit<EditedLine, "accessorialChargeId">) =>
    Number(chargeTotals(rules, { base: freightCharge, charges: [line] }).totals[0] ?? 0);
  const { totals } = chargeTotals(rules, {
    base: freightCharge,
    charges: additionalCharges.map((charge) => ({
      method: charge.method,
      amount: charge.amount,
      unit: charge.unit,
    })),
  });

  const payload = (charges: AdditionalCharge[]) =>
    charges.map((c) => ({
      id: c.id,
      accessorialChargeId: c.accessorialChargeId,
      method: c.method,
      amount: c.amount ?? 0,
      unit: c.unit ?? 1,
    }));

  // A line edit is saved first, while this payer still bills the charge; moving the
  // line to another payer after it can leave this item with nothing and cancel it.
  const saveLine = async (row: LineRow, edit: LineEdit) => {
    if (edit.line && row.editable) {
      const next = payload(additionalCharges);
      next[row.editable.index] = { ...next[row.editable.index], ...edit.line };
      try {
        await saveChargesAsync({ additionalCharges: next });
      } catch {
        return;
      }
    }
    if (edit.payerId && row.reassign) {
      reassign.mutate({
        chargeName: row.name,
        input: {
          chargeKind: row.reassign.kind === "Freight" ? "Freight" : "Accessorial",
          additionalChargeId: row.reassign.additionalChargeId ?? undefined,
          allocations: [
            { billToCustomerId: edit.payerId, method: "Percent", percent: "100", amount: null },
          ],
        },
      });
    }
  };

  // A charge with no split goes to the shipment's payer. Added from another payer's
  // bill, it is added first and then moved to the payer chosen.
  const addLine = async (edit: LineEdit) => {
    if (!edit.line) return;
    const known = additionalCharges.flatMap((charge) => (charge.id ? [charge.id] : []));
    setAddBaseline(known);
    let saved: BillingQueueItem;
    try {
      saved = await saveChargesAsync({
        additionalCharges: [...payload(additionalCharges), edit.line],
      });
    } catch {
      return;
    }
    const added = saved.shipment?.additionalCharges?.find(
      (charge) => charge.id && !known.includes(charge.id),
    );
    if (edit.payerId && edit.payerId !== shipmentPayerId && added?.id) {
      reassign.mutate({
        chargeName: chargeName(added, t("Charge")),
        input: {
          chargeKind: "Accessorial",
          additionalChargeId: added.id,
          allocations: [
            { billToCustomerId: edit.payerId, method: "Percent", percent: "100", amount: null },
          ],
        },
      });
    }
  };

  const accessorialRow = (
    index: number,
    charge: AdditionalCharge,
    overrides: Partial<LineRow>,
  ) => ({
    key: `charge-${charge.id ?? index}`,
    chargeId: charge.id ?? undefined,
    name: chargeName(charge, t("Charge")),
    source: charge.isTariff ? t("Tariff") : charge.isSystemGenerated ? t("Rule") : t("Added"),
    basis: chargeBasis(charge),
    qty: charge.isTariff ? quantityText(charge.quantity) : quantityText(charge.unit ?? 1),
    rate:
      charge.method === "Percentage"
        ? t("{0}%", Number(charge.amount ?? 0))
        : formatCurrency(Number(charge.isTariff ? (charge.rate ?? 0) : (charge.amount ?? 0))),
    payer: payerName,
    amount: Number(totals[index] ?? 0),
    editable:
      isEditable && !charge.isTariff && !charge.isSystemGenerated ? { index, charge } : undefined,
    ...overrides,
  });

  const rows: LineRow[] = [];
  if (share) {
    const freight = share.lines.find((line) => line.kind === "Freight");
    if (freight) {
      rows.push({
        key: "freight",
        testId: "payer-line-freight",
        name: t("Line haul"),
        source: t("Rated"),
        basis: shipment.formulaTemplate?.name ?? t("Line haul"),
        qty: "—",
        rate: "—",
        payer: payerName,
        amount: Number(freight.amount ?? 0),
        secondary: freight.partial
          ? t("of {0}", formatCurrency(Number(freight.chargeTotal ?? 0)))
          : null,
        reassign: canReassign ? freight : undefined,
      });
    }
    for (const line of share.lines.filter((entry) => entry.kind === "Accessorial")) {
      const index = additionalCharges.findIndex((row) => row.id === line.additionalChargeId);
      const charge = index >= 0 ? additionalCharges[index] : undefined;
      const base: Partial<LineRow> = {
        key: `payer-line-${line.additionalChargeId}`,
        chargeId: line.additionalChargeId ?? undefined,
        testId: `payer-line-${line.additionalChargeId}`,
        amount: Number(line.amount ?? 0),
        secondary: line.partial ? t("of {0}", formatCurrency(Number(line.chargeTotal ?? 0))) : null,
        reassign: canReassign ? line : undefined,
      };
      rows.push(
        charge
          ? accessorialRow(index, charge, base)
          : {
              name: line.description,
              source: t("Added"),
              basis: line.description,
              qty: "—",
              rate: "—",
              payer: payerName,
              amount: 0,
              key: "",
              ...base,
            },
      );
    }
  } else {
    const freightLines = shipment.freightLines ?? [];
    if (freightLines.length === 0) {
      rows.push({
        key: "freight",
        name: t("Line haul"),
        source: t("Rated"),
        basis: shipment.formulaTemplate?.name ?? t("Flat"),
        qty: "1",
        rate: formatCurrency(freightCharge),
        payer: payerName,
        amount: freightCharge,
      });
    }
    freightLines.forEach((line, index) => {
      rows.push({
        key: `freight-${index}`,
        name: line.label,
        source: t("Tariff"),
        basis: chargeLineBasis(line, t) ?? t("Flat"),
        qty: quantityText(line.quantity),
        rate: line.rate != null ? formatCurrency(Number(line.rate)) : "—",
        payer: payerName,
        amount: Number(line.amount ?? 0),
      });
    });
    additionalCharges.forEach((charge, index) => rows.push(accessorialRow(index, charge, {})));
  }

  const isArriving = (row: LineRow) =>
    addBaseline !== null && row.chargeId != null && !addBaseline.includes(row.chargeId);
  const showAddEditor = adding && !rows.some(isArriving);
  const unbilled = (item.review?.issues ?? []).filter(
    (issue) => isSuggestion(issue) && !issue.resolvedAt,
  );
  const accessorialTotal = share
    ? Number(share.accessorialAmount ?? 0)
    : Number(shipment.otherChargeAmount ?? 0);
  const linehaulTotal = share ? Number(share.freightAmount ?? 0) : freightCharge;
  const billTotal = share ? Number(share.totalAmount ?? 0) : totalCharge;
  const cost = profitability?.estimatedCost != null ? Number(profitability.estimatedCost) : null;
  const margin = shipmentMargin(profitability, t);

  return (
    <section aria-label={t("Charges")} className="flex flex-col gap-2 text-sm">
      <div className="flex flex-wrap items-center gap-2">
        <h3 className="text-foreground font-semibold">{t("Charges")}</h3>
        <span className="flex-1" />
        {item.ratedBy ? (
          <span className="text-foreground-muted text-xs">
            {rt(
              "Rated by <name>{0}</name>",
              { name: () => <BillingQueueRatedBy item={item} ratedBy={item.ratedBy!} /> },
              ratedByName(item.ratedBy, t),
            )}
          </span>
        ) : null}
        <TariffTerms inputs={shipment.tariffInputs ?? undefined} />
        {isEditable ? (
          <Button
            size="xs"
            variant="outline"
            disabled={isPending}
            onClick={() => setRerateOpen(true)}
          >
            {t("Re-rate")}
          </Button>
        ) : null}
      </div>

      {warnings.length > 0 || resolutionError ? (
        <Alert size="sm" variant="warning">
          <AlertTriangleIcon />
          <AlertDescription>
            <ul className="list-disc space-y-0.5 pl-4">
              {[...warnings, ...(resolutionError ? [resolutionError] : [])].map((w) => (
                <li key={w}>{w}</li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      ) : null}
      {share && share.payers.length > 1 && (isEditable || canReassign) ? (
        <Alert size="sm" variant="info">
          <InfoCircleIcon />
          <AlertDescription>
            {t(
              "Charges belong to the shipment, so changing one here also changes the other payers' bills.",
            )}
          </AlertDescription>
        </Alert>
      ) : null}

      <div role="table" aria-label={t("Charge lines")} className="border-border rounded-md border">
        <div
          role="row"
          className={cn(
            GRID,
            "border-border text-foreground-subtle h-8 border-b px-2.5 text-xs font-medium",
          )}
        >
          <span role="columnheader">{t("Charge")}</span>
          <span role="columnheader" className="text-right">
            {t("Qty")}
          </span>
          <span role="columnheader" className="text-right">
            {t("Rate")}
          </span>
          <span role="columnheader">{t("Payer")}</span>
          <span role="columnheader" className="text-right">
            {t("Amount")}
          </span>
          <span />
        </div>

        <AnimatePresence initial={false}>
          {rows.map((row) =>
            editingKey === row.key && row.editable ? (
              <LineEditor
                key={row.key}
                charge={row.editable.charge}
                unfoldFrom={editingFrom}
                payerId={item.billToCustomerId ?? ""}
                payerName={row.payer}
                canChangePayer={Boolean(row.reassign)}
                busy={busy}
                price={price}
                onSplit={row.reassign ? () => setReassignLine(row.reassign!) : undefined}
                onCancel={(height) => {
                  setFoldingRow({ key: row.key, from: height });
                  setEditingKey(null);
                }}
                onRemove={() => {
                  const next = [...additionalCharges];
                  next.splice(row.editable!.index, 1);
                  saveCharges({ additionalCharges: payload(next) });
                }}
                onSave={(edit) => void saveLine(row, edit)}
              />
            ) : (
              <ViewRow
                key={row.key}
                row={row}
                busy={isPending}
                entrance={
                  isArriving(row)
                    ? { kind: "arrive" }
                    : foldingRow?.key === row.key
                      ? { kind: "fold", from: foldingRow.from }
                      : undefined
                }
                onEntered={() => (isArriving(row) ? setAddBaseline(null) : setFoldingRow(null))}
                onEdit={
                  row.editable
                    ? (height) => {
                        setAdding(false);
                        setEditingFrom(height);
                        setFoldingRow(null);
                        setEditingKey(row.key);
                      }
                    : undefined
                }
                onReassign={row.reassign ? () => setReassignLine(row.reassign!) : undefined}
              />
            ),
          )}
        </AnimatePresence>

        {unbilled.map((issue) => (
          <UnbilledRow
            key={issue.id}
            issue={issue}
            busy={resolve.isPending}
            canAct={canUpdate && isEditable}
            onResolve={(optionKey) => resolve.mutate({ issueId: issue.id, optionKey })}
          />
        ))}

        <AnimatePresence initial={false}>
          {showAddEditor ? (
            <LineEditor
              key="new-line"
              payerId={newLinePayerId}
              payerName={payerName}
              canChangePayer={canReassign && share !== null}
              busy={busy}
              price={price}
              onCancel={() => setAdding(false)}
              onSave={(edit) => void addLine(edit)}
            />
          ) : null}
        </AnimatePresence>

        <div className="flex flex-wrap items-start gap-4 p-2.5">
          <div className="flex flex-1 gap-1.5">
            {isEditable ? (
              <Button
                size="xs"
                variant="ghost"
                disabled={busy || adding}
                onClick={() => {
                  setEditingKey(null);
                  setAdding(true);
                }}
              >
                <PlusIcon className="size-3.5" />
                {t("Add charge")}
              </Button>
            ) : null}
            {freightShare && canReassign ? (
              <Button
                size="xs"
                variant="ghost"
                className="text-foreground-muted"
                disabled={busy}
                onClick={() => setReassignLine(freightShare)}
              >
                {t("Split between payers")}
              </Button>
            ) : null}
          </div>
          <dl className="grid min-w-60 grid-cols-[auto_9rem] gap-y-1 [&>dd]:pl-4">
            <dt className="text-foreground-muted">{t("Line haul")}</dt>
            <dd className="text-right font-mono">{formatCurrency(linehaulTotal)}</dd>
            <dt className="text-foreground-muted">{t("Accessorials")}</dt>
            <dd className="text-right font-mono">{formatCurrency(accessorialTotal)}</dd>
            <dt className="border-border border-t pt-1.5 font-semibold">{t("Total")}</dt>
            <dd
              className="border-border border-t pt-1.5 text-right font-mono font-semibold"
              data-testid={share ? "payer-bill-total" : undefined}
            >
              {formatCurrency(billTotal)}
            </dd>
            {cost != null ? (
              <>
                <dt className="text-foreground-muted">{t("Carrier cost")}</dt>
                <dd className="text-foreground-muted text-right font-mono">
                  {formatCurrency(cost)}
                </dd>
              </>
            ) : null}
            {margin ? (
              <>
                <dt className="text-foreground-muted">{t("Margin")}</dt>
                <dd
                  className={cn(
                    "text-right font-mono whitespace-nowrap",
                    margin.thin ? "text-warning-foreground" : "text-success-foreground",
                  )}
                >
                  {margin.text}
                </dd>
              </>
            ) : null}
          </dl>
        </div>
      </div>

      {share && share.otherPayerLines.length > 0 ? (
        <OtherPayerLines
          lines={share.otherPayerLines}
          additionalCharges={additionalCharges}
          canReassign={canReassign}
          onReassign={setReassignLine}
        />
      ) : null}

      <AlertDialog
        open={staleSplit !== null}
        onOpenChange={(open) => {
          if (open) return;
          setStaleSplit(null);
          setAddBaseline(null);
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
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setStaleSplit(null);
                setAddBaseline(null);
              }}
            >
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

      {rerateOpen && (
        <BillingQueueRerateDialog
          open={rerateOpen}
          onOpenChange={setRerateOpen}
          itemId={item.id}
          currentTemplateId={shipment.formulaTemplateId ?? undefined}
          currentInputs={shipment.tariffInputs ?? undefined}
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
    </section>
  );
}

function Chip({
  children,
  tone = "neutral",
}: {
  children: ReactNode;
  tone?: "neutral" | "warning";
}) {
  return (
    <span
      className={cn(
        "rounded-xs px-1.25 text-2xs font-medium",
        tone === "warning"
          ? "bg-warning-subtle text-warning-foreground"
          : "bg-muted text-foreground-muted",
      )}
    >
      {children}
    </span>
  );
}

/** How a line folds shut or opens on the sheet; arriving and leaving rows share it. */
const LINE_FOLD = { duration: 0.22, ease: [0.4, 0, 0.2, 1] } as const;
/** How an editor opens out of its line, and folds back into it on cancel. */
const EDITOR_SPRING = { type: "spring", bounce: 0, visualDuration: 0.2 } as const;

/**
 * How a line comes onto the sheet: a line just added opens from nothing; a line whose
 * editor was canceled folds down from the editor's height. Either way its content fades
 * in once there is room for it.
 */
type RowEntrance = { kind: "arrive" } | { kind: "fold"; from: number };

const rowMotion: Variants = {
  hidden: { height: 0 },
  folded: (entrance: RowEntrance) => ({ height: entrance.kind === "fold" ? entrance.from : 0 }),
  shown: (entrance: RowEntrance | undefined) => ({
    height: "auto",
    transition: { height: entrance?.kind === "fold" ? EDITOR_SPRING : LINE_FOLD },
  }),
};
const rowContentMotion: Variants = {
  hidden: { opacity: 0 },
  folded: { opacity: 0 },
  shown: { opacity: 1, transition: { duration: 0.16, delay: 0.06, ease: "easeIn" } },
};
/** The editor's tint, left behind as it folds back into its line and fading out. */
const rowTintMotion: Variants = {
  hidden: { opacity: 0 },
  folded: { opacity: 1 },
  shown: { opacity: 0, transition: { duration: 0.2, ease: "easeOut" } },
};

function ViewRow({
  row,
  busy,
  entrance,
  onEntered,
  onEdit,
  onReassign,
}: {
  row: LineRow;
  busy: boolean;
  /** How the line comes onto the sheet when it is not simply there already. */
  entrance?: RowEntrance;
  onEntered?: () => void;
  /** Opens the line's editor, told the height the line stands at so it unfolds from there. */
  onEdit?: (height: number) => void;
  onReassign?: () => void;
}) {
  const t = useT();
  const rowRef = useRef<HTMLDivElement>(null);

  return (
    <m.div
      ref={rowRef}
      role="row"
      data-testid={row.testId}
      className={cn("relative", entrance && "overflow-hidden")}
      custom={entrance}
      variants={rowMotion}
      initial={entrance === undefined ? false : entrance.kind === "arrive" ? "hidden" : "folded"}
      animate="shown"
      onAnimationComplete={(definition) => {
        if (entrance && definition === "shown") onEntered?.();
      }}
    >
      <m.span
        aria-hidden
        variants={rowTintMotion}
        className="bg-muted pointer-events-none absolute inset-0"
      />
      {/* Padding and border sit inside the part that opens, so it changes one height. */}
      <m.div
        variants={rowContentMotion}
        className={cn(
          GRID,
          "group border-border hover:bg-surface-hover relative min-h-10 border-b px-2.5 py-1",
        )}
      >
        <span className="flex min-w-0 flex-col gap-px">
          <span className="flex items-center gap-1.5">
            <span className={cn("font-medium", row.muted && "text-foreground-muted")}>
              {row.name}
            </span>
            <Chip>{row.source}</Chip>
          </span>
          <span className="text-foreground-muted truncate text-xs">{row.basis}</span>
        </span>
        <span className="text-right font-mono">{row.qty}</span>
        <span className="text-right font-mono">{row.rate}</span>
        <span className="text-foreground-muted truncate">{row.payer}</span>
        <span className="flex min-w-0 flex-col items-end">
          <span
            className="max-w-full truncate font-mono font-medium"
            title={formatCurrency(row.amount)}
          >
            {formatCurrency(row.amount)}
          </span>
          {row.secondary ? (
            <span className="text-foreground-muted text-xs">{row.secondary}</span>
          ) : null}
        </span>
        <span className="flex justify-end opacity-0 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100">
          {onEdit ? (
            <Button
              size="icon-xs"
              variant="ghost"
              aria-label={t("Edit {0}", row.name)}
              disabled={busy}
              onClick={() => onEdit(rowRef.current?.offsetHeight ?? 0)}
            >
              <Edit02Icon className="size-3" />
            </Button>
          ) : onReassign ? (
            <Button
              size="icon-xs"
              variant="ghost"
              aria-label={t("Change payer for {0}", row.name)}
              disabled={busy}
              onClick={onReassign}
            >
              <Users01Icon className="size-3" />
            </Button>
          ) : null}
        </span>
      </m.div>
    </m.div>
  );
}

const editLineSchema = z.object({
  accessorialChargeId: z.string().min(1, { error: () => translate("Required") }),
  billToCustomerId: z.string().min(1, { error: () => translate("Required") }),
  method: accessorialChargeMethodSchema,
  unit: z.string().refine((value) => /^\d+$/u.test(value.trim()) && Number(value) >= 1, {
    error: () => translate("Must be at least 1"),
  }),
  amount: z
    .string()
    .refine(
      (value) => value.trim() !== "" && Number.isFinite(Number(value)) && Number(value) >= 0,
      {
        error: () => translate("Must be positive"),
      },
    ),
});
type EditLineValues = z.infer<typeof editLineSchema>;

export type EditedLine = {
  accessorialChargeId: string;
  method: AccessorialChargeMethod;
  unit: number;
  amount: number;
};

/**
 * An editor opening: the row unfolds from the height the line stood at, on a spring
 * with no bounce so it eases out evenly, while the fields fade up into place left to
 * right and the footer last, all on one stagger. A removed line folds shut and fades.
 */
const editorMotion: Variants = {
  closed: (from: number) => ({ height: from }),
  open: {
    height: "auto",
    transition: {
      height: EDITOR_SPRING,
      staggerChildren: 0.015,
    },
  },
  removed: {
    height: 0,
    opacity: 0,
    transition: {
      height: LINE_FOLD,
      opacity: { duration: 0.16, ease: "easeOut" },
    },
  },
};
const editorPartMotion: Variants = {
  closed: { opacity: 0, y: 4 },
  open: { opacity: 1, y: 0, transition: { duration: 0.16, ease: [0.33, 1, 0.68, 1] } },
};

/** What saving a line changes: the line itself, who pays it, or both. */
type LineEdit = {
  line: EditedLine | null;
  payerId: string | null;
};

/**
 * A charge line edited in place, or a new one being added: which accessorial it
 * bills, its quantity and rate, who pays it, and what it does to the bill before it
 * is saved. Without `charge` the row adds a line.
 */
function LineEditor({
  charge,
  unfoldFrom = 0,
  payerId,
  payerName,
  canChangePayer,
  busy,
  price,
  onSplit,
  onCancel,
  onRemove,
  onSave,
}: {
  /** The line being edited; absent when adding one. */
  charge?: AdditionalCharge;
  /** The height, in pixels, the row unfolds from: the line it replaces, or nothing. */
  unfoldFrom?: number;
  payerId: string;
  payerName: string;
  /** Whether the line can move to another payer from here. */
  canChangePayer: boolean;
  busy: boolean;
  /** What a charge billed this way comes to, priced as the shipment prices it. */
  price: (line: Omit<EditedLine, "accessorialChargeId">) => number;
  /** Opens the split between payers for a saved line. */
  onSplit?: () => void;
  /** Closes the editor, told the height it stands at so its line folds down from there. */
  onCancel: (height: number) => void;
  /** Removes a saved line. */
  onRemove?: () => void;
  onSave: (edit: LineEdit) => void;
}) {
  const t = useT();
  const adding = charge === undefined;
  const [pickedPayer, setPickedPayer] = useState(payerName);
  // The row clips its fields while it unfolds, then lets focus rings and menus out.
  const [unfolded, setUnfolded] = useState(false);
  const isPresent = useIsPresent();
  const formRef = useRef<HTMLFormElement>(null);
  const cancel = () => onCancel(formRef.current?.offsetHeight ?? 0);
  const saved: EditLineValues = {
    accessorialChargeId: charge?.accessorialChargeId ?? "",
    billToCustomerId: payerId,
    method: charge?.method ?? "Flat",
    unit: String(charge?.unit ?? 1),
    amount: charge ? String(charge.amount ?? 0) : "",
  };
  const form = useForm<EditLineValues>({
    resolver: zodResolver(editLineSchema),
    defaultValues: saved,
  });
  const draft = useWatch({ control: form.control });
  const percentage = draft.method === "Percentage";
  const priced = (values: Partial<EditLineValues>) =>
    price({
      method: values.method ?? saved.method,
      unit: Math.max(1, Math.round(Number(values.unit) || 1)),
      amount: Number(values.amount) || 0,
    });
  const before = adding ? 0 : priced(saved);
  const after = priced(draft);
  const delta = after - before;
  const lineChanged =
    adding ||
    draft.accessorialChargeId !== saved.accessorialChargeId ||
    draft.method !== saved.method ||
    Number(draft.unit) !== Number(saved.unit) ||
    Number(draft.amount) !== Number(saved.amount);
  const payerChanged = draft.billToCustomerId !== saved.billToCustomerId;
  const changed = lineChanged || payerChanged;

  const submit = form.handleSubmit((values) =>
    onSave({
      line: lineChanged
        ? {
            accessorialChargeId: values.accessorialChargeId,
            method: values.method,
            unit: Number(values.unit),
            amount: Number(values.amount),
          }
        : null,
      payerId: adding || payerChanged ? values.billToCustomerId : null,
    }),
  );

  const message = adding
    ? draft.accessorialChargeId
      ? t("Adds {0} to the bill when you add it", formatCurrency(after))
      : t("Choose the accessorial this line bills")
    : !changed
      ? t("No change yet")
      : !lineChanged
        ? t("Moves the line to {0} when you save", pickedPayer)
        : Math.abs(delta) < 0.005
          ? t("Same amount; the line changes when you save")
          : delta > 0
            ? t(
                "+{0} against the saved line; the invoice total updates when you save",
                formatCurrency(delta),
              )
            : t(
                "−{0} against the saved line; the invoice total updates when you save",
                formatCurrency(-delta),
              );

  return (
    <m.form
      ref={formRef}
      role="row"
      aria-label={adding ? t("New charge line") : undefined}
      className={cn((!unfolded || !isPresent) && "overflow-hidden")}
      custom={unfoldFrom}
      variants={editorMotion}
      initial="closed"
      animate="open"
      exit="removed"
      onAnimationComplete={(definition) => {
        if (definition === "open") setUnfolded(true);
      }}
      onSubmit={(event) => void submit(event)}
    >
      {/* The row's padding and border sit inside the part that animates, so opening
          and removing change one height and the fields hold still while it clips. */}
      <div className="border-border bg-muted flex flex-col gap-1 border-b px-2.5 pt-2 pb-2.5 not-dark:[--field:var(--background)]">
        <div className={GRID}>
          <m.div variants={editorPartMotion} className="min-w-0">
            <AccessorialChargeAutocompleteField
              control={form.control}
              name="accessorialChargeId"
              placeholder={t("Choose an accessorial…")}
              triggerClassName="h-7 text-xs"
              onOptionChange={(option) => {
                if (!option) return;
                const { method, amount } = accessorialOptionDefaults(option);
                if (method) form.setValue("method", method, { shouldDirty: true });
                if (amount != null) form.setValue("amount", String(amount), { shouldDirty: true });
              }}
            />
          </m.div>
          <m.div variants={editorPartMotion} className="min-w-0">
            <NumberField
              control={form.control}
              name="unit"
              valueType="string"
              aria-label={t("Quantity")}
              placeholder="1"
              max={MAX_LINE_QUANTITY}
              stepper={false}
              disabled={percentage}
              inputClassName="text-right font-mono"
            />
          </m.div>
          <m.div variants={editorPartMotion} className="min-w-0">
            <NumberField
              control={form.control}
              name="amount"
              valueType="string"
              aria-label={percentage ? t("Percent of line haul") : t("Rate")}
              placeholder="0.00"
              decimalScale={4}
              max={percentage ? 100 : MAX_LINE_RATE}
              stepper={false}
              inputClassName="text-right font-mono"
            />
          </m.div>
          <m.div variants={editorPartMotion} className="min-w-0">
            <CustomerAutocompleteField
              control={form.control}
              name="billToCustomerId"
              placeholder={t("Choose a payer")}
              triggerClassName="h-7 text-xs"
              disabled={!canChangePayer || busy}
              onOptionChange={(option) => {
                if (option) setPickedPayer(option.label);
              }}
            />
          </m.div>
          <m.span
            variants={editorPartMotion}
            className={cn(
              "truncate text-right font-mono font-medium",
              adding && after === 0 && "text-foreground-subtle font-normal",
            )}
            title={formatCurrency(after)}
          >
            {formatCurrency(after)}
          </m.span>
          {adding ? (
            <m.span variants={editorPartMotion} className="flex justify-end">
              <Button
                type="button"
                size="icon-xs"
                variant="ghost"
                aria-label={t("Discard new line")}
                disabled={busy}
                onClick={cancel}
              >
                <XCloseIcon className="size-3" />
              </Button>
            </m.span>
          ) : (
            <span />
          )}
        </div>
        <m.div variants={editorPartMotion} className="flex items-center gap-1.5">
          <span className="text-foreground-muted min-w-0 flex-1 text-xs">{message}</span>
          {onSplit ? (
            <Button type="button" size="xs" variant="ghost" disabled={busy} onClick={onSplit}>
              {t("Split between payers")}
            </Button>
          ) : null}
          {onRemove ? (
            <Button
              type="button"
              size="xs"
              variant="ghost"
              className="text-destructive"
              disabled={busy}
              onClick={onRemove}
            >
              {t("Remove line")}
            </Button>
          ) : null}
          {adding ? null : (
            <Button type="button" size="xs" variant="outline" disabled={busy} onClick={cancel}>
              {t("Cancel")}
            </Button>
          )}
          <Button type="submit" size="xs" disabled={busy || !changed} isLoading={busy}>
            {adding ? t("Add line") : t("Save line")}
          </Button>
        </m.div>
      </div>
    </m.form>
  );
}

const unbilledTags = {
  b: (chunk: ReactNode) => <span className="font-medium">{chunk}</span>,
  m: (chunk: ReactNode) => <span className="font-mono">{chunk}</span>,
};

/**
 * A charge the shipment carries proof of but does not bill, offered as the line that
 * would: marked with Nova when Nova read it off the paperwork, and with a rule's mark
 * when a person entered it.
 */
function UnbilledRow({
  issue,
  busy,
  canAct,
  onResolve,
}: {
  issue: BillingIssue;
  busy: boolean;
  canAct: boolean;
  onResolve: (optionKey: string) => void;
}) {
  const t = useT();
  const rt = useRichT();
  const add = issue.options.find((option) => option.effect.kind === "add");
  const skip = issue.options.find((option) => option.effect.kind === "accept");

  return (
    <div className="border-border bg-muted/50 grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2.5 border-b px-2.5 py-2">
      <span className="flex min-w-0 items-start gap-2">
        {issue.source === "Agent" ? (
          <NovaBadge size="xs" className="mt-0.5" label={t("Nova found this")} />
        ) : (
          <AssistMark
            role="img"
            aria-hidden={false}
            aria-label={t("A rule found this")}
            className="text-foreground-muted mt-0.5 size-3 shrink-0"
          />
        )}
        <span className="flex min-w-0 flex-col gap-px">
          <span>
            {add?.effect.label && add.amount != null
              ? rt(
                  "<b>{0}</b> <m>{1}</m> isn't billed",
                  unbilledTags,
                  add.effect.label,
                  formatCurrency(Number(add.amount)),
                )
              : issue.summary}
          </span>
          {issue.reasoning ? (
            <span className="text-foreground-muted text-xs">{issue.reasoning}</span>
          ) : null}
        </span>
      </span>
      {canAct ? (
        <span className="flex gap-1.5">
          {skip ? (
            <Button
              size="xs"
              variant="ghost"
              className="text-foreground-muted"
              disabled={busy}
              onClick={() => onResolve(skip.key)}
            >
              {t("Not billable")}
            </Button>
          ) : null}
          {add ? (
            <Button size="xs" variant="outline" disabled={busy} onClick={() => onResolve(add.key)}>
              {add.amount != null
                ? t("Add {0}", formatCurrency(Number(add.amount)))
                : t("Add charge")}
            </Button>
          ) : null}
        </span>
      ) : null}
    </div>
  );
}

/** Lines the shipment bills to its other payers, so a reviewer still sees the whole load. */
function OtherPayerLines({
  lines,
  additionalCharges,
  canReassign,
  onReassign,
}: {
  lines: PayerShareLine[];
  additionalCharges: AdditionalCharge[];
  canReassign: boolean;
  onReassign: (line: PayerShareLine) => void;
}) {
  const t = useT();

  return (
    <details className="group text-xs">
      <summary className="text-foreground-muted hover:text-foreground cursor-pointer py-1">
        {t("Billed to other payers ({0})", lines.length)}
      </summary>
      <ul className="border-border mt-1 rounded-md border">
        {lines.map((line) => {
          const charge = additionalCharges.find((row) => row.id === line.additionalChargeId);
          const name =
            line.kind === "Freight" ? t("Line haul") : chargeName(charge, line.description);
          const testId =
            line.kind === "Freight"
              ? "other-payer-line-freight"
              : `other-payer-line-${line.additionalChargeId}`;
          return (
            <li
              key={testId}
              data-testid={testId}
              className="border-border flex items-center gap-2 border-b px-2.5 py-1.5 last:border-b-0"
            >
              <span className="flex min-w-0 flex-1 flex-col">
                <span className="text-foreground-muted">{name}</span>
                <span className="text-foreground-subtle">{describeLinePayers(line)}</span>
              </span>
              <span className="text-foreground-muted font-mono">
                {formatCurrency(Number(line.chargeTotal ?? 0))}
              </span>
              {canReassign ? (
                <Button
                  size="icon-xs"
                  variant="ghost"
                  aria-label={t("Change payer for {0}", name)}
                  onClick={() => onReassign(line)}
                >
                  <Users01Icon className="size-3" />
                </Button>
              ) : null}
            </li>
          );
        })}
      </ul>
    </details>
  );
}

/** The values the tariff rated the shipment with; re-rating changes them. */
function TariffTerms({ inputs }: { inputs: Record<string, unknown> | undefined }) {
  const t = useT();
  const entries = Object.entries(inputs ?? {}).slice(0, 3);
  if (entries.length === 0) return null;

  return (
    <ul className="text-foreground-muted flex items-center gap-2.5 text-xs" aria-label={t("Terms")}>
      {entries.map(([name, value]) => (
        <li key={name}>
          <span>{humanizeTariffName(name)}</span>{" "}
          <span className="text-foreground font-mono">
            {formatTariffValue(value as TariffWireValue, t)}
          </span>
        </li>
      ))}
    </ul>
  );
}
