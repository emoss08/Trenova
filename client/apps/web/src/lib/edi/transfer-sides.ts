/** Which side of a tender the reading organization is on. */
export type TransferDirection = "inbound" | "outbound";

/**
 * Which side of the tender an organization is on: the one it was tendered to
 * receives it, even when it also sent it, as a partner's 204 read into the
 * same organization is.
 */
export function transferDirection(
  transfer: { targetOrganizationId: string },
  organizationId: string | null | undefined,
): TransferDirection {
  return transfer.targetOrganizationId === organizationId ? "inbound" : "outbound";
}

/**
 * The other party, as the reader's own organization knows them. Each side of a
 * Trenova connection keeps a partner record for the other, and row-level
 * security hides the other side's: the sender's record of the receiver lives
 * on the source side, the receiver's record of the sender on the target side.
 * A partner's 204 is one record on both.
 */
export function counterparty<Partner>(
  transfer: { sourcePartner?: Partner | null; targetPartner?: Partner | null },
  direction: TransferDirection,
): Partner | null {
  return (direction === "inbound" ? transfer.targetPartner : transfer.sourcePartner) ?? null;
}
