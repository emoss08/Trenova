import type { ShipmentBillingRequirement } from "@trenova/shared/types/shipment";

type PacketRequirement = Pick<ShipmentBillingRequirement, "documentIds">;

/**
 * The documents an invoice packet carries, in the order they follow the invoice.
 * Until the biller picks, the item sends each document satisfying one of the
 * customer's required types, requirement by requirement, which is how approval
 * assembles it; once they pick, the pick and its order are the packet, even empty.
 */
export function packetDocumentIds(
  packetIds: readonly string[],
  picked: boolean,
  requirements: readonly PacketRequirement[],
): string[] {
  if (picked || packetIds.length > 0) {
    return [...packetIds];
  }
  const ids: string[] = [];
  for (const requirement of requirements) {
    for (const id of requirement.documentIds) {
      if (!ids.includes(id)) ids.push(id);
    }
  }
  return ids;
}

/**
 * The packet after one document is added or taken out. It starts from the packet
 * the biller was looking at, so taking out one required document never silently
 * drops the others, and an added document goes last.
 */
export function packetAfterToggle(
  documentId: string,
  include: boolean,
  current: readonly string[],
): string[] {
  if (include) {
    return current.includes(documentId) ? [...current] : [...current, documentId];
  }
  return current.filter((id) => id !== documentId);
}

/**
 * The packet with one document at `index`: moved there if it was already in the
 * packet, put there if it was not. An index past the end puts it last.
 */
export function packetAfterPlace(
  documentId: string,
  index: number,
  current: readonly string[],
): string[] {
  const rest = current.filter((id) => id !== documentId);
  rest.splice(Math.max(0, Math.min(index, rest.length)), 0, documentId);
  return rest;
}
