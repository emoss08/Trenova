import { getNotificationDescriptor } from "@/components/notification-center/notification-registry";
import { AI_AUDIT_EXPORT_LIST_KEY } from "@/lib/graphql/ai-audit";
import { queries } from "@/lib/queries";
import { RESOURCE_QUERY_KEY_MAP, queryKeyPrefix } from "@trenova/shared/hooks/realtime-patching";
import type { Notification } from "@trenova/shared/types/notification";
import { describe, expect, it } from "vitest";
import { aiAuditExportNotice } from "../ai-audit-exports";

/*
The resource names and notification kinds below are the ones aiauditservice
publishes (services/tms/internal/core/ports/services/aiaudit.go):
AIAuditExportResource "ai-audit-export", AIAuditChainResource
"ai-audit-chain", and the events ai_audit_export_ready,
ai_audit_export_failed and ai_audit_chain_mismatch. A name spelled
differently here matches nothing and fails in silence.
*/

function startsWith(key: readonly unknown[], prefix: readonly unknown[]): boolean {
  return prefix.every((part, index) => key[index] === part);
}

describe("AI audit realtime", () => {
  it("moves the exports table when an export is written, expires or fails", () => {
    const roots = RESOURCE_QUERY_KEY_MAP["ai-audit-export"] ?? [];

    expect(roots.some((root) => startsWith([AI_AUDIT_EXPORT_LIST_KEY], queryKeyPrefix(root)))).toBe(
      true,
    );
  });

  it("reads the chain's status again when a check finishes", () => {
    const roots = RESOURCE_QUERY_KEY_MAP["ai-audit-chain"] ?? [];
    const key = queries.aiAudit.chainStatus().queryKey;

    expect(roots.some((root) => startsWith(key, queryKeyPrefix(root)))).toBe(true);
  });
});

describe("aiAuditExportNotice", () => {
  it("downloads a finished export from its notice", () => {
    expect(
      aiAuditExportNotice({
        eventType: "ai_audit_export_ready",
        data: {
          kind: "ai_audit_export_ready",
          exportId: "aiax_1",
          status: "Succeeded",
          link: '/admin/agent-control?tab=audit&audit=exports&fieldFilters=[{"field":"id","operator":"eq","value":"aiax_1"}]',
        },
      }),
    ).toEqual({
      kind: "download",
      exportId: "aiax_1",
      link: '/admin/agent-control?tab=audit&audit=exports&fieldFilters=[{"field":"id","operator":"eq","value":"aiax_1"}]',
    });
  });

  // The server sets the link only when the catalog knows the record kind; a
  // notice without one still lands on the export, through the same registry.
  it("opens a failed export's row, building the link when the notice has none", () => {
    const notice = aiAuditExportNotice({
      eventType: "ai_audit_export_failed",
      data: { exportId: "aiax_2", status: "Failed" },
    });

    expect(notice?.kind).toBe("open");
    const url = new URL(notice?.link ?? "", "http://localhost");
    expect(url.pathname).toBe("/admin/agent-control");
    expect(url.searchParams.get("tab")).toBe("audit");
    expect(url.searchParams.get("audit")).toBe("exports");
    expect(JSON.parse(url.searchParams.get("fieldFilters") ?? "[]")).toEqual([
      { field: "id", operator: "eq", value: "aiax_2" },
    ]);
  });

  it("never follows a link off the app", () => {
    const notice = aiAuditExportNotice({
      eventType: "ai_audit_export_failed",
      data: { link: "//evil.example/x" },
    });

    expect(notice).toEqual({ kind: "open", link: "/admin/agent-control?tab=audit&audit=exports" });
  });

  it("leaves other notifications alone", () => {
    expect(aiAuditExportNotice({ eventType: "report_run_completed", data: {} })).toBeNull();
  });
});

describe("AI audit notifications in the notification center", () => {
  const notification = (eventType: string, data: Record<string, unknown>) =>
    ({ eventType, data }) as unknown as Notification;

  it("files every AI audit notice under AI Control with a way into the trail", () => {
    for (const eventType of [
      "ai_audit_export_ready",
      "ai_audit_export_failed",
      "ai_audit_chain_mismatch",
    ]) {
      expect(getNotificationDescriptor(eventType).category, eventType).toBe("AI Control");
    }

    expect(
      getNotificationDescriptor("ai_audit_export_ready").getLink?.(
        notification("ai_audit_export_ready", { exportId: "aiax_1" }),
      ),
    ).toContain("audit=exports");
    expect(
      getNotificationDescriptor("ai_audit_chain_mismatch").getLink?.(
        notification("ai_audit_chain_mismatch", { failedSeq: 812 }),
      ),
    ).toBe("/admin/agent-control?tab=audit&audit=trail");
  });
});
