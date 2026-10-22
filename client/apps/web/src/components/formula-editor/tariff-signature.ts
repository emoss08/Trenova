// tariff-signature.ts is signature help from `tariff.signature`: while the cursor is inside a
// call's parentheses (a function, a method, a list method, a test constructor, a charge's
// options), the callee's parameters float above it with the one being written in bold, named
// arguments tracked, and that parameter's doc under them.

import { StateField, type EditorState, type Extension } from "@codemirror/state";
import { showTooltip, type Tooltip } from "@codemirror/view";
import { tariffSignature, type TariffSignatureHelp } from "@trenova/shared/rules/tariff";
import { appendTariffDoc, unlessUnanswered } from "./tariff-doc";
import type { TariffLanguageService } from "./tariff-service";

export function renderTariffSignature(help: NonNullable<TariffSignatureHelp>): HTMLElement | null {
  const signature = help.signatures[help.active] ?? help.signatures[0];
  if (!signature) return null;
  const dom = document.createElement("div");
  dom.className = "cm-tariff-signature";
  const code = document.createElement("code");
  code.className = "cm-tariff-signature-code";
  const active =
    help.activeParameter === null ? undefined : signature.parameters[help.activeParameter];
  if (active) {
    const [from, to] = active.label;
    code.appendChild(document.createTextNode(signature.label.slice(0, from)));
    const bold = document.createElement("span");
    bold.className = "cm-tariff-signature-active";
    bold.textContent = signature.label.slice(from, to);
    code.appendChild(bold);
    code.appendChild(document.createTextNode(signature.label.slice(to)));
  } else {
    code.textContent = signature.label;
  }
  dom.appendChild(code);
  const doc = document.createElement("div");
  doc.className = "cm-tariff-signature-body";
  if (appendTariffDoc(doc, active?.doc ?? signature.doc)) dom.appendChild(doc);
  return dom;
}

function signatureTooltip(service: TariffLanguageService, state: EditorState): Tooltip | null {
  const selection = state.selection.main;
  if (!selection.empty) return null;
  const help = unlessUnanswered(
    () =>
      tariffSignature(service.rules(), {
        source: state.doc.toString(),
        offset: selection.head,
        context: service.context(),
      }),
    null,
  );
  if (!help || help.signatures.length === 0) return null;
  return {
    pos: selection.head,
    above: true,
    strictSide: true,
    arrow: false,
    create: () => ({ dom: renderTariffSignature(help) ?? document.createElement("div") }),
  };
}

/** Signature help, recomputed as the cursor moves or the text changes. */
export function tariffSignatureHelp(service: TariffLanguageService): Extension {
  return StateField.define<Tooltip | null>({
    create: () => null,
    update(value, transaction) {
      if (!transaction.docChanged && !transaction.selection) return value;
      return signatureTooltip(service, transaction.state);
    },
    provide: (self) => showTooltip.from(self),
  });
}
