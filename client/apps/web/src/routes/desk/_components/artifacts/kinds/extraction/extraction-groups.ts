import { defineLabels } from "@trenova/shared/i18n/labels";
import type { ExtractedField, Extraction, MissingField } from "./extraction-payload";

export type GroupId = "parties" | "stops" | "references" | "freight" | "charges" | "other";

/** How a reading stands: settled, read but not fully sure, needing a person, or not found. */
export type FieldState = "settled" | "fair" | "doubtful" | "missing";

/** A row of the list: a value that was read, or one the document was expected to carry. */
export type FieldRow =
  | { kind: "read"; key: string; label: string; field: ExtractedField; state: FieldState }
  | { kind: "missing"; key: string; label: string; missing: MissingField; state: "missing" };

export type StopGroup = {
  sequence: number;
  role: string;
  rows: FieldRow[];
  /** A part of the stop needs a person. */
  doubtful: boolean;
};

export type FieldGroup = {
  id: GroupId;
  rows: FieldRow[];
  /** Only the stops group has stops; its `rows` are what belongs to no stop. */
  stops: StopGroup[];
};

/** Read at or above this, a value is settled; below the review line it needs a person. */
export const SETTLED_CONFIDENCE = 0.9;

const GROUP_ORDER: readonly GroupId[] = [
  "parties",
  "stops",
  "references",
  "freight",
  "charges",
  "other",
];

type Place = { group: GroupId; order: number; stopRole?: "pickup" | "delivery" };

/**
 * Where each key document intelligence writes is read: the parties, then each stop with its
 * window and numbers, then the references, the freight and the charges. A key not here is
 * listed under Other, never dropped.
 */
const PLACES: Readonly<Record<string, Place>> = {
  shipper: { group: "parties", order: 0 },
  consignee: { group: "parties", order: 1 },
  billTo: { group: "parties", order: 2 },
  carrierName: { group: "parties", order: 3 },
  carrierContact: { group: "parties", order: 4 },
  pickupStop: { group: "stops", order: 0 },
  deliveryStop: { group: "stops", order: 1 },
  pickupDate: { group: "stops", order: 10, stopRole: "pickup" },
  pickupWindow: { group: "stops", order: 11, stopRole: "pickup" },
  pickupNumber: { group: "stops", order: 12, stopRole: "pickup" },
  deliveryDate: { group: "stops", order: 10, stopRole: "delivery" },
  deliveryWindow: { group: "stops", order: 11, stopRole: "delivery" },
  deliveryNumber: { group: "stops", order: 12, stopRole: "delivery" },
  loadNumber: { group: "references", order: 0 },
  referenceNumber: { group: "references", order: 1 },
  poNumber: { group: "references", order: 2 },
  bol: { group: "references", order: 3 },
  proNumber: { group: "references", order: 4 },
  appointmentNumber: { group: "references", order: 5 },
  scac: { group: "references", order: 6 },
  containerNumber: { group: "references", order: 7 },
  trailerNumber: { group: "references", order: 8 },
  tractorNumber: { group: "references", order: 9 },
  equipmentType: { group: "freight", order: 0 },
  serviceType: { group: "freight", order: 1 },
  commodity: { group: "freight", order: 2 },
  weight: { group: "freight", order: 3 },
  pieceCount: { group: "freight", order: 4 },
  pieces: { group: "freight", order: 4 },
  rate: { group: "charges", order: 0 },
  fuelSurcharge: { group: "charges", order: 1 },
  paymentTerms: { group: "charges", order: 2 },
  instructions: { group: "other", order: 0 },
  receiptNotes: { group: "other", order: 1 },
  signature: { group: "other", order: 2 },
};

/** Each known key as the reader sees it, in sentence case and in the reader's language. */
export const FIELD_LABELS = defineLabels({
  shipper: "Shipper",
  consignee: "Consignee",
  billTo: "Bill to",
  carrierName: "Carrier",
  carrierContact: "Carrier contact",
  pickupStop: "Pickup stop",
  deliveryStop: "Delivery stop",
  pickupDate: "Pickup date",
  pickupWindow: "Pickup window",
  pickupNumber: "Pickup #",
  deliveryDate: "Delivery date",
  deliveryWindow: "Delivery window",
  deliveryNumber: "Delivery #",
  loadNumber: "Load #",
  referenceNumber: "Reference #",
  poNumber: "PO #",
  bol: "BOL #",
  proNumber: "Pro #",
  appointmentNumber: "Appointment #",
  scac: "SCAC",
  containerNumber: "Container #",
  trailerNumber: "Trailer #",
  tractorNumber: "Tractor #",
  equipmentType: "Equipment",
  serviceType: "Service",
  commodity: "Commodity",
  weight: "Weight",
  pieceCount: "Pieces",
  pieces: "Pieces",
  rate: "Rate",
  fuelSurcharge: "Fuel surcharge",
  paymentTerms: "Payment terms",
  instructions: "Instructions",
  receiptNotes: "Receipt notes",
  signature: "Signature",
});

/** A stop's parts, as each row of a stop is labelled. */
export const STOP_PART_LABELS = defineLabels({
  name: "Name",
  address: "Address",
  window: "Window",
  whole: "Stop",
});

export const GROUP_LABELS = defineLabels({
  parties: "Parties",
  stops: "Stops",
  references: "References",
  freight: "Equipment and freight",
  charges: "Charges",
  other: "Other",
});

const STOP_PART_ORDER = { name: 0, address: 1, window: 2, whole: 0 } as const;

function fold(text: string): string {
  return text.toLowerCase().replace(/[^\p{L}\p{N}]/gu, "");
}

const KEY_BY_FOLDED: ReadonlyMap<string, string> = new Map(
  Object.keys(PLACES).map((key) => [fold(key), key]),
);

/** The catalog key of a missing field, read from its key or, on an older card, its label. */
function missingKey(missing: MissingField): string {
  return (
    KEY_BY_FOLDED.get(fold(missing.key)) ?? KEY_BY_FOLDED.get(fold(missing.label)) ?? missing.key
  );
}

function labelOf(key: string, fallback: string): string {
  return Object.hasOwn(FIELD_LABELS, key)
    ? FIELD_LABELS[key as keyof typeof FIELD_LABELS]
    : fallback;
}

/** How a reading stands, from whether it needs a look and how sure the reading was. */
export function fieldState(field: ExtractedField): FieldState {
  if (field.needsLook) return "doubtful";
  return field.confidence >= SETTLED_CONFIDENCE ? "settled" : "fair";
}

function readRow(field: ExtractedField): FieldRow {
  const label = field.stop ? STOP_PART_LABELS[field.stop.part] : labelOf(field.key, field.label);
  return { kind: "read", key: field.key, label, field, state: fieldState(field) };
}

function missingRow(missing: MissingField): FieldRow {
  const key = missingKey(missing) || fold(missing.label);
  return {
    kind: "missing",
    key: `missing:${key}`,
    label: labelOf(key, missing.label),
    missing,
    state: "missing",
  };
}

/** A settled date or window read apart from its stop that says only what the stop's own does. */
function repeatsStop(stop: StopGroup, field: ExtractedField): boolean {
  const value = fold(field.value);
  return stop.rows.some(
    (row) =>
      row.kind === "read" &&
      row.field.stop?.part === "window" &&
      fold(row.field.value).includes(value),
  );
}

function placeOf(key: string): Place {
  return PLACES[key] ?? { group: "other", order: Number.MAX_SAFE_INTEGER };
}

function orderOf(row: FieldRow): number {
  if (row.kind === "read" && row.field.stop) return STOP_PART_ORDER[row.field.stop.part];
  return placeOf(row.kind === "read" ? row.field.key : missingKey(row.missing)).order;
}

function byOrder(a: FieldRow, b: FieldRow): number {
  return orderOf(a) - orderOf(b) || a.label.localeCompare(b.label);
}

/**
 * The reading grouped the way a dispatcher reads a load: who, then the stops in order with
 * what each one's window and numbers are, then the references, the freight and the charges.
 * A field the document lacked stands where it would have been read. A pickup or delivery
 * window read apart from its stop joins the first stop of that role, unless it is settled and
 * repeats what the stop's own window says; with no such stop it stands in the stops group.
 */
export function groupExtraction(extraction: Pick<Extraction, "fields" | "missing">): FieldGroup[] {
  const groups = new Map<GroupId, FieldGroup>(
    GROUP_ORDER.map((id) => [id, { id, rows: [], stops: [] }]),
  );
  const stops = new Map<number, StopGroup>();
  const loose: ExtractedField[] = [];

  for (const field of extraction.fields) {
    if (field.stop) {
      const stop = stops.get(field.stop.sequence) ?? {
        sequence: field.stop.sequence,
        role: field.stop.role,
        rows: [],
        doubtful: false,
      };
      stop.rows.push(readRow(field));
      stop.doubtful ||= field.needsLook;
      stops.set(field.stop.sequence, stop);
    } else {
      loose.push(field);
    }
  }

  const ordered = [...stops.values()].sort((a, b) => a.sequence - b.sequence);
  for (const field of loose) {
    const place = placeOf(field.key);
    const stop = place.stopRole
      ? ordered.find((candidate) => candidate.role === place.stopRole)
      : undefined;
    if (stop) {
      if (!field.needsLook && repeatsStop(stop, field)) continue;
      stop.rows.push(readRow(field));
      stop.doubtful ||= field.needsLook;
    } else {
      groups.get(place.group)?.rows.push(readRow(field));
    }
  }
  for (const missing of extraction.missing) {
    groups.get(placeOf(missingKey(missing)).group)?.rows.push(missingRow(missing));
  }

  for (const stop of ordered) stop.rows.sort(byOrder);
  const stopsGroup = groups.get("stops");
  if (stopsGroup) stopsGroup.stops = ordered;

  return GROUP_ORDER.flatMap((id) => {
    const group = groups.get(id);
    if (!group || (group.rows.length === 0 && group.stops.length === 0)) return [];
    group.rows.sort(byOrder);
    return [group];
  });
}

/** Every row in the order the list draws them, which is the order the arrow keys walk. */
export function rowsInOrder(groups: readonly FieldGroup[]): FieldRow[] {
  return groups.flatMap((group) => [...group.stops.flatMap((stop) => stop.rows), ...group.rows]);
}

/** A thing for a person to settle: a field read loosely, or a stop with a doubtful part. */
export type CheckItem =
  | { kind: "field"; row: FieldRow & { kind: "read" } }
  | { kind: "stop"; stop: StopGroup };

export type ReviewTally = {
  check: CheckItem[];
  missing: (FieldRow & { kind: "missing" })[];
  /** Read and not in doubt. */
  read: number;
};

/** What the tally above the list counts: a stop under review counts once, whatever its parts. */
export function reviewTally(groups: readonly FieldGroup[]): ReviewTally {
  const check: CheckItem[] = [];
  const missing: ReviewTally["missing"] = [];
  let read = 0;
  for (const group of groups) {
    for (const stop of group.stops) {
      if (stop.doubtful) check.push({ kind: "stop", stop });
      for (const row of stop.rows) if (row.state !== "doubtful") read += 1;
    }
    for (const row of group.rows) {
      if (row.kind === "missing") missing.push(row);
      else if (row.state === "doubtful") check.push({ kind: "field", row });
      else read += 1;
    }
  }
  return { check, missing, read };
}

/** The first row a tally item stands for, which choosing the item selects. */
export function firstRowOf(item: CheckItem): FieldRow | undefined {
  return item.kind === "field"
    ? item.row
    : (item.stop.rows.find((row) => row.state === "doubtful") ?? item.stop.rows[0]);
}

/** The pages that hold a field to check, for the dot on each in the page switcher. */
export function pagesToCheck(fields: readonly ExtractedField[]): Set<number> {
  return new Set(fields.flatMap((field) => (field.needsLook && field.box ? [field.page] : [])));
}
