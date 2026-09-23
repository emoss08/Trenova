import { recordAtLocation, recordPath } from "@/config/record-links";

const DESK_PATH = "/desk";

/**
 * Whether a location is inside the Desk, which runs outside the app shell and
 * so has no corner panel.
 */
export function isDeskPath(pathname: string): boolean {
  return pathname === DESK_PATH || pathname.startsWith(`${DESK_PATH}/`);
}

/**
 * Where a conversation opens: the Desk, which shows any of the person's
 * conversations — a kept one from the list, and a quick question from the
 * palette that was never kept, reached from the notice that its answer is in.
 * Built from the record-link registry like every other record's address.
 */
export function conversationPath(threadId: string): string {
  return recordPath("assistant_thread", threadId);
}

/** The conversation a location shows at the Desk, or null for any other page. */
export function conversationAtPath(pathname: string): string | null {
  let record: ReturnType<typeof recordAtLocation>;
  try {
    record = recordAtLocation(pathname, "");
  } catch {
    // A malformed escape in the address names no conversation.
    return null;
  }

  return record?.entityType === "assistant_thread" && record.entityId !== ""
    ? record.entityId
    : null;
}
