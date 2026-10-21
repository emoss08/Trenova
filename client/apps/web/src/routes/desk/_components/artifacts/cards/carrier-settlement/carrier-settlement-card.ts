import {
  carrierCostEventTypeChoices,
  carrierPaymentMethodChoices,
  choiceLabel,
} from "@/lib/choices";
import type {
  DeskCarrierMoveInvoice,
  DeskCarrierSettlement,
  DeskCarrierSettlementGroup,
} from "@/lib/graphql/settlement-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { StatementGroup } from "../../record-card/card-settlement";
import { cardDay, cardWeekday } from "../../record-card/record-card-values";
import { minor, periodLabel, signedMinor } from "../../record-card/settlement-values";

/** The carrier the settlement pays, by name. */
export function carrierName(record: DeskCarrierSettlement, t: TranslateFn): string {
  return record.carrier?.name.trim() || t("Unknown carrier");
}

/** What heads the card: the number, the period, and the carrier's terms. */
export function settlementKicker(record: DeskCarrierSettlement, t: TranslateFn): string[] {
  const kicker = [record.settlementNumber, periodLabel(record.periodStart, record.periodEnd, t)];
  const termDays = record.carrier?.paymentTermDays;
  if (termDays != null && termDays > 0) {
    kicker.push(t("Net {0}", termDays));
  }
  return kicker;
}

/** Under the carrier's name: how many loads it pays and the carrier's MC number. */
export function settlementSubtitle(record: DeskCarrierSettlement, t: TranslateFn): string {
  return [
    record.shipmentCount > 0
      ? t("{0, plural, one {Payable for # load} other {Payable for # loads}}", record.shipmentCount)
      : t("No loads this period"),
    record.carrier?.mcNumber ? t("MC {0}", record.carrier.mcNumber) : "",
  ]
    .filter((part) => part !== "")
    .join(" · ");
}

/** The figure beside the name: the net payable, and what it means where the settlement stands. */
export function settlementFigure(
  record: DeskCarrierSettlement,
  t: TranslateFn,
): { amount: string; caption: string; struck: boolean } {
  const amount = minor(record.netPayableMinor, record.currencyCode);
  switch (record.status) {
    case "Voided":
      return { amount, caption: t("net payable · voided"), struck: true };
    case "Paid":
      return {
        amount,
        caption: t("net payable · paid {0}", cardDay(record.paidAt)),
        struck: false,
      };
    default:
      return {
        amount,
        caption: t("net payable · pays {0}", cardWeekday(record.payDate)),
        struck: false,
      };
  }
}

/** How a move's invoice stands against what the move costs here. */
export type MoveStanding = "matched" | "off" | "waiting" | "none";

/** A move's invoice read as the card reads it: matched, off the buy rate, waiting to be accepted, or not in. */
export function moveStanding(move: DeskCarrierMoveInvoice): MoveStanding {
  switch (move.match?.status) {
    case "Matched":
    case "Resolved":
      return "matched";
    case "Variance":
      return "off";
    case "Suggested":
      return "waiting";
    default:
      return "none";
  }
}

/** How many moves stand each way, for the strip's keys and the head's facts. */
export function moveTally(record: DeskCarrierSettlement): Record<MoveStanding, number> {
  const tally: Record<MoveStanding, number> = { matched: 0, off: 0, waiting: 0, none: 0 };
  for (const move of record.moveInvoices ?? []) {
    tally[moveStanding(move)] += 1;
  }
  return tally;
}

/** The facts beside the status pill: the invoices that need someone (off the buy rate, waiting, not in), and how a paid one went out. */
export function headFacts(record: DeskCarrierSettlement, t: TranslateFn): string[] {
  const facts: string[] = [];
  if (record.status !== "Voided" && record.status !== "Paid") {
    const tally = moveTally(record);
    if (tally.off > 0) {
      facts.push(
        t(
          "{0, plural, one {# invoice off the buy rate} other {# invoices off the buy rate}}",
          tally.off,
        ),
      );
    }
    if (tally.waiting > 0) {
      facts.push(
        t(
          "{0, plural, one {# invoice waiting to be accepted} other {# invoices waiting to be accepted}}",
          tally.waiting,
        ),
      );
    }
    if (tally.none > 0) {
      facts.push(
        t("{0, plural, one {# load not invoiced} other {# loads not invoiced}}", tally.none),
      );
    }
  }
  if (record.status === "Paid" && record.paymentMethod.trim()) {
    facts.push(
      record.paymentReference.trim()
        ? t("{0} · {1}", paymentMethodLabel(record.paymentMethod), record.paymentReference.trim())
        : paymentMethodLabel(record.paymentMethod),
    );
  }
  return facts;
}

function paymentMethodLabel(method: string): string {
  return choiceLabel(carrierPaymentMethodChoices, method as never);
}

/** One move that needs someone: its invoice off the buy rate, waiting to be accepted, or not in. */
export type MoveRow = {
  moveId: string;
  standing: Exclude<MoveStanding, "matched">;
  title: string;
  pro: string;
  line: string;
  variance: string;
  status: string;
};

/** A variance as the card writes it: what the invoice is over or under the buy rate by. */
function varianceLabel(variance: number, currency: string, t: TranslateFn): string {
  if (variance === 0) return "";
  return variance > 0
    ? t("+{0}", minor(variance, currency))
    : t("−{0}", minor(-variance, currency));
}

/**
 * The moves that need someone, off the buy rate first, then those waiting to
 * be accepted, then those with no invoice in yet. A matched move is counted,
 * not listed.
 */
export function moveRows(record: DeskCarrierSettlement, t: TranslateFn): MoveRow[] {
  const order: Record<Exclude<MoveStanding, "matched">, number> = { off: 0, waiting: 1, none: 2 };
  const rows: MoveRow[] = [];
  for (const move of record.moveInvoices ?? []) {
    const standing = moveStanding(move);
    if (standing === "matched") continue;
    const match = move.match;
    const currency = match?.currencyCode || record.currencyCode;
    rows.push({
      moveId: move.moveId,
      standing,
      title: match?.invoiceNumber ? t("Invoice {0}", match.invoiceNumber) : t("No invoice yet"),
      pro: move.proNumber,
      line: match
        ? t(
            "Billed {0} · buy rate {1}",
            minor(match.invoiceTotalMinor, currency),
            minor(match.expectedTotalMinor, currency),
          )
        : t("Costs {0} on this statement", minor(move.costMinor, record.currencyCode)),
      variance: match ? varianceLabel(match.varianceMinor, currency, t) : "",
      status:
        standing === "off"
          ? match && match.varianceMinor < 0
            ? t("Under")
            : t("Over")
          : standing === "waiting"
            ? t("Waiting")
            : t("Not invoiced"),
    });
  }
  return rows.sort((a, b) => order[a.standing] - order[b.standing]);
}

/** What the invoices off the buy rate come to together, over or under it. */
export function offBy(record: DeskCarrierSettlement, t: TranslateFn): string {
  let total = 0;
  let currency = record.currencyCode;
  for (const move of record.moveInvoices ?? []) {
    if (moveStanding(move) === "off" && move.match) {
      total += move.match.varianceMinor;
      currency = move.match.currencyCode || currency;
    }
  }
  return varianceLabel(total, currency, t);
}

function costName(group: DeskCarrierSettlementGroup, t: TranslateFn): string {
  switch (group.eventType) {
    case "LinehaulCost":
      return t("Linehaul");
    case "FuelSurcharge":
      return choiceLabel(carrierCostEventTypeChoices, group.eventType);
    default:
      return group.description.trim() || choiceLabel(carrierCostEventTypeChoices, group.eventType);
  }
}

function costBasis(group: DeskCarrierSettlementGroup, t: TranslateFn): string {
  if (group.eventType === "Adjustment") {
    return group.manual ? t("Flat") : t("Invoice variance accepted");
  }
  return t("As rate confirmed");
}

function costDetail(group: DeskCarrierSettlementGroup, t: TranslateFn): string {
  if (group.eventType === "LinehaulCost" && group.description.trim()) {
    return group.shipmentCount > 0
      ? t(
          "{0} · {1, plural, one {# load} other {# loads}}",
          group.description.trim(),
          group.shipmentCount,
        )
      : group.description.trim();
  }
  if (group.eventType === "Adjustment" && group.manual) {
    return group.description.trim() ? "" : t("Added by hand");
  }
  return group.shipmentCount > 0
    ? t("{0, plural, one {# load} other {# loads}}", group.shipmentCount)
    : "";
}

/**
 * The costs as the card reads them: the load costs (linehaul, fuel and
 * accessorials, one row per kind and description) over the adjustments, each
 * group with its total.
 */
export function costGroups(record: DeskCarrierSettlement, t: TranslateFn): StatementGroup[] {
  const currency = record.currencyCode;
  const loads = (record.costSummary ?? []).filter((group) => group.eventType !== "Adjustment");
  const adjustments = (record.costSummary ?? []).filter(
    (group) => group.eventType === "Adjustment",
  );
  const rows = (groups: readonly DeskCarrierSettlementGroup[], prefix: string) =>
    groups.map((group, index) => ({
      key: `${prefix}:${index}`,
      name: costName(group, t),
      detail: costDetail(group, t),
      basis: costBasis(group, t),
      amount: signedMinor(group.amountMinor, currency),
      manual: group.manual,
    }));
  const total = (groups: readonly DeskCarrierSettlementGroup[]) =>
    signedMinor(
      groups.reduce((sum, group) => sum + group.amountMinor, 0),
      currency,
    );
  const out: StatementGroup[] = [];
  if (loads.length > 0) {
    out.push({
      key: "loads",
      title: t("Load costs"),
      total: total(loads),
      rows: rows(loads, "load"),
    });
  }
  if (adjustments.length > 0) {
    out.push({
      key: "adjustments",
      title: t("Adjustments"),
      total: total(adjustments),
      rows: rows(adjustments, "adjustment"),
    });
  }
  return out;
}

/** How many lines the statement holds in all, for the section's aside. */
export function lineTotal(record: DeskCarrierSettlement): number {
  return (record.costSummary ?? []).reduce((sum, group) => sum + group.lineCount, 0);
}

/**
 * Where the money goes: the remit-to address, how the carrier is paid, and its terms.
 * `withheld` is set when the carrier's payment details are outside its reference view and
 * the person may not read carriers: the address and the carrier's own method and terms are
 * then unknown, not missing, and `method` is null unless the settlement sets its own.
 */
export function remittance(
  record: DeskCarrierSettlement,
  t: TranslateFn,
): { withheld: boolean; lines: string[]; method: string | null; terms: string } {
  const carrier = record.carrier;
  const ownMethod = record.paymentMethod.trim() ? paymentMethodLabel(record.paymentMethod) : "";
  if (carrier && (carrier.paymentMethod === null || carrier.paymentTermDays === null)) {
    return {
      withheld: true,
      lines: [],
      method: ownMethod || null,
      terms: t("Pays {0}", cardDay(record.payDate)),
    };
  }
  const lines = carrier
    ? [
        carrier.remitToName?.trim() || carrier.name,
        carrier.remitAddressLine1?.trim() ?? "",
        carrier.remitAddressLine2?.trim() ?? "",
        [
          carrier.remitCity?.trim(),
          [carrier.remitState?.abbreviation, carrier.remitPostalCode?.trim()]
            .filter(Boolean)
            .join(" "),
        ]
          .filter(Boolean)
          .join(", "),
      ].filter((line) => line !== "")
    : [];
  const hasAddress = Boolean(carrier?.remitAddressLine1?.trim() || carrier?.remitCity?.trim());
  return {
    withheld: false,
    lines: hasAddress ? lines : [],
    method:
      ownMethod ||
      (carrier?.paymentMethod
        ? t("{0} · the carrier's preferred", paymentMethodLabel(carrier.paymentMethod))
        : ""),
    terms:
      carrier?.paymentTermDays != null && carrier.paymentTermDays > 0
        ? t("Net {0} · pays {1}", carrier.paymentTermDays, cardDay(record.payDate))
        : t("Pays {0}", cardDay(record.payDate)),
  };
}
