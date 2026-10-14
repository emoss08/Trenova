import { translate } from "@trenova/shared/i18n/runtime";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  diagnosticTally,
  envelopeLayers,
  foldTape,
  journeyPosition,
  messageJourney,
  messageLinks,
  messageStanding,
  tapeSegments,
  wayAnchor,
  wayHalt,
} from "../edi-message-card";
import { plainReading } from "../../../record-card/edi-reading";
import { HOUR, NOW, ediInspection, ediMessage } from "./fixtures";

const t = translate as TranslateFn;

const stations = (overrides: Record<string, unknown>) =>
  messageJourney(ediMessage(overrides), t).map((station) => [
    station.key,
    station.state,
    station.detail,
  ]);

describe("messageStanding", () => {
  it("puts the worst news first: not generated, dead-lettered, failed, then rejected", () => {
    const standing = (overrides: Record<string, unknown>) =>
      messageStanding(ediMessage(overrides), t);

    expect(standing({ status: "Failed", deliveryStatus: "DeadLettered" })).toEqual({
      label: "Not generated",
      tone: "warn",
      mark: "pause",
    });
    expect(standing({ deliveryStatus: "DeadLettered", ackStatus: "Rejected" }).label).toBe(
      "Dead-lettered",
    );
    expect(standing({ deliveryStatus: "Failed" }).label).toBe("Delivery failed");
    expect(standing({ ackStatus: "Rejected" })).toEqual({
      label: "Rejected by partner",
      tone: "warn",
      mark: "pause",
    });
    expect(standing({})).toEqual({ label: "Accepted", tone: "success", mark: "check" });
  });

  it("reads a good outbound message by how far it has come", () => {
    const standing = (overrides: Record<string, unknown>) =>
      messageStanding(ediMessage(overrides), t).label;

    expect(standing({ deliveryStatus: "Queued", ackStatus: "Pending" })).toBe("Queued");
    expect(standing({ deliveryStatus: "Sending", ackStatus: "Pending" })).toBe("Sending");
    expect(standing({ ackStatus: "Pending" })).toBe("Awaiting acknowledgment");
    expect(standing({ ackStatus: "NotExpected" })).toBe("Delivered");
    expect(standing({ deliveryStatus: "", ackStatus: "" })).toBe("Generated");
  });

  it("reads a partner's acknowledgment by its verdict on what was sent", () => {
    const ack = (code: string) =>
      messageStanding(
        ediMessage({
          direction: "Inbound",
          transactionSet: "997",
          deliveryStatus: "",
          payloadSnapshot: { functionalAck: { transactionAcknowledgmentCode: code } },
        }),
        t,
      );
    expect(ack("R")).toEqual({ label: "Rejected our message", tone: "warn", mark: "pause" });
    expect(ack("E").label).toBe("Accepted with errors");
    expect(ack("P").label).toBe("Partly accepted");
    expect(ack("A")).toEqual({ label: "Received", tone: "success", mark: "check" });
    expect(
      messageStanding(
        ediMessage({
          direction: "Inbound",
          transactionSet: "999",
          deliveryStatus: "",
          payloadSnapshot: { implementationAck: { groupAcknowledgmentCode: "M" } },
        }),
        t,
      ).label,
    ).toBe("Rejected our message");
  });

  it("reads an inbound message as received, or as one that could not be read", () => {
    expect(messageStanding(ediMessage({ direction: "Inbound", deliveryStatus: "" }), t).label).toBe(
      "Received",
    );
    expect(messageStanding(ediMessage({ direction: "Inbound", status: "Failed" }), t).label).toBe(
      "Could not be read",
    );
  });
});

describe("messageJourney", () => {
  it("follows an outbound message from generated through delivered to acknowledged", () => {
    const journey = messageJourney(ediMessage(), t);
    expect(journey.map((station) => [station.key, station.state, station.detail])).toEqual([
      ["made", "done", "6 segments"],
      ["sent", "done", "On the first attempt"],
      ["answered", "done", "Accepted by the partner"],
    ]);
    expect(journey.map((station) => station.at)).toEqual([
      NOW - 2 * HOUR,
      NOW - 2 * HOUR + 30,
      NOW - 2 * HOUR + 300,
    ]);
    expect(journeyPosition(journey)).toBe(2);
  });

  it("stops the envelope where delivery failed, and counts the attempts", () => {
    const journey = messageJourney(
      ediMessage({ deliveryStatus: "DeadLettered", deliveryAttempts: 5, ackStatus: "Pending" }),
      t,
    );
    expect(journey.map((station) => [station.key, station.state, station.detail])).toEqual([
      ["made", "done", "6 segments"],
      ["sent", "failed", "Gave up after 5 attempts"],
      ["answered", "ahead", "Once it is delivered"],
    ]);
    expect(journeyPosition(journey)).toBe(1);
  });

  it("waits on the partner once delivered, and stands still while queued", () => {
    expect(stations({ ackStatus: "Pending", ackReceivedAt: null }).at(-1)).toEqual([
      "answered",
      "ahead",
      "Waiting on the partner",
    ]);
    expect(
      stations({ deliveryStatus: "Queued", deliveryAttempts: 0, ackStatus: "Pending" }),
    ).toEqual([
      ["made", "done", "6 segments"],
      ["sent", "ahead", "Queued to send"],
      ["answered", "ahead", "Once it is delivered"],
    ]);
    expect(stations({ deliveryStatus: "Sending", deliveryAttempts: 2 })[1]).toEqual([
      "sent",
      "now",
      "Sending, attempt 2",
    ]);
  });

  it("goes nowhere past a message that could not be generated", () => {
    expect(stations({ status: "Failed", deliveryStatus: "", ackStatus: "Pending" })).toEqual([
      ["made", "failed", "It could not be generated"],
      ["sent", "skipped", "Never sent"],
      ["answered", "skipped", "Nothing to acknowledge"],
    ]);
  });

  it("delivers inside Trenova to an internal partner, and skips the wire to one with no connection", () => {
    const internal = ediMessage({
      deliveryStatus: "",
      ackStatus: "NotExpected",
      partner: { ...ediMessage().partner, kind: "Internal" },
    });
    expect(messageJourney(internal, t)[1]).toMatchObject({
      state: "done",
      detail: "Inside Trenova",
    });
    expect(stations({ deliveryStatus: "", ackStatus: "NotExpected" })[1]).toEqual([
      "sent",
      "skipped",
      "Not sent over a connection",
    ]);
  });

  it("follows an inbound tender from the file it came in to the tender it became", () => {
    const inbound = {
      direction: "Inbound",
      transactionSet: "204",
      deliveryStatus: "",
      ackStatus: "NotExpected",
      inboundFileId: "ediinf_1",
    };
    expect(stations({ ...inbound, transferId: "edilt_1" })).toEqual([
      ["received", "done", "In an inbound file"],
      ["read", "done", "6 segments"],
      ["tender", "done", "Waiting in inbound tenders"],
    ]);
    expect(stations({ ...inbound, transferId: "", inboundFileId: "" })).toEqual([
      ["received", "done", "Sent to us directly"],
      ["read", "done", "6 segments"],
      ["tender", "skipped", "No tender made from it"],
    ]);
    expect(stations({ ...inbound, transactionSet: "214", status: "Failed" })).toEqual([
      ["received", "done", "In an inbound file"],
      ["read", "failed", "It could not be read"],
    ]);
  });
});

describe("wayHalt", () => {
  it("says why delivery gave up or failed, on which attempt and when, at the delivered stop", () => {
    expect(
      wayHalt(
        ediMessage({
          deliveryStatus: "DeadLettered",
          deliveryAttempts: 5,
          deliveryLastAttemptAt: NOW - HOUR,
          deliveryLastError: "  SFTP upload refused: 553 file name not allowed\n",
          ackStatus: "Pending",
        }),
        t,
      ),
    ).toEqual({
      station: "sent",
      label: "Why delivery gave up",
      attempt: 5,
      at: NOW - HOUR,
      text: "SFTP upload refused: 553 file name not allowed",
    });
    expect(
      wayHalt(
        ediMessage({
          deliveryStatus: "Failed",
          deliveryAttempts: 0,
          deliveryLastAttemptAt: null,
          deliveryLastError: "Connection reset",
          ackStatus: "Pending",
        }),
        t,
      ),
    ).toEqual({
      station: "sent",
      label: "Why the last attempt failed",
      attempt: null,
      at: null,
      text: "Connection reset",
    });
  });

  it("says why the partner rejected it, or why its answer could not be read, at the acknowledged stop", () => {
    expect(
      wayHalt(
        ediMessage({ ackStatus: "Rejected", ackLastError: "AK5*R*5", ackReceivedAt: NOW }),
        t,
      ),
    ).toEqual({
      station: "answered",
      label: "Why the partner rejected it",
      attempt: null,
      at: NOW,
      text: "AK5*R*5",
    });
    expect(
      wayHalt(ediMessage({ ackStatus: "Failed", ackLastError: "No AK1 segment" }), t)?.label,
    ).toBe("Why its acknowledgment could not be read");
  });

  it("has nothing to say of a message on its way, or of a failure that left no words", () => {
    expect(wayHalt(ediMessage(), t)).toBeNull();
    expect(
      wayHalt(
        ediMessage({
          deliveryStatus: "DeadLettered",
          deliveryLastError: "   ",
          ackStatus: "Pending",
        }),
        t,
      ),
    ).toBeNull();
    expect(
      wayHalt(ediMessage({ deliveryStatus: "Sent", deliveryLastError: "an old retry's error" }), t),
    ).toBeNull();
    expect(wayHalt(ediMessage({ ackStatus: "Accepted", ackLastError: "stale" }), t)).toBeNull();
  });
});

describe("wayAnchor", () => {
  it("rests on the start and end edges, and on the middle of any column between", () => {
    expect([0, 1, 2].map((index) => wayAnchor(index, 3))).toEqual([
      { edge: "start" },
      { share: 0.5 },
      { edge: "end" },
    ]);
    expect([1, 2].map((index) => wayAnchor(index, 4))).toEqual([
      { share: 0.375 },
      { share: 0.625 },
    ]);
    expect(wayAnchor(1, 2)).toEqual({ edge: "end" });
  });

  it("points nowhere for a stop that is not on the way", () => {
    expect(wayAnchor(-1, 3)).toBeNull();
    expect(wayAnchor(3, 3)).toBeNull();
  });
});

describe("envelopeLayers", () => {
  it("names each layer's control number, and closes each once inspected", () => {
    const layers = envelopeLayers(ediMessage(), ediInspection().inspection, t);
    expect(
      layers.map((layer) => [
        layer.pair,
        layer.control,
        layer.detail,
        layer.check,
        layer.checkText,
      ]),
    ).toEqual([
      ["ISA · IEA", "000000417", "X12 004010", "ok", "IEA closes it"],
      ["GS · GE", "417", "Group QM", "ok", "GE closes it"],
      ["ST · SE", "0001", "214 · 6 segments", "ok", "SE closes it"],
    ]);
  });

  it("says nothing of the trailers before the X12 is inspected", () => {
    expect(envelopeLayers(ediMessage(), undefined, t).map((layer) => layer.check)).toEqual([
      null,
      null,
      null,
    ]);
  });

  it("names the trailer that does not close its layer, and the count that disagrees", () => {
    const inspection = ediInspection({
      inspection: {
        envelope: {
          isaControlNumber: "000000417",
          ieaControlNumber: "000000418",
          expectedGroups: 1,
          actualGroups: 1,
        },
        groups: [
          {
            index: 0,
            functionalIdCode: "QM",
            gsControlNumber: "417",
            geControlNumber: "417",
            expectedCount: 2,
            actualCount: 1,
            startSegmentIndex: 1,
          },
        ],
        transactions: [
          {
            index: 0,
            stControlNumber: "0001",
            seControlNumber: "0001",
            expectedSegments: 7,
            actualSegments: 6,
            startSegmentIndex: 2,
          },
        ],
      },
    }).inspection;
    expect(
      envelopeLayers(ediMessage(), inspection, t).map((layer) => [layer.check, layer.checkText]),
    ).toEqual([
      ["bad", "IEA closes a different interchange"],
      ["bad", "GE counts 2 transactions, found 1"],
      ["bad", "SE counts 7 segments, found 6"],
    ]);

    const missing = ediInspection({
      inspection: {
        envelope: { isaControlNumber: "000000417", actualGroups: 0 },
        groups: [],
        transactions: [],
      },
    }).inspection;
    expect(
      envelopeLayers(ediMessage(), missing, t).map((layer) => [layer.check, layer.checkText]),
    ).toEqual([
      ["bad", "No IEA trailer found"],
      ["bad", "No functional group found"],
      ["bad", "No transaction set found"],
    ]);

    const open = ediInspection({
      inspection: {
        groups: [{ index: 0, gsControlNumber: "417", actualCount: 1, startSegmentIndex: 1 }],
        transactions: [
          { index: 0, stControlNumber: "0001", actualSegments: 6, startSegmentIndex: 2 },
        ],
      },
    }).inspection;
    expect(
      envelopeLayers(ediMessage(), open, t).map((layer) => [layer.check, layer.checkText]),
    ).toEqual([
      ["ok", "IEA closes it"],
      ["bad", "No GE trailer found"],
      ["bad", "No SE trailer found"],
    ]);
  });
});

describe("plainReading", () => {
  it("reads a 214 as its status at the stop, with the facts behind it", () => {
    const reading = plainReading(ediMessage(), t);
    expect(reading).toMatchObject({
      headline: "Arrived at delivery",
      line: "Tyson Foods DC, Springdale, AR 72762",
      state: "info",
    });
    expect(reading?.facts.map((fact) => [fact.label, fact.value])).toEqual([
      ["Status code", "X1 · reason NS"],
      ["Stop", "Delivery, stop 3"],
      ["PRO", "S2610-1366"],
      ["BOL", "BOL-55120"],
      ["Appointment", "APT-7781"],
      ["Equipment", "TRL-5521"],
    ]);
  });

  it("reads a late or delayed 214 as a warning, and a delivery as good news", () => {
    const status = ediMessage().payloadSnapshot?.shipmentStatus;
    const with214 = (shipmentStatus: Record<string, unknown>) =>
      plainReading(ediMessage({ payloadSnapshot: { transactionSet: "214", shipmentStatus } }), t);
    expect(with214({ ...status, statusCode: "SD", lateMinutes: 95 })).toMatchObject({
      headline: "Shipment delayed",
      state: "warn",
    });
    expect(with214({ ...status, statusCode: "SD", lateMinutes: 95 })?.facts).toContainEqual({
      key: "late",
      label: "Late by",
      value: "1h 35m",
    });
    expect(with214({ ...status, statusCode: "D1" })?.state).toBe("ok");
    expect(with214({ statusCode: "ZZ" })).toMatchObject({ headline: "Status ZZ", line: "" });
  });

  it("reads a 204 as its route, load and charge, and an update as an update", () => {
    const tender = {
      bol: "BOL-88",
      customerLabel: "Acme Foods",
      weight: 42000,
      pieces: 22,
      totalChargeAmount: "2880.00",
      moves: [
        {
          loaded: true,
          sequence: 1,
          stops: [
            {
              locationId: "loc_1",
              locationName: "Acme Plant 2",
              locationCity: "Dallas",
              locationStateCode: "TX",
              type: "Pickup",
              scheduleType: "Window",
              sequence: 1,
              scheduledWindowStart: NOW,
            },
            {
              locationId: "loc_2",
              locationName: "Acme DC",
              locationCity: "Houston",
              locationStateCode: "TX",
              type: "Delivery",
              scheduleType: "Window",
              sequence: 2,
            },
          ],
        },
      ],
    };
    const reading = plainReading(
      ediMessage({
        transactionSet: "204",
        payloadSnapshot: { transactionSet: "204", purposeCode: "04", loadTender: tender },
      }),
      t,
    );
    expect(reading).toMatchObject({
      headline: "Updated load tender",
      line: "Acme Plant 2, Dallas, TX to Acme DC, Houston, TX",
      figure: "$2,880.00",
    });
    expect(reading?.stops.map((stop) => [stop.kind, stop.place])).toEqual([
      ["Pickup", "Acme Plant 2, Dallas, TX"],
      ["Delivery", "Acme DC, Houston, TX"],
    ]);
    expect(reading?.facts.map((fact) => [fact.label, fact.value])).toEqual([
      ["BOL", "BOL-88"],
      ["Customer", "Acme Foods"],
      ["Weight", "42,000 lb"],
      ["Pieces", "22"],
    ]);
    expect(
      plainReading(ediMessage({ transactionSet: "204", payloadSnapshot: { shipment: tender } }), t)
        ?.headline,
    ).toBe("Load tender");
  });

  it("reads a 210 as its bill and every charge line", () => {
    const reading = plainReading(
      ediMessage({
        transactionSet: "210",
        payloadSnapshot: {
          transactionSet: "210",
          invoice: {
            invoiceNumber: "INV-2610-0042",
            billToName: "Acme Foods",
            billToCity: "Dallas",
            billToStateCode: "TX",
            currencyCode: "USD",
            totalAmount: "2930.50",
            proNumber: "S2610-1366",
            invoiceDate: Date.UTC(2026, 9, 12, 15) / 1000,
            lineCharges: [
              { sequence: 1, code: "400", description: "Line haul", amount: "2800.00" },
              { sequence: 2, code: "FSC", amount: "130.50" },
            ],
          },
        },
      }),
      t,
    );
    expect(reading).toMatchObject({
      headline: "Invoice INV-2610-0042",
      line: "Acme Foods, Dallas, TX",
      figure: "$2,930.50",
    });
    expect(reading?.facts.find((fact) => fact.key === "date")?.value).toBe("Oct 12, 2026");
    expect(reading?.lines.map((line) => [line.label, line.amount])).toEqual([
      ["Line haul", "$2,800.00"],
      ["FSC", "$130.50"],
    ]);
  });

  it("reads a 990 as accepted, declined with why, or only answered", () => {
    const respond = (tenderResponse: Record<string, unknown>) =>
      plainReading(ediMessage({ transactionSet: "990", payloadSnapshot: { tenderResponse } }), t);
    expect(respond({ responseCode: "A", bol: "BOL-88" })).toMatchObject({
      headline: "Accepted the tender",
      state: "ok",
    });
    expect(respond({ responseCode: "D", rejectionReason: "No capacity" })).toMatchObject({
      headline: "Declined the tender",
      line: "No capacity",
      state: "bad",
    });
    expect(respond({})?.headline).toBe("Answered the tender");
  });

  it("reads a 997 as its verdict on what it answers, and every fault it names", () => {
    const ack = (functionalAck: Record<string, unknown>) =>
      plainReading(ediMessage({ transactionSet: "997", payloadSnapshot: { functionalAck } }), t);
    const rejected = ack({
      originalTransactionSet: "214",
      originalTransactionControlNumber: "0001",
      originalGroupControlNumber: "417",
      transactionAcknowledgmentCode: "R",
      acceptedTransactionSetCount: 0,
      receivedTransactionSetCount: 1,
      diagnostics: [
        { segmentId: "AT7", elementPosition: 1, errorCode: "7", message: "Invalid code value" },
        { errorCode: "5" },
      ],
    });
    expect(rejected).toMatchObject({
      headline: "Rejected",
      line: "Answers 214 transaction 0001 in group 417",
      state: "bad",
      figure: "0 of 1 accepted",
    });
    expect(rejected?.faults.map((fault) => [fault.label, fault.amount])).toEqual([
      ["AT7, element 1", "Invalid code value · code 7"],
      ["Envelope", "code 5"],
    ]);
    expect(ack({ groupAcknowledgmentCode: "A" })).toMatchObject({
      headline: "Accepted",
      state: "ok",
    });
    expect(ack({ transactionAcknowledgmentCode: "E" })).toMatchObject({
      headline: "Accepted with errors",
      state: "warn",
    });
    expect(ack({ transactionAcknowledgmentCode: "P" })?.headline).toBe("Partly accepted");
  });

  it("has nothing to read when no payload was kept, or the payload holds another set", () => {
    expect(plainReading(ediMessage({ payloadSnapshot: null }), t)).toBeNull();
    expect(plainReading(ediMessage({ payloadSnapshot: { transactionSet: "214" } }), t)).toBeNull();
  });
});

describe("tapeSegments and foldTape", () => {
  it("lays the X12 out segment by segment, the envelope apart from the body", () => {
    const { rows, loose } = tapeSegments(ediInspection().inspection);
    expect(rows.map((row) => [row.id, row.role])).toEqual([
      ["ISA", "envelope"],
      ["GS", "envelope"],
      ["ST", "envelope"],
      ["B10", "body"],
      ["LX", "body"],
      ["AT7", "body"],
      ["MS1", "body"],
      ["SE", "envelope"],
      ["GE", "envelope"],
      ["IEA", "envelope"],
    ]);
    expect(rows[5]?.elements[0]).toEqual({
      position: 1,
      value: "X1",
      label: "Shipment Status Code",
      empty: false,
    });
    expect(loose).toEqual([]);
  });

  it("pins each diagnostic to the segment it names, and keeps one naming none apart", () => {
    const diagnostic = (overrides: Record<string, unknown>) => ({
      severity: "Error",
      code: "X12_CODE",
      source: "validation",
      message: "Invalid code value",
      ...overrides,
    });
    const { rows, loose } = tapeSegments(
      ediInspection({
        inspection: {
          diagnostics: [
            diagnostic({ segmentId: "AT7", segmentIndex: 5, elementPosition: 1 }),
            diagnostic({ segmentId: "MS1", segmentIndex: 2, severity: "Warning" }),
            diagnostic({ severity: "Warning", code: "SEPARATOR", message: "Mixed separators" }),
          ],
        },
      }).inspection,
    );
    expect(rows.flatMap((row) => row.marks.map((mark) => [row.id, mark.severity]))).toEqual([
      ["AT7", "Error"],
      ["MS1", "Warning"],
    ]);
    expect(loose.map((mark) => mark.code)).toEqual(["SEPARATOR"]);
  });

  it("folds the tape to its head and the marked segments past it, counting what it hides", () => {
    const { rows } = tapeSegments(
      ediInspection({
        inspection: {
          diagnostics: [
            {
              severity: "Error",
              code: "X",
              source: "validation",
              segmentId: "MS1",
              segmentIndex: 6,
              message: "Bad city",
            },
          ],
        },
      }).inspection,
    );
    const runs = foldTape(rows, 3, false).map((run) =>
      run.kind === "fold" ? `+${run.count}` : run.row.id,
    );
    expect(runs).toEqual(["ISA", "GS", "ST", "+3", "MS1", "+3"]);
    expect(foldTape(rows, 3, true)).toHaveLength(10);
    expect(foldTape(rows, 12, false)).toHaveLength(10);
  });

  it("tallies errors and warnings, and leaves information out", () => {
    expect(
      diagnosticTally([
        { severity: "Error" },
        { severity: "Warning" },
        { severity: "Info" },
        { severity: "Error" },
      ]),
    ).toEqual({ errors: 2, warnings: 1 });
  });
});

describe("messageLinks", () => {
  it("names the load by its PRO, and the acknowledgment that answered it", () => {
    expect(messageLinks(ediMessage(), t)).toEqual([
      {
        key: "shipment",
        entity: "shipment",
        id: "shp_1",
        title: "S2610-1366",
        line: "Shipment · BOL BOL-55120",
      },
      {
        key: "ack",
        entity: "edi_message",
        id: "edimsg_ack",
        title: "Its acknowledgment",
        line: expect.stringMatching(/^Received /u),
      },
    ]);
  });

  it("names the invoice, the message a 997 answers and the file a message arrived in", () => {
    const links = messageLinks(
      ediMessage({
        transactionSet: "997",
        direction: "Inbound",
        shipmentId: "",
        invoiceId: "inv_1",
        ackMessageId: "",
        inboundFileId: "ediinf_1",
        payloadSnapshot: {
          functionalAck: {
            sourceMessageId: "edimsg_1",
            originalTransactionSet: "214",
            originalTransactionControlNumber: "0001",
          },
          invoice: { invoiceNumber: "INV-9" },
        },
      }),
      t,
    );
    expect(links.map((link) => [link.key, link.entity, link.id, link.title, link.line])).toEqual([
      ["invoice", "invoice", "inv_1", "INV-9", "Invoice"],
      ["source", "edi_message", "edimsg_1", "The message it answers", "214 · ST 0001"],
      ["file", "edi_inbound_file", "ediinf_1", "The file it arrived in", "Inbound file"],
    ]);
  });
});
