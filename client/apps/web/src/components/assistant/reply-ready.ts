import { isAppPath } from "@/lib/app-path";
import { conversationAtPath, conversationPath, isDeskPath } from "@/lib/conversation-path";
import { z } from "zod";

/** The notification a reply nobody was watching leaves behind when it ends. */
export const ASSISTANT_REPLY_READY = "assistant_reply_ready";

const replyStatusSchema = z.enum(["Completed", "Refused", "Failed"]);

export type ReplyReadyStatus = z.infer<typeof replyStatusSchema>;

const replyReadyDataSchema = z.object({
  kind: z.string().optional(),
  threadId: z.string().min(1),
  turnId: z.string().optional().default(""),
  // A status this client does not know yet is still a finished reply; it is
  // read as one that completed rather than dropping the notice.
  status: replyStatusSchema.catch("Completed"),
  link: z.string().optional(),
});

export type ReplyReady = {
  threadId: string;
  turnId: string;
  status: ReplyReadyStatus;
  /** Where the conversation opens; always a path inside the app. */
  link: string;
};

type NotificationLike = {
  eventType?: string | null;
  data?: Record<string, unknown> | null;
};

/**
 * The finished reply a notification announces, or null when it announces
 * something else. Recognised by its event type or by the kind its data
 * carries, since both name it.
 *
 * The link is the server's when it is a path inside the app and the Desk's
 * address for the conversation otherwise: a link followed on a click must
 * never be able to leave the app.
 */
export function parseReplyReady(notification: NotificationLike): ReplyReady | null {
  const data = notification.data ?? null;
  if (notification.eventType !== ASSISTANT_REPLY_READY && data?.kind !== ASSISTANT_REPLY_READY) {
    return null;
  }

  const parsed = replyReadyDataSchema.safeParse(data);
  if (!parsed.success) {
    return null;
  }

  const { threadId, turnId, status, link } = parsed.data;

  return {
    threadId,
    turnId,
    status,
    link: isAppPath(link) ? link : conversationPath(threadId),
  };
}

export type ViewingState = {
  pathname: string;
  /** Whether the corner panel is open, and on which conversation. */
  panelOpen: boolean;
  panelThreadId: string | null;
};

/**
 * Whether the person is already looking at a conversation: open at the Desk,
 * or open in the corner panel. A notice that its reply is ready would then be
 * news about the thing in front of them.
 *
 * The panel's remembered state is ignored at the Desk, which has no panel:
 * the store can say "open on this thread" while nothing is drawing it.
 */
export function isViewingThread(threadId: string, viewing: ViewingState): boolean {
  if (isDeskPath(viewing.pathname)) {
    return conversationAtPath(viewing.pathname) === threadId;
  }

  return viewing.panelOpen && viewing.panelThreadId === threadId;
}
