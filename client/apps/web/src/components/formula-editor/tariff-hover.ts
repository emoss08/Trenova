// tariff-hover.ts is the hover card from `tariff.hover`: what the name under the pointer is, its
// type with its unit, and its doc, plus what it came to in the last preview when the trace bound
// it.

import { hoverTooltip, type Tooltip } from "@codemirror/view";
import { translate } from "@trenova/shared/i18n/runtime";
import { tariffHover, type TariffHover } from "@trenova/shared/rules/tariff";
import type { TariffTraceEvent } from "@trenova/shared/types/tariff";
import { appendTariffDoc, unlessUnanswered } from "./tariff-doc";
import type { TariffLanguageService } from "./tariff-service";

/**
 * The value the last preview gave the name under a position: the latest `let`, `var` or loop
 * binding of that name whose statement contains the position.
 */
export function previewValueAt(
  trace: readonly TariffTraceEvent[],
  name: string,
  position: number,
): string | null {
  let found: string | null = null;
  for (const event of trace) {
    if (event.kind !== "bind" && event.kind !== "assign" && event.kind !== "iteration") continue;
    if (event.name !== name) continue;
    if (event.range.from <= position && position <= event.range.to) found = event.display;
  }
  if (found !== null) return found;
  for (const event of trace) {
    if ((event.kind === "bind" || event.kind === "assign") && event.name === name) {
      found = event.display;
    }
  }
  return found;
}

/** The hover card's element. */
export function renderTariffHover(hover: NonNullable<TariffHover>, value: string | null) {
  const dom = document.createElement("div");
  dom.className = "cm-tariff-hover";
  const head = document.createElement("div");
  head.className = "cm-tariff-hover-head";
  const title = document.createElement("code");
  title.className = "cm-tariff-hover-title";
  title.textContent = hover.title;
  head.appendChild(title);
  if (hover.type) {
    const type = document.createElement("span");
    type.className = "cm-tariff-hover-type";
    type.textContent = hover.type;
    head.appendChild(type);
  }
  dom.appendChild(head);
  const body = document.createElement("div");
  body.className = "cm-tariff-hover-body";
  if (appendTariffDoc(body, hover.doc)) dom.appendChild(body);
  if (value !== null) {
    const row = document.createElement("div");
    row.className = "cm-tariff-hover-value";
    row.textContent = translate("In the preview: {0}", value);
    dom.appendChild(row);
  }
  return dom;
}

export function tariffHoverCard(service: TariffLanguageService) {
  return hoverTooltip((view, position): Tooltip | null => {
    const source = view.state.doc.toString();
    const hover = unlessUnanswered(
      () =>
        tariffHover(service.rules(), {
          source,
          position,
          kind: service.kind(),
          context: service.context(),
        }),
      null,
    );
    if (!hover) return null;
    const name = source.slice(hover.range.from, hover.range.to);
    const value = previewValueAt(service.trace(), name, position);
    return {
      pos: hover.range.from,
      end: hover.range.to,
      above: true,
      create: () => ({ dom: renderTariffHover(hover, value) }),
    };
  });
}
