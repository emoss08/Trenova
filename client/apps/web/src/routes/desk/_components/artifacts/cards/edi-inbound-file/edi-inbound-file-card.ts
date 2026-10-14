import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatFileSize, upperFirst } from "@trenova/shared/lib/utils";
import type {
  EDIInboundFile,
  EDIInspectionDiagnostic,
  EDIMessage,
  EDIX12Inspection,
} from "@trenova/shared/types/edi";
import { plainReading } from "../../record-card/edi-reading";
import { functionalGroupName, transactionSetName } from "../../record-card/edi-transaction-sets";
import type { CardTone } from "../../record-card/record-card";

/** Where the file stands, as the pill at the card's head says it. */
export type FileStanding = {
  label: string;
  tone: CardTone;
  mark: "dot" | "check" | "pause";
};

/** Where the file stands: still being worked, done, done with notes, held, or turned away. */
export function fileStanding(file: EDIInboundFile, t: TranslateFn): FileStanding {
  switch (file.status) {
    case "Received":
      return { label: t("Received"), tone: "progress", mark: "dot" };
    case "Parsed":
      return { label: t("Handling its sets"), tone: "progress", mark: "dot" };
    case "Processed":
      return { label: t("Processed"), tone: "success", mark: "check" };
    case "PartiallyProcessed":
      return { label: t("Partly processed"), tone: "warn", mark: "pause" };
    case "Quarantined":
      return { label: t("Quarantined"), tone: "warn", mark: "pause" };
    case "Duplicate":
      return { label: t("Duplicate interchange"), tone: "muted", mark: "dot" };
  }
}

/** How far a stage of the file's intake came. */
export type IntakeState = "done" | "now" | "partial" | "failed" | "ahead" | "skipped";

/** One stage of the file's intake: received, its envelope read, its transaction sets routed. */
export type IntakeStage = {
  key: "received" | "read" | "routed";
  label: string;
  detail: string;
  state: IntakeState;
  at: number | null;
};

/** Whether the inbound parser read the file's ISA envelope. */
function envelopeRead(file: EDIInboundFile): boolean {
  return Boolean(file.interchangeControlNumber?.trim());
}

function arrival(file: EDIInboundFile, t: TranslateFn): string {
  switch (file.method) {
    case "AS2":
      return t("Over AS2");
    case "SFTP":
      return t("From the SFTP mailbox");
    case "VAN":
      return t("Through the VAN");
    case "Internal":
      return t("From another Trenova organization");
  }
}

/**
 * The file's intake as three stages: it was received, its interchange
 * envelope was read, and its transaction sets were routed to tenders,
 * statuses, invoices and acknowledgments. A file that is not X12 fails at the
 * envelope; one that was read but routed nothing fails at routing; a
 * duplicate is read and never routed again.
 */
export function fileIntake(file: EDIInboundFile, t: TranslateFn): IntakeStage[] {
  const read = envelopeRead(file);
  const received: IntakeStage = {
    key: "received",
    label: t("Received"),
    detail: arrival(file, t),
    state: "done",
    at: file.receivedAt,
  };
  const envelope: IntakeStage = {
    key: "read",
    label: t("Envelope read"),
    detail: read ? t("Interchange {0}", file.interchangeControlNumber ?? "") : "",
    state: "done",
    at: null,
  };
  const routed: IntakeStage = {
    key: "routed",
    label: t("Sets handled"),
    detail: "",
    state: "ahead",
    at: null,
  };

  switch (file.status) {
    case "Received":
      return [
        received,
        { ...envelope, state: "now", detail: t("Reading the interchange") },
        routed,
      ];
    case "Parsed":
      return [
        received,
        envelope,
        {
          ...routed,
          state: "now",
          detail: t(
            "Handling {0, plural, one {# transaction set} other {# transaction sets}}",
            file.transactionCount,
          ),
        },
      ];
    case "Processed":
      return [
        received,
        envelope,
        {
          ...routed,
          state: "done",
          at: file.processedAt ?? null,
          detail: t(
            "{0, plural, one {# transaction set handled} other {# transaction sets handled}}",
            file.transactionCount,
          ),
        },
      ];
    case "PartiallyProcessed":
      return [
        received,
        envelope,
        {
          ...routed,
          state: "partial",
          at: file.processedAt ?? null,
          detail: t(
            "{0, plural, one {# note to review} other {# notes to review}}",
            processingNotes(file).length,
          ),
        },
      ];
    case "Quarantined":
      return read
        ? [
            received,
            envelope,
            {
              ...routed,
              state: "failed",
              at: file.processedAt ?? null,
              detail: t("Nothing handled"),
            },
          ]
        : [
            received,
            { ...envelope, state: "failed", detail: t("Could not be read") },
            { ...routed, state: "skipped" },
          ];
    case "Duplicate":
      return [received, envelope, { ...routed, state: "skipped", detail: t("Already processed") }];
  }
}

/** One thing processing said of the file, tied to the transaction set it names when it names one. */
export type ProcessingNote = {
  key: string;
  text: string;
  severity: "bad" | "warn" | "info";
  set: string | null;
  control: string | null;
};

/**
 * How each warning the inbound service writes begins. The service joins its
 * warnings with "; ", and a warning can carry "; " of its own ("…for ACME;
 * review manually"), so a fragment that does not open with one of these
 * continues the warning before it.
 */
const NOTE_LEADS = [
  "acknowledgment ",
  "carrier invoice ",
  "failed to ",
  "freight invoice ",
  "load tender ",
  "shipment status ",
  "tender response ",
  "transaction ",
] as const;

const TRANSACTION_REF = /\b(\d{3})\/([A-Za-z0-9]+)\b/;
const FAILED = /^transaction \S+ failed: |^failed to /;

/**
 * What processing said of the file, one note per warning: a failure as a
 * fault, a warning as something to review, and the reason a duplicate was
 * turned away as a plain fact. A quarantined file's reason is a fault.
 */
export function processingNotes(file: EDIInboundFile): ProcessingNote[] {
  const reason = file.failureReason?.trim() ?? "";
  if (!reason) return [];

  const notes = reason.split("; ").reduce<string[]>((joined, fragment) => {
    const previous = joined.at(-1);
    if (previous !== undefined && !NOTE_LEADS.some((lead) => fragment.startsWith(lead))) {
      joined[joined.length - 1] = `${previous}; ${fragment}`;
    } else {
      joined.push(fragment);
    }
    return joined;
  }, []);

  return notes.map((note, index) => {
    const ref = TRANSACTION_REF.exec(note);
    return {
      key: `note-${index}`,
      text: upperFirst(note),
      severity:
        file.status === "Duplicate"
          ? "info"
          : file.status === "Quarantined" || FAILED.test(note)
            ? "bad"
            : "warn",
      set: ref?.[1] ?? null,
      control: ref?.[2] ?? null,
    };
  });
}

/** What an ISA interchange ID qualifier says the ID beside it is. */
export function qualifierName(code: string, t: TranslateFn): string {
  switch (code) {
    case "":
      return "";
    case "01":
      return t("D-U-N-S number");
    case "02":
      return t("SCAC");
    case "08":
      return t("UCC communications ID");
    case "12":
      return t("Phone number");
    case "14":
      return t("D-U-N-S with suffix");
    case "ZZ":
      return t("Mutually defined");
    default:
      return t("Qualifier {0}", code);
  }
}

/** The checksum as a short fingerprint a person can compare by eye. */
export function checksumPrint(checksum: string): string {
  return (
    checksum
      .trim()
      .toLowerCase()
      .slice(0, 16)
      .match(/.{1,4}/g)
      ?.join(" ") ?? ""
  );
}

/** One side of the interchange's address, as its ISA names it. */
export type RoutingParty = { qualifier: string; qualifierName: string; id: string; name: string };

/** The file's routing label: who sent it to whom, how, and the file itself. */
export type RoutingLabel = {
  from: RoutingParty;
  to: RoutingParty;
  interchange: string;
  channel: string;
  way: { label: string; value: string };
  fingerprint: string;
  size: string;
};

const AS2_PATH = "as2:";

/**
 * The file's routing label: the sender and receiver its ISA names, under the
 * partner it was received for; the interchange control number; the channel it
 * came in on and the AS2 message or mailbox path it came as; and the file's
 * fingerprint and size. A file never read as X12 has no sender or receiver.
 */
export function routingLabel(file: EDIInboundFile, t: TranslateFn): RoutingLabel {
  const party = (qualifier: string | null | undefined, id: string | null | undefined) => {
    const code = qualifier?.trim() ?? "";
    return { qualifier: code, qualifierName: qualifierName(code, t), id: id?.trim() ?? "" };
  };
  const as2 = file.method === "AS2" && file.remotePath.startsWith(AS2_PATH);
  return {
    from: {
      ...party(file.isaSenderQualifier, file.isaSenderId),
      name: file.partner?.name ?? "",
    },
    to: { ...party(file.isaReceiverQualifier, file.isaReceiverId), name: "" },
    interchange: file.interchangeControlNumber?.trim() ?? "",
    channel: file.communicationProfile?.name ?? "",
    way: as2
      ? { label: t("AS2 message ID"), value: file.remotePath.slice(AS2_PATH.length) }
      : { label: t("Path"), value: file.remotePath },
    fingerprint: checksumPrint(file.checksum),
    size: formatFileSize(file.sizeBytes),
  };
}

/** Whether a count in the file matches what its trailer declared. */
export type CountCheck = { expected: number; actual: number; ok: boolean };

/** What became of one transaction set in the file. */
export type EntryState = "routed" | "noted" | "failed" | "pending" | "skipped";

/** One transaction set the file carried, with the message it was kept as. */
export type ManifestEntry = {
  key: string;
  set: string;
  name: string;
  control: string;
  segments: number;
  /** Its SE01 against the segments counted; null when only the kept message is left to read. */
  countCheck: CountCheck | null;
  message: EDIMessage | null;
  state: EntryState;
  verdict: string;
  /** What it says, in a line: where a tender goes, what an acknowledgment answers. */
  gist: string;
  notes: ProcessingNote[];
  findings: EDIInspectionDiagnostic[];
};

/** One functional group the file carried and the transaction sets inside it. */
export type ManifestGroup = {
  key: string;
  code: string;
  name: string;
  control: string;
  /** Its GE01 against the sets counted; null when only the kept messages are left to read. */
  countCheck: CountCheck | null;
  entries: ManifestEntry[];
};

const SET_GROUP: Record<EDIMessage["transactionSet"], string> = {
  "204": "SM",
  "210": "IM",
  "214": "QM",
  "990": "GF",
  "997": "FA",
  "999": "FA",
};

function countCheck(expected: number, actual: number): CountCheck {
  return { expected, actual, ok: expected === actual };
}

type InspectedTransaction = EDIX12Inspection["transactions"][number];
type InspectedGroup = EDIX12Inspection["groups"][number];

function holds(group: InspectedGroup, transaction: InspectedTransaction): boolean {
  return (
    transaction.startSegmentIndex > group.startSegmentIndex &&
    (group.endSegmentIndex === 0 || transaction.startSegmentIndex < group.endSegmentIndex)
  );
}

function within(transaction: InspectedTransaction, segmentIndex: number): boolean {
  return (
    segmentIndex >= transaction.startSegmentIndex &&
    (transaction.endSegmentIndex === 0 || segmentIndex <= transaction.endSegmentIndex)
  );
}

function entryVerdict(
  file: EDIInboundFile,
  message: EDIMessage | null,
  notes: readonly ProcessingNote[],
  t: TranslateFn,
): { state: EntryState; verdict: string } {
  const failed = notes.some((note) => note.severity === "bad");
  if (failed) return { state: "failed", verdict: t("Not handled") };
  if (file.status === "Received" || file.status === "Parsed") {
    return { state: "pending", verdict: t("Waiting") };
  }
  if (!message) return { state: "skipped", verdict: t("Not processed") };
  if (notes.length > 0) return { state: "noted", verdict: t("Handled with notes") };
  return { state: "routed", verdict: t("Handled") };
}

function gistOf(message: EDIMessage | null, name: string, t: TranslateFn): string {
  const reading = message ? plainReading(message, t) : null;
  if (!reading) return "";
  return [reading.headline === name ? "" : reading.headline, reading.line]
    .filter(Boolean)
    .join(" · ");
}

function entryFor(
  file: EDIInboundFile,
  notes: readonly ProcessingNote[],
  base: {
    key: string;
    set: string;
    control: string;
    segments: number;
    countCheck: CountCheck | null;
    message: EDIMessage | null;
    findings: EDIInspectionDiagnostic[];
  },
  t: TranslateFn,
): ManifestEntry {
  const name = transactionSetName(base.set, t);
  const own = notes.filter((note) => note.set === base.set && note.control === base.control);
  return {
    ...base,
    name,
    ...entryVerdict(file, base.message, own, t),
    gist: gistOf(base.message, name, t),
    notes: own,
  };
}

/**
 * The messages kept from the file, claimed one transaction set at a time: by
 * set, control number and group, or, for a message kept before each set was
 * filed under its own group, by set and control number alone.
 */
function messageClaimer(messages: readonly EDIMessage[]) {
  const unclaimed = new Set(messages);
  return (set: string, control: string, group: string): EDIMessage | null => {
    const candidates = [...unclaimed].filter(
      (message) =>
        message.transactionSet === set && message.transactionControlNumber.trim() === control,
    );
    const match =
      candidates.find((message) => message.groupControlNumber.trim() === group) ?? candidates[0];
    if (!match) return null;
    unclaimed.delete(match);
    return match;
  };
}

/**
 * What the file carried, group by group: each transaction set with its
 * control number and segment count, whether its trailer counts it right, the
 * message it was kept as, what became of it, and the notes and findings that
 * name it. Read from the file's own interchange when it is kept; from the
 * messages kept from it when the raw file was purged.
 */
export function fileManifest(
  file: EDIInboundFile,
  inspection: EDIX12Inspection | undefined,
  notes: readonly ProcessingNote[],
  t: TranslateFn,
): ManifestGroup[] {
  const messages = file.messages ?? [];
  if (!inspection) return keptManifest(file, messages, notes, t);

  const claim = messageClaimer(messages);
  const findingsIn = (transaction: InspectedTransaction) =>
    inspection.diagnostics.filter(
      (finding) => finding.segmentIndex > 0 && within(transaction, finding.segmentIndex),
    );
  const groups: { group: InspectedGroup | null; transactions: InspectedTransaction[] }[] =
    inspection.groups.map((group) => ({ group, transactions: [] }));
  const loose: InspectedTransaction[] = [];
  for (const transaction of inspection.transactions) {
    const home = groups.find(({ group }) => group !== null && holds(group, transaction));
    (home?.transactions ?? loose).push(transaction);
  }
  if (loose.length > 0) groups.push({ group: null, transactions: loose });

  return groups
    .filter(({ transactions }) => transactions.length > 0)
    .map(({ group, transactions }) => {
      const code = group?.functionalIdCode?.trim() ?? "";
      const control = group?.gsControlNumber?.trim() ?? "";
      return {
        key: group ? `group-${group.index}` : "group-none",
        code,
        name: functionalGroupName(code, t),
        control,
        countCheck: group ? countCheck(group.expectedCount, group.actualCount) : null,
        entries: transactions.map((transaction) => {
          const set = transaction.transactionSet?.trim() ?? "";
          const stControl = transaction.stControlNumber?.trim() ?? "";
          return entryFor(
            file,
            notes,
            {
              key: `tx-${transaction.index}`,
              set,
              control: stControl,
              segments: transaction.actualSegments,
              countCheck: countCheck(transaction.expectedSegments, transaction.actualSegments),
              message: claim(set, stControl, control),
              findings: findingsIn(transaction),
            },
            t,
          );
        }),
      };
    });
}

function keptManifest(
  file: EDIInboundFile,
  messages: readonly EDIMessage[],
  notes: readonly ProcessingNote[],
  t: TranslateFn,
): ManifestGroup[] {
  const groups = new Map<string, ManifestGroup>();
  for (const message of messages) {
    const control = message.groupControlNumber.trim();
    const code = SET_GROUP[message.transactionSet];
    const key = `${code}-${control}`;
    const group = groups.get(key) ?? {
      key: `group-${key}`,
      code,
      name: functionalGroupName(code, t),
      control,
      countCheck: null,
      entries: [],
    };
    group.entries.push(
      entryFor(
        file,
        notes,
        {
          key: `msg-${message.id}`,
          set: message.transactionSet,
          control: message.transactionControlNumber.trim(),
          segments: message.segmentCount,
          countCheck: null,
          message,
          findings: [],
        },
        t,
      ),
    );
    groups.set(key, group);
  }
  return [...groups.values()];
}

/** A run of the interchange's segments: envelope between sets, or one transaction set. */
export type RulerRun = {
  key: string;
  kind: "envelope" | "transaction";
  start: number;
  span: number;
  /** Which functional group a transaction set sits in, counted from one; null for the envelope. */
  group: number | null;
  state: "ok" | "warn" | "bad";
};

function worstOf(findings: readonly EDIInspectionDiagnostic[]): RulerRun["state"] {
  if (findings.some((finding) => finding.severity === "Error")) return "bad";
  if (findings.some((finding) => finding.severity === "Warning")) return "warn";
  return "ok";
}

/**
 * The whole interchange as one ruler, segment for segment: each transaction
 * set a run as long as its segments, shaded by the group it sits in and
 * flagged by the worst finding in it, the envelope between them as the runs
 * that hold them apart.
 */
export function segmentRuler(inspection: EDIX12Inspection): RulerRun[] {
  const runs: RulerRun[] = [];
  const groupOf = (transaction: InspectedTransaction) => {
    const index = inspection.groups.findIndex((group) => holds(group, transaction));
    return index === -1 ? null : index + 1;
  };
  const findingsBetween = (start: number, end: number) =>
    inspection.diagnostics.filter(
      (finding) => finding.segmentIndex >= start && finding.segmentIndex <= end,
    );
  const envelope = (start: number, end: number) => {
    if (end < start) return;
    runs.push({
      key: `env-${start}`,
      kind: "envelope",
      start,
      span: end - start + 1,
      group: null,
      state: worstOf(findingsBetween(start, end)),
    });
  };

  let next = 1;
  for (const transaction of inspection.transactions) {
    const end = transaction.endSegmentIndex || inspection.summary.segmentCount;
    envelope(next, transaction.startSegmentIndex - 1);
    runs.push({
      key: `tx-${transaction.index}`,
      kind: "transaction",
      start: transaction.startSegmentIndex,
      span: end - transaction.startSegmentIndex + 1,
      group: groupOf(transaction),
      state: worstOf(findingsBetween(transaction.startSegmentIndex, end)),
    });
    next = end + 1;
  }
  envelope(next, inspection.summary.segmentCount);
  return runs;
}

/** The findings that sit on no transaction set: the envelope's, and the file's as a whole. */
export function looseFindings(inspection: EDIX12Inspection): EDIInspectionDiagnostic[] {
  return inspection.diagnostics.filter(
    (finding) =>
      finding.segmentIndex <= 0 ||
      !inspection.transactions.some((transaction) => within(transaction, finding.segmentIndex)),
  );
}

/** Whether the file holds an X12 interchange at all. */
export function isInterchange(inspection: EDIX12Inspection): boolean {
  return inspection.segments.some((segment) => segment.segmentId === "ISA");
}

/** The first lines of a file that is not X12, each cut to a readable length, and how many more. */
export function rawGlimpse(
  raw: string,
  maxLines: number,
  maxChars = 120,
): { lines: string[]; more: number } {
  const all = raw.split(/\r?\n/);
  if (all.at(-1) === "") all.pop();
  return {
    lines: all
      .slice(0, maxLines)
      .map((line) => (line.length > maxChars ? `${line.slice(0, maxChars)}…` : line)),
    more: Math.max(all.length - maxLines, 0),
  };
}
