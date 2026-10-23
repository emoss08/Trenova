// tariff-hover.ts is the hover card from `tariff.hover`: what the thing under the pointer is (its
// declaration as Tariff code, its type and unit, its doc, an input's bounds) and, given the last
// preview's trace, what a binding came to.

import type { Extension } from "@codemirror/state";
import { hoverTooltip, type Command, type EditorView, type Tooltip } from "@codemirror/view";
import { translateMessage } from "@trenova/shared/i18n/message";
import { translate } from "@trenova/shared/i18n/runtime";
import { formatShownValue } from "@trenova/shared/lib/tariff-value";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import { tariffHover, type TariffHover } from "@trenova/shared/rules/tariff";
import { cursorTooltip, openCursorTooltip } from "./cursor-tooltip";
import { appendTariffCode, appendTariffDoc, unlessUnanswered } from "./tariff-doc";
import { tariffServiceFacet, type TariffLanguageService } from "./tariff-service";

/** The hover card's element. */
export function renderTariffHover(
  hover: NonNullable<TariffHover>,
  rules: RulesRuntime,
): HTMLElement {
  const { contents } = hover;
  const dom = document.createElement("div");
  dom.className = "cm-tariff-hover";
  if (contents.signature || contents.type) {
    const head = document.createElement("div");
    head.className = "cm-tariff-hover-head";
    if (contents.signature) {
      const signature = document.createElement("code");
      signature.className = "cm-tariff-hover-title";
      appendTariffCode(signature, contents.signature, rules);
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
  if (contents.constraint) {
    const row = document.createElement("div");
    row.className = "cm-tariff-hover-constraint";
    const label = document.createElement("span");
    label.className = "cm-tariff-hover-label";
    label.textContent = translate("Bounds");
    const bound = document.createElement("span");
    bound.textContent = translateMessage(contents.constraint);
    row.append(label, bound);
    dom.appendChild(row);
  }
  if (contents.value) {
    const row = document.createElement("div");
    row.className = "cm-tariff-hover-value";
    row.textContent =
      contents.value.times > 1
        ? translate(
            "In the preview: {0}, the last of {1} times it was bound",
            formatShownValue(contents.value, translate),
            contents.value.times,
          )
        : translate("In the preview: {0}", formatShownValue(contents.value, translate));
    dom.appendChild(row);
  }
  return dom;
}

function hoverAt(service: TariffLanguageService, view: EditorView, position: number) {
  return unlessUnanswered(
    () =>
      tariffHover(service.rules(), {
        source: view.state.doc.toString(),
        offset: position,
        context: service.context(),
        kind: service.kind(),
        values: service.trace(),
      }),
    null,
  );
}

/** Opens the hover card for the name under the cursor (`K` in Vim); false when there is none. */
export const showHoverAtCursor: Command = (view) => {
  const service = view.state.facet(tariffServiceFacet);
  if (!service) return false;
  const hover = hoverAt(service, view, view.state.selection.main.head);
  if (!hover) return false;
  openCursorTooltip(view, {
    pos: hover.range.from,
    end: hover.range.to,
    create: () => renderTariffHover(hover, service.rules()),
  });
  return true;
};

export function tariffHoverCard(service: TariffLanguageService): Extension {
  return [
    tariffServiceFacet.of(service),
    cursorTooltip,
    hoverTooltip((view, position): Tooltip | null => {
      const hover = hoverAt(service, view, position);
      if (!hover) return null;
      return {
        pos: hover.range.from,
        end: hover.range.to,
        above: true,
        create: () => ({ dom: renderTariffHover(hover, service.rules()) }),
      };
    }),
  ];
}
