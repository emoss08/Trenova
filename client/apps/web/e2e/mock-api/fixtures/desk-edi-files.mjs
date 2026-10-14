// The EDI inbound files the Desk cards thread names. Each file and its
// inspection is the Go service's own JSON, the same fixtures the card's unit
// tests read, moved in time so it was received a moment before the mock's now.
import batchFile from "../../../src/routes/desk/_components/artifacts/cards/edi-inbound-file/__tests__/batch-file.json" with { type: "json" };
import duplicateFile from "../../../src/routes/desk/_components/artifacts/cards/edi-inbound-file/__tests__/duplicate-file.json" with { type: "json" };
import notX12Inspection from "../../../src/routes/desk/_components/artifacts/cards/edi-inbound-file/__tests__/not-x12.inspection.json" with { type: "json" };
import partnerBatchInspection from "../../../src/routes/desk/_components/artifacts/cards/edi-inbound-file/__tests__/partner-batch.inspection.json" with { type: "json" };
import quarantinedFile from "../../../src/routes/desk/_components/artifacts/cards/edi-inbound-file/__tests__/quarantined-file.json" with { type: "json" };
import simulator204Inspection from "../../../src/routes/desk/_components/artifacts/cards/edi-inbound-file/__tests__/simulator-204.inspection.json" with { type: "json" };
import simulatorFile from "../../../src/routes/desk/_components/artifacts/cards/edi-inbound-file/__tests__/simulator-file.json" with { type: "json" };

const HOUR = 3_600;
const DAY = 86_400;

/** Every Unix-second instant in a fixture moved by `delta`, the rest copied as it is. */
function shifted(value, delta) {
  if (Array.isArray(value)) return value.map((entry) => shifted(entry, delta));
  if (value === null || typeof value !== "object") return value;
  return Object.fromEntries(
    Object.entries(value).map(([key, entry]) => [
      key,
      /(At|Start|End)$/.test(key) && typeof entry === "number" && entry > 0
        ? entry + delta
        : shifted(entry, delta),
    ]),
  );
}

/** A file received `ago` seconds before the anchor, with its inspection. */
function received(file, inspection, anchor, ago, overrides = {}) {
  const moved = shifted(file, anchor - ago - file.receivedAt);
  return {
    file: { ...moved, ...overrides },
    inspection: overrides.rawPurgedAt ? null : inspection,
  };
}

/** Each EDI inbound file the cards thread names, with its inspection when its raw file is kept. */
export function deskEdiInboundFiles(anchor) {
  const purgedAt = anchor - 2 * DAY;
  return {
    ediinf_sim: received(simulatorFile, simulator204Inspection, anchor, 20 * 60),
    ediinf_acme: received(batchFile, partnerBatchInspection, anchor, 2 * HOUR),
    ediinf_acme_old: received(batchFile, partnerBatchInspection, anchor, 120 * DAY, {
      id: "ediinf_acme_old",
      fileName: "ACME_20260614_0905.x12",
      remotePath: "/inbound/acme/ACME_20260614_0905.x12",
      rawContent: "",
      rawPurgedAt: purgedAt,
    }),
    ediinf_bad: received(quarantinedFile, notX12Inspection, anchor, 5 * HOUR),
    ediinf_dup: received(duplicateFile, simulator204Inspection, anchor, 10 * 60),
  };
}

/** The cards thread's EDI inbound file conversations: each slug's file, its title, and what was asked and said. */
export const DESK_EDI_FILE_CONVERSATIONS = {
  "edi-file-sim": {
    recordId: "ediinf_sim",
    title: "EDI inbound file sim-204-1791716400000000000.edi",
    question: "Did the partner simulator's tender come through?",
    reply:
      "Yes: the simulator's file came in over AS2 twenty minutes ago and its one 204 was routed into inbound tenders. [sim-204-1791716400000000000.edi](artifact:art_edi-file-sim) has the interchange.",
  },
  "edi-file-acme": {
    recordId: "ediinf_acme",
    title: "EDI inbound file ACME_20261013_0912.x12",
    question: "Why is Acme's morning file only partly processed?",
    reply:
      "Its second tender is a change that matched no pending tender and still needs a location mapped, and its SE01 miscounts the segments. The first tender and the 997 went through. [ACME_20261013_0912.x12](artifact:art_edi-file-acme) pins each note to its transaction set.",
  },
  "edi-file-old": {
    recordId: "ediinf_acme_old",
    title: "EDI inbound file ACME_20260614_0905.x12",
    question: "Pull up Acme's file from June.",
    reply:
      "It was partly processed; the raw file has since been purged under the retention policy, so only the messages kept from it remain. [ACME_20260614_0905.x12](artifact:art_edi-file-old) shows what is kept.",
  },
  "edi-file-bad": {
    recordId: "ediinf_bad",
    title: "EDI inbound file tenders-week-42.txt",
    question: "What landed in Acme's SFTP mailbox this morning?",
    reply:
      "A plain-text email body, not X12, so it was quarantined before anything was read. [tenders-week-42.txt](artifact:art_edi-file-bad) shows how it begins.",
  },
  "edi-file-dup": {
    recordId: "ediinf_dup",
    title: "EDI inbound file sim-204-1791716400000000001.edi",
    question: "Why was the simulator's last file marked duplicate?",
    reply:
      "It reused interchange control number 000000042, which was already processed for this partner, so nothing in it was processed again. [sim-204-1791716400000000001.edi](artifact:art_edi-file-dup) has both.",
  },
  "edi-file-gone": {
    recordId: "ediinf_gone",
    title: "EDI inbound file GLOBEX_0602.x12",
    question: "Show me Globex's file from last week.",
    reply:
      "That file has been removed since I looked it up. [GLOBEX_0602.x12](artifact:art_edi-file-gone) is what I found then.",
  },
};
