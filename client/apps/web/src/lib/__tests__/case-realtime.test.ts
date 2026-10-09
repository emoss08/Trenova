import type { ResourceInvalidationEvent } from "@trenova/shared/hooks/realtime-patching";
import type { AssistantThread, AssistantThreadList } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { boundCaseRecords, touchesCase } from "../case-realtime";

function event(overrides: Partial<ResourceInvalidationEvent>): ResourceInvalidationEvent {
  return { organizationId: "org", businessUnitId: "bu", resource: "shipments", ...overrides };
}

function caseThread(subjectId: string): AssistantThread {
  return {
    id: `athr_${subjectId}`,
    businessUnitId: "bu",
    organizationId: "org",
    userId: "u",
    agentDefinitionId: "agd",
    title: "",
    status: "Active",
    lastMessageAt: 0,
    preferredProviderId: "",
    origin: "Desk",
    pinned: false,
    subjectType: "Shipment",
    subjectId,
    canContinue: true,
    version: 0,
    createdAt: 0,
    updatedAt: 0,
    case: {
      state: "Working",
      openWaits: 0,
      record: {
        type: "Shipment",
        id: subjectId,
        label: "1",
        status: "InTransit",
        closed: false,
        closedAs: "",
        invoiceId: "",
        customerId: "",
        carrierIds: [],
      },
    },
  };
}

/**
 * The data channel carries every change in the organization; a case is
 * refetched only when a change names its record, so a busy dispatch floor
 * does not refetch the rail on every shipment edit.
 */
describe("touchesCase", () => {
  const list: AssistantThreadList = {
    items: [caseThread("shp_case"), { ...caseThread("shp_plain"), case: undefined }],
    total: 2,
  };
  const bound = boundCaseRecords(list);

  it("reads the records only the cases are about, once per list", () => {
    expect([...bound]).toEqual(["shp_case"]);
    expect(boundCaseRecords(list)).toBe(bound);
    expect(boundCaseRecords(undefined).size).toBe(0);
  });

  it("matches a change to the record itself", () => {
    expect(touchesCase(event({ recordId: "shp_case" }), bound)).toBe(true);
    expect(touchesCase(event({ entityId: "shp_case" }), bound)).toBe(true);
  });

  it("matches what the checklist reads through the record it names", () => {
    expect(
      touchesCase(
        event({ resource: "document", recordId: "doc_1", entity: { resourceId: "shp_case" } }),
        bound,
      ),
    ).toBe(true);
    expect(
      touchesCase(event({ resource: "billing_queue", entity: { shipmentId: "shp_case" } }), bound),
    ).toBe(true);
  });

  it("ignores other records, other resources and a conversation that is no case", () => {
    expect(touchesCase(event({ recordId: "shp_other" }), bound)).toBe(false);
    expect(touchesCase(event({ recordId: "shp_plain" }), bound)).toBe(false);
    expect(touchesCase(event({ resource: "workers", recordId: "shp_case" }), bound)).toBe(false);
    expect(touchesCase(event({ recordId: "shp_case" }), new Set())).toBe(false);
  });
});
