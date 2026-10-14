import { translate } from "@trenova/shared/i18n/runtime";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { cardMoment, cardMomentAfter, money } from "../../../record-card/record-card-values";
import {
  counterparty,
  decisionGates,
  mappingLedger,
  reasonNote,
  tenderCharges,
  tenderFreight,
  tenderRoute,
  tenderSlip,
  transferDirection,
  transferLinks,
} from "../edi-transfer-card";
import { transferStanding } from "../../../record-card/edi-transfer-standing";
import { NOW, mappingPreview, transfer } from "./fixtures";

const t = translate as TranslateFn;
const ORG = "org_1";

const gates = (name: Parameters<typeof transfer>[0], overrides = {}) => {
  const record = transfer(name, overrides);
  return decisionGates(record, transferDirection(record, ORG), t).map((gate) => [
    gate.key,
    gate.state,
    gate.detail,
    gate.at,
  ]);
};

describe("transferDirection and counterparty", () => {
  it("reads a tender our organization receives as inbound, whoever sent it", () => {
    expect(transferDirection(transfer("mappingRequired"), ORG)).toBe("inbound");
    expect(transferDirection(transfer("outbound"), ORG)).toBe("outbound");
    expect(transferDirection(transfer("outbound"), "org_lonestar")).toBe("inbound");
  });

  it("names the other party by the partner record our own organization keeps for them", () => {
    expect(counterparty(transfer("mappingRequired"), "inbound")?.name).toBe("Acme Foods");
    expect(counterparty(transfer("outbound"), "outbound")?.name).toBe("Lone Star Logistics");
    expect(counterparty(transfer("outbound"), "inbound")?.name).toBe("Trenova Freight");
    expect(
      counterparty(transfer("outbound", { sourcePartner: null }), "outbound")?.name,
    ).toBeUndefined();
  });
});

describe("transferStanding", () => {
  it("says where a tender stands in the words of the side reading it", () => {
    const standing = (name: Parameters<typeof transfer>[0], overrides = {}) => {
      const record = transfer(name, overrides);
      return transferStanding(record, transferDirection(record, ORG), t);
    };
    expect(standing("mappingRequired")).toEqual({
      label: "Needs mapping",
      tone: "warn",
      mark: "pause",
    });
    expect(standing("mappingRequired", { status: "PendingApproval" })).toEqual({
      label: "Awaiting your decision",
      tone: "progress",
      mark: "dot",
    });
    expect(standing("outbound")).toEqual({
      label: "Awaiting their decision",
      tone: "progress",
      mark: "dot",
    });
    expect(standing("mappingRequired", { status: "Submitted" })).toEqual({
      label: "Submitted",
      tone: "progress",
      mark: "dot",
    });
    expect(standing("approved", { status: "Processing" })).toEqual({
      label: "Building the shipment",
      tone: "progress",
      mark: "dot",
    });
    expect(standing("approved")).toEqual({ label: "Accepted", tone: "success", mark: "check" });
    expect(standing("rejected")).toEqual({ label: "Declined", tone: "muted", mark: "dot" });
    expect(standing("superseded")).toEqual({
      label: "Replaced by a change",
      tone: "muted",
      mark: "dot",
    });
    expect(standing("superseded", { failureReason: "" })).toEqual({
      label: "Canceled",
      tone: "muted",
      mark: "dot",
    });
    expect(standing("rejected", { status: "Expired" })).toEqual({
      label: "Expired",
      tone: "muted",
      mark: "dot",
    });
    expect(standing("failed")).toEqual({ label: "Failed", tone: "danger", mark: "x" });
  });
});

describe("tenderSlip", () => {
  it("heads the card with the bill of lading and says whether the tender is a change", () => {
    expect(tenderSlip(transfer("mappingRequired"), t)).toEqual({
      title: "LT-88241",
      purpose: "Change",
    });
    expect(tenderSlip(transfer("approved"), t)).toEqual({ title: "LT-88240", purpose: "" });
    const blank = transfer("approved");
    blank.tenderPayload.bol = "";
    blank.tenderPayload.purposeCode = "01";
    expect(tenderSlip(blank, t)).toEqual({ title: "Load tender", purpose: "Cancellation" });
  });
});

describe("decisionGates", () => {
  it("stops at mapping when the tender names records nobody mapped yet", () => {
    expect(gates("mappingRequired")).toEqual([
      ["received", "done", "Read from an EDI 204", NOW - 7200],
      ["mapped", "warn", "3 of 5 mapped", null],
      ["decided", "ahead", "", null],
      ["built", "ahead", "", null],
    ]);
  });

  it("follows an accepted tender to the shipment it built", () => {
    expect(gates("approved")).toEqual([
      ["received", "done", "Read from an EDI 204", NOW - 7200],
      ["mapped", "done", "All 5 mapped", null],
      ["decided", "done", "Accepted", NOW - 3600],
      ["built", "done", "Shipment created", NOW - 3600 + 9],
    ]);
    expect(gates("approved", { status: "Processing", processedAt: null })).toEqual([
      ["received", "done", "Read from an EDI 204", NOW - 7200],
      ["mapped", "done", "All 5 mapped", null],
      ["decided", "done", "Accepted", NOW - 3600],
      ["built", "now", "Creating the shipment", NOW - 3600 + 2],
    ]);
  });

  it("closes the way where the tender was declined, replaced or failed", () => {
    expect(gates("rejected").slice(2)).toEqual([
      ["decided", "closed", "Declined", NOW - 5 * 3600],
      ["built", "skipped", "", null],
    ]);
    expect(gates("superseded").slice(2)).toEqual([
      ["decided", "closed", "Replaced by a change", NOW - 3 * 3600],
      ["built", "skipped", "", null],
    ]);
    expect(gates("rejected", { status: "Expired", rejectedAt: null }).slice(2)).toEqual([
      ["decided", "closed", "Expired before a decision", null],
      ["built", "skipped", "", null],
    ]);
    expect(gates("failed").slice(2)).toEqual([
      ["decided", "done", "Accepted", NOW - 4 * 3600],
      ["built", "failed", "Could not be created", NOW - 4 * 3600 + 5],
    ]);
  });

  it("waits on the other side for a tender we sent", () => {
    expect(gates("outbound")).toEqual([
      ["received", "done", "Tendered from shipment", NOW - 2400],
      ["mapped", "done", "Nothing to map", null],
      ["decided", "now", "Awaiting their decision", null],
      ["built", "ahead", "", null],
    ]);
  });
});

describe("tenderRoute", () => {
  it("lays out each stop with its window and whether its location is mapped", () => {
    const route = tenderRoute(transfer("mappingRequired"), undefined, t);
    expect(route.distance).toBe("271 mi");
    expect(route.stops).toEqual([
      {
        key: "0-1",
        role: "pick",
        kind: "Pickup",
        place: "Acme Plant 2",
        line: "4100 Irving Blvd, Dallas, TX 75247",
        window: `${cardMoment(NOW + 86_400)} – ${cardMomentAfter(NOW + 86_400, NOW + 86_400 + 7200)}`,
        load: "24 pieces · 42,000 lb",
        mapping: { state: "mapped", label: "Acme Plant 2 (Dallas)" },
      },
      {
        key: "0-2",
        role: "drop",
        kind: "Delivery",
        place: "HEB San Antonio",
        line: "San Antonio, TX",
        window: cardMoment(NOW + 2 * 86_400),
        load: "",
        mapping: { state: "missing", label: "Not mapped" },
      },
    ]);
  });

  it("reads a stop's mapping from the live preview when there is one", () => {
    const preview = mappingPreview();
    const location = preview.all.find((row) => row.sourceId === "HEB-SAT");
    if (location) {
      location.resolved = true;
      location.targetLabel = "HEB San Antonio DC";
    }
    const route = tenderRoute(transfer("mappingRequired"), preview, t);
    expect(route.stops[1]?.mapping).toEqual({ state: "mapped", label: "HEB San Antonio DC" });
  });
});

describe("tenderFreight and tenderCharges", () => {
  it("states what is moving and for whom", () => {
    expect(tenderFreight(transfer("approved"), t)).toEqual({
      facts: [
        { key: "customer", label: "Customer", value: "Acme Foods" },
        { key: "service", label: "Service", value: "Default service type for inbound tenders" },
        { key: "weight", label: "Weight", value: "42,000 lb" },
        { key: "pieces", label: "Pieces", value: "24" },
        { key: "temperature", label: "Temperature", value: "34–38 °F" },
      ],
      commodities: [{ key: "FROZEN-0", label: "Frozen poultry", load: "24 pieces · 42,000 lb" }],
    });
  });

  it("writes the rate as a ledger to its total", () => {
    expect(tenderCharges(transfer("approved"), t)).toEqual({
      lines: [
        { key: "freight", label: "Freight", amount: money("2650") },
        { key: "FSC-0", label: "Fuel surcharge", amount: money("230") },
      ],
      total: money("2880"),
    });
    const bare = transfer("approved");
    bare.tenderPayload.additionalCharges = [];
    bare.tenderPayload.freightChargeAmount = null;
    bare.tenderPayload.totalChargeAmount = null;
    expect(tenderCharges(bare, t)).toEqual({
      lines: [{ key: "other", label: "Other charges", amount: money("230") }],
      total: "",
    });
  });
});

describe("mappingLedger", () => {
  it("lists each partner code against the record it maps to, unmapped first", () => {
    const ledger = mappingLedger(transfer("mappingRequired"), undefined);
    expect(ledger.resolved).toBe(3);
    expect(ledger.total).toBe(5);
    expect(ledger.rows.map((row) => [row.entity, row.source, row.target, row.resolved])).toEqual([
      ["Location", "HEB San Antonio", "", false],
      ["Accessorial charge", "Fuel surcharge", "", false],
      ["Customer", "Acme Foods", "Acme Foods, Inc.", true],
      ["Location", "Acme Plant 2", "Acme Plant 2 (Dallas)", true],
      ["Commodity", "Frozen poultry", "Frozen poultry", true],
    ]);
  });

  it("prefers the live preview to the snapshot taken when the tender came in", () => {
    const preview = mappingPreview();
    preview.all = preview.all.map((row) => ({ ...row, resolved: true, targetLabel: "Mapped" }));
    expect(mappingLedger(transfer("mappingRequired"), preview).resolved).toBe(5);
  });
});

describe("transferLinks", () => {
  it("names the 204 a tender was read from and the shipment it built", () => {
    expect(transferLinks(transfer("approved"), "inbound", t)).toEqual([
      {
        key: "message",
        entity: "edi_message",
        id: "edimsg_acme_204a",
        title: "Load tender 204",
        line: "The EDI message it was read from",
      },
      {
        key: "built",
        entity: "shipment",
        id: "shp_acme_88240",
        title: "Shipment it built",
        line: "Created when the tender was accepted",
      },
    ]);
  });

  it("names the shipment a tender we sent was cut from", () => {
    expect(transferLinks(transfer("outbound"), "outbound", t)).toEqual([
      {
        key: "source",
        entity: "shipment",
        id: "shp_s2610_1366",
        title: "Shipment tendered",
        line: "Our shipment this tender offers",
      },
    ]);
  });
});

describe("reasonNote", () => {
  it("keeps the words written down where the tender closed", () => {
    expect(reasonNote(transfer("rejected"), t)).toEqual({
      tone: "muted",
      label: "Why it was declined",
      text: "No reefer capacity on the Dallas to Houston lane that day",
    });
    expect(reasonNote(transfer("failed"), t)).toEqual({
      tone: "bad",
      label: "Why the shipment was not created",
      text: "customer Acme Foods, Inc. has no billing profile",
    });
    expect(reasonNote(transfer("superseded"), t)).toEqual({
      tone: "muted",
      label: "Why it was canceled",
      text: "Superseded by inbound load tender change 0002",
    });
    expect(reasonNote(transfer("approved"), t)).toBeNull();
    expect(reasonNote(transfer("rejected", { rejectionReason: "" }), t)).toBeNull();
  });
});
