import type { Notification } from "@trenova/shared/types/notification";
import { describe, expect, it } from "vitest";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import {
  buildNotificationCatalog,
  SYSTEM_PREFERENCE_KEY,
  TCA_PREFERENCE_KEY,
  getNotificationDescriptor,
  getNotificationLink,
  resolveNotificationDescriptor,
} from "./notification-registry";

const DISPATCH_EVENT_TYPES = [
  "tender_accepted",
  "tender_needs_review",
  "tender_waterfall_exhausted",
  "rate_confirmation_issue_failed",
  "tender_delivery_failed",
  "tender_entries_skipped",
] as const;

function notification(overrides: Partial<Notification> = {}): Notification {
  return {
    id: "not_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    targetUserId: null,
    eventType: "tender_accepted",
    priority: "high",
    channel: "global",
    title: "Tender accepted",
    message: "Blue Ridge Freight accepted the tender",
    data: null,
    relatedEntities: null,
    source: "tender",
    readAt: null,
    dismissedAt: null,
    createdAt: 1_700_000_000,
    ...overrides,
  };
}

describe("notification registry — dispatch tender descriptors", () => {
  const fallback = getNotificationDescriptor("some.unregistered.event");

  it.each(DISPATCH_EVENT_TYPES)("registers %s under the Dispatch category", (eventType) => {
    const descriptor = getNotificationDescriptor(eventType);

    expect(descriptor).not.toBe(fallback);
    expect(descriptor.category).toBe("Dispatch");
    expect(descriptor.icon).not.toBe(fallback.icon);
  });

  it.each(DISPATCH_EVENT_TYPES)("prefers the producer's data.link for %s", (eventType) => {
    const link = getNotificationLink(
      notification({ eventType, data: { link: "/dispatch/console?move=smv_01" } }),
    );

    expect(link).toBe("/dispatch/console?move=smv_01");
  });

  it.each(DISPATCH_EVENT_TYPES)(
    "falls back to the dispatch console for %s when data carries no link",
    (eventType) => {
      expect(getNotificationLink(notification({ eventType }))).toBe("/dispatch/console");
    },
  );

  it.each(DISPATCH_EVENT_TYPES)(
    "falls back to the dispatch console for %s when data.link is not a usable string",
    (eventType) => {
      expect(getNotificationLink(notification({ eventType, data: { link: "" } }))).toBe(
        "/dispatch/console",
      );
      expect(getNotificationLink(notification({ eventType, data: { link: 42 } }))).toBe(
        "/dispatch/console",
      );
    },
  );
});

describe("notification registry — generic link handling", () => {
  it("surfaces data.link for an unregistered event type", () => {
    const link = getNotificationLink(
      notification({ eventType: "some.unregistered.event", data: { link: "/dispatch/console" } }),
    );

    expect(link).toBe("/dispatch/console");
  });

  it("surfaces data.link for a tca.* data alert", () => {
    const link = getNotificationLink(
      notification({ eventType: "tca.table_change", data: { link: "/shipment-management" } }),
    );

    expect(link).toBe("/shipment-management");
  });

  it("still returns null for a link-less notification", () => {
    expect(getNotificationLink(notification({ eventType: "some.unregistered.event" }))).toBeNull();
    expect(getNotificationLink(notification({ eventType: "tca.table_change" }))).toBeNull();
  });
});

describe("notification registry — invoice notifications", () => {
  it("files a shared invoice under Billing with the sharer as its avatar", () => {
    const descriptor = getNotificationDescriptor("invoice_shared");
    const shared = notification({
      eventType: "invoice_shared",
      data: { sharedById: "usr_marcus", sharedByName: "Marcus Bell" },
    });

    expect(descriptor).not.toBe(getNotificationDescriptor("some.unregistered.event"));
    expect(descriptor.category).toBe("Billing");
    expect(descriptor.avatar?.(shared)).toEqual({ userId: "usr_marcus", name: "Marcus Bell" });
  });

  it("opens a shared invoice on the tab the sharer linked", () => {
    const link = getNotificationLink(
      notification({
        eventType: "invoice_shared",
        data: { link: "/billing/invoices?item=inv_1&tab=charges" },
        relatedEntities: { invoiceId: "inv_1" },
      }),
    );

    expect(link).toBe("/billing/invoices?item=inv_1&tab=charges");
  });

  it("opens a shared invoice from its id when the link is missing", () => {
    const link = getNotificationLink(
      notification({ eventType: "invoice_shared", relatedEntities: { invoiceId: "inv_1" } }),
    );

    expect(link).toBe("/billing/invoices?item=inv_1");
  });

  it("selects the invoice for a reconciliation warning using the parameter the page reads", () => {
    const link = getNotificationLink(
      notification({
        eventType: "invoice_reconciliation_warning",
        relatedEntities: { invoiceId: "inv_9" },
      }),
    );

    expect(link).toBe("/billing/invoices?item=inv_9");
  });

  it("falls back to the invoice list when no invoice is named", () => {
    expect(getNotificationLink(notification({ eventType: "invoice_shared" }))).toBe(
      "/billing/invoices",
    );
    expect(getNotificationLink(notification({ eventType: "invoice_reconciliation_warning" }))).toBe(
      "/billing/invoices",
    );
  });
});

describe("notification registry — carrier intelligence notifications", () => {
  const CARRIER_INTEL_EVENT_TYPES = [
    "carrier_intel_block",
    "carrier_intel_change",
    "carrier_intel_digest",
    "carrier_intel_provider_paused",
    "carrier_equipment_mismatch",
    "carrier_intel_spend_soft_cap",
    "carrier_intel_spend_cap",
  ] as const;
  const fallback = getNotificationDescriptor("some.unregistered.event");

  it.each(CARRIER_INTEL_EVENT_TYPES)(
    "registers %s under the Carrier Intelligence category",
    (eventType) => {
      const descriptor = getNotificationDescriptor(eventType);

      expect(descriptor).not.toBe(fallback);
      expect(descriptor.category).toBe("Carrier Intelligence");
      expect(descriptor.icon).not.toBe(fallback.icon);
    },
  );

  it.each(CARRIER_INTEL_EVENT_TYPES)("prefers the producer's data.link for %s", (eventType) => {
    const link = getNotificationLink(
      notification({ eventType, data: { link: "/dispatch/carriers?panelEntityId=car_7" } }),
    );

    expect(link).toBe("/dispatch/carriers?panelEntityId=car_7");
  });

  it("opens the carrier's intelligence tab for a block when the link is missing", () => {
    const link = getNotificationLink(
      notification({
        eventType: "carrier_intel_block",
        relatedEntities: { carrierId: "car_1", dotNumber: "1234567" },
      }),
    );

    expect(link).toBe("/dispatch/carriers?panelType=edit&panelEntityId=car_1&tab=intelligence");
  });

  it("falls back to the monitoring page for a change on a subject that is not a carrier", () => {
    expect(
      getNotificationLink(
        notification({ eventType: "carrier_intel_change", relatedEntities: { dotNumber: "99" } }),
      ),
    ).toBe("/dispatch/carrier-monitoring");
    expect(getNotificationLink(notification({ eventType: "carrier_intel_digest" }))).toBe(
      "/dispatch/carrier-monitoring",
    );
  });

  it("sends spend cap notifications to the usage tab without a link", () => {
    expect(getNotificationLink(notification({ eventType: "carrier_intel_spend_soft_cap" }))).toBe(
      "/dispatch/carrier-monitoring?tab=usage",
    );
    expect(getNotificationLink(notification({ eventType: "carrier_intel_spend_cap" }))).toBe(
      "/dispatch/carrier-monitoring?tab=usage",
    );
  });

  it("sends a paused provider to the integrations catalog without a link", () => {
    expect(getNotificationLink(notification({ eventType: "carrier_intel_provider_paused" }))).toBe(
      "/admin/integrations?category=CarrierCompliance",
    );
  });
});

describe("notification registry — pending proposals", () => {
  it("opens the proposals list the notice links to, filtered to its run", () => {
    const link =
      "/admin/agent-control?activity=proposals&fieldFilters=%5B%7B%22field%22%3A%22runId%22%7D%5D&tab=activity";
    for (const eventType of ["agent.proposals_pending", "agent.proposals_reminder"]) {
      expect(getNotificationDescriptor(eventType).category).toBe("AI Control");
      expect(getNotificationLink(notification({ eventType, data: { link } }))).toBe(link);
    }
  });
});

describe("notification registry — earned autonomy", () => {
  it("files a tier change under AI Control and opens the agents tab it names", () => {
    for (const eventType of ["agent.tool_promoted", "agent.tool_demoted"]) {
      const descriptor = getNotificationDescriptor(eventType);
      expect(descriptor.category).toBe("AI Control");
      expect(
        getNotificationLink(
          notification({ eventType, data: { link: "/admin/agent-control?tab=agents" } }),
        ),
      ).toBe("/admin/agent-control?tab=agents");
    }
  });

  it("falls back to the AI Control page when the notice carries no link", () => {
    expect(getNotificationLink(notification({ eventType: "agent.tool_demoted" }))).toBe(
      "/admin/agent-control",
    );
  });
});

/*
A suite run that scored an agent worse than its baseline: data carries link
(the Quality tab with the agent and run open), suiteRunId, agentDefinitionId,
severity (Critical when a case failed hard, Warning otherwise) and changes.
*/
describe("notification registry — quality regressions", () => {
  const link = "/admin/agent-control?tab=quality&agent=agdef_1&suiteRun=asr_1";

  it("files a regression under AI Control and opens the run it names", () => {
    const n = notification({
      eventType: "agent.quality_regression",
      data: { link, suiteRunId: "asr_1", agentDefinitionId: "agdef_1", severity: "Warning" },
    });

    expect(getNotificationDescriptor("agent.quality_regression").category).toBe("AI Control");
    expect(getNotificationLink(n)).toBe(link);
    expect(resolveNotificationDescriptor(n).tileClass).toBe("bg-warning-subtle");
  });

  it("takes the danger tone when a case failed hard", () => {
    const descriptor = resolveNotificationDescriptor(
      notification({
        eventType: "agent.quality_regression",
        data: { link, severity: "Critical" },
      }),
    );

    expect(descriptor.tileClass).toBe("bg-danger-subtle");
    expect(descriptor.iconClass).toBe("text-destructive");
  });

  it("falls back to the AI Control page when the notice carries no link", () => {
    expect(getNotificationLink(notification({ eventType: "agent.quality_regression" }))).toBe(
      "/admin/agent-control",
    );
  });
});

/*
A finished reply nobody was watching: data carries kind "assistant_reply_ready",
threadId, turnId, status (Completed | Refused | Failed) and link.
*/
describe("notification registry — assistant replies", () => {
  const replyData = {
    kind: "assistant_reply_ready",
    threadId: "athr_1",
    turnId: "atrn_1",
    status: "Completed",
    link: "/desk/t/athr_1",
  };

  it("opens the conversation the reply was written in", () => {
    expect(
      getNotificationLink(notification({ eventType: "assistant_reply_ready", data: replyData })),
    ).toBe("/desk/t/athr_1");
  });

  it("recognises the reply by its data when the event type is something else", () => {
    const n = notification({ eventType: "assistant.turn_finished", data: replyData });

    expect(getNotificationLink(n)).toBe("/desk/t/athr_1");
    expect(resolveNotificationDescriptor(n).icon).toBe(AssistMark);
  });

  it("never navigates off the app on the strength of the notice's link", () => {
    expect(
      getNotificationLink(
        notification({
          eventType: "assistant_reply_ready",
          data: { ...replyData, link: "https://example.com" },
        }),
      ),
    ).toBe("/desk/t/athr_1");
  });

  it("wears the assistant's mark, neutral for a completed or refused reply", () => {
    for (const status of ["Completed", "Refused"]) {
      const descriptor = resolveNotificationDescriptor(
        notification({ eventType: "assistant_reply_ready", data: { ...replyData, status } }),
      );

      expect(descriptor.icon).toBe(AssistMark);
      expect(descriptor.tileClass).toBe("bg-muted");
    }
  });

  it("takes the danger tone for a reply that failed", () => {
    const descriptor = resolveNotificationDescriptor(
      notification({
        eventType: "assistant_reply_ready",
        data: { ...replyData, status: "Failed" },
      }),
    );

    expect(descriptor.tileClass).toBe("bg-danger-subtle");
    expect(descriptor.iconClass).toBe("text-destructive");
  });

  it("leaves every other kind's descriptor as its event type has it", () => {
    const n = notification({ eventType: "tender_accepted" });

    expect(resolveNotificationDescriptor(n)).toBe(getNotificationDescriptor("tender_accepted"));
  });
});

describe("notification catalog", () => {
  const catalog = buildNotificationCatalog();
  const types = catalog.flatMap((category) => category.types);

  it("lists every kind once, grouped by the registry's category", () => {
    const keys = types.map((type) => type.key);
    expect(new Set(keys).size).toBe(keys.length);
    const dispatch = catalog.find((category) => category.name === "Dispatch");
    expect(new Set(dispatch?.types.map((type) => type.key))).toEqual(new Set(DISPATCH_EVENT_TYPES));
    for (const category of catalog) {
      expect(category.types.length).toBeGreaterThan(0);
    }
  });

  it("gives every kind a sentence a person reads, not its key", () => {
    for (const type of types) {
      expect(type.label).not.toBe(type.key);
      expect(type.label.length).toBeGreaterThan(3);
    }
  });

  it("puts table change alerts and system notices behind one key each, and locks the required ones", () => {
    expect(types.find((type) => type.key === TCA_PREFERENCE_KEY)?.requiredReason).toBeNull();
    expect(types.find((type) => type.key === SYSTEM_PREFERENCE_KEY)?.requiredReason).not.toBeNull();
    expect(
      types.find((type) => type.key === "ai_audit_chain_mismatch")?.requiredReason,
    ).not.toBeNull();
    expect(types.filter((type) => type.requiredReason !== null)).toHaveLength(2);
  });

  it("orders categories the way the settings page reads them and gives each a description", () => {
    expect(catalog.map((category) => category.name).slice(0, 3)).toEqual([
      "Mentions",
      "Assistant",
      "Dispatch",
    ]);
    expect(catalog.at(-1)?.name).toBe("System");
    for (const category of catalog) {
      expect(category.description).not.toBe("");
    }
  });

  it("marks as urgent exactly the kinds the server delivers as urgent", () => {
    const goSource = readFileSync(
      resolve(
        __dirname,
        "../../../../../../services/tms/internal/modules/workspace/domain/notification/preference.go",
      ),
      "utf8",
    );
    const body = goSource.slice(goSource.indexOf("func urgentEventTypes()"));
    const block = body.slice(0, body.indexOf("\n}\n"));
    const serverUrgent = [...block.matchAll(/"([a-z0-9_.]+)"/g)].map((match) => match[1]);
    serverUrgent.push("ai_audit_chain_mismatch");

    const clientUrgent = types.filter((type) => type.urgent).map((type) => type.key);
    expect(new Set(clientUrgent)).toEqual(new Set(serverUrgent));
  });
});

describe("notification registry — reports that need attention", () => {
  const needsAttention = (data: Record<string, unknown>) =>
    notification({ eventType: "report_needs_attention", channel: "user", data });

  it("files the notice under Reports as a warning", () => {
    const descriptor = getNotificationDescriptor("report_needs_attention");

    expect(descriptor.category).toBe("Reports");
    expect(descriptor.iconClass).toBe("text-warning");
  });

  it("opens the one report a notice names in the builder", () => {
    expect(getNotificationLink(needsAttention({ reportCount: 1, definitionIds: ["rd_1"] }))).toBe(
      "/reports/explore/rd_1",
    );
  });

  it("opens the library for a notice about several reports, or one that names none", () => {
    expect(
      getNotificationLink(needsAttention({ reportCount: 2, definitionIds: ["rd_1", "rd_2"] })),
    ).toBe("/reports");
    expect(getNotificationLink(needsAttention({ reportCount: 1 }))).toBe("/reports");
    expect(getNotificationLink(needsAttention({ definitionIds: [""] }))).toBe("/reports");
  });
});
