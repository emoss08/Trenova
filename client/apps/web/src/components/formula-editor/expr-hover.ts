import { translate } from "@trenova/shared/i18n/runtime";
import { hoverTooltip, type Tooltip } from "@codemirror/view";
import type { FormulaValueSource } from "@trenova/shared/types/formula-template";
import { normalizeType, typeWords } from "./expr-check";
import { collectIdentifierTokens, letBindings } from "./expr-lint";
import { isKnownFunction, type KnownIdentifiers } from "./known-identifiers";

export type HoverVariable = {
  name: string;
  value: unknown;
  source: FormulaValueSource;
};

export type HoveredIdentifier = { name: string; from: number; to: number; isCall: boolean };

const SOURCE_WORDS: Record<FormulaValueSource, string> = {
  field: "from shipment",
  computed: "computed from shipment",
  input: "your input",
  override: "engine override",
  default: "declared default",
  sample: "sample data",
  provided: "market data feed",
};

/** The identifier under a document position, if the position is on one. */
export function findHoveredIdentifier(doc: string, pos: number): HoveredIdentifier | null {
  for (const token of collectIdentifierTokens(doc)) {
    if (pos >= token.from && pos < token.to) {
      return { name: token.name, from: token.from, to: token.to, isCall: token.isCall };
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

function element(tag: string, className: string, text?: string): HTMLElement {
  const node = document.createElement(tag);
  node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

/** The card drawn for a name: what it is, what it holds on the preview load, and where that came from. */
export function renderHoverCard(
  hovered: HoveredIdentifier,
  known: KnownIdentifiers,
  bindings: ReadonlySet<string>,
  value: HoverVariable | undefined,
): HTMLElement | null {
  const card = element("div", "cm-expr-hover");
  const head = element("div", "cm-expr-hover-head");
  const name = element("code", "cm-expr-hover-name", hovered.name);
  head.appendChild(name);

  const root = hovered.name.split(".")[0];
  const fn = hovered.isCall
    ? known.functions.find((item) => item.name === hovered.name)
    : undefined;
  const doc = known.variables.find((item) => item.name === hovered.name);

  if (hovered.isCall) {
    if (fn) {
      name.textContent = fn.signature;
      head.appendChild(element("span", "cm-expr-hover-tag", translate("function")));
      card.appendChild(head);
      if (fn.description) card.appendChild(element("div", "cm-expr-hover-body", fn.description));
      if (fn.example) card.appendChild(element("code", "cm-expr-hover-example", fn.example));
      return card;
    }
    if (isKnownFunction(known, hovered.name)) {
      head.appendChild(element("span", "cm-expr-hover-tag", translate("built-in function")));
      card.appendChild(head);
      return card;
    }
    head.appendChild(
      element("span", "cm-expr-hover-tag cm-expr-hover-tag-bad", translate("unknown")),
    );
    card.appendChild(head);
    card.appendChild(
      element(
        "div",
        "cm-expr-hover-body",
        translate("Not a formula function. Pick one from the reference."),
      ),
    );
    return card;
  }

  if (bindings.has(root)) {
    head.appendChild(element("span", "cm-expr-hover-tag", translate("let, this formula's own")));
    card.appendChild(head);
  } else if (doc) {
    head.appendChild(element("span", "cm-expr-hover-tag", typeWords(normalizeType(doc.type))));
    if (doc.custom) head.appendChild(element("span", "cm-expr-hover-tag", translate("input")));
    if (doc.nullable) {
      head.appendChild(
        element("span", "cm-expr-hover-tag cm-expr-hover-tag-warn", translate("can be empty")),
      );
    }
    card.appendChild(head);
    if (doc.description) card.appendChild(element("div", "cm-expr-hover-body", doc.description));
    if (doc.enum && doc.enum.length > 0) {
      card.appendChild(
        element("div", "cm-expr-hover-body", translate("One of: {0}", doc.enum.join(", "))),
      );
    }
  } else if (known.variableRoots.has(root)) {
    head.appendChild(element("span", "cm-expr-hover-tag", translate("record")));
    card.appendChild(head);
  } else {
    head.appendChild(
      element("span", "cm-expr-hover-tag cm-expr-hover-tag-bad", translate("unknown")),
    );
    card.appendChild(head);
    card.appendChild(
      element(
        "div",
        "cm-expr-hover-body",
        translate(
          "Not a shipment field or one of this template's inputs. Declare it under Inputs, or pick a field from the reference.",
        ),
      ),
    );
    return card;
  }

  if (value) {
    const line = element("div", "cm-expr-hover-value");
    line.appendChild(element("code", "cm-expr-hover-value-text", `= ${formatValue(value.value)}`));
    line.appendChild(element("span", "cm-expr-hover-source", SOURCE_WORDS[value.source]));
    card.appendChild(line);
  }
  return card;
}

/**
 * Shows, on hover, what a name is and what it resolved to in the most recent
 * preview. The getters are read at hover time so the extension is built once
 * and stays current as previews run and inputs change.
 */
export function createExprHover(
  getValues: () => ReadonlyMap<string, HoverVariable>,
  getKnown: () => KnownIdentifiers,
) {
  return hoverTooltip(
    (view, pos): Tooltip | null => {
      const doc = view.state.doc.toString();
      const hovered = findHoveredIdentifier(doc, pos);
      if (!hovered) return null;
      const card = renderHoverCard(
        hovered,
        getKnown(),
        letBindings(doc),
        getValues().get(hovered.name),
      );
      if (!card) return null;
      return {
        pos: hovered.from,
        end: hovered.to,
        above: true,
        create: () => ({ dom: card }),
      };
    },
    { hoverTime: 250 },
  );
}
