import { translate } from "@trenova/shared/i18n/runtime";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatFileSize } from "@trenova/shared/lib/utils";
import { describe, expect, it } from "vitest";
import {
  checksumPrint,
  fileIntake,
  fileManifest,
  fileStanding,
  looseFindings,
  processingNotes,
  qualifierName,
  rawGlimpse,
  routingLabel,
  segmentRuler,
} from "../edi-inbound-file-card";
import { ediInboundFileInspectionSchema } from "@trenova/shared/types/edi";
import { fileInspection, inboundFile, wireFileInspection, wireInboundFile } from "./fixtures";

const t = translate as TranslateFn;

const intake = (name: Parameters<typeof inboundFile>[0], overrides = {}) =>
  fileIntake(inboundFile(name, overrides), t).map((stage) => [
    stage.key,
    stage.state,
    stage.detail,
    stage.at,
  ]);

const manifest = (
  file: ReturnType<typeof inboundFile>,
  inspection?: ReturnType<typeof fileInspection>,
) => fileManifest(file, inspection, processingNotes(file), t);

describe("fileStanding", () => {
  it("names each status with the tone it reads in", () => {
    const standing = (status: string) => fileStanding(inboundFile("simulator", { status }), t);
    expect(
      ["Received", "Parsed", "Processed", "PartiallyProcessed", "Quarantined", "Duplicate"].map(
        standing,
      ),
    ).toEqual([
      { label: "Received", tone: "progress", mark: "dot" },
      { label: "Handling its sets", tone: "progress", mark: "dot" },
      { label: "Processed", tone: "success", mark: "check" },
      { label: "Partly processed", tone: "warn", mark: "pause" },
      { label: "Quarantined", tone: "warn", mark: "pause" },
      { label: "Duplicate interchange", tone: "muted", mark: "dot" },
    ]);
  });
});

describe("fileIntake", () => {
  it("follows the simulator's AS2 file from receipt through routing", () => {
    expect(intake("simulator")).toEqual([
      ["received", "done", "Over AS2", 1_791_716_400],
      ["read", "done", "Interchange 000000042", null],
      ["routed", "done", "1 transaction set handled", 1_791_716_404],
    ]);
  });

  it("counts what is left to review on a file routed with notes", () => {
    expect(intake("batch")).toEqual([
      ["received", "done", "From the SFTP mailbox", 1_791_626_400],
      ["read", "done", "Interchange 000000317", null],
      ["routed", "partial", "3 notes to review", 1_791_626_404],
    ]);
  });

  it("stops at the envelope when a file was never X12", () => {
    expect(intake("quarantined")).toEqual([
      ["received", "done", "From the SFTP mailbox", 1_791_702_000],
      ["read", "failed", "Could not be read", null],
      ["routed", "skipped", "", null],
    ]);
  });

  it("fails at routing when a file was read but nothing in it could be routed", () => {
    expect(
      intake("batch", {
        status: "Quarantined",
        failureReason: "inbound file partner could not be resolved: not found",
      }),
    ).toEqual([
      ["received", "done", "From the SFTP mailbox", 1_791_626_400],
      ["read", "done", "Interchange 000000317", null],
      ["routed", "failed", "Nothing handled", 1_791_626_404],
    ]);
  });

  it("skips routing for an interchange already processed", () => {
    expect(intake("duplicate")).toEqual([
      ["received", "done", "Over AS2", 1_791_718_200],
      ["read", "done", "Interchange 000000042", null],
      ["routed", "skipped", "Already processed", null],
    ]);
  });

  it("shows where a file still being worked is", () => {
    expect(
      intake("simulator", {
        status: "Received",
        interchangeControlNumber: "",
        processedAt: null,
        transactionCount: 0,
      }),
    ).toEqual([
      ["received", "done", "Over AS2", 1_791_716_400],
      ["read", "now", "Reading the interchange", null],
      ["routed", "ahead", "", null],
    ]);
    expect(intake("batch", { status: "Parsed", processedAt: null, failureReason: "" })).toEqual([
      ["received", "done", "From the SFTP mailbox", 1_791_626_400],
      ["read", "done", "Interchange 000000317", null],
      ["routed", "now", "Handling 3 transaction sets", null],
    ]);
  });

  it("says how the file came for every way a file can come", () => {
    const came = (method: string) => fileIntake(inboundFile("simulator", { method }), t)[0]?.detail;
    expect(["AS2", "SFTP", "VAN", "Internal"].map(came)).toEqual([
      "Over AS2",
      "From the SFTP mailbox",
      "Through the VAN",
      "From another Trenova organization",
    ]);
  });
});

describe("processingNotes", () => {
  it("splits the joined warnings, keeping a warning's own semicolon inside it", () => {
    expect(processingNotes(inboundFile("batch"))).toEqual([
      {
        key: "note-0",
        text: "Load tender change 204/0002 did not match a pending tender for LT-88241; review manually",
        severity: "warn",
        set: "204",
        control: "0002",
      },
      {
        key: "note-1",
        text: "Load tender 204/0002 requires mapping before approval (1 unresolved entities)",
        severity: "warn",
        set: "204",
        control: "0002",
      },
      {
        key: "note-2",
        text: "Acknowledgment for control number 0001 could not be matched to an outbound message",
        severity: "warn",
        set: null,
        control: null,
      },
    ]);
  });

  it("marks failures as faults and ties each to the transaction it names", () => {
    const notes = processingNotes(
      inboundFile("batch", {
        failureReason:
          "transaction 204/0001 failed: tender LT-1 has no stops; " +
          "failed to generate 997 acknowledgment for 204/0001: partner has no FA profile",
      }),
    );
    expect(notes.map((note) => [note.severity, note.set, note.control, note.text])).toEqual([
      ["bad", "204", "0001", "Transaction 204/0001 failed: tender LT-1 has no stops"],
      [
        "bad",
        "204",
        "0001",
        "Failed to generate 997 acknowledgment for 204/0001: partner has no FA profile",
      ],
    ]);
  });

  it("reads a quarantine reason as a fault and a duplicate's as a fact", () => {
    expect(processingNotes(inboundFile("quarantined"))).toEqual([
      {
        key: "note-0",
        text: "Inbound file does not start with an ISA interchange header",
        severity: "bad",
        set: null,
        control: null,
      },
    ]);
    expect(processingNotes(inboundFile("duplicate")).map((note) => note.severity)).toEqual([
      "info",
    ]);
    expect(processingNotes(inboundFile("simulator"))).toEqual([]);
  });
});

describe("routingLabel", () => {
  it("addresses the simulator's file from its ISA, naming the AS2 message it came in", () => {
    expect(routingLabel(inboundFile("simulator"), t)).toEqual({
      from: {
        qualifier: "ZZ",
        qualifierName: "Mutually defined",
        id: "SIMPARTNER",
        name: "EDI Partner Simulator",
      },
      to: { qualifier: "ZZ", qualifierName: "Mutually defined", id: "TRENOVA", name: "" },
      interchange: "000000042",
      channel: "Simulator AS2",
      way: { label: "AS2 message ID", value: "<sim-1791716400000000000@edi-partner-sim>" },
      fingerprint: "9f2c 1a7b 4e8d 3f60",
      size: formatFileSize(494),
    });
  });

  it("names the path a mailbox file was picked up from, and blanks what was never read", () => {
    const label = routingLabel(inboundFile("quarantined"), t);
    expect(label.way).toEqual({ label: "Path", value: "/inbound/acme/tenders-week-42.txt" });
    expect(label.from).toEqual({
      qualifier: "",
      qualifierName: "",
      id: "",
      name: "Acme Foods",
    });
    expect(label.interchange).toBe("");
  });
});

describe("qualifierName", () => {
  it("names the ISA qualifiers partners use, and numbers any other", () => {
    expect(
      ["01", "02", "08", "12", "14", "ZZ", "30", ""].map((code) => qualifierName(code, t)),
    ).toEqual([
      "D-U-N-S number",
      "SCAC",
      "UCC communications ID",
      "Phone number",
      "D-U-N-S with suffix",
      "Mutually defined",
      "Qualifier 30",
      "",
    ]);
  });
});

describe("checksumPrint", () => {
  it("prints the first sixteen hex digits of the checksum in fours", () => {
    expect(checksumPrint("4D1E9A0C7B3F2E8D6A5C")).toBe("4d1e 9a0c 7b3f 2e8d");
    expect(checksumPrint("abc")).toBe("abc");
    expect(checksumPrint("")).toBe("");
  });
});

describe("fileManifest", () => {
  it("lists the batch group by group, each set tied to the message it was kept as", () => {
    const groups = manifest(inboundFile("batch"), fileInspection("batch"));
    expect(
      groups.map((group) => [group.code, group.name, group.control, group.countCheck]),
    ).toEqual([
      ["SM", "Load tenders", "317", { expected: 2, actual: 2, ok: true }],
      ["FA", "Acknowledgments", "318", { expected: 1, actual: 1, ok: true }],
    ]);
    expect(
      groups.flatMap((group) =>
        group.entries.map((entry) => [
          entry.set,
          entry.name,
          entry.control,
          entry.segments,
          entry.message?.id ?? null,
          entry.state,
          entry.verdict,
          entry.gist,
        ]),
      ),
    ).toEqual([
      [
        "204",
        "Load tender",
        "0001",
        10,
        "edimsg_acme_204a",
        "routed",
        "Handled",
        "Acme Foods DC to Kroger Houston",
      ],
      [
        "204",
        "Load tender",
        "0002",
        9,
        "edimsg_acme_204b",
        "noted",
        "Handled with notes",
        "Updated load tender · Acme Foods DC to HEB San Antonio",
      ],
      [
        "997",
        "Functional acknowledgment",
        "0001",
        6,
        "edimsg_acme_997",
        "routed",
        "Handled",
        "Accepted · Answers 210 transaction 0001 in group 433",
      ],
    ]);
  });

  it("pins the notes and findings on the transaction they belong to", () => {
    const [tenders, acks] = manifest(inboundFile("batch"), fileInspection("batch"));
    const changed = tenders?.entries[1];
    expect(changed?.notes.map((note) => note.key)).toEqual(["note-0", "note-1"]);
    expect(changed?.findings.map((finding) => finding.code)).toEqual(["x12.count.se01_mismatch"]);
    expect(changed?.countCheck).toEqual({ expected: 7, actual: 9, ok: false });
    expect(tenders?.entries[0]?.countCheck).toEqual({ expected: 10, actual: 10, ok: true });
    expect(acks?.entries[0]?.notes).toEqual([]);
  });

  it("matches a message kept under the first group's number before the group fix", () => {
    const file = inboundFile("batch");
    const legacy = {
      ...file,
      messages: file.messages?.map((message) =>
        message.transactionSet === "997" ? { ...message, groupControlNumber: "317" } : message,
      ),
    };
    const ids = manifest(legacy, fileInspection("batch")).flatMap((group) =>
      group.entries.map((entry) => entry.message?.id ?? null),
    );
    expect(ids).toEqual(["edimsg_acme_204a", "edimsg_acme_204b", "edimsg_acme_997"]);
  });

  it("marks a set with no kept message by why it has none", () => {
    const verdicts = (overrides: Record<string, unknown>) =>
      manifest(inboundFile("simulator", overrides), fileInspection("simulator")).flatMap((group) =>
        group.entries.map((entry) => [entry.state, entry.verdict]),
      );
    expect(verdicts({ messages: [] })).toEqual([["skipped", "Not processed"]]);
    expect(verdicts({ messages: [], status: "Parsed", processedAt: null })).toEqual([
      ["pending", "Waiting"],
    ]);
    expect(
      verdicts({
        messages: [],
        status: "Quarantined",
        failureReason: "transaction 204/0001 failed: document type 204 is not configured",
      }),
    ).toEqual([["failed", "Not handled"]]);
    expect(
      manifest(inboundFile("duplicate"), fileInspection("simulator"))[0]?.entries[0]?.verdict,
    ).toBe("Not processed");
  });

  it("lists the kept messages by group when the raw file is gone", () => {
    const groups = manifest(inboundFile("batch", { rawContent: "", rawPurgedAt: 1_791_700_000 }));
    expect(
      groups.map((group) => [
        group.code,
        group.control,
        group.countCheck,
        group.entries.map((entry) => [entry.control, entry.segments, entry.countCheck]),
      ]),
    ).toEqual([
      [
        "SM",
        "317",
        null,
        [
          ["0001", 10, null],
          ["0002", 9, null],
        ],
      ],
      ["FA", "318", null, [["0001", 6, null]]],
    ]);
  });

  it("lists nothing for a file that held no interchange", () => {
    expect(manifest(inboundFile("quarantined"), fileInspection("quarantined"))).toEqual([]);
  });
});

describe("segmentRuler", () => {
  it("lays the interchange out as runs of envelope and transaction, flagging the bad one", () => {
    expect(
      segmentRuler(fileInspection("batch")).map((run) => [
        run.kind,
        run.start,
        run.span,
        run.group,
        run.state,
      ]),
    ).toEqual([
      ["envelope", 1, 2, null, "ok"],
      ["transaction", 3, 10, 1, "ok"],
      ["transaction", 13, 9, 1, "bad"],
      ["envelope", 22, 2, null, "ok"],
      ["transaction", 24, 6, 2, "ok"],
      ["envelope", 30, 2, null, "ok"],
    ]);
  });
});

describe("looseFindings", () => {
  it("keeps the findings no transaction holds", () => {
    expect(looseFindings(fileInspection("batch"))).toEqual([]);
    expect(looseFindings(fileInspection("quarantined")).map((finding) => finding.code)).toEqual([
      "x12.separator.fallback",
      "x12.segment.id_malformed",
      "x12.segment.missing_terminator",
      "x12.envelope.missing_isa",
      "x12.envelope.missing_gs",
      "x12.envelope.missing_st",
      "x12.envelope.missing_se",
      "x12.envelope.missing_ge",
      "x12.envelope.missing_iea",
    ]);
  });
});

describe("rawGlimpse", () => {
  it("shows the first lines of a file that is not X12 and counts the rest", () => {
    expect(rawGlimpse(inboundFile("quarantined").rawContent ?? "", 2)).toEqual({
      lines: ["Hello from the Acme EDI team,", "please find this week's tenders attached."],
      more: 1,
    });
  });

  it("cuts a long line and drops the trailing blank one", () => {
    expect(rawGlimpse(`${"x".repeat(200)}\r\n`, 6, 10)).toEqual({
      lines: ["xxxxxxxxxx…"],
      more: 0,
    });
  });
});

describe("a batch of many sets", () => {
  // Some partners put every set in a group of its own and number each ST 0001,
  // so a set is told from the others only by its group. Shaped from the Go
  // inspection of Acme's batch, repeated.
  const SETS = 400;
  const wire = wireFileInspection("batch") as {
    groups: Record<string, unknown>[];
    transactions: Record<string, unknown>[];
    diagnostics: Record<string, unknown>[];
    summary: Record<string, unknown>;
  };
  const gsControl = (index: number) => String(1000 + index);
  const inspection = ediInboundFileInspectionSchema.parse({
    ...wire,
    groups: Array.from({ length: SETS }, (_, index) => ({
      ...wire.groups[0],
      index: index + 1,
      gsControlNumber: gsControl(index),
      geControlNumber: gsControl(index),
      expectedCount: 1,
      actualCount: 1,
      startSegmentIndex: 2 + index * 12,
      endSegmentIndex: 13 + index * 12,
    })),
    transactions: Array.from({ length: SETS }, (_, index) => ({
      ...wire.transactions[0],
      index: index + 1,
      stControlNumber: "0001",
      seControlNumber: "0001",
      startSegmentIndex: 3 + index * 12,
      endSegmentIndex: 12 + index * 12,
    })),
    diagnostics: Array.from({ length: SETS }, (_, index) => ({
      ...wire.diagnostics[0],
      segmentIndex: 12 + index * 12,
      message: `SE01 of set ${index}`,
    })),
    summary: {
      ...wire.summary,
      segmentCount: 3 + SETS * 12,
      groupCount: SETS,
      transactionCount: SETS,
    },
  });
  const [kept] = (wireInboundFile("batch") as { messages: Record<string, unknown>[] }).messages;
  const file = inboundFile("batch", {
    messages: Array.from({ length: SETS }, (_, index) => ({
      ...kept,
      id: `edimsg_set_${index}`,
      transactionControlNumber: "0001",
      groupControlNumber: gsControl(SETS - 1 - index),
    })),
  });

  it("files each set under its own group with the message kept from that group and its own finding", () => {
    const groups = manifest(file, inspection);

    expect(groups).toHaveLength(SETS);
    expect(
      groups.map((group) => [
        group.control,
        group.entries.map((entry) => [entry.message?.id, entry.findings.map((f) => f.message)]),
      ]),
    ).toEqual(
      Array.from({ length: SETS }, (_, index) => [
        gsControl(index),
        [[`edimsg_set_${SETS - 1 - index}`, [`SE01 of set ${index}`]]],
      ]),
    );
    expect(looseFindings(inspection)).toEqual([]);
  });

  it("shades each set on the ruler by its own group and flags it by its own finding", () => {
    const sets = segmentRuler(inspection).filter((run) => run.kind === "transaction");

    expect(sets.map((run) => [run.group, run.state])).toEqual(
      Array.from({ length: SETS }, (_, index) => [index + 1, "bad"]),
    );
  });
});
