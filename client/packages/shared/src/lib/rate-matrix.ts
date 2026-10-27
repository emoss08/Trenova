import { translate } from "@trenova/shared/i18n/runtime";
import type {
  RateMatrixCell,
  RateMatrixDimension,
  RateMatrixDimensionKind,
  RateMatrixValueType,
} from "../types/rate";
import { defineLabels } from "@trenova/shared/i18n/labels";

/**
 * A tariff arrives as a sheet — origin zone down the side, destination zone or
 * weight break across the top — and that is the only shape a pricing analyst
 * will check for holes. The server stores cells flat, one row per coordinate,
 * because that is what indexes and looks up quickly; this module is what turns
 * the flat list back into the sheet somebody can read.
 *
 * It only lays cells out and labels them. Whether the sheet has gaps, overlaps,
 * duplicates or a capped top band is a rule, checked by the rules module
 * (`matrixCoverage` in `@trenova/shared/rules/matrix`) with the same half-open
 * bands, `[min, max)`, the engine prices with.
 */

/**
 * The unit a matrix's cells are written in, the way a Tariff literal writes it
 * (`$`, `$/cwt`, `%`), so a cell reads as the value a tariff will get. A plain
 * number has none.
 */
export function matrixValueUnit(valueType: RateMatrixValueType): string {
  switch (valueType) {
    case "Number":
    case "Int":
      return "";
    case "Percent":
      return "%";
    default:
      return `$${valueType.slice("Money".length)}`;
  }
}

/** How many axes a matrix can declare, mirroring `ratematrix.MaxDimensions`. */
export const MAX_MATRIX_DIMENSIONS = 4;

export type MatrixAxisPosition = 0 | 1 | 2 | 3;

/** One bucket along an axis: a zone, a class, or a weight band. */
export type MatrixAxisBucket = {
  /** Stable identity, unique within the axis. */
  readonly key: string;
  /** What to print in the header. */
  readonly label: string;
  /** Present on Range axes only. */
  readonly min: number | null;
  readonly max: number | null;
  /** Present on postal range axes only: the range's first and last postal code. */
  readonly postal?: PostalRange;
};

/** A range of postal codes, both ends inclusive and of one length, normalised. */
export type PostalRange = { readonly from: string; readonly to: string };

export type MatrixGrid = {
  readonly rows: readonly MatrixAxisBucket[];
  readonly columns: readonly MatrixAxisBucket[];
  /** Cells for the selected slice, keyed `${rowKey}\u0000${columnKey}`. */
  readonly cells: ReadonlyMap<string, RateMatrixCell>;
  /** Coordinates with more than one cell — the engine would pick one and the sheet does not say which. */
  readonly duplicates: readonly string[];
  /** Coordinates the sheet draws but nothing prices. */
  readonly blanks: readonly string[];
};

const DIMENSION_KIND_LABEL: Record<RateMatrixDimensionKind, string> = defineLabels({
  Zone: "Zone",
  Zip3: "Postal prefix",
  Zip5: "Postal code",
  ZipRange: "Postal range",
  State: "State",
  Country: "Country",
  WeightBreak: "Weight break",
  Distance: "Distance",
  PieceCount: "Pieces",
  LinearFeet: "Linear feet",
  FreightClass: "Freight class",
  EquipmentType: "Equipment",
  ServiceType: "Service",
  Custom: "Custom",
  Quantity: "Quantity",
});

/** What the axis is called, preferring whatever the author typed. */
export function axisLabel(dimension: RateMatrixDimension): string {
  const typed = dimension.label?.trim();

  return typed ? typed : DIMENSION_KIND_LABEL[dimension.kind];
}

const KEY_FIELDS = ["d0Key", "d1Key", "d2Key", "d3Key"] as const;
const MIN_FIELDS = ["d0Min", "d1Min", "d2Min", "d3Min"] as const;
const MAX_FIELDS = ["d0Max", "d1Max", "d2Max", "d3Max"] as const;
const KEY_TO_FIELDS = ["d0KeyTo", "d1KeyTo", "d2KeyTo", "d3KeyTo"] as const;

export function cellKeyAt(cell: RateMatrixCell, position: MatrixAxisPosition): string {
  return cell[KEY_FIELDS[position]] ?? "";
}

/** A postal range axis's last postal code; empty on every other axis. */
export function cellKeyToAt(cell: RateMatrixCell, position: MatrixAxisPosition): string {
  return cell[KEY_TO_FIELDS[position]] ?? "";
}

/**
 * Whether an axis holds postal ranges: each cell names its first and last postal code, and a
 * postal code matches when its first characters fall between them, compared character by
 * character (`tables.Code[...]` in docs/engineering/formula-language.md).
 */
export function isPostalRangeAxis(dimension: Pick<RateMatrixDimension, "kind">): boolean {
  return dimension.kind === "ZipRange";
}

/** A postal code as a range stores and matches it: trimmed and upper case. */
export function normalizePostal(text: string): string {
  return text.trim().toUpperCase();
}

/** Orders two texts by Unicode code point, character by character, as the engine does. */
export function comparePostal(a: string, b: string): number {
  const left = [...a];
  const right = [...b];
  const shared = Math.min(left.length, right.length);
  for (let index = 0; index < shared; index++) {
    const difference = (left[index]?.codePointAt(0) ?? 0) - (right[index]?.codePointAt(0) ?? 0);
    if (difference !== 0) return difference;
  }
  return left.length - right.length;
}

function postalLength(text: string): number {
  return [...text].length;
}

/** A postal range's printed name: its two ends, both of which it covers. */
export function postalRangeLabel(range: PostalRange): string {
  if (range.from === range.to) return range.from;
  return translate("{0} through {1}", range.from, range.to);
}

/** Whether two postal ranges of one length share a postal code. */
export function postalRangesOverlap(a: PostalRange, b: PostalRange): boolean {
  return comparePostal(a.from, b.to) <= 0 && comparePostal(b.from, a.to) <= 0;
}

/**
 * Why a postal range cannot join an axis, or null when it can: both ends are needed, of one
 * length, in order, of the axis's length, and clear of every other range. `existing` is the
 * axis's ranges, without the one being edited.
 */
export function postalRangeProblem(
  range: PostalRange,
  existing: readonly PostalRange[],
): string | null {
  if (range.from === "" || range.to === "") {
    return translate("A postal range needs its first and its last postal code.");
  }
  if (postalLength(range.from) !== postalLength(range.to)) {
    return translate("Both ends of a postal range have the same number of characters.");
  }
  if (comparePostal(range.from, range.to) > 0) {
    return translate("{0} comes after {1}; a range starts at its lower end.", range.from, range.to);
  }
  const length = existing[0] ? postalLength(existing[0].from) : postalLength(range.from);
  if (postalLength(range.from) !== length) {
    return translate(
      "Every range on this axis has {0, plural, one {# character} other {# characters}}.",
      length,
    );
  }
  const clash = existing.find((other) => postalRangesOverlap(range, other));
  if (clash) {
    return translate(
      "This range overlaps {0}, so a postal code in both would match twice.",
      postalRangeLabel(clash),
    );
  }
  return null;
}

/** A cell's postal range on one axis. */
export function cellPostalRangeAt(cell: RateMatrixCell, position: MatrixAxisPosition): PostalRange {
  return { from: cellKeyAt(cell, position), to: cellKeyToAt(cell, position) };
}

/** The cell moved to another postal range on one axis. */
export function withCellPostalRange(
  cell: RateMatrixCell,
  position: MatrixAxisPosition,
  range: PostalRange,
): RateMatrixCell {
  return { ...cell, [KEY_FIELDS[position]]: range.from, [KEY_TO_FIELDS[position]]: range.to };
}

export function cellBoundsAt(
  cell: RateMatrixCell,
  position: MatrixAxisPosition,
): { min: number | null; max: number | null } {
  return {
    min: toNumber(cell[MIN_FIELDS[position]]),
    max: toNumber(cell[MAX_FIELDS[position]]),
  };
}

/**
 * Where a cell sits along one axis, as a single comparable string.
 *
 * Exact axes are their key. Range axes are their bounds, because two bands with
 * the same lower bound and different upper bounds are different buckets and
 * collapsing them would silently hide one.
 */
export function bucketKeyOf(cell: RateMatrixCell, dimension: RateMatrixDimension): string {
  const position = dimension.position as MatrixAxisPosition;

  if (isPostalRangeAxis(dimension)) {
    return postalBucketKey(cellPostalRangeAt(cell, position));
  }

  if (dimension.matchMode === "Exact") {
    return cellKeyAt(cell, position);
  }

  const { min, max } = cellBoundsAt(cell, position);

  return `${min ?? ""}..${max ?? ""}`;
}

/** The bucket key of a postal range: its two ends. */
export function postalBucketKey(range: PostalRange): string {
  return `${range.from}..${range.to}`;
}

/**
 * A cell's full position across every axis the matrix declares.
 *
 * This is the cell's identity: two cells sharing one are the same coordinate,
 * whichever two axes happen to be drawn. An editor keying edits by the visible
 * row and column alone would change cells on the sheets it is not showing.
 */
export function cellCoordinate(
  cell: RateMatrixCell,
  dimensions: readonly RateMatrixDimension[],
): string {
  return dimensions.map((dimension) => bucketKeyOf(cell, dimension)).join(" · ");
}

function bucketOf(cell: RateMatrixCell, dimension: RateMatrixDimension): MatrixAxisBucket {
  const position = dimension.position as MatrixAxisPosition;
  const key = bucketKeyOf(cell, dimension);

  if (isPostalRangeAxis(dimension)) {
    const postal = cellPostalRangeAt(cell, position);
    return { key, label: postalRangeLabel(postal), min: null, max: null, postal };
  }

  if (dimension.matchMode === "Exact") {
    return { key, label: key || "—", min: null, max: null };
  }

  const { min, max } = cellBoundsAt(cell, position);

  return { key, label: bandLabel(min, max), min, max };
}

/**
 * A band's printed name.
 *
 * The upper bound is exclusive, so it is shown as the point the next band takes
 * over rather than as a value this one covers. Writing "1,000 – 2,000" would
 * read as inclusive and mislead somebody checking whether 2,000 lb is priced
 * here or one tier up.
 */
export function bandLabel(min: number | null, max: number | null): string {
  const format = (value: number) => value.toLocaleString();

  if (min !== null && max !== null) return translate("{0} – under {1}", format(min), format(max));
  if (min !== null) return translate("{0} and over", format(min));
  if (max !== null) return translate("under {0}", format(max));

  return translate("Any");
}

export type BuildGridInput = {
  readonly dimensions: readonly RateMatrixDimension[];
  readonly cells: readonly RateMatrixCell[];
  readonly rowPosition: MatrixAxisPosition;
  readonly columnPosition: MatrixAxisPosition | null;
  /**
   * The bucket key fixing every axis that is neither row nor column. A matrix
   * with three or four axes is a stack of sheets, and this says which one is on
   * top.
   */
  readonly slice?: Readonly<Record<number, string>>;
};

/**
 * Project the flat cell list onto a two-axis sheet.
 *
 * Axes beyond the two being drawn are pinned by `slice`; a cell whose position
 * on a pinned axis differs is on another sheet and is left out entirely rather
 * than folded into this one.
 */
export function buildMatrixGrid(input: BuildGridInput): MatrixGrid {
  const { dimensions, cells, rowPosition, columnPosition, slice = {} } = input;

  const byPosition = new Map<number, RateMatrixDimension>();
  for (const dimension of dimensions) {
    byPosition.set(dimension.position, dimension);
  }

  const rowDimension = byPosition.get(rowPosition);
  if (!rowDimension) {
    return emptyGrid();
  }

  const columnDimension = columnPosition === null ? null : byPosition.get(columnPosition);

  const rows = new Map<string, MatrixAxisBucket>();
  const columns = new Map<string, MatrixAxisBucket>();
  const placed = new Map<string, RateMatrixCell>();
  const duplicates = new Set<string>();

  for (const cell of cells) {
    if (!onSlice(cell, dimensions, slice, rowPosition, columnPosition)) continue;

    const rowBucket = bucketOf(cell, rowDimension);
    const columnBucket = columnDimension ? bucketOf(cell, columnDimension) : SINGLE_COLUMN;

    rows.set(rowBucket.key, rowBucket);
    columns.set(columnBucket.key, columnBucket);

    const coordinate = coordinateKey(rowBucket.key, columnBucket.key);
    if (placed.has(coordinate)) {
      duplicates.add(coordinate);
      continue;
    }

    placed.set(coordinate, cell);
  }

  const orderedRows = sortBuckets([...rows.values()], rowDimension);
  const orderedColumns = columnDimension
    ? sortBuckets([...columns.values()], columnDimension)
    : [...columns.values()];

  const blanks: string[] = [];
  for (const row of orderedRows) {
    for (const column of orderedColumns) {
      const coordinate = coordinateKey(row.key, column.key);
      if (!placed.has(coordinate)) blanks.push(coordinate);
    }
  }

  return {
    rows: orderedRows,
    columns: orderedColumns,
    cells: placed,
    duplicates: [...duplicates],
    blanks,
  };
}

const SINGLE_COLUMN: MatrixAxisBucket = { key: "", label: "Rate", min: null, max: null };

export function coordinateKey(rowKey: string, columnKey: string): string {
  return `${rowKey}\u0000${columnKey}`;
}

function emptyGrid(): MatrixGrid {
  return { rows: [], columns: [], cells: new Map(), duplicates: [], blanks: [] };
}

function onSlice(
  cell: RateMatrixCell,
  dimensions: readonly RateMatrixDimension[],
  slice: Readonly<Record<number, string>>,
  rowPosition: MatrixAxisPosition,
  columnPosition: MatrixAxisPosition | null,
): boolean {
  for (const dimension of dimensions) {
    if (dimension.position === rowPosition || dimension.position === columnPosition) continue;

    const pinned = slice[dimension.position];
    if (pinned === undefined) continue;

    if (bucketKeyOf(cell, dimension) !== pinned) return false;
  }

  return true;
}

function sortBuckets(
  buckets: MatrixAxisBucket[],
  dimension: RateMatrixDimension,
): MatrixAxisBucket[] {
  if (isPostalRangeAxis(dimension)) {
    return buckets.sort((a, b) => comparePostal(a.postal?.from ?? "", b.postal?.from ?? ""));
  }

  if (dimension.matchMode === "Range") {
    return buckets.sort((a, b) => (a.min ?? -Infinity) - (b.min ?? -Infinity));
  }

  return buckets.sort((a, b) => a.key.localeCompare(b.key, undefined, { numeric: true }));
}

/**
 * The buckets a slice axis offers, so the sheet selector has something to list.
 */
export function sliceBuckets(
  dimension: RateMatrixDimension,
  cells: readonly RateMatrixCell[],
): MatrixAxisBucket[] {
  const seen = new Map<string, MatrixAxisBucket>();

  for (const cell of cells) {
    const bucket = bucketOf(cell, dimension);
    seen.set(bucket.key, bucket);
  }

  return sortBuckets([...seen.values()], dimension);
}

function toNumber(value: string | number | null | undefined): number | null {
  if (value === null || value === undefined || value === "") return null;

  const parsed = typeof value === "number" ? value : Number(value);

  return Number.isFinite(parsed) ? parsed : null;
}

/** The postal ranges an axis holds, each once, in order. */
export function postalRangesOf(
  dimension: RateMatrixDimension,
  cells: readonly RateMatrixCell[],
): PostalRange[] {
  return sliceBuckets(dimension, cells).flatMap((bucket) => (bucket.postal ? [bucket.postal] : []));
}

/** The cells with every cell of one postal range moved to another, on every sheet. */
export function rerangePostalAxis(
  cells: readonly RateMatrixCell[],
  dimension: RateMatrixDimension,
  from: PostalRange,
  to: PostalRange,
): RateMatrixCell[] {
  const position = dimension.position as MatrixAxisPosition;
  const key = postalBucketKey(from);
  return cells.map((cell) =>
    bucketKeyOf(cell, dimension) === key ? withCellPostalRange(cell, position, to) : cell,
  );
}

/** The cells without one postal range's cells, on every sheet. */
export function withoutPostalRange(
  cells: readonly RateMatrixCell[],
  dimension: RateMatrixDimension,
  range: PostalRange,
): RateMatrixCell[] {
  const key = postalBucketKey(range);
  return cells.filter((cell) => bucketKeyOf(cell, dimension) !== key);
}

export type AddPostalRangeInput = {
  readonly dimensions: readonly RateMatrixDimension[];
  readonly cells: readonly RateMatrixCell[];
  /** The postal range axis the range joins. */
  readonly position: MatrixAxisPosition;
  readonly range: PostalRange;
};

/**
 * The cells a new postal range adds: one for every coordinate the other axes already price on
 * any sheet, so the range arrives as a full row (or column, or sheet) on every sheet for the
 * person to fill in. Their values are zero until they do. A matrix whose only axis is the range
 * gets one cell; one whose other axes price nothing yet gets none.
 */
export function addPostalRange(input: AddPostalRangeInput): RateMatrixCell[] {
  const { dimensions, cells, position, range } = input;
  const others = dimensions.filter((dimension) => dimension.position !== position);
  if (others.length === 0) {
    return [withCellPostalRange(blankCell(), position, range)];
  }

  const templates = new Map<string, RateMatrixCell>();
  for (const cell of cells) {
    const coordinate = cellCoordinate(cell, others);
    if (!templates.has(coordinate)) templates.set(coordinate, cell);
  }

  return [...templates.values()].map((template) => ({
    ...withCellPostalRange(template, position, range),
    id: undefined,
    rateMatrixId: undefined,
    value: 0,
    minCharge: undefined,
    deficitEligible: true,
  }));
}

function blankCell(): RateMatrixCell {
  return {
    d0Key: "",
    d1Key: "",
    d2Key: "",
    d3Key: "",
    d0KeyTo: "",
    d1KeyTo: "",
    d2KeyTo: "",
    d3KeyTo: "",
    value: 0,
    deficitEligible: true,
  } as RateMatrixCell;
}
