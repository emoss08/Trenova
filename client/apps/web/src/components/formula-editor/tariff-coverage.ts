// tariff-coverage.ts draws the last test run's coverage in the gutter: a bar beside every line of
// a region no test or property case reached, and on hover a tooltip saying so and what kind of
// region it is. The regions follow edits until the next run replaces them. With no coverage
// (the file does not check, a test waits for table cells, or nothing has run) the gutter is not
// drawn at all.

import {
  Compartment,
  RangeSet,
  StateEffect,
  StateField,
  type EditorState,
  type Extension,
} from "@codemirror/state";
import { EditorView, GutterMarker, gutter, showTooltip, type Tooltip } from "@codemirror/view";
import { TARIFF_UNREACHED_LABELS, unreachedRegions } from "@/lib/tariff-coverage";
import type { TariffCoverage, TariffRegionKind } from "@trenova/shared/types/tariff";

type Gap = { from: number; to: number; kind: TariffRegionKind };
type Hover = { pos: number; kind: TariffRegionKind };
type CoverageState = { gaps: readonly Gap[]; hover: Hover | null };

const setCoverage = StateEffect.define<TariffCoverage | null>();
const setHover = StateEffect.define<Hover | null>();

const coverageField = StateField.define<CoverageState>({
  create: () => ({ gaps: [], hover: null }),
  update(value, transaction) {
    let next = value;
    if (transaction.docChanged) {
      next = {
        hover: null,
        gaps: value.gaps.map((gap) => ({
          ...gap,
          from: transaction.changes.mapPos(gap.from, 1),
          to: transaction.changes.mapPos(gap.to, -1),
        })),
      };
    }
    for (const effect of transaction.effects) {
      if (effect.is(setCoverage)) {
        next = {
          hover: null,
          gaps: unreachedRegions(effect.value).map((region) => ({
            from: region.range.from,
            to: region.range.to,
            kind: region.kind,
          })),
        };
      }
      if (effect.is(setHover)) next = { ...next, hover: effect.value };
    }
    return next;
  },
  provide: (field) =>
    showTooltip.from(field, (state): Tooltip | null => {
      const hover = state.hover;
      if (hover === null) return null;
      return {
        pos: hover.pos,
        above: true,
        create: () => {
          const dom = document.createElement("div");
          dom.className = "cm-tariff-coverage-tip";
          dom.textContent = TARIFF_UNREACHED_LABELS[hover.kind];
          return { dom };
        },
      };
    }),
});

class GapMarker extends GutterMarker {
  readonly kind: TariffRegionKind;

  constructor(kind: TariffRegionKind) {
    super();
    this.kind = kind;
  }

  override eq(other: GutterMarker): boolean {
    return other instanceof GapMarker && other.kind === this.kind;
  }

  override toDOM() {
    const dom = document.createElement("span");
    dom.className = "cm-tariff-coverage-gap";
    dom.setAttribute("aria-label", TARIFF_UNREACHED_LABELS[this.kind]);
    return dom;
  }
}

/** The line starts each unreached region covers, each once, marked with its region's kind. */
export function coverageGapLines(
  state: EditorState,
): { line: number; from: number; kind: TariffRegionKind }[] {
  const seen = new Map<number, { line: number; from: number; kind: TariffRegionKind }>();
  const length = state.doc.length;
  for (const gap of state.field(coverageField).gaps) {
    if (gap.from > length) continue;
    const first = state.doc.lineAt(Math.min(gap.from, length)).number;
    const last = state.doc.lineAt(Math.min(Math.max(gap.to, gap.from), length)).number;
    for (let number = first; number <= last; number += 1) {
      if (seen.has(number)) continue;
      seen.set(number, { line: number, from: state.doc.line(number).from, kind: gap.kind });
    }
  }
  return [...seen.values()].sort((a, b) => a.from - b.from);
}

/**
 * Shows the gutter's tooltip for the line starting at `lineFrom` when no test reaches it, and
 * hides it for any other line or for null (the pointer left the gutter).
 */
export function showCoverageTip(view: EditorView, lineFrom: number | null): void {
  const gap =
    lineFrom === null
      ? undefined
      : coverageGapLines(view.state).find((candidate) => candidate.from === lineFrom);
  const hover = gap ? { pos: gap.from, kind: gap.kind } : null;
  const current = view.state.field(coverageField).hover;
  if (current?.pos === hover?.pos && current?.kind === hover?.kind) return;
  view.dispatch({ effects: setHover.of(hover) });
}

const coverageGutter = gutter({
  class: "cm-tariff-coverage-gutter",
  markers: (view) =>
    RangeSet.of(coverageGapLines(view.state).map((gap) => new GapMarker(gap.kind).range(gap.from))),
  domEventHandlers: {
    mouseover(view, line) {
      showCoverageTip(view, line.from);
      return false;
    },
    mouseleave(view) {
      showCoverageTip(view, null);
      return false;
    },
  },
});

const gutterSlot = new Compartment();

/** The coverage state, with the gutter left out until there is coverage to draw. */
export function tariffCoverage(): Extension {
  return [coverageField, gutterSlot.of([])];
}

/** Draws a run's coverage, or hides the gutter when there is none. */
export function setTariffCoverage(view: EditorView, coverage: TariffCoverage | null): void {
  if (view.state.field(coverageField, false) === undefined) return;
  view.dispatch({
    effects: [setCoverage.of(coverage), gutterSlot.reconfigure(coverage ? coverageGutter : [])],
  });
}
