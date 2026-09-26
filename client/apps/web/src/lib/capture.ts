import type {
  CaptureBatchStatus,
  CaptureItemStatus,
  CapturePixelType,
  CaptureRequestStatus,
  CaptureSeparatorStrategy,
  CaptureSource,
  CaptureSuggestionSource,
} from "@/lib/graphql/capture";
import type { CaptureRequestFailureCode } from "@trenova/graphql/generated/graphql";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { BadgeAttrProps } from "@trenova/shared/lib/status-phase";
import type { DocumentCategory } from "@trenova/shared/types/document-type";

/**
 * The records a captured document may be filed onto. The server holds the
 * same list (`capture.FileableResources`) and refuses anything else, so this
 * is the order the pickers offer them in, not a second rule.
 */
export const CAPTURE_RECORD_KINDS = [
  "shipment",
  "worker",
  "tractor",
  "trailer",
  "customer",
  "carrier",
] as const;

export type CaptureRecordKind = (typeof CAPTURE_RECORD_KINDS)[number];

export function isCaptureRecordKind(value: string): value is CaptureRecordKind {
  return (CAPTURE_RECORD_KINDS as readonly string[]).includes(value);
}

export function captureRecordKindLabel(t: TranslateFn, kind: CaptureRecordKind): string {
  switch (kind) {
    case "shipment":
      return t("Shipment");
    case "worker":
      return t("Worker");
    case "tractor":
      return t("Tractor");
    case "trailer":
      return t("Trailer");
    case "customer":
      return t("Customer");
    case "carrier":
      return t("Carrier");
  }
}

/**
 * The document types offered for a kind of record. Shipments and workers have
 * categories of their own; the rest take any type, as an upload does.
 */
export function captureDocumentCategory(kind: CaptureRecordKind): DocumentCategory | null {
  switch (kind) {
    case "shipment":
      return "Shipment";
    case "worker":
      return "Worker";
    case "tractor":
    case "trailer":
    case "customer":
    case "carrier":
      return null;
  }
}

/**
 * What a person calls a stack: the print job's name, else the scanner it came
 * from, else what it is.
 */
export function captureBatchTitle(
  t: TranslateFn,
  batch: { jobName: string; sourceName: string; source: CaptureSource },
): string {
  if (batch.jobName !== "") {
    return batch.jobName;
  }
  if (batch.sourceName !== "") {
    return batch.sourceName;
  }

  return batch.source === "Print" ? t("Printed document") : t("Scanned stack");
}

/** Where a stack is: arriving, being read, waiting on a person, done. */
export function captureBatchStatusAttrs(
  t: TranslateFn,
): Record<CaptureBatchStatus, BadgeAttrProps> {
  return {
    Receiving: {
      phase: "active",
      text: t("Receiving"),
      description: t("Pages are still arriving from the scanner"),
    },
    Sealed: {
      phase: "queued",
      text: t("Queued"),
      description: t("Every page arrived; the stack is waiting to be read"),
    },
    Processing: {
      phase: "active",
      text: t("Reading"),
      description: t("Trenova is splitting the stack and suggesting where each part goes"),
    },
    Ready: {
      phase: "awaiting",
      text: t("To file"),
      description: t("Split and waiting for a person to file it"),
    },
    PartiallyFiled: {
      phase: "awaiting",
      text: t("Partly filed"),
      description: t("Some documents are filed and some are still waiting"),
    },
    Filed: { phase: "complete", text: t("Filed"), description: t("Every document is filed") },
    Discarded: {
      phase: "closed",
      text: t("Discarded"),
      description: t("Thrown away without filing what was left"),
    },
    Expired: {
      phase: "failed",
      text: t("Expired"),
      description: t("Not filed before the retention date; its pages were deleted"),
    },
    Failed: {
      phase: "failed",
      text: t("Failed"),
      description: t("The stack could not be read"),
    },
  };
}

export function captureItemStatusAttrs(t: TranslateFn): Record<CaptureItemStatus, BadgeAttrProps> {
  return {
    Proposed: { phase: "awaiting", text: t("To file") },
    Filing: { phase: "active", text: t("Filing") },
    Filed: { phase: "complete", text: t("Filed") },
    Discarded: { phase: "closed", text: t("Discarded") },
    Failed: { phase: "failed", text: t("Failed") },
  };
}

export function captureRequestStatusAttrs(
  t: TranslateFn,
): Record<CaptureRequestStatus, BadgeAttrProps> {
  return {
    Pending: {
      phase: "queued",
      text: t("Waiting for the scanner"),
      description: t("Trenova Capture has not picked the request up yet"),
    },
    Delivered: { phase: "active", text: t("At the scanner") },
    InProgress: { phase: "active", text: t("Scanning") },
    Completed: { phase: "complete", text: t("Done") },
    Canceled: { phase: "closed", text: t("Cancelled") },
    Expired: {
      phase: "failed",
      text: t("Not picked up"),
      description: t("The computer was off or Trenova Capture was not running"),
    },
    Failed: { phase: "failed", text: t("Failed") },
  };
}

export function captureRequestFailureLabel(
  t: TranslateFn,
  code: CaptureRequestFailureCode,
): string {
  switch (code) {
    case "SOURCE_UNAVAILABLE":
      return t("The scanner is not connected");
    case "SOURCE_BUSY":
      return t("The scanner is busy with another job");
    case "PAPER_JAM":
      return t("The paper jammed");
    case "FEEDER_EMPTY":
      return t("The feeder is empty");
    case "CANCELED_BY_USER":
      return t("Cancelled at the scanner");
    case "DRIVER_ERROR":
      return t("The scanner's driver reported an error");
    case "UPLOAD_FAILED":
      return t("The pages could not be uploaded");
    case "NOT_DELIVERED":
      return t("Trenova Capture never picked the request up");
    case "INTERNAL":
      return t("Trenova Capture ran into a problem");
  }
}

export function captureSourceLabel(t: TranslateFn, source: CaptureSource): string {
  switch (source) {
    case "Scan":
      return t("Scan");
    case "Print":
      return t("Print");
  }
}

export function captureSuggestionSourceLabel(
  t: TranslateFn,
  source: CaptureSuggestionSource,
): string {
  switch (source) {
    case "CoverSheet":
      return t("From a cover sheet");
    case "Request":
      return t("Scanned into this record");
    case "Classifier":
      return t("Read from the document");
    case "Person":
      return t("Set by a person");
  }
}

export function capturePixelTypeLabel(t: TranslateFn, pixelType: CapturePixelType): string {
  switch (pixelType) {
    case "BlackWhite":
      return t("Black and white");
    case "Grayscale":
      return t("Grayscale");
    case "Color":
      return t("Color");
  }
}

export function captureSeparatorLabel(t: TranslateFn, strategy: CaptureSeparatorStrategy): string {
  switch (strategy) {
    case "PatchCode":
      return t("Patch code sheets");
    case "CoverSheet":
      return t("Trenova cover sheets");
    case "BlankPage":
      return t("Blank pages");
    case "FixedPageCount":
      return t("Every few pages");
  }
}

const DAY_SECONDS = 86_400;

/** How close to deletion a stack is before it says so. */
export const CAPTURE_RETENTION_WARNING_SECONDS = 7 * DAY_SECONDS;

export type CaptureRetention =
  | { state: "later" }
  | { state: "soon"; daysLeft: number }
  | { state: "due" };

/**
 * How near a stack's unfiled pages are to being deleted. Whole days are
 * counted up, so a stack with three hours left still says one day rather
 * than none.
 */
export function captureRetention(retainUntil: number, now: number): CaptureRetention {
  const remaining = retainUntil - now;
  if (remaining <= 0) {
    return { state: "due" };
  }
  if (remaining > CAPTURE_RETENTION_WARNING_SECONDS) {
    return { state: "later" };
  }

  return { state: "soon", daysLeft: Math.ceil(remaining / DAY_SECONDS) };
}

/** How long a finished request stays on its record's page, so the person sees how it went. */
export const CAPTURE_REQUEST_LINGER_SECONDS = 15 * 60;

type RequestTiming = {
  isOpen: boolean;
  completedAt: number | null;
  createdAt: number;
};

/**
 * The requests a record's page shows: everything still open, and anything
 * that finished in the last quarter of an hour, newest first. Older ones are
 * history, and the intake queue holds what they produced.
 */
export function captureRequestsToShow<T extends RequestTiming>(
  requests: readonly T[],
  now: number,
): T[] {
  return requests
    .filter(
      (request) =>
        request.isOpen ||
        (request.completedAt !== null &&
          now - request.completedAt <= CAPTURE_REQUEST_LINGER_SECONDS),
    )
    .sort((a, b) => b.createdAt - a.createdAt);
}

/** A pairing code's length without its dash; the server's `capture.UserCodeLength`. */
export const CAPTURE_PAIRING_CODE_LENGTH = 8;

/**
 * A pairing code however it was typed or pasted — lower case, with or without
 * the dash, with stray spaces — as the letters the server compares. It keeps
 * only letters, as the server does, and never more than a code holds.
 */
export function normalizePairingCode(raw: string): string {
  return raw
    .toUpperCase()
    .replace(/[^A-Z]/g, "")
    .slice(0, CAPTURE_PAIRING_CODE_LENGTH);
}

/** A code the way the companion shows it: two groups of four. */
export function formatPairingCode(raw: string): string {
  const code = normalizePairingCode(raw);
  return code.length > 4 ? `${code.slice(0, 4)}-${code.slice(4)}` : code;
}
