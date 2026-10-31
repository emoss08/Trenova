import type { Document } from "@trenova/shared/types/document";

type PacketDocument = Pick<Document, "id" | "documentTypeId">;

/**
 * Whether a document goes out with the invoice. An item that names no packet sends
 * the documents its customer requires; once the biller picks, the pick is the packet.
 */
export function isInPacket(
  doc: PacketDocument,
  packetIds: readonly string[],
  requiredTypes: ReadonlySet<string>,
): boolean {
  if (packetIds.length > 0) {
    return packetIds.includes(doc.id);
  }
  return Boolean(doc.documentTypeId && requiredTypes.has(doc.documentTypeId));
}

/**
 * The packet after one document is ticked or unticked. The first change starts from
 * the default the biller was looking at, so unticking one required document never
 * silently drops the others.
 */
export function packetAfterToggle(
  doc: PacketDocument,
  include: boolean,
  documents: readonly PacketDocument[],
  packetIds: readonly string[],
  requiredTypes: ReadonlySet<string>,
): string[] {
  const current =
    packetIds.length > 0
      ? [...packetIds]
      : documents.filter((entry) => isInPacket(entry, [], requiredTypes)).map((entry) => entry.id);
  if (include) {
    return current.includes(doc.id) ? current : [...current, doc.id];
  }
  return current.filter((id) => id !== doc.id);
}
