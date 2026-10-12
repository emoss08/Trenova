import type { ResourceInvalidationEvent } from "@trenova/shared/hooks/realtime-patching";
import type { ThreadPages } from "@/lib/thread-list";
import type { AssistantThread } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { boundCaseRecords, boundCaseRecordsIn, mayTouchCase, touchesCase } from "../case-realtime";

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
  const list: ThreadPages = {
    pages: [
      { items: [{ ...caseThread("shp_plain"), case: undefined }], nextCursor: "c1" },
      { items: [caseThread("shp_case")], nextCursor: "" },
    ],
    pageParams: ["", "c1"],
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

/**
 * The rail reads each shelf as a list of its own, so a case about a record
 * may be held by any of them; a change to its record reaches it whichever.
 */
describe("boundCaseRecordsIn", () => {
  const list = (subjectId: string): ThreadPages => ({
    pages: [{ items: [caseThread(subjectId)], nextCursor: "" }],
    pageParams: [""],
  });

  it("reads the records of the cases every list holds", () => {
    const bound = boundCaseRecordsIn([list("shp_today"), list("shp_older")]);

    expect([...bound].sort()).toEqual(["shp_older", "shp_today"]);
    expect(touchesCase(event({ recordId: "shp_older" }), bound)).toBe(true);
  });

  it("reads one list as that list, and none as none", () => {
    const only = list("shp_one");

    expect(boundCaseRecordsIn([only])).toBe(boundCaseRecords(only));
    expect(boundCaseRecordsIn([]).size).toBe(0);
  });
});

describe("mayTouchCase", () => {
  it("passes only the kinds of record a case is about or reads", () => {
    expect(mayTouchCase(event({ resource: "shipments" }))).toBe(true);
    expect(mayTouchCase(event({ resource: "document" }))).toBe(true);
    expect(mayTouchCase(event({ resource: "workers" }))).toBe(false);
  });
});
