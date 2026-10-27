// tariff.ts colours Tariff from the rules module's own tokens (`tariff.tokens`): the module is
// the only Tariff grammar, so the editor and every read-only view of a source spend its classes
// through this one map onto the shared syntax roles.

import type { CodeRole, CodeSpan } from "@trenova/shared/lib/highlight/code";
import type { TariffToken, TariffTokenKind } from "@trenova/shared/rules/tariff";

/**
 * The syntax role each token class is drawn in, after Gruvbox: keywords red, functions aqua,
 * fields blue, types and units yellow, charge codes and the text's escapes and interpolation
 * braces orange, literals purple, text green. A plain name stays in the ink.
 */
export const TARIFF_TOKEN_ROLES: Record<TariffTokenKind, CodeRole | undefined> = {
  keyword: "keyword",
  name: undefined,
  function: "function",
  field: "property",
  type: "type",
  code: "meta",
  number: "number",
  unit: "type",
  money: "number",
  percent: "number",
  date: "number",
  text: "string",
  escape: "meta",
  interpolation: "meta",
  punct: "operator",
  comment: "comment",
  doc: "comment",
  error: undefined,
};

/** The module's tokens as spans `HighlightedCode` draws. */
export function tariffCodeSpans(tokens: readonly TariffToken[]): CodeSpan[] {
  return tokens.map((token) => ({
    from: token.from,
    to: token.to,
    role: TARIFF_TOKEN_ROLES[token.kind],
  }));
}
