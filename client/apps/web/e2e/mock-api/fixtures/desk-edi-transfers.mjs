// The EDI load tenders the Desk cards thread names. Each transfer and its
// mapping preview is the Go service's own JSON, the same fixtures the card's
// unit tests read, moved in time so it was submitted a moment before the
// mock's now, and handed to the mock's organization in place of the test's.
import approved from "../../../src/routes/desk/_components/artifacts/cards/edi-transfer/__tests__/approved.json" with { type: "json" };
import failed from "../../../src/routes/desk/_components/artifacts/cards/edi-transfer/__tests__/failed.json" with { type: "json" };
import mappingRequiredPreview from "../../../src/routes/desk/_components/artifacts/cards/edi-transfer/__tests__/mapping-required.preview.json" with { type: "json" };
import mappingRequired from "../../../src/routes/desk/_components/artifacts/cards/edi-transfer/__tests__/mapping-required.json" with { type: "json" };
import outbound from "../../../src/routes/desk/_components/artifacts/cards/edi-transfer/__tests__/outbound.json" with { type: "json" };
import rejected from "../../../src/routes/desk/_components/artifacts/cards/edi-transfer/__tests__/rejected.json" with { type: "json" };
import superseded from "../../../src/routes/desk/_components/artifacts/cards/edi-transfer/__tests__/superseded.json" with { type: "json" };
import { ORG } from "./session.mjs";

const MINUTE = 60;
const HOUR = 3_600;
const DAY = 86_400;
const FIXTURE_ORG = "org_1";

/**
 * Every Unix-second instant in a fixture moved by `delta`, the test's
 * organization swapped for the mock's, the rest copied as it is.
 */
function moved(value, delta) {
  if (Array.isArray(value)) return value.map((entry) => moved(entry, delta));
  if (value === FIXTURE_ORG) return ORG.id;
  if (value === null || typeof value !== "object") return value;
  return Object.fromEntries(
    Object.entries(value).map(([key, entry]) => [
      key,
      /(At|Start|End)$/.test(key) && typeof entry === "number" && entry > 0
        ? entry + delta
        : moved(entry, delta),
    ]),
  );
}

/** A transfer submitted `ago` seconds before the anchor, with its preview when it has one. */
function submitted(transfer, preview, anchor, ago, overrides = {}) {
  return {
    transfer: { ...moved(transfer, anchor - ago - transfer.submittedAt), ...overrides },
    preview: preview ? moved(preview, 0) : null,
  };
}

/** Each EDI transfer the cards thread names, with its mapping preview when it is open to a decision. */
export function deskEdiTransfers(anchor) {
  return {
    edilt_acme_88241: submitted(mappingRequired, mappingRequiredPreview, anchor, 2 * HOUR),
    edilt_acme_88240: submitted(approved, null, anchor, 2 * HOUR),
    edilt_acme_88199: submitted(rejected, null, anchor, 3 * DAY),
    edilt_acme_88237: submitted(superseded, null, anchor, DAY),
    edilt_acme_88233: submitted(failed, null, anchor, 5 * HOUR),
    edilt_s2610_1366: submitted(outbound, null, anchor, 40 * MINUTE),
  };
}

/** The tenders each inbound file opened, in the order the API lists them. */
export const DESK_EDI_FILE_TENDERS = {
  ediinf_acme: ["edilt_acme_88240", "edilt_acme_88241"],
};

/** The cards thread's EDI transfer conversations: each slug's tender, its title, and what was asked and said. */
export const DESK_EDI_TRANSFER_CONVERSATIONS = {
  "tender-accepted": {
    recordId: "edilt_acme_88240",
    title: "Load tender LT-88240",
    question: "Did Acme's LT-88240 turn into a shipment?",
    reply:
      "Yes: Maya accepted it this morning and the shipment was built a few seconds later. [LT-88240](artifact:art_tender-accepted) has the route and the rate.",
  },
  "tender-mapping": {
    recordId: "edilt_acme_88241",
    title: "Load tender LT-88241",
    question: "What is holding up Acme's change to LT-88241?",
    reply:
      "Two of its codes are not mapped yet: the San Antonio delivery and the fuel surcharge. Map those and it can be accepted. [LT-88241](artifact:art_tender-mapping) lists them first.",
  },
  "tender-declined": {
    recordId: "edilt_acme_88199",
    title: "Load tender LT-88199",
    question: "Why did we turn down LT-88199?",
    reply:
      "It was declined three days ago for lack of reefer capacity on the Dallas to Houston lane. [LT-88199](artifact:art_tender-declined) keeps the reason.",
  },
  "tender-outbound": {
    recordId: "edilt_s2610_1366",
    title: "Load tender BOL-55120",
    question: "Has Lone Star picked up the load we tendered them?",
    reply:
      "Not yet: S2610-1366 went to them 40 minutes ago and is waiting on their decision. [BOL-55120](artifact:art_tender-outbound) shows what they were sent.",
  },
  "tender-failed": {
    recordId: "edilt_acme_88233",
    title: "Load tender LT-88233",
    question: "Acme says LT-88233 was accepted but there's no shipment?",
    reply:
      "It was accepted, but building the shipment failed. [LT-88233](artifact:art_tender-failed) has the error the service wrote.",
  },
  "tender-superseded": {
    recordId: "edilt_acme_88237",
    title: "Load tender LT-88237",
    question: "What happened to LT-88237?",
    reply:
      "Acme sent a change that replaced it, so it was canceled. [LT-88237](artifact:art_tender-superseded) says which.",
  },
  "tender-gone": {
    recordId: "edilt_globex_0602",
    title: "Load tender GX-0602",
    question: "Pull up Globex's tender from last week.",
    reply:
      "That tender has been removed since I looked it up. [GX-0602](artifact:art_tender-gone) is what I found then.",
  },
};
