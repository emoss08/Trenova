import type { DeskEdiPartnerCard, EdiPartnerMessage } from "@/lib/graphql/edi-partner-card";
import { translate } from "@trenova/shared/i18n/runtime";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  ackRuler,
  attentionItems,
  certificateReading,
  connectionExchange,
  credentialChips,
  deliveryFigure,
  headFacts,
  mappingChips,
  messageRowLine,
  messageRowStatus,
  partnerAddress,
  partnerExchange,
  partnerKicker,
  partnerStanding,
  partnerSubtitle,
  partnerUsage,
  pulseDays,
  readinessChecks,
  transportEnds,
  transportFacts,
  transportLine,
  transportPolls,
} from "../edi-partner-card";
import acmeFixture from "./acme-trouble.json";
import krogerFixture from "./kroger-setup.json";
import lonestarFixture from "./lonestar-internal.json";
import tysonFixture from "./tyson-trading.json";

const t = translate as TranslateFn;
const DAY = 86_400;

const tyson = tysonFixture.card as unknown as DeskEdiPartnerCard;
const acme = acmeFixture.card as unknown as DeskEdiPartnerCard;
const kroger = krogerFixture.card as unknown as DeskEdiPartnerCard;
const lonestar = lonestarFixture.card as unknown as DeskEdiPartnerCard;
const NOW = tysonFixture.now;

function withCard(card: DeskEdiPartnerCard, change: (copy: DeskEdiPartnerCard) => void) {
  const copy = structuredClone(card);
  change(copy);
  return copy;
}

describe("partnerStanding", () => {
  it("reads a ready partner with nothing wrong as trading", () => {
    expect(partnerStanding(tyson, 0, t)).toEqual({
      label: "Trading",
      tone: "success",
      mark: "check",
    });
  });

  it("asks for a look when anything needs one", () => {
    expect(partnerStanding(acme, 4, t)).toEqual({
      label: "Needs a look",
      tone: "warn",
      mark: "dot",
    });
  });

  it("says a partner still being set up is not ready, whatever its traffic", () => {
    expect(partnerStanding(kroger, 0, t).label).toBe("Not ready");
  });

  it("puts an inactive partner before everything else", () => {
    const off = withCard(acme, (copy) => {
      copy.partner.status = "Inactive";
    });
    expect(partnerStanding(off, 4, t)).toEqual({ label: "Inactive", tone: "muted", mark: "pause" });
  });

  it("follows a Trenova-to-Trenova partner's connection", () => {
    expect(partnerStanding(lonestar, 0, t).label).toBe("Connected");
    const pending = withCard(lonestar, (copy) => {
      copy.partner.connection!.status = "PendingAcceptance";
    });
    expect(partnerStanding(pending, 0, t)).toEqual({
      label: "Waiting on them to accept",
      tone: "warn",
      mark: "dot",
    });
    const revoked = withCard(lonestar, (copy) => {
      copy.partner.connection!.status = "Revoked";
    });
    expect(partnerStanding(revoked, 0, t)).toEqual({ label: "Revoked", tone: "danger", mark: "x" });
  });
});

describe("the head", () => {
  it("names the code, the kind and the customer", () => {
    expect(partnerKicker(tyson, t)).toEqual(["TYSON", "External partner", "Customer Tyson Foods"]);
    expect(partnerKicker(lonestar, t)).toEqual(["LONESTAR", "Trenova to Trenova"]);
  });

  it("uses the partner's own description, else which ways documents travel", () => {
    expect(partnerSubtitle(tyson, t)).toBe(
      "Tenders loads to us and takes status updates and invoices back",
    );
    expect(partnerSubtitle(kroger, t)).toBe("Sends documents to us and takes ours back");
    const inboundOnly = withCard(kroger, (copy) => {
      copy.partner.enabledForOutbound = false;
    });
    expect(partnerSubtitle(inboundOnly, t)).toBe("Sends documents to us");
    expect(partnerSubtitle(lonestar, t)).toBe(
      "Another Trenova organization, Lone Star Logistics, that we trade with directly",
    );
  });

  it("reads production or test from the active Lading version's interchange", () => {
    expect(partnerUsage(tyson)).toBe("production");
    expect(partnerUsage(kroger)).toBe("test");
    expect(partnerUsage(lonestar)).toBeNull();
    const noVersion = withCard(tyson, (copy) => {
      copy.lading.active = null;
    });
    expect(partnerUsage(noVersion)).toBeNull();
  });

  it("lists production, the latest message and the line under the head", () => {
    expect(headFacts(tyson, NOW, t)).toEqual(["Production", "Last message 4m ago", "AS2"]);
    expect(headFacts(kroger, NOW, t)).toEqual(["Test", "Nothing sent or received in 7 days"]);
  });

  it("shows the delivered share, amber under 95%, and none without a delivery tried", () => {
    expect(deliveryFigure(tyson, t)).toEqual({
      value: "99.8%",
      caption: "delivered · last 7 days",
      warn: false,
    });
    expect(deliveryFigure(acme, t)).toMatchObject({ value: "91.4%", warn: true });
    expect(deliveryFigure(kroger, t)).toBeNull();
  });
});

describe("partnerExchange", () => {
  it("lays each set on its lane with its count and the answer it waits for", () => {
    const exchange = partnerExchange(tyson, t);
    expect(exchange.us).toEqual({ name: "Us", id: "ZZ · TRENOVA" });
    expect(exchange.them).toEqual({ name: "Tyson Foods", id: "ZZ · TYSONFOODS" });
    expect(
      exchange.outbound.map(({ set, name, note, state, count }) => ({
        set,
        name,
        note,
        state,
        count,
      })),
    ).toEqual([
      {
        set: "210",
        name: "Freight invoice",
        note: "Waits on a 997 within 1d",
        state: null,
        count: 96,
      },
      { set: "214", name: "Shipment status", note: "3 failed to send", state: "warn", count: 1284 },
      {
        set: "990",
        name: "Response to a tender",
        note: "No acknowledgment asked",
        state: null,
        count: 42,
      },
    ]);
    expect(exchange.inbound.map(({ set, note, count }) => ({ set, note, count }))).toEqual([
      { set: "204", note: "We send a 997 back", count: 44 },
      { set: "997", note: "Answers what we send", count: 1402 },
    ]);
  });

  it("puts a dead letter before a failure, in red", () => {
    const lane = partnerExchange(acme, t).outbound.find((set) => set.set === "210");
    expect(lane).toMatchObject({ note: "1 dead-lettered", state: "bad" });
  });

  it("marks a set still in test and leaves its count open while nothing has traded", () => {
    const exchange = partnerExchange(kroger, t);
    expect(exchange.outbound).toEqual([
      expect.objectContaining({
        set: "214",
        test: true,
        count: null,
        note: "Waits on a 997 within 2h",
      }),
    ]);
  });

  it("draws a Trenova-to-Trenova exchange from its connection, uncounted", () => {
    const exchange = connectionExchange(lonestar, t);
    expect(exchange.counted).toBe(false);
    expect(exchange.outbound.map((set) => set.set)).toEqual(["204", "214"]);
    expect(exchange.inbound.map((set) => set.note)).toEqual([
      "We accept or decline",
      "Straight to accounts payable",
    ]);
  });
});

describe("partnerAddress", () => {
  it("reads the envelope from us to them out of the active version's partner file", () => {
    expect(partnerAddress(kroger)).toMatchObject({
      us: { qualifier: "ZZ", id: "TRENOVA" },
      them: { qualifier: "ZZ", id: "KROGERTMS" },
      usage: "T",
    });
    expect(partnerAddress(tyson)?.usage).toBe("P");
  });

  it("has nothing to draw without an active version that names both ends", () => {
    expect(partnerAddress(lonestar)).toBeNull();
    const unnamed = withCard(tyson, (copy) => {
      if (copy.lading.active?.interchange) copy.lading.active.interchange.theirId = null;
    });
    expect(partnerAddress(unnamed)).toBeNull();
  });
});

describe("partnerExchange outside the active version", () => {
  it("marks a set that still has traffic but is no longer in the active version", () => {
    const dropped = withCard(tyson, (copy) => {
      if (copy.lading.active) {
        copy.lading.active.exchanges = copy.lading.active.exchanges.filter(
          (exchange) => exchange.transactionSet !== "990",
        );
      }
    });
    expect(partnerExchange(dropped, t).outbound.find((set) => set.set === "990")).toMatchObject({
      note: "Not in the active Lading version",
      off: true,
      test: false,
    });
  });
});

describe("pulseDays", () => {
  it("folds the hourly series into fourteen of the viewer's days, ending today", () => {
    const days = pulseDays(tyson.activity.series, NOW);
    expect(days).toHaveLength(14);
    expect(days.at(-1)?.today).toBe(true);
    expect(days.reduce((sum, day) => sum + day.sent, 0)).toBe(
      tyson.activity.series.reduce((sum, point) => sum + point.outboundCount, 0),
    );
    expect(days.reduce((sum, day) => sum + day.failed, 0)).toBe(3);
  });

  it("draws an empty day as an empty day", () => {
    const days = pulseDays([], NOW);
    expect(days.every((day) => day.sent === 0 && day.received === 0)).toBe(true);
  });
});

describe("ackRuler", () => {
  it("measures the turnaround against the shortest promise", () => {
    expect(ackRuler(tyson, t)).toMatchObject({
      average: 372,
      slowest: 2460,
      promise: 240 * 60,
      line: "Back in 6m on average, 41m at the slowest 5%",
    });
  });

  it("says when the slowest run past the promise", () => {
    expect(ackRuler(acme, t)?.line).toBe(
      "Back in 52m on average, but the slowest 5% take longer than the 4h promised",
    );
  });

  it("draws nothing for a partner with no turnaround and no promise", () => {
    expect(ackRuler(lonestar, t)).toBeNull();
  });
});

describe("attentionItems", () => {
  it("lists what wants a look, worst first, each with what it is about", () => {
    const items = attentionItems(acme, NOW, t);
    expect(items.map(({ key, state, title }) => ({ key, state, title }))).toEqual([
      { key: "poll:edicp_acme_sftp", state: "bad", title: "We can't pick up their files" },
      { key: "dead-lettered", state: "bad", title: "A message was dead-lettered" },
      {
        key: "overdue",
        state: "warn",
        title: "3 messages are waiting on an acknowledgment past 4h",
      },
      { key: "quarantined", state: "warn", title: "A file was set aside" },
    ]);
    expect(items[0].quote).toBe(
      "ssh: handshake failed: ssh: unable to authenticate, attempted methods [none publickey], no supported methods remain",
    );
    expect(items[0].when).toBe("2h ago");
    expect(items[2].body).toBe("The oldest has waited 6h 12m.");
    expect(items[2].messages?.map((message) => message.id)).toEqual([
      "edimsg_acme_214_a",
      "edimsg_acme_214_b",
      "edimsg_acme_214_c",
    ]);
  });

  it("finds nothing on a healthy partner", () => {
    expect(attentionItems(tyson, NOW, t)).toEqual([]);
  });

  it("flags a required credential missing and a certificate running out", () => {
    const card = withCard(tyson, (copy) => {
      copy.transports[0].credentials = copy.transports[0].credentials.map((credential) =>
        credential.key === "privateKey" ? { ...credential, onFile: false } : credential,
      );
      copy.transports[0].certificates[0].notAfter = NOW + 9 * DAY;
    });
    expect(attentionItems(card, NOW, t).map(({ state, title }) => ({ state, title }))).toEqual([
      { state: "bad", title: "Our private key is missing" },
      { state: "warn", title: "Their encryption certificate expires in 9 days" },
    ]);
  });

  it("does not count a pickup that has worked since it failed", () => {
    const recovered = withCard(acme, (copy) => {
      copy.transports[0].lastPollSuccessAt = NOW - 60;
      copy.transports[0].lastPollAttemptAt = NOW - 60;
    });
    expect(attentionItems(recovered, NOW, t).some((item) => item.key.startsWith("poll:"))).toBe(
      false,
    );
  });
});

describe("certificateReading", () => {
  const certificate = tyson.transports[0].certificates[0];

  it("reads a good certificate as good until its date, with the life it has used", () => {
    const reading = certificateReading(certificate, NOW, t);
    expect(reading.state).toBe("ok");
    expect(reading.title).toBe("Their encryption certificate");
    expect(reading.used).toBeCloseTo(160 / 390, 3);
  });

  it("reads an expired one as bad and one it cannot parse as a warning", () => {
    expect(
      certificateReading({ ...certificate, notAfter: NOW - DAY, expired: true }, NOW, t),
    ).toMatchObject({
      state: "bad",
      title: "Their encryption certificate has expired",
      used: 1,
    });
    expect(
      certificateReading({ ...certificate, readable: false, notAfter: null }, NOW, t),
    ).toMatchObject({ state: "warn", title: "Their encryption certificate can't be read" });
  });
});

describe("the line we reach them over", () => {
  const as2 = tyson.transports[0];
  const sftp = acme.transports[0];

  it("names each end of the line", () => {
    expect(transportEnds(as2, t)).toEqual({
      usLabel: "Our AS2 ID",
      us: "TRENOVA-AS2",
      themLabel: "Their AS2 ID",
      them: "TYSONFOODS-PROD",
    });
    expect(transportEnds(sftp, t)).toEqual({
      usLabel: "We sign in as",
      us: "trenova",
      themLabel: "Their server",
      them: "sftp.acmefoods.example:22",
    });
  });

  it("lists the settings a person checks it by", () => {
    expect(transportFacts(as2, NOW, t).map((fact) => [fact.label, fact.value])).toEqual([
      ["Endpoint", "https://edi.tyson.example/as2"],
      ["Receipts", "Signed receipt, right away"],
      ["Signing", "SHA-256"],
      ["Encryption", "AES-256-CBC"],
      ["Retries", "5 tries, 1m backing off to 30m"],
    ]);
    expect(transportFacts(sftp, NOW, t).map((fact) => [fact.label, fact.value])).toEqual([
      ["Signs in with", "Private key"],
      ["Drops files in", "/inbound"],
      ["Picks up from", "/outbound/tenders, then moves them to /outbound/archive"],
      ["Files named", "TRENOVA_{set}_{control}.edi"],
      ["Last pickup that worked", "2h ago"],
    ]);
  });

  it("names the credentials that matter, never certificates twice and never a value", () => {
    expect(credentialChips(as2, t)).toEqual([
      { key: "privateKey", label: "Our private key", onFile: true },
      { key: "basicAuthPassword", label: "Basic auth password", onFile: true },
    ]);
    expect(credentialChips(sftp, t)).toEqual([
      { key: "knownHostKey", label: "Pinned host key", onFile: true },
      { key: "password", label: "Password or private key", onFile: true },
    ]);
  });

  it("says how the line last went", () => {
    expect(transportLine(sftp, NOW, t)?.state).toBe("warn");
    expect(transportLine(as2, NOW, t)).toBeNull();
    expect(transportPolls(sftp)).toBe(true);
    expect(transportPolls(as2)).toBe(false);
  });
});

describe("readinessChecks", () => {
  it("lists each check in a fixed order, with what is missing for each one not done", () => {
    expect(readinessChecks(kroger, t)).toEqual([
      { key: "details", label: "Details", complete: true, note: "" },
      {
        key: "communication-profile",
        label: "Transport",
        complete: false,
        note: "No AS2, SFTP or VAN line is set up to reach them.",
      },
      {
        key: "mappings",
        label: "Mappings",
        complete: false,
        note: "None of their codes are matched to our records yet.",
      },
      { key: "inbound-document-profile", label: "Inbound sets", complete: true, note: "" },
      { key: "outbound-document-profile", label: "Outbound sets", complete: true, note: "" },
      {
        key: "test-case",
        label: "Test case",
        complete: false,
        note: "Keep one of their documents passing in the designer.",
      },
    ]);
  });
});

describe("mappingChips", () => {
  it("counts their codes by kind, most first, in whole words", () => {
    expect(mappingChips(tyson, t).map((chip) => chip.text)).toEqual([
      "41 locations",
      "12 accessorial charges",
      "9 failure reasons",
      "6 commodities",
      "3 service types",
      "2 customers",
    ]);
    expect(mappingChips(kroger, t)).toEqual([]);
  });
});

describe("listed messages", () => {
  const message = (overrides: Partial<EdiPartnerMessage>): EdiPartnerMessage => ({
    id: "edimsg_1",
    direction: "Outbound",
    transactionSet: "214",
    interchangeControlNumber: "000000417",
    status: "Generated",
    deliveryStatus: "Sent",
    deliveryAttempts: 1,
    deliveryLastAttemptAt: null,
    deliveryLastError: null,
    deliverySentAt: NOW,
    ackStatus: "Accepted",
    ackReceivedAt: NOW,
    ackLastError: null,
    generatedAt: NOW,
    ...overrides,
  });

  it("reads delivery before the acknowledgment, and an inbound one as received", () => {
    expect(messageRowStatus(message({}), t)).toEqual({ text: "Accepted", state: "ok" });
    expect(messageRowStatus(message({ deliveryStatus: "DeadLettered" }), t)).toEqual({
      text: "Dead-lettered",
      state: "bad",
    });
    expect(messageRowStatus(message({ ackStatus: "Pending" }), t)).toEqual({
      text: "Waiting on an acknowledgment",
      state: null,
    });
    expect(
      messageRowStatus(message({ direction: "Inbound", deliveryStatus: null, ackStatus: null }), t),
    ).toEqual({ text: "Received", state: null });
  });

  it("puts what went wrong beside the interchange number", () => {
    expect(messageRowLine(message({}), t)).toBe("Interchange 000000417");
    expect(
      messageRowLine(
        message({ deliveryStatus: "DeadLettered", deliveryLastError: "550 mailbox unavailable" }),
        t,
      ),
    ).toBe("Interchange 000000417 · 550 mailbox unavailable");
    expect(messageRowLine(message({ ackStatus: "Rejected", ackLastError: "AK5*R" }), t)).toBe(
      "Interchange 000000417 · AK5*R",
    );
  });
});
