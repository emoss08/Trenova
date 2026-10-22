// tariff-hover.ts is the hover card from `tariff.hover`: what the thing under the pointer is (its
// declaration as Tariff code, its type and unit, its doc) and, given the last preview's trace,
// what a binding came to.

import { hoverTooltip, type Tooltip } from "@codemirror/view";
import { translate } from "@trenova/shared/i18n/runtime";
import { tariffHover, type TariffHover } from "@trenova/shared/rules/tariff";
import { appendTariffDoc, unlessUnanswered } from "./tariff-doc";
import type { TariffLanguageService } from "./tariff-service";

/** The hover card's element. */
export function renderTariffHover(hover: NonNullable<TariffHover>): HTMLElement {
  const { contents } = hover;
  const dom = document.createElement("div");
  dom.className = "cm-tariff-hover";
  if (contents.signature || contents.type) {
    const head = document.createElement("div");
    head.className = "cm-tariff-hover-head";
    if (contents.signature) {
      const signature = document.createElement("code");
      signature.className = "cm-tariff-hover-title";
      signature.textContent = contents.signature;
      head.appendChild(signature);
    }
    const type = [contents.type, contents.unit].filter(Boolean).join(" · ");
    if (type && contents.type !== contents.signature) {
      const tag = document.createElement("span");
      tag.className = "cm-tariff-hover-type";
      tag.textContent = type;
      head.appendChild(tag);
    }
    dom.appendChild(head);
  }
  const body = document.createElement("div");
  body.className = "cm-tariff-hover-body";
  if (appendTariffDoc(body, contents.doc)) dom.appendChild(body);
  if (contents.value) {
    const row = document.createElement("div");
    row.className = "cm-tariff-hover-value";
    row.textContent = translate("In the preview: {0}", contents.value);
    dom.appendChild(row);
  }
  return dom;
}

export function tariffHoverCard(service: TariffLanguageService) {
  return hoverTooltip((view, position): Tooltip | null => {
    const hover = unlessUnanswered(
      () =>
        tariffHover(service.rules(), {
          source: view.state.doc.toString(),
          offset: position,
          context: service.context(),
          values: service.trace(),
        }),
      null,
    );
    if (!hover) return null;
    return {
      pos: hover.range.from,
      end: hover.range.to,
      above: true,
      create: () => ({ dom: renderTariffHover(hover) }),
    };
  });
}
