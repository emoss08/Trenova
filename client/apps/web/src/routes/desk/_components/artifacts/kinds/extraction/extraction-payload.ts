import type { PageBox } from "./page-geometry";

/** Why a reading needs a person: two readings disagree, it was marked for review, or it is unsure. */
export type FieldReason = "conflict" | "review" | "low";

/** Which part of a stop a field is; `whole` is a stop read before stops were split into parts. */
export type StopPart = "name" | "address" | "window" | "whole";

export type FieldStop = {
  sequence: number;
  /** As the reading wrote it, lower case: pickup, delivery or anything else. */
  role: string;
  part: StopPart;
  /** The stop's city and state, as the route's ends are named; empty on an older card. */
  place: string;
  appointment: boolean;
};

/** One value read off the page, how sure the reading is, and where it was. */
export type ExtractedField = {
  key: string;
  label: string;
  value: string;
  confidence: number;
  needsLook: boolean;
  reason: FieldReason | null;
  /** Every value the document gave for a field whose readings disagree. */
  readings: string[];
  /** The page it was read from, from 1; 0 when the reading does not say. */
  page: number;
  box: PageBox | null;
  stop: FieldStop | null;
};

/** A field the document was expected to carry and did not. */
export type MissingField = { key: string; label: string };

/** A page's drawn size, for its frame before the drawing arrives; 0 when not known. */
export type PageSize = { number: number; width: number; height: number };

export type Extraction = {
  documentId: string;
  fileName: string;
  contentType: string;
  documentKind: string;
  kindLabel: string;
  /** How sure the reader was of the kind, 0 to 1; null on a card read before it was kept. */
  kindConfidence: number | null;
  confidence: number;
  reviewStatus: string;
  pageCount: number;
  pages: PageSize[];
  fields: ExtractedField[];
  missing: MissingField[];
  attachedShipmentId: string;
};

type Json = Record<string, unknown>;

const STOP_PARTS: readonly StopPart[] = ["name", "address", "window"];
const LEGACY_STOP_KEY = /^stop(\d+)$/;

function isObject(value: unknown): value is Json {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function text(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function number(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

function fraction(value: unknown): number {
  return Math.min(1, Math.max(0, number(value)));
}

function boxOf(value: unknown): PageBox | null {
  if (!isObject(value)) return null;
  const box = {
    x: fraction(value.x),
    y: fraction(value.y),
    w: fraction(value.w),
    h: fraction(value.h),
  };
  return box.w > 0 && box.h > 0 ? box : null;
}

function reasonOf(field: Json, needsLook: boolean, confidence: number): FieldReason | null {
  const reason = text(field.reason);
  if (reason === "conflict" || reason === "review" || reason === "low") return reason;
  if (!needsLook) return null;
  return confidence < 0.8 ? "low" : "review";
}

function stopOf(field: Json, key: string, label: string): FieldStop | null {
  const sequence = number(field.stop);
  const part = text(field.part);
  if (sequence > 0 && (STOP_PARTS as readonly string[]).includes(part)) {
    return {
      sequence,
      role: text(field.role).toLowerCase(),
      part: part as StopPart,
      place: text(field.place),
      appointment: field.appointment === true,
    };
  }
  const legacy = LEGACY_STOP_KEY.exec(key);
  if (legacy) {
    return {
      sequence: Number(legacy[1]),
      role: label.toLowerCase(),
      part: "whole",
      place: "",
      appointment: false,
    };
  }
  return null;
}

function fieldOf(field: Json): ExtractedField | null {
  const key = text(field.key);
  const value = text(field.value);
  if (key === "" || value === "") return null;
  const label = text(field.label) || key;
  const confidence = fraction(field.confidence);
  const needsLook = field.needsLook === true;
  const readings = Array.isArray(field.readings) ? field.readings.map(text).filter(Boolean) : [];
  return {
    key,
    label,
    value,
    confidence,
    needsLook,
    reason: reasonOf(field, needsLook, confidence),
    readings,
    page: Math.max(0, Math.trunc(number(field.page))),
    box: boxOf(field.box),
    stop: stopOf(field, key, label),
  };
}

function missingOf(payload: Json): MissingField[] {
  if (Array.isArray(payload.missingFields)) {
    return payload.missingFields.flatMap((entry) => {
      if (!isObject(entry)) return [];
      const label = text(entry.label);
      const key = text(entry.key);
      return label === "" && key === "" ? [] : [{ key: key || label, label: label || key }];
    });
  }
  if (Array.isArray(payload.missing)) {
    return payload.missing.map(text).flatMap((label) => (label === "" ? [] : [{ key: "", label }]));
  }
  return [];
}

function pagesOf(payload: Json): PageSize[] {
  if (!Array.isArray(payload.pages)) return [];
  return payload.pages.flatMap((page) => {
    if (!isObject(page) || number(page.number) < 1) return [];
    return [
      {
        number: Math.trunc(number(page.number)),
        width: number(page.width),
        height: number(page.height),
      },
    ];
  });
}

/**
 * An extraction artifact's payload, read defensively. A card saved before a field was kept
 * reads as it did then: no page sizes, no reasons (a field that needs a look is low when it is
 * under 80% and marked for review otherwise), one row per stop, and missing fields by label.
 */
export function extractionFrom(payload: Json): Extraction {
  const fields = Array.isArray(payload.fields)
    ? payload.fields.flatMap((field) => {
        const read = isObject(field) ? fieldOf(field) : null;
        return read ? [read] : [];
      })
    : [];
  const pages = pagesOf(payload);
  const named = fields.reduce((most, field) => Math.max(most, field.page), 0);
  const kindConfidence =
    typeof payload.kindConfidence === "number" ? fraction(payload.kindConfidence) : null;

  return {
    documentId: text(payload.documentId),
    fileName: text(payload.fileName),
    contentType: text(payload.contentType).toLowerCase(),
    documentKind: text(payload.documentKind),
    kindLabel: text(payload.kindLabel),
    kindConfidence: kindConfidence !== null && kindConfidence > 0 ? kindConfidence : null,
    confidence: fraction(payload.confidence),
    reviewStatus: text(payload.reviewStatus),
    pageCount: Math.max(1, Math.trunc(number(payload.pageCount)), pages.length, named),
    pages,
    fields,
    missing: missingOf(payload),
    attachedShipmentId: text(payload.attachedShipmentId),
  };
}

/** Whether the file is an image rather than a PDF, from its type or, on an older card, its name. */
export function isImageFile(extraction: Pick<Extraction, "contentType" | "fileName">): boolean {
  if (extraction.contentType !== "") return extraction.contentType.startsWith("image/");
  return /\.(png|jpe?g|gif|webp|tiff?|bmp|heic)$/i.test(extraction.fileName);
}
