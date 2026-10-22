// tariff-signature.ts is signature help from `tariff.signature`: while the cursor is inside a
// call's parentheses, the callee's parameters float above it with the one being written in
// bold, and its doc under them.

import { StateField, type EditorState, type Extension } from "@codemirror/state";
import { showTooltip, type Tooltip } from "@codemirror/view";
import { tariffSignature, type TariffSignature } from "@trenova/shared/rules/tariff";
import { appendTariffDoc, unlessUnanswered } from "./tariff-doc";
import type { TariffLanguageService } from "./tariff-service";

export function renderTariffSignature(signature: NonNullable<TariffSignature>): HTMLElement {
  const dom = document.createElement("div");
  dom.className = "cm-tariff-signature";
  const code = document.createElement("code");
  code.className = "cm-tariff-signature-code";
  const open = signature.label.indexOf("(");
  code.appendChild(document.createTextNode(open >= 0 ? signature.label.slice(0, open + 1) : ""));
  signature.params.forEach((param, index) => {
    if (index > 0) code.appendChild(document.createTextNode(", "));
    const part = document.createElement("span");
    if (index === signature.active) part.className = "cm-tariff-signature-active";
    part.textContent = param.label;
    code.appendChild(part);
  });
  const close = signature.label.lastIndexOf(")");
  code.appendChild(document.createTextNode(close >= 0 ? signature.label.slice(close) : ""));
  dom.appendChild(code);
  const active = signature.params[signature.active];
  const doc = document.createElement("div");
  doc.className = "cm-tariff-signature-body";
  if (appendTariffDoc(doc, active?.doc ?? signature.doc)) dom.appendChild(doc);
  return dom;
}

function signatureTooltip(service: TariffLanguageService, state: EditorState): Tooltip | null {
  const selection = state.selection.main;
  if (!selection.empty) return null;
  const signature = unlessUnanswered(
    () =>
      tariffSignature(service.rules(), {
        source: state.doc.toString(),
        position: selection.head,
        kind: service.kind(),
        context: service.context(),
      }),
    null,
  );
  if (!signature) return null;
  return {
    pos: selection.head,
    above: true,
    strictSide: true,
    arrow: false,
    create: () => ({ dom: renderTariffSignature(signature) }),
  };
}

/** Signature help, recomputed as the cursor moves or the text changes. */
export function tariffSignatureHelp(service: TariffLanguageService): Extension {
  const field = StateField.define<Tooltip | null>({
    create: () => null,
    update(value, transaction) {
      if (!transaction.docChanged && !transaction.selection) return value;
      return signatureTooltip(service, transaction.state);
    },
    provide: (self) => showTooltip.from(self),
  });
  return field;
}
