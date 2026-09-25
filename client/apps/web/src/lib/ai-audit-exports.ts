import { recordPath } from "@/config/record-links";
import { fetchAIAuditExportDownload } from "@/lib/graphql/ai-audit";
import { downloadFromUrl } from "@trenova/shared/lib/utils";

/** The notification kinds aiauditservice sends a requester about their export. */
export const AI_AUDIT_EXPORT_READY_EVENT = "ai_audit_export_ready";
export const AI_AUDIT_EXPORT_FAILED_EVENT = "ai_audit_export_failed";
/** Sent to everyone who reads the trail when its chain fails verification. */
export const AI_AUDIT_CHAIN_MISMATCH_EVENT = "ai_audit_chain_mismatch";

/** Where the trail's exports list opens, for a notice that names no single export. */
export const AI_AUDIT_EXPORTS_PATH = "/admin/agent-control?tab=audit&audit=exports";
/** Where the trail opens. */
export const AI_AUDIT_TRAIL_PATH = "/admin/agent-control?tab=audit&audit=trail";

/**
 * Fetches the one-minute link to an export's file and downloads it. Only the
 * export's requester is given a link; anyone else's attempt is refused by
 * the server and the refusal is thrown to the caller.
 */
export async function downloadAIAuditExport(exportId: string): Promise<void> {
  const file = await fetchAIAuditExportDownload(exportId);
  downloadFromUrl(file.url, file.fileName);
}

type NotificationLike = {
  eventType?: string | null;
  data?: Record<string, unknown> | null;
};

export type AIAuditExportNotice =
  | { kind: "download"; exportId: string; link: string }
  | { kind: "open"; link: string };

function sameOriginPath(value: unknown): string | null {
  return typeof value === "string" && value.startsWith("/") && !value.startsWith("//")
    ? value
    : null;
}

/**
 * What a notification about an AI audit export offers: a finished file is
 * downloaded from the toast, a failed one opens the exports list. The link
 * the server sends is built from the record-link registry; a notice without
 * one falls back to the same registry.
 */
export function aiAuditExportNotice(notification: NotificationLike): AIAuditExportNotice | null {
  const exportId =
    typeof notification.data?.exportId === "string" && notification.data.exportId !== ""
      ? notification.data.exportId
      : null;
  const link =
    sameOriginPath(notification.data?.link) ??
    (exportId ? recordPath("ai_audit_export", exportId) : AI_AUDIT_EXPORTS_PATH);

  switch (notification.eventType) {
    case AI_AUDIT_EXPORT_READY_EVENT:
      return exportId ? { kind: "download", exportId, link } : { kind: "open", link };
    case AI_AUDIT_EXPORT_FAILED_EVENT:
      return { kind: "open", link };
    default:
      return null;
  }
}
