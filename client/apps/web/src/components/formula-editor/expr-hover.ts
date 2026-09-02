import { hoverTooltip, type Tooltip } from "@codemirror/view";
import type { FormulaValueSource } from "@trenova/shared/types/formula-template";
import { collectIdentifierTokens } from "./expr-lint";

export type HoverVariable = {
  name: string;
  value: unknown;
  source: FormulaValueSource;
};

export type HoveredIdentifier = { name: string; from: number; to: number };

const SOURCE_WORDS: Record<FormulaValueSource, string> = {
  field: "from shipment",
  computed: "computed from shipment",
  input: "your input",
  override: "engine override",
  default: "declared default",
  sample: "sample data",
  provided: "market data feed",
};

/** The variable identifier under a document position, if the position is on one. */
export function findHoveredIdentifier(doc: string, pos: number): HoveredIdentifier | null {
  for (const token of collectIdentifierTokens(doc)) {
    if (token.isCall) continue;
    if (pos >= token.from && pos < token.to) {
      return { name: token.name, from: token.from, to: token.to };
    }
  }
  return null;
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) return "empty";
  if (typeof value === "string") return JSON.stringify(value);
  if (typeof value === "number" || typeof value === "boolean" || typeof value === "bigint") {
    return String(value);
  }
  return JSON.stringify(value);
}

/** One line: what the variable was in the last preview and where it came from. */
export function formatHoverValue(variable: HoverVariable): string {
  return `${variable.name} = ${formatValue(variable.value)} (${SOURCE_WORDS[variable.source]})`;
}

/**
 * Shows, on hover, what an identifier resolved to in the most recent preview.
 * The getter is read at hover time so the extension is built once and stays
 * current as previews run.
 */
export function createExprHover(getValues: () => ReadonlyMap<string, HoverVariable>) {
  return hoverTooltip((view, pos): Tooltip | null => {
    const hovered = findHoveredIdentifier(view.state.doc.toString(), pos);
    if (!hovered) return null;
    const variable = getValues().get(hovered.name);
    if (!variable) return null;

    return {
      pos: hovered.from,
      end: hovered.to,
      above: true,
      create: () => {
        const dom = document.createElement("div");
        dom.className = "cm-expr-hover";
        dom.textContent = formatHoverValue(variable);
        return { dom };
      },
    };
  });
}
