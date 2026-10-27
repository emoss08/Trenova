// lading-hover.ts is the hover card from `lading.hover`: what the thing under the pointer is (its
// declaration or the guide's description, coloured as Lading), its type, its doc and, given the
// last run, what it came to and where that was read from. `K` in Vim opens it at the cursor.

import type { Extension } from "@codemirror/state";
import { hoverTooltip, type Command, type EditorView, type Tooltip } from "@codemirror/view";
import { ladingHover, type LadingHover } from "@trenova/shared/edi/lading";
import type { EdiRuntime } from "@trenova/shared/edi/runtime";
import { translate } from "@trenova/shared/i18n/runtime";
import { cursorTooltip, openCursorTooltip } from "../formula-editor/cursor-tooltip";
import { appendLadingCode, appendLadingDoc } from "./lading-doc";
import {
  ladingRequest,
  ladingServiceFacet,
  unlessRefused,
  type LadingLanguageService,
} from "./lading-service";

function valueRow(text: string): HTMLElement {
  const row = document.createElement("div");
  row.className = "cm-tariff-hover-value";
  row.textContent = text;
  return row;
}

/** The hover card's element. */
export function renderLadingHover(hover: LadingHover, edi: EdiRuntime): HTMLElement {
  const { contents } = hover;
  const dom = document.createElement("div");
  dom.className = "cm-tariff-hover";
  if (contents.signature || contents.type) {
    const head = document.createElement("div");
    head.className = "cm-tariff-hover-head";
    if (contents.signature) {
      const signature = document.createElement("code");
      signature.className = "cm-tariff-hover-title";
      appendLadingCode(signature, contents.signature, edi);
      head.appendChild(signature);
    }
    if (contents.type && contents.type !== contents.signature) {
      const type = document.createElement("span");
      type.className = "cm-tariff-hover-type";
      type.textContent = contents.type;
      head.appendChild(type);
    }
    dom.appendChild(head);
  }
  const body = document.createElement("div");
  body.className = "cm-tariff-hover-body";
  if (appendLadingDoc(body, contents.doc)) dom.appendChild(body);
  if (contents.value) {
    dom.appendChild(valueRow(translate("In the last run: {0}", contents.value.value)));
    const sources = contents.value.sources ?? [];
    if (sources.length > 0)
      dom.appendChild(valueRow(translate("Read from {0}", sources.join(", "))));
    const steps = contents.value.steps ?? [];
    if (steps.length > 0) dom.appendChild(valueRow(translate("Through {0}", steps.join(" · "))));
  }
  return dom;
}

function hoverAt(
  service: LadingLanguageService,
  view: EditorView,
  position: number,
): LadingHover | null {
  return unlessRefused(
    () =>
      ladingHover(service.edi(), {
        ...ladingRequest(service, view.state.doc.toString(), position),
        values: [...service.values()],
      }),
    null,
  );
}

/** Opens the hover card for the name under the cursor (`K` in Vim); false when there is none. */
export const showLadingHoverAtCursor: Command = (view) => {
  const service = view.state.facet(ladingServiceFacet);
  if (!service) return false;
  const hover = hoverAt(service, view, view.state.selection.main.head);
  if (!hover) return false;
  openCursorTooltip(view, {
    pos: hover.range.from,
    end: hover.range.to,
    create: () => renderLadingHover(hover, service.edi()),
  });
  return true;
};

export function ladingHoverCard(service: LadingLanguageService): Extension {
  return [
    cursorTooltip,
    hoverTooltip((view, position): Tooltip | null => {
      const hover = hoverAt(service, view, position);
      if (!hover) return null;
      return {
        pos: hover.range.from,
        end: hover.range.to,
        above: true,
        create: () => ({ dom: renderLadingHover(hover, service.edi()) }),
      };
    }),
  ];
}
