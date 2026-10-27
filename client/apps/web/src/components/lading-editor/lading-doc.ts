// lading-doc.ts shows what the EDI module says about a name: a doc (an author's doc comment as
// written, or a catalog message translated) with its `backticked` spans as code, and a piece of
// Lading (a signature, a declaration) coloured from the module's own tokens.

import { ladingDocText, ladingTokens, type LadingDoc } from "@trenova/shared/edi/lading";
import type { EdiRuntime } from "@trenova/shared/edi/runtime";
import { translateMessage, type CatalogMessage } from "@trenova/shared/i18n/message";
import { appendDocText } from "../formula-editor/tariff-doc";
import { LADING_TOKEN_ROLES } from "./lading-highlight";
import { unlessRefused } from "./lading-service";

/** A doc as text in the reader's language; empty for none. */
export function ladingDoc(doc: LadingDoc | null | undefined): string {
  return ladingDocText(doc, (message: CatalogMessage) => translateMessage(message));
}

/** Appends a doc to an element; returns false when there is none. */
export function appendLadingDoc(parent: HTMLElement, doc: LadingDoc | null | undefined): boolean {
  const text = ladingDoc(doc);
  if (!text) return false;
  appendDocText(parent, text);
  return true;
}

/** Appends a piece of Lading in the editor's syntax colours; text the lexer refuses is plain. */
export function appendLadingCode(parent: HTMLElement, code: string, edi: EdiRuntime): void {
  const tokens = unlessRefused(() => ladingTokens(edi, code), []);
  let at = 0;
  for (const token of tokens) {
    const { from, to } = token.range;
    if (from < at || to > code.length) continue;
    if (from > at) parent.appendChild(document.createTextNode(code.slice(at, from)));
    const role = LADING_TOKEN_ROLES[token.kind];
    const text = code.slice(from, to);
    if (role) {
      const span = document.createElement("span");
      span.className = `cm-syntax-${role}`;
      span.textContent = text;
      parent.appendChild(span);
    } else {
      parent.appendChild(document.createTextNode(text));
    }
    at = to;
  }
  if (at < code.length) parent.appendChild(document.createTextNode(code.slice(at)));
}
