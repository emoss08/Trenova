import type { InboundClassification } from "@/lib/graphql/inbox";

/**
 * What a reader looks for first in each kind of message, by the keys Nova
 * gives the facts it reads. A tender is its lane and when to answer by; an
 * invoice is its number and what is owed; a dispute is the money in dispute.
 * Each entry is an order of preference: the first keys present lead.
 */
const LEAD_KEYS: Partial<Record<InboundClassification, readonly string[]>> = {
  Tender: ["lane", "respondBy", "loads", "pickup"],
  RateConfirmation: ["lane", "rate", "shipment"],
  Invoice: ["totalDue", "amount", "invoice", "terms", "dueDate"],
  ProofOfDelivery: ["shipment", "exception", "signature"],
  StatusRequest: ["purchaseOrder", "shipment", "receivingCutoff", "pickupDay"],
  DetentionDispute: ["disputedAmount", "invoice", "lateArrival"],
};

/** How many facts lead a reading; the rest follow as they were read. */
const LEAD_COUNT = 2;

export type LedFacts<T> = { lead: T[]; rest: T[] };

/**
 * Splits what Nova read into the facts that lead for this kind of message
 * and the rest, in the order they were read. A kind with no lead of its own,
 * or a reading with none of its lead keys, leads with nothing, and reads as a
 * plain list.
 */
export function leadFacts<T extends { key: string | null }>(
  classification: InboundClassification | null | undefined,
  facts: readonly T[],
): LedFacts<T> {
  const wanted = (classification && LEAD_KEYS[classification]) || [];
  const lead: T[] = [];
  for (const key of wanted) {
    const found = facts.find((fact) => fact.key === key);
    if (found && lead.length < LEAD_COUNT) {
      lead.push(found);
    }
  }

  return { lead, rest: facts.filter((fact) => !lead.includes(fact)) };
}
