import { hoverTooltip, type Tooltip } from "@codemirror/view";
import { translateMessage } from "@trenova/shared/i18n/message";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { translate } from "@trenova/shared/i18n/runtime";
import { formulaIdentifier, type FormulaIdentifier } from "@trenova/shared/rules/formula";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import type { FormulaValueSource } from "@trenova/shared/types/formula-template";
import { lostFocus } from "./expr-signature";
import { itemFieldDoc, type KnownIdentifiers, type VariableDoc } from "./known-identifiers";

export type HoverVariable = {
  name: string;
  value: unknown;
  source: FormulaValueSource;
};

const SOURCE_WORDS: Record<FormulaValueSource, string> = defineLabels({
  field: "from shipment",
  computed: "computed from shipment",
  input: "your input",
  override: "engine override",
  default: "declared default",
  sample: "sample data",
  provided: "market data feed",
});

/** The identifier under a document position and what it is, as the rules module reads it. */
export function findHoveredIdentifier(
  rules: RulesRuntime,
  doc: string,
  pos: number,
  known: KnownIdentifiers,
): FormulaIdentifier | null {
  return formulaIdentifier(rules, { source: doc, position: pos, schema: known.schema });
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) return translate("empty");
  if (typeof value === "string") return JSON.stringify(value);
  if (typeof value === "number" || typeof value === "boolean" || typeof value === "bigint") {
    return String(value);
  }
  if (Array.isArray(value)) {
    return translate("{0, plural, one {# item} other {# items}}", value.length);
  }
  if (typeof value === "object") {
    return translate(
      "{0, plural, one {# field} other {# fields}}",
      Object.keys(value as Record<string, unknown>).length,
    );
  }
  return String(value);
}

/** One line: what the variable was in the last preview and where it came from. */
export function formatHoverValue(variable: HoverVariable): string {
  return translate(
    "{0} = {1} ({2})",
    variable.name,
    formatValue(variable.value),
    SOURCE_WORDS[variable.source],
  );
}

/** What a schema type is called on the card. */
function typeWords(type: string): string {
  switch (type.toLowerCase()) {
    case "number":
    case "integer":
    case "decimal":
      return translate("a number");
    case "string":
    case "text":
      return translate("text");
    case "boolean":
      return translate("true/false");
    case "date":
    case "datetime":
      return translate("a date");
    case "array":
    case "list":
      return translate("a list");
    case "object":
      return translate("a record");
    default:
      return translate("a value");
  }
}

const POINTER_WORDS: Record<string, string> = defineLabels({
  "#": "The item the predicate is looking at",
  "#index": "The item's position in the list, from 0",
  "#acc": "What reduce has added up so far",
});

function element(tag: string, className: string, text?: string): HTMLElement {
  const node = document.createElement(tag);
  node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

function tag(text: string, tone?: "warn" | "bad"): HTMLElement {
  // i18n-ignore: CSS class names, not text
  const className = tone ? `cm-expr-hover-tag cm-expr-hover-tag-${tone}` : "cm-expr-hover-tag";
  return element("span", className, text);
}

function variableBody(card: HTMLElement, head: HTMLElement, doc: VariableDoc) {
  head.appendChild(tag(typeWords(doc.type)));
  if (doc.custom) head.appendChild(tag(translate("input")));
  if (doc.nullable) head.appendChild(tag(translate("can be empty"), "warn"));
  card.appendChild(head);
  if (doc.description) card.appendChild(element("div", "cm-expr-hover-body", doc.description));
  if (doc.enum && doc.enum.length > 0) {
    card.appendChild(
      element("div", "cm-expr-hover-body", translate("One of: {0}", doc.enum.join(", "))),
    );
  }
}

function unknownBody(card: HTMLElement, head: HTMLElement, text: string) {
  head.appendChild(tag(translate("unknown"), "bad"));
  card.appendChild(head);
  card.appendChild(element("div", "cm-expr-hover-body", text));
}

/**
 * The card drawn for a name: what it is, what it holds on the preview load,
 * and where that came from. Null when there is nothing to say, such as a name
 * the schema has not arrived to vouch for.
 */
export function renderHoverCard(
  hovered: FormulaIdentifier,
  known: KnownIdentifiers,
  value: HoverVariable | undefined,
): HTMLElement | null {
  const card = element("div", "cm-expr-hover");
  const head = element("div", "cm-expr-hover-head");
  const name = element("code", "cm-expr-hover-name", hovered.name);
  head.appendChild(name);

  switch (hovered.kind) {
    case "function": {
      const fn = known.callables.get(hovered.name);
      if (fn) {
        name.textContent = fn.signature;
        head.appendChild(tag(translate("function")));
        card.appendChild(head);
        card.appendChild(element("div", "cm-expr-hover-body", translateMessage(fn.description)));
        card.appendChild(element("code", "cm-expr-hover-example", translateMessage(fn.example)));
        return card;
      }
      const disabled = known.disabled.get(hovered.name);
      unknownBody(
        card,
        head,
        disabled
          ? translateMessage(disabled.message)
          : translate("Not a formula function. Pick one from the reference."),
      );
      return card;
    }
    case "method":
      head.appendChild(tag(translate("method")));
      card.appendChild(head);
      return card;
    case "let":
      head.appendChild(tag(translate("let, this formula's own")));
      card.appendChild(head);
      return card;
    case "pointer": {
      const words = POINTER_WORDS[hovered.name];
      card.appendChild(head);
      if (words) card.appendChild(element("div", "cm-expr-hover-body", words));
      return card;
    }
    case "field": {
      const doc = itemFieldDoc(known, hovered.name);
      if (!doc) return null;
      variableBody(card, head, doc);
      return card;
    }
    case "unknown":
      unknownBody(
        card,
        head,
        translate(
          "Not a shipment field or one of this template's inputs. Declare it under Inputs, or pick a field from the reference.",
        ),
      );
      return card;
    default: {
      const doc = known.variables.find((item) => item.name === hovered.name);
      if (doc) {
        variableBody(card, head, doc);
      } else if (known.ready && known.variableRoots.has(hovered.name.split(".")[0])) {
        head.appendChild(tag(translate("record")));
        card.appendChild(head);
      } else {
        return null;
      }
    }
  }

  if (value) {
    const line = element("div", "cm-expr-hover-value");
    line.appendChild(
      element("code", "cm-expr-hover-value-text", translate("= {0}", formatValue(value.value))),
    );
    line.appendChild(element("span", "cm-expr-hover-source", SOURCE_WORDS[value.source]));
    card.appendChild(line);
  }
  return card;
}

/**
 * Shows, on hover, what a name is and what it resolved to in the most recent
 * preview. The getters are read at hover time so the extension is built once
 * and stays current as previews run and inputs change. The card leaves with
 * the editor's focus, so it never floats over a dialog.
 */
export function createExprHover(
  getRules: () => RulesRuntime,
  getValues: () => ReadonlyMap<string, HoverVariable>,
  getKnown: () => KnownIdentifiers,
) {
  return hoverTooltip(
    (view, pos): Tooltip | null => {
      const known = getKnown();
      const hovered = findHoveredIdentifier(getRules(), view.state.doc.toString(), pos, known);
      if (!hovered) return null;
      const card = renderHoverCard(hovered, known, getValues().get(hovered.name));
      if (!card) return null;
      return {
        pos: hovered.from,
        end: hovered.to,
        above: true,
        create: () => ({ dom: card }),
      };
    },
    { hoverTime: 250, hideOn: lostFocus },
  );
}
