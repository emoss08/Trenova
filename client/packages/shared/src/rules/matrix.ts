// Typed calls for the rules module's rate matrix operations (`matrix.*`). The browser checks a
// whole matrix for gaps, overlaps, duplicate coordinates and a capped top band in one call; the
// rules are in docs/engineering/rules-engine.md, "Rate matrices", and testdata/matrix.json pins
// them.

import type { CatalogMessage } from "@trenova/shared/i18n/message";
import type { RulesRuntime } from "./runtime";
import { type DecimalInput, type DecimalString, wireDecimal, wireInteger, wireText } from "./wire";

/** The highest axis position a matrix can declare; a matrix has at most four axes. */
const MAX_MATRIX_POSITION = 3;

/** One axis. `label` is what messages call it: pass the translated name a screen shows. */
export type MatrixDimensionInput = {
  readonly position: number;
  readonly matchMode?: string | null;
  readonly kind?: string | null;
  readonly label?: string | null;
};

/** One cell: its key (and a postal range's last key) and band on each axis; other fields are ignored. */
export type MatrixCellInput = {
  readonly id?: string | null;
  readonly d0Key?: string | null;
  readonly d1Key?: string | null;
  readonly d2Key?: string | null;
  readonly d3Key?: string | null;
  readonly d0KeyTo?: string | null;
  readonly d1KeyTo?: string | null;
  readonly d2KeyTo?: string | null;
  readonly d3KeyTo?: string | null;
  readonly d0Min?: DecimalInput;
  readonly d0Max?: DecimalInput;
  readonly d1Min?: DecimalInput;
  readonly d1Max?: DecimalInput;
  readonly d2Min?: DecimalInput;
  readonly d2Max?: DecimalInput;
  readonly d3Min?: DecimalInput;
  readonly d3Max?: DecimalInput;
};

export type MatrixCoverageKind = "gap" | "overlap" | "duplicate" | "unbounded" | "empty";

/** Something wrong with a matrix that the engine will not complain about. */
export type MatrixCoverageIssue = {
  readonly kind: MatrixCoverageKind;
  /** The banded axis the finding is on; null for the whole matrix. */
  readonly position: number | null;
  readonly message: CatalogMessage;
  /** An overlap's two bands. */
  readonly bands?: readonly { min: DecimalString | null; max: DecimalString | null }[];
  /** The other axes' coordinates of every slice with the same finding. */
  readonly slices?: readonly string[];
};

function wireCell(cell: MatrixCellInput | null | undefined) {
  if (!cell) return null;
  return {
    id: wireText(cell.id) || null,
    d0Key: wireText(cell.d0Key),
    d1Key: wireText(cell.d1Key),
    d2Key: wireText(cell.d2Key),
    d3Key: wireText(cell.d3Key),
    d0KeyTo: wireText(cell.d0KeyTo),
    d1KeyTo: wireText(cell.d1KeyTo),
    d2KeyTo: wireText(cell.d2KeyTo),
    d3KeyTo: wireText(cell.d3KeyTo),
    d0Min: wireDecimal(cell.d0Min),
    d0Max: wireDecimal(cell.d0Max),
    d1Min: wireDecimal(cell.d1Min),
    d1Max: wireDecimal(cell.d1Max),
    d2Min: wireDecimal(cell.d2Min),
    d2Max: wireDecimal(cell.d2Max),
    d3Min: wireDecimal(cell.d3Min),
    d3Max: wireDecimal(cell.d3Max),
  };
}

/**
 * The axes the module can read. An axis being edited may not have a valid position yet; it is
 * left out of the check rather than refusing it, and the form's own schema reports it.
 */
function wireDimensions(dimensions: readonly MatrixDimensionInput[]) {
  return dimensions.flatMap((dimension) => {
    const position = wireInteger(dimension.position);
    if (position === null || position < 0 || position > MAX_MATRIX_POSITION) return [];
    return [
      {
        position,
        matchMode: wireText(dimension.matchMode),
        kind: wireText(dimension.kind),
        label: wireText(dimension.label),
      },
    ];
  });
}

/** `matrix.coverage`: every finding for one matrix. */
export function matrixCoverage(
  rules: RulesRuntime,
  input: {
    readonly dimensions: readonly MatrixDimensionInput[];
    readonly cells: readonly (MatrixCellInput | null | undefined)[];
  },
): MatrixCoverageIssue[] {
  const output = rules.call("matrix.coverage", {
    dimensions: wireDimensions(input.dimensions),
    cells: input.cells.map(wireCell),
  }) as { issues: MatrixCoverageIssue[] };
  return output.issues;
}
