import type { InboundMessageRow } from "@/lib/graphql/inbox";

/**
 * The lane predicates, read off a message the way the server reads them
 * (`InboundMessage.IsWaiting/IsSnoozed/IsAwaitingReply`): waiting means a
 * person still has to act and nothing has parked it.
 */
type LaneFields = Pick<InboundMessageRow, "needsReview" | "snooze" | "awaitingReply">;

export function isWaiting(message: LaneFields): boolean {
  return message.needsReview && message.snooze === null && message.awaitingReply === null;
}

export function isSnoozed(message: LaneFields): boolean {
  return message.needsReview && message.snooze !== null;
}

export function isAwaitingReply(message: LaneFields): boolean {
  return message.needsReview && message.awaitingReply !== null;
}

/** The name a person knows the sender by: their display name, else the address. */
export function senderName(message: Pick<InboundMessageRow, "fromName" | "fromAddress">): string {
  const name = message.fromName.trim();

  return name === "" ? message.fromAddress : name;
}

/** "Luis" for Luis Ortega, the mailbox's local part for a bare address. */
export function senderFirstName(
  message: Pick<InboundMessageRow, "fromName" | "fromAddress">,
): string {
  const name = message.fromName.trim();
  if (name !== "") {
    return name.split(/\s+/)[0];
  }

  return message.fromAddress.split("@")[0];
}
