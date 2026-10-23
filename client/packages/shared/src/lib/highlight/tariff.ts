// tariff.ts colours Tariff from the rules module's own tokens (`tariff.tokens`): the module is
// the only Tariff grammar, so the editor and every read-only view of a source spend its classes
// through this one map onto the shared syntax roles.

import type { CodeRole, CodeSpan } from "@trenova/shared/lib/highlight/code";
import type { TariffToken, TariffTokenKind } from "@trenova/shared/rules/tariff";

/** The syntax role each token class is drawn in; names and punctuation's neighbours stay plain. */
export const TARIFF_TOKEN_ROLES: Record<TariffTokenKind, CodeRole | undefined> = {
  keyword: "keyword",
  name: undefined,
  type: "type",
  code: "meta",
  number: "number",
  unit: "property",
  money: "number",
  percent: "number",
  date: "number",
  text: "string",
  escape: "meta",
  interpolation: "keyword",
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
