// tariff-doc.ts shows a doc the module sent: a catalog message (translated) or an author's doc
// comment (their words, shown as written), with `backticked` spans set as code.

import { translateMessage } from "@trenova/shared/i18n/message";
import { RulesRequestError } from "@trenova/shared/rules/runtime";
import type { TariffDoc } from "@trenova/shared/rules/tariff";

/** A doc as text in the reader's language, or null for none. */
export function tariffDocText(doc: TariffDoc): string | null {
  if (doc === null) return null;
  return typeof doc === "string" ? doc : translateMessage(doc);
}

/** Appends a doc to an element, `code` spans as `<code>`; returns false when there is none. */
export function appendTariffDoc(parent: HTMLElement, doc: TariffDoc): boolean {
  const text = tariffDocText(doc);
  if (!text) return false;
  text.split("`").forEach((part, index) => {
    if (part === "") return;
    if (index % 2 === 1) {
      const code = document.createElement("code");
      code.textContent = part;
      parent.appendChild(code);
    } else {
      parent.appendChild(document.createTextNode(part));
    }
  });
  return true;
}

const reported = new Set<string>();

/**
 * Calls one of the editor's assisting operations (completion, hover, signature help, inlay
 * hints, go to definition, rename). A request the module refuses is a caller bug, so it is
 * logged, once per operation, for the developer; but it must never break typing, so the editor
 * carries on as if the operation had nothing to offer. A trap or any other failure still throws.
 */
export function unlessUnanswered<T>(ask: () => T, nothing: T): T {
  try {
    return ask();
  } catch (error) {
    if (!(error instanceof RulesRequestError)) throw error;
    if (!reported.has(error.op)) {
      reported.add(error.op);
      console.error(error);
    }
    return nothing;
  }
}
